import os
import socket
import subprocess
import time
from pathlib import Path
from typing import Generator

import pytest
import requests

from fixtures import types


def _collector_root() -> Path:
    configured = os.environ.get("SIGNOZ_OTEL_COLLECTOR_ROOT", "/home/cbw/code/OtelCollector")
    root = Path(configured).resolve()
    if not (root / "cmd/siginsightotelcollector/main.go").is_file():
        raise RuntimeError(
            "SIGNOZ_OTEL_COLLECTOR_ROOT must point to a Collector checkout containing "
            "cmd/siginsightotelcollector/main.go"
        )
    return root


def _free_tcp_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as listener:
        listener.bind(("127.0.0.1", 0))
        return listener.getsockname()[1]


def _host_clickhouse_dsn(clickhouse: types.TestContainerClickhouse, database: str) -> str:
    host = clickhouse.container.host_configs["9000"]
    username = clickhouse.env["SIGNOZ_TELEMETRYSTORE_CLICKHOUSE_USERNAME"]
    password = clickhouse.env["SIGNOZ_TELEMETRYSTORE_CLICKHOUSE_PASSWORD"]
    suffix = f"/{database}" if database else ""
    return f"tcp://{username}:{password}@{host.address}:{host.port}{suffix}"


def _run_migration(binary: Path, dsn: str, command: list[str]) -> None:
    result = subprocess.run(
        [str(binary), "migrate", *command, f"--clickhouse-dsn={dsn}"],
        check=False,
        capture_output=True,
        text=True,
        timeout=15 * 60,
    )
    if result.returncode != 0:
        raise RuntimeError(
            f"current Collector migration {' '.join(command)} failed:\n{result.stdout}\n{result.stderr}"
        )


@pytest.fixture(name="current_collector", scope="package")
def current_collector(
    clickhouse: types.TestContainerClickhouse,
    tmp_path_factory: pytest.TempPathFactory,
) -> Generator[str, None, None]:
    """Run the current Collector source against the test's ClickHouse instance."""

    root = _collector_root()
    work_dir = tmp_path_factory.mktemp("current-collector")
    binary = work_dir / "siginsight-otel-collector"
    build = subprocess.run(
        [
            "go",
            "build",
            "-tags=remove_all_sd",
            "-o",
            str(binary),
            "./cmd/siginsightotelcollector",
        ],
        cwd=root,
        check=False,
        capture_output=True,
        text=True,
        timeout=15 * 60,
    )
    if build.returncode != 0:
        raise RuntimeError(f"failed to build current Collector:\n{build.stdout}\n{build.stderr}")

    root_dsn = _host_clickhouse_dsn(clickhouse, "")
    for command in (["bootstrap"], ["sync", "up"], ["async", "up"], ["sync", "check"]):
        _run_migration(binary, root_dsn, command)

    port = _free_tcp_port()
    environment = os.environ | {
        "SIGINSIGHT_TEST_OTLP_HTTP_ENDPOINT": f"127.0.0.1:{port}",
        "SIGINSIGHT_TEST_TRACES_DSN": _host_clickhouse_dsn(clickhouse, "signoz_traces"),
        "SIGINSIGHT_TEST_LOGS_DSN": _host_clickhouse_dsn(clickhouse, "signoz_logs"),
        "SIGINSIGHT_TEST_METRICS_DSN": _host_clickhouse_dsn(clickhouse, "signoz_metrics"),
        "SIGINSIGHT_TEST_METER_DSN": _host_clickhouse_dsn(clickhouse, "signoz_meter"),
        "SIGINSIGHT_TEST_METADATA_DSN": _host_clickhouse_dsn(clickhouse, "signoz_metadata"),
    }
    config = root / "tests/integration-fixtures/lightweight-query-engine.yaml"
    process = subprocess.Popen(  # pylint: disable=consider-using-with
        [str(binary), f"--config={config}"],
        cwd=root,
        env=environment,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
    )
    endpoint = f"http://127.0.0.1:{port}"
    try:
        for _ in range(50):
            if process.poll() is not None:
                output = process.stdout.read() if process.stdout else ""
                raise RuntimeError(f"current Collector stopped before readiness:\n{output}")
            try:
                response = requests.post(f"{endpoint}/v1/logs", json={"resourceLogs": []}, timeout=1)
                if response.status_code == 200:
                    break
            except requests.RequestException:
                pass
            time.sleep(0.1)
        else:
            raise TimeoutError("current Collector did not open its OTLP HTTP endpoint")

        yield endpoint
    finally:
        process.terminate()
        try:
            process.wait(timeout=15)
        except subprocess.TimeoutExpired:
            process.kill()
            process.wait(timeout=5)

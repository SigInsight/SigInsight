import os

import docker
import pytest
from testcontainers.core.container import Network

from fixtures import dev, types
from fixtures.logger import setup_logger

logger = setup_logger(__name__)


@pytest.fixture(name="migrator", scope="package")
def migrator(
    network: Network,
    clickhouse: types.TestContainerClickhouse,
    request: pytest.FixtureRequest,
    pytestconfig: pytest.Config,
) -> types.Operation:
    """
    Package-scoped fixture for running schema migrations.
    """

    def create() -> None:
        version = request.config.getoption("--schema-migrator-version")
        image = os.getenv(
            "SIGINSIGHT_OTEL_COLLECTOR_IMAGE",
            f"ghcr.io/siginsight/siginsight-otel-collector:{version}",
        )
        client = docker.from_env()
        dsn = clickhouse.env["SIGNOZ_TELEMETRYSTORE_CLICKHOUSE_DSN"]

        container = client.containers.run(
            image=image,
            command=f"migrate bootstrap --clickhouse-dsn={dsn}",
            detach=True,
            auto_remove=False,
            network=network.id,
        )

        result = container.wait()

        if result["StatusCode"] != 0:
            logs = container.logs().decode(encoding="utf-8")
            container.remove()
            print(logs)
            raise RuntimeError("failed to run migrations on clickhouse")

        container.remove()

        container = client.containers.run(
            image=image,
            command=f"migrate sync up --clickhouse-dsn={dsn}",
            detach=True,
            auto_remove=False,
            network=network.id,
        )

        result = container.wait()
        if result["StatusCode"] != 0:
            logs = container.logs().decode(encoding="utf-8")
            container.remove()
            print(logs)
            raise RuntimeError("failed to run sync migrations on clickhouse")
        container.remove()

        container = client.containers.run(
            image=image,
            command=f"migrate async up --clickhouse-dsn={dsn}",
            detach=True,
            auto_remove=False,
            network=network.id,
        )

        result = container.wait()
        if result["StatusCode"] != 0:
            logs = container.logs().decode(encoding="utf-8")
            container.remove()
            print(logs)
            raise RuntimeError("failed to run async migrations on clickhouse")
        container.remove()

        check = client.containers.run(
            image=image,
            command=f"migrate sync check --clickhouse-dsn={dsn}",
            detach=True,
            auto_remove=False,
            network=network.id,
        )
        result = check.wait()
        if result["StatusCode"] != 0:
            logs = check.logs().decode(encoding="utf-8")
            check.remove()
            print(logs)
            raise RuntimeError("failed to verify sync migrations on clickhouse")
        check.remove()

        return types.Operation(name="migrator", container_id=None)

    def delete(operation: types.Operation) -> None:
        if not operation.container_id:
            return

        client = docker.from_env()
        container = client.containers.get(operation.container_id)
        container.stop(timeout=10)
        container.remove()

    def restore(cache: dict) -> types.Operation:
        return types.Operation(
            name=cache["name"],
            container_id=cache.get("container_id"),
        )

    return dev.wrap(
        request,
        pytestconfig,
        "migrator",
        lambda: types.Operation(name="", container_id=None),
        create,
        delete,
        restore,
    )

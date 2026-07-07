import docker
import pytest
import time
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
        client = docker.from_env()
        dsn = clickhouse.env["SIGNOZ_TELEMETRYSTORE_CLICKHOUSE_DSN"]

        container = client.containers.run(
            image=f"signoz/signoz-schema-migrator:{version}",
            command=f"sync --replication=true --cluster-name=cluster --up= --dsn={dsn}",
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
            image=f"signoz/signoz-schema-migrator:{version}",
            command=f"async --replication=true --cluster-name=cluster --up= --dsn={dsn}",
            detach=True,
            auto_remove=False,
            network=network.id,
        )

        time.sleep(5)
        container.reload()
        if container.status == "exited":
            result = container.wait()
            if result["StatusCode"] != 0:
                logs = container.logs().decode(encoding="utf-8")
                container.remove()
                print(logs)
                raise RuntimeError("failed to run async migrations on clickhouse")

        return types.Operation(name="migrator", container_id=container.id)

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

import docker
import docker.errors
import pytest
from testcontainers.core.container import DockerContainer, Network

from fixtures import dev, types
from fixtures.logger import setup_logger

logger = setup_logger(__name__)


@pytest.fixture(name="keeper", scope="package")
def keeper(
    network: Network,
    tmp_path_factory: pytest.TempPathFactory,
    request: pytest.FixtureRequest,
    pytestconfig: pytest.Config,
) -> types.TestContainerDocker:
    """
    Package-scoped fixture for ClickHouse Keeper.
    """

    def create() -> types.TestContainerDocker:
        version = request.config.getoption("--keeper-version")
        keeper_config = """
        <clickhouse>
            <logger>
                <level>information</level>
                <console>true</console>
            </logger>
            <listen_host>0.0.0.0</listen_host>
            <keeper_server>
                <tcp_port>9181</tcp_port>
                <server_id>1</server_id>
                <log_storage_path>/var/lib/clickhouse/coordination/log</log_storage_path>
                <snapshot_storage_path>/var/lib/clickhouse/coordination/snapshots</snapshot_storage_path>
                <coordination_settings>
                    <operation_timeout_ms>10000</operation_timeout_ms>
                    <min_session_timeout_ms>10000</min_session_timeout_ms>
                    <session_timeout_ms>100000</session_timeout_ms>
                    <raft_logs_level>information</raft_logs_level>
                </coordination_settings>
                <raft_configuration>
                    <server>
                        <id>1</id>
                        <hostname>localhost</hostname>
                        <port>9234</port>
                    </server>
                </raft_configuration>
            </keeper_server>
        </clickhouse>
        """

        tmp_dir = tmp_path_factory.mktemp("keeper")
        config_file_path = tmp_dir / "keeper.xml"
        config_file_path.write_text(keeper_config, encoding="utf-8")

        container = DockerContainer(image=f"clickhouse/clickhouse-server:{version}")
        container.with_command(
            "clickhouse-keeper --config-file=/etc/clickhouse-keeper/keeper_config.xml"
        )
        container.with_volume_mapping(
            str(config_file_path),
            "/etc/clickhouse-keeper/keeper_config.xml",
            "ro",
        )
        container.with_exposed_ports(9181)
        container.with_network(network=network)

        container.start()
        return types.TestContainerDocker(
            id=container.get_wrapped_container().id,
            host_configs={
                "9181": types.TestContainerUrlConfig(
                    scheme="tcp",
                    address=container.get_container_host_ip(),
                    port=container.get_exposed_port(9181),
                )
            },
            container_configs={
                "9181": types.TestContainerUrlConfig(
                    scheme="tcp",
                    address=container.get_wrapped_container().name,
                    port=9181,
                )
            },
        )

    def delete(container: types.TestContainerDocker):
        client = docker.from_env()
        try:
            client.containers.get(container_id=container.id).stop()
            client.containers.get(container_id=container.id).remove(v=True)
        except docker.errors.NotFound:
            logger.info(
                "Skipping removal of Keeper, Keeper(%s) not found. Maybe it was manually removed?",
                {"id": container.id},
            )

    def restore(cache: dict) -> types.TestContainerDocker:
        return types.TestContainerDocker.from_cache(cache)

    return dev.wrap(
        request,
        pytestconfig,
        "keeper",
        lambda: types.TestContainerDocker(id="", host_configs={}, container_configs={}),
        create,
        delete,
        restore,
    )

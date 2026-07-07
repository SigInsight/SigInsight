import os
from typing import Any, Generator

import clickhouse_connect
import clickhouse_connect.driver
import clickhouse_connect.driver.client
import docker
import docker.errors
import pytest
from testcontainers.clickhouse import ClickHouseContainer
from testcontainers.core.container import Network

from fixtures import dev, types
from fixtures.logger import setup_logger

logger = setup_logger(__name__)


@pytest.fixture(name="clickhouse", scope="package")
def clickhouse(
    tmpfs: Generator[types.LegacyPath, Any, None],
    network: Network,
    zookeeper: types.TestContainerDocker,
    request: pytest.FixtureRequest,
    pytestconfig: pytest.Config,
) -> types.TestContainerClickhouse:
    """
    Package-scoped fixture for Clickhouse TestContainer.
    """

    def create() -> types.TestContainerClickhouse:
        version = request.config.getoption("--clickhouse-version")
        histogram_quantile_image = os.getenv(
            "SIGNOZ_HISTOGRAM_QUANTILE_IMAGE_TEMPLATE",
            "ghcr.io/siginsight/clickhouse-init-histogram-quantile:{clickhouse_version}-latest",
        ).format(clickhouse_version=version)
        clickhouse_image = os.getenv(
            "SIGNOZ_CLICKHOUSE_IMAGE_TEMPLATE",
            "clickhouse/clickhouse-server:{clickhouse_version}",
        ).format(clickhouse_version=version)

        container = ClickHouseContainer(
            image=clickhouse_image,
            port=9000,
            username="signoz",
            password="password",
        )

        cluster_config = f"""
        <clickhouse>
            <logger>
                <level>information</level>
                <formatting>
                    <type>json</type>
                </formatting>
                <log>/var/log/clickhouse-server/clickhouse-server.log</log>
                <errorlog>/var/log/clickhouse-server/clickhouse-server.err.log</errorlog>
                <size>1000M</size>
                <count>3</count>
                <console>1</console>
            </logger>

            <macros>
                <shard>01</shard>
                <replica>01</replica>
            </macros>

            <zookeeper>
                <node>
                    <host>{zookeeper.container_configs["2181"].address}</host>
                    <port>{zookeeper.container_configs["2181"].port}</port>
                </node>
            </zookeeper>

            <remote_servers>
                <cluster>
                    <shard>
                        <replica>
                            <host>127.0.0.1</host>
                            <port>9000</port>
                        </replica>
                    </shard>
                </cluster>
            </remote_servers>

            <user_defined_executable_functions_config>*function.xml</user_defined_executable_functions_config>
            <user_scripts_path>/var/lib/clickhouse/user_scripts/</user_scripts_path>

            <distributed_ddl>
                <path>/clickhouse/task_queue/ddl</path>
                <profile>default</profile>
            </distributed_ddl>

            <storage_configuration>
                <disks>
                    <default>
                        <keep_free_space_bytes>1024</keep_free_space_bytes>
                    </default>
                    <cold>
                        <type>local</type>
                        <path>/var/lib/clickhouse/cold/</path>
                        <keep_free_space_bytes>1024</keep_free_space_bytes>
                    </cold>
                </disks>
                <policies>
                    <default>
                        <volumes>
                            <default>
                                <disk>default</disk>
                            </default>
                            <cold>
                                <disk>cold</disk>
                            </cold>
                        </volumes>
                    </default>
                    <tiered>
                        <volumes>
                            <default>
                                <disk>default</disk>
                            </default>
                            <cold>
                                <disk>cold</disk>
                            </cold>
                        </volumes>
                    </tiered>
                </policies>
            </storage_configuration>
        </clickhouse>
        """

        custom_function_config = """
        <functions>
            <function>
                <type>executable</type>
                <name>histogramQuantile</name>
                <return_type>Float64</return_type>
                <argument>
                    <type>Array(Float64)</type>
                    <name>buckets</name>
                </argument>
                <argument>
                    <type>Array(Float64)</type>
                    <name>counts</name>
                </argument>
                <argument>
                    <type>Float64</type>
                    <name>quantile</name>
                </argument>
                <format>CSV</format>
                <command>./histogramQuantile</command>
            </function>
        </functions>
        """

        tmp_dir = tmpfs("clickhouse")
        cluster_config_file_path = os.path.join(tmp_dir, "cluster.xml")
        with open(cluster_config_file_path, "w", encoding="utf-8") as f:
            f.write(cluster_config)

        custom_function_file_path = os.path.join(tmp_dir, "custom-function.xml")
        with open(custom_function_file_path, "w", encoding="utf-8") as f:
            f.write(custom_function_config)

        container.with_volume_mapping(
            cluster_config_file_path, "/etc/clickhouse-server/config.d/cluster.xml"
        )
        container.with_volume_mapping(
            custom_function_file_path,
            "/etc/clickhouse-server/custom-function.xml",
        )
        container.with_network(network)
        container.start()

        # Copy the histogramQuantile binary from the helper image into the running ClickHouse container.
        wrapped = container.get_wrapped_container()
        client = docker.from_env()
        helper_container = client.containers.create(histogram_quantile_image)
        try:
            exit_code, output = wrapped.exec_run(
                ["mkdir", "-p", "/var/lib/clickhouse/user_scripts"]
            )
            if exit_code != 0:
                raise RuntimeError(
                    f"Failed to create ClickHouse user_scripts directory: {output.decode()}"
                )

            archive, _ = helper_container.get_archive(
                "/opt/siginsight/bin/histogramQuantile"
            )
            if not wrapped.put_archive(
                "/var/lib/clickhouse/user_scripts", b"".join(archive)
            ):
                raise RuntimeError(
                    f"Failed to copy histogramQuantile binary from image {histogram_quantile_image}"
                )

            exit_code, output = wrapped.exec_run(
                ["chmod", "+x", "/var/lib/clickhouse/user_scripts/histogramQuantile"]
            )
        finally:
            helper_container.remove(force=True)

        if exit_code != 0:
            raise RuntimeError(
                f"Failed to install histogramQuantile binary from image {histogram_quantile_image}: {output.decode()}"
            )

        connection = clickhouse_connect.get_client(
            user=container.username,
            password=container.password,
            host=container.get_container_host_ip(),
            port=container.get_exposed_port(8123),
        )

        return types.TestContainerClickhouse(
            container=types.TestContainerDocker(
                id=container.get_wrapped_container().id,
                host_configs={
                    "9000": types.TestContainerUrlConfig(
                        "tcp",
                        container.get_container_host_ip(),
                        container.get_exposed_port(9000),
                    ),
                    "8123": types.TestContainerUrlConfig(
                        "tcp",
                        container.get_container_host_ip(),
                        container.get_exposed_port(8123),
                    ),
                },
                container_configs={
                    "9000": types.TestContainerUrlConfig(
                        "tcp", container.get_wrapped_container().name, 9000
                    ),
                    "8123": types.TestContainerUrlConfig(
                        "tcp", container.get_wrapped_container().name, 8123
                    ),
                },
            ),
            conn=connection,
            env={
                "SIGNOZ_TELEMETRYSTORE_CLICKHOUSE_DSN": f"tcp://{container.username}:{container.password}@{container.get_wrapped_container().name}:{9000}",
                "SIGNOZ_TELEMETRYSTORE_CLICKHOUSE_USERNAME": container.username,
                "SIGNOZ_TELEMETRYSTORE_CLICKHOUSE_PASSWORD": container.password,
                "SIGNOZ_TELEMETRYSTORE_CLICKHOUSE_CLUSTER": "cluster",
            },
        )

    def delete(container: types.TestContainerClickhouse) -> None:
        client = docker.from_env()
        try:
            client.containers.get(container_id=container.container.id).stop()
            client.containers.get(container_id=container.container.id).remove(v=True)
        except docker.errors.NotFound:
            logger.info(
                "Skipping removal of Clickhouse, Clickhouse(%s) not found. Maybe it was manually removed?",
                {"id": container.id},
            )

    def restore(cache: dict) -> types.TestContainerClickhouse:
        container = types.TestContainerDocker.from_cache(cache["container"])
        host_config = container.host_configs["8123"]
        env = cache["env"]

        conn = clickhouse_connect.get_client(
            user=env["SIGNOZ_TELEMETRYSTORE_CLICKHOUSE_USERNAME"],
            password=env["SIGNOZ_TELEMETRYSTORE_CLICKHOUSE_PASSWORD"],
            host=host_config.address,
            port=host_config.port,
        )

        return types.TestContainerClickhouse(
            container=container,
            conn=conn,
            env=env,
        )

    return dev.wrap(
        request,
        pytestconfig,
        "clickhouse",
        empty=lambda: types.TestContainerSQL(
            container=types.TestContainerDocker(
                id="", host_configs={}, container_configs={}
            ),
            conn=None,
            env={},
        ),
        create=create,
        delete=delete,
        restore=restore,
    )

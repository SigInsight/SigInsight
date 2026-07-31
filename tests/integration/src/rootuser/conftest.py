import pytest
from testcontainers.core.container import Network

from fixtures import types
from fixtures.signoz import create_signoz

ROOT_USER_EMAIL = "rootuser@integration.test"
ROOT_USER_PASSWORD = "password123Z$"


@pytest.fixture(name="signoz", scope="package")
def signoz_rootuser(
    network: Network,
    sqlstore: types.TestContainerSQL,
    clickhouse: types.TestContainerClickhouse,
    request: pytest.FixtureRequest,
    pytestconfig: pytest.Config,
) -> types.SigNoz:
    """
    Package-scoped fixture for SigNoz with root user and impersonation enabled.
    """
    return create_signoz(
        network=network,
        sqlstore=sqlstore,
        clickhouse=clickhouse,
        request=request,
        pytestconfig=pytestconfig,
        cache_key="signoz-rootuser",
        env_overrides={
            "SIGINSIGHT_IDENTN_IMPERSONATION_ENABLED": True,
            "SIGINSIGHT_IDENTN_TOKENIZER_ENABLED": False,
            "SIGINSIGHT_IDENTN_APIKEY_ENABLED": False,
            "SIGINSIGHT_USER_ROOT_ENABLED": True,
            "SIGINSIGHT_USER_ROOT_EMAIL": ROOT_USER_EMAIL,
            "SIGINSIGHT_USER_ROOT_PASSWORD": ROOT_USER_PASSWORD,
        },
    )

from typing import Callable, List

import pytest
from wiremock.client import (
    Mapping,
    Mappings,
    Requests,
)
from wiremock.constants import Config

from fixtures import types


@pytest.fixture(name="make_http_mocks", scope="function")
def make_http_mocks() -> Callable[[types.TestContainerDocker, List[Mapping]], None]:
    def _make_http_mocks(
        container: types.TestContainerDocker, mappings: List[Mapping]
    ) -> None:
        Config.base_url = container.host_configs["8080"].get("/__admin")

        for mapping in mappings:
            Mappings.create_mapping(mapping=mapping)

    yield _make_http_mocks

    Mappings.delete_all_mappings()
    Requests.reset_request_journal()

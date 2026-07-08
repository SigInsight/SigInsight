from http import HTTPStatus
from typing import Callable, Tuple

import pytest
import requests

from fixtures import dev, types
from fixtures.logger import setup_logger

logger = setup_logger(__name__)

USER_ADMIN_NAME = "admin"
USER_ADMIN_EMAIL = "admin@integration.test"
USER_ADMIN_PASSWORD = "password123Z$"
USER_ADMIN_ORG_NAME = "integration.test"

USER_EDITOR_NAME = "editor"
USER_EDITOR_EMAIL = "editor@integration.test"
USER_EDITOR_PASSWORD = "password123Z$"


@pytest.fixture(name="create_user_admin", scope="package")
def create_user_admin(
    signoz: types.SigNoz, request: pytest.FixtureRequest, pytestconfig: pytest.Config
) -> types.Operation:
    def create() -> types.Operation:
        response = requests.post(
            signoz.self.host_configs["8080"].get("/api/v1/register"),
            json={
                "name": USER_ADMIN_NAME,
                "orgName": USER_ADMIN_ORG_NAME,
                "email": USER_ADMIN_EMAIL,
                "password": USER_ADMIN_PASSWORD,
            },
            timeout=5,
        )

        if response.status_code == HTTPStatus.OK:
            return types.Operation(name="create_user_admin")

        if response.status_code == HTTPStatus.BAD_REQUEST:
            context_response = requests.get(
                signoz.self.host_configs["8080"].get("/api/v2/sessions/context"),
                params={
                    "email": USER_ADMIN_EMAIL,
                    "ref": f"{signoz.self.host_configs['8080'].base()}",
                },
                timeout=5,
            )

            if context_response.status_code == HTTPStatus.OK:
                org_id = context_response.json()["data"]["orgs"][0]["id"]
                token_response = requests.post(
                    signoz.self.host_configs["8080"].get(
                        "/api/v2/sessions/email_password"
                    ),
                    json={
                        "email": USER_ADMIN_EMAIL,
                        "password": USER_ADMIN_PASSWORD,
                        "orgId": org_id,
                    },
                    timeout=5,
                )
                if token_response.status_code == HTTPStatus.OK:
                    return types.Operation(name="create_user_admin")

        raise AssertionError(
            f"failed to create or reuse admin user: register={response.status_code} {response.text}"
        )

    def delete(_: types.Operation) -> None:
        pass

    def restore(cache: dict) -> types.Operation:
        return types.Operation(name=cache["name"])

    return dev.wrap(
        request,
        pytestconfig,
        "create_user_admin",
        lambda: types.Operation(name=""),
        create,
        delete,
        restore,
    )


@pytest.fixture(name="get_session_context", scope="function")
def get_session_context(signoz: types.SigNoz) -> Callable[[str, str], str]:
    def _get_session_context(email: str) -> str:
        response = requests.get(
            signoz.self.host_configs["8080"].get("/api/v2/sessions/context"),
            params={
                "email": email,
                "ref": f"{signoz.self.host_configs['8080'].base()}",
            },
            timeout=5,
        )

        assert response.status_code == HTTPStatus.OK
        return response.json()["data"]

    return _get_session_context


@pytest.fixture(name="get_token", scope="function")
def get_token(signoz: types.SigNoz) -> Callable[[str, str], str]:
    def _get_token(email: str, password: str) -> str:
        response = requests.get(
            signoz.self.host_configs["8080"].get("/api/v2/sessions/context"),
            params={
                "email": email,
                "ref": f"{signoz.self.host_configs['8080'].base()}",
            },
            timeout=5,
        )

        assert response.status_code == HTTPStatus.OK
        org_id = response.json()["data"]["orgs"][0]["id"]

        response = requests.post(
            signoz.self.host_configs["8080"].get("/api/v2/sessions/email_password"),
            json={
                "email": email,
                "password": password,
                "orgId": org_id,
            },
            timeout=5,
        )

        assert response.status_code == HTTPStatus.OK
        return response.json()["data"]["accessToken"]

    return _get_token


@pytest.fixture(name="get_tokens", scope="function")
def get_tokens(signoz: types.SigNoz) -> Callable[[str, str], Tuple[str, str]]:
    def _get_tokens(email: str, password: str) -> str:
        response = requests.get(
            signoz.self.host_configs["8080"].get("/api/v2/sessions/context"),
            params={
                "email": email,
                "ref": f"{signoz.self.host_configs['8080'].base()}",
            },
            timeout=5,
        )

        assert response.status_code == HTTPStatus.OK
        org_id = response.json()["data"]["orgs"][0]["id"]

        response = requests.post(
            signoz.self.host_configs["8080"].get("/api/v2/sessions/email_password"),
            json={
                "email": email,
                "password": password,
                "orgId": org_id,
            },
            timeout=5,
        )

        assert response.status_code == HTTPStatus.OK
        access_token = response.json()["data"]["accessToken"]
        refresh_token = response.json()["data"]["refreshToken"]
        return access_token, refresh_token

    return _get_tokens

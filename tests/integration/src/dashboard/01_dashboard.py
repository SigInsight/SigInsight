from http import HTTPStatus
from typing import Callable

import requests

from fixtures.auth import USER_ADMIN_EMAIL, USER_ADMIN_PASSWORD
from fixtures.types import Operation, SigNoz


def test_create_and_delete_dashboard(
    signoz: SigNoz,
    create_user_admin: Operation,  # pylint: disable=unused-argument
    get_token: Callable[[str, str], str],
):
    admin_token = get_token(USER_ADMIN_EMAIL, USER_ADMIN_PASSWORD)

    response = requests.post(
        signoz.self.host_configs["8080"].get("/api/v1/dashboards"),
        json={"title": "Sample Title", "uploadedGrafana": False, "version": "v5"},
        headers={"Authorization": f"Bearer {admin_token}"},
        timeout=2,
    )

    assert response.status_code == HTTPStatus.CREATED
    assert response.json()["status"] == "success"
    data = response.json()["data"]
    dashboard_id = data["id"]

    response = requests.delete(
        signoz.self.host_configs["8080"].get(f"/api/v1/dashboards/{dashboard_id}"),
        headers={"Authorization": f"Bearer {admin_token}"},
        timeout=2,
    )

    assert response.status_code == HTTPStatus.NO_CONTENT

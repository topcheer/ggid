"""Passkey/WebAuthn registration and authentication API calls.

Usage::

    from ggid.passkey import PasskeyClient
    pk = PasskeyClient("https://ggid.example.com")
    options = pk.begin_registration(token, "my-passkey")
"""

from __future__ import annotations
import json
from typing import Any
import urllib.request


class PasskeyClient:
    """Client for WebAuthn/Passkey API calls."""

    def __init__(self, base_url: str):
        self.base_url = base_url.rstrip("/")

    def begin_registration(self, access_token: str, device_name: str) -> dict:
        """Begin WebAuthn registration. Returns server challenge options."""
        return self._post(
            "/api/v1/auth/mfa/enroll",
            {"type": "webauthn", "name": device_name},
            access_token,
        )

    def finish_registration(self, access_token: str, device_id: str, attestation: str) -> dict:
        """Finish WebAuthn registration by verifying the attestation."""
        return self._post(
            "/api/v1/auth/mfa/verify",
            {"device_id": device_id, "code": attestation},
            access_token,
        )

    def begin_login(self, username: str) -> dict:
        """Begin WebAuthn login. Returns server challenge options."""
        return self._post(
            "/api/v1/auth/webauthn/login/begin",
            {"username": username},
        )

    def finish_login(self, assertion: str) -> dict:
        """Finish WebAuthn login by verifying the assertion."""
        return self._post(
            "/api/v1/auth/webauthn/login/finish",
            {"assertion": assertion},
        )

    def _post(self, path: str, body: dict, token: str | None = None) -> dict:
        url = f"{self.base_url}{path}"
        data = json.dumps(body).encode()
        req = urllib.request.Request(url, data=data, method="POST")
        req.add_header("Content-Type", "application/json")
        if token:
            req.add_header("Authorization", f"Bearer {token}")
        with urllib.request.urlopen(req) as resp:
            return json.loads(resp.read())

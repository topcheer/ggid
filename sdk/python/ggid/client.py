"""GGID API client for user management and permission checking."""

import httpx
from typing import Optional
from ggid.jwt import JWTVerifier


class GGIDClient:
    """Async client for GGID IAM Platform APIs.

    Usage:
        client = GGIDClient(gateway_url="https://iam.example.com")
        users = await client.list_users(token="eyJ...")
    """

    def __init__(
        self,
        gateway_url: str,
        tenant_id: str = "00000000-0000-0000-0000-000000000001",
        jwks_url: str = "",
    ):
        self.base_url = gateway_url.rstrip("/")
        self.tenant_id = tenant_id
        self._client = httpx.AsyncClient(timeout=30)
        self._verifier: Optional[JWTVerifier] = None
        if jwks_url:
            self._verifier = JWTVerifier(jwks_url=jwks_url)

    async def close(self):
        await self._client.aclose()

    async def __aenter__(self):
        return self

    async def __aexit__(self, *args):
        await self.close()

    def _headers(self, token: str = "") -> dict:
        h = {"X-Tenant-ID": self.tenant_id, "Content-Type": "application/json"}
        if token:
            h["Authorization"] = f"Bearer {token}"
        return h

    # --- User Management ---

    async def list_users(self, token: str, limit: int = 50) -> dict:
        resp = await self._client.get(
            f"{self.base_url}/api/v1/users",
            params={"limit": limit},
            headers=self._headers(token),
        )
        resp.raise_for_status()
        return resp.json()

    async def get_user(self, token: str, user_id: str) -> dict:
        resp = await self._client.get(
            f"{self.base_url}/api/v1/users/{user_id}",
            headers=self._headers(token),
        )
        resp.raise_for_status()
        return resp.json()

    async def create_user(
        self, token: str, username: str, email: str, password: str, name: str = ""
    ) -> dict:
        resp = await self._client.post(
            f"{self.base_url}/api/v1/users",
            json={"username": username, "email": email, "password": password, "name": name},
            headers=self._headers(token),
        )
        resp.raise_for_status()
        return resp.json()

    async def delete_user(self, token: str, user_id: str) -> bool:
        resp = await self._client.delete(
            f"{self.base_url}/api/v1/users/{user_id}",
            headers=self._headers(token),
        )
        return resp.status_code in (200, 204)

    async def update_user(
        self, token: str, user_id: str, email: str = "", phone: str = "", status: str = ""
    ) -> dict:
        """Update a user's attributes. Only non-empty fields are sent."""
        payload = {}
        if email:
            payload["email"] = email
        if phone:
            payload["phone"] = phone
        if status:
            payload["status"] = status
        resp = await self._client.patch(
            f"{self.base_url}/api/v1/users/{user_id}",
            json=payload,
            headers=self._headers(token),
        )
        resp.raise_for_status()
        return resp.json()

    # --- Auth ---

    async def login(self, username: str, password: str) -> dict:
        resp = await self._client.post(
            f"{self.base_url}/api/v1/auth/login",
            json={"username": username, "password": password},
            headers=self._headers(),
        )
        resp.raise_for_status()
        return resp.json()

    async def register(self, username: str, email: str, password: str, name: str = "") -> dict:
        resp = await self._client.post(
            f"{self.base_url}/api/v1/auth/register",
            json={"username": username, "email": email, "password": password, "name": name},
            headers=self._headers(),
        )
        resp.raise_for_status()
        return resp.json()

    # --- RBAC ---

    async def list_roles(self, token: str) -> dict:
        resp = await self._client.get(
            f"{self.base_url}/api/v1/roles",
            headers=self._headers(token),
        )
        resp.raise_for_status()
        return resp.json()

    async def check_permission(
        self, token: str, resource: str, action: str, user_id: str = ""
    ) -> bool:
        resp = await self._client.post(
            f"{self.base_url}/api/v1/policies/check",
            json={"resource": resource, "action": action, "user_id": user_id},
            headers=self._headers(token),
        )
        if resp.status_code == 200:
            data = resp.json()
            return data.get("allowed", False)
        return False

    # --- JWT ---

    async def verify_token(self, token: str):
        if not self._verifier:
            raise RuntimeError("no jwks_url configured")
        return await self._verifier.verify(token)

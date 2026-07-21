"""GGID Client — HTTP client for all GGID API calls."""

import json
import requests
from typing import Any, Optional
from dataclasses import dataclass, field


class GGIDError(Exception):
    """GGID API error."""

    def __init__(self, message: str, status_code: int = 0, body: Any = None):
        super().__init__(message)
        self.status_code = status_code
        self.body = body


@dataclass
class GGIDConfig:
    base_url: str = "http://localhost:8080"
    tenant_id: str = "00000000-0000-0000-0000-000000000001"
    api_key: Optional[str] = None
    timeout: int = 30


class GGIDClient:
    """Main GGID API client."""

    def __init__(self, config: GGIDConfig):
        self.config = config
        self.base_url = config.base_url.rstrip("/")
        self._session = requests.Session()

    def _headers(self, token: Optional[str] = None) -> dict:
        h = {
            "Content-Type": "application/json",
            "X-Tenant-ID": self.config.tenant_id,
        }
        if self.config.api_key:
            h["X-API-Key"] = self.config.api_key
        if token:
            h["Authorization"] = f"Bearer {token}"
        return h

    def _request(
        self,
        method: str,
        path: str,
        body: Any = None,
        token: Optional[str] = None,
        params: Optional[dict] = None,
    ) -> Any:
        url = f"{self.base_url}{path}"
        resp = self._session.request(
            method,
            url,
            json=body if body is not None else None,
            params=params,
            headers=self._headers(token),
            timeout=self.config.timeout,
        )
        if resp.status_code >= 400:
            try:
                err_body = resp.json()
            except Exception:
                err_body = resp.text
            raise GGIDError(
                f"API error {resp.status_code}: {err_body}",
                status_code=resp.status_code,
                body=err_body,
            )
        if resp.status_code == 204 or not resp.text:
            return None
        return resp.json()

    # --- User Management ---

    def register(self, username: str, email: str, password: str) -> dict:
        return self._request("POST", "/api/v1/auth/register", {
            "username": username, "email": email, "password": password,
        })

    def login(self, username: str, password: str) -> dict:
        return self._request("POST", "/api/v1/auth/login", {
            "username": username, "password": password,
        })

    def get_user(self, token: str, user_id: str) -> dict:
        return self._request("GET", f"/api/v1/users/{user_id}", token=token)

    def list_users(self, token: str, **params) -> dict:
        return self._request("GET", "/api/v1/users", token=token, params=params)

    def create_user(self, token: str, data: dict) -> dict:
        return self._request("POST", "/api/v1/users", data, token=token)

    def update_user(self, token: str, user_id: str, data: dict) -> dict:
        return self._request("PUT", f"/api/v1/users/{user_id}", data, token=token)

    def delete_user(self, token: str, user_id: str) -> None:
        self._request("DELETE", f"/api/v1/users/{user_id}", token=token)

    # --- OAuth/OIDC ---

    def get_oidc_discovery(self) -> dict:
        return self._request("GET", "/.well-known/openid-configuration")

    def get_jwks(self) -> dict:
        return self._request("GET", "/.well-known/jwks.json")

    def get_user_info(self, access_token: str) -> dict:
        return self._request("GET", "/oauth/userinfo", token=access_token)

    def introspect_token(self, token: str, client_id: str, client_secret: str) -> dict:
        return self._request("POST", "/oauth/introspect", {
            "token": token, "client_id": client_id, "client_secret": client_secret,
        })

    def revoke_token(self, token: str, access_token: str) -> None:
        self._request("POST", "/oauth/revoke", {"token": token}, token=access_token)

    # --- Roles CRUD ---

    def create_role(self, token: str, name: str, key: str, description: str = "") -> dict:
        return self._request("POST", "/api/v1/roles", {
            "name": name, "key": key, "description": description,
        }, token=token)

    def get_role(self, token: str, role_id: str) -> dict:
        return self._request("GET", f"/api/v1/roles/{role_id}", token=token)

    def list_roles(self, token: str) -> dict:
        return self._request("GET", "/api/v1/roles", token=token)

    def update_role(self, token: str, role_id: str, name: str = None, description: str = None) -> dict:
        body = {}
        if name: body["name"] = name
        if description: body["description"] = description
        return self._request("PUT", f"/api/v1/roles/{role_id}", body, token=token)

    def delete_role(self, token: str, role_id: str) -> None:
        self._request("DELETE", f"/api/v1/roles/{role_id}", token=token)

    # --- RBAC ---

    def check_permission(self, token: str, resource: str, action: str) -> dict:
        """Check if user has permission for resource+action. Returns {allowed, reason}."""
        return self._request("GET", "/api/v1/policies/check", token=token,
                             params={"resource": resource, "action": action})

    def assign_role(self, token: str, user_id: str, role_id: str) -> dict:
        return self._request("POST", f"/api/v1/policies/roles/{role_id}/users/{user_id}", {
            "user_id": user_id, "role_id": role_id,
        }, token=token)

    def revoke_role(self, token: str, user_id: str, role_id: str) -> None:
        self._request("DELETE", f"/api/v1/policies/roles/{role_id}/users/{user_id}", token=token)

    def get_user_roles(self, token: str, user_id: str) -> list:
        return self._request("GET", f"/api/v1/policies/users/{user_id}/roles", token=token)

    def list_permissions(self, token: str) -> list:
        """Get the permission tree."""
        return self._request("GET", "/api/v1/policies/permissions/tree", token=token)

    # --- ABAC ---

    def check_policy(self, token: str, subject: str, resource: str, action: str,
                     context: Optional[dict] = None) -> dict:
        """Full ABAC policy evaluation via POST."""
        body = {"subject": subject, "resource": resource, "action": action}
        if context: body["context"] = context
        return self._request("POST", "/api/v1/policies/abac/evaluate", body, token=token)

    def evaluate_abac(self, token: str, action: str, resource: str, subject: str,
                      conditions: Optional[list] = None, tenant_id: Optional[str] = None) -> dict:
        """ABAC evaluation with structured conditions [{field, operator, value}]."""
        body: dict = {"action": action, "resource": resource, "subject": subject}
        if conditions: body["conditions"] = conditions
        if tenant_id: body["tenant_id"] = tenant_id
        return self._request("POST", "/api/v1/policies/abac/evaluate", body, token=token)

    # --- Audit ---

    def list_audit_events(self, token: str, **params) -> dict:
        return self._request("GET", "/api/v1/audit/events", token=token, params=params)

    # --- Agent Identity ---

    def register_agent(self, token: str, name: str, agent_type: str,
                     allowed_scopes: list[str], owner_user_id: str = "",
                     description: str = "", max_delegation_depth: int = 3,
                     rate_limit_per_min: int = 60) -> dict:
        """Register a new AI agent identity."""
        body = {
            "name": name,
            "type": agent_type,
            "owner_user_id": owner_user_id,
            "description": description,
            "allowed_scopes": allowed_scopes,
            "max_delegation_depth": max_delegation_depth,
            "rate_limit_per_min": rate_limit_per_min,
        }
        return self._request("POST", "/api/v1/agents/register", body, token=token)

    def list_agents(self, token: str) -> dict:
        """List all agents for the current tenant."""
        return self._request("GET", "/api/v1/agents", token=token)

    def exchange_agent_token(self, agent_id: str, subject_token: str,
                             scopes: list[str]) -> dict:
        """Exchange a user access token for an agent-scoped token."""
        body = {
            "agent_id": agent_id,
            "subject_token": subject_token,
            "scope": scopes,
        }
        return self._request("POST", "/api/v1/agents/token", body)

    def verify_agent_token(self, token: str) -> dict:
        """Verify an agent token and return its claims."""
        return self._request("POST", "/api/v1/agents/verify", {"token": token})

    # --- Access Request (IGA) ---

    def create_access_request(self, token: str, user_id: str, resource: str,
                              action: str, reason: str = "") -> dict:
        """Create an access request for review/approval workflow."""
        body = {
            "user_id": user_id,
            "resource": resource,
            "action": action,
            "reason": reason,
        }
        return self._request("POST", "/api/v1/access-requests", body, token=token)

    def list_access_requests(self, token: str) -> dict:
        """List access requests for the current tenant."""
        return self._request("GET", "/api/v1/access-requests", token=token)

    def approve_access_request(self, token: str, request_id: str,
                               comment: str = "") -> dict:
        """Approve an access request."""
        body = {"comment": comment}
        return self._request("POST", f"/api/v1/access-requests/{request_id}/approve", body, token=token)

    def reject_access_request(self, token: str, request_id: str,
                              comment: str = "") -> dict:
        """Reject an access request."""
        body = {"comment": comment}
        return self._request("POST", f"/api/v1/access-requests/{request_id}/reject", body, token=token)

    def list_webhooks(self, token: str) -> list:
        """List all webhooks for the current tenant."""
        return self._request("GET", "/api/v1/webhooks", token=token)

    def create_webhook(self, token: str, url: str, events: list[str],
                       secret: Optional[str] = None) -> dict:
        """Create a new webhook.

        Args:
            token: Bearer token with webhook:write permission.
            url: Webhook endpoint URL.
            events: List of event types (e.g. ["user.created", "user.deleted"]).
            secret: Optional shared secret for HMAC signature verification.
        """
        body: dict = {"url": url, "events": events}
        if secret:
            body["secret"] = secret
        return self._request("POST", "/api/v1/webhooks", body, token=token)

    def delete_webhook(self, token: str, webhook_id: str) -> dict:
        """Delete a webhook by ID."""
        return self._request("DELETE", f"/api/v1/webhooks/{webhook_id}", token=token)

    def client_credentials(self, client_id: str, client_secret: str, scope: str = "") -> dict:
        """Obtain an access token using client_credentials grant (M2M)."""
        return self._request("POST", "/oauth/token", {
            "grant_type": "client_credentials",
            "client_id": client_id,
            "client_secret": client_secret,
            "scope": scope,
        })

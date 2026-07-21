"""JWT verification for GGID tokens using PyJWT."""

import jwt
import requests
import time
from typing import Optional, Any
from dataclasses import dataclass


class JWTError(Exception):
    """JWT verification error."""
    pass


@dataclass
class JWTClaims:
    sub: str
    tenant_id: str
    scopes: list
    roles: list
    permissions: list  # Fine-grained permissions (inventory:read, orders:write)
    exp: int
    iat: int
    iss: str
    raw: dict

    @property
    def is_expired(self) -> bool:
        return time.time() > self.exp

    @property
    def has_admin(self) -> bool:
        """Check if user has admin-level permission."""
        return "admin" in (self.permissions or []) or "admin" in (self.roles or [])

    def has_permission(self, permission: str) -> bool:
        """Check if user has a fine-grained permission. Admin bypasses."""
        if self.has_admin:
            return True
        return permission in (self.permissions or [])


class JWTVerifier:
    """Verifies GGID JWT tokens using JWKS from the OAuth service."""

    def __init__(self, base_url: str = "http://localhost:8080",
                 jwks_uri: Optional[str] = None,
                 issuer: Optional[str] = None,
                 cache_ttl: int = 3600):
        self.base_url = base_url.rstrip("/")
        self._jwks_uri = jwks_uri
        self._issuer = issuer
        self._jwks: Optional[dict] = None
        self._jwks_fetched_at: float = 0
        self._cache_ttl = cache_ttl

    def _get_jwks(self) -> dict:
        """Fetch JWKS with caching."""
        if self._jwks and (time.time() - self._jwks_fetched_at) < self._cache_ttl:
            return self._jwks

        uri = self._jwks_uri or f"{self.base_url}/.well-known/jwks.json"
        resp = requests.get(uri, timeout=10)
        resp.raise_for_status()
        self._jwks = resp.json()
        self._jwks_fetched_at = time.time()
        return self._jwks

    def _get_key(self, kid: str):
        """Get the signing key for a given key ID."""
        jwks = self._get_jwks()
        for key in jwks.get("keys", []):
            if key.get("kid") == kid:
                from jwt import PyJWK
                return PyJWK(key).key
        raise JWTError(f"key not found for kid: {kid}")

    def verify(self, token: str) -> JWTClaims:
        """Verify a JWT token and return claims."""
        try:
            unverified_header = jwt.get_unverified_header(token)
            kid = unverified_header.get("kid")
            if not kid:
                raise JWTError("missing kid in token header")

            key = self._get_key(kid)
            decoded = jwt.decode(
                token,
                key=key,
                algorithms=["RS256"],
                issuer=self._issuer,
                options={"verify_aud": False},
            )
            return self._parse_claims(decoded)
        except jwt.PyJWTError as e:
            raise JWTError(f"JWT verification failed: {e}") from e

    def _parse_claims(self, claims: dict) -> JWTClaims:
        """Parse raw JWT claims into structured JWTClaims."""
        # scopes: standard OAuth2 "scope" claim (space-delimited string)
        raw_scopes = claims.get("scope", "")
        if isinstance(raw_scopes, str):
            raw_scopes = raw_scopes.split()
        return JWTClaims(
            sub=claims.get("sub", ""),
            tenant_id=claims.get("tenant_id", ""),
            scopes=raw_scopes,
            roles=claims.get("roles", []),
            permissions=claims.get("permissions", []),
            exp=claims.get("exp", 0),
            iat=claims.get("iat", 0),
            iss=claims.get("iss", ""),
            raw=claims,
        )

    def verify_scopes(self, token: str, required_scopes: list) -> bool:
        """Verify token has all required scopes."""
        claims = self.verify(token)
        token_scopes = set(claims.scopes)
        return all(s in token_scopes for s in required_scopes)

    def verify_roles(self, token: str, required_roles: list) -> bool:
        """Verify token has any of the required roles."""
        claims = self.verify(token)
        token_roles = set(claims.roles)
        return any(r in token_roles for r in required_roles)

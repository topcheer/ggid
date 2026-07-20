"""JWT verification with JWKS caching for GGID IAM Platform."""

import time
import json
import httpx
import jwt as pyjwt
from jwt.algorithms import RSAAlgorithm
from dataclasses import dataclass, field
from typing import Optional


class JWTError(Exception):
    """Raised when JWT verification fails."""
    pass


@dataclass
class JWTClaims:
    """Standard JWT claims extracted from a verified token."""
    sub: str
    email: str = ""
    name: str = ""
    tenant_id: str = ""
    roles: list = field(default_factory=list)
    permissions: list = field(default_factory=list)  # Fine-grained permissions
    exp: int = 0
    iat: int = 0
    iss: str = ""
    raw: dict = field(default_factory=dict)

    def has_permission(self, permission: str) -> bool:
        """Check if user has a fine-grained permission. Admin bypasses."""
        if "admin" in (self.permissions or []):
            return True
        return permission in (self.permissions or [])


class JWTVerifier:
    """Verify RS256 JWTs against GGID's JWKS endpoint with caching."""

    def __init__(
        self,
        jwks_url: str,
        issuer: str = "",
        cache_ttl: int = 300,
    ):
        self.jwks_url = jwks_url
        self.issuer = issuer
        self.cache_ttl = cache_ttl
        self._jwks: dict = {}
        self._jwks_fetched_at: float = 0

    async def _fetch_jwks(self) -> dict:
        """Fetch JWKS from the gateway, with caching."""
        now = time.time()
        if self._jwks and (now - self._jwks_fetched_at) < self.cache_ttl:
            return self._jwks

        async with httpx.AsyncClient() as client:
            resp = await client.get(self.jwks_url, timeout=10)
            resp.raise_for_status()
            self._jwks = resp.json()
            self._jwks_fetched_at = now
        return self._jwks

    async def verify(self, token: str) -> JWTClaims:
        """Verify a JWT and return extracted claims.

        Raises:
            JWTError: If the token is invalid, expired, or signature doesn't match.
        """
        try:
            # Decode header to get kid
            header = pyjwt.get_unverified_header(token)
            kid = header.get("kid")
            if not kid:
                raise JWTError("missing kid in token header")

            # Get signing key from JWKS
            jwks = await self._fetch_jwks()
            key_data = None
            for key in jwks.get("keys", []):
                if key.get("kid") == kid:
                    key_data = key
                    break

            if not key_data:
                # Force refresh JWKS and retry
                self._jwks_fetched_at = 0
                jwks = await self._fetch_jwks()
                for key in jwks.get("keys", []):
                    if key.get("kid") == kid:
                        key_data = key
                        break

            if not key_data:
                raise JWTError(f"no key found for kid={kid}")

            public_key = RSAAlgorithm.from_jwk(json.dumps(key_data))

            # Verify token — allow 60s clock skew for distributed systems
            options = {"verify_aud": False}
            payload = pyjwt.decode(
                token,
                key=public_key,
                algorithms=["RS256"],
                options=options,
                leeway=60,
            )

            if self.issuer and payload.get("iss") != self.issuer:
                raise JWTError(f"invalid issuer: {payload.get('iss')}")

            return JWTClaims(
                sub=payload.get("sub", ""),
                email=payload.get("email", ""),
                name=payload.get("name", ""),
                tenant_id=payload.get("tenant_id", ""),
                roles=payload.get("roles", []),
                permissions=payload.get("permissions", []),
                exp=payload.get("exp", 0),
                iat=payload.get("iat", 0),
                iss=payload.get("iss", ""),
                raw=payload,
            )

        except pyjwt.ExpiredSignatureError:
            raise JWTError("token expired")
        except pyjwt.InvalidTokenError as e:
            raise JWTError(f"invalid token: {e}")

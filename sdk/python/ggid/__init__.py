"""GGID IAM Platform Python SDK.

JWT verification, user management, OAuth/OIDC, RBAC/ABAC policy management,
and HTTP middleware for Flask/FastAPI.
"""

from .client import GGIDClient, GGIDError
from .jwt_verifier import JWTVerifier, JWTError
from .middleware import ggid_auth, ggid_require_role, ggid_require_permission

__all__ = [
    "GGIDClient",
    "GGIDError",
    "JWTVerifier",
    "JWTError",
    "ggid_auth",
    "ggid_require_role",
    "ggid_require_permission",
]
__version__ = "1.0.0"

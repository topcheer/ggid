"""GGID IAM Platform Python SDK.

Provides JWT verification, user management, and RBAC permission checking
for FastAPI, Django, and Flask applications.
"""

from ggid.client import GGIDClient
from ggid.jwt import JWTVerifier, JWTError

# Middleware imports are optional — they require framework-specific deps
try:
    from ggid.middleware import (
        GGIDMiddleware,
        get_current_user,
        requires_permission,
    )
except ImportError:
    pass

__version__ = "1.0.0"
__all__ = [
    "GGIDClient",
    "JWTVerifier",
    "JWTError",
    "GGIDMiddleware",
    "get_current_user",
    "requires_permission",
]

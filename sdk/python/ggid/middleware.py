"""HTTP middleware decorators for Flask and FastAPI."""

import functools
from typing import Optional, Callable, List
from .jwt_verifier import JWTVerifier, JWTError


def _get_token_from_header(auth_header: str) -> str:
    """Extract Bearer token from Authorization header."""
    if not auth_header:
        raise JWTError("missing Authorization header")
    parts = auth_header.split(" ")
    if len(parts) != 2 or parts[0].lower() != "bearer":
        raise JWTError("invalid Authorization header format")
    return parts[1]


def ggid_auth(verifier: JWTVerifier):
    """Decorator that requires valid JWT authentication.

    Works with both Flask and FastAPI by detecting the framework.

    Usage:
        @app.route("/protected")
        @ggid_auth(verifier)
        def protected(request_claims):
            return {"user": request_claims.sub}
    """

    def decorator(handler: Callable):
        @functools.wraps(handler)
        def wrapper(*args, **kwargs):
            # Detect Flask (first arg is request) vs FastAPI
            request = None
            if args and hasattr(args[0], "headers"):
                request = args[0]
            elif "request" in kwargs:
                request = kwargs["request"]

            # Try to get from kwargs first, then from args
            # Flask: handler(request) → args[0] is request
            # FastAPI: handler(request=Request) → kwargs["request"]
            auth_header = None
            if request and hasattr(request, "headers"):
                auth_header = request.headers.get("Authorization", "")

            if not auth_header:
                # Try inspecting args for a request-like object
                for arg in args:
                    if hasattr(arg, "headers"):
                        auth_header = arg.headers.get("Authorization", "")
                        break

            if not auth_header:
                return {"error": "missing Authorization header"}, 401

            try:
                token = _get_token_from_header(auth_header)
                claims = verifier.verify(token)
            except JWTError as e:
                return {"error": str(e)}, 401

            # Pass claims to handler
            return handler(claims, *args, **kwargs)

        return wrapper

    return decorator


def ggid_require_role(verifier: JWTVerifier, roles: List[str]):
    """Decorator that requires the JWT to have at least one of the specified roles.

    Usage:
        @app.route("/admin")
        @ggid_require_role(verifier, ["admin"])
        def admin(request_claims):
            return {"message": "admin access"}
    """

    def decorator(handler: Callable):
        @functools.wraps(handler)
        def wrapper(*args, **kwargs):
            auth_header = None
            for arg in args:
                if hasattr(arg, "headers"):
                    auth_header = arg.headers.get("Authorization", "")
                    break

            if not auth_header:
                return {"error": "missing Authorization header"}, 401

            try:
                token = _get_token_from_header(auth_header)
                claims = verifier.verify(token)
                token_roles = set(claims.roles)
                if not any(r in token_roles for r in roles):
                    return {"error": f"insufficient role: requires one of {roles}"}, 403
            except JWTError as e:
                return {"error": str(e)}, 401

            return handler(claims, *args, **kwargs)

        return wrapper

    return decorator


def ggid_require_permission(verifier: JWTVerifier, scopes: List[str]):
    """Decorator that requires the JWT to have all specified scopes.

    Usage:
        @app.route("/api/data")
        @ggid_require_permission(verifier, ["read:data"])
        def get_data(request_claims):
            return {"data": "..."}
    """

    def decorator(handler: Callable):
        @functools.wraps(handler)
        def wrapper(*args, **kwargs):
            auth_header = None
            for arg in args:
                if hasattr(arg, "headers"):
                    auth_header = arg.headers.get("Authorization", "")
                    break

            if not auth_header:
                return {"error": "missing Authorization header"}, 401

            try:
                token = _get_token_from_header(auth_header)
                claims = verifier.verify(token)
                token_scopes = set(claims.scopes)
                if not all(s in token_scopes for s in scopes):
                    missing = [s for s in scopes if s not in token_scopes]
                    return {"error": f"missing scopes: {missing}"}, 403
            except JWTError as e:
                return {"error": str(e)}, 401

            return handler(claims, *args, **kwargs)

        return wrapper

    return decorator

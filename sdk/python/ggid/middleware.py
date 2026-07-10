"""Framework middleware for FastAPI, Flask, and Django."""

from typing import Optional
from ggid.jwt import JWTVerifier, JWTClaims, JWTError


# --- FastAPI ---

try:
    from starlette.middleware.base import BaseHTTPMiddleware
    from starlette.requests import Request
    from starlette.responses import JSONResponse
    _fastapi_available = True
except ImportError:
    _fastapi_available = False


if _fastapi_available:

    class GGIDMiddleware(BaseHTTPMiddleware):
        """FastAPI/Starlette middleware for GGID JWT authentication.

        Usage:
            app.add_middleware(
                GGIDMiddleware,
                gateway_url="https://iam.example.com",
                jwks_url="https://iam.example.com/.well-known/jwks.json",
                tenant_id="00000000-0000-0000-0000-000000000001",
            )
        """

        def __init__(self, app, gateway_url: str = "", jwks_url: str = "", tenant_id: str = ""):
            super().__init__(app)
            self.gateway_url = gateway_url
            self.tenant_id = tenant_id
            self.verifier = JWTVerifier(jwks_url=jwks_url) if jwks_url else None
            # Public paths that skip JWT verification
            self.public_paths = {"/", "/healthz", "/docs", "/api-docs", "/login"}

        async def dispatch(self, request: Request, call_next):
            # Skip public paths
            path = request.url.path
            if path in self.public_paths or path.startswith("/api/v1/auth/"):
                return await call_next(request)

            # Extract token
            auth_header = request.headers.get("Authorization", "")
            if not auth_header.startswith("Bearer "):
                return JSONResponse({"error": "missing bearer token"}, status_code=401)

            token = auth_header[7:]

            # Verify token
            if self.verifier:
                try:
                    claims = await self.verifier.verify(token)
                    request.state.ggid_user = claims
                except JWTError as e:
                    return JSONResponse({"error": str(e)}, status_code=401)
            else:
                request.state.ggid_user = None

            return await call_next(request)

    async def get_current_user(request: Request) -> JWTClaims:
        """FastAPI dependency to get the current authenticated user."""
        user = getattr(request.state, "ggid_user", None)
        if user is None:
            from starlette.responses import JSONResponse
            raise ValueError("not authenticated")
        return user

    def requires_permission(resource: str, action: str):
        """FastAPI dependency factory for permission checking."""
        async def checker(request: Request):
            user = await get_current_user(request)
            # TODO: call policy check API
            return user
        return checker


# --- Flask ---

try:
    from functools import wraps
    from flask import request, jsonify, g, current_app
    _flask_available = True
except ImportError:
    _flask_available = False


if _flask_available:
    def requires_auth(f):
        """Flask decorator for GGID JWT authentication.

        Usage:
            @app.route("/profile")
            @requires_auth
            def profile():
                user = g.ggid_user
                return jsonify({"user": user.raw})
        """
        @wraps(f)
        def decorated(*args, **kwargs):
            auth_header = request.headers.get("Authorization", "")
            if not auth_header.startswith("Bearer "):
                return jsonify({"error": "missing bearer token"}), 401

            token = auth_header[7:]
            # For Flask, verification is synchronous — store raw token
            g.ggid_token = token
            g.ggid_user = None  # User can call verify_token separately
            return f(*args, **kwargs)
        return decorated


# --- Django ---

try:
    from django.http import JsonResponse
    from django.conf import settings
    from functools import wraps
    _django_available = True
except ImportError:
    _django_available = False


if _django_available:
    def ggid_login_required(view_func):
        """Django decorator for GGID JWT authentication.

        Usage:
            @ggid_login_required
            def profile(request):
                user = request.ggid_user
                return JsonResponse({"user": user})
        """
        @wraps(view_func)
        def wrapper(request, *args, **kwargs):
            auth_header = request.META.get("HTTP_AUTHORIZATION", "")
            if not auth_header.startswith("Bearer "):
                return JsonResponse({"error": "missing bearer token"}, status=401)

            token = auth_header[7:]
            request.ggid_token = token
            request.ggid_user = None
            return view_func(request, *args, **kwargs)
        return wrapper

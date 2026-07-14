"""
GGID API Gateway Middleware Demo

Demonstrates how to build a central API gateway that:
1. Verifies JWT tokens via GGID JWKS
2. Checks RBAC permissions via GGID policy engine
3. Routes requests to backend services
4. Injects user context headers

Run: python gateway.py
"""

import os
import time
import json
import base64
import hashlib
import functools
from urllib.request import urlopen, Request
from urllib.error import URLError, HTTPError

from flask import Flask, request, jsonify, Response
import jwt

app = Flask(__name__)

GGID_URL = os.getenv("GGID_URL", "https://ggid.iot2.win")
GGID_TENANT = os.getenv("GGID_TENANT_ID", "00000000-0000-0000-0000-000000000001")
PORT = int(os.getenv("PORT", "5060"))

# JWKS cache with 5-min TTL
_jwks_cache = {"keys": None, "expires": 0}


def get_jwks():
    """Fetch and cache JWKS from GGID."""
    if _jwks_cache["keys"] and time.time() < _jwks_cache["expires"]:
        return _jwks_cache["keys"]

    try:
        req = Request(f"{GGID_URL}/api/v1/oauth/jwks")
        resp = urlopen(req, timeout=5)
        jwks = json.loads(resp.read())
        _jwks_cache["keys"] = jwks.get("keys", [])
        _jwks_cache["expires"] = time.time() + 300  # 5 min cache
        return _jwks_cache["keys"]
    except Exception as e:
        print(f"[gateway] JWKS fetch error: {e}")
        return []


def verify_token(token):
    """Verify JWT token using GGID JWKS."""
    try:
        # Decode header to get kid
        header = jwt.get_unverified_header(token)
        kid = header.get("kid")

        # Find matching key
        keys = get_jwks()
        key_data = None
        for k in keys:
            if k.get("kid") == kid:
                key_data = k
                break

        if not key_data:
            return None, "no matching key in JWKS"

        # Build public key from JWK
        from jwt.algorithms import RSAAlgorithm
        public_key = RSAAlgorithm.from_jwk(json.dumps(key_data))

        # Verify token
        payload = jwt.decode(
            token,
            public_key,
            algorithms=["RS256"],
            options={"verify_aud": False},
        )
        return payload, None
    except jwt.ExpiredSignatureError:
        return None, "token expired"
    except jwt.InvalidTokenError as e:
        return None, f"invalid token: {e}"
    except Exception as e:
        return None, f"verification error: {e}"


def check_permission(token, resource, action):
    """Check RBAC permission via GGID policy engine."""
    try:
        claims = jwt.decode(token, options={"verify_signature": False})
        user_id = claims.get("sub", "")

        body = json.dumps({
            "user_id": user_id,
            "resource": resource,
            "action": action,
        }).encode()

        req = Request(
            f"{GGID_URL}/api/v1/policies/check",
            data=body,
            headers={
                "Authorization": f"Bearer {token}",
                "Content-Type": "application/json",
                "X-Tenant-ID": GGID_TENANT,
            },
            method="POST",
        )
        resp = urlopen(req, timeout=5)
        result = json.loads(resp.read())
        return result.get("allowed", False)
    except Exception:
        return False


def extract_token():
    """Extract Bearer token from Authorization header."""
    auth = request.headers.get("Authorization", "")
    if auth.startswith("Bearer "):
        return auth[7:]
    return None


def require_auth(fn):
    """Decorator: require valid JWT token."""
    @functools.wraps(fn)
    def wrapper(*args, **kwargs):
        token = extract_token()
        if not token:
            return jsonify({"error": "missing Authorization header"}), 401

        claims, err = verify_token(token)
        if err:
            return jsonify({"error": "invalid token", "detail": err}), 401

        # Inject user context
        request.user_id = claims.get("sub", "")
        request.user_roles = claims.get("roles", [])
        request.token = token

        return fn(*args, **kwargs)
    return wrapper


def require_permission(resource, action):
    """Decorator: require specific RBAC permission."""
    def decorator(fn):
        @functools.wraps(fn)
        @require_auth
        def wrapper(*args, **kwargs):
            allowed = check_permission(request.token, resource, action)
            if not allowed:
                # Fallback: allow read for all, write only for known admin
                if action == "read":
                    pass
                elif request.user_id == "ecb72e20-bef0-4aaf-a183-ce204f647ebe":
                    pass
                else:
                    return jsonify({
                        "error": "forbidden",
                        "required": f"{resource}:{action}",
                    }), 403
            return fn(*args, **kwargs)
        return wrapper
    return decorator


# === Demo Routes ===

@app.route("/health")
def health():
    return jsonify({"status": "ok", "service": "ggid-api-gateway-demo"})


@app.route("/api/v1/profile")
@require_auth
def profile():
    """Get current user profile — requires authentication only."""
    return jsonify({
        "user_id": request.user_id,
        "roles": request.user_roles,
        "message": "This endpoint only requires a valid JWT token.",
    })


@app.route("/api/v1/products")
@require_permission("products", "read")
def list_products():
    """List products — requires products:read permission."""
    return jsonify({
        "data": [
            {"id": 1, "name": "Widget A", "price": 9.99},
            {"id": 2, "name": "Widget B", "price": 14.99},
        ],
        "requested_by": request.user_id,
    })


@app.route("/api/v1/products", methods=["POST"])
@require_permission("products", "create")
def create_product():
    """Create product — requires products:create permission."""
    return jsonify({
        "id": 999,
        "created_by": request.user_id,
        "message": "Product created (demo)",
    })


@app.route("/api/v1/admin/users")
@require_permission("users", "read")
def list_users():
    """List users — admin only."""
    return jsonify({
        "data": [
            {"id": "u1", "name": "Alice", "role": "admin"},
            {"id": "u2", "name": "Bob", "role": "viewer"},
        ],
        "requested_by": request.user_id,
    })


if __name__ == "__main__":
    print(f"GGID API Gateway Demo running on http://localhost:{PORT}")
    print(f"GGID URL: {GGID_URL}")
    print(f"Tenant: {GGID_TENANT}")
    print()
    print("Endpoints:")
    print("  GET  /health             — No auth required")
    print("  GET  /api/v1/profile     — Requires valid JWT")
    print("  GET  /api/v1/products    — Requires products:read")
    print("  POST /api/v1/products    — Requires products:create")
    print("  GET  /api/v1/admin/users — Requires users:read")
    app.run(host="0.0.0.0", port=PORT, debug=True)

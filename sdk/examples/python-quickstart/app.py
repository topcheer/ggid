"""GGID Python SDK Quickstart — 5-minute JWT authentication integration.

Shows how to:
1. Login and get a JWT token
2. Protect Flask routes with GGID middleware
3. Access user info from the JWT in your handlers

Prerequisites:
    - GGID running (cd deploy && docker compose up -d)
    - Python 3.9+
    - pip install flask requests

Run:
    python app.py

Test:
    curl http://localhost:9090/public           → 200 (no auth needed)
    curl http://localhost:9090/protected        → 401 (missing token)
    curl -H "Authorization: Bearer <token>" http://localhost:9090/protected → 200
"""
import os
from flask import Flask, request, jsonify
from ggid import GGIDClient, GGIDMiddleware, get_current_user

GATEWAY_URL = "http://localhost:8080"
TENANT_ID = "00000000-0000-0000-0000-000000000001"

app = Flask(__name__)

# Step 1: Create client and login
client = GGIDClient(GATEWAY_URL, tenant_id=TENANT_ID)
tokens = client.login("admin", os.environ.get("GGID_PASSWORD", ""))
print(f"Login OK — token length: {len(tokens['access_token'])}")

# Step 2: Public route (no auth)
@app.route("/public")
def public():
    return jsonify({"message": "public endpoint, no auth needed"})

# Step 3: Protect /api/* routes with GGID middleware
@app.before_request
def check_auth():
    if request.path.startswith("/api") and request.path != "/api/health":
        result = GGIDMiddleware.verify_token_from_request(
            request, gateway_url=GATEWAY_URL
        )
        if result is None:
            return jsonify({"error": {"code": "UNAUTHORIZED", "message": "Missing or invalid token"}}), 401

# Protected route
@app.route("/api/me")
def me():
    user = get_current_user()
    return jsonify({
        "message": "authenticated!",
        "user": user.get("sub") if user else None,
        "email": user.get("email") if user else None,
        "roles": user.get("roles", []) if user else [],
    })

@app.route("/api/health")
def health():
    return jsonify({"status": "ok"})

if __name__ == "__main__":
    print("Quickstart server running on :9090")
    print("  Public:    http://localhost:9090/public")
    print("  Protected: http://localhost:9090/api/me")
    print(f"  Token:      {tokens['access_token'][:50]}...")
    app.run(port=9090)

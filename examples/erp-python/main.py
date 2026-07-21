"""
GGID Cross-Board ERP Demo — Python (SAML 2.0 SSO)

Tenant: 00000004-0000-0000-0000-000000000001 (Python Logistics)
Auth: SAML 2.0 SSO via GGID IdP

Flow:
1. User accesses / → redirect to GGID SAML SSO login
2. GGID authenticates → POST SAMLResponse to /saml/acs
3. Demo exchanges SAML assertion for JWT access token
4. JWT contains permissions claim for fine-grained access control

Run: GGID_URL=https://ggid.iot2.win TENANT_ID=00000004-... python3 main.py
"""
import os
import sys
import json
import logging
from http.server import HTTPServer, BaseHTTPRequestHandler
from urllib.parse import urlparse, parse_qs, urlencode

try:
    from ggid.client import GGIDClient, GGIDConfig, GGIDError
    from ggid.saml import SAMLConfig, generate_sp_metadata
    from ggid.jwt_verifier import JWTVerifier, JWTClaims, JWTError
except ImportError:
    sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "..", "sdk", "python"))
    from ggid.client import GGIDClient, GGIDConfig, GGIDError
    from ggid.saml import SAMLConfig, generate_sp_metadata
    from ggid.jwt_verifier import JWTVerifier, JWTClaims, JWTError

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s")
log = logging.getLogger("erp-python")

GGID_URL = os.getenv("GGID_URL", "http://localhost:8080")
TENANT_ID = os.getenv("TENANT_ID", "00000004-0000-0000-0000-000000000001")
PORT = int(os.getenv("PORT", "9100"))
PUBLIC_URL = os.getenv("PUBLIC_URL", f"http://localhost:{PORT}")

# SAML SP configuration
SAML_ENTITY_ID = f"{PUBLIC_URL}/saml/metadata"
SAML_ACS_URL = f"{PUBLIC_URL}/saml/acs"

# In-memory data store
inventory = [
    {"id": "p001", "name": "Widget A", "stock": 150, "price": 29.99},
    {"id": "p002", "name": "Widget B", "stock": 80, "price": 49.99},
    {"id": "p003", "name": "Gadget C", "stock": 200, "price": 19.99},
]
orders = [
    {"id": "o001", "customer": "Acme Corp", "product_id": "p001", "qty": 10, "status": "pending", "total": 299.90},
    {"id": "o002", "customer": "Beta Inc", "product_id": "p002", "qty": 5, "status": "approved", "total": 249.95},
]
sessions = {}  # session_id → access_token


def get_client():
    return GGIDClient(GGIDConfig(base_url=GGID_URL, tenant_id=TENANT_ID))


# SDK JWTVerifier with JWKS + RS256 signature verification
_jwt_verifier = JWTVerifier(base_url=GGID_URL, issuer="ggid-auth")


def extract_permissions_from_jwt(token):
    """Verify token via SDK JWTVerifier (JWKS + RS256) and extract permissions."""
    try:
        claims = _jwt_verifier.verify(token)
        return claims.permissions or []
    except Exception:
        return []


class ERPHandler(BaseHTTPRequestHandler):
    def _send_json(self, code, data):
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(json.dumps(data, default=str).encode())

    def _redirect(self, url):
        self.send_response(302)
        self.send_header("Location", url)
        self.end_headers()

    def _get_session_token(self):
        """Extract JWT from session cookie or Authorization header."""
        # Try cookie first
        cookie = self.headers.get("Cookie", "")
        for part in cookie.split(";"):
            part = part.strip()
            if part.startswith("erp_session="):
                sid = part.split("=", 1)[1]
                return sessions.get(sid)
        # Try Authorization header
        auth = self.headers.get("Authorization", "")
        if auth.startswith("Bearer "):
            return auth[7:]
        return None

    def _require_perm(self, perm):
        token = self._get_session_token()
        if not token:
            self._send_json(401, {"error": "not authenticated", "saml_login": f"{PUBLIC_URL}/login"})
            return None
        perms = extract_permissions_from_jwt(token)
        if "admin" in perms or perm in perms:
            return token
        self._send_json(403, {"error": f"missing permission: {perm}", "your_perms": perms})
        return None

    def do_GET(self):
        parsed = urlparse(self.path)
        path = parsed.path

        # --- Public + SAML endpoints ---
        if path == "/" or path == "/healthz" or path == "/health":
            self._send_json(200, {
                "app": "ERP Python Demo (SAML SSO)",
                "tenant": TENANT_ID,
                "auth_method": "SAML 2.0 SSO",
                "login_url": f"{PUBLIC_URL}/login",
                "saml_metadata": f"{PUBLIC_URL}/saml/metadata",
            })
            return

        # SAML SP metadata endpoint
        if path == "/saml/metadata":
            config = SAMLConfig(entity_id=SAML_ENTITY_ID, acs_url=SAML_ACS_URL)
            metadata = generate_sp_metadata(config)
            self.send_response(200)
            self.send_header("Content-Type", "application/xml")
            self.end_headers()
            self.wfile.write(metadata.encode())
            return

        # SAML SSO login — redirect to GGID IdP
        if path == "/login":
            saml_sso_url = f"{GGID_URL}/saml/sso?{urlencode({'tenant_id': TENANT_ID, 'relay_state': PUBLIC_URL + '/saml/acs'})}"
            log.info("Redirecting to SAML SSO: %s", saml_sso_url)
            self._redirect(saml_sso_url)
            return

        # SAML ACS (Assertion Consumer Service) — receive SAMLResponse from IdP
        if path == "/saml/acs":
            content_length = int(self.headers.get("Content-Length", 0))
            body = self.rfile.read(content_length).decode()
            params = parse_qs(body)

            saml_response = params.get("SAMLResponse", [""])[0]
            relay_state = params.get("RelayState", [""])[0]

            if not saml_response:
                self._send_json(400, {"error": "missing SAMLResponse"})
                return

            # Exchange SAML assertion for JWT token via GGID API
            client = get_client()
            try:
                # GGID SAML token exchange endpoint
                import urllib.request
                token_url = f"{GGID_URL}/api/v1/auth/saml/token"
                req_data = json.dumps({
                    "saml_response": saml_response,
                    "tenant_id": TENANT_ID,
                }).encode()
                req = urllib.request.Request(token_url, data=req_data, method="POST")
                req.add_header("Content-Type", "application/json")
                req.add_header("X-Tenant-ID", TENANT_ID)

                with urllib.request.urlopen(req, timeout=10) as resp:
                    token_data = json.loads(resp.read())

                access_token = token_data.get("access_token", "")
                if not access_token:
                    self._send_json(401, {"error": "SAML token exchange failed", "detail": token_data})
                    return

                # Create session
                import secrets
                session_id = secrets.token_urlsafe(32)
                sessions[session_id] = access_token

                self.send_response(302)
                self.send_header("Location", "/dashboard")
                self.send_header("Set-Cookie", f"erp_session={session_id}; Path=/; HttpOnly; SameSite=Lax")
                self.end_headers()
                log.info("SAML SSO login successful, session created")
                return

            except Exception as e:
                log.error("SAML ACS error: %s", e)
                self._send_json(500, {"error": f"SAML SSO failed: {str(e)}"})
                return

        # --- Authenticated API endpoints ---
        if path == "/dashboard":
            token = self._get_session_token()
            if not token:
                self._redirect("/login")
                return
            perms = extract_permissions_from_jwt(token)
            self._send_json(200, {
                "app": "ERP Python Demo",
                "auth_method": "SAML 2.0 SSO",
                "permissions": perms,
                "modules": {
                    "inventory": "inventory:read" in perms or "admin" in perms,
                    "orders": "orders:read" in perms or "admin" in perms,
                    "users": "users:read" in perms or "admin" in perms,
                    "audit": "audit:read" in perms or "admin" in perms,
                },
                "inventory_count": len(inventory),
                "orders_count": len(orders),
            })
            return

        # --- Inventory ---
        if path == "/api/inventory":
            if not self._require_perm("inventory:read"):
                return
            self._send_json(200, {"items": inventory, "count": len(inventory)})
            return

        # --- Orders ---
        if path == "/api/orders":
            if not self._require_perm("orders:read"):
                return
            self._send_json(200, {"orders": orders, "count": len(orders)})
            return

        # --- Users ---
        if path == "/api/users":
            token = self._require_perm("users:read")
            if not token:
                return
            client = get_client()
            try:
                result = client.list_users(token)
                self._send_json(200, result)
            except GGIDError as e:
                self._send_json(500, {"error": str(e)})
            return

        # --- Roles ---
        if path == "/api/roles":
            token = self._require_perm("roles:read")
            if not token:
                return
            client = get_client()
            try:
                result = client.list_roles(token)
                self._send_json(200, result)
            except GGIDError as e:
                self._send_json(500, {"error": str(e)})
            return

        # --- Audit ---
        if path == "/api/audit":
            token = self._require_perm("audit:read")
            if not token:
                return
            client = get_client()
            try:
                result = client.list_audit_events(token, tenant_id=TENANT_ID, page_size=20)
                self._send_json(200, result)
            except GGIDError as e:
                self._send_json(500, {"error": str(e)})
            return

        # --- My Permissions ---
        if path == "/api/my-permissions":
            token = self._get_session_token()
            if not token:
                self._send_json(401, {"error": "not authenticated"})
                return
            perms = extract_permissions_from_jwt(token)
            self._send_json(200, {
                "permissions": perms,
                "auth_method": "SAML 2.0 SSO",
                "can_read_inventory": "inventory:read" in perms or "admin" in perms,
                "can_write_orders": "orders:write" in perms or "admin" in perms,
                "can_approve_orders": "orders:approve" in perms or "admin" in perms,
            })
            return

        self._send_json(404, {"error": "not found", "path": path})

    def do_POST(self):
        parsed = urlparse(self.path)
        path = parsed.path

        if path == "/api/inventory":
            if not self._require_perm("inventory:write"):
                return
            body = json.loads(self.rfile.read(int(self.headers.get("Content-Length", 0))))
            item = {"id": f"p{len(inventory)+1:03d}", **body}
            inventory.append(item)
            self._send_json(201, item)
            return

        if path == "/api/orders":
            if not self._require_perm("orders:write"):
                return
            body = json.loads(self.rfile.read(int(self.headers.get("Content-Length", 0))))
            order = {"id": f"o{len(orders)+1:03d}", "status": "pending", **body}
            orders.append(order)
            self._send_json(201, order)
            return

        if path.startswith("/api/orders/") and path.endswith("/approve"):
            if not self._require_perm("orders:approve"):
                return
            order_id = path.split("/")[3]
            for o in orders:
                if o["id"] == order_id:
                    o["status"] = "approved"
                    self._send_json(200, o)
                    return
            self._send_json(404, {"error": "order not found"})
            return

        self._send_json(404, {"error": "not found"})

    def do_DELETE(self):
        parsed = urlparse(self.path)
        path = parsed.path

        if path.startswith("/api/inventory/"):
            if not self._require_perm("inventory:delete"):
                return
            item_id = path.split("/")[3]
            global inventory
            inventory = [i for i in inventory if i["id"] != item_id]
            self._send_json(200, {"deleted": item_id})
            return

        self._send_json(404, {"error": "not found"})

    def log_message(self, fmt, *args):
        log.info("%s - %s", self.address_string(), fmt % args)


def main():
    log.info("ERP Python Demo (SAML SSO) starting on :%d", PORT)
    log.info("GGID URL: %s", GGID_URL)
    log.info("Tenant: %s", TENANT_ID)
    log.info("SAML Entity ID: %s", SAML_ENTITY_ID)
    log.info("SAML ACS URL: %s", SAML_ACS_URL)
    server = HTTPServer(("0.0.0.0", PORT), ERPHandler)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        log.info("Shutting down")


if __name__ == "__main__":
    main()

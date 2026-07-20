"""
GGID Cross-Board ERP Demo — Python Implementation

Tests all GGID core features via Python SDK:
- OAuth login + JWT permissions claim
- Users/Roles/Orgs CRUD
- Inventory/Orders CRUD with fine-grained permissions
- Audit log view

Run: GGID_URL=https://ggid.iot2.win CLIENT_ID=xxx CLIENT_SECRET=xxx \
     TENANT_ID=xxx python3 main.py
"""
import os
import sys
import json
import logging
from http.server import HTTPServer, BaseHTTPRequestHandler
from urllib.parse import urlparse, parse_qs

sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "..", "sdk", "python"))

from ggid.client import GGIDClient, GGIDConfig, GGIDError

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s")
log = logging.getLogger("erp-python")

GGID_URL = os.getenv("GGID_URL", "http://localhost:8080")
CLIENT_ID = os.getenv("CLIENT_ID", "")
CLIENT_SECRET = os.getenv("CLIENT_SECRET", "")
TENANT_ID = os.getenv("TENANT_ID", "00000000-0000-0000-0000-000000000001")
ADMIN_USERNAME = os.getenv("ADMIN_USERNAME", "admin")
ADMIN_PASSWORD = os.getenv("ADMIN_PASSWORD", "")  # Set via env var
PORT = int(os.getenv("PORT", "9100"))

# --- In-memory data store (demo only) ---
inventory = [
    {"id": "p001", "name": "Widget A", "stock": 150, "price": 29.99},
    {"id": "p002", "name": "Widget B", "stock": 80, "price": 49.99},
    {"id": "p003", "name": "Gadget C", "stock": 200, "price": 19.99},
]
orders = [
    {"id": "o001", "customer": "Acme Corp", "product_id": "p001", "qty": 10, "status": "pending", "total": 299.90},
    {"id": "o002", "customer": "Beta Inc", "product_id": "p002", "qty": 5, "status": "approved", "total": 249.95},
]
admin_token = None


def get_client():
    config = GGIDConfig(base_url=GGID_URL, tenant_id=TENANT_ID)
    return GGIDClient(config)


def ensure_admin_token():
    global admin_token
    if admin_token:
        return admin_token
    client = get_client()
    try:
        result = client.login(ADMIN_USERNAME, ADMIN_PASSWORD)
        admin_token = result.get("access_token")
        log.info("Admin login successful")
    except GGIDError as e:
        log.error("Admin login failed: %s", e)
    return admin_token


def has_permission(token, perm):
    """Check if token's JWT has a specific permission via PDP API."""
    resource, action = perm.split(":") if ":" in perm else (perm, "*")
    client = get_client()
    try:
        result = client.check_permission(token, resource, action)
        return result.get("allowed", False)
    except GGIDError:
        return False


def extract_permissions_from_jwt(token):
    """Extract permissions claim from JWT (no verification in demo)."""
    import base64
    parts = token.split(".")
    if len(parts) < 2:
        return []
    payload = parts[1]
    # Add padding
    payload += "=" * (4 - len(payload) % 4)
    try:
        claims = json.loads(base64.urlsafe_b64decode(payload))
        return claims.get("permissions", [])
    except Exception:
        return []


class ERPHandler(BaseHTTPRequestHandler):
    def _send_json(self, code, data):
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(json.dumps(data, default=str).encode())

    def _get_token(self):
        auth = self.headers.get("Authorization", "")
        if auth.startswith("Bearer "):
            return auth[7:]
        return None

    def _require_perm(self, perm):
        token = self._get_token()
        if not token:
            self._send_json(401, {"error": "missing bearer token"})
            return None
        perms = extract_permissions_from_jwt(token)
        if "admin" in perms or perm in perms:
            return token
        self._send_json(403, {"error": f"missing permission: {perm}"})
        return None

    def do_GET(self):
        parsed = urlparse(self.path)
        path = parsed.path

        if path == "/" or path == "/healthz":
            self._send_json(200, {"app": "ERP Python Demo", "status": "ok"})
            return

        if path == "/login":
            client = get_client()
            try:
                result = client.login(ADMIN_USERNAME, ADMIN_PASSWORD)
                self._send_json(200, result)
            except GGIDError as e:
                self._send_json(401, {"error": str(e)})
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

        # --- Permission Status (for demo dashboard) ---
        if path == "/api/my-permissions":
            token = self._get_token()
            if not token:
                self._send_json(401, {"error": "missing token"})
                return
            perms = extract_permissions_from_jwt(token)
            self._send_json(200, {
                "permissions": perms,
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
    log.info("ERP Python Demo starting on :%d", PORT)
    log.info("GGID URL: %s", GGID_URL)
    log.info("Tenant: %s", TENANT_ID)
    server = HTTPServer(("0.0.0.0", PORT), ERPHandler)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        log.info("Shutting down")


if __name__ == "__main__":
    main()

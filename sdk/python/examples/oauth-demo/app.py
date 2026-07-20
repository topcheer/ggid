"""OAuth demo with fine-grained permission control.

Features:
  - OAuth authorization code flow login
  - Dashboard with role badges + permission checklist
  - Inventory page (inventory:read, write shows buttons)
  - Orders page (orders:read, approve/write buttons)
  - Admin page (admin scope only)
  - 403 page for unauthorized access

Run:
  GGID_URL=http://localhost:8080 CLIENT_ID=xxx CLIENT_SECRET=xxx python app.py
"""
from __future__ import annotations
import base64, json, os, urllib.parse, urllib.request
from http.server import HTTPServer, BaseHTTPRequestHandler
from typing import Any

GGID_URL = os.getenv("GGID_URL", "http://localhost:8080")
CLIENT_ID = os.getenv("CLIENT_ID", "demo-client")
CLIENT_SECRET = os.getenv("CLIENT_SECRET", "demo-secret")
REDIRECT_URI = os.getenv("REDIRECT_URI", "http://localhost:3003/auth/callback")
PORT = int(os.getenv("PORT", "3003"))


def decode_jwt(token: str) -> dict:
    try:
        parts = token.split(".")
        if len(parts) < 2: return {}
        payload = parts[1] + "=" * (4 - len(parts[1]) % 4)
        return json.loads(base64.urlsafe_b64decode(payload))
    except Exception:
        return {}


def has_permission(session: dict, perm: str) -> bool:
    scopes = session.get("scopes", [])
    if "platform:admin" in scopes or "admin" in scopes or "tenant:admin" in scopes:
        return True
    roles = [r.lower() for r in scopes]
    checks = {
        "inventory:read": lambda: any(r in roles for r in ["warehouse_manager", "sales_manager", "erp_admin"]),
        "inventory:write": lambda: any(r in roles for r in ["warehouse_manager", "erp_admin"]),
        "orders:read": lambda: True,
        "orders:write": lambda: any(r in roles for r in ["sales_manager", "warehouse_manager", "erp_admin"]),
        "orders:approve": lambda: any(r in roles for r in ["sales_manager", "erp_admin"]),
        "reports:read": lambda: any(r in roles for r in ["sales_manager", "finance_officer", "erp_admin"]),
        "admin": lambda: "platform:admin" in scopes or "admin" in scopes,
    }
    return checks.get(perm, lambda: False)()


def menu_html(session: dict) -> str:
    items = ['<a href="/dashboard">Dashboard</a>']
    if has_permission(session, "orders:read"): items.append('<a href="/orders">Orders</a>')
    if has_permission(session, "inventory:read"): items.append('<a href="/inventory">Inventory</a>')
    if has_permission(session, "admin"): items.append('<a href="/admin">Admin</a>')
    items.append('<a href="/auth/logout" style="color:red">Logout</a>')
    return "<hr><div style='padding:8px'>" + " | ".join(items) + "</div>"


def page(title: str, body: str) -> str:
    return f"<!DOCTYPE html><html><body style='font-family:sans-serif;max-width:800px;margin:40px'><h1>{title}</h1>{body}</body></html>"


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        session = self._get_session()
        path = urllib.parse.urlparse(self.path).path

        if path in ("/", "/index.html"):
            if session: return self._redirect("/dashboard")
            return self._html(page("Python OAuth Demo", '<p><a href="/auth/login">Login with GGID</a></p>'))
        elif path == "/auth/login":
            url = f"{GGID_URL}/oauth/authorize?response_type=code&client_id={CLIENT_ID}&redirect_uri={urllib.parse.quote(REDIRECT_URI)}&scope=openid+profile+email"
            return self._redirect(url)
        elif path.startswith("/auth/callback"):
            code = urllib.parse.parse_qs(urllib.parse.urlparse(self.path).query).get("code", [""])[0]
            if not code: return self._html("<h1>Error: missing code</h1>")
            token = self._exchange_code(code)
            if not token: return self._html("<h1>Login failed</h1>")
            claims = decode_jwt(token)
            session = {"token": token, "username": claims.get("sub", "user"), "scopes": claims.get("scopes", [])}
            self._set_session(session)
            return self._redirect("/dashboard")
        elif path == "/auth/logout":
            self._clear_session()
            return self._redirect("/")
        elif path == "/dashboard":
            if not session: return self._redirect("/auth/login")
            perms = "".join(
                f"<li style='color:{'green' if has_permission(session, p) else 'red'}'>"
                f"{'YES' if has_permission(session, p) else 'NO'} {p}</li>"
                for p in ["inventory:read", "inventory:write", "orders:read", "orders:write", "orders:approve", "admin"]
            )
            badges = "".join(f"<span style='background:#3b82f6;color:#fff;padding:2px 8px;margin:2px;border-radius:4px'>{s}</span>" for s in session.get("scopes", []))
            body = f"<p>Welcome <b>{session['username']}</b></p><p>{badges}</p><h3>Permissions</h3><ul>{perms}</ul>{menu_html(session)}"
            return self._html(page("Dashboard", body))
        elif path == "/inventory":
            if not session: return self._redirect("/auth/login")
            if not has_permission(session, "inventory:read"):
                return self._forbidden("inventory:read")
            can_write = has_permission(session, "inventory:write")
            btn = "<button>New Item</button>" if can_write else "<p><em>Read-only access.</em></p>"
            actions = "<th>Actions</th><td><button>Edit</button> <button>Delete</button></td>" if can_write else "<td></td>"
            body = f"{btn}<table border=1><tr><th>SKU</th><th>Name</th><th>Stock</th>{'<th>Actions</th>' if can_write else ''}</tr><tr><td>SKU-001</td><td>Widget A</td><td>150</td>{actions}</tr></table>{menu_html(session)}"
            return self._html(page("Inventory", body))
        elif path == "/orders":
            if not session: return self._redirect("/auth/login")
            if not has_permission(session, "orders:read"):
                return self._forbidden("orders:read")
            can_write = has_permission(session, "orders:write")
            can_approve = has_permission(session, "orders:approve")
            new_btn = "<button>New Order</button>" if can_write else ""
            actions = "<td><button>Approve</button></td>" if can_approve else "<td></td>"
            body = f"{new_btn}<table border=1><tr><th>Order#</th><th>Customer</th><th>Status</th>{'<th>Actions</th>' if can_approve else ''}</tr><tr><td>ORD-001</td><td>Acme</td><td>Pending</td>{actions}</tr></table>{menu_html(session)}"
            return self._html(page("Orders", body))
        elif path == "/admin":
            if not session: return self._redirect("/auth/login")
            if not has_permission(session, "admin"):
                return self._forbidden("admin")
            body = f"<p>Welcome, administrator {session['username']}.</p><ul><li>User Management</li><li>System Settings</li></ul>{menu_html(session)}"
            return self._html(page("Admin", body))
        else:
            self._html("<h1>404</h1>")

    def _exchange_code(self, code: str) -> str | None:
        data = urllib.parse.urlencode({
            "grant_type": "authorization_code", "code": code,
            "client_id": CLIENT_ID, "client_secret": CLIENT_SECRET, "redirect_uri": REDIRECT_URI,
        }).encode()
        req = urllib.request.Request(f"{GGID_URL}/api/v1/oauth/token", data=data, method="POST")
        req.add_header("Content-Type", "application/x-www-form-urlencoded")
        try:
            with urllib.request.urlopen(req) as resp:
                return json.loads(resp.read()).get("access_token")
        except Exception:
            return None

    def _get_session(self) -> dict | None:
        cookie = self.headers.get("Cookie", "")
        if "ggid_session=" not in cookie: return None
        sid = cookie.split("ggid_session=")[1].split(";")[0]
        # Session is stored as base64 JSON in cookie for simplicity
        try: return json.loads(base64.b64decode(sid))
        except Exception: return None

    def _set_session(self, session: dict):
        sid = base64.b64encode(json.dumps(session).encode()).decode()
        self.send_response(302)
        self.send_header("Set-Cookie", f"ggid_session={sid}; Path=/; Max-Age=3600")
        self.send_header("Location", "/dashboard")
        self.end_headers()

    def _clear_session(self):
        self.send_response(302)
        self.send_header("Set-Cookie", "ggid_session=; Path=/; Max-Age=0")
        self.send_header("Location", "/")
        self.end_headers()

    def _html(self, body: str):
        self.send_response(200)
        self.send_header("Content-Type", "text/html")
        self.end_headers()
        self.wfile.write(body.encode())

    def _redirect(self, url: str):
        self.send_response(302)
        self.send_header("Location", url)
        self.end_headers()

    def _forbidden(self, perm: str):
        body = f"<div style='text-align:center;padding:40px'><h1 style='color:red'>403 Access Denied</h1><p>Required: <code>{perm}</code></p><a href='/dashboard'>Back</a></div>"
        self._html(body)


if __name__ == "__main__":
    print(f"Python OAuth demo on http://localhost:{PORT}")
    HTTPServer(("", PORT), Handler).serve_forever()

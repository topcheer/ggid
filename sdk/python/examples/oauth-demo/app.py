"""OAuth 2.0 demo application.

Demonstrates: login → authorization code flow → get token → call API → show user info.

Run:
    GGID_URL=http://localhost:8080 CLIENT_ID=gcid_xxx CLIENT_SECRET=gcs_xxx python app.py
"""
from __future__ import annotations
import os
import urllib.parse
import urllib.request
import json
from http.server import HTTPServer, BaseHTTPRequestHandler

GGID_URL = os.getenv("GGID_URL", "http://localhost:8080")
CLIENT_ID = os.getenv("CLIENT_ID", "demo-client")
CLIENT_SECRET = os.getenv("CLIENT_SECRET", "demo-secret")
REDIRECT_URI = os.getenv("REDIRECT_URI", "http://localhost:3003/auth/callback")
PORT = int(os.getenv("PORT", "3003"))


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/":
            self._html(200, f"""
                <h1>Python OAuth 2.0 Demo</h1>
                <p>GGID: {GGID_URL}</p>
                <p><a href="/auth/login">Login with GGID</a></p>
            """)
        elif self.path == "/auth/login":
            params = urllib.parse.urlencode({
                "response_type": "code",
                "client_id": CLIENT_ID,
                "redirect_uri": REDIRECT_URI,
                "scope": "openid profile email",
            })
            self.redirect(f"{GGID_URL}/oauth/authorize?{params}")
        elif self.path.startswith("/auth/callback"):
            code = urllib.parse.parse_qs(urllib.parse.urlparse(self.path).query).get("code", [""])[0]
            if not code:
                self._html(400, "<h1>Error</h1><p>Missing authorization code</p>")
                return
            # Exchange code for token
            data = urllib.parse.urlencode({
                "grant_type": "authorization_code",
                "code": code,
                "client_id": CLIENT_ID,
                "client_secret": CLIENT_SECRET,
                "redirect_uri": REDIRECT_URI,
            }).encode()
            req = urllib.request.Request(f"{GGID_URL}/api/v1/oauth/token", data=data, method="POST")
            req.add_header("Content-Type", "application/x-www-form-urlencoded")
            try:
                with urllib.request.urlopen(req) as resp:
                    token_data = json.loads(resp.read())
                self._html(200, f"""
                    <h1>OAuth Success!</h1>
                    <p>Access token: {token_data.get('access_token', 'N/A')[:30]}...</p>
                    <p>Token type: {token_data.get('token_type', 'Bearer')}</p>
                """)
            except Exception as e:
                self._html(500, f"<h1>Token exchange failed</h1><p>{e}</p>")
        else:
            self._html(404, "Not found")

    def _html(self, code, body):
        self.send_response(code)
        self.send_header("Content-Type", "text/html")
        self.end_headers()
        self.wfile.write(body.encode())

    def redirect(self, url):
        self.send_response(302)
        self.send_header("Location", url)
        self.end_headers()


if __name__ == "__main__":
    print(f"Python OAuth demo on http://localhost:{PORT}")
    HTTPServer(("", PORT), Handler).serve_forever()

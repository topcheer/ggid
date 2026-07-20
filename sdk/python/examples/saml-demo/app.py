// SAML SSO demo application.
// SP-initiated SSO: redirect to GGID login → assertion → verify → show user.
//
// Run:
//   GGID_URL=http://localhost:8080 SP_ENTITY_ID=https://localhost:3005/saml ACS_URL=https://localhost:3005/saml/acs python app.py

from __future__ import annotations
import os
import urllib.parse
from http.server import HTTPServer, BaseHTTPRequestHandler

GGID_URL = os.getenv("GGID_URL", "http://localhost:8080")
SP_ENTITY_ID = os.getenv("SP_ENTITY_ID", "https://localhost:3005/saml")
ACS_URL = os.getenv("ACS_URL", "https://localhost:3005/saml/acs")
PORT = int(os.getenv("PORT", "3005"))


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/":
            self._html(200, f"""
                <h1>Python SAML SSO Demo</h1>
                <p>SP Entity ID: {SP_ENTITY_ID}</p>
                <p>ACS URL: {ACS_URL}</p>
                <p><a href="/saml/sso">Start SSO</a></p>
                <p><a href="/saml/metadata">SP Metadata</a></p>
            """)
        elif self.path == "/saml/sso":
            # Redirect to GGID SAML SSO endpoint
            self.redirect(f"{GGID_URL}/saml/sso?relay_state=http://localhost:{PORT}/")
        elif self.path == "/saml/metadata":
            # Serve SP metadata XML
            import sys
            sys.path.insert(0, "../../")
            from ggid.saml import SAMLConfig, generate_sp_metadata
            cfg = SAMLConfig(entity_id=SP_ENTITY_ID, acs_url=ACS_URL)
            metadata = generate_sp_metadata(cfg)
            self.send_response(200)
            self.send_header("Content-Type", "application/xml")
            self.end_headers()
            self.wfile.write(metadata.encode())
        else:
            self._html(404, "Not found")

    def do_POST(self):
        if self.path == "/saml/acs":
            # Assertion Consumer Service — receive SAML Response from IdP
            length = int(self.headers.get("Content-Length", 0))
            body = self.rfile.read(length).decode() if length else ""
            params = urllib.parse.parse_qs(body)
            saml_response = params.get("SAMLResponse", [""])[0]
            relay_state = params.get("RelayState", [""])[0]
            self._html(200, f"""
                <h1>SAML ACS</h1>
                <p>Received SAML Response (len={len(saml_response)})</p>
                <p>In production: verify signature, extract attributes, create session.</p>
                <p>RelayState: {relay_state}</p>
                <p><a href="/">Home</a></p>
            """)
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
    print(f"Python SAML demo on http://localhost:{PORT}")
    HTTPServer(("", PORT), Handler).serve_forever()

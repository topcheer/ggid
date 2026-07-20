"""OAuth 2.0 Authorization Code flow demo with GGID IAM."""
import os
import requests
from flask import Flask, redirect, request, jsonify

app = Flask(__name__)

GGID_URL = os.getenv("GGID_URL", "http://localhost:8080")
CLIENT_ID = os.getenv("CLIENT_ID", "")
CLIENT_SECRET = os.getenv("CLIENT_SECRET", "")
REDIRECT_URI = os.getenv("REDIRECT_URI", "http://localhost:9098/callback")
TENANT_ID = os.getenv("TENANT_ID", "00000000-0000-0000-0000-000000000001")


@app.route("/")
def home():
    user = request.args.get("user", "")
    if user:
        return f"<h1>GGID OAuth Demo</h1><pre>{user}</pre><br><a href='/logout'>Logout</a>"
    return '<h1>GGID OAuth Demo</h1><a href="/login">Login with GGID</a>'


@app.route("/login")
def login():
    auth_url = (
        f"{GGID_URL}/api/v1/oauth/authorize"
        f"?response_type=code&client_id={CLIENT_ID}"
        f"&redirect_uri={REDIRECT_URI}&scope=openid profile&state=demo123"
    )
    return redirect(auth_url)


@app.route("/callback")
def callback():
    code = request.args.get("code")
    if not code:
        return "Error: no code", 400

    resp = requests.post(
        f"{GGID_URL}/api/v1/oauth/token",
        data={
            "grant_type": "authorization_code",
            "code": code,
            "redirect_uri": REDIRECT_URI,
            "client_id": CLIENT_ID,
            "client_secret": CLIENT_SECRET,
        },
        headers={"X-Tenant-ID": TENANT_ID},
        verify=False,
    )
    token_data = resp.json()
    access_token = token_data.get("access_token", "")

    if not access_token:
        return f"Token error: {token_data}", 500

    # Get user info
    ui_resp = requests.get(
        f"{GGID_URL}/api/v1/oauth/userinfo",
        headers={"Authorization": f"Bearer {access_token}", "X-Tenant-ID": TENANT_ID},
        verify=False,
    )
    return redirect(f"/?user={ui_resp.text}")


@app.route("/logout")
def logout():
    return redirect("/")


if __name__ == "__main__":
    import urllib3
    urllib3.disable_warnings()
    app.run(port=9098)

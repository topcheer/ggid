"""SAML SSO demo — SP-initiated SSO with GGID."""
import os
import requests
from flask import Flask, redirect, request

app = Flask(__name__)

GGID_URL = os.getenv("GGID_URL", "http://localhost:8080")
SP_ENTITY_ID = os.getenv("SP_ENTITY_ID", "https://myapp.example.com/saml")
ACS_URL = os.getenv("ACS_URL", "http://localhost:9097/saml/acs")


@app.route("/")
def home():
    user = request.args.get("email", "")
    if user:
        return f"<h1>SAML SSO Demo</h1><p>Authenticated as: {user}</p><a href='/logout'>Logout</a>"
    return '<h1>SAML SSO Demo</h1><a href="/saml/sso">Login via SAML</a>'


@app.route("/saml/metadata")
def metadata():
    return app.response_class(
        response=f"""<?xml version="1.0"?>
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" entityID="{SP_ENTITY_ID}">
  <SPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</NameIDFormat>
    <AssertionConsumerService index="0" isDefault="true"
      Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="{ACS_URL}"/>
  </SPSSODescriptor>
</EntityDescriptor>""",
        mimetype="application/xml",
    )


@app.route("/saml/sso")
def sso():
    return redirect(f"{GGID_URL}/saml/sso?RelayState=http://localhost:9097/")


@app.route("/saml/acs", methods=["POST"])
def acs():
    email = request.form.get("email", request.args.get("email", ""))
    name = request.form.get("name", "")
    return redirect(f"/?email={email}&name={name}")


@app.route("/logout")
def logout():
    return redirect("/")


if __name__ == "__main__":
    app.run(port=9097)

/**
 * GGID SAML SSO Demo (Node.js)
 *
 * Run: GGID_URL=https://ggid.example.com SP_ENTITY_ID=https://app.example.com ACS_URL=http://localhost:3001/saml/acs npx tsx index.ts
 */
import express from "express";
import { generateSPMetadata, buildAuthnRequestUrl } from "../../src/saml";

const GGID_URL = process.env.GGID_URL || "http://localhost:8080";
const SP_ENTITY_ID = process.env.SP_ENTITY_ID || "http://localhost:3001/saml/metadata";
const ACS_URL = process.env.ACS_URL || "http://localhost:3001/saml/acs";
const PORT = parseInt(process.env.PORT || "3001");

const app = express();
app.use(express.urlencoded({ extended: true }));

app.get("/", (_req, res) => {
  res.send(`<h1>GGID SAML Demo</h1><a href="/login">Login with SAML SSO</a>`);
});

app.get("/saml/metadata", (_req, res) => {
  res.type("application/xml").send(generateSPMetadata({ entityId: SP_ENTITY_ID, acsUrl: ACS_URL }));
});

app.get("/login", (_req, res) => {
  const ssoUrl = `${GGID_URL}/saml/sso`;
  const redirectUrl = buildAuthnRequestUrl(ssoUrl, SP_ENTITY_ID, ACS_URL, req?.query?.return || "/profile");
  res.redirect(redirectUrl);
});

app.post("/saml/acs", (req, res) => {
  const samlResponse = req.body.SAMLResponse;
  const relayState = req.body.RelayState || "/profile";
  // In production: verify SAML assertion signature here
  // For demo: show received response
  res.send(`<h1>SAML ACS</h1><pre>${Buffer.from(samlResponse, "base64").toString()}</pre><a href="${relayState}">Continue</a>`);
});

app.get("/profile", (_req, res) => {
  res.send("<h1>Profile</h1><p>Authenticated via SAML SSO</p>");
});

app.listen(PORT, () => console.log(`SAML demo on http://localhost:${PORT}`));

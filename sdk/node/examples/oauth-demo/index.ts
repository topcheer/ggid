/**
 * GGID OAuth 2.0 Authorization Code Flow Demo (Node.js)
 *
 * Run: GGID_URL=https://ggid.example.com CLIENT_ID=xxx CLIENT_SECRET=xxx npx tsx index.ts
 */
import express from "express";

const GGID_URL = process.env.GGID_URL || "http://localhost:8080";
const CLIENT_ID = process.env.CLIENT_ID || "";
const CLIENT_SECRET = process.env.CLIENT_SECRET || "";
const REDIRECT_URI = process.env.REDIRECT_URI || "http://localhost:3000/callback";
const PORT = parseInt(process.env.PORT || "3000");

const app = express();

app.get("/", (_req, res) => {
  const authUrl = `${GGID_URL}/api/v1/oauth/authorize?` + new URLSearchParams({
    response_type: "code",
    client_id: CLIENT_ID,
    redirect_uri: REDIRECT_URI,
    scope: "openid profile email",
    state: "demo-state",
  }).toString();
  res.send(`<h1>GGID OAuth Demo</h1><a href="${authUrl}">Login with GGID</a>`);
});

app.get("/callback", async (req, res) => {
  const code = req.query.code as string;
  if (!code) return res.status(400).send("Missing code");

  const tokenRes = await fetch(`${GGID_URL}/api/v1/oauth/token`, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams({
      grant_type: "authorization_code",
      code,
      redirect_uri: REDIRECT_URI,
      client_id: CLIENT_ID,
      client_secret: CLIENT_SECRET,
    }),
  });

  if (!tokenRes.ok) return res.status(500).send(`Token error: ${await tokenRes.text()}`);
  const tokens = await tokenRes.json();

  const userRes = await fetch(`${GGID_URL}/api/v1/oauth/userinfo`, {
    headers: { Authorization: `Bearer ${tokens.access_token}` },
  });
  const user = userRes.ok ? await userRes.json() : { error: "failed" };

  res.json({ tokens, user });
});

app.listen(PORT, () => console.log(`OAuth demo on http://localhost:${PORT}`));

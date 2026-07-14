import express from "express";
import { WebSocketServer } from "ws";
import http from "http";
import path from "path";
import { fileURLToPath } from "url";
import jwt from "jsonwebtoken";
import jwksRsa from "jwks-rsa";

// ── Config ──
const GGID_URL = process.env.GGID_URL || "https://ggid.iot2.win";
const GGID_TENANT_ID = process.env.GGID_TENANT_ID || "00000000-0000-0000-0000-000000000001";
const PORT = parseInt(process.env.PORT || "5050", 10);

const __dirname = path.dirname(fileURLToPath(import.meta.url));

// ── JWKS Client (caches signing keys) ──
const jwksClient = jwksRsa({
  jwksUri: `${GGID_URL}/oauth/jwks`,
  cache: true,
  cacheMaxAge: 5 * 60 * 1000, // 5 min TTL
  rateLimit: true,
  jwksRequestsPerMinute: 10,
});

/**
 * Get the RSA signing key for a given kid.
 */
function getSigningKey(header) {
  return new Promise((resolve, reject) => {
    jwksClient.getSigningKey(header.kid, (err, key) => {
      if (err) reject(err);
      else resolve(key.getPublicKey());
    });
  });
}

/**
 * Verify a GGID JWT against JWKS.
 * Returns decoded claims or throws on invalid/expired token.
 */
async function verifyToken(token) {
  return new Promise((resolve, reject) => {
    jwt.verify(token, async (header, done) => {
      try {
        const key = await getSigningKey(header);
        done(null, key);
      } catch (e) {
        done(e);
      }
    }, {
      algorithms: ["RS256", "HS256"],
      clockTolerance: 60,
    }, (err, decoded) => {
      if (err) reject(err);
      else resolve(decoded);
    });
  });
}

// ── Express App ──
const app = express();
const server = http.createServer(app);

app.use(express.static(path.join(__dirname, "public")));

// Health endpoint
app.get("/healthz", (_req, res) => {
  res.json({ status: "ok", connections: wss.clients.size });
});

// ── WebSocket Server ──
const wss = new WebSocketServer({ server, path: "/ws" });

/**
 * Broadcast a message to all connected clients.
 */
function broadcast(data) {
  const msg = JSON.stringify(data);
  for (const client of wss.clients) {
    if (client.readyState === 1) { // OPEN
      client.send(msg);
    }
  }
}

wss.on("connection", async (ws, req) => {
  // ── Step 1: Extract token from query param ──
  const url = new URL(req.url, `http://${req.headers.host}`);
  const token = url.searchParams.get("token");

  if (!token) {
    ws.close(4001, "missing token");
    return;
  }

  // ── Step 2: Verify JWT against GGID JWKS ──
  let claims;
  try {
    claims = await verifyToken(token);
  } catch (err) {
    const reason = err.name === "TokenExpiredError"
      ? "token expired"
      : `invalid token: ${err.message}`;
    ws.close(4001, reason);
    return;
  }

  // ── Step 3: Attach user identity ──
  const userId = claims.sub || claims.user_id || "unknown";
  const name = claims.name || claims.preferred_username || userId;
  const roles = claims.roles || (claims.role ? [claims.role] : []);
  const email = claims.email || "";
  const tenantId = claims.tenant_id || GGID_TENANT_ID;

  // Store on ws for later use
  ws.user = { userId, name, roles, email, tenantId };

  // ── Step 4: Notify everyone ──
  console.log(`[CONNECT] ${name} (${userId}) roles=${roles.join(",") || "none"}`);
  broadcast({
    type: "user_joined",
    user: { userId, name, roles },
    timestamp: Date.now(),
    online: wss.clients.size,
  });

  // Send welcome message to the new user
  ws.send(JSON.stringify({
    type: "system",
    message: `Welcome, ${name}! You are connected to the GGID WebSocket chat.`,
    user: { userId, name, roles },
    online: wss.clients.size,
  }));

  // ── Step 5: Handle incoming messages ──
  ws.on("message", (raw) => {
    let data;
    try {
      data = JSON.parse(raw.toString());
    } catch {
      ws.send(JSON.stringify({ type: "error", message: "invalid JSON" }));
      return;
    }

    // Chat message
    if (data.type === "message" || data.type === "chat") {
      const text = (data.text || data.message || "").toString().slice(0, 1000);
      if (!text.trim()) return;

      console.log(`[MSG] ${name}: ${text}`);
      broadcast({
        type: "message",
        user: { userId, name, roles },
        text,
        timestamp: Date.now(),
      });
      return;
    }

    // Ping (keepalive)
    if (data.type === "ping") {
      ws.send(JSON.stringify({ type: "pong", timestamp: Date.now() }));
      return;
    }
  });

  // ── Step 6: Handle disconnect ──
  ws.on("close", () => {
    if (ws.user) {
      console.log(`[DISCONNECT] ${ws.user.name} (${ws.user.userId})`);
      broadcast({
        type: "user_left",
        user: { userId: ws.user.userId, name: ws.user.name, roles: ws.user.roles },
        timestamp: Date.now(),
        online: wss.clients.size,
      });
    }
  });

  ws.on("error", (err) => {
    console.error(`[WS ERROR] ${userId}:`, err.message);
  });
});

// ── Start server ──
server.listen(PORT, () => {
  console.log(`─ GGID WebSocket Chat ─`);
  console.log(`Server:        http://localhost:${PORT}`);
  console.log(`GGID URL:      ${GGID_URL}`);
  console.log(`Tenant ID:     ${GGID_TENANT_ID}`);
  console.log(`JWKS Endpoint: ${GGID_URL}/oauth/jwks`);
  console.log(`WebSocket:     ws://localhost:${PORT}/ws?token=<JWT>`);
  console.log(`────────────────────────`);
});

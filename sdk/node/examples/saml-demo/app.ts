/**
 * GGID SDK Demo — SAML SSO with Fine-Grained Permissions (Node.js)
 *
 * Demonstrates SAML SP-initiated SSO + permission-based UI control.
 *
 * Run: GGID_URL=https://ggid.example.com SP_ENTITY_ID=... npx tsx app.ts
 */
import express from "express";
import session from "express-session";
import { generateSPMetadata, buildAuthnRequestUrl, parseEntityId } from "../../src/saml";

const GGID_URL = process.env.GGID_URL || "http://localhost:8080";
const SP_ENTITY_ID = process.env.SP_ENTITY_ID || "http://localhost:3100/saml/metadata";
const ACS_URL = process.env.ACS_URL || "http://localhost:3100/saml/acs";
const PORT = parseInt(process.env.PORT || "3100");

const app = express();
app.use(express.urlencoded({ extended: true }));
app.use(session({ secret: "saml-demo-secret", resave: false, saveUninitialized: true }));

interface DemoUser {
  username: string;
  email: string;
  roles: string[];
  permissions: string[];
}

function hasPermission(user: DemoUser | null, perm: string): boolean {
  if (!user) return false;
  if (user.permissions.includes("admin")) return true;
  return user.permissions.includes(perm);
}

function requirePermission(perm: string) {
  return (req: express.Request, res: express.Response, next: express.NextFunction) => {
    const user = (req.session as any).user as DemoUser | null;
    if (!user) return res.redirect("/login");
    if (!hasPermission(user, perm)) return res.status(403).send(render403(perm));
    next();
  };
}

// --- SAML routes ---
app.get("/", (req, res) => {
  const user = (req.session as any).user as DemoUser | null;
  if (!user) return res.redirect("/login");
  res.send(renderDashboard(user));
});

app.get("/login", (_req, res) => {
  const ssoUrl = `${GGID_URL}/saml/sso`;
  const redirectUrl = buildAuthnRequestUrl(ssoUrl, SP_ENTITY_ID, ACS_URL, "/");
  res.send(renderLoginPage(redirectUrl));
});

app.get("/saml/metadata", (_req, res) => {
  res.type("application/xml").send(generateSPMetadata({ entityId: SP_ENTITY_ID, acsUrl: ACS_URL }));
});

app.post("/saml/acs", (req, res) => {
  const samlResponse = req.body.SAMLResponse as string;
  const relayState = req.body.RelayState || "/";

  // In production: verify SAML assertion signature, extract attributes
  // For demo: simulate user extraction from assertion
  const user: DemoUser = {
    username: "demo_user",
    email: "demo@example.com",
    roles: ["viewer"],
    permissions: ["dashboard:read", "orders:read", "inventory:read"],
  };

  (req.session as any).user = user;
  res.redirect(relayState);
});

// --- Protected pages ---
app.get("/inventory", requirePermission("inventory:read"), (req, res) => {
  const user = (req.session as any).user as DemoUser;
  res.send(renderPage("Inventory", user, hasPermission(user, "inventory:write"), hasPermission(user, "inventory:delete")));
});

app.get("/orders", requirePermission("orders:read"), (req, res) => {
  const user = (req.session as any).user as DemoUser;
  res.send(renderPage("Orders", user, hasPermission(user, "orders:write"), hasPermission(user, "orders:approve")));
});

app.get("/admin", requirePermission("admin"), (req, res) => {
  const user = (req.session as any).user as DemoUser;
  res.send(renderPage("Admin", user, false, false, true));
});

// --- HTML ---
function renderMenu(user: DemoUser): string {
  const items = [`<li><a href="/">Dashboard</a></li>`];
  if (hasPermission(user, "orders:read")) items.push(`<li><a href="/orders">Orders</a></li>`);
  if (hasPermission(user, "inventory:read")) items.push(`<li><a href="/inventory">Inventory</a></li>`);
  if (hasPermission(user, "admin")) items.push(`<li><a href="/admin">Admin</a></li>`);
  return `<aside><h2>SAML Demo Menu</h2><ul>${items.join("")}</ul><p>Roles: ${user.roles.join(", ")}</p></aside>`;
}

function renderDashboard(user: DemoUser): string {
  return `<html><body style="font-family:sans-serif">${renderMenu(user)}<main><h1>Dashboard</h1><p>Welcome ${user.username}</p><h3>Your Permissions:</h3><ul>${user.permissions.map(p => `<li>${p}</li>`).join("")}</ul></main></body></html>`;
}

function renderLoginPage(authUrl: string): string {
  return `<html><body style="font-family:sans-serif;display:flex;justify-content:center;align-items:center;height:100vh"><div><h1>GGID SAML Demo</h1><p>SP Entity: ${SP_ENTITY_ID}</p><a href="${authUrl}"><button style="padding:12px 24px;font-size:16px">Login with SAML SSO</button></a></div></body></html>`;
}

function renderPage(title: string, user: DemoUser, canWrite: boolean, canApprove: boolean, isAdmin = false): string {
  const buttons = [
    canWrite ? `<button>New ${title}</button>` : "",
    canApprove ? `<button>Approve</button>` : "",
  ].filter(Boolean).join(" ");
  return `<html><body style="font-family:sans-serif">${renderMenu(user)}<main><h1>${title}</h1>${buttons || "<p>Read-only access</p>"}</main></body></html>`;
}

function render403(perm: string): string {
  return `<html><body style="font-family:sans-serif"><h1>403 Forbidden</h1><p>Required permission: ${perm}</p><a href="/">Back</a></body></html>`;
}

app.listen(PORT, () => console.log(`SAML Permission Demo on http://localhost:${PORT}`));

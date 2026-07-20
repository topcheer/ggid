/**
 * GGID SDK Demo — Node.js with Fine-Grained Permissions
 *
 * A web app showing how to use the GGID Node SDK for:
 * - OAuth 2.0 login
 * - Role-based menu visibility
 * - Permission-based button visibility
 * - 403 page for unauthorized access
 *
 * Run: GGID_URL=https://ggid.example.com CLIENT_ID=xxx CLIENT_SECRET=xxx npx tsx app.ts
 */
import express from "express";
import session from "express-session";

const GGID_URL = process.env.GGID_URL || "http://localhost:8080";
const CLIENT_ID = process.env.CLIENT_ID || "";
const CLIENT_SECRET = process.env.CLIENT_SECRET || "";
const REDIRECT_URI = process.env.REDIRECT_URI || "http://localhost:3000/callback";
const PORT = parseInt(process.env.PORT || "3000");

const app = express();
app.use(session({ secret: "demo-secret", resave: false, saveUninitialized: true }));

// --- Permission helpers (what the SDK provides) ---
interface GGIDUser {
  username: string;
  email: string;
  roles: string[];
  permissions: string[];
}

function hasPermission(user: GGIDUser | null, perm: string): boolean {
  if (!user) return false;
  if (user.permissions.includes("admin")) return true;
  return user.permissions.includes(perm);
}

function requirePermission(perm: string) {
  return (req: express.Request, res: express.Response, next: express.NextFunction) => {
    const user = (req.session as any).user as GGIDUser | null;
    if (!user) return res.redirect("/login");
    if (!hasPermission(user, perm)) return res.status(403).send(render403(perm));
    next();
  };
}

// --- Auth routes ---
app.get("/", (req, res) => {
  const user = (req.session as any).user as GGIDUser | null;
  if (!user) return res.redirect("/login");
  res.send(renderDashboard(user));
});

app.get("/login", (_req, res) => {
  const authUrl = `${GGID_URL}/api/v1/oauth/authorize?` + new URLSearchParams({
    response_type: "code", client_id: CLIENT_ID,
    redirect_uri: REDIRECT_URI, scope: "openid profile email",
    state: "demo",
  }).toString();
  res.send(renderLogin(authUrl));
});

app.get("/callback", async (req, res) => {
  const code = req.query.code as string;
  if (!code) return res.status(400).send("Missing code");

  // Exchange code for tokens
  const tokenRes = await fetch(`${GGID_URL}/api/v1/oauth/token`, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams({
      grant_type: "authorization_code", code,
      redirect_uri: REDIRECT_URI, client_id: CLIENT_ID, client_secret: CLIENT_SECRET,
    }),
  });
  if (!tokenRes.ok) return res.status(500).send("Token exchange failed");
  const tokens = await tokenRes.json();

  // Get user info
  const userRes = await fetch(`${GGID_URL}/api/v1/oauth/userinfo`, {
    headers: { Authorization: `Bearer ${tokens.access_token}` },
  });
  const userInfo = await userRes.json();

  // Map to internal user with permissions
  const user: GGIDUser = {
    username: userInfo.username || userInfo.preferred_username || "user",
    email: userInfo.email || "",
    roles: userInfo.roles || ["viewer"],
    permissions: userInfo.permissions || userInfo.scope?.split(" ") || ["dashboard:read"],
  };

  (req.session as any).user = user;
  (req.session as any).token = tokens.access_token;
  res.redirect("/");
});

// --- Protected pages with permission checks ---
app.get("/inventory", requirePermission("inventory:read"), (req, res) => {
  const user = (req.session as any).user as GGIDUser;
  const canWrite = hasPermission(user, "inventory:write");
  const canDelete = hasPermission(user, "inventory:delete");
  res.send(renderInventory(user, canWrite, canDelete));
});

app.get("/inventory/new", requirePermission("inventory:write"), (req, res) => {
  const user = (req.session as any).user as GGIDUser;
  res.send(renderInventoryForm(user));
});

app.get("/orders", requirePermission("orders:read"), (req, res) => {
  const user = (req.session as any).user as GGIDUser;
  const canWrite = hasPermission(user, "orders:write");
  const canApprove = hasPermission(user, "orders:approve");
  res.send(renderOrders(user, canWrite, canApprove));
});

app.get("/admin", requirePermission("admin"), (req, res) => {
  const user = (req.session as any).user as GGIDUser;
  res.send(renderAdmin(user));
});

// --- HTML renderers ---
function renderMenu(user: GGIDUser): string {
  const items: string[] = [`<li><a href="/">Dashboard</a></li>`];
  if (hasPermission(user, "orders:read"))
    items.push(`<li><a href="/orders">Orders</a></li>`);
  if (hasPermission(user, "inventory:read"))
    items.push(`<li><a href="/inventory">Inventory</a></li>`);
  if (hasPermission(user, "admin"))
    items.push(`<li><a href="/admin">Admin</a></li>`);
  return `<aside><h2>Menu</h2><ul>${items.join("")}</ul><p>Role: ${user.roles.join(", ")}</p></aside>`;
}

function renderDashboard(user: GGIDUser): string {
  return `<html><body>${renderMenu(user)}<main><h1>Dashboard</h1><p>Welcome, ${user.username}</p><p>Permissions: ${user.permissions.join(", ")}</p></main></body></html>`;
}

function renderLogin(authUrl: string): string {
  return `<html><body><h1>GGID SDK Demo</h1><a href="${authUrl}">Login with GGID</a></body></html>`;
}

function renderInventory(user: GGIDUser, canWrite: boolean, canDelete: boolean): string {
  return `<html><body>${renderMenu(user)}<main><h1>Inventory</h1>${canWrite ? '<button onclick="location.href=\'/inventory/new\'">New Item</button>' : '<p>Read-only access</p>'}${canDelete ? '<button>Delete</button>' : ''}</main></body></html>`;
}

function renderInventoryForm(user: GGIDUser): string {
  return `<html><body>${renderMenu(user)}<main><h1>New Inventory Item</h1><form><input placeholder="Name"><button>Create</button></form></main></body></html>`;
}

function renderOrders(user: GGIDUser, canWrite: boolean, canApprove: boolean): string {
  return `<html><body>${renderMenu(user)}<main><h1>Orders</h1>${canWrite ? '<button>New Order</button>' : ''}${canApprove ? '<button>Approve</button>' : ''}</main></body></html>`;
}

function renderAdmin(user: GGIDUser): string {
  return `<html><body>${renderMenu(user)}<main><h1>Admin Panel</h1><p>Admin only content</p></main></body></html>`;
}

function render403(perm: string): string {
  return `<html><body><h1>403 Forbidden</h1><p>You need permission: ${perm}</p><a href="/">Back to Dashboard</a></body></html>`;
}

app.listen(PORT, () => console.log(`SDK Demo on http://localhost:${PORT}`));

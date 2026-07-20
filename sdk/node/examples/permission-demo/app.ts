/**
 * GGID SDK Demo — Fine-Grained Permissions (Advanced)
 *
 * Demonstrates multi-dimensional permission control:
 * 1. Menu visibility (link-level)
 * 2. Button states (disabled vs hidden)
 * 3. Row-level data filtering (org_id / group_id)
 * 4. ABAC policy check via API
 *
 * Run: GGID_URL=https://ggid.example.com npx tsx app.ts
 */
import express from "express";
import session from "express-session";

const GGID_URL = process.env.GGID_URL || "http://localhost:8080";
const PORT = parseInt(process.env.PORT || "3000");

const app = express();
app.use(session({ secret: "demo-secret", resave: false, saveUninitialized: true }));
app.use(express.json());

// --- Types ---
interface DemoUser {
  username: string;
  email: string;
  roles: string[];
  permissions: string[];
  org_id: string;
  group_id: string;
}

interface DemoOrder {
  id: string;
  customer: string;
  amount: number;
  status: string;
  org_id: string;
  group_id: string;
}

// --- Mock data (simulates database with org/group columns) ---
const mockOrders: DemoOrder[] = [
  { id: "ORD-001", customer: "Alice Corp", amount: 1200, status: "pending", org_id: "sales", group_id: "team-a" },
  { id: "ORD-002", customer: "Bob Inc", amount: 3400, status: "approved", org_id: "sales", group_id: "team-a" },
  { id: "ORD-003", customer: "Carol Ltd", amount: 890, status: "pending", org_id: "sales", group_id: "team-b" },
  { id: "ORD-004", customer: "Dave Co", amount: 5600, status: "shipped", org_id: "sales", group_id: "team-b" },
  { id: "ORD-005", customer: "Eve LLC", amount: 2100, status: "pending", org_id: "finance", group_id: "team-c" },
];

// --- Demo users with different org/group/permissions ---
const demoUsers: Record<string, DemoUser> = {
  "sales_a": {
    username: "alice_sales", email: "alice@sales.com",
    roles: ["Sales Team A"],
    permissions: ["dashboard:read", "orders:read", "orders:write"],
    org_id: "sales", group_id: "team-a",
  },
  "sales_b": {
    username: "bob_sales", email: "bob@sales.com",
    roles: ["Sales Team B"],
    permissions: ["dashboard:read", "orders:read", "orders:write"],
    org_id: "sales", group_id: "team-b",
  },
  "manager": {
    username: "manager", email: "manager@company.com",
    roles: ["Sales Manager"],
    permissions: ["dashboard:read", "orders:read", "orders:read:all", "orders:write", "orders:approve", "inventory:read", "admin"],
    org_id: "sales", group_id: "*",
  },
};

// --- Permission helpers ---
function hasPermission(user: DemoUser | null, perm: string): boolean {
  if (!user) return false;
  if (user.permissions.includes("admin")) return true;
  return user.permissions.includes(perm) || user.permissions.includes(`${perm}:all`);
}

function canSeeAllData(user: DemoUser): boolean {
  return hasPermission(user, "orders:read:all") || hasPermission(user, "admin");
}

/** Filter orders by user's org_id and group_id (row-level security) */
function filterOrders(user: DemoUser): DemoOrder[] {
  if (canSeeAllData(user)) return mockOrders;
  return mockOrders.filter(o => o.org_id === user.org_id && (user.group_id === "*" || o.group_id === user.group_id));
}

// --- Routes ---
app.get("/", (req, res) => {
  const user = (req.session as any).user as DemoUser | null;
  if (!user) return res.redirect("/login");
  res.send(renderDashboard(user));
});

app.get("/login", (_req, res) => {
  res.send(renderLogin());
});

app.post("/login", (req, res) => {
  const userId = req.body.user_id || "sales_a";
  const user = demoUsers[userId] || demoUsers["sales_a"];
  (req.session as any).user = user;
  res.redirect("/");
});

// --- Dashboard ---
app.get("/dashboard", (req, res) => {
  const user = (req.session as any).user as DemoUser | null;
  if (!user) return res.redirect("/login");
  const userOrders = filterOrders(user);
  res.send(renderDashboard(user, userOrders));
});

// --- Orders with row-level filtering ---
app.get("/orders", (req, res) => {
  const user = (req.session as any).user as DemoUser | null;
  if (!user) return res.redirect("/login");
  if (!hasPermission(user, "orders:read")) {
    return res.status(403).send(render403("orders:read"));
  }
  const userOrders = filterOrders(user);
  res.send(renderOrders(user, userOrders));
});

// --- Inventory with button disabled states ---
app.get("/inventory", (req, res) => {
  const user = (req.session as any).user as DemoUser | null;
  if (!user) return res.redirect("/login");
  if (!hasPermission(user, "inventory:read")) {
    return res.status(403).send(render403("inventory:read"));
  }
  const canWrite = hasPermission(user, "inventory:write");
  const canDelete = hasPermission(user, "inventory:delete");
  res.send(renderInventory(user, canWrite, canDelete));
});

// --- ABAC policy check (simulates POST /api/v1/policies/check) ---
app.post("/api/policy-check", (req, res) => {
  const user = (req.session as any).user as DemoUser | null;
  if (!user) return res.status(401).json({ error: "not authenticated" });

  const { resource_type, action, attributes } = req.body;
  const allowed = hasPermission(user, `${resource_type}:${action}`);

  // Return data_filter for row-level security
  let dataFilter: Record<string, any> = {};
  if (allowed && resource_type === "orders" && !canSeeAllData(user)) {
    dataFilter = { org_id: user.org_id, group_id: user.group_id };
  }

  res.json({ allowed, data_filter: dataFilter });
});

// --- HTML renderers ---
function renderMenu(user: DemoUser): string {
  const items: string[] = [`<li><a href="/">Dashboard</a></li>`];
  if (hasPermission(user, "orders:read")) items.push(`<li><a href="/orders">Orders ${canSeeAllData(user) ? '(All Teams)' : `(${user.group_id})`}</a></li>`);
  if (hasPermission(user, "inventory:read")) items.push(`<li><a href="/inventory">Inventory</a></li>`);

  const roleBadges = user.roles.map(r => `<span class="badge">${r}</span>`).join("");

  return `
    <aside style="width:240px;background:#1a1a2e;color:#fff;padding:20px;min-height:100vh">
      <h2 style="margin:0 0 8px">GGID Demo</h2>
      <div style="margin-bottom:16px">${roleBadges}</div>
      <div style="margin-bottom:16px;padding:8px;background:#16213e;border-radius:4px;font-size:12px">
        <div>Org: <strong>${user.org_id}</strong></div>
        <div>Group: <strong>${user.group_id}</strong></div>
      </div>
      <ul style="list-style:none;padding:0">${items.join("")}</ul>
    </aside>`;
}

function renderDashboard(user: DemoUser, orders: DemoOrder[] = []): string {
  const totalAmount = orders.reduce((sum, o) => sum + o.amount, 0);
  return `<html><head><style>
    body{display:flex;font-family:system-ui;margin:0}
    .badge{background:#0f3460;padding:2px 8px;border-radius:4px;font-size:11px;margin-right:4px}
    .stat{background:#f8f9fa;padding:16px;border-radius:8px;text-align:center}
    .stat h3{margin:0;font-size:24px;color:#1890ff}
    .stat p{margin:4px 0 0;font-size:12px;color:#999}
    .filter-notice{background:#e6f7ff;border:1px solid #91d5ff;padding:8px 12px;border-radius:4px;margin-bottom:16px;font-size:13px}
  </style></head><body>${renderMenu(user)}
    <main style="flex:1;padding:24px">
      <h1>Dashboard</h1>
      <p>Welcome, <strong>${user.username}</strong></p>
      <div style="display:flex;gap:16px;margin:24px 0">
        <div class="stat"><h3>${orders.length}</h3><p>Visible Orders</p></div>
        <div class="stat"><h3>$${totalAmount.toLocaleString()}</h3><p>Total Amount</p></div>
        <div class="stat"><h3>${user.permissions.length}</h3><p>Permissions</p></div>
      </div>
      <div class="filter-notice">
        ${canSeeAllData(user)
          ? "You can see <strong>all teams'</strong> data (orders:read:all)"
          : `You can only see data for <strong>${user.org_id}/${user.group_id}</strong> (row-level filtered)`}
      </div>
      <h3>Your Permissions:</h3>
      <ul>${user.permissions.map(p => `<li><code>${p}</code></li>`).join("")}</ul>
    </main></body></html>`;
}

function renderOrders(user: DemoUser, orders: DemoOrder[]): string {
  const canWrite = hasPermission(user, "orders:write");
  const canApprove = hasPermission(user, "orders:approve");

  return `<html><head><style>
    body{display:flex;font-family:system-ui;margin:0}
    .badge{background:#0f3460;padding:2px 8px;border-radius:4px;font-size:11px;margin-right:4px}
    table{width:100%;border-collapse:collapse}
    th,td{padding:8px 12px;text-align:left;border-bottom:1px solid #eee}
    th{background:#f5f5f5}
    .filter-notice{background:#e6f7ff;border:1px solid #91d5ff;padding:8px 12px;border-radius:4px;margin-bottom:16px;font-size:13px}
    .btn{padding:6px 16px;border:none;border-radius:4px;cursor:pointer;font-size:13px}
    .btn-primary{background:#1890ff;color:#fff}
    .btn-disabled{background:#d9d9d9;color:#999;cursor:not-allowed}
    .btn-danger{background:#ff4d4f;color:#fff}
    .status-pending{color:#faad14;font-weight:bold}
    .status-approved{color:#52c41a;font-weight:bold}
    .status-shipped{color:#1890ff;font-weight:bold}
  </style></head><body>${renderMenu(user)}
    <main style="flex:1;padding:24px">
      <div style="display:flex;justify-content:space-between;align-items:center">
        <h1>Orders</h1>
        <div>
          <button class="btn ${canWrite ? 'btn-primary' : 'btn-disabled'}" ${canWrite ? '' : 'disabled'}>
            ${canWrite ? '+ New Order' : 'New Order (no write permission)'}
          </button>
        </div>
      </div>
      <div class="filter-notice">
        ${canSeeAllData(user)
          ? "Showing <strong>all teams'</strong> orders"
          : `Showing only <strong>${user.group_id}</strong> orders (row-level filtered by org_id/group_id)`}
      </div>
      <table>
        <thead><tr><th>Order ID</th><th>Customer</th><th>Amount</th><th>Status</th><th>Team</th>${canApprove ? '<th>Action</th>' : ''}</tr></thead>
        <tbody>
          ${orders.map(o => `<tr>
            <td>${o.id}</td>
            <td>${o.customer}</td>
            <td>$${o.amount.toLocaleString()}</td>
            <td class="status-${o.status}">${o.status}</td>
            <td>${o.org_id}/${o.group_id}</td>
            ${canApprove ? `<td><button class="btn btn-primary" ${o.status === 'pending' ? '' : 'disabled'}>Approve</button></td>` : ''}
          </tr>`).join("")}
        </tbody>
      </table>
    </main></body></html>`;
}

function renderInventory(user: DemoUser, canWrite: boolean, canDelete: boolean): string {
  return `<html><head><style>
    body{display:flex;font-family:system-ui;margin:0}
    .btn{padding:6px 16px;border:none;border-radius:4px;cursor:pointer;font-size:13px}
    .btn-primary{background:#1890ff;color:#fff}
    .btn-disabled{background:#d9d9d9;color:#999;cursor:not-allowed}
    .btn-danger{background:#ff4d4f;color:#fff}
    .btn-danger-disabled{background:#ffccc7;color:#ff4d4f;cursor:not-allowed}
  </style></head><body>${renderMenu(user)}
    <main style="flex:1;padding:24px">
      <h1>Inventory</h1>
      <div style="margin-bottom:16px">
        <button class="btn ${canWrite ? 'btn-primary' : 'btn-disabled'}" ${canWrite ? '' : 'disabled'}>
          ${canWrite ? '+ New Item' : 'New Item (no write permission)'}
        </button>
        <button class="btn ${canDelete ? 'btn-danger' : 'btn-danger-disabled'}" ${canDelete ? '' : 'disabled'} style="margin-left:8px">
          ${canDelete ? 'Delete Selected' : 'Delete (no delete permission)'}
        </button>
      </div>
      <table style="width:100%;border-collapse:collapse">
        <thead><tr><th>SKU</th><th>Name</th><th>Qty</th></tr></thead>
        <tbody><tr><td colspan="3" style="color:#999;padding:24px;text-align:center">No inventory data (demo)</td></tr></tbody>
      </table>
    </main></body></html>`;
}

function renderLogin(): string {
  return `<html><head><style>
    body{font-family:system-serif;display:flex;justify-content:center;align-items:center;height:100vh;background:#f0f2f5}
    .card{background:#fff;padding:32px;border-radius:8px;box-shadow:0 2px 8px rgba(0,0,0,0.1);width:360px}
    .user-btn{display:block;width:100%;padding:12px;margin:8px 0;border:1px solid #d9d9d9;border-radius:4px;cursor:pointer;text-align:left;font-size:14px}
    .user-btn:hover{border-color:#1890ff;background:#e6f7ff}
    .badge{display:inline-block;padding:2px 8px;border-radius:4px;font-size:11px;margin-left:4px}
  </style></head><body>
    <div class="card">
      <h2>GGID Demo Login</h2>
      <p style="color:#999">Select a demo user to see different permission behaviors:</p>
      <form method="POST" action="/login">
        <button class="user-btn" name="user_id" value="sales_a">
          Alice — Sales Team A<br><small>orders:read/write, filtered to team-a</small>
          <span class="badge" style="background:#e6f7ff;color:#1890ff">team-a</span>
        </button>
        <button class="user-btn" name="user_id" value="sales_b">
          Bob — Sales Team B<br><small>orders:read/write, filtered to team-b</small>
          <span class="badge" style="background:#f6ffed;color:#52c41a">team-b</span>
        </button>
        <button class="user-btn" name="user_id" value="manager">
          Manager<br><small>orders:read:all, approve, admin</small>
          <span class="badge" style="background:#fff1f0;color:#f5222d">all</span>
        </button>
      </form>
    </div>
  </body></html>`;
}

function render403(perm: string): string {
  return `<html><body style="font-family:system-serif;padding:40px">
    <h1>403 Forbidden</h1>
    <p>You need permission: <code>${perm}</code></p>
    <a href="/">Back to Dashboard</a>
  </body></html>`;
}

app.listen(PORT, () => console.log(`Fine-Grained Permission Demo on http://localhost:${PORT}`));

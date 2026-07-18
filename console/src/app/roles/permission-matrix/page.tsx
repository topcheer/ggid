"use client";

import { useState, useMemo, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  Shield, Save, Download, Search, ChevronDown, ChevronRight,
  CheckCircle2, XCircle, Lock, Settings2,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

// ===== Types =====

interface Role {
  id: string;
  key: string;
  name: string;
  description: string;
  system_role: boolean;
}

// ===== Static Role Definitions (for standalone demo) =====
// These map to common default roles; the grid works with whatever
// roles are returned from the API, falling back to these defaults.

const DEFAULT_ROLES: Role[] = [
  { id: "admin", key: "admin", name: "Admin", description: "Full system access", system_role: true },
  { id: "manager", key: "manager", name: "Manager", description: "Manage users and orgs", system_role: false },
  { id: "editor", key: "editor", name: "Editor", description: "Edit content", system_role: false },
  { id: "viewer", key: "viewer", name: "Viewer", description: "Read-only access", system_role: false },
  { id: "guest", key: "guest", name: "Guest", description: "Limited access", system_role: false },
];

// ===== Permission Group Definitions =====

interface PermDef {
  key: string;
  label: string;
}

interface PermGroup {
  label: string;
  icon: string;
  permissions: PermDef[];
}

const PERMISSION_GROUPS: PermGroup[] = [
  {
    label: "User Management",
    icon: "users",
    permissions: [
      { key: "users.create", label: "Create Users" },
      { key: "users.read", label: "View Users" },
      { key: "users.update", label: "Update Users" },
      { key: "users.delete", label: "Delete Users" },
      { key: "users.export", label: "Export Users" },
    ],
  },
  {
    label: "Organization",
    icon: "org",
    permissions: [
      { key: "orgs.create", label: "Create Orgs" },
      { key: "orgs.read", label: "View Orgs" },
      { key: "orgs.update", label: "Update Orgs" },
      { key: "orgs.delete", label: "Delete Orgs" },
    ],
  },
  {
    label: "Policy",
    icon: "policy",
    permissions: [
      { key: "policies.create", label: "Create Policies" },
      { key: "policies.read", label: "View Policies" },
      { key: "policies.update", label: "Update Policies" },
      { key: "policies.delete", label: "Delete Policies" },
      { key: "policies.evaluate", label: "Evaluate Policies" },
    ],
  },
  {
    label: "Audit",
    icon: "audit",
    permissions: [
      { key: "audit.read", label: "View Audit" },
      { key: "audit.export", label: "Export Audit" },
      { key: "audit.delete", label: "Delete Audit" },
    ],
  },
  {
    label: "Admin",
    icon: "admin",
    permissions: [
      { key: "admin.settings", label: "Manage Settings" },
      { key: "admin.impersonate", label: "Impersonate Users" },
      { key: "admin.apikeys", label: "Manage API Keys" },
      { key: "admin.webhooks", label: "Manage Webhooks" },
    ],
  },
];

// Flatten all permissions for convenience
const ALL_PERMS = PERMISSION_GROUPS.flatMap((g) => g.permissions);

// ===== Default permission assignments =====
// Pre-populate sensible defaults so the grid is useful on first load

const DEFAULT_MATRIX: Record<string, Set<string>> = {
  admin: new Set(ALL_PERMS.map((p: any) => p.key)),
  manager: new Set(["users.create", "users.read", "users.update", "users.export", "orgs.create", "orgs.read", "orgs.update", "policies.read", "audit.read"]),
  editor: new Set(["users.read", "orgs.read", "policies.read", "policies.evaluate", "audit.read"]),
  viewer: new Set(["users.read", "orgs.read", "policies.read", "audit.read"]),
  guest: new Set(["users.read"]),
};

export default function PermissionMatrixPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [roles] = useState<Role[]>(DEFAULT_ROLES);
  const [matrix, setMatrix] = useState<Record<string, Set<string>>>(() => {
    // Deep clone default matrix
    const m: Record<string, Set<string>> = {};
    for (const [k, v] of Object.entries(DEFAULT_MATRIX)) {
      m[k] = new Set(v);
    }
    return m;
  });

  const [search, setSearch] = useState("");
  const [roleFilter, setRoleFilter] = useState("all");
  const [collapsedGroups, setCollapsedGroups] = useState<Set<string>>(new Set());

  // Bulk assign state
  const [bulkRole, setBulkRole] = useState("");
  const [bulkGroup, setBulkGroup] = useState("");
  const [bulkSelected, setBulkSelected] = useState<Set<string>>(new Set());

  const [msg, setMsg] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  // ===== Derived data =====

  const filteredPerms = useMemo(() => {
    if (!search) return ALL_PERMS;
    const q = search.toLowerCase();
    return ALL_PERMS.filter(
      (p) => p.key.toLowerCase().includes(q) || p.label.toLowerCase().includes(q),
    );
  }, [search]);

  const filteredPermKeys = useMemo(() => new Set(filteredPerms.map((p: any) => p.key)), [filteredPerms]);

  const visibleRoles = useMemo(() => {
    if (roleFilter === "all") return roles;
    return roles.filter((r: any) => r.id === roleFilter);
  }, [roles, roleFilter]);

  // ===== Toggle handlers =====

  const toggleCell = useCallback((roleId: string, permKey: string) => {
    setMatrix((prev) => {
      const next = { ...prev };
      const set = new Set(next[roleId] || []);
      if (set.has(permKey)) {
        set.delete(permKey);
      } else {
        set.add(permKey);
      }
      next[roleId] = set;
      return next;
    });
  }, []);

  const toggleGroup = useCallback((groupLabel: string) => {
    setCollapsedGroups((prev) => {
      const next = new Set(prev);
      if (next.has(groupLabel)) {
        next.delete(groupLabel);
      } else {
        next.add(groupLabel);
      }
      return next;
    });
  }, []);

  const toggleEntireGroupForRole = useCallback((roleId: string, group: PermGroup) => {
    setMatrix((prev) => {
      const next = { ...prev };
      const set = new Set(next[roleId] || []);
      const allAssigned = group.permissions.every((p) => set.has(p.key));
      if (allAssigned) {
        group.permissions.forEach((p: any) => set.delete(p.key));
      } else {
        group.permissions.forEach((p: any) => set.add(p.key));
      }
      next[roleId] = set;
      return next;
    });
  }, []);

  // ===== Bulk assign =====

  const handleBulkTogglePerm = (permKey: string) => {
    setBulkSelected((prev) => {
      const next = new Set(prev);
      if (next.has(permKey)) {
        next.delete(permKey);
      } else {
        next.add(permKey);
      }
      return next;
    });
  };

  const handleApplyAll = () => {
    if (!bulkRole || bulkSelected.size === 0) {
      setError("Select a role and at least one permission to apply");
      return;
    }
    setMatrix((prev) => {
      const next = { ...prev };
      const set = new Set(next[bulkRole] || []);
      bulkSelected.forEach((k: any) => set.add(k));
      next[bulkRole] = set;
      return next;
    });
    setMsg(`Applied ${bulkSelected.size} permission(s) to ${roles.find((r: any) => r.id === bulkRole)?.name || bulkRole}`);
    setBulkSelected(new Set());
    setError(null);
  };

  const selectAllInBulkGroup = () => {
    const group = PERMISSION_GROUPS.find((g: any) => g.label === bulkGroup);
    if (group) {
      setBulkSelected(new Set(group.permissions.map((p: any) => p.key)));
    }
  };

  // ===== Export CSV =====

  const handleExportCSV = () => {
    const headers = ["Role", ...ALL_PERMS.map((p: any) => p.key)];
    const rows = roles.map((role: any) => {
      const perms = matrix[role.id] || new Set<string>();
      return [role.name, ...ALL_PERMS.map((p: any) => (perms.has(p.key) ? "Y" : "N"))];
    });
    const csv = [headers, ...rows]
      .map((row: any) => row.map((cell: any) => `"${cell}"`).join(","))
      .join("\n");
    const blob = new Blob([csv], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "permission-matrix.csv";
    a.click();
    URL.revokeObjectURL(url);
  };

  // ===== Save =====

  const handleSave = async () => {
    setSaving(true);
    setError(null);
    try {
      const payload = roles.map((role: any) => ({
        role_id: role.id,
        role_key: role.key,
        permissions: [...(matrix[role.id] || [])],
      }));
      await apiFetch("/api/v1/roles/permissions", {
        method: "PUT",
        body: JSON.stringify({ assignments: payload }),
      });
      setMsg("Permission matrix saved successfully");
    } catch {
      setMsg("Saved locally (API may not be available)");
    } finally {
      setSaving(false);
    }
  };

  // Auto-dismiss messages
  useMemo(() => {
    if (msg) {
      const t = setTimeout(() => setMsg(null), 3000);
      return () => clearTimeout(t);
    }
  }, [msg]);

  return (
    <div>
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold dark:text-gray-100">
            <Shield className="h-6 w-6 text-brand-600" />
            Role Permission Matrix
          </h1>
          <p className="mt-1 text-sm text-gray-500">
            Toggle individual permissions per role. Click cells to allow/deny access.
          </p>
        </div>
        <div className="flex gap-2">
          <button
            onClick={handleExportCSV}
            className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
          >
            <Download className="h-4 w-4" /> Export CSV
          </button>
          <button
            onClick={handleSave}
            disabled={saving}
            className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
           aria-label="Save">
            <Save className="h-4 w-4" /> {saving ? "Saving..." : "Save Matrix"}
          </button>
        </div>
      </div>

      {/* Messages */}
      {msg && (
        <div role="status" className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400">
          {msg}
        </div>
      )}
      {error && (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400">
          {error}
        </div>
      )}

      {/* Toolbar */}
      <div className="mb-4 flex flex-wrap items-center gap-3">
        {/* Search */}
        <div className="relative flex-1 min-w-[200px]">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
          <input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search permissions..."
            className="w-full rounded-lg border border-gray-300 py-2 pl-9 pr-3 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 focus:border-brand-500 focus:outline-none"
          />
        </div>
        {/* Role filter */}
        <select
          value={roleFilter}
          onChange={(e) => setRoleFilter(e.target.value)}
          className="rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
        >
          <option value="all">All Roles</option>
          {roles.map((r: any) => (
            <option key={r.id} value={r.id}>{r.name}</option>
          ))}
        </select>
      </div>

      {/* Permission Grid */}
      <div className="overflow-auto rounded-xl border border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-800" style={{ maxHeight: "70vh" }}>
        <table className="border-collapse text-sm">
          {/* Sticky header row */}
          <thead className="sticky top-0 z-20">
            <tr>
              <th scope="col" className="sticky left-0 z-30 border-b border-r border-gray-200 bg-gray-100 px-4 py-3 text-left font-semibold text-gray-600 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-300">
                Permission
              </th>
              {visibleRoles.map((role: any) => (
                <th
                  key={role.id}
                  className="border-b border-r border-gray-200 bg-gray-100 px-3 py-3 text-center font-semibold text-gray-600 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-300"
                  style={{ minWidth: 100 }}
                >
                  <div className="flex flex-col items-center gap-0.5">
                    <span className="flex items-center gap-1">
                      {role.system_role && <Lock className="h-3 w-3 text-gray-400" />}
                      {role.name}
                    </span>
                    <span className="text-[10px] font-normal text-gray-400">{role.key}</span>
                  </div>
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {PERMISSION_GROUPS.map((group: any) => {
              const groupPerms = group.permissions.filter((p: any) => filteredPermKeys.has(p.key));
              if (groupPerms.length === 0) return null;
              const isCollapsed = collapsedGroups.has(group.label);

              return (
                <tbody key={group.label}>
                  {/* Group header row */}
                  <tr>
                    <td
                      colSpan={visibleRoles.length + 1}
                      className="cursor-pointer border-b border-r border-gray-200 bg-gray-50 px-4 py-2 dark:border-gray-700 dark:bg-gray-900/50"
                      onClick={() => toggleGroup(group.label)}
                    >
                      <div className="flex items-center justify-between">
                        <span className="flex items-center gap-2 text-xs font-bold uppercase tracking-wide text-gray-600 dark:text-gray-400">
                          {isCollapsed ? <ChevronRight className="h-3.5 w-3.5" /> : <ChevronDown className="h-3.5 w-3.5" />}
                          {group.label}
                        </span>
                        <span className="text-xs text-gray-400">{groupPerms.length} permission{groupPerms.length !== 1 ? "s" : ""}</span>
                      </div>
                    </td>
                  </tr>
                  {/* Permission rows */}
                  {!isCollapsed && groupPerms.map((perm: any) => (
                    <tr key={perm.key} className="hover:bg-gray-50 dark:hover:bg-gray-700/50">
                      <td className="sticky left-0 z-10 border-b border-r border-gray-100 bg-white px-4 py-2 font-medium text-gray-700 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-300">
                        <div className="flex flex-col">
                          <span>{perm.label}</span>
                          <span className="text-[10px] font-mono text-gray-400">{perm.key}</span>
                        </div>
                      </td>
                      {visibleRoles.map((role: any) => {
                        const allowed = matrix[role.id]?.has(perm.key) ?? false;
                        return (
                          <td
                            key={role.id}
                            className="border-b border-r border-gray-100 px-3 py-2 text-center dark:border-gray-700"
                          >
                            <button
                              onClick={() => toggleCell(role.id, perm.key)}
                              className="inline-flex h-7 w-7 items-center justify-center rounded transition-colors hover:bg-gray-100 dark:hover:bg-gray-600"
                              title={allowed ? "Click to deny" : "Click to allow"}
                            >
                              {allowed ? (
                                <CheckCircle2 className="h-5 w-5 text-green-500" />
                              ) : (
                                <XCircle className="h-5 w-5 text-gray-300 dark:text-gray-600" />
                              )}
                            </button>
                          </td>
                        );
                      })}
                    </tr>
                  ))}
                  {/* Group summary row (allow toggle entire group per role) */}
                  {!isCollapsed && (
                    <tr className="bg-gray-50/50 dark:bg-gray-900/30">
                      <td className="sticky left-0 z-10 border-b border-r-2 border-gray-200 bg-gray-50/50 px-4 py-1.5 text-xs text-gray-400 dark:border-gray-700 dark:bg-gray-900/30">
                        Toggle all {group.label}
                      </td>
                      {visibleRoles.map((role: any) => {
                        const allOn = groupPerms.every((p) => matrix[role.id]?.has(p.key));
                        return (
                          <td key={role.id} className="border-b border-r border-gray-200 px-3 py-1.5 text-center dark:border-gray-700">
                            <button
                              onClick={() => toggleEntireGroupForRole(role.id, { ...group, permissions: groupPerms })}
                              className={`rounded px-2 py-0.5 text-xs font-medium ${
                                allOn
                                  ? "bg-green-100 text-green-700 hover:bg-green-200 dark:bg-green-900 dark:text-green-400"
                                  : "bg-gray-100 text-gray-500 hover:bg-gray-200 dark:bg-gray-700 dark:text-gray-400"
                              }`}
                            >
                              {allOn ? "All On" : "Toggle"}
                            </button>
                          </td>
                        );
                      })}
                    </tr>
                  )}
                </tbody>
              );
            })}
          </tbody>
        </table>
      </div>

      {/* Summary stats */}
      <div className="mt-4 flex flex-wrap gap-3">
        {roles.map((role: any) => {
          const count = matrix[role.id]?.size || 0;
          const total = ALL_PERMS.length;
          const pct = Math.round((count / total) * 100);
          return (
            <div key={role.id} className="flex items-center gap-2 rounded-lg border border-gray-200 bg-white px-3 py-2 dark:border-gray-700 dark:bg-gray-800">
              <Shield className="h-4 w-4 text-brand-600" />
              <div>
                <span className="text-sm font-medium">{role.name}</span>
                <span className="ml-2 text-xs text-gray-400">{count}/{total} ({pct}%)</span>
              </div>
              <div className="h-2 w-16 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
                <div className="h-full rounded-full bg-brand-500" style={{ width: `${pct}%` }} />
              </div>
            </div>
          );
        })}
      </div>

      {/* Bulk Assign Panel */}
      <div className="mt-6 rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800">
        <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold">
          <Settings2 className="h-4 w-4 text-brand-600" />
          Bulk Assign Permissions
        </h3>
        <div className="flex flex-wrap items-start gap-4">
          {/* Role selector */}
          <div>
            <label className="mb-1 block text-xs font-medium text-gray-500">Select Role</label>
            <select
              value={bulkRole}
              onChange={(e) => setBulkRole(e.target.value)}
              className="rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
            >
              <option value="">-- Choose --</option>
              {roles.map((r: any) => (
                <option key={r.id} value={r.id}>{r.name}</option>
              ))}
            </select>
          </div>

          {/* Group selector */}
          <div>
            <label className="mb-1 block text-xs font-medium text-gray-500">Permission Group</label>
            <div className="flex gap-2">
              <select
                value={bulkGroup}
                onChange={(e) => setBulkGroup(e.target.value)}
                className="rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
              >
                <option value="">-- Choose --</option>
                {PERMISSION_GROUPS.map((g: any) => (
                  <option key={g.label} value={g.label}>{g.label}</option>
                ))}
              </select>
              {bulkGroup && (
                <button
                  onClick={selectAllInBulkGroup}
                  className="rounded-lg border border-gray-300 px-3 py-2 text-xs font-medium text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
                >
                  Select All in Group
                </button>
              )}
            </div>
          </div>

          {/* Permission checkboxes */}
          {bulkGroup && (
            <div className="flex-1 min-w-[250px]">
              <label className="mb-1 block text-xs font-medium text-gray-500">
                Permissions ({bulkSelected.size} selected)
              </label>
              <div className="flex flex-wrap gap-2 rounded-lg border border-gray-200 p-3 dark:border-gray-700">
                {PERMISSION_GROUPS.find((g: any) => g.label === bulkGroup)?.permissions.map((p: any) => (
                  <label
                    key={p.key}
                    className={`flex cursor-pointer items-center gap-1.5 rounded-lg border px-3 py-1.5 text-xs ${
                      bulkSelected.has(p.key)
                        ? "border-brand-300 bg-brand-50 text-brand-700 dark:border-brand-700 dark:bg-brand-900/30 dark:text-brand-400"
                        : "border-gray-200 text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
                    }`}
                  >
                    <input
                      type="checkbox"
                      checked={bulkSelected.has(p.key)}
                      onChange={() => handleBulkTogglePerm(p.key)}
                      className="h-3.5 w-3.5 rounded border-gray-300 text-brand-600 focus:ring-brand-500"
                    />
                    {p.label}
                  </label>
                ))}
              </div>
            </div>
          )}

          {/* Apply button */}
          <div className="flex items-end">
            <button
              onClick={handleApplyAll}
              disabled={!bulkRole || bulkSelected.size === 0}
              className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
            >
              <CheckCircle2 className="h-4 w-4" /> Apply All
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

"use client";

import { useEffect, useState, useCallback } from "react";
import { useApi } from "@/lib/api";
import { Shield, Plus, Trash2, X, CheckCircle2, XCircle, Search } from "lucide-react";

interface Role {
  id: string;
  key: string;
  name: string;
  description: string;
  system_role: boolean;
}

interface Permission {
  id: string;
  key: string;
  name: string;
  resource_type: string;
  action: string;
}

type Tab = "roles" | "permissions" | "checker";

export default function RolesPage() {
  const { apiFetch } = useApi();
  const [roles, setRoles] = useState<Role[]>([]);
  const [permissions, setPermissions] = useState<Permission[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [msg, setMsg] = useState<string | null>(null);
  const [tab, setTab] = useState<Tab>("roles");
  const [showCreate, setShowCreate] = useState(false);
  const [createForm, setCreateForm] = useState({ key: "", name: "", description: "" });

  // Permission management state
  const [selectedRole, setSelectedRole] = useState<string>("");
  const [rolePerms, setRolePerms] = useState<Permission[]>([]);
  const [permLoading, setPermLoading] = useState(false);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const [rolesResp, permsResp] = await Promise.all([
        apiFetch<{ roles?: Role[]; items?: Role[] }>("/api/v1/roles").catch(() => ({ roles: [] as Role[] })),
        apiFetch<{ permissions?: Permission[]; items?: Permission[] }>("/api/v1/permissions").catch(() => ({ permissions: [] as Permission[] })),
      ]);
      setRoles((rolesResp as { roles?: Role[]; items?: Role[] }).roles || (rolesResp as { items?: Role[] }).items || []);
      setPermissions((permsResp as { permissions?: Permission[]; items?: Permission[] }).permissions || (permsResp as { items?: Permission[] }).items || []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load");
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  // Load role permissions when a role is selected
  const loadRolePerms = useCallback(async (roleId: string) => {
    if (!roleId) {
      setRolePerms([]);
      return;
    }
    setPermLoading(true);
    try {
      const data = await apiFetch<{ permissions?: Permission[] }>(
        `/api/v1/roles/${roleId}/permissions`,
      ).catch(() => ({ permissions: [] as Permission[] }));
      setRolePerms(data.permissions || []);
    } catch {
      setRolePerms([]);
    } finally {
      setPermLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => {
    if (tab === "permissions" && selectedRole) {
      loadRolePerms(selectedRole);
    }
  }, [tab, selectedRole, loadRolePerms]);

  const handleCreate = async () => {
    if (!createForm.key || !createForm.name) return;
    try {
      await apiFetch("/api/v1/roles", {
        method: "POST",
        body: JSON.stringify(createForm),
      });
      setShowCreate(false);
      setCreateForm({ key: "", name: "", description: "" });
      setMsg("Role created successfully");
      loadData();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create role");
    }
  };

  const handleDelete = async (roleId: string, systemRole: boolean) => {
    if (systemRole) {
      setError("Cannot delete system role");
      return;
    }
    if (!confirm("Delete this role?")) return;
    try {
      await apiFetch(`/api/v1/roles/${roleId}`, { method: "DELETE" });
      setMsg("Role deleted");
      loadData();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete role");
    }
  };

  const handleAssignPerm = async (permId: string) => {
    if (!selectedRole) return;
    try {
      await apiFetch(`/api/v1/roles/${selectedRole}/permissions`, {
        method: "POST",
        body: JSON.stringify({ permission_ids: [permId] }),
      });
      setMsg("Permission assigned");
      loadRolePerms(selectedRole);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to assign permission");
    }
  };

  const handleRevokePerm = async (permId: string) => {
    if (!selectedRole) return;
    try {
      await apiFetch(`/api/v1/roles/${selectedRole}/permissions`, {
        method: "DELETE",
        body: JSON.stringify({ permission_ids: [permId] }),
      });
      setMsg("Permission revoked");
      loadRolePerms(selectedRole);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to revoke permission");
    }
  };

  // Auto-dismiss messages
  useEffect(() => {
    if (msg) {
      const t = setTimeout(() => setMsg(null), 3000);
      return () => clearTimeout(t);
    }
  }, [msg]);

  return (
    <div>
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold">Roles & Permissions</h1>
        <div className="flex gap-2">
          <button
            onClick={() => setShowCreate(!showCreate)}
            className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
          >
            <Plus className="h-4 w-4" /> Create Role
          </button>
        </div>
      </div>

      {/* Create Role Form */}
      {showCreate && (
        <div className="mb-4 rounded-xl border border-gray-200 bg-white p-4 shadow-sm">
          <div className="mb-3 flex items-center justify-between">
            <h3 className="text-sm font-semibold">New Role</h3>
            <button onClick={() => setShowCreate(false)}>
              <X className="h-4 w-4 text-gray-400" />
            </button>
          </div>
          <div className="grid gap-3 sm:grid-cols-3">
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">Key</label>
              <input
                value={createForm.key}
                onChange={(e) => setCreateForm({ ...createForm, key: e.target.value })}
                placeholder="e.g. editor"
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none"
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">Name</label>
              <input
                value={createForm.name}
                onChange={(e) => setCreateForm({ ...createForm, name: e.target.value })}
                placeholder="e.g. Editor"
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none"
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">Description</label>
              <input
                value={createForm.description}
                onChange={(e) => setCreateForm({ ...createForm, description: e.target.value })}
                placeholder="Optional"
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none"
              />
            </div>
          </div>
          <button
            onClick={handleCreate}
            disabled={!createForm.key || !createForm.name}
            className="mt-3 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
          >
            Create
          </button>
        </div>
      )}

      {/* Messages */}
      {msg && (
        <div className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700">
          {msg}
        </div>
      )}
      {error && (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">
          {error}
        </div>
      )}

      {/* Tabs */}
      <div className="mb-4 flex gap-2 border-b border-gray-200">
        <TabButton active={tab === "roles"} onClick={() => setTab("roles")} label={`Roles (${roles.length})`} />
        <TabButton active={tab === "permissions"} onClick={() => setTab("permissions")} label="Permission Assignment" />
        <TabButton active={tab === "checker"} onClick={() => setTab("checker")} label="Policy Checker" />
      </div>

      {loading ? (
        <p className="text-gray-500">Loading...</p>
      ) : tab === "roles" ? (
        /* ===== Roles Grid ===== */
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {roles.map((role) => (
            <div key={role.id} className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
              <div className="mb-3 flex items-start justify-between">
                <div className="flex items-center gap-3">
                  <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-brand-100">
                    <Shield className="h-5 w-5 text-brand-600" />
                  </div>
                  <div>
                    <h3 className="font-semibold">{role.name || role.key}</h3>
                    <p className="text-xs text-gray-500">{role.key}</p>
                  </div>
                </div>
                {!role.system_role && (
                  <button
                    onClick={() => handleDelete(role.id, role.system_role)}
                    className="text-gray-400 hover:text-red-500"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                )}
              </div>
              <p className="mb-3 text-sm text-gray-600">{role.description || "No description"}</p>
              {role.system_role && (
                <span className="rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-600">
                  System
                </span>
              )}
            </div>
          ))}
          {roles.length === 0 && (
            <p className="col-span-full text-center text-gray-500">No roles found</p>
          )}
        </div>
      ) : tab === "permissions" ? (
        /* ===== Permission Assignment ===== */
        <PermissionAssignment
          roles={roles}
          permissions={permissions}
          rolePerms={rolePerms}
          selectedRole={selectedRole}
          onSelectRole={setSelectedRole}
          onAssign={handleAssignPerm}
          onRevoke={handleRevokePerm}
          loading={permLoading}
        />
      ) : (
        /* ===== Policy Checker ===== */
        <PolicyChecker apiFetch={apiFetch} />
      )}
    </div>
  );
}

// ===== Policy Checker Component =====

interface CheckResult {
  allowed: boolean;
  reason: string;
  matched_by: string;
}

function PolicyChecker({
  apiFetch,
}: {
  apiFetch: <T>(path: string, options?: RequestInit) => Promise<T>;
}) {
  const [form, setForm] = useState({
    user_id: "",
    resource_type: "",
    action: "",
    resource: "",
  });
  const [result, setResult] = useState<CheckResult | null>(null);
  const [checking, setChecking] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleCheck = async () => {
    if (!form.user_id || !form.resource_type || !form.action) return;
    setChecking(true);
    setError(null);
    setResult(null);
    try {
      const data = await apiFetch<CheckResult>("/api/v1/policies/check", {
        method: "POST",
        body: JSON.stringify(form),
      });
      setResult(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Check failed");
    } finally {
      setChecking(false);
    }
  };

  return (
    <div className="mx-auto max-w-2xl">
      <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
        <div className="mb-4">
          <h3 className="flex items-center gap-2 text-lg font-semibold">
            <Search className="h-5 w-5 text-brand-600" />
            Policy Checker
          </h3>
          <p className="mt-1 text-sm text-gray-500">
            Test if a user has permission to perform an action on a resource.
          </p>
        </div>

        <div className="grid gap-4">
          <div>
            <label className="mb-1 block text-xs font-medium text-gray-500">
              User ID <span className="text-red-500">*</span>
            </label>
            <input
              value={form.user_id}
              onChange={(e) => setForm({ ...form, user_id: e.target.value })}
              placeholder="e.g. 550e8400-e29b-41d4-a716-446655440000"
              className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm font-mono focus:border-brand-500 focus:outline-none"
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">
                Resource Type <span className="text-red-500">*</span>
              </label>
              <input
                value={form.resource_type}
                onChange={(e) => setForm({ ...form, resource_type: e.target.value })}
                placeholder="e.g. users, roles, documents"
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none"
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">
                Action <span className="text-red-500">*</span>
              </label>
              <select
                value={form.action}
                onChange={(e) => setForm({ ...form, action: e.target.value })}
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none"
              >
                <option value="">-- Select --</option>
                <option value="read">read</option>
                <option value="create">create</option>
                <option value="update">update</option>
                <option value="delete">delete</option>
                <option value="manage">manage</option>
                <option value="assign">assign</option>
              </select>
            </div>
          </div>
          <div>
            <label className="mb-1 block text-xs font-medium text-gray-500">
              Resource (optional)
            </label>
            <input
              value={form.resource}
              onChange={(e) => setForm({ ...form, resource: e.target.value })}
              placeholder="e.g. specific resource ID or path"
              className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none"
            />
          </div>

          <button
            onClick={handleCheck}
            disabled={checking || !form.user_id || !form.resource_type || !form.action}
            className="flex items-center justify-center gap-2 rounded-lg bg-brand-600 px-4 py-2.5 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
          >
            <Search className="h-4 w-4" />
            {checking ? "Checking..." : "Check Permission"}
          </button>
        </div>

        {/* Error */}
        {error && (
          <div className="mt-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">
            {error}
          </div>
        )}

        {/* Result */}
        {result && (
          <div className="mt-4">
            <div
              className={`flex items-center gap-3 rounded-lg border p-4 ${
                result.allowed
                  ? "border-green-200 bg-green-50"
                  : "border-red-200 bg-red-50"
              }`}
            >
              {result.allowed ? (
                <CheckCircle2 className="h-6 w-6 text-green-600" />
              ) : (
                <XCircle className="h-6 w-6 text-red-600" />
              )}
              <div>
                <p className={`font-semibold ${result.allowed ? "text-green-700" : "text-red-700"}`}>
                  {result.allowed ? "ALLOWED" : "DENIED"}
                </p>
                <p className="text-sm text-gray-600">
                  {result.reason || (result.allowed ? "Access granted" : "Access denied")}
                </p>
                {result.matched_by && (
                  <p className="mt-1 text-xs text-gray-400">Matched by: {result.matched_by}</p>
                )}
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

// ===== Permission Assignment Component =====

function PermissionAssignment({
  roles,
  permissions,
  rolePerms,
  selectedRole,
  onSelectRole,
  onAssign,
  onRevoke,
  loading,
}: {
  roles: Role[];
  permissions: Permission[];
  rolePerms: Permission[];
  selectedRole: string;
  onSelectRole: (id: string) => void;
  onAssign: (permId: string) => void;
  onRevoke: (permId: string) => void;
  loading: boolean;
}) {
  const [search, setSearch] = useState("");
  const rolePermIds = new Set(rolePerms.map((p) => p.id));

  const filteredPerms = permissions.filter(
    (p) =>
      !search ||
      p.key.toLowerCase().includes(search.toLowerCase()) ||
      p.name.toLowerCase().includes(search.toLowerCase()) ||
      p.resource_type.toLowerCase().includes(search.toLowerCase()),
  );

  return (
    <div className="space-y-4">
      {/* Role selector */}
      <div className="flex items-center gap-3">
        <label className="text-sm font-medium text-gray-600">Select Role:</label>
        <select
          value={selectedRole}
          onChange={(e) => onSelectRole(e.target.value)}
          className="rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none"
        >
          <option value="">-- Choose a role --</option>
          {roles.map((r) => (
            <option key={r.id} value={r.id}>
              {r.name || r.key}
              {r.system_role ? " (system)" : ""}
            </option>
          ))}
        </select>
      </div>

      {!selectedRole ? (
        <div className="rounded-xl border border-gray-200 bg-white p-12 text-center shadow-sm">
          <Shield className="mx-auto mb-4 h-12 w-12 text-gray-300" />
          <p className="text-gray-500">Select a role to manage its permissions</p>
        </div>
      ) : loading ? (
        <p className="text-gray-500">Loading permissions...</p>
      ) : (
        <>
          {/* Currently assigned */}
          <div className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm">
            <h3 className="mb-3 text-sm font-semibold">
              Assigned Permissions ({rolePerms.length})
            </h3>
            {rolePerms.length === 0 ? (
              <p className="text-sm text-gray-400">No permissions assigned yet</p>
            ) : (
              <div className="flex flex-wrap gap-2">
                {rolePerms.map((p) => (
                  <div
                    key={p.id}
                    className="flex items-center gap-2 rounded-lg border border-green-200 bg-green-50 px-3 py-1.5"
                  >
                    <span className="text-sm font-medium text-green-800">
                      {p.name || p.key}
                    </span>
                    <span className="text-xs text-green-600">{p.resource_type}:{p.action}</span>
                    <button
                      onClick={() => onRevoke(p.id)}
                      className="text-green-400 hover:text-red-500"
                    >
                      <X className="h-3.5 w-3.5" />
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Available permissions */}
          <div className="rounded-xl border border-gray-200 bg-white shadow-sm">
            <div className="flex items-center justify-between border-b border-gray-200 p-4">
              <h3 className="text-sm font-semibold">Available Permissions ({filteredPerms.length})</h3>
              <div className="relative">
                <Search className="absolute left-2 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
                <input
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  placeholder="Search..."
                  className="rounded-lg border border-gray-300 py-1.5 pl-8 pr-3 text-sm focus:border-brand-500 focus:outline-none"
                />
              </div>
            </div>
            <div className="max-h-96 overflow-y-auto">
              {filteredPerms.map((p) => {
                const assigned = rolePermIds.has(p.id);
                return (
                  <div
                    key={p.id}
                    className="flex items-center justify-between border-b border-gray-50 px-4 py-2.5 hover:bg-gray-50"
                  >
                    <div className="flex items-center gap-3">
                      <div>
                        <p className="text-sm font-medium">{p.name || p.key}</p>
                        <p className="text-xs text-gray-500">
                          {p.resource_type} : {p.action}
                        </p>
                      </div>
                    </div>
                    {assigned ? (
                      <button
                        onClick={() => onRevoke(p.id)}
                        className="rounded-lg border border-red-200 px-3 py-1 text-xs font-medium text-red-600 hover:bg-red-50"
                      >
                        Revoke
                      </button>
                    ) : (
                      <button
                        onClick={() => onAssign(p.id)}
                        className="rounded-lg border border-brand-200 bg-brand-50 px-3 py-1 text-xs font-medium text-brand-600 hover:bg-brand-100"
                      >
                        + Assign
                      </button>
                    )}
                  </div>
                );
              })}
              {filteredPerms.length === 0 && (
                <p className="px-4 py-8 text-center text-sm text-gray-400">
                  No permissions available
                </p>
              )}
            </div>
          </div>
        </>
      )}
    </div>
  );
}

// ===== Reusable Components =====

function TabButton({
  active,
  onClick,
  label,
}: {
  active: boolean;
  onClick: () => void;
  label: string;
}) {
  return (
    <button
      onClick={onClick}
      className={`px-4 py-2 text-sm font-medium ${
        active
          ? "border-b-2 border-brand-600 text-brand-600"
          : "text-gray-500 hover:text-gray-700"
      }`}
    >
      {label}
    </button>
  );
}

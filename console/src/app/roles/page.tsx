"use client";

import { useEffect, useState, useCallback } from "react";
import { useApi } from "@/lib/api";
import { Shield, Plus, Trash2, X, CheckCircle2, XCircle, Search, Copy, GitBranch, Layers, Pencil, Users } from "lucide-react";

interface Role {
  id: string;
  key: string;
  name: string;
  description: string;
  system_role: boolean;
  parent_role_id?: string;
}

interface Permission {
  id: string;
  key: string;
  name: string;
  resource_type: string;
  action: string;
}

type Tab = "roles" | "permissions" | "checker" | "matrix" | "hierarchy" | "abac";

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
  const [editingRole, setEditingRole] = useState<Role | null>(null);
  const [editForm, setEditForm] = useState({ key: "", name: "", description: "" });
  const [roleUsers, setRoleUsers] = useState<Record<string, { id: string; username: string; email: string }[]>>({});

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

  const handleCloneRole = async (role: Role) => {
    try {
      const cloneKey = `${role.key}-copy-${Date.now().toString(36)}`;
      await apiFetch("/api/v1/roles", {
        method: "POST",
        body: JSON.stringify({
          key: cloneKey,
          name: `${role.name} (Copy)`,
          description: role.description || "",
          parent_role_id: role.id,
        }),
      });
      setMsg(`Role cloned as ${cloneKey}`);
      loadData();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to clone role");
    }
  };

  const handleBatchAssign = async (permIds: string[]) => {
    if (!selectedRole || permIds.length === 0) return;
    try {
      await apiFetch(`/api/v1/roles/${selectedRole}/permissions`, {
        method: "POST",
        body: JSON.stringify({ permission_ids: permIds }),
      });
      setMsg(`${permIds.length} permissions assigned`);
      loadRolePerms(selectedRole);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to assign permissions");
    }
  };

  const handleEditRole = (role: Role) => {
    setEditingRole(role);
    setEditForm({ key: role.key, name: role.name, description: role.description || "" });
  };

  const handleSaveEdit = async () => {
    if (!editingRole) return;
    try {
      await apiFetch(`/api/v1/roles/${editingRole.id}`, {
        method: "PUT",
        body: JSON.stringify(editForm),
      });
      setEditingRole(null);
      setMsg("Role updated successfully");
      loadData();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update role");
    }
  };

  const loadRoleUsers = async (roleId: string) => {
    if (roleUsers[roleId]) return;
    try {
      const data = await apiFetch<{ users?: { id: string; username: string; email: string }[] }>(
        `/api/v1/roles/${roleId}/users`,
      ).catch(() => ({ users: [] }));
      setRoleUsers((prev) => ({ ...prev, [roleId]: (data as { users?: { id: string; username: string; email: string }[] }).users || [] }));
    } catch {
      setRoleUsers((prev) => ({ ...prev, [roleId]: [] }));
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
        <h1 className="text-2xl font-bold dark:text-gray-100">Roles & Permissions</h1>
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
            <button onClick={() => setShowCreate(false)} aria-label="Close">
              <X className="h-4 w-4 text-gray-400" />
            </button>
          </div>
          <div className="grid gap-3 sm:grid-cols-3">
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">Key</label>
              <input
                value={createForm.key}
                onChange={(e) => setCreateForm({ ...createForm, key: e.target.value })}
                placeholder="e.g. editor"
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 focus:border-brand-500 focus:outline-none"
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">Name</label>
              <input
                value={createForm.name}
                onChange={(e) => setCreateForm({ ...createForm, name: e.target.value })}
                placeholder="e.g. Editor"
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 focus:border-brand-500 focus:outline-none"
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">Description</label>
              <input
                value={createForm.description}
                onChange={(e) => setCreateForm({ ...createForm, description: e.target.value })}
                placeholder="Optional"
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 focus:border-brand-500 focus:outline-none"
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

      {/* Edit Role Form */}
      {editingRole && (
        <div className="mb-4 rounded-xl border border-gray-200 bg-white p-4 shadow-sm">
          <div className="mb-3 flex items-center justify-between">
            <h3 className="text-sm font-semibold">Edit Role: {editingRole.name || editingRole.key}</h3>
            <button onClick={() => setEditingRole(null)} aria-label="Close">
              <X className="h-4 w-4 text-gray-400" />
            </button>
          </div>
          <div className="grid gap-3 sm:grid-cols-3">
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">Key</label>
              <input
                value={editForm.key}
                onChange={(e) => setEditForm({ ...editForm, key: e.target.value })}
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 focus:border-brand-500 focus:outline-none"
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">Name</label>
              <input
                value={editForm.name}
                onChange={(e) => setEditForm({ ...editForm, name: e.target.value })}
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 focus:border-brand-500 focus:outline-none"
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">Description</label>
              <input
                value={editForm.description}
                onChange={(e) => setEditForm({ ...editForm, description: e.target.value })}
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 focus:border-brand-500 focus:outline-none"
              />
            </div>
          </div>
          <button
            onClick={handleSaveEdit}
            className="mt-3 flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
          >
            <Pencil className="h-4 w-4" /> Save Changes
          </button>
        </div>
      )}

      {/* Tabs */}
      <div className="mb-4 flex gap-2 border-b border-gray-200 overflow-x-auto">
        <TabButton active={tab === "roles"} onClick={() => setTab("roles")} label={`Roles (${roles.length})`} />
        <TabButton active={tab === "permissions"} onClick={() => setTab("permissions")} label="Permissions" />
        <TabButton active={tab === "hierarchy"} onClick={() => setTab("hierarchy")} label="Hierarchy" />
        <TabButton active={tab === "matrix"} onClick={() => setTab("matrix")} label="Matrix" />
        <TabButton active={tab === "checker"} onClick={() => setTab("checker")} label="Checker" />
        <TabButton active={tab === "abac"} onClick={() => setTab("abac")} label="ABAC Builder" />
      </div>

      {loading ? (
        <p className="text-gray-500">Loading...</p>
      ) : tab === "roles" ? (
        /* ===== Roles Grid ===== */
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {roles.map((role) => (
            <div key={role.id} className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800">
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
                <div className="flex items-center gap-1">
                  <button
                    onClick={() => handleEditRole(role)}
                    className="text-gray-400 hover:text-brand-500"
                    title="Edit role"
                  >
                    <Pencil className="h-4 w-4" />
                  </button>
                  <button
                    onClick={() => { loadRoleUsers(role.id); }}
                    className="text-gray-400 hover:text-brand-500"
                    title="View users with this role"
                  >
                    <Users className="h-4 w-4" />
                  </button>
                  <button
                    onClick={() => handleCloneRole(role)}
                    className="text-gray-400 hover:text-brand-500"
                    title="Clone role"
                  >
                    <Copy className="h-4 w-4" />
                  </button>
                  {!role.system_role && (
                    <button
                      onClick={() => handleDelete(role.id, role.system_role)}
                      className="text-gray-400 hover:text-red-500"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  )}
                </div>
              </div>
              <p className="mb-3 text-sm text-gray-600 dark:text-gray-400">{role.description || "No description"}</p>
              {/* Users with this role */}
              {roleUsers[role.id] && (
                <div className="mb-3 rounded-lg bg-gray-50 p-2 dark:bg-gray-700">
                  <p className="mb-1 text-xs font-medium text-gray-500">
                    <Users className="mr-1 inline h-3 w-3" />Users with this role ({roleUsers[role.id].length})
                  </p>
                  {roleUsers[role.id].length === 0 ? (
                    <p className="text-[10px] text-gray-400">No users assigned</p>
                  ) : (
                    <div className="flex flex-wrap gap-1">
                      {roleUsers[role.id].slice(0, 5).map((u) => (
                        <span key={u.id} className="rounded bg-white px-1.5 py-0.5 text-[10px] dark:bg-gray-600">
                          {u.username}
                        </span>
                      ))}
                      {roleUsers[role.id].length > 5 && (
                        <span className="text-[10px] text-gray-400">+{roleUsers[role.id].length - 5} more</span>
                      )}
                    </div>
                  )}
                </div>
              )}
              <div className="flex flex-wrap gap-1">
                {role.system_role && (
                  <span className="rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-600">
                    System
                  </span>
                )}
                {role.parent_role_id && (
                  <span className="flex items-center gap-1 rounded-full bg-blue-50 px-2 py-0.5 text-xs font-medium text-blue-600">
                    <GitBranch className="h-3 w-3" />
                    Inherits
                  </span>
                )}
              </div>
            </div>
          ))}
          {roles.length === 0 && (
            <p className="col-span-full text-center text-gray-500">No roles found</p>
          )}
        </div>
      ) : tab === "hierarchy" ? (
        /* ===== Role Hierarchy Tree ===== */
        <RoleHierarchyTree roles={roles} apiFetch={apiFetch} />
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
          onBatchAssign={handleBatchAssign}
          loading={permLoading}
        />
      ) : tab === "matrix" ? (
        <RolePermissionMatrix roles={roles} permissions={permissions} apiFetch={apiFetch} />
      ) : tab === "abac" ? (
        /* ===== ABAC Condition Builder ===== */
        <ABACConditionBuilder apiFetch={apiFetch} />
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
      <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
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
            <label className="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">
              User ID <span className="text-red-500">*</span>
            </label>
            <input
              value={form.user_id}
              onChange={(e) => setForm({ ...form, user_id: e.target.value })}
              placeholder="e.g. 550e8400-e29b-41d4-a716-446655440000"
              className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 font-mono focus:border-brand-500 focus:outline-none"
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">
                Resource Type <span className="text-red-500">*</span>
              </label>
              <input
                value={form.resource_type}
                onChange={(e) => setForm({ ...form, resource_type: e.target.value })}
                placeholder="e.g. users, roles, documents"
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 focus:border-brand-500 focus:outline-none"
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">
                Action <span className="text-red-500">*</span>
              </label>
              <select
                value={form.action}
                onChange={(e) => setForm({ ...form, action: e.target.value })}
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 focus:border-brand-500 focus:outline-none"
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
            <label className="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">
              Resource (optional)
            </label>
            <input
              value={form.resource}
              onChange={(e) => setForm({ ...form, resource: e.target.value })}
              placeholder="e.g. specific resource ID or path"
              className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 focus:border-brand-500 focus:outline-none"
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
                <p className="text-sm text-gray-600 dark:text-gray-400">
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
  onBatchAssign,
  loading,
}: {
  roles: Role[];
  permissions: Permission[];
  rolePerms: Permission[];
  selectedRole: string;
  onSelectRole: (id: string) => void;
  onAssign: (permId: string) => void;
  onRevoke: (permId: string) => void;
  onBatchAssign: (permIds: string[]) => void;
  loading: boolean;
}) {
  const [search, setSearch] = useState("");
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const rolePermIds = new Set(rolePerms.map((p) => p.id));

  const filteredPerms = permissions.filter(
    (p) =>
      !search ||
      p.key.toLowerCase().includes(search.toLowerCase()) ||
      p.name.toLowerCase().includes(search.toLowerCase()) ||
      p.resource_type.toLowerCase().includes(search.toLowerCase()),
  );

  // Group permissions by resource_type
  const groupedPerms = filteredPerms.reduce<Record<string, typeof permissions>>((acc, p) => {
    const key = p.resource_type || "other";
    if (!acc[key]) acc[key] = [];
    acc[key].push(p);
    return acc;
  }, {});
  const resourceGroups = Object.keys(groupedPerms).sort();

  return (
    <div className="space-y-4">
      {/* Role selector */}
      <div className="flex items-center gap-3">
        <label className="text-sm font-medium text-gray-600">Select Role:</label>
        <select
          value={selectedRole}
          onChange={(e) => onSelectRole(e.target.value)}
          className="rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 focus:border-brand-500 focus:outline-none"
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
              <div className="flex items-center gap-3">
                <h3 className="text-sm font-semibold">Available Permissions ({filteredPerms.length})</h3>
                {selected.size > 0 && (
                  <button
                    onClick={() => { onBatchAssign([...selected]); setSelected(new Set()); }}
                    className="flex items-center gap-1 rounded-lg bg-brand-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-brand-700"
                  >
                    <Layers className="h-3.5 w-3.5" />
                    Assign {selected.size} selected
                  </button>
                )}
              </div>
              <div className="flex items-center gap-2">
                <button
                  onClick={() => {
                    if (selected.size === filteredPerms.length) {
                      setSelected(new Set());
                    } else {
                      setSelected(new Set(filteredPerms.map((p) => p.id)));
                    }
                  }}
                  className="text-xs font-medium text-brand-600 hover:text-brand-700"
                >
                  {selected.size === filteredPerms.length && selected.size > 0 ? "Deselect all" : "Select all"}
                </button>
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
            </div>
            <div className="max-h-96 overflow-y-auto">
              {resourceGroups.map((resource) => (
                <div key={resource}>
                  <div className="sticky top-0 flex items-center justify-between border-b border-gray-100 dark:border-gray-700 bg-gray-50 px-4 py-1.5">
                    <span className="text-xs font-bold uppercase tracking-wide text-gray-600">{resource}</span>
                    <span className="text-xs text-gray-400">{groupedPerms[resource].length} permission{groupedPerms[resource].length !== 1 ? "s" : ""}</span>
                  </div>
                  {groupedPerms[resource].map((p) => {
                    const assigned = rolePermIds.has(p.id);
                    const isChecked = selected.has(p.id);
                    return (
                      <div
                        key={p.id}
                        className="flex items-center justify-between border-b border-gray-50 px-4 py-2.5 hover:bg-gray-50 dark:hover:bg-gray-700"
                      >
                        <div className="flex items-center gap-3">
                          {!assigned && (
                            <input
                              type="checkbox"
                              checked={isChecked}
                              onChange={() => {
                                const next = new Set(selected);
                                if (isChecked) { next.delete(p.id); } else { next.add(p.id); }
                                setSelected(next);
                              }}
                              className="h-4 w-4 rounded border-gray-300 text-brand-600 focus:ring-brand-500"
                            />
                          )}
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
                </div>
              ))}
              {filteredPerms.length === 0 && (
                <p className="px-4 py-8 text-center text-sm text-gray-400 dark:text-gray-500">
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

// ===== Role-Permission Matrix =====

function RolePermissionMatrix({
  roles,
  permissions,
  apiFetch,
}: {
  roles: Role[];
  permissions: Permission[];
  apiFetch: <T>(path: string, options?: RequestInit) => Promise<T>;
}) {
  const [matrix, setMatrix] = useState<Record<string, Set<string>>>({});
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const loadMatrix = async () => {
      setLoading(true);
      const m: Record<string, Set<string>> = {};
      for (const role of roles) {
        try {
          const data = await apiFetch<{ permissions?: Permission[] }>(`/api/v1/roles/${role.id}/permissions`);
          m[role.id] = new Set((data.permissions || []).map((p) => p.id));
        } catch {
          m[role.id] = new Set();
        }
      }
      setMatrix(m);
      setLoading(false);
    };
    if (roles.length > 0) loadMatrix();
    else setLoading(false);
  }, [roles, apiFetch]);

  if (loading) return <p className="text-gray-500">Loading matrix...</p>;
  if (roles.length === 0 || permissions.length === 0) {
    return <p className="text-gray-500">No roles or permissions to display</p>;
  }

  // Group permissions by resource_type
  const grouped = permissions.reduce((acc, p) => {
    const key = p.resource_type || "other";
    if (!acc[key]) acc[key] = [];
    acc[key].push(p);
    return acc;
  }, {} as Record<string, Permission[]>);

  return (
    <div className="overflow-x-auto rounded-xl border border-gray-200 bg-white shadow-sm">
      <table className="w-full text-sm">
        <thead className="sticky top-0 border-b border-gray-200 bg-gray-50">
          <tr>
            <th className="px-3 py-2 text-left font-medium text-gray-500">Role</th>
            {Object.entries(grouped).map(([resource, perms]) => (
              <th key={resource} className="px-3 py-2 text-center font-medium text-gray-500">
                {resource}
                <span className="ml-1 text-xs text-gray-400">({perms.length})</span>
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-100">
          {roles.map((role) => (
            <tr key={role.id} className="hover:bg-gray-50 dark:hover:bg-gray-700">
              <td className="px-3 py-2 font-medium">
                {role.name}
                {role.system_role && <span className="ml-1 text-xs text-gray-400">(system)</span>}
              </td>
              {Object.entries(grouped).map(([resource, perms]) => {
                const rolePerms = matrix[role.id];
                const hasAll = perms.every((p) => rolePerms?.has(p.id));
                const hasSome = perms.some((p) => rolePerms?.has(p.id));
                return (
                  <td key={resource} className="px-3 py-2 text-center">
                    {hasAll ? (
                      <CheckCircle2 className="mx-auto h-5 w-5 text-green-500" />
                    ) : hasSome ? (
                      <span className="text-xs text-amber-600">partial</span>
                    ) : (
                      <XCircle className="mx-auto h-5 w-5 text-gray-300" />
                    )}
                  </td>
                );
              })}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

// ===== Role Hierarchy Tree =====

function RoleHierarchyTree({
  roles,
  apiFetch,
}: {
  roles: Role[];
  apiFetch: <T>(path: string, options?: RequestInit) => Promise<T>;
}) {
  const [effectivePerms, setEffectivePerms] = useState<Record<string, string[]>>({});
  const [expanded, setExpanded] = useState<Set<string>>(new Set());

  // Build parent->children map
  const childMap = roles.reduce<Record<string, Role[]>>((acc, role) => {
    if (role.parent_role_id) {
      if (!acc[role.parent_role_id]) acc[role.parent_role_id] = [];
      acc[role.parent_role_id].push(role);
    }
    return acc;
  }, {});

  const roots = roles.filter((r) => !r.parent_role_id);

  const loadEffective = async (roleId: string) => {
    try {
      const data = await apiFetch<{ permissions?: { key: string }[] }>(
        `/api/v1/roles/${roleId}/effective-permissions`,
      ).catch(() => ({ permissions: [] }));
      setEffectivePerms((prev) => ({
        ...prev,
        [roleId]: (data.permissions || []).map((p) => p.key),
      }));
    } catch { /* ignore */ }
  };

  const toggleExpand = (roleId: string) => {
    const next = new Set(expanded);
    if (next.has(roleId)) {
      next.delete(roleId);
    } else {
      next.add(roleId);
      loadEffective(roleId);
    }
    setExpanded(next);
  };

  const renderNode = (role: Role, depth: number): React.ReactElement => {
    const children = childMap[role.id] || [];
    const isExpanded = expanded.has(role.id);
    const perms = effectivePerms[role.id] || [];

    return (
      <div key={role.id} style={{ marginLeft: depth * 24 }}>
        <div className="flex items-center gap-2 py-2">
          {children.length > 0 ? (
            <button
              onClick={() => toggleExpand(role.id)}
              className="text-gray-400 hover:text-gray-600"
            >
              {isExpanded ? "▼" : "▶"}
            </button>
          ) : (
            <span className="w-4" />
          )}
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-brand-100">
            <Shield className="h-4 w-4 text-brand-600" />
          </div>
          <div className="flex-1">
            <span className="font-medium">{role.name || role.key}</span>
            {role.system_role && (
              <span className="ml-2 rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-500">System</span>
            )}
            <span className="ml-2 text-xs text-gray-400">{role.key}</span>
          </div>
          {perms.length > 0 && (
            <div className="flex flex-wrap gap-1">
              {perms.slice(0, 5).map((p) => (
                <span key={p} className="rounded bg-blue-50 px-1.5 py-0.5 text-xs text-blue-600">{p}</span>
              ))}
              {perms.length > 5 && <span className="text-xs text-gray-400">+{perms.length - 5} more</span>}
            </div>
          )}
          {children.length > 0 && (
            <span className="text-xs text-gray-400">{children.length} child{children.length !== 1 ? "ren" : ""}</span>
          )}
        </div>
        {isExpanded && children.length > 0 && (
          <div className="border-l border-gray-200">
            {children.map((child) => renderNode(child, depth + 1))}
          </div>
        )}
      </div>
    );
  };

  return (
    <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800">
      <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold">
        <GitBranch className="h-4 w-4 text-brand-600" />
        Role Hierarchy & Inheritance
      </h3>
      <p className="mb-4 text-xs text-gray-500">
        Parent roles inherit permissions from child roles. Click to expand and view effective permissions.
      </p>
      {roots.length === 0 ? (
        <p className="py-8 text-center text-sm text-gray-400 dark:text-gray-500">
          No roles with hierarchy. Set <code className="rounded bg-gray-100 px-1">parent_role_id</code> when creating roles to build inheritance chains.
        </p>
      ) : (
        <div>
          {roots.map((root) => renderNode(root, 0))}
        </div>
      )}
    </div>
  );
}

// ===== ABAC Condition Builder =====

interface ConditionRule {
  attribute: string;
  operator: string;
  value: string;
}

const ABAC_ATTRIBUTES = [
  "user.role", "user.department", "user.location", "user.clearance_level",
  "resource.type", "resource.owner", "resource.department", "resource.classification",
  "request.ip", "request.time", "request.method", "request.user_agent",
  "environment.risk_score", "environment.device_type",
];

const ABAC_OPERATORS = ["eq", "ne", "in", "not_in", "gt", "lt", "gte", "lte", "contains", "regex"];

function ABACConditionBuilder({
  apiFetch,
}: {
  apiFetch: <T>(path: string, options?: RequestInit) => Promise<T>;
}) {
  const [rules, setRules] = useState<ConditionRule[]>([
    { attribute: "user.role", operator: "eq", value: "admin" },
  ]);
  const [combineMode, setCombineMode] = useState<"AND" | "OR">("AND");
  const [effect, setEffect] = useState<"allow" | "deny">("allow");
  const [policyName, setPolicyName] = useState("");
  const [msg, setMsg] = useState<string | null>(null);

  const addRule = () => {
    setRules([...rules, { attribute: "request.ip", operator: "eq", value: "" }]);
  };

  const removeRule = (idx: number) => {
    setRules(rules.filter((_, i) => i !== idx));
  };

  const updateRule = (idx: number, field: keyof ConditionRule, value: string) => {
    setRules(rules.map((r, i) => i === idx ? { ...r, [field]: value } : r));
  };

  const generatedJSON = JSON.stringify({
    name: policyName || "abac_policy",
    effect: effect,
    combine: combineMode,
    conditions: rules.map((r) => ({
      attribute: r.attribute,
      operator: r.operator,
      value: r.value,
    })),
  }, null, 2);

  const handleSave = async () => {
    try {
      await apiFetch("/api/v1/policies", {
        method: "POST",
        body: generatedJSON,
      });
      setMsg("ABAC policy saved successfully");
      setTimeout(() => setMsg(null), 3000);
    } catch {
      setMsg("Failed to save policy (policy API may not be available)");
      setTimeout(() => setMsg(null), 3000);
    }
  };

  return (
    <div className="space-y-4">
      <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800">
        <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold">
          <Layers className="h-4 w-4 text-brand-600" />
          ABAC Condition Builder
        </h3>
        <p className="mb-4 text-xs text-gray-500">
          Build attribute-based access control rules visually. The generated JSON can be saved as a policy.
        </p>

        {/* Policy metadata */}
        <div className="mb-4 grid gap-3 sm:grid-cols-3">
          <div>
            <label className="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">Policy Name</label>
            <input
              value={policyName}
              onChange={(e) => setPolicyName(e.target.value)}
              placeholder="e.g. restrict_admin_access"
              className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
            />
          </div>
          <div>
            <label className="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">Combine Mode</label>
            <select
              value={combineMode}
              onChange={(e) => setCombineMode(e.target.value as "AND" | "OR")}
              className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
            >
              <option value="AND">AND (all must match)</option>
              <option value="OR">OR (any can match)</option>
            </select>
          </div>
          <div>
            <label className="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">Effect</label>
            <select
              value={effect}
              onChange={(e) => setEffect(e.target.value as "allow" | "deny")}
              className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
            >
              <option value="allow">Allow</option>
              <option value="deny">Deny</option>
            </select>
          </div>
        </div>

        {/* Condition rules */}
        <div className="space-y-2">
          {rules.map((rule, idx) => (
            <div key={idx} className="flex items-center gap-2">
              <span className="flex h-6 w-6 items-center justify-center rounded-full bg-gray-100 text-xs font-medium text-gray-500 dark:text-gray-400">
                {idx + 1}
              </span>
              <select
                value={rule.attribute}
                onChange={(e) => updateRule(idx, "attribute", e.target.value)}
                className="flex-1 rounded-lg border border-gray-300 px-2 py-1.5 text-sm"
              >
                {ABAC_ATTRIBUTES.map((attr) => (
                  <option key={attr} value={attr}>{attr}</option>
                ))}
              </select>
              <select
                value={rule.operator}
                onChange={(e) => updateRule(idx, "operator", e.target.value)}
                className="w-28 rounded-lg border border-gray-300 px-2 py-1.5 text-sm font-mono"
              >
                {ABAC_OPERATORS.map((op) => (
                  <option key={op} value={op}>{op}</option>
                ))}
              </select>
              <input
                value={rule.value}
                onChange={(e) => updateRule(idx, "value", e.target.value)}
                placeholder="value"
                className="flex-1 rounded-lg border border-gray-300 px-2 py-1.5 text-sm"
              />
              <button
                onClick={() => removeRule(idx)}
                className="text-gray-400 hover:text-red-500"
              >
                <X className="h-4 w-4" />
              </button>
            </div>
          ))}
        </div>

        <div className="mt-3 flex items-center gap-2">
          <button
            onClick={addRule}
            className="flex items-center gap-1 rounded-lg border border-gray-300 px-3 py-1.5 text-xs font-medium text-gray-600 hover:bg-gray-50 dark:hover:bg-gray-700"
          >
            <Plus className="h-3.5 w-3.5" /> Add Condition
          </button>
        </div>

        {/* Generated JSON */}
        <div className="mt-4">
          <label className="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">Generated Policy JSON</label>
          <pre className="max-h-48 overflow-auto rounded-lg bg-gray-900 p-4 text-xs text-green-400">
            {generatedJSON}
          </pre>
        </div>

        <div className="mt-3 flex items-center gap-2">
          <button
            onClick={handleSave}
            className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
          >
            Save Policy
          </button>
          <button
            onClick={() => navigator.clipboard.writeText(generatedJSON)}
            className="rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 text-gray-600 hover:bg-gray-50 dark:hover:bg-gray-700"
          >
            Copy JSON
          </button>
          {msg && <span className="text-sm text-green-600">{msg}</span>}
        </div>
      </div>
    </div>
  );
}

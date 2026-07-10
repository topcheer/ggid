"use client";

import { useEffect, useState, useCallback } from "react";
import { useApi } from "@/lib/api";
import { Shield, Plus, Trash2, X } from "lucide-react";

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

export default function RolesPage() {
  const { apiFetch } = useApi();
  const [roles, setRoles] = useState<Role[]>([]);
  const [permissions, setPermissions] = useState<Permission[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [msg, setMsg] = useState<string | null>(null);
  const [tab, setTab] = useState<"roles" | "permissions">("roles");
  const [showCreate, setShowCreate] = useState(false);
  const [createForm, setCreateForm] = useState({ key: "", name: "", description: "" });

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

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold">Roles & Permissions</h1>
        <button
          onClick={() => setShowCreate(!showCreate)}
          className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
        >
          <Plus className="h-4 w-4" /> Create Role
        </button>
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
        <button
          onClick={() => setTab("roles")}
          className={`px-4 py-2 text-sm font-medium ${
            tab === "roles"
              ? "border-b-2 border-brand-600 text-brand-600"
              : "text-gray-500 hover:text-gray-700"
          }`}
        >
          Roles ({roles.length})
        </button>
        <button
          onClick={() => setTab("permissions")}
          className={`px-4 py-2 text-sm font-medium ${
            tab === "permissions"
              ? "border-b-2 border-brand-600 text-brand-600"
              : "text-gray-500 hover:text-gray-700"
          }`}
        >
          Permissions ({permissions.length})
        </button>
      </div>

      {loading ? (
        <p className="text-gray-500">Loading...</p>
      ) : tab === "roles" ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {roles.map((role) => (
            <div
              key={role.id}
              className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm"
            >
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
      ) : (
        <div className="overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm">
          <table className="w-full">
            <thead className="border-b border-gray-200 bg-gray-50">
              <tr>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Permission</th>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Resource</th>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Action</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {permissions.map((perm) => (
                <tr key={perm.id} className="hover:bg-gray-50">
                  <td className="px-4 py-3">
                    <p className="text-sm font-medium">{perm.name || perm.key}</p>
                    <p className="text-xs text-gray-500">{perm.key}</p>
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-600">{perm.resource_type}</td>
                  <td className="px-4 py-3">
                    <span className="rounded-full bg-blue-50 px-2 py-0.5 text-xs font-medium text-blue-700">
                      {perm.action}
                    </span>
                  </td>
                </tr>
              ))}
              {permissions.length === 0 && (
                <tr>
                  <td colSpan={3} className="px-4 py-8 text-center text-gray-500">
                    No permissions found
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

"use client";

import { useState, useEffect, useCallback } from "react";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import {
  Shield,
  ArrowLeft,
  Plus,
  Trash2,
  Save,
  Loader2,
  CheckCircle2,
  Users,
  KeyRound,
} from "lucide-react";

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

interface AssignedUser {
  id: string;
  username: string;
  email: string;
}

export default function RoleDetailPage() {
  const params = useParams();
  const roleId = params.id as string;
  const router = useRouter();
  const t = useTranslations();
  const { apiFetch } = useApi();

  // Guard: redirect to list if ID is missing or literal "[id]"
  useEffect(() => {
    if (!roleId || roleId === "[id]") {
      router.replace("/roles");
    }
  }, [roleId, router]);

  const [role, setRole] = useState<Role | null>(null);
  const [permissions, setPermissions] = useState<Permission[]>([]);
  const [assignedUsers, setAssignedUsers] = useState<AssignedUser[]>([]);
  const [allPermissions, setAllPermissions] = useState<Permission[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [msg, setMsg] = useState("");

  // Edit form
  const [editName, setEditName] = useState("");
  const [editDesc, setEditDesc] = useState("");
  const [showAddPerm, setShowAddPerm] = useState(false);

  const loadRole = useCallback(async () => {
    setLoading(true);
    try {
      const [roleResp, permsResp, usersResp, allPermsResp] = await Promise.all([
        apiFetch<Role>(`/api/v1/roles/${roleId}`).catch(() => null),
        apiFetch<{ permissions?: Permission[] }>(`/api/v1/roles/${roleId}/permissions`).catch(() => ({ permissions: [] })),
        apiFetch<{ users?: AssignedUser[] }>(`/api/v1/roles/${roleId}/users`).catch(() => ({ users: [] })),
        apiFetch<{ permissions?: Permission[] }>(`/api/v1/permissions`).catch(() => ({ permissions: [] })),
      ]);
      if (roleResp) {
        setRole(roleResp);
        setEditName(roleResp.name);
        setEditDesc(roleResp.description);
      }
      setPermissions(permsResp?.permissions ?? []);
      setAssignedUsers(usersResp?.users ?? []);
      setAllPermissions(allPermsResp?.permissions ?? []);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, [apiFetch, roleId]);

  useEffect(() => {
    loadRole();
  }, [loadRole]);

  const handleSaveRole = async () => {
    if (!role) return;
    setSaving(true);
    try {
      await apiFetch(`/api/v1/roles/${roleId}`, {
        method: "PUT",
        body: JSON.stringify({ name: editName, description: editDesc }),
      });
      setRole({ ...role, name: editName, description: editDesc });
      setMsg("Role updated");
    } catch {
      setMsg("Failed to update role");
    } finally {
      setSaving(false);
      setTimeout(() => setMsg(""), 3000);
    }
  };

  const handleAddPermission = async (permId: string) => {
    try {
      await apiFetch(`/api/v1/roles/${roleId}/permissions`, {
        method: "POST",
        body: JSON.stringify({ permission_id: permId }),
      });
      const perm = allPermissions.find((p: any) => p.id === permId);
      if (perm) setPermissions([...permissions, perm]);
      setShowAddPerm(false);
      setMsg("Permission added");
      setTimeout(() => setMsg(""), 3000);
    } catch {
      setMsg("Failed to add permission");
    }
  };

  const handleRemovePermission = async (permId: string) => {
    try {
      await apiFetch(`/api/v1/roles/${roleId}/permissions/${permId}`, {
        method: "DELETE",
      });
      setPermissions(permissions.filter((p: any) => p.id !== permId));
      setMsg("Permission removed");
      setTimeout(() => setMsg(""), 3000);
    } catch {
      setMsg("Failed to remove permission");
    }
  };

  const handleRemoveUser = async (userId: string) => {
    try {
      await apiFetch(`/api/v1/roles/${roleId}/users/${userId}`, {
        method: "DELETE",
      });
      setAssignedUsers(assignedUsers.filter((u: any) => u.id !== userId));
      setMsg("User unassigned");
      setTimeout(() => setMsg(""), 3000);
    } catch {
      setMsg("Failed to unassign user");
    }
  };

  const availablePermissions = allPermissions.filter(
    (p) => !permissions.some((ep: any) => ep.id === p.id)
  );

  const inputCls =
    "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";
  const cardCls =
    "rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  if (loading) {
    return (
      <div className="flex items-center justify-center py-24">
        <Loader2 className="h-8 w-8 animate-spin text-indigo-600" />
      </div>
    );
  }

  if (!role) {
    return (
      <div className="py-12 text-center">
        <p className="text-gray-500">Role not found.</p>
        <Link href="/roles" className="mt-3 inline-block text-indigo-600 hover:underline">
          ← Back to Roles
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Link
            href="/roles"
            className="rounded-lg p-2 text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700"
          >
            <ArrowLeft className="h-5 w-5" />
          </Link>
          <div>
            <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
              <Shield className="h-7 w-7 text-indigo-600" />
              {role.name}
            </h1>
            <div className="mt-1 flex items-center gap-2">
              <code className="rounded bg-gray-100 px-2 py-0.5 text-xs text-gray-500 dark:bg-gray-700">
                {role.key}
              </code>
              {role.system_role && (
                <span className="rounded-full bg-blue-100 px-2 py-0.5 text-xs font-medium text-blue-700 dark:bg-blue-900/30 dark:text-blue-400">
                  System Role
                </span>
              )}
            </div>
          </div>
        </div>
        {msg && <span className="text-sm text-green-600">{msg}</span>}
      </div>

      <div className="grid gap-6 lg:grid-cols-3">
        {/* Left: Role editor + Permissions */}
        <div className="space-y-6 lg:col-span-2">
          {/* Role editor */}
          <div className={cardCls}>
            <h3 className="mb-4 text-sm font-semibold text-gray-700 dark:text-gray-300">
              Role Details
            </h3>
            <div className="space-y-3">
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500">Name</label>
                <input
                  aria-label="Role name"
                  className={inputCls}
                  value={editName}
                  onChange={(e) => setEditName(e.target.value)}
                  disabled={role.system_role}
                />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500">Description</label>
                <textarea
                  aria-label="Role description"
                  className={inputCls}
                  rows={2}
                  value={editDesc}
                  onChange={(e) => setEditDesc(e.target.value)}
                  disabled={role.system_role}
                />
              </div>
              {!role.system_role && (
                <button
                  onClick={handleSaveRole}
                  disabled={saving}
                  aria-label="Save role changes"
                  className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50 focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 dark:focus:ring-offset-gray-800"
                >
                  {saving ? <Loader2 className="mr-1 inline h-4 w-4 animate-spin" /> : <Save className="mr-1 inline h-4 w-4" />}
                  Save Changes
                </button>
              )}
            </div>
          </div>

          {/* Permissions */}
          <div className={cardCls}>
            <div className="mb-4 flex items-center justify-between">
              <h3 className="flex items-center gap-2 text-sm font-semibold text-gray-700 dark:text-gray-300">
                <KeyRound className="h-4 w-4" /> Permissions ({permissions.length})
              </h3>
              {!role.system_role && (
                <button
                  onClick={() => setShowAddPerm(!showAddPerm)}
                  aria-label="Add permission to role"
                  className="rounded-lg border border-gray-300 px-3 py-1.5 text-xs font-medium text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700 focus:ring-2 focus:ring-brand-500 focus:ring-offset-2 dark:focus:ring-offset-gray-800"
                >
                  <Plus className="mr-1 inline h-3.5 w-3.5" /> Add Permission
                </button>
              )}
            </div>

            {showAddPerm && (
              <div className="mb-4 space-y-2 rounded-lg border border-gray-200 p-3 dark:border-gray-700">
                {availablePermissions.length === 0 ? (
                  <p className="text-sm text-gray-400">All permissions already assigned.</p>
                ) : (
                  availablePermissions.map((perm: any) => (
                    <div
                      key={perm.id}
                      className="flex items-center justify-between rounded-lg px-3 py-1.5 hover:bg-gray-50 dark:hover:bg-gray-700/50"
                    >
                      <div>
                        <span className="text-sm font-medium text-gray-800 dark:text-gray-200">{perm.name}</span>
                        <span className="ml-2 font-mono text-xs text-gray-400">{perm.resource_type}:{perm.action}</span>
                      </div>
                      <button
                        onClick={() => handleAddPermission(perm.id)}
                        className="rounded bg-indigo-50 px-2 py-1 text-xs font-medium text-indigo-600 hover:bg-indigo-100 dark:bg-indigo-900/20 dark:text-indigo-400"
                      >
                        Add
                      </button>
                    </div>
                  ))
                )}
              </div>
            )}

            <div className="space-y-2">
              {permissions.map((perm: any) => (
                <div
                  key={perm.id}
                  className="flex items-center justify-between rounded-lg border border-gray-100 px-3 py-2 dark:border-gray-700/50"
                >
                  <div className="flex items-center gap-2">
                    <CheckCircle2 className="h-4 w-4 text-green-500" />
                    <div>
                      <span className="text-sm font-medium text-gray-800 dark:text-gray-200">{perm.name}</span>
                      <span className="ml-2 font-mono text-xs text-gray-400">
                        {perm.resource_type}:{perm.action}
                      </span>
                    </div>
                  </div>
                  {!role.system_role && (
                    <button
                      onClick={() => handleRemovePermission(perm.id)}
                      className="rounded-lg p-1.5 text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  )}
                </div>
              ))}
              {permissions.length === 0 && (
                <p className="py-4 text-center text-sm text-gray-400">No permissions assigned.</p>
              )}
            </div>
          </div>
        </div>

        {/* Right: Assigned users */}
        <div className={cardCls}>
          <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold text-gray-700 dark:text-gray-300">
            <Users className="h-4 w-4" /> Assigned Users ({assignedUsers.length})
          </h3>
          <div className="space-y-2">
            {assignedUsers.map((user: any) => (
              <div
                key={user.id}
                className="flex items-center justify-between rounded-lg border border-gray-100 px-3 py-2 dark:border-gray-700/50"
              >
                <Link href={`/users/${user.id}`} className="flex-1">
                  <div>
                    <p className="text-sm font-medium text-gray-800 dark:text-gray-200">{user.username}</p>
                    <p className="text-xs text-gray-400">{user.email}</p>
                  </div>
                </Link>
                {!role.system_role && (
                  <button
                    onClick={() => handleRemoveUser(user.id)}
                    className="rounded-lg p-1.5 text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </button>
                )}
              </div>
            ))}
            {assignedUsers.length === 0 && (
              <p className="py-4 text-center text-sm text-gray-400">No users assigned.</p>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

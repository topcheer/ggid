"use client";

import { useState, useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import { useApi } from "@/lib/api";
import {
  ArrowLeft,
  Lock,
  Unlock,
  KeyRound,
  Save,
  Shield,
  Trash2,
} from "lucide-react";

interface User {
  id: string;
  username: string;
  email: string;
  phone: string;
  status: string;
  email_verified: boolean;
  display_name: string;
  locale: string;
  timezone: string;
  created_at: string;
}

interface Role {
  id: string;
  key: string;
  name: string;
  description: string;
  system_role: boolean;
}

export default function UserDetailPage({ params }: { params: { id: string } }) {
  const { apiFetch } = useApi();
  const router = useRouter();
  const [user, setUser] = useState<User | null>(null);
  const [roles, setRoles] = useState<Role[]>([]);
  const [userRoles, setUserRoles] = useState<Role[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editing, setEditing] = useState(false);
  const [editForm, setEditForm] = useState({ display_name: "", email: "", phone: "" });
  const [resetPassword, setResetPassword] = useState("");
  const [msg, setMsg] = useState<string | null>(null);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const u = await apiFetch<User>(`/api/v1/users/${params.id}`);
      setUser(u);
      setEditForm({ display_name: u.display_name || "", email: u.email || "", phone: u.phone || "" });

      const rolesResp = await apiFetch<{ roles?: Role[] }>("/api/v1/roles").catch(() => ({ roles: [] }));
      setRoles(rolesResp.roles || []);

      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load user");
    } finally {
      setLoading(false);
    }
  }, [params.id, apiFetch]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const handleSave = async () => {
    try {
      await apiFetch(`/api/v1/users/${params.id}`, {
        method: "PUT",
        body: JSON.stringify(editForm),
      });
      setEditing(false);
      setMsg("User updated successfully");
      loadData();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update user");
    }
  };

  const handleLock = async (lock: boolean) => {
    try {
      await apiFetch(`/api/v1/users/${params.id}/${lock ? "lock" : "unlock"}`, {
        method: "POST",
      });
      setMsg(`User ${lock ? "locked" : "unlocked"} successfully`);
      loadData();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed");
    }
  };

  const handleResetPassword = async () => {
    if (!resetPassword) return;
    try {
      await apiFetch("/api/v1/auth/password/reset", {
        method: "POST",
        body: JSON.stringify({ user_id: params.id, new_password: resetPassword }),
      });
      setResetPassword("");
      setMsg("Password reset successfully");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to reset password");
    }
  };

  const handleAssignRole = async (roleId: string) => {
    try {
      await apiFetch(`/api/v1/users/${params.id}/roles`, {
        method: "POST",
        body: JSON.stringify({ role_id: roleId }),
      });
      setMsg("Role assigned");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to assign role");
    }
  };

  const handleDelete = async () => {
    if (!confirm("Are you sure you want to delete this user?")) return;
    try {
      await apiFetch(`/api/v1/users/${params.id}`, { method: "DELETE" });
      router.push("/users");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete");
    }
  };

  if (loading) return <p className="py-8 text-center text-gray-500">Loading...</p>;
  if (error && !user)
    return (
      <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-red-700">{error}</div>
    );
  if (!user) return <p className="py-8 text-center text-gray-500">User not found</p>;

  return (
    <div>
      {/* Header */}
      <div className="mb-6 flex items-center gap-4">
        <button
          onClick={() => router.push("/users")}
          className="flex items-center gap-1 text-sm text-gray-500 hover:text-gray-700"
        >
          <ArrowLeft className="h-4 w-4" /> Back to Users
        </button>
      </div>

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

      <div className="grid gap-6 lg:grid-cols-2">
        {/* User Info Card */}
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-lg font-semibold">User Information</h2>
            <button
              onClick={() => setEditing(!editing)}
              className="rounded-lg px-3 py-1.5 text-sm font-medium text-brand-600 hover:bg-brand-50"
            >
              {editing ? "Cancel" : "Edit"}
            </button>
          </div>

          {editing ? (
            <div className="space-y-3">
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500">Display Name</label>
                <input
                  value={editForm.display_name}
                  onChange={(e) => setEditForm({ ...editForm, display_name: e.target.value })}
                  className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none"
                />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500">Email</label>
                <input
                  value={editForm.email}
                  onChange={(e) => setEditForm({ ...editForm, email: e.target.value })}
                  className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none"
                />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500">Phone</label>
                <input
                  value={editForm.phone}
                  onChange={(e) => setEditForm({ ...editForm, phone: e.target.value })}
                  className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none"
                />
              </div>
              <button
                onClick={handleSave}
                className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
              >
                <Save className="h-4 w-4" /> Save Changes
              </button>
            </div>
          ) : (
            <dl className="space-y-3">
              <div className="flex justify-between">
                <dt className="text-sm text-gray-500">Username</dt>
                <dd className="text-sm font-medium">{user.username}</dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-sm text-gray-500">Email</dt>
                <dd className="text-sm font-medium">{user.email}</dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-sm text-gray-500">Display Name</dt>
                <dd className="text-sm font-medium">{user.display_name || "-"}</dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-sm text-gray-500">Status</dt>
                <dd>
                  <span
                    className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                      user.status === "active"
                        ? "bg-green-50 text-green-700"
                        : "bg-red-50 text-red-700"
                    }`}
                  >
                    {user.status}
                  </span>
                </dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-sm text-gray-500">Email Verified</dt>
                <dd className="text-sm font-medium">{user.email_verified ? "Yes" : "No"}</dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-sm text-gray-500">Created</dt>
                <dd className="text-sm font-medium">
                  {new Date(user.created_at).toLocaleDateString()}
                </dd>
              </div>
            </dl>
          )}
        </div>

        {/* Actions Card */}
        <div className="space-y-4">
          {/* Lock/Unlock */}
          <div className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm">
            <h3 className="mb-3 text-sm font-semibold">Account Actions</h3>
            <div className="flex gap-2">
              {user.status === "locked" ? (
                <button
                  onClick={() => handleLock(false)}
                  className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium hover:bg-gray-50"
                >
                  <Unlock className="h-4 w-4" /> Unlock
                </button>
              ) : (
                <button
                  onClick={() => handleLock(true)}
                  className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium hover:bg-gray-50"
                >
                  <Lock className="h-4 w-4" /> Lock
                </button>
              )}
              <button
                onClick={handleDelete}
                className="flex items-center gap-1.5 rounded-lg border border-red-300 px-3 py-2 text-sm font-medium text-red-600 hover:bg-red-50"
              >
                <Trash2 className="h-4 w-4" /> Delete
              </button>
            </div>
          </div>

          {/* Reset Password */}
          <div className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm">
            <h3 className="mb-3 text-sm font-semibold">Reset Password</h3>
            <div className="flex gap-2">
              <input
                type="password"
                value={resetPassword}
                onChange={(e) => setResetPassword(e.target.value)}
                placeholder="New password"
                className="flex-1 rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none"
              />
              <button
                onClick={handleResetPassword}
                disabled={!resetPassword}
                className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-3 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
              >
                <KeyRound className="h-4 w-4" /> Reset
              </button>
            </div>
          </div>

          {/* Role Assignment */}
          <div className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm">
            <h3 className="mb-3 text-sm font-semibold">Assign Role</h3>
            <div className="space-y-2">
              {roles.map((role) => (
                <div key={role.id} className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <Shield className="h-4 w-4 text-gray-400" />
                    <span className="text-sm font-medium">{role.name || role.key}</span>
                  </div>
                  <button
                    onClick={() => handleAssignRole(role.id)}
                    className="rounded-lg border border-gray-300 px-2 py-1 text-xs font-medium hover:bg-gray-50"
                  >
                    Assign
                  </button>
                </div>
              ))}
              {roles.length === 0 && (
                <p className="text-sm text-gray-500">No roles available</p>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

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
  Activity,
  Building2,
  UserMinus,
  UserPlus,
  Fingerprint,
  Globe,
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

interface OrgMembership {
  id: string;
  name: string;
  path: string;
  role: string;
  parent_id?: string;
}

interface AuditEvent {
  id: string;
  action: string;
  resource_type: string;
  result: string;
  created_at: string;
  ip_address: string;
}

interface Credential {
  id: string;
  name: string;
  type: string;
  transports?: string[];
  aaguid?: string;
  created_at?: string;
  last_used_at?: string;
}

interface SocialConnection {
  id: string;
  provider: string;
  provider_user_id: string;
  email?: string;
  name?: string;
  created_at?: string;
}

type DetailTab = "info" | "roles" | "organizations" | "activity";

export default function UserDetailPage({ params }: { params: { id: string } }) {
  const { apiFetch } = useApi();
  const router = useRouter();
  const [user, setUser] = useState<User | null>(null);
  const [roles, setRoles] = useState<Role[]>([]);
  const [userRoles, setUserRoles] = useState<Role[]>([]);
  const [orgs, setOrgs] = useState<OrgMembership[]>([]);
  const [orgsLoading, setOrgsLoading] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editing, setEditing] = useState(false);
  const [editForm, setEditForm] = useState({ display_name: "", email: "", phone: "", status: "" });
  const [resetPassword, setResetPassword] = useState("");
  const [msg, setMsg] = useState<string | null>(null);
  const [detailTab, setDetailTab] = useState<DetailTab>("info");
  const [activity, setActivity] = useState<AuditEvent[]>([]);
  const [activityLoading, setActivityLoading] = useState(false);
  const [credentials, setCredentials] = useState<Credential[]>([]);
  const [socialConnections, setSocialConnections] = useState<SocialConnection[]>([]);

  const loadActivity = useCallback(async () => {
    setActivityLoading(true);
    try {
      const data = await apiFetch<{ events?: AuditEvent[] }>(
        `/api/v1/audit/events?actor_id=${params.id}&page_size=20`,
      ).catch(() => ({ events: [] }));
      setActivity(data.events || []);
    } catch {
      setActivity([]);
    } finally {
      setActivityLoading(false);
    }
  }, [apiFetch, params.id]);

  const loadUserRoles = useCallback(async () => {
    try {
      const data = await apiFetch<{ roles?: Role[] }>(
        `/api/v1/users/${params.id}/roles`,
      ).catch(() => ({ roles: [] }));
      setUserRoles(data.roles || []);
    } catch {
      setUserRoles([]);
    }
  }, [apiFetch, params.id]);

  const loadOrgs = useCallback(async () => {
    setOrgsLoading(true);
    try {
      const data = await apiFetch<{ organizations?: OrgMembership[] }>(
        `/api/v1/org/memberships?user_id=${params.id}`,
      ).catch(() => ({ organizations: [] }));
      setOrgs(data.organizations || []);
    } catch {
      setOrgs([]);
    } finally {
      setOrgsLoading(false);
    }
  }, [apiFetch, params.id]);

  useEffect(() => {
    if (detailTab === "activity") loadActivity();
    if (detailTab === "roles") loadUserRoles();
    if (detailTab === "organizations") loadOrgs();
  }, [detailTab, loadActivity, loadUserRoles, loadOrgs]);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const u = await apiFetch<User>(`/api/v1/users/${params.id}`);
      setUser(u);
      setEditForm({ display_name: u.display_name || "", email: u.email || "", phone: u.phone || "", status: u.status || "active" });

      const [rolesResp, credsResp, socialResp] = await Promise.all([
        apiFetch<{ roles?: Role[] }>("/api/v1/roles").catch(() => ({ roles: [] })),
        apiFetch<{ credentials?: Credential[] }>(`/api/v1/users/${params.id}/credentials`).catch(() => ({ credentials: [] })),
        apiFetch<{ connections?: SocialConnection[] }>(`/api/v1/users/${params.id}/social`).catch(() => ({ connections: [] })),
      ]);
      setRoles((rolesResp as { roles?: Role[] }).roles || []);
      setCredentials((credsResp as { credentials?: Credential[] }).credentials || []);
      setSocialConnections((socialResp as { connections?: SocialConnection[] }).connections || []);

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
      setMsg("Role assigned successfully");
      loadUserRoles();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to assign role");
    }
  };

  const handleRevokeRole = async (roleId: string) => {
    try {
      await apiFetch(`/api/v1/users/${params.id}/roles/${roleId}`, {
        method: "DELETE",
      });
      setMsg("Role revoked successfully");
      loadUserRoles();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to revoke role");
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

  const assignedRoleIds = new Set(userRoles.map((r) => r.id));

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

      {/* Tab switcher */}
      <div className="mb-4 flex gap-2 border-b border-gray-200">
        <button
          onClick={() => setDetailTab("info")}
          className={`px-4 py-2 text-sm font-medium ${detailTab === "info" ? "border-b-2 border-brand-600 text-brand-600" : "text-gray-500"}`}
        >
          Profile
        </button>
        <button
          onClick={() => setDetailTab("roles")}
          className={`flex items-center gap-1.5 px-4 py-2 text-sm font-medium ${detailTab === "roles" ? "border-b-2 border-brand-600 text-brand-600" : "text-gray-500"}`}
        >
          <Shield className="h-4 w-4" /> Roles
        </button>
        <button
          onClick={() => setDetailTab("organizations")}
          className={`flex items-center gap-1.5 px-4 py-2 text-sm font-medium ${detailTab === "organizations" ? "border-b-2 border-brand-600 text-brand-600" : "text-gray-500"}`}
        >
          <Building2 className="h-4 w-4" /> Organizations
        </button>
        <button
          onClick={() => setDetailTab("activity")}
          className={`flex items-center gap-1.5 px-4 py-2 text-sm font-medium ${detailTab === "activity" ? "border-b-2 border-brand-600 text-brand-600" : "text-gray-500"}`}
        >
          <Activity className="h-4 w-4" /> Activity
        </button>
      </div>

      {/* ===== Activity Tab ===== */}
      {detailTab === "activity" && (
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <h3 className="mb-4 text-sm font-semibold">Recent Activity</h3>
          {activityLoading ? (
            <p className="text-gray-500">Loading...</p>
          ) : activity.length === 0 ? (
            <p className="py-8 text-center text-gray-400">No activity recorded</p>
          ) : (
            <div className="relative space-y-4">
              {activity.map((event, idx) => {
                const iconMap: Record<string, string> = {
                  "user.login": "bg-green-100 text-green-600",
                  "user.logout": "bg-gray-100 text-gray-600",
                  "user.register": "bg-blue-100 text-blue-600",
                  "user.password.change": "bg-amber-100 text-amber-600",
                  "role.assign": "bg-purple-100 text-purple-600",
                };
                return (
                  <div key={event.id} className="flex gap-3">
                    {idx < activity.length - 1 && (
                      <div className="absolute left-[19px] mt-8 h-[calc(100%-2rem)] w-px bg-gray-100" />
                    )}
                    <div className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-full ${iconMap[event.action] || "bg-gray-100 text-gray-500"}`}>
                      <Activity className="h-3.5 w-3.5" />
                    </div>
                    <div className="flex-1 pb-4">
                      <div className="flex items-center justify-between">
                        <span className="text-sm font-medium">{event.action}</span>
                        <span className="text-xs text-gray-400">
                          {new Date(event.created_at).toLocaleString()}
                        </span>
                      </div>
                      <div className="mt-1 flex items-center gap-2">
                        <span className={`rounded-full px-2 py-0.5 text-xs ${event.result === "success" ? "bg-green-50 text-green-700" : "bg-red-50 text-red-700"}`}>
                          {event.result}
                        </span>
                        {event.resource_type && (
                          <span className="text-xs text-gray-400">on {event.resource_type}</span>
                        )}
                        {event.ip_address && (
                          <span className="font-mono text-xs text-gray-400">{event.ip_address}</span>
                        )}
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}

      {/* ===== Roles Tab ===== */}
      {detailTab === "roles" && (
        <div className="grid gap-6 lg:grid-cols-2">
          {/* Assigned Roles */}
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
            <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold">
              <Shield className="h-4 w-4 text-brand-600" />
              Assigned Roles ({userRoles.length})
            </h3>
            {userRoles.length === 0 ? (
              <p className="py-6 text-center text-sm text-gray-400">No roles assigned</p>
            ) : (
              <div className="space-y-2">
                {userRoles.map((role) => (
                  <div
                    key={role.id}
                    className="flex items-center justify-between rounded-lg border border-gray-100 px-3 py-2"
                  >
                    <div className="flex items-center gap-2">
                      <Shield className="h-4 w-4 text-gray-400" />
                      <div>
                        <span className="text-sm font-medium">{role.name || role.key}</span>
                        {role.system_role && (
                          <span className="ml-2 rounded-full bg-amber-50 px-1.5 py-0.5 text-xs text-amber-600">
                            System
                          </span>
                        )}
                      </div>
                    </div>
                    {!role.system_role && (
                      <button
                        onClick={() => handleRevokeRole(role.id)}
                        className="flex items-center gap-1 rounded-lg border border-red-300 px-2 py-1 text-xs font-medium text-red-600 hover:bg-red-50"
                      >
                        <UserMinus className="h-3.5 w-3.5" /> Revoke
                      </button>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Available Roles */}
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
            <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold">
              <UserPlus className="h-4 w-4 text-brand-600" />
              Available Roles
            </h3>
            <div className="space-y-2">
              {roles
                .filter((r) => !assignedRoleIds.has(r.id))
                .map((role) => (
                  <div
                    key={role.id}
                    className="flex items-center justify-between rounded-lg border border-gray-100 px-3 py-2"
                  >
                    <div className="flex items-center gap-2">
                      <Shield className="h-4 w-4 text-gray-400" />
                      <div>
                        <span className="text-sm font-medium">{role.name || role.key}</span>
                        {role.description && (
                          <p className="text-xs text-gray-400">{role.description}</p>
                        )}
                      </div>
                    </div>
                    <button
                      onClick={() => handleAssignRole(role.id)}
                      className="flex items-center gap-1 rounded-lg border border-brand-300 px-2 py-1 text-xs font-medium text-brand-600 hover:bg-brand-50"
                    >
                      <UserPlus className="h-3.5 w-3.5" /> Assign
                    </button>
                  </div>
                ))}
              {roles.filter((r) => !assignedRoleIds.has(r.id)).length === 0 && (
                <p className="py-6 text-center text-sm text-gray-400">
                  All roles already assigned
                </p>
              )}
            </div>
          </div>
        </div>
      )}

      {/* ===== Organizations Tab ===== */}
      {detailTab === "organizations" && (
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold">
            <Building2 className="h-4 w-4 text-brand-600" />
            Organization Memberships ({orgs.length})
          </h3>
          {orgsLoading ? (
            <p className="py-8 text-center text-sm text-gray-500">Loading...</p>
          ) : orgs.length === 0 ? (
            <p className="py-8 text-center text-sm text-gray-400">
              Not a member of any organization
            </p>
          ) : (
            <div className="space-y-2">
              {orgs.map((org) => (
                <div
                  key={org.id}
                  className="flex items-center justify-between rounded-lg border border-gray-100 px-4 py-3"
                >
                  <div className="flex items-center gap-3">
                    <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-brand-50">
                      <Building2 className="h-5 w-5 text-brand-600" />
                    </div>
                    <div>
                      <span className="text-sm font-medium">{org.name}</span>
                      <p className="font-mono text-xs text-gray-400">{org.path}</p>
                    </div>
                  </div>
                  <span className="rounded-full bg-gray-100 px-2.5 py-0.5 text-xs font-medium text-gray-600">
                    {org.role || "member"}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* ===== Profile Tab (default) ===== */}
      {detailTab === "info" && (
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
                <div>
                  <label className="mb-1 block text-xs font-medium text-gray-500">Status</label>
                  <select
                    value={editForm.status}
                    onChange={(e) => setEditForm({ ...editForm, status: e.target.value })}
                    className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none"
                  >
                    <option value="active">Active</option>
                    <option value="inactive">Inactive</option>
                    <option value="locked">Locked</option>
                  </select>
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

            {/* Quick Roles Summary */}
            <div className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm">
              <div className="mb-3 flex items-center justify-between">
                <h3 className="flex items-center gap-1.5 text-sm font-semibold">
                  <Shield className="h-4 w-4 text-gray-400" /> Roles
                </h3>
                <button
                  onClick={() => setDetailTab("roles")}
                  className="text-xs font-medium text-brand-600 hover:underline"
                >
                  Manage →
                </button>
              </div>
              <div className="flex flex-wrap gap-1.5">
                {userRoles.length === 0 ? (
                  <span className="text-sm text-gray-400">No roles assigned</span>
                ) : (
                  userRoles.map((role) => (
                    <span
                      key={role.id}
                      className="rounded-full bg-brand-50 px-2.5 py-1 text-xs font-medium text-brand-700"
                    >
                      {role.name || role.key}
                    </span>
                  ))
                )}
              </div>
            </div>

            {/* WebAuthn Credentials */}
            <div className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm">
              <h3 className="mb-3 flex items-center gap-1.5 text-sm font-semibold">
                <Fingerprint className="h-4 w-4 text-gray-400" /> WebAuthn Credentials ({credentials.length})
              </h3>
              {credentials.length === 0 ? (
                <p className="text-sm text-gray-400">No WebAuthn credentials registered</p>
              ) : (
                <div className="space-y-1.5">
                  {credentials.map((cred) => (
                    <div key={cred.id} className="flex items-center justify-between rounded border border-gray-100 px-2 py-1.5">
                      <div className="flex items-center gap-2">
                        <Fingerprint className="h-3 w-3 text-gray-400" />
                        <span className="text-xs font-medium">{cred.name || cred.id.slice(0, 8)}</span>
                        {cred.type && <span className="rounded bg-blue-50 px-1.5 py-0.5 text-[10px] text-blue-600">{cred.type}</span>}
                      </div>
                      {cred.created_at && <span className="text-[10px] text-gray-400">{new Date(cred.created_at).toLocaleDateString()}</span>}
                    </div>
                  ))}
                </div>
              )}
            </div>

            {/* Social Connections */}
            <div className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm">
              <h3 className="mb-3 flex items-center gap-1.5 text-sm font-semibold">
                <Globe className="h-4 w-4 text-gray-400" /> Social Connections ({socialConnections.length})
              </h3>
              {socialConnections.length === 0 ? (
                <p className="text-sm text-gray-400">No social providers connected</p>
              ) : (
                <div className="space-y-1.5">
                  {socialConnections.map((conn) => (
                    <div key={conn.id} className="flex items-center justify-between rounded border border-gray-100 px-2 py-1.5">
                      <div className="flex items-center gap-2">
                        <Globe className="h-3 w-3 text-gray-400" />
                        <span className="text-xs font-medium capitalize">{conn.provider}</span>
                      </div>
                      {conn.email && <span className="text-[10px] text-gray-400">{conn.email}</span>}
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

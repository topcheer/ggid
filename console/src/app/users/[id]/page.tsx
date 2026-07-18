"use client";

import { useState, useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";
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
  Cloud,
  RefreshCw,
  Smartphone,
  Monitor,
  AlertTriangle,
  X,
  Clock,
  ShieldCheck,
  Loader2,
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

interface Session {
  id: string;
  device: string;
  ip_address: string;
  last_active: string;
  user_agent?: string;
  location?: string;
}

interface ScimData {
  external_id: string;
  source: string;
  last_synced: string;
}

type DrawerTab = "profile" | "sessions" | "credentials" | "audit";

function deviceIcon(ua?: string) {
  if (!ua) return Monitor;
  if (/mobile|android|iphone/i.test(ua)) return Smartphone;
  return Monitor;
}

function credentialTypeColor(type: string): string {
  switch (type) {
    case "password":
      return "bg-blue-50 text-blue-700 dark:bg-blue-950 dark:text-blue-400";
    case "totp":
      return "bg-purple-50 text-purple-700 dark:bg-purple-950 dark:text-purple-400";
    case "webauthn":
      return "bg-green-50 text-green-700 dark:bg-green-950 dark:text-green-400";
    case "social":
      return "bg-amber-50 text-amber-700 dark:bg-amber-950 dark:text-amber-400";
    default:
      return "bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300";
  }
}

export default function UserDetailPage({ params }: { params: { id: string } }) {
  const t = useTranslations();
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
  const [activeTab, setActiveTab] = useState<DrawerTab>("profile");
  const [activity, setActivity] = useState<AuditEvent[]>([]);
  const [activityLoading, setActivityLoading] = useState(false);
  const [credentials, setCredentials] = useState<Credential[]>([]);
  const [socialConnections, setSocialConnections] = useState<SocialConnection[]>([]);
  const [scimData, setScimData] = useState<ScimData | null>(null);
  const [scimLoading, setScimLoading] = useState(false);
  const [scimSyncing, setScimSyncing] = useState(false);

  // Sessions state
  const [sessions, setSessions] = useState<Session[]>([]);
  const [sessionsLoading, setSessionsLoading] = useState(false);
  const [revokingSession, setRevokingSession] = useState<string | null>(null);

  // Deleting credential state
  const [deletingCred, setDeletingCred] = useState<string | null>(null);

  // Impersonate dialog state
  const [showImpersonate, setShowImpersonate] = useState(false);
  const [impersonating, setImpersonating] = useState(false);

  const loadScim = useCallback(async () => {
    setScimLoading(true);
    try {
      const data = await apiFetch<ScimData>(
        `/api/v1/users/${params.id}/scim`,
      ).catch(() => null);
      setScimData(data);
    } catch {
      setScimData(null);
    } finally {
      setScimLoading(false);
    }
  }, [apiFetch, params.id]);

  const handleScimSync = async () => {
    setScimSyncing(true);
    try {
      await apiFetch(`/api/v1/users/${params.id}/scim/sync`, {
        method: "POST",
      });
      setMsg("SCIM sync triggered successfully");
      loadScim();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to sync SCIM");
    } finally {
      setScimSyncing(false);
    }
  };

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

  const loadSessions = useCallback(async () => {
    setSessionsLoading(true);
    try {
      const data = await apiFetch<{ sessions?: Session[] } | Session[]>(
        `/api/v1/users/${params.id}/sessions`,
      ).catch(() => ({ sessions: [] }));
      const list = Array.isArray(data) ? data : data.sessions || [];
      setSessions(list);
    } catch {
      setSessions([]);
    } finally {
      setSessionsLoading(false);
    }
  }, [apiFetch, params.id]);

  const handleRevokeSession = async (sessionId: string) => {
    setRevokingSession(sessionId);
    try {
      await apiFetch(`/api/v1/users/${params.id}/sessions/${sessionId}`, {
        method: "DELETE",
      });
      setMsg("Session revoked successfully");
      loadSessions();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to revoke session");
    } finally {
      setRevokingSession(null);
    }
  };

  const handleDeleteCredential = async (credId: string) => {
    if (!confirm("Delete this credential? This cannot be undone.")) return;
    setDeletingCred(credId);
    try {
      await apiFetch(`/api/v1/users/${params.id}/credentials/${credId}`, {
        method: "DELETE",
      });
      setMsg("Credential deleted successfully");
      setCredentials((prev) => prev.filter((c: any) => c.id !== credId));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete credential");
    } finally {
      setDeletingCred(null);
    }
  };

  const handleImpersonate = async () => {
    setImpersonating(true);
    try {
      const resp = await apiFetch<{ token?: string; access_token?: string }>(
        `/api/v1/users/${params.id}/impersonate`,
        { method: "POST" },
      );
      const token = resp.token || resp.access_token;
      if (token) {
        localStorage.setItem("ggid_access_token", token);
        setMsg("Impersonation active — you are now this user");
        window.location.href = "/";
      } else {
        setMsg("Impersonation token received");
      }
      setShowImpersonate(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to impersonate");
    } finally {
      setImpersonating(false);
    }
  };

  useEffect(() => {
    if (activeTab === "audit") loadActivity();
  }, [activeTab, loadActivity]);

  useEffect(() => {
    if (activeTab === "sessions") loadSessions();
  }, [activeTab, loadSessions]);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const u = await apiFetch<User>(`/api/v1/users/${params.id}`);
      setUser(u);
      setEditForm({ display_name: u.display_name || "", email: u.email || "", phone: u.phone || "", status: u.status || "active" });

      const [rolesResp, credsResp, socialResp, userRolesResp, orgsResp, scimResp] = await Promise.all([
        apiFetch<{ roles?: Role[] }>("/api/v1/roles").catch(() => ({ roles: [] })),
        apiFetch<{ credentials?: Credential[] }>(`/api/v1/users/${params.id}/credentials`).catch(() => ({ credentials: [] })),
        apiFetch<{ connections?: SocialConnection[] }>(`/api/v1/users/${params.id}/social`).catch(() => ({ connections: [] })),
        apiFetch<{ roles?: Role[] }>(`/api/v1/users/${params.id}/roles`).catch(() => ({ roles: [] })),
        apiFetch<{ organizations?: OrgMembership[] }>(`/api/v1/org/memberships?user_id=${params.id}`).catch(() => ({ organizations: [] })),
        apiFetch<ScimData>(`/api/v1/users/${params.id}/scim`).catch(() => null),
      ]);
      setRoles((rolesResp as { roles?: Role[] }).roles || []);
      setCredentials((credsResp as { credentials?: Credential[] }).credentials || []);
      setSocialConnections((socialResp as { connections?: SocialConnection[] }).connections || []);
      setUserRoles((userRolesResp as { roles?: Role[] }).roles || []);
      setOrgs((orgsResp as { organizations?: OrgMembership[] }).organizations || []);
      setScimData(scimResp);

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

  const closeDrawer = () => router.push("/users");

  if (loading)
    return (
      <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30" onClick={closeDrawer}>
        <Loader2 className="h-6 w-6 animate-spin text-white" />
      </div>
    );

  if (error && !user)
    return (
      <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30" onClick={closeDrawer}>
        <div className="rounded-lg bg-white p-6 text-red-700 dark:bg-gray-800 dark:text-red-400" onClick={(e) => e.stopPropagation()}>
          {error}
          <button aria-label="action" onClick={closeDrawer} className="mt-3 block text-sm text-brand-600">{t("userDetail.goBack")}</button>
        </div>
      </div>
    );

  if (!user)
    return (
      <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30" onClick={closeDrawer}>
        <p className="text-white">{t("userDetail.userNotFound")}</p>
      </div>
    );

  const assignedRoleIds = new Set(userRoles.map((r: any) => r.id));

  // Combine all credential types (password, totp, webauthn, social)
  const allCredentials: { id: string; type: string; name: string; created_at?: string; last_used_at?: string }[] = [
    ...credentials.map((c: any) => ({ id: c.id, type: c.type || "webauthn", name: c.name || c.id.slice(0, 8), created_at: c.created_at, last_used_at: c.last_used_at })),
    ...socialConnections.map((c: any) => ({ id: c.id, type: "social", name: `${c.provider}${c.name ? ` (${c.name})` : ""}`, created_at: c.created_at })),
  ];

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 z-40 bg-black/30 transition-opacity"
        onClick={closeDrawer}
      />

      {/* Slide-over drawer */}
      <div className="fixed right-0 top-0 z-50 flex h-full w-full max-w-2xl flex-col bg-gray-50 shadow-2xl dark:bg-gray-900 animate-[slideIn_0.2s_ease-out]">
        <style>{`
          @keyframes slideIn {
            from { transform: translateX(100%); }
            to { transform: translateX(0); }
          }
        `}</style>

        {/* Drawer header */}
        <div className="flex items-center justify-between border-b border-gray-200 bg-white px-6 py-4 dark:border-gray-700 dark:bg-gray-800">
          <div className="flex items-center gap-3">
            <button
              onClick={closeDrawer}
              className="flex items-center gap-1 text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
            >
              <ArrowLeft className="h-4 w-4" /> Close
            </button>
            <span className="text-gray-300">|</span>
            <h2 className="text-lg font-semibold dark:text-gray-100">{user.display_name || user.username}</h2>
            <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${
              user.status === "active"
                ? "bg-green-50 text-green-700 dark:bg-green-950 dark:text-green-400"
                : "bg-red-50 text-red-700 dark:bg-red-950 dark:text-red-400"
            }`}>
              {user.status}
            </span>
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setShowImpersonate(true)}
              className="flex items-center gap-1.5 rounded-lg border border-amber-300 bg-amber-50 px-3 py-1.5 text-xs font-medium text-amber-700 hover:bg-amber-100 dark:border-amber-700 dark:bg-amber-950 dark:text-amber-400"
            >
              <ShieldCheck className="h-3.5 w-3.5" /> Impersonate
            </button>
            <button
              onClick={closeDrawer}
              className="rounded-lg p-1.5 text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700"
             aria-label="Close">
              <X className="h-4 w-4" />
            </button>
          </div>
        </div>

        {/* Messages */}
        {msg && (
          <div role="status" className="mx-6 mt-3 rounded-lg border border-green-200 bg-green-50 p-2.5 text-sm text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400">
            {msg}
          </div>
        )}
        {error && (
          <div className="mx-6 mt-3 rounded-lg border border-red-200 bg-red-50 p-2.5 text-sm text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400">
            {error}
          </div>
        )}

        {/* Tab navigation */}
        <div className="flex gap-1 border-b border-gray-200 bg-white px-6 dark:border-gray-700 dark:bg-gray-800">
          {([
            { key: "profile" as const, label: "Profile" },
            { key: "sessions" as const, label: "Sessions" },
            { key: "credentials" as const, label: "Credentials" },
            { key: "audit" as const, label: "Audit Trail" },
          ]).map((tab: any) => (
            <button
              key={tab.key}
              onClick={() => setActiveTab(tab.key)}
              className={`relative px-4 py-3 text-sm font-medium transition ${
                activeTab === tab.key
                  ? "text-brand-600"
                  : "text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
              }`}
            >
              {tab.label}
              {activeTab === tab.key && (
                <span className="absolute bottom-0 left-0 h-0.5 w-full rounded-full bg-brand-600" />
              )}
            </button>
          ))}
        </div>

        {/* Drawer content — scrollable */}
        <div className="flex-1 overflow-y-auto px-6 py-4">
          {/* ── Profile Tab ── */}
          {activeTab === "profile" && (
            <div className="space-y-4">
              {/* User Info */}
              <div className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-800">
                <div className="mb-3 flex items-center justify-between">
                  <h3 className="text-sm font-semibold dark:text-gray-100">{t("userDetail.userInformation")}</h3>
                  <button
                    onClick={() => setEditing(!editing)}
                    className="rounded-lg px-2 py-1 text-xs font-medium text-brand-600 hover:bg-brand-50"
                  >
                    {editing ? "Cancel" : "Edit"}
                  </button>
                </div>

                {editing ? (
                  <div className="space-y-3">
                    <div>
                      <label className="mb-1 block text-xs font-medium text-gray-500">{t("userDetail.displayName")}</label>
                      <input
                        value={editForm.display_name}
                        onChange={(e) => setEditForm({ ...editForm, display_name: e.target.value })}
                        className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none dark:border-gray-600 dark:bg-gray-700"
                      />
                    </div>
                    <div>
                      <label className="mb-1 block text-xs font-medium text-gray-500">Email</label>
                      <input
                        value={editForm.email}
                        onChange={(e) => setEditForm({ ...editForm, email: e.target.value })}
                        className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none dark:border-gray-600 dark:bg-gray-700"
                      />
                    </div>
                    <div>
                      <label className="mb-1 block text-xs font-medium text-gray-500">{t("userDetail.phone")}</label>
                      <input
                        value={editForm.phone}
                        onChange={(e) => setEditForm({ ...editForm, phone: e.target.value })}
                        className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none dark:border-gray-600 dark:bg-gray-700"
                      />
                    </div>
                    <div>
                      <label className="mb-1 block text-xs font-medium text-gray-500">{t("common.status")}</label>
                      <select
                        value={editForm.status}
                        onChange={(e) => setEditForm({ ...editForm, status: e.target.value })}
                        className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none dark:border-gray-600 dark:bg-gray-700"
                      >
                        <option value="active">Active</option>
                        <option value="inactive">Inactive</option>
                        <option value="locked">{t("userDetail.locked")}</option>
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
                  <dl className="space-y-2">
                    <div className="flex justify-between">
                      <dt className="text-sm text-gray-500">{t("userDetail.username")}</dt>
                      <dd className="text-sm font-medium dark:text-gray-200">{user.username}</dd>
                    </div>
                    <div className="flex justify-between">
                      <dt className="text-sm text-gray-500">Email</dt>
                      <dd className="text-sm font-medium dark:text-gray-200">{user.email}</dd>
                    </div>
                    <div className="flex justify-between">
                      <dt className="text-sm text-gray-500">Display Name</dt>
                      <dd className="text-sm font-medium dark:text-gray-200">{user.display_name || "-"}</dd>
                    </div>
                    <div className="flex justify-between">
                      <dt className="text-sm text-gray-500">Email Verified</dt>
                      <dd className="text-sm font-medium dark:text-gray-200">{user.email_verified ? "Yes" : "No"}</dd>
                    </div>
                    {scimData && (
                      <>
                        <div className="flex justify-between border-t border-gray-100 pt-2 dark:border-gray-700">
                          <dt className="text-sm text-gray-500">External ID</dt>
                          <dd className="font-mono text-sm font-medium dark:text-gray-200">{scimData.external_id || "-"}</dd>
                        </div>
                        <div className="flex justify-between">
                          <dt className="text-sm text-gray-500">SCIM Source</dt>
                          <dd>
                            <span className="rounded-full bg-brand-50 px-2 py-0.5 text-xs font-medium capitalize text-brand-700">
                              {scimData.source || "unknown"}
                            </span>
                          </dd>
                        </div>
                      </>
                    )}
                    <div className="flex justify-between">
                      <dt className="text-sm text-gray-500">Created</dt>
                      <dd className="text-sm font-medium dark:text-gray-200">
                        {new Date(user.created_at).toLocaleDateString()}
                      </dd>
                    </div>
                  </dl>
                )}
              </div>

              {/* Roles & Orgs quick view */}
              <div className="grid gap-4 sm:grid-cols-2">
                <div className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-800">
                  <h3 className="mb-2 flex items-center gap-1.5 text-xs font-semibold dark:text-gray-100">
                    <Shield className="h-3.5 w-3.5 text-gray-400" /> Roles ({userRoles.length})
                  </h3>
                  <div className="flex flex-wrap gap-1.5">
                    {userRoles.length === 0 ? (
                      <span className="text-xs text-gray-400">{t("userDetail.noRoles")}</span>
                    ) : (
                      userRoles.map((role: any) => (
                        <span key={role.id} className="rounded-full bg-brand-50 px-2.5 py-0.5 text-xs font-medium text-brand-700">
                          {role.name || role.key}
                        </span>
                      ))
                    )}
                  </div>
                </div>
                <div className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-800">
                  <h3 className="mb-2 flex items-center gap-1.5 text-xs font-semibold dark:text-gray-100">
                    <Building2 className="h-3.5 w-3.5 text-gray-400" /> Orgs ({orgs.length})
                  </h3>
                  {orgs.length === 0 ? (
                    <span className="text-xs text-gray-400">{t("userDetail.noMemberships")}</span>
                  ) : (
                    orgs.map((org: any) => (
                      <div key={org.id} className="text-xs text-gray-600 dark:text-gray-300">
                        {org.name} <span className="text-gray-400">({org.role || "member"})</span>
                      </div>
                    ))
                  )}
                </div>
              </div>

              {/* Actions */}
              <div className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-800">
                <h3 className="mb-2 text-xs font-semibold dark:text-gray-100">{t("userDetail.accountActions")}</h3>
                <div className="flex flex-wrap gap-2">
                  {user.status === "locked" ? (
                    <button
                      onClick={() => handleLock(false)}
                      className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-1.5 text-xs font-medium hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300"
                    >
                      <Unlock className="h-3.5 w-3.5" /> Unlock
                    </button>
                  ) : (
                    <button
                      onClick={() => handleLock(true)}
                      className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-1.5 text-xs font-medium hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300"
                    >
                      <Lock className="h-3.5 w-3.5" /> Lock
                    </button>
                  )}
                  <button
                    onClick={handleDelete}
                    className="flex items-center gap-1.5 rounded-lg border border-red-300 px-3 py-1.5 text-xs font-medium text-red-600 hover:bg-red-50"
                  >
                    <Trash2 className="h-3.5 w-3.5" /> Delete
                  </button>
                </div>
              </div>

              {/* Reset Password */}
              <div className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-800">
                <h3 className="mb-2 text-xs font-semibold dark:text-gray-100">{t("userDetail.resetPassword")}</h3>
                <div className="flex gap-2">
                  <input
                    type="password"
                    value={resetPassword}
                    onChange={(e) => setResetPassword(e.target.value)}
                    placeholder="New password"
                    className="flex-1 rounded-lg border border-gray-300 px-3 py-1.5 text-sm focus:border-brand-500 focus:outline-none dark:border-gray-600 dark:bg-gray-700"
                  />
                  <button
                    onClick={handleResetPassword}
                    disabled={!resetPassword}
                    className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
                  >
                    <KeyRound className="h-3.5 w-3.5" /> Reset
                  </button>
                </div>
              </div>
            </div>
          )}

          {/* ── Sessions Tab ── */}
          {activeTab === "sessions" && (
            <div className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-800">
              <h3 className="mb-3 text-sm font-semibold dark:text-gray-100">Active Sessions ({sessions.length})</h3>
              {sessionsLoading ? (
                <p className="py-4 text-center text-sm text-gray-500">Loading...</p>
              ) : sessions.length === 0 ? (
                <p className="py-6 text-center text-sm text-gray-400">No active sessions</p>
              ) : (
                <div className="space-y-2">
                  {sessions.map((s: any) => {
                    const DevIcon = deviceIcon(s.user_agent);
                    return (
                      <div
                        key={s.id}
                        className="flex items-center justify-between rounded-lg border border-gray-100 p-3 dark:border-gray-700"
                      >
                        <div className="flex items-center gap-3">
                          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-gray-50 dark:bg-gray-700">
                            <DevIcon className="h-4 w-4 text-gray-500 dark:text-gray-400" />
                          </div>
                          <div>
                            <div className="text-sm font-medium dark:text-gray-200">{s.device || s.user_agent || "Unknown device"}</div>
                            <div className="flex items-center gap-2 text-xs text-gray-400">
                              <span className="font-mono">{s.ip_address || "—"}</span>
                              {s.last_active && (
                                <>
                                  <span>•</span>
                                  <span className="flex items-center gap-0.5">
                                    <Clock className="h-3 w-3" />
                                    {new Date(s.last_active).toLocaleString()}
                                  </span>
                                </>
                              )}
                              {s.location && <span>• {s.location}</span>}
                            </div>
                          </div>
                        </div>
                        <button
                          onClick={() => handleRevokeSession(s.id)}
                          disabled={revokingSession === s.id}
                          className="flex items-center gap-1 rounded-lg border border-red-300 px-2 py-1 text-xs font-medium text-red-600 hover:bg-red-50 disabled:opacity-50"
                        >
                          {revokingSession === s.id ? (
                            <RefreshCw className="h-3 w-3 animate-spin" />
                          ) : (
                            <X className="h-3 w-3" />
                          )}
                          Revoke
                        </button>
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          )}

          {/* ── Credentials Tab ── */}
          {activeTab === "credentials" && (
            <div className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-800">
              <h3 className="mb-3 text-sm font-semibold dark:text-gray-100">Credentials ({allCredentials.length})</h3>
              {allCredentials.length === 0 ? (
                <p className="py-6 text-center text-sm text-gray-400">No credentials registered</p>
              ) : (
                <div className="space-y-2">
                  {allCredentials.map((cred: any) => {
                    const Icon = cred.type === "social" ? Globe : cred.type === "webauthn" ? Fingerprint : cred.type === "totp" ? Shield : KeyRound;
                    return (
                      <div
                        key={cred.id}
                        className="flex items-center justify-between rounded-lg border border-gray-100 p-3 dark:border-gray-700"
                      >
                        <div className="flex items-center gap-3">
                          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-gray-50 dark:bg-gray-700">
                            <Icon className="h-4 w-4 text-gray-500 dark:text-gray-400" />
                          </div>
                          <div>
                            <div className="flex items-center gap-2">
                              <span className="text-sm font-medium dark:text-gray-200">{cred.name}</span>
                              <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${credentialTypeColor(cred.type)}`}>
                                {cred.type}
                              </span>
                            </div>
                            <div className="flex items-center gap-2 text-xs text-gray-400">
                              {cred.created_at && (
                                <span>Created: {new Date(cred.created_at).toLocaleDateString()}</span>
                              )}
                              {cred.last_used_at && (
                                <>
                                  <span>•</span>
                                  <span>Last used: {new Date(cred.last_used_at).toLocaleDateString()}</span>
                                </>
                              )}
                            </div>
                          </div>
                        </div>
                        <button
                          onClick={() => handleDeleteCredential(cred.id)}
                          disabled={deletingCred === cred.id}
                          className="text-red-500 hover:text-red-700 disabled:opacity-50"
                          title="Delete credential"
                        >
                          {deletingCred === cred.id ? (
                            <RefreshCw className="h-3.5 w-3.5 animate-spin" />
                          ) : (
                            <Trash2 className="h-3.5 w-3.5" />
                          )}
                        </button>
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          )}

          {/* ── Audit Trail Tab ── */}
          {activeTab === "audit" && (
            <div className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-800">
              <h3 className="mb-3 text-sm font-semibold dark:text-gray-100">Recent Events</h3>
              {activityLoading ? (
                <p className="py-4 text-center text-sm text-gray-500">Loading...</p>
              ) : activity.length === 0 ? (
                <p className="py-6 text-center text-sm text-gray-400">No events recorded</p>
              ) : (
                <div className="space-y-1">
                  {activity.slice(0, 20).map((event: any) => (
                    <div
                      key={event.id}
                      className="flex items-start justify-between rounded-lg border border-gray-50 p-2.5 dark:border-gray-700/50"
                    >
                      <div className="flex items-center gap-2">
                        <div className={`flex h-7 w-7 items-center justify-center rounded-full ${
                          event.result === "success"
                            ? "bg-green-100 dark:bg-green-950"
                            : "bg-red-100 dark:bg-red-950"
                        }`}>
                          <Activity className={`h-3 w-3 ${event.result === "success" ? "text-green-600" : "text-red-600"}`} />
                        </div>
                        <div>
                          <span className="text-sm font-medium dark:text-gray-200">{event.action}</span>
                          {event.resource_type && (
                            <span className="ml-2 text-xs text-gray-400">on {event.resource_type}</span>
                          )}
                          <div className="flex items-center gap-2 text-xs text-gray-400">
                            {event.ip_address && <span className="font-mono">{event.ip_address}</span>}
                            <span>•</span>
                            <span>{new Date(event.created_at).toLocaleString()}</span>
                          </div>
                        </div>
                      </div>
                      <span className={`rounded-full px-2 py-0.5 text-xs ${
                        event.result === "success"
                          ? "bg-green-50 text-green-700 dark:bg-green-950 dark:text-green-400"
                          : "bg-red-50 text-red-700 dark:bg-red-950 dark:text-red-400"
                      }`}>
                        {event.result}
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {/* ── Impersonate Confirmation Dialog ── */}
      {showImpersonate && (
        <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/50">
          <div role="dialog" aria-modal="true" className="mx-4 max-w-md rounded-xl bg-white p-6 shadow-2xl dark:bg-gray-800">
            <div className="mb-4 flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-amber-100 dark:bg-amber-950">
                <AlertTriangle className="h-5 w-5 text-amber-600" />
              </div>
              <h3 className="text-lg font-semibold dark:text-gray-100">Impersonate User</h3>
            </div>
            <div className="mb-4 space-y-2">
              <p className="text-sm text-gray-600 dark:text-gray-300">
                You are about to impersonate <strong>{user.display_name || user.username}</strong> ({user.email}).
              </p>
              <div className="rounded-lg border border-amber-200 bg-amber-50 p-3 dark:border-amber-800 dark:bg-amber-950/50">
                <p className="text-xs text-amber-700 dark:text-amber-400">
                  <strong>Warning:</strong> All actions performed during impersonation will be attributed to you, not the impersonated user. This action is logged for audit purposes. Make sure you have explicit authorization.
                </p>
              </div>
            </div>
            <div className="flex justify-end gap-2">
              <button
                onClick={() => setShowImpersonate(false)}
                disabled={impersonating}
                className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-600 hover:bg-gray-50 disabled:opacity-50 dark:border-gray-600 dark:text-gray-300"
              >
                Cancel
              </button>
              <button
                onClick={handleImpersonate}
                disabled={impersonating}
                className="flex items-center gap-1.5 rounded-lg bg-amber-600 px-4 py-2 text-sm font-medium text-white hover:bg-amber-700 disabled:opacity-50"
              >
                {impersonating ? (
                  <RefreshCw className="h-4 w-4 animate-spin" />
                ) : (
                  <ShieldCheck className="h-4 w-4" />
                )}
                Confirm Impersonation
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}

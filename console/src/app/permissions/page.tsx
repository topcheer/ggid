"use client";

import { useState, useMemo, useCallback, useEffect } from "react";
import { useApi } from "@/lib/api";
import {
  Shield,
  Search,
  ChevronDown,
  ChevronRight,
  KeyRound,
  Lock,
  Unlock,
  Plus,
  X,
  CheckSquare,
  Square,
  Crown,
  Fingerprint,
  Users as UsersIcon,
  Network,
  FileText,
  Key,
} from "lucide-react";

// ===== Types =====

interface Role {
  id: string;
  key: string;
  name: string;
  description: string;
  system_role: boolean;
}

// ===== Permission Definitions =====

interface PermDef {
  key: string;
  description: string;
}

interface ServiceGroup {
  service: string;
  icon: typeof KeyRound;
  color: string;
  bgColor: string;
  textColor: string;
  permissions: PermDef[];
}

const SERVICE_GROUPS: ServiceGroup[] = [
  {
    service: "Auth",
    icon: Fingerprint,
    color: "purple",
    bgColor: "bg-purple-100 dark:bg-purple-900/30",
    textColor: "text-purple-700 dark:text-purple-400",
    permissions: [
      { key: "login", description: "Authenticate and establish a session" },
      { key: "logout", description: "Terminate the current session" },
      { key: "password.reset", description: "Initiate and complete password resets" },
      { key: "mfa.setup", description: "Configure multi-factor authentication" },
      { key: "mfa.verify", description: "Verify MFA challenges during login" },
      { key: "token.refresh", description: "Exchange refresh tokens for access tokens" },
      { key: "token.revoke", description: "Revoke active tokens and sessions" },
    ],
  },
  {
    service: "Identity",
    icon: UsersIcon,
    color: "blue",
    bgColor: "bg-blue-100 dark:bg-blue-900/30",
    textColor: "text-blue-700 dark:text-blue-400",
    permissions: [
      { key: "users.create", description: "Create new user accounts" },
      { key: "users.read", description: "View user profiles and details" },
      { key: "users.update", description: "Modify user attributes and settings" },
      { key: "users.delete", description: "Permanently delete user accounts" },
      { key: "users.export", description: "Export user data in bulk" },
      { key: "users.import", description: "Import users from external sources" },
    ],
  },
  {
    service: "Policy",
    icon: Crown,
    color: "amber",
    bgColor: "bg-amber-100 dark:bg-amber-900/30",
    textColor: "text-amber-700 dark:text-amber-400",
    permissions: [
      { key: "policies.create", description: "Create new RBAC/ABAC policies" },
      { key: "policies.read", description: "View and inspect policies" },
      { key: "policies.update", description: "Modify existing policy rules" },
      { key: "policies.delete", description: "Delete policies" },
      { key: "policies.evaluate", description: "Evaluate access decisions against policies" },
      { key: "roles.assign", description: "Assign or revoke roles from users" },
    ],
  },
  {
    service: "Organization",
    icon: Network,
    color: "green",
    bgColor: "bg-green-100 dark:bg-green-900/30",
    textColor: "text-green-700 dark:text-green-400",
    permissions: [
      { key: "orgs.create", description: "Create new organizations" },
      { key: "orgs.read", description: "View organization hierarchies" },
      { key: "orgs.update", description: "Modify organization details and structure" },
      { key: "orgs.delete", description: "Delete organizations" },
      { key: "orgs.manage_members", description: "Add or remove org members" },
    ],
  },
  {
    service: "Audit",
    icon: FileText,
    color: "rose",
    bgColor: "bg-rose-100 dark:bg-rose-900/30",
    textColor: "text-rose-700 dark:text-rose-400",
    permissions: [
      { key: "audit.read", description: "Query and view audit event logs" },
      { key: "audit.export", description: "Export audit logs for compliance" },
      { key: "audit.delete", description: "Delete or purge audit records" },
      { key: "reports.create", description: "Create custom audit reports" },
      { key: "reports.schedule", description: "Schedule recurring report delivery" },
    ],
  },
  {
    service: "OAuth",
    icon: Key,
    color: "indigo",
    bgColor: "bg-indigo-100 dark:bg-indigo-900/30",
    textColor: "text-indigo-700 dark:text-indigo-400",
    permissions: [
      { key: "clients.create", description: "Register new OAuth client applications" },
      { key: "clients.read", description: "View OAuth client configurations" },
      { key: "clients.update", description: "Modify client settings and redirect URIs" },
      { key: "clients.delete", description: "Remove OAuth client registrations" },
      { key: "tokens.revoke", description: "Revoke issued OAuth tokens" },
      { key: "consent.manage", description: "Manage user consent records" },
    ],
  },
];

// Flatten all permissions with their service context
const ALL_PERMS = SERVICE_GROUPS.flatMap((g) =>
  g.permissions.map((p) => ({ ...p, service: g.service, color: g.color, bgColor: g.bgColor, textColor: g.textColor })),
);

// ===== Default Roles =====

const DEFAULT_ROLES: Role[] = [
  { id: "admin", key: "admin", name: "Admin", description: "Full system access", system_role: true },
  { id: "manager", key: "manager", name: "Manager", description: "Manage users and orgs", system_role: false },
  { id: "editor", key: "editor", name: "Editor", description: "Edit content", system_role: false },
  { id: "viewer", key: "viewer", name: "Viewer", description: "Read-only access", system_role: false },
  { id: "guest", key: "guest", name: "Guest", description: "Limited access", system_role: false },
];

// Role color mapping (for badges)
const ROLE_COLORS: Record<string, string> = {
  admin: "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400",
  manager: "bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400",
  editor: "bg-teal-100 text-teal-700 dark:bg-teal-900/30 dark:text-teal-400",
  viewer: "bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-300",
  guest: "bg-slate-100 text-slate-700 dark:bg-slate-900/30 dark:text-slate-400",
};

const getRoleBadgeClass = (roleKey: string): string =>
  ROLE_COLORS[roleKey] || "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400";

// ===== Main Component =====

export default function PermissionsPage() {
  const { apiFetch } = useApi();
  const [roles, setRoles] = useState<Role[]>(DEFAULT_ROLES);

  // permission → set of role keys that have it
  const [permRoles, setPermRoles] = useState<Record<string, Set<string>>>(() => {
    // Default: admin has all, manager/edirot/viewer have read perms
    const m: Record<string, Set<string>> = {};
    ALL_PERMS.forEach((p) => {
      m[p.key] = new Set<string>(["admin"]);
    });
    // Give manager some defaults
    ["login", "logout", "password.reset", "mfa.setup", "mfa.verify", "token.refresh",
     "users.create", "users.read", "users.update", "users.export",
     "policies.read", "policies.evaluate", "roles.assign",
     "orgs.create", "orgs.read", "orgs.update", "orgs.manage_members",
     "audit.read"].forEach((k) => m[k]?.add("manager"));
    ["login", "logout", "password.reset", "token.refresh",
     "users.read", "users.update",
     "policies.read", "policies.evaluate",
     "orgs.read", "audit.read"].forEach((k) => m[k]?.add("editor"));
    ["login", "logout", "password.reset", "token.refresh",
     "users.read", "policies.read", "orgs.read", "audit.read"].forEach((k) => m[k]?.add("viewer"));
    ["login", "logout"].forEach((k) => m[k]?.add("guest"));
    return m;
  });

  const [search, setSearch] = useState("");
  const [collapsedServices, setCollapsedServices] = useState<Set<string>>(new Set());
  const [expandedPerms, setExpandedPerms] = useState<Set<string>>(new Set());

  // Batch grant state
  const [selectedPerms, setSelectedPerms] = useState<Set<string>>(new Set());
  const [batchRole, setBatchRole] = useState("");

  const [msg, setMsg] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  // ---- Load roles from API ----
  useEffect(() => {
    const loadRoles = async () => {
      try {
        const data = await apiFetch<{ roles?: Role[] }>("/api/v1/roles");
        if (data.roles && data.roles.length > 0) {
          setRoles(data.roles);
        }
      } catch {
        // Fall back to defaults
      } finally {
        setLoading(false);
      }
    };
    loadRoles();
  }, [apiFetch]);

  // Auto-dismiss messages
  useEffect(() => {
    if (msg) {
      const t = setTimeout(() => setMsg(null), 3000);
      return () => clearTimeout(t);
    }
  }, [msg]);

  // ---- Derived: filtered permissions ----
  const filteredPerms = useMemo(() => {
    if (!search.trim()) return ALL_PERMS;
    const q = search.toLowerCase();
    return ALL_PERMS.filter(
      (p) => p.key.toLowerCase().includes(q) || p.description.toLowerCase().includes(q) || p.service.toLowerCase().includes(q),
    );
  }, [search]);

  const filteredPermKeys = useMemo(() => new Set(filteredPerms.map((p) => p.key)), [filteredPerms]);

  // Per-service filtered groups
  const visibleGroups = useMemo(() => {
    return SERVICE_GROUPS.map((group) => ({
      ...group,
      permissions: group.permissions.filter((p) => filteredPermKeys.has(p.key)),
    })).filter((g) => g.permissions.length > 0);
  }, [filteredPermKeys]);

  // ---- Toggle handlers ----
  const toggleService = useCallback((service: string) => {
    setCollapsedServices((prev) => {
      const next = new Set(prev);
      if (next.has(service)) {
        next.delete(service);
      } else {
        next.add(service);
      }
      return next;
    });
  }, []);

  const togglePermExpand = useCallback((permKey: string) => {
    setExpandedPerms((prev) => {
      const next = new Set(prev);
      if (next.has(permKey)) {
        next.delete(permKey);
      } else {
        next.add(permKey);
      }
      return next;
    });
  }, []);

  const toggleRoleForPerm = useCallback((permKey: string, roleKey: string) => {
    setPermRoles((prev) => {
      const next = { ...prev };
      const set = new Set(next[permKey] || []);
      if (set.has(roleKey)) {
        set.delete(roleKey);
      } else {
        set.add(roleKey);
      }
      next[permKey] = set;
      return next;
    });
    // Persist to API
    apiFetch("/api/v1/roles/permissions", {
      method: "PUT",
      body: JSON.stringify({ permission: permKey, roles: [...(permRoles[permKey] || [])] }),
    }).catch(() => {});
  }, [apiFetch, permRoles]);

  // ---- Batch grant ----
  const toggleSelectedPerm = (permKey: string) => {
    setSelectedPerms((prev) => {
      const next = new Set(prev);
      if (next.has(permKey)) {
        next.delete(permKey);
      } else {
        next.add(permKey);
      }
      return next;
    });
  };

  const handleBatchGrant = async () => {
    if (!batchRole) {
      setError("Select a role to grant permissions to");
      return;
    }
    if (selectedPerms.size === 0) {
      setError("Select at least one permission");
      return;
    }
    setError(null);
    setPermRoles((prev) => {
      const next = { ...prev };
      selectedPerms.forEach((permKey) => {
        const set = new Set(next[permKey] || []);
        set.add(batchRole);
        next[permKey] = set;
      });
      return next;
    });
    try {
      await apiFetch("/api/v1/roles/permissions/batch", {
        method: "POST",
        body: JSON.stringify({
          role_key: batchRole,
          permissions: [...selectedPerms],
        }),
      });
    } catch {
      // API may not exist yet — local state is updated
    }
    const roleName = roles.find((r) => r.key === batchRole)?.name || batchRole;
    setMsg(`Granted ${selectedPerms.size} permission(s) to ${roleName}`);
    setSelectedPerms(new Set());
    setBatchRole("");
  };

  // ---- Render ----
  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <p className="text-gray-500 dark:text-gray-400">Loading permissions...</p>
      </div>
    );
  }

  return (
    <div>
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold dark:text-gray-100">
            <Shield className="h-6 w-6 text-brand-600" />
            Permission Explorer
          </h1>
          <p className="mt-1 text-sm text-gray-500">
            Browse all permissions grouped by service. Click to expand and manage role assignments.
          </p>
        </div>
      </div>

      {/* Messages */}
      {msg && (
        <div className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400">
          {msg}
        </div>
      )}
      {error && (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400">
          {error}
        </div>
      )}

      {/* Search + Batch grant toolbar */}
      <div className="mb-4 flex flex-wrap items-center gap-3">
        {/* Search */}
        <div className="relative flex-1 min-w-[200px]">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
          <input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search permissions by name or description..."
            className="w-full rounded-lg border border-gray-300 py-2 pl-9 pr-3 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 focus:border-brand-500 focus:outline-none"
          />
        </div>

        {/* Batch grant controls */}
        <div className="flex items-center gap-2">
          <span className="text-sm text-gray-500">
            {selectedPerms.size > 0 ? `${selectedPerms.size} selected` : "No selection"}
          </span>
          <select
            value={batchRole}
            onChange={(e) => setBatchRole(e.target.value)}
            className="rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
          >
            <option value="">Select role...</option>
            {roles.map((r) => (
              <option key={r.id} value={r.key}>
                {r.name}
              </option>
            ))}
          </select>
          <button
            onClick={handleBatchGrant}
            disabled={selectedPerms.size === 0 || !batchRole}
            className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            <Plus className="h-4 w-4" /> Grant to Role
          </button>
          {selectedPerms.size > 0 && (
            <button
              onClick={() => setSelectedPerms(new Set())}
              className="flex items-center gap-1 rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-400 dark:hover:bg-gray-700"
            >
              <X className="h-4 w-4" /> Clear
            </button>
          )}
        </div>
      </div>

      {/* Permission groups */}
      <div className="space-y-3">
        {visibleGroups.map((group) => {
          const Icon = group.icon;
          const isCollapsed = collapsedServices.has(group.service);
          const allPermKeys = group.permissions.map((p) => p.key);
          const allSelected = allPermKeys.every((k) => selectedPerms.has(k));

          return (
            <div
              key={group.service}
              className="overflow-hidden rounded-xl border border-gray-200 bg-white shadow dark:border-gray-700 dark:bg-gray-800-sm dark:border-gray-700 dark:bg-gray-800"
            >
              {/* Service header */}
              <div
                className="flex cursor-pointer items-center justify-between border-b border-gray-200 px-4 py-3 hover:bg-gray-50 dark:border-gray-700 dark:hover:bg-gray-700/50"
                onClick={() => toggleService(group.service)}
              >
                <div className="flex items-center gap-3">
                  <div className={`flex h-8 w-8 items-center justify-center rounded-lg ${group.bgColor}`}>
                    <Icon className={`h-5 w-5 ${group.textColor}`} />
                  </div>
                  <div>
                    <span className="flex items-center gap-2 font-bold text-gray-800 dark:text-gray-200">
                      {isCollapsed ? <ChevronRight className="h-4 w-4 text-gray-400" /> : <ChevronDown className="h-4 w-4 text-gray-400" />}
                      {group.service}
                    </span>
                    <span className="text-xs text-gray-400">{group.permissions.length} permissions</span>
                  </div>
                </div>
                {/* Select all in group */}
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    if (allSelected) {
                      setSelectedPerms((prev) => {
                        const next = new Set(prev);
                        allPermKeys.forEach((k) => next.delete(k));
                        return next;
                      });
                    } else {
                      setSelectedPerms((prev) => {
                        const next = new Set(prev);
                        allPermKeys.forEach((k) => next.add(k));
                        return next;
                      });
                    }
                  }}
                  className="flex items-center gap-1 rounded-lg px-2 py-1 text-xs text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700"
                >
                  {allSelected ? (
                    <CheckSquare className="h-4 w-4 text-brand-600" />
                  ) : (
                    <Square className="h-4 w-4" />
                  )}
                  Select all
                </button>
              </div>

              {/* Permission rows */}
              {!isCollapsed && (
                <div className="divide-y divide-gray-100 dark:divide-gray-700/50">
                  {group.permissions.map((perm) => {
                    const fullPerm = ALL_PERMS.find((p) => p.key === perm.key)!;
                    const rolesWithPerm = roles.filter((r) => (permRoles[perm.key] || new Set()).has(r.key));
                    const isExpanded = expandedPerms.has(perm.key);
                    const isSelected = selectedPerms.has(perm.key);
                    const rolesSet = permRoles[perm.key] || new Set<string>();

                    return (
                      <div key={perm.key} className="transition-colors hover:bg-gray-50 dark:hover:bg-gray-700/30">
                        {/* Permission row */}
                        <div className="flex items-start gap-3 px-4 py-3">
                          {/* Checkbox for batch */}
                          <button
                            onClick={() => toggleSelectedPerm(perm.key)}
                            className="mt-1 flex-shrink-0"
                          >
                            {isSelected ? (
                              <CheckSquare className="h-5 w-5 text-brand-600" />
                            ) : (
                              <Square className="h-5 w-5 text-gray-300 hover:text-gray-400" />
                            )}
                          </button>

                          {/* Permission info */}
                          <div className="min-w-0 flex-1">
                            <div className="flex items-center gap-2">
                              <button
                                onClick={() => togglePermExpand(perm.key)}
                                className="flex items-center gap-1"
                              >
                                {isExpanded ? (
                                  <ChevronDown className="h-3.5 w-3.5 text-gray-400" />
                                ) : (
                                  <ChevronRight className="h-3.5 w-3.5 text-gray-400" />
                                )}
                              </button>
                              <code className="rounded bg-gray-100 px-1.5 py-0.5 font-mono text-sm font-semibold text-gray-800 dark:bg-gray-700 dark:text-gray-200">
                                {perm.key}
                              </code>
                              <span className={`rounded-full px-2 py-0.5 text-[10px] font-bold uppercase ${group.bgColor} ${group.textColor}`}>
                                {group.service}
                              </span>
                            </div>
                            <p className="ml-6 mt-0.5 text-sm text-gray-500">{perm.description}</p>

                            {/* Role badges */}
                            <div className="ml-6 mt-2 flex flex-wrap items-center gap-1.5">
                              {rolesWithPerm.length === 0 ? (
                                <span className="text-xs italic text-gray-400">No roles assigned</span>
                              ) : (
                                rolesWithPerm.map((r) => (
                                  <span
                                    key={r.id}
                                    className={`flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${getRoleBadgeClass(r.key)}`}
                                  >
                                    {r.system_role && <Lock className="h-2.5 w-2.5" />}
                                    {r.name}
                                  </span>
                                ))
                              )}
                              <span className="ml-1 text-xs text-gray-400">
                                ({rolesWithPerm.length} role{rolesWithPerm.length !== 1 ? "s" : ""})
                              </span>
                            </div>
                          </div>

                          {/* Expand toggle */}
                          <button
                            onClick={() => togglePermExpand(perm.key)}
                            className="mt-1 flex-shrink-0 rounded-lg p-1 text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700"
                            title={isExpanded ? "Collapse" : "Expand to manage roles"}
                          >
                            {isExpanded ? <Unlock className="h-4 w-4" /> : <KeyRound className="h-4 w-4" />}
                          </button>
                        </div>

                        {/* Expanded: role management */}
                        {isExpanded && (
                          <div className="border-t border-gray-100 bg-gray-50 px-4 py-3 dark:border-gray-700/50 dark:bg-gray-900/30">
                            <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-gray-500">
                              Manage roles for "{perm.key}"
                            </p>
                            <div className="flex flex-wrap gap-2">
                              {roles.map((r) => {
                                const has = rolesSet.has(r.key);
                                return (
                                  <button
                                    key={r.id}
                                    onClick={() => toggleRoleForPerm(perm.key, r.key)}
                                    className={`flex items-center gap-1.5 rounded-lg border px-3 py-1.5 text-sm font-medium transition-all ${
                                      has
                                        ? "border-brand-400 bg-brand-50 text-brand-700 dark:border-brand-600 dark:bg-brand-900/30 dark:text-brand-400"
                                        : "border-gray-300 text-gray-500 hover:border-gray-400 dark:border-gray-600 dark:text-gray-400"
                                    }`}
                                  >
                                    {has ? <CheckSquare className="h-3.5 w-3.5" /> : <Square className="h-3.5 w-3.5" />}
                                    {r.system_role && <Lock className="h-3 w-3 text-gray-400" />}
                                    {r.name}
                                  </button>
                                );
                              })}
                            </div>
                          </div>
                        )}
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          );
        })}
      </div>

      {/* Empty state */}
      {visibleGroups.length === 0 && (
        <div className="rounded-xl border border-gray-200 bg-white p-12 text-center shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <Search className="mx-auto mb-4 h-12 w-12 text-gray-300" />
          <p className="text-gray-500">No permissions match "{search}"</p>
        </div>
      )}

      {/* Summary footer */}
      <div className="mt-6 flex items-center justify-between rounded-xl bg-gray-50 px-4 py-3 text-sm text-gray-500 dark:bg-gray-800/50 dark:text-gray-400">
        <span>
          {ALL_PERMS.length} total permissions across {SERVICE_GROUPS.length} services
        </span>
        <span>{roles.length} roles available</span>
      </div>
    </div>
  );
}

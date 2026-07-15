"use client";

import { useEffect, useState, useCallback, useMemo } from "react";
import { useApi } from "@/lib/api";
import {
  Shield,
  Lock,
  CheckCircle2,
  XCircle,
  Save,
  Layers,
  ChevronRight,
  ArrowRight,
  GitBranch,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

// ===== Types =====

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

// ===== Permission Group Definitions =====

const PERMISSION_GROUPS: { label: string; permissions: string[] }[] = [
  {
    label: "Identity",
    permissions: [
      "users.create",
      "users.read",
      "users.update",
      "users.delete",
      "roles.create",
      "roles.read",
    ],
  },
  {
    label: "Policy",
    permissions: ["policies.create", "policies.read", "policies.update"],
  },
  {
    label: "OAuth",
    permissions: ["clients.create", "clients.read", "tokens.revoke"],
  },
  {
    label: "Audit",
    permissions: ["audit.read", "audit.export"],
  },
];

const ALL_PERMISSION_KEYS = PERMISSION_GROUPS.flatMap((g) => g.permissions);

// ===== Main Component =====

export default function RolePermissionsMatrixPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [roles, setRoles] = useState<Role[]>([]);
  const [permissions, setPermissions] = useState<Permission[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [msg, setMsg] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  // matrix[roleId] = Set<permissionId>  (local working copy, not yet saved)
  const [matrix, setMatrix] = useState<Record<string, Set<string>>>({});
  // original snapshot for dirty-check
  const [originalMatrix, setOriginalMatrix] = useState<Record<string, Set<string>>>({});

  // Bulk assign state
  const [bulkRole, setBulkRole] = useState("");
  const [bulkGroup, setBulkGroup] = useState<string>("");

  // ---- Data loading ----
  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const [rolesResp, permsResp] = await Promise.all([
        apiFetch<{ roles?: Role[]; items?: Role[] }>("/api/v1/roles").catch(() => ({ roles: [] as Role[] })),
        apiFetch<{ permissions?: Permission[]; items?: Permission[] }>("/api/v1/permissions").catch(() => ({ permissions: [] as Permission[] })),
      ]);
      const roleList = (rolesResp as { roles?: Role[]; items?: Role[] }).roles || (rolesResp as { items?: Role[] }).items || [];
      const permList = (permsResp as { permissions?: Permission[]; items?: Permission[] }).permissions || (permsResp as { items?: Permission[] }).items || [];

      setRoles(roleList);
      setPermissions(permList);

      // Load each role's current permissions
      const m: Record<string, Set<string>> = {};
      await Promise.all(
        roleList.map(async (role) => {
          try {
            const data = await apiFetch<{ permissions?: Permission[] }>(`/api/v1/roles/${role.id}/permissions`);
            m[role.id] = new Set((data.permissions || []).map((p) => p.id));
          } catch {
            m[role.id] = new Set();
          }
        }),
      );
      setMatrix(m);
      // deep-copy snapshot
      setOriginalMatrix(Object.fromEntries(Object.entries(m).map(([k, v]) => [k, new Set(v)])));
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  // ---- Derived data ----

  // Map permission keys (like "users.create") to permission objects
  const permByKey = useMemo(() => {
    const m = new Map<string, Permission>();
    for (const p of permissions) m.set(p.key, p);
    return m;
  }, [permissions]);

  // Build inheritance chain: ordered list of role IDs from root to leaf
  // We compute the topological order so parent comes before child
  const inheritanceChain = useMemo(() => {
    const roleMap = new Map(roles.map((r) => [r.id, r]));
    const visited = new Set<string>();
    const chain: string[] = [];

    const visit = (id: string) => {
      if (visited.has(id)) return;
      visited.add(id);
      const role = roleMap.get(id);
      if (role?.parent_role_id) visit(role.parent_role_id);
      chain.push(id);
    };
    roles.forEach((r) => visit(r.id));
    return chain;
  }, [roles]);

  // Roles sorted by inheritance order for display
  const orderedRoles = useMemo(() => {
    const map = new Map(roles.map((r) => [r.id, r]));
    return inheritanceChain.map((id) => map.get(id)).filter(Boolean) as Role[];
  }, [roles, inheritanceChain]);

  // For each role, compute inherited permissions from parent chain
  // A permission is "inherited" if it belongs to any ancestor role and the role itself doesn't directly have it
  const inheritedPerms = useMemo(() => {
    const result: Record<string, Set<string>> = {};
    const roleMap = new Map(roles.map((r) => [r.id, r]));

    for (const role of roles) {
      const inherited = new Set<string>();
      let parentId = role.parent_role_id;
      while (parentId) {
        const parentPerms = matrix[parentId];
        if (parentPerms) {
          parentPerms.forEach((p) => {
            // Only mark as inherited if the role itself doesn't directly have it
            const rolePerms = matrix[role.id];
            if (!rolePerms?.has(p)) {
              inherited.add(p);
            }
          });
        }
        const parent = roleMap.get(parentId);
        parentId = parent?.parent_role_id;
      }
      result[role.id] = inherited;
    }
    return result;
  }, [roles, matrix]);

  // ---- Cell toggle ----
  const toggleCell = (roleId: string, permId: string, isInherited: boolean) => {
    if (isInherited) return; // can't toggle inherited
    setMatrix((prev) => {
      const next = { ...prev };
      const current = new Set(next[roleId] || []);
      if (current.has(permId)) {
        current.delete(permId);
      } else {
        current.add(permId);
      }
      next[roleId] = current;
      return next;
    });
  };

  // ---- Dirty check ----
  const isDirty = useMemo(() => {
    for (const roleId of Object.keys(matrix)) {
      const orig = originalMatrix[roleId] || new Set();
      const curr = matrix[roleId] || new Set();
      if (orig.size !== curr.size) return true;
      for (const p of curr) {
        if (!orig.has(p)) return true;
      }
      for (const p of orig) {
        if (!curr.has(p)) return true;
      }
    }
    return false;
  }, [matrix, originalMatrix]);

  // ---- Save ----
  const handleSave = async () => {
    setSaving(true);
    setError(null);
    try {
      // For each role that changed, PUT the full permission set
      const changedRoles = orderedRoles.filter((role) => {
        const orig = originalMatrix[role.id] || new Set();
        const curr = matrix[role.id] || new Set();
        if (orig.size !== curr.size) return true;
        for (const p of curr) if (!orig.has(p)) return true;
        for (const p of orig) if (!curr.has(p)) return true;
        return false;
      });

      await Promise.all(
        changedRoles.map(async (role) => {
          const permIds = [...(matrix[role.id] || [])];
          await apiFetch(`/api/v1/roles/${role.id}/permissions`, {
            method: "PUT",
            body: JSON.stringify({ permission_ids: permIds }),
          });
        }),
      );

      setOriginalMatrix(Object.fromEntries(Object.entries(matrix).map(([k, v]) => [k, new Set(v)])));
      setMsg(`Saved permissions for ${changedRoles.length} role(s)`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to save");
    } finally {
      setSaving(false);
    }
  };

  // ---- Bulk apply ----
  const handleBulkApply = () => {
    if (!bulkRole || !bulkGroup) return;
    const groupDef = PERMISSION_GROUPS.find((g) => g.label === bulkGroup);
    if (!groupDef) return;

    const permIds = groupDef.permissions
      .map((key) => permByKey.get(key)?.id)
      .filter(Boolean) as string[];

    setMatrix((prev) => {
      const next = { ...prev };
      const current = new Set(next[bulkRole] || []);
      permIds.forEach((id) => current.add(id));
      next[bulkRole] = current;
      return next;
    });
    setMsg(`Applied all ${bulkGroup} permissions to ${roles.find((r) => r.id === bulkRole)?.name || bulkRole}`);
  };

  // Auto-dismiss messages
  useEffect(() => {
    if (msg) {
      const t = setTimeout(() => setMsg(null), 3000);
      return () => clearTimeout(t);
    }
  }, [msg]);

  // ---- Render ----
  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <p className="text-gray-500">Loading permissions matrix...</p>
      </div>
    );
  }

  return (
    <div>
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold dark:text-gray-100">Role Permissions Matrix</h1>
          <p className="mt-1 text-sm text-gray-500">
            Toggle permissions per role. Inherited permissions are locked.
          </p>
        </div>
        <button
          onClick={handleSave}
          disabled={!isDirty || saving}
          className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
        >
          <Save className="h-4 w-4" />
          {saving ? "Saving..." : "Save Changes"}
        </button>
      </div>

      {/* Inheritance chain indicator */}
      <div className="mb-4 flex items-center gap-2 rounded-lg border border-blue-200 bg-blue-50 p-3">
        <GitBranch className="h-4 w-4 text-blue-500" />
        <span className="text-sm font-medium text-blue-700">Inheritance:</span>
        <div className="flex items-center gap-1 text-sm text-blue-600">
          {orderedRoles.map((role, idx) => (
            <span key={role.id} className="flex items-center gap-1">
              <span className="font-medium">{role.name || role.key}</span>
              {idx < orderedRoles.length - 1 && <ArrowRight className="h-3 w-3 text-blue-400" />}
            </span>
          ))}
        </div>
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

      {/* Bulk Assign Bar */}
      <div className="mb-4 flex flex-wrap items-center gap-3 rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-800">
        <span className="flex items-center gap-1.5 text-sm font-semibold">
          <Layers className="h-4 w-4 text-brand-600" />
          Bulk Assign:
        </span>
        <select
          value={bulkRole}
          onChange={(e) => setBulkRole(e.target.value)}
          className="rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 focus:border-brand-500 focus:outline-none"
        >
          <option value="">-- Select role --</option>
          {orderedRoles.map((r) => (
            <option key={r.id} value={r.id}>
              {r.name || r.key}
              {r.system_role ? " (system)" : ""}
            </option>
          ))}
        </select>
        <select
          value={bulkGroup}
          onChange={(e) => setBulkGroup(e.target.value)}
          className="rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 focus:border-brand-500 focus:outline-none"
        >
          <option value="">-- Select group --</option>
          {PERMISSION_GROUPS.map((g) => (
            <option key={g.label} value={g.label}>
              {g.label} ({g.permissions.length})
            </option>
          ))}
        </select>
        <button
          onClick={handleBulkApply}
          disabled={!bulkRole || !bulkGroup}
          className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
        >
          Apply to All
        </button>
      </div>

      {/* Matrix Table */}
      {roles.length === 0 ? (
        <div className="rounded-xl border border-gray-200 bg-white p-12 text-center shadow-sm">
          <Shield className="mx-auto mb-4 h-12 w-12 text-gray-300" />
          <p className="text-gray-500">No roles found</p>
        </div>
      ) : (
        <div className="overflow-auto rounded-xl border border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-800" style={{ maxHeight: "70vh" }}>
          <table className="border-collapse text-sm">
            {/* Group header row */}
            <thead className="sticky top-0 z-20">
              <tr>
                <th className="sticky left-0 z-30 border-b border-r border-gray-200 bg-gray-50 px-4 py-2 text-left dark:border-gray-700 dark:bg-gray-900">
                  <span className="text-xs font-bold uppercase tracking-wide text-gray-500">Role</span>
                </th>
                {PERMISSION_GROUPS.map((group) => (
                  <th
                    key={group.label}
                    className="border-b border-r border-gray-200 bg-gray-50 px-2 py-2 text-center dark:border-gray-700 dark:bg-gray-900"
                    colSpan={group.permissions.length}
                  >
                    <span className="text-xs font-bold uppercase tracking-wide text-gray-600 dark:text-gray-400">
                      {group.label}
                    </span>
                  </th>
                ))}
              </tr>
              {/* Permission key header row */}
              <tr>
                <th className="sticky left-0 z-30 border-b border-r border-gray-200 bg-gray-50 px-4 py-2 text-left dark:border-gray-700 dark:bg-gray-900">
                  &nbsp;
                </th>
                {PERMISSION_GROUPS.map((group) =>
                  group.permissions.map((permKey) => (
                    <th
                      key={permKey}
                      className="sticky top-[33px] border-b border-r border-gray-100 px-1 py-2 text-center font-mono text-[10px] font-normal text-gray-400 dark:border-gray-700"
                      title={permKey}
                    >
                      <div className="flex flex-col items-center gap-0.5">
                        <span>{permKey.split(".")[1] || permKey}</span>
                        <span className="text-[8px] text-gray-300">{permKey.split(".")[0]}</span>
                      </div>
                    </th>
                  )),
                )}
              </tr>
            </thead>
            <tbody>
              {orderedRoles.map((role, rowIdx) => {
                const rolePerms = matrix[role.id] || new Set();
                const roleInherited = inheritedPerms[role.id] || new Set();
                const isLastRow = rowIdx === orderedRoles.length - 1;

                return (
                  <tr key={role.id} className="hover:bg-gray-50 dark:hover:bg-gray-700/50">
                    {/* Role name (sticky) */}
                    <td className="sticky left-0 z-10 border-b border-r border-gray-100 bg-white px-4 py-3 dark:border-gray-700 dark:bg-gray-800">
                      <div className="flex items-center gap-2">
                        {role.parent_role_id && (
                          <ChevronRight className="h-3 w-3 text-gray-300" />
                        )}
                        <Shield className={`h-4 w-4 ${role.system_role ? "text-gray-400" : "text-brand-500"}`} />
                        <div>
                          <span className="font-medium text-gray-800 dark:text-gray-200">
                            {role.name || role.key}
                          </span>
                          {role.system_role && (
                            <span className="ml-1 text-xs text-gray-400">(system)</span>
                          )}
                        </div>
                      </div>
                    </td>
                    {/* Permission cells */}
                    {PERMISSION_GROUPS.map((group) =>
                      group.permissions.map((permKey) => {
                        const permObj = permByKey.get(permKey);
                        const permId = permObj?.id || "";
                        const isAllowed = rolePerms.has(permId);
                        const isInherited = roleInherited.has(permId);

                        return (
                          <td
                            key={`${group.label}-${permKey}`}
                            className={`border-b border-r border-gray-100 px-1 py-3 text-center dark:border-gray-700 ${
                              !isLastRow ? "" : ""
                            }`}
                          >
                            <button
                              onClick={() => toggleCell(role.id, permId, isInherited)}
                              disabled={isInherited}
                              className={`flex h-6 w-6 items-center justify-center rounded transition-colors ${
                                isInherited
                                  ? "cursor-not-allowed"
                                  : "cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-600"
                              }`}
                              title={
                                isInherited
                                  ? `Inherited from parent (${permKey})`
                                  : isAllowed
                                    ? `Revoke ${permKey}`
                                    : `Grant ${permKey}`
                              }
                            >
                              {isAllowed ? (
                                <CheckCircle2 className="h-5 w-5 text-green-500" />
                              ) : isInherited ? (
                                <Lock className="h-3.5 w-3.5 text-gray-300" />
                              ) : (
                                <XCircle className="h-5 w-5 text-gray-200 hover:text-red-400" />
                              )}
                            </button>
                          </td>
                        );
                      }),
                    )}
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {/* Legend */}
      <div className="mt-4 flex items-center gap-6 text-xs text-gray-500">
        <span className="flex items-center gap-1.5">
          <CheckCircle2 className="h-4 w-4 text-green-500" />
          Allowed (click to revoke)
        </span>
        <span className="flex items-center gap-1.5">
          <XCircle className="h-4 w-4 text-gray-200" />
          Denied (click to grant)
        </span>
        <span className="flex items-center gap-1.5">
          <Lock className="h-3.5 w-3.5 text-gray-300" />
          Inherited from parent (not directly editable)
        </span>
      </div>
    </div>
  );
}

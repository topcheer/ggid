"use client";

import { useEffect, useState, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  Building2,
  Users,
  Settings,
  X,
  Plus,
  Trash2,
  ChevronDown,
  ChevronRight,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

// ===== Types =====

interface Organization {
  id: string;
  name: string;
  path: string;
  parent_id?: string;
  description?: string;
}

interface Member {
  id: string;
  user_id: string;
  org_id: string;
  status: string;
  title: string;
}

interface Role {
  id: string;
  key: string;
  name: string;
}

// ===== Tree Node =====

interface OrgNode extends Organization {
  children: OrgNode[];
}

function buildTree(orgs: Organization[]): OrgNode[] {
  const t = useTranslations();

  const map = new Map<string, OrgNode>();
  const roots: OrgNode[] = [];

  for (const org of orgs) {
    map.set(org.id, { ...org, children: [] });
  }
  for (const org of orgs) {
    const node = map.get(org.id)!;
    if (org.parent_id && map.has(org.parent_id)) {
      map.get(org.parent_id)!.children.push(node);
    } else {
      roots.push(node);
    }
  }
  return roots;
}

// ===== Main Component =====

export default function OrgChartPage() {
  const { apiFetch } = useApi();
  const [orgs, setOrgs] = useState<Organization[]>([]);
  const [memberCounts, setMemberCounts] = useState<Record<string, number>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [msg, setMsg] = useState<string | null>(null);

  // Expand/collapse state — default root expanded
  const [expanded, setExpanded] = useState<Set<string>>(new Set());

  // Drag-and-drop state
  const [draggingId, setDraggingId] = useState<string | null>(null);
  const [dragOverId, setDragOverId] = useState<string | null>(null);

  // Settings drawer state
  const [drawerOrg, setDrawerOrg] = useState<Organization | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(false);

  // ---- Data loading ----
  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const data = await apiFetch<{ organizations?: Organization[] }>("/api/v1/orgs");
      const list = data.organizations || [];
      setOrgs(list);

      // Fetch member counts in parallel
      const counts: Record<string, number> = {};
      await Promise.all(
        list.map(async (org) => {
          try {
            const memData = await apiFetch<{ members?: Member[] }>(`/api/v1/orgs/${org.id}/members`);
            counts[org.id] = memData.members?.length || 0;
          } catch {
            counts[org.id] = 0;
          }
        }),
      );
      setMemberCounts(counts);

      // Default: expand root orgs
      const roots = list.filter((o) => !o.parent_id);
      setExpanded(new Set(roots.map((r) => r.id)));

      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load organizations");
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  // Auto-dismiss messages
  useEffect(() => {
    if (msg) {
      const t = setTimeout(() => setMsg(null), 3000);
      return () => clearTimeout(t);
    }
  }, [msg]);

  const tree = buildTree(orgs);
  const orgMap = new Map(orgs.map((o) => [o.id, o]));

  // ---- Expand/collapse toggle ----
  const toggleExpand = (id: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  // ---- Drag and Drop ----
  const handleDragStart = (e: React.DragEvent, orgId: string) => {
    setDraggingId(orgId);
    e.dataTransfer.effectAllowed = "move";
    e.dataTransfer.setData("text/plain", orgId);
  };

  const handleDragOver = (e: React.DragEvent, orgId: string) => {
    e.preventDefault();
    e.dataTransfer.dropEffect = "move";
    if (draggingId && orgId !== draggingId) {
      setDragOverId(orgId);
    }
  };

  const handleDragLeave = () => {
    setDragOverId(null);
  };

  const handleDrop = async (e: React.DragEvent, targetOrgId: string) => {
    e.preventDefault();
    setDragOverId(null);
    const sourceOrgId = draggingId;
    setDraggingId(null);

    if (!sourceOrgId || sourceOrgId === targetOrgId) return;

    // Prevent dropping onto own descendant
    const sourceNode = tree.find((n) => n.id === sourceOrgId);
    const isDescendant = (node: OrgNode | undefined, targetId: string): boolean => {
      if (!node) return false;
      if (node.id === targetId) return true;
      return node.children.some((c) => isDescendant(c, targetId));
    };
    if (isDescendant(sourceNode, targetOrgId)) {
      setError("Cannot reassign an org to its own descendant");
      return;
    }

    // Optimistically update parent_id in local state
    setOrgs((prev) =>
      prev.map((o) => (o.id === sourceOrgId ? { ...o, parent_id: targetOrgId } : o)),
    );

    try {
      await apiFetch(`/api/v1/orgs/${sourceOrgId}`, {
        method: "PUT",
        body: JSON.stringify({ parent_id: targetOrgId }),
      });
      const sourceOrg = orgMap.get(sourceOrgId);
      const targetOrg = orgMap.get(targetOrgId);
      setMsg(`Moved "${sourceOrg?.name}" under "${targetOrg?.name}"`);
      // Reload to reflect server-side path changes
      loadData();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to reassign organization");
      loadData(); // revert
    }
  };

  const handleDragEnd = () => {
    setDraggingId(null);
    setDragOverId(null);
  };

  // ---- Open settings drawer ----
  const openSettings = (org: Organization) => {
    setDrawerOrg(org);
    setDrawerOpen(true);
  };

  const closeDrawer = () => {
    setDrawerOpen(false);
    setDrawerOrg(null);
  };

  // ---- Render tree node recursively ----
  const renderNode = (node: OrgNode, depth: number): React.ReactElement => {
    const isExpanded = expanded.has(node.id);
    const isDragOver = dragOverId === node.id;
    const isDragging = draggingId === node.id;
    const count = memberCounts[node.id] || 0;

    return (
      <div key={node.id} className="relative">
        <div
          draggable
          onDragStart={(e) => handleDragStart(e, node.id)}
          onDragOver={(e) => handleDragOver(e, node.id)}
          onDragLeave={handleDragLeave}
          onDrop={(e) => handleDrop(e, node.id)}
          onDragEnd={handleDragEnd}
          className={`flex items-center gap-3 rounded-xl border p-3 shadow-sm transition-all ${
            isDragOver
              ? "border-blue-400 bg-blue-50 ring-2 ring-blue-300 dark:border-blue-500 dark:bg-blue-900/20"
              : "border-gray-200 bg-white hover:shadow-md dark:border-gray-700 dark:bg-gray-800"
          } ${isDragging ? "opacity-40" : ""} ${depth === 0 ? "ring-1 ring-brand-200" : ""}`}
          style={{ marginLeft: `${depth * 32}px` }}
        >
          {/* Connecting line for children */}
          {depth > 0 && (
            <div
              className="absolute border-l-2 border-gray-200 dark:border-gray-600"
              style={{
                left: `${(depth - 1) * 32 + 16}px`,
                top: 0,
                height: "50%",
              }}
            />
          )}
          {depth > 0 && (
            <div
              className="absolute border-t-2 border-gray-200 dark:border-gray-600"
              style={{
                left: `${(depth - 1) * 32 + 16}px`,
                top: "50%",
                width: "16px",
              }}
            />
          )}

          {/* Expand/collapse toggle */}
          <button
            onClick={() => toggleExpand(node.id)}
            className={`flex h-6 w-6 items-center justify-center rounded text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 ${
              node.children.length > 0 ? "" : "invisible"
            }`}
          >
            {isExpanded ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
          </button>

          {/* Org icon */}
          <div className={`flex h-9 w-9 items-center justify-center rounded-lg ${depth === 0 ? "bg-brand-100" : "bg-gray-100 dark:bg-gray-700"}`}>
            <Building2 className={`h-5 w-5 ${depth === 0 ? "text-brand-600" : "text-gray-500"}`} />
          </div>

          {/* Name + description */}
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <span className={`truncate ${depth === 0 ? "text-base font-bold" : "text-sm font-semibold"}`}>
                {node.name}
              </span>
              {node.path && (
                <span className="font-mono text-xs text-gray-400">{node.path}</span>
              )}
            </div>
            {node.description && (
              <p className="truncate text-xs text-gray-500">{node.description}</p>
            )}
          </div>

          {/* Member count badge */}
          <span className="flex items-center gap-1 rounded-full bg-blue-50 px-2.5 py-1 text-xs font-medium text-blue-600 dark:bg-blue-900/30 dark:text-blue-400">
            <Users className="h-3 w-3" />
            {count}
          </span>

          {/* Child count */}
          {node.children.length > 0 && (
            <span className="text-xs text-gray-400">
              {node.children.length} sub-org{node.children.length !== 1 ? "s" : ""}
            </span>
          )}

          {/* Settings gear */}
          <button
            onClick={() => openSettings(node)}
            className="flex h-8 w-8 items-center justify-center rounded-lg text-gray-400 hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-gray-700"
            title="Organization settings"
          >
            <Settings className="h-4 w-4" />
          </button>
        </div>

        {/* Render children */}
        {isExpanded && node.children.length > 0 && (
          <div className="mt-1 space-y-1">
            {node.children.map((child) => renderNode(child, depth + 1))}
          </div>
        )}
      </div>
    );
  };

  // ---- Render ----
  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <p className="text-gray-500">Loading organization chart...</p>
      </div>
    );
  }

  return (
    <div>
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold dark:text-gray-100">Organization Chart</h1>
          <p className="mt-1 text-sm text-gray-500">
            Drag org cards to reassign. Click the gear to edit settings.
          </p>
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

      {/* Tree */}
      {tree.length === 0 ? (
        <div className="rounded-xl border border-gray-200 bg-white p-12 text-center shadow-sm">
          <Building2 className="mx-auto mb-4 h-12 w-12 text-gray-300" />
          <p className="text-gray-500">No organizations found</p>
        </div>
      ) : (
        <div className="space-y-1">
          {tree.map((root) => renderNode(root, 0))}
        </div>
      )}

      {/* Settings Drawer */}
      {drawerOpen && drawerOrg && (
        <SettingsDrawer
          org={drawerOrg}
          memberCount={memberCounts[drawerOrg.id] || 0}
          apiFetch={apiFetch}
          onClose={closeDrawer}
          onSaved={() => {
            closeDrawer();
            loadData();
          }}
        />
      )}
    </div>
  );
}

// ===== Settings Drawer Component =====

function SettingsDrawer({
  org,
  memberCount,
  apiFetch,
  onClose,
  onSaved,
}: {
  org: Organization;
  memberCount: number;
  apiFetch: <T>(path: string, options?: RequestInit) => Promise<T>;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [name, setName] = useState(org.name);
  const [description, setDescription] = useState(org.description || "");
  const [defaultRole, setDefaultRole] = useState("");
  const [roles, setRoles] = useState<Role[]>([]);
  const [members, setMembers] = useState<Member[]>([]);
  const [loadingMembers, setLoadingMembers] = useState(true);
  const [saving, setSaving] = useState(false);
  const [addMemberId, setAddMemberId] = useState("");
  const [error, setError] = useState<string | null>(null);

  // Load roles + members
  useEffect(() => {
    const load = async () => {
      try {
        const [rolesResp, membersResp] = await Promise.all([
          apiFetch<{ roles?: Role[] }>(`/api/v1/roles`).catch(() => ({ roles: [] as Role[] })),
          apiFetch<{ members?: Member[] }>(`/api/v1/orgs/${org.id}/members`).catch(() => ({ members: [] as Member[] })),
        ]);
        setRoles(rolesResp.roles || []);
        setMembers(membersResp.members || []);
      } catch {
        // ignore
      } finally {
        setLoadingMembers(false);
      }
    };
    load();
  }, [org.id, apiFetch]);

  const handleSave = async () => {
    setSaving(true);
    setError(null);
    try {
      const body: Record<string, string> = { name };
      if (description) body.description = description;
      await apiFetch(`/api/v1/orgs/${org.id}`, {
        method: "PUT",
        body: JSON.stringify(body),
      });
      onSaved();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to save");
    } finally {
      setSaving(false);
    }
  };

  const handleAddMember = async () => {
    if (!addMemberId) return;
    try {
      await apiFetch(`/api/v1/orgs/${org.id}/members`, {
        method: "POST",
        body: JSON.stringify({ user_id: addMemberId, org_id: org.id, status: "active" }),
      });
      setAddMemberId("");
      // Reload members
      const data = await apiFetch<{ members?: Member[] }>(`/api/v1/orgs/${org.id}/members`);
      setMembers(data.members || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to add member");
    }
  };

  const handleRemoveMember = async (memberId: string) => {
    try {
      await apiFetch(`/api/v1/orgs/${org.id}/members/${memberId}`, {
        method: "DELETE",
      });
      setMembers((prev) => prev.filter((m) => m.id !== memberId));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to remove member");
    }
  };

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 z-40 bg-black/30"
        onClick={onClose}
      />

      {/* Drawer */}
      <div className="fixed right-0 top-0 z-50 h-full w-96 overflow-y-auto bg-white shadow-2xl dark:bg-gray-800">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-gray-200 p-4 dark:border-gray-700">
          <h2 className="flex items-center gap-2 text-lg font-semibold">
            <Building2 className="h-5 w-5 text-brand-600" />
            Org Settings
          </h2>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600" aria-label="Close">
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="space-y-5 p-4">
          {error && (
            <div className="rounded-lg border border-red-200 bg-red-50 p-2 text-sm text-red-700">
              {error}
            </div>
          )}

          {/* Org name */}
          <div>
            <label className="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">
              Organization Name
            </label>
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 focus:border-brand-500 focus:outline-none"
            />
          </div>

          {/* Description */}
          <div>
            <label className="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">
              Description
            </label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={3}
              className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 focus:border-brand-500 focus:outline-none"
            />
          </div>

          {/* Default role */}
          <div>
            <label className="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">
              Default Role Assignment
            </label>
            <select
              value={defaultRole}
              onChange={(e) => setDefaultRole(e.target.value)}
              className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 focus:border-brand-500 focus:outline-none"
            >
              <option value="">-- None --</option>
              {roles.map((r) => (
                <option key={r.id} value={r.id}>
                  {r.name || r.key}
                </option>
              ))}
            </select>
          </div>

          {/* Save button */}
          <button
            onClick={handleSave}
            disabled={saving || !name}
            className="w-full rounded-lg bg-brand-600 px-4 py-2.5 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
          >
            {saving ? "Saving..." : "Save Settings"}
          </button>

          {/* Divider */}
          <div className="border-t border-gray-200 pt-4 dark:border-gray-700" />

          {/* Members section */}
          <div>
            <div className="mb-2 flex items-center justify-between">
              <h3 className="flex items-center gap-1.5 text-sm font-semibold">
                <Users className="h-4 w-4 text-blue-500" />
                Members ({members.length})
              </h3>
            </div>

            {/* Add member */}
            <div className="mb-3 flex gap-2">
              <input
                value={addMemberId}
                onChange={(e) => setAddMemberId(e.target.value)}
                placeholder="User ID"
                className="flex-1 rounded-lg border border-gray-300 px-3 py-1.5 text-xs dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 focus:border-brand-500 focus:outline-none"
              />
              <button
                onClick={handleAddMember}
                disabled={!addMemberId}
                className="flex items-center gap-1 rounded-lg bg-brand-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-brand-700 disabled:opacity-50"
              >
                <Plus className="h-3.5 w-3.5" /> Add
              </button>
            </div>

            {/* Member list */}
            {loadingMembers ? (
              <p className="text-xs text-gray-400">Loading members...</p>
            ) : members.length === 0 ? (
              <p className="text-xs text-gray-400">No members yet</p>
            ) : (
              <div className="space-y-1">
                {members.map((m) => (
                  <div
                    key={m.id}
                    className="flex items-center justify-between rounded-lg border border-gray-100 px-3 py-2 dark:border-gray-700"
                  >
                    <div className="min-w-0 flex-1">
                      <p className="truncate font-mono text-xs text-gray-600 dark:text-gray-400">
                        {m.user_id.slice(0, 16)}...
                      </p>
                      {m.title && <p className="text-xs text-gray-400">{m.title}</p>}
                    </div>
                    <span className={`mr-2 rounded-full px-2 py-0.5 text-xs ${
                      m.status === "active" ? "bg-green-100 text-green-700" : "bg-gray-100 text-gray-500"
                    }`}>
                      {m.status}
                    </span>
                    <button
                      onClick={() => handleRemoveMember(m.id)}
                      className="text-gray-400 hover:text-red-500"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>
    </>
  );
}

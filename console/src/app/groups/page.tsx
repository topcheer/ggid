"use client";

import { useState, useCallback, useEffect } from "react";
import { useApi } from "@/lib/api";
import {
  Plus,
  Trash2,
  ChevronDown,
  ChevronRight,
  Users as UsersIcon,
  Shield,
  UserPlus,
  UserMinus,
  X,
} from "lucide-react";

interface Group {
  id: string;
  name: string;
  description?: string;
  parent_id?: string;
  member_count?: number;
  roles?: { id: string; key: string; name: string }[];
  members?: GroupMember[];
  created_at: string;
}

interface GroupMember {
  id: string;
  username: string;
  email: string;
}

interface Role {
  id: string;
  key: string;
  name: string;
}

export default function GroupsPage() {
  const { apiFetch } = useApi();
  const [groups, setGroups] = useState<Group[]>([]);
  const [roles, setRoles] = useState<Role[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [msg, setMsg] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [addMemberInput, setAddMemberInput] = useState("");

  const [form, setForm] = useState({
    name: "",
    description: "",
    parent_id: "",
  });

  const loadGroups = useCallback(async () => {
    setLoading(true);
    try {
      const data = await apiFetch<{ groups?: Group[]; items?: Group[] }>(
        "/api/v1/groups",
      ).catch(() => ({ groups: [] }) as { groups?: Group[]; items?: Group[] });
      setGroups(data.groups || data.items || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load groups");
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  const loadRoles = useCallback(async () => {
    try {
      const data = await apiFetch<{ roles?: Role[] }>("/api/v1/roles").catch(() => ({ roles: [] }));
      setRoles(data.roles || []);
    } catch {
      // ignore
    }
  }, [apiFetch]);

  useEffect(() => {
    loadGroups();
    loadRoles();
  }, [loadGroups, loadRoles]);

  useEffect(() => {
    if (msg) {
      const t = setTimeout(() => setMsg(null), 3000);
      return () => clearTimeout(t);
    }
  }, [msg]);

  const handleCreate = async () => {
    try {
      await apiFetch("/api/v1/groups", {
        method: "POST",
        body: JSON.stringify({
          name: form.name,
          description: form.description,
          parent_id: form.parent_id || undefined,
        }),
      });
      setForm({ name: "", description: "", parent_id: "" });
      setShowCreate(false);
      setMsg("Group created");
      loadGroups();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create group");
    }
  };

  const handleDelete = async (groupId: string, name: string) => {
    if (!confirm(`Delete group "${name}"?`)) return;
    try {
      await apiFetch(`/api/v1/groups/${groupId}`, { method: "DELETE" });
      setMsg("Group deleted");
      loadGroups();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete");
    }
  };

  const handleAssignRole = async (groupId: string, roleId: string) => {
    if (!roleId) return;
    try {
      await apiFetch(`/api/v1/groups/${groupId}/roles`, {
        method: "POST",
        body: JSON.stringify({ role_id: roleId }),
      });
      setMsg("Role assigned");
      loadGroups();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to assign role");
    }
  };

  const handleAddMember = async (groupId: string) => {
    const username = addMemberInput.trim();
    if (!username) return;
    try {
      await apiFetch(`/api/v1/groups/${groupId}/members`, {
        method: "POST",
        body: JSON.stringify({ username }),
      });
      setAddMemberInput("");
      setMsg("Member added");
      loadGroups();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to add member");
    }
  };

  const handleRemoveMember = async (groupId: string, memberId: string) => {
    try {
      await apiFetch(`/api/v1/groups/${groupId}/members/${memberId}`, {
        method: "DELETE",
      });
      setMsg("Member removed");
      loadGroups();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to remove member");
    }
  };

  // Build hierarchy tree
  const buildHierarchy = (allGroups: Group[], parentId?: string, depth = 0): { group: Group; depth: number }[] => {
    const children = allGroups.filter((g) => (parentId ? g.parent_id === parentId : !g.parent_id));
    const result: { group: Group; depth: number }[] = [];
    for (const child of children) {
      result.push({ group: child, depth });
      result.push(...buildHierarchy(allGroups, child.id, depth + 1));
    }
    return result;
  };

  const hierarchicalGroups = buildHierarchy(groups);
  const groupNameById = (id?: string) => groups.find((g) => g.id === id)?.name || "—";

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold dark:text-gray-100">Groups / Teams</h1>
        <button
          onClick={() => { setShowCreate(!showCreate); setError(null); }}
          className="flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
        >
          <Plus className="h-4 w-4" /> New Group
        </button>
      </div>

      {msg && (
        <div className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700">{msg}</div>
      )}
      {error && (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">{error}</div>
      )}

      {/* Create form */}
      {showCreate && (
        <div className="mb-6 rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <div className="mb-4 flex items-center justify-between">
            <h3 className="text-sm font-semibold dark:text-gray-100">Create New Group</h3>
            <button onClick={() => setShowCreate(false)} className="text-gray-400 hover:text-gray-600">
              <X className="h-5 w-5" />
            </button>
          </div>
          <div className="grid gap-4 sm:grid-cols-3">
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">Name *</label>
              <input
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
                placeholder="e.g. Engineering Team"
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">Description</label>
              <input
                value={form.description}
                onChange={(e) => setForm({ ...form, description: e.target.value })}
                placeholder="Brief description"
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">Parent Group</label>
              <select
                value={form.parent_id}
                onChange={(e) => setForm({ ...form, parent_id: e.target.value })}
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
              >
                <option value="">— Root (no parent) —</option>
                {groups.map((g) => (
                  <option key={g.id} value={g.id}>{g.name}</option>
                ))}
              </select>
            </div>
          </div>
          <button
            onClick={handleCreate}
            disabled={!form.name}
            className="mt-4 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
          >
            Create Group
          </button>
        </div>
      )}

      {/* Groups table */}
      {loading ? (
        <p className="text-gray-500">Loading...</p>
      ) : hierarchicalGroups.length === 0 ? (
        <div className="rounded-xl border border-gray-200 bg-white p-12 text-center shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <UsersIcon className="mx-auto mb-4 h-12 w-12 text-gray-300" />
          <p className="text-gray-500">No groups created yet</p>
        </div>
      ) : (
        <div className="overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <table className="w-full">
            <thead className="border-b border-gray-200 bg-gray-50 dark:border-gray-700 dark:bg-gray-700/50">
              <tr>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Name</th>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Members</th>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Parent</th>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Roles</th>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Created</th>
                <th className="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
              {hierarchicalGroups.map(({ group, depth }) => (
                <>
                  <tr key={group.id} className="hover:bg-gray-50 dark:hover:bg-gray-700/50">
                    <td className="px-4 py-3">
                      <div className="flex items-center" style={{ paddingLeft: `${depth * 24}px` }}>
                        {depth > 0 && <span className="mr-1 text-gray-300">└</span>}
                        <button
                          onClick={() => setExpandedId(expandedId === group.id ? null : group.id)}
                          className="mr-1.5 text-gray-400 hover:text-gray-600"
                        >
                          {expandedId === group.id ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
                        </button>
                        <div>
                          <p className="text-sm font-medium dark:text-gray-100">{group.name}</p>
                          {group.description && (
                            <p className="text-xs text-gray-500">{group.description}</p>
                          )}
                        </div>
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <span className="inline-flex items-center gap-1 rounded-full bg-blue-50 px-2 py-0.5 text-xs font-medium text-blue-700">
                        <UsersIcon className="h-3 w-3" />
                        {group.member_count ?? group.members?.length ?? 0}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-500">{groupNameById(group.parent_id)}</td>
                    <td className="px-4 py-3">
                      <div className="flex flex-wrap gap-1">
                        {(group.roles || []).map((r) => (
                          <span key={r.id} className="rounded-full bg-purple-50 px-2 py-0.5 text-xs text-purple-700">
                            {r.name || r.key}
                          </span>
                        ))}
                        {(!group.roles || group.roles.length === 0) && (
                          <span className="text-xs text-gray-300">—</span>
                        )}
                      </div>
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-500">
                      {group.created_at ? new Date(group.created_at).toLocaleDateString() : "—"}
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex justify-end">
                        <button
                          onClick={() => handleDelete(group.id, group.name)}
                          title="Delete"
                          className="rounded p-1.5 text-gray-400 hover:bg-red-50 hover:text-red-600"
                        >
                          <Trash2 className="h-4 w-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                  {/* Expanded row */}
                  {expandedId === group.id && (
                    <tr key={`${group.id}-expand`} className="bg-gray-50 dark:bg-gray-700/30">
                      <td colSpan={6} className="px-8 py-4">
                        <div className="space-y-4">
                          {/* Role assignment */}
                          <div>
                            <h4 className="mb-2 flex items-center gap-1.5 text-xs font-semibold text-gray-600">
                              <Shield className="h-3.5 w-3.5" /> Assign Role
                            </h4>
                            <div className="flex gap-2">
                              <select
                                onChange={(e) => { handleAssignRole(group.id, e.target.value); e.target.value = ""; }}
                                defaultValue=""
                                className="rounded-lg border border-gray-300 px-3 py-1.5 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
                              >
                                <option value="" disabled>Select a role to assign...</option>
                                {roles.map((r) => (
                                  <option key={r.id} value={r.id}>{r.name || r.key}</option>
                                ))}
                              </select>
                            </div>
                          </div>

                          {/* Member management */}
                          <div>
                            <h4 className="mb-2 flex items-center gap-1.5 text-xs font-semibold text-gray-600">
                              <UsersIcon className="h-3.5 w-3.5" /> Members ({group.members?.length ?? 0})
                            </h4>
                            <div className="mb-2 flex gap-2">
                              <input
                                value={addMemberInput}
                                onChange={(e) => setAddMemberInput(e.target.value)}
                                onKeyDown={(e) => { if (e.key === "Enter") handleAddMember(group.id); }}
                                placeholder="Enter username to add..."
                                className="rounded-lg border border-gray-300 px-3 py-1.5 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
                              />
                              <button
                                onClick={() => handleAddMember(group.id)}
                                disabled={!addMemberInput.trim()}
                                className="flex items-center gap-1 rounded-lg bg-brand-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-brand-700 disabled:opacity-50"
                              >
                                <UserPlus className="h-3 w-3" /> Add
                              </button>
                            </div>
                            <div className="space-y-1">
                              {(group.members || []).map((m) => (
                                <div key={m.id} className="flex items-center justify-between rounded-lg bg-white px-3 py-1.5 dark:bg-gray-800">
                                  <div>
                                    <span className="text-sm font-medium dark:text-gray-200">{m.username}</span>
                                    <span className="ml-2 text-xs text-gray-500">{m.email}</span>
                                  </div>
                                  <button
                                    onClick={() => handleRemoveMember(group.id, m.id)}
                                    className="rounded p-1 text-gray-400 hover:bg-red-50 hover:text-red-600"
                                  >
                                    <UserMinus className="h-3.5 w-3.5" />
                                  </button>
                                </div>
                              ))}
                              {(!group.members || group.members.length === 0) && (
                                <p className="text-xs text-gray-400">No members in this group</p>
                              )}
                            </div>
                          </div>
                        </div>
                      </td>
                    </tr>
                  )}
                </>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

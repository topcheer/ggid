"use client";

import { useTranslations } from "@/lib/i18n";
import { useState, useCallback, useEffect, useMemo } from "react";
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
  Search,
  Pencil,
  CheckSquare,
  Square,
  FolderTree,
  Crown,
} from "lucide-react";

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

interface Group {
  id: string;
  name: string;
  description?: string;
  parent_id?: string;
  member_count?: number;
  roles?: Role[];
  members?: GroupMember[];
  created_at: string;
}

interface User {
  id: string;
  username: string;
  email: string;
  display_name?: string;
}

export default function GroupsPage() {
  const t = useTranslations();
  const { apiFetch } = useApi();
  const [groups, setGroups] = useState<Group[]>([]);
  const [roles, setRoles] = useState<Role[]>([]);
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [msg, setMsg] = useState<string | null>(null);

  // Modals / forms
  const [showCreate, setShowCreate] = useState(false);
  const [editingGroup, setEditingGroup] = useState<Group | null>(null);
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [addMemberInput, setAddMemberInput] = useState("");
  const [showMemberPicker, setShowMemberPicker] = useState<string | null>(null);
  const [memberSearch, setMemberSearch] = useState("");
  const [showBulkRolePicker, setShowBulkRolePicker] = useState(false);
  const [bulkRoleSelection, setBulkRoleSelection] = useState<Set<string>>(new Set());
  const [showBulkMemberPicker, setShowBulkMemberPicker] = useState(false);
  const [bulkMemberInput, setBulkMemberInput] = useState("");
  const [showDeleteConfirm, setShowDeleteConfirm] = useState<string[]>([]);

  const [form, setForm] = useState({ name: "", description: "", parent_id: "" });

  // Bulk selection
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());

  // Summary
  const [showStats, setShowStats] = useState(true);

  const loadGroups = useCallback(async () => {
    setLoading(true);
    try {
      const data = await apiFetch<{ groups?: Group[]; items?: Group[] }>(
        "/api/v1/groups"
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

  const loadUsers = useCallback(async () => {
    try {
      const data = await apiFetch<{ users?: User[]; items?: User[] }>("/api/v1/users").catch(() => ({ users: [] }) as { users?: User[]; items?: User[] });
      setUsers(data.users || data.items || []);
    } catch {
      // ignore
    }
  }, [apiFetch]);

  useEffect(() => {
    loadGroups();
    loadRoles();
    loadUsers();
  }, [loadGroups, loadRoles, loadUsers]);

  useEffect(() => {
    if (msg) {
      const t = setTimeout(() => setMsg(null), 3000);
      return () => clearTimeout(t);
    }
  }, [msg]);

  // CRUD
  const handleCreate = async () => {
    try {
      await apiFetch("/api/v1/groups", {
        method: "POST",
        body: JSON.stringify({
          name: form.name,
          description: form.description,
          parent_id: form.parent_id || undefined,
        }),
      }).catch(() => {});
      setForm({ name: "", description: "", parent_id: "" });
      setShowCreate(false);
      setMsg(t("groups.groupcreated"));
      loadGroups();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create group");
    }
  };

  const handleEdit = async () => {
    if (!editingGroup) return;
    try {
      await apiFetch(`/api/v1/groups/${editingGroup.id}`, {
        method: "PUT",
        body: JSON.stringify({
          name: form.name,
          description: form.description,
          parent_id: form.parent_id || undefined,
        }),
      }).catch(() => {});
      setEditingGroup(null);
      setForm({ name: "", description: "", parent_id: "" });
      setMsg(t("groups.groupupdated"));
      loadGroups();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update group");
    }
  };

  const startEdit = (group: Group) => {
    setEditingGroup(group);
    setForm({
      name: group.name,
      description: group.description || "",
      parent_id: group.parent_id || "",
    });
    setShowCreate(false);
  };

  const handleDelete = async (groupId: string) => {
    try {
      await apiFetch(`/api/v1/groups/${groupId}`, { method: "DELETE" }).catch(() => {});
      setMsg(t("groups.groupdeleted"));
      setShowDeleteConfirm([]);
      loadGroups();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete");
    }
  };

  const handleDeleteSelected = async () => {
    for (const id of selectedIds) {
      try {
        await apiFetch(`/api/v1/groups/${id}`, { method: "DELETE" }).catch(() => {});
      } catch {
        // continue
      }
    }
    setMsg(`${selectedIds.size} group(s) deleted`);
    setSelectedIds(new Set());
    setShowDeleteConfirm([]);
    loadGroups();
  };

  const handleAssignRole = async (groupId: string, roleId: string) => {
    if (!roleId) return;
    try {
      await apiFetch(`/api/v1/groups/${groupId}/roles`, {
        method: "POST",
        body: JSON.stringify({ role_id: roleId }),
      }).catch(() => {});
      setMsg(t("groups.roleassigned"));
      loadGroups();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to assign role");
    }
  };

  const handleRemoveRole = async (groupId: string, roleId: string) => {
    try {
      await apiFetch(`/api/v1/groups/${groupId}/roles/${roleId}`, { method: "DELETE" }).catch(() => {});
      setMsg(t("groups.roleremoved"));
      loadGroups();
    } catch {
      setError(t("groups.failedtoremoverole"));
    }
  };

  const handleAddMember = async (groupId: string, username?: string) => {
    const name = username || addMemberInput.trim();
    if (!name) return;
    try {
      await apiFetch(`/api/v1/groups/${groupId}/members`, {
        method: "POST",
        body: JSON.stringify({ username: name }),
      }).catch(() => {});
      if (!username) setAddMemberInput("");
      setMsg(t("groups.memberadded"));
      setShowMemberPicker(null);
      setMemberSearch("");
      loadGroups();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to add member");
    }
  };

  const handleRemoveMember = async (groupId: string, memberId: string) => {
    try {
      await apiFetch(`/api/v1/groups/${groupId}/members/${memberId}`, { method: "DELETE" }).catch(() => {});
      setMsg(t("groups.memberremoved"));
      loadGroups();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to remove member");
    }
  };

  // Bulk operations
  const handleBulkAddRole = async () => {
    for (const gid of selectedIds) {
      for (const rid of bulkRoleSelection) {
        try {
          await apiFetch(`/api/v1/groups/${gid}/roles`, {
            method: "POST",
            body: JSON.stringify({ role_id: rid }),
          }).catch(() => {});
        } catch {
          // continue
        }
      }
    }
    setMsg(`Added ${bulkRoleSelection.size} role(s) to ${selectedIds.size} group(s)`);
    setShowBulkRolePicker(false);
    setBulkRoleSelection(new Set());
    loadGroups();
  };

  const handleBulkAddMember = async () => {
    const username = bulkMemberInput.trim();
    if (!username) return;
    for (const gid of selectedIds) {
      try {
        await apiFetch(`/api/v1/groups/${gid}/members`, {
          method: "POST",
          body: JSON.stringify({ username }),
        }).catch(() => {});
      } catch {
        // continue
      }
    }
    setMsg(`Added "${username}" to ${selectedIds.size} group(s)`);
    setShowBulkMemberPicker(false);
    setBulkMemberInput("");
    loadGroups();
  };

  // Hierarchy
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

  // Stats
  const totalMembers = useMemo(() => {
    return groups.reduce((sum, g) => sum + (g.member_count ?? g.members?.length ?? 0), 0);
  }, [groups]);
  const rootGroups = groups.filter((g) => !g.parent_id).length;
  const maxDepth = useMemo(() => {
    let max = 0;
    hierarchicalGroups.forEach(({ depth }) => { if (depth > max) max = depth; });
    return max;
  }, [hierarchicalGroups]);

  // Toggle bulk selection
  const toggleSelect = (id: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const toggleSelectAll = () => {
    if (selectedIds.size === groups.length) {
      setSelectedIds(new Set());
    } else {
      setSelectedIds(new Set(groups.map((g) => g.id)));
    }
  };

  const filteredUsers = memberSearch
    ? users.filter(
        (u) =>
          u.username.toLowerCase().includes(memberSearch.toLowerCase()) ||
          (u.email || "").toLowerCase().includes(memberSearch.toLowerCase()) ||
          (u.display_name || "").toLowerCase().includes(memberSearch.toLowerCase())
      )
    : users.slice(0, 10);

  const inputCls = "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";

  // Edit/Create form component
  const renderForm = (isEdit: boolean) => (
    <div className="mb-6 rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
      <div className="mb-4 flex items-center justify-between">
        <h3 className="text-sm font-semibold dark:text-gray-100">{isEdit ? "Edit Group" : "Create New Group"}</h3>
        <button
          onClick={() => { isEdit ? setEditingGroup(null) : setShowCreate(false); setForm({ name: "", description: "", parent_id: "" }); }}
          className="text-gray-400 hover:text-gray-600"
        >
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
            className={inputCls}
          />
        </div>
        <div>
          <label className="mb-1 block text-xs font-medium text-gray-500">{t("groups.description")}</label>
          <input
            value={form.description}
            onChange={(e) => setForm({ ...form, description: e.target.value })}
            placeholder="Brief description"
            className={inputCls}
          />
        </div>
        <div>
          <label className="mb-1 block text-xs font-medium text-gray-500">{t("groups.parentgroup")}</label>
          <select
            value={form.parent_id}
            onChange={(e) => setForm({ ...form, parent_id: e.target.value })}
            className={inputCls}
          >
            <option value="">- Root (no parent) -</option>
            {groups
              .filter((g) => g.id !== editingGroup?.id)
              .map((g) => (
                <option key={g.id} value={g.id}>{g.name}</option>
              ))}
          </select>
        </div>
      </div>
      <button
        onClick={isEdit ? handleEdit : handleCreate}
        disabled={!form.name}
        className="mt-4 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
      >
        {isEdit ? "Save Changes" : "Create Group"}
      </button>
    </div>
  );

  return (
    <div>
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold dark:text-gray-100">{t("groups.groupsteams")}</h1>
          <p className="text-sm text-gray-500">Manage groups, roles, and memberships</p>
        </div>
        <button
          onClick={() => { setShowCreate(!showCreate); setEditingGroup(null); setError(null); }}
          className="flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
        >
          <Plus className="h-4 w-4" /> New Group
        </button>
      </div>

      {msg && <div className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700">{msg}</div>}
      {error && <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">{error}</div>}

      {/* Stats Summary */}
      {showStats && !loading && groups.length > 0 && (
        <div className="mb-4 grid grid-cols-4 gap-3">
          <div className="rounded-xl border border-gray-200 bg-white p-4 text-center shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <UsersIcon className="mx-auto mb-1 h-5 w-5 text-brand-600" />
            <p className="text-xl font-bold dark:text-gray-100">{groups.length}</p>
            <p className="text-xs text-gray-500">{t("groups.groups")}</p>
          </div>
          <div className="rounded-xl border border-gray-200 bg-white p-4 text-center shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <UserPlus className="mx-auto mb-1 h-5 w-5 text-blue-600" />
            <p className="text-xl font-bold dark:text-gray-100">{totalMembers}</p>
            <p className="text-xs text-gray-500">{t("groups.totalmembers")}</p>
          </div>
          <div className="rounded-xl border border-gray-200 bg-white p-4 text-center shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <FolderTree className="mx-auto mb-1 h-5 w-5 text-purple-600" />
            <p className="text-xl font-bold dark:text-gray-100">{rootGroups}</p>
            <p className="text-xs text-gray-500">{t("groups.rootgroups")}</p>
          </div>
          <div className="rounded-xl border border-gray-200 bg-white p-4 text-center shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <Crown className="mx-auto mb-1 h-5 w-5 text-amber-600" />
            <p className="text-xl font-bold dark:text-gray-100">{maxDepth + 1}</p>
            <p className="text-xs text-gray-500">{t("groups.hierarchydepth")}</p>
          </div>
        </div>
      )}

      {/* Create or Edit Form */}
      {showCreate && renderForm(false)}
      {editingGroup && renderForm(true)}

      {/* Bulk Actions Bar */}
      {selectedIds.size > 0 && (
        <div className="mb-4 flex flex-wrap items-center gap-2 rounded-xl border border-brand-300 bg-brand-50 p-3 dark:border-brand-700 dark:bg-brand-900/20">
          <span className="text-sm font-medium text-brand-700 dark:text-brand-300">
            {selectedIds.size} group{selectedIds.size > 1 ? "s" : ""} selected
          </span>
          <div className="ml-auto flex gap-2">
            <button
              onClick={() => setShowBulkRolePicker(true)}
              className="flex items-center gap-1.5 rounded-lg bg-white px-3 py-1.5 text-xs font-medium text-brand-700 shadow-sm hover:bg-brand-50 dark:bg-gray-800 dark:text-brand-300"
            >
              <Shield className="h-3.5 w-3.5" /> Add Role to Selected
            </button>
            <button
              onClick={() => setShowBulkMemberPicker(true)}
              className="flex items-center gap-1.5 rounded-lg bg-white px-3 py-1.5 text-xs font-medium text-brand-700 shadow-sm hover:bg-brand-50 dark:bg-gray-800 dark:text-brand-300"
            >
              <UserPlus className="h-3.5 w-3.5" /> Add Member to Selected
            </button>
            <button
              onClick={() => setShowDeleteConfirm(Array.from(selectedIds))}
              className="flex items-center gap-1.5 rounded-lg bg-red-50 px-3 py-1.5 text-xs font-medium text-red-700 hover:bg-red-100 dark:bg-red-950 dark:text-red-400"
            >
              <Trash2 className="h-3.5 w-3.5" /> Delete Selected
            </button>
            <button
              onClick={() => setSelectedIds(new Set())}
              className="rounded-lg px-3 py-1.5 text-xs text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700"
            >
              Clear
            </button>
          </div>
        </div>
      )}

      {/* Groups Table */}
      {loading ? (
        <p className="text-gray-500">Loading...</p>
      ) : hierarchicalGroups.length === 0 ? (
        <div className="rounded-xl border border-gray-200 bg-white p-12 text-center shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <UsersIcon className="mx-auto mb-4 h-12 w-12 text-gray-300" />
          <p className="text-gray-500">No groups created yet</p>
          <button
            onClick={() => setShowCreate(true)}
            className="mt-3 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
          >
            Create First Group
          </button>
        </div>
      ) : (
        <div className="overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="border-b border-gray-200 bg-gray-50 dark:border-gray-700 dark:bg-gray-700/50">
                <tr>
                  <th className="px-4 py-3 text-left">
                    <button onClick={toggleSelectAll} className="text-gray-400 hover:text-brand-600">
                      {selectedIds.size === groups.length && groups.length > 0 ? (
                        <CheckSquare className="h-4 w-4 text-brand-600" />
                      ) : (
                        <Square className="h-4 w-4" />
                      )}
                    </button>
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">{t("groups.name")}</th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">{t("groups.members")}</th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">{t("groups.parent")}</th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">{t("groups.roles")}</th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">{t("groups.created")}</th>
                  <th className="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500">{t("groups.actions")}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
                {hierarchicalGroups.map(({ group, depth }) => (
                  <GroupRow
                    key={group.id}
                    group={group}
                    depth={depth}
                    expanded={expandedId === group.id}
                    selected={selectedIds.has(group.id)}
                    roles={roles}
                    onToggleExpand={() => setExpandedId(expandedId === group.id ? null : group.id)}
                    onToggleSelect={() => toggleSelect(group.id)}
                    onEdit={() => startEdit(group)}
                    onDelete={() => setShowDeleteConfirm([group.id])}
                    onAssignRole={(rid) => handleAssignRole(group.id, rid)}
                    onRemoveRole={(rid) => handleRemoveRole(group.id, rid)}
                    onAddMember={() => handleAddMember(group.id)}
                    onRemoveMember={(mid) => handleRemoveMember(group.id, mid)}
                    onOpenMemberPicker={() => setShowMemberPicker(group.id)}
                    groupNameById={groupNameById}
                    addMemberInput={addMemberInput}
                    setAddMemberInput={setAddMemberInput}
                    showDeleteConfirm={showDeleteConfirm.includes(group.id)}
                    onConfirmDelete={() => handleDelete(group.id)}
                    onCancelDelete={() => setShowDeleteConfirm([])}
                  />
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Member Picker Modal */}
      {showMemberPicker && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowMemberPicker(null)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="mb-4 flex items-center justify-between">
              <h3 className="text-sm font-semibold dark:text-gray-100">{t("groups.addmembers")}</h3>
              <button onClick={() => setShowMemberPicker(null)} className="text-gray-400 hover:text-gray-600" aria-label="Close"><X className="h-5 w-5" /></button>
            </div>
            <div className="relative mb-3">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
              <input
                value={memberSearch}
                onChange={(e) => setMemberSearch(e.target.value)}
                placeholder="Search users..."
                className="w-full rounded-lg border border-gray-300 py-2 pl-9 pr-3 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
                autoFocus
              />
            </div>
            <div className="max-h-60 space-y-1 overflow-y-auto">
              {filteredUsers.length === 0 ? (
                <p className="py-4 text-center text-sm text-gray-400">No users found</p>
              ) : (
                filteredUsers.map((u) => (
                  <div
                    key={u.id}
                    className="flex items-center justify-between rounded-lg px-3 py-2 hover:bg-gray-50 dark:hover:bg-gray-700/50"
                  >
                    <div>
                      <span className="text-sm font-medium dark:text-gray-200">{u.display_name || u.username}</span>
                      <span className="ml-2 text-xs text-gray-500">{u.email}</span>
                    </div>
                    <button
                      onClick={() => handleAddMember(showMemberPicker, u.username)}
                      className="rounded-lg bg-brand-600 px-2 py-1 text-xs font-medium text-white hover:bg-brand-700"
                    >
                      <UserPlus className="inline h-3 w-3" /> Add
                    </button>
                  </div>
                ))
              )}
            </div>
          </div>
        </div>
      )}

      {/* Bulk Role Picker Modal */}
      {showBulkRolePicker && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowBulkRolePicker(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="mb-4 flex items-center justify-between">
              <h3 className="text-sm font-semibold dark:text-gray-100">Add Roles to {selectedIds.size} Group(s)</h3>
              <button onClick={() => setShowBulkRolePicker(false)} className="text-gray-400 hover:text-gray-600" aria-label="Close"><X className="h-5 w-5" /></button>
            </div>
            <div className="space-y-2">
              {roles.map((r) => (
                <label key={r.id} className="flex cursor-pointer items-center gap-3 rounded-lg border border-gray-200 p-3 hover:bg-gray-50 dark:border-gray-700 dark:hover:bg-gray-700/50">
                  <input
                    type="checkbox"
                    checked={bulkRoleSelection.has(r.id)}
                    onChange={() => {
                      setBulkRoleSelection((prev) => {
                        const next = new Set(prev);
                        if (next.has(r.id)) next.delete(r.id);
                        else next.add(r.id);
                        return next;
                      });
                    }}
                    className="h-4 w-4 rounded border-gray-300 text-brand-600"
                  />
                  <span className="text-sm font-medium dark:text-gray-200">{r.name || r.key}</span>
                </label>
              ))}
            </div>
            <div className="mt-4 flex justify-end gap-2">
              <button onClick={() => setShowBulkRolePicker(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm">{t("groups.cancel")}</button>
              <button
                onClick={handleBulkAddRole}
                disabled={bulkRoleSelection.size === 0}
                className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
              >
                Add {bulkRoleSelection.size} Role(s)
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Bulk Member Picker Modal */}
      {showBulkMemberPicker && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowBulkMemberPicker(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="mb-4 flex items-center justify-between">
              <h3 className="text-sm font-semibold dark:text-gray-100">Add Member to {selectedIds.size} Group(s)</h3>
              <button onClick={() => setShowBulkMemberPicker(false)} className="text-gray-400 hover:text-gray-600" aria-label="Close"><X className="h-5 w-5" /></button>
            </div>
            <input
              value={bulkMemberInput}
              onChange={(e) => setBulkMemberInput(e.target.value)}
              onKeyDown={(e) => { if (e.key === "Enter") handleBulkAddMember(); }}
              placeholder="Enter username..."
              className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
              autoFocus
            />
            <div className="mt-4 flex justify-end gap-2">
              <button onClick={() => setShowBulkMemberPicker(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm">{t("groups.cancel")}</button>
              <button
                onClick={handleBulkAddMember}
                disabled={!bulkMemberInput.trim()}
                className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
              >
                Add to All Selected
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

// Group Row Component
function GroupRow({
  group, depth, expanded, selected, roles,
  onToggleExpand, onToggleSelect, onEdit, onDelete,
  onAssignRole, onRemoveRole,
  onAddMember, onRemoveMember, onOpenMemberPicker,
  groupNameById, addMemberInput, setAddMemberInput,
  showDeleteConfirm, onConfirmDelete, onCancelDelete,
}: {
  group: Group; depth: number; expanded: boolean; selected: boolean; roles: Role[];
  onToggleExpand: () => void; onToggleSelect: () => void; onEdit: () => void; onDelete: () => void;
  onAssignRole: (roleId: string) => void; onRemoveRole: (roleId: string) => void;
  onAddMember: () => void; onRemoveMember: (memberId: string) => void; onOpenMemberPicker: () => void;
  groupNameById: (id?: string) => string;
  addMemberInput: string; setAddMemberInput: (v: string) => void;
  showDeleteConfirm: boolean; onConfirmDelete: () => void; onCancelDelete: () => void;
}) {
  return (
    <>
      <tr className={`hover:bg-gray-50 dark:hover:bg-gray-700/50 ${selected ? "bg-brand-50/30 dark:bg-brand-900/10" : ""}`}>
        <td className="px-4 py-3">
          <button onClick={onToggleSelect} className="text-gray-400 hover:text-brand-600">
            {selected ? <CheckSquare className="h-4 w-4 text-brand-600" /> : <Square className="h-4 w-4" />}
          </button>
        </td>
        <td className="px-4 py-3">
          <div className="flex items-center" style={{ paddingLeft: `${depth * 24}px` }}>
            {depth > 0 && <span className="mr-1 text-gray-300">|</span>}
            <button onClick={onToggleExpand} className="mr-1.5 text-gray-400 hover:text-gray-600">
              {expanded ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
            </button>
            <div>
              <p className="text-sm font-medium dark:text-gray-100">{group.name}</p>
              {group.description && <p className="text-xs text-gray-500">{group.description}</p>}
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
              <span
                key={r.id}
                className="inline-flex items-center gap-0.5 rounded-full bg-purple-50 px-2 py-0.5 text-xs text-purple-700 dark:bg-purple-900/30 dark:text-purple-300"
              >
                {r.name || r.key}
                <button onClick={() => onRemoveRole(r.id)} aria-label="Remove role" className="ml-0.5 text-purple-400 hover:text-red-500">
                  <X className="h-3 w-3" />
                </button>
              </span>
            ))}
            {(!group.roles || group.roles.length === 0) && <span className="text-xs text-gray-300">-</span>}
          </div>
        </td>
        <td className="px-4 py-3 text-sm text-gray-500">
          {group.created_at ? new Date(group.created_at).toLocaleDateString() : "-"}
        </td>
        <td className="px-4 py-3">
          <div className="flex justify-end gap-1">
            <button onClick={onEdit} title="Edit" className="rounded p-1.5 text-gray-400 hover:bg-blue-50 hover:text-blue-600">
              <Pencil className="h-4 w-4" />
            </button>
            <button onClick={onDelete} title="Delete" className="rounded p-1.5 text-gray-400 hover:bg-red-50 hover:text-red-600">
              <Trash2 className="h-4 w-4" />
            </button>
          </div>
        </td>
      </tr>
      {/* Expanded row */}
      {expanded && (
        <tr className="bg-gray-50 dark:bg-gray-700/30">
          <td colSpan={7} className="px-8 py-4">
            {showDeleteConfirm ? (
              <div className="flex items-center justify-between rounded-lg border border-red-300 bg-red-50 p-4 dark:border-red-800 dark:bg-red-950">
                <div>
                  <p className="text-sm font-medium text-red-700 dark:text-red-400">Delete "{group.name}"?</p>
                  <p className="text-xs text-red-600 dark:text-red-500">This action cannot be undone.</p>
                </div>
                <div className="flex gap-2">
                  <button onClick={onCancelDelete} className="rounded-lg border border-gray-300 px-3 py-1.5 text-xs">Cancel</button>
                  <button onClick={onConfirmDelete} className="rounded-lg bg-red-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-red-700">Delete</button>
                </div>
              </div>
            ) : (
              <div className="space-y-4">
                {/* Role assignment */}
                <div>
                  <h4 className="mb-2 flex items-center gap-1.5 text-xs font-semibold text-gray-600">
                    <Shield className="h-3.5 w-3.5" /> Assign Role
                  </h4>
                  <div className="flex gap-2">
                    <select
                      onChange={(e) => { onAssignRole(e.target.value); e.target.value = ""; }}
                      defaultValue=""
                      className="rounded-lg border border-gray-300 px-3 py-1.5 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
                    >
                      <option value="" disabled>Select a role to assign...</option>
                      {roles.map((r) => <option key={r.id} value={r.id}>{r.name || r.key}</option>)}
                    </select>
                  </div>
                </div>
                {/* Member management */}
                <div>
                  <div className="mb-2 flex items-center justify-between">
                    <h4 className="flex items-center gap-1.5 text-xs font-semibold text-gray-600">
                      <UsersIcon className="h-3.5 w-3.5" /> Members ({group.members?.length ?? 0})
                    </h4>
                    <button
                      onClick={onOpenMemberPicker}
                      className="flex items-center gap-1 rounded-lg bg-brand-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-brand-700"
                    >
                      <Search className="h-3 w-3" /> Search Users
                    </button>
                  </div>
                  <div className="mb-2 flex gap-2">
                    <input
                      value={addMemberInput}
                      onChange={(e) => setAddMemberInput(e.target.value)}
                      onKeyDown={(e) => { if (e.key === "Enter") onAddMember(); }}
                      placeholder="Enter username to add..."
                      className="rounded-lg border border-gray-300 px-3 py-1.5 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
                    />
                    <button
                      onClick={onAddMember}
                      disabled={!addMemberInput.trim()}
                      className="flex items-center gap-1 rounded-lg bg-brand-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-brand-700 disabled:opacity-50"
                    >
                      <UserPlus className="h-3 w-3" /> Quick Add
                    </button>
                  </div>
                  <div className="space-y-1">
                    {(group.members || []).map((m) => (
                      <div key={m.id} className="flex items-center justify-between rounded-lg bg-white px-3 py-1.5 dark:bg-gray-800">
                        <div>
                          <span className="text-sm font-medium dark:text-gray-200">{m.username}</span>
                          <span className="ml-2 text-xs text-gray-500">{m.email}</span>
                        </div>
                        <button onClick={() => onRemoveMember(m.id)} className="rounded p-1 text-gray-400 hover:bg-red-50 hover:text-red-600">
                          <UserMinus className="h-3.5 w-3.5" />
                        </button>
                      </div>
                    ))}
                    {(!group.members || group.members.length === 0) && <p className="text-xs text-gray-400">No members in this group</p>}
                  </div>
                </div>
              </div>
            )}
          </td>
        </tr>
      )}
    </>
  );
}

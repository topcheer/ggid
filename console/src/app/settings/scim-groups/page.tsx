"use client";

import { useState, useEffect, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
import { useTranslations } from "@/lib/i18n";
  Users, Plus, Trash2, X, AlertCircle, Loader2, Check, UserMinus,
} from "lucide-react";

interface ScimGroup {
  id: string;
  display_name: string;
  members: { value: string; display: string }[];
  meta: { created: string; last_modified: string };
}

export default function ScimGroupsPage() {
  const t = useTranslations();
  const { apiFetch } = useApi();
  const [groups, setGroups] = useState<ScimGroup[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState<ScimGroup | null>(null);
  const [selectedGroup, setSelectedGroup] = useState<ScimGroup | null>(null);
  const [form, setForm] = useState({ display_name: "", member_ids: "" });
  const [creating, setCreating] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiFetch<{ Resources?: ScimGroup[]; groups?: ScimGroup[]; items?: ScimGroup[] }>("/api/v1/scim/Groups").catch(() => null);
      setGroups(data?.Resources ?? data?.groups ?? data?.items ?? []);
    } catch {
      setError("Failed to load SCIM groups");
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => { load(); }, [load]);

  const handleCreate = async () => {
    if (!form.display_name.trim()) return;
    setCreating(true);
    try {
      const memberIds = form.member_ids.split(",").map((s) => s.trim()).filter(Boolean);
      await apiFetch("/api/v1/scim/Groups", {
        method: "POST",
        body: JSON.stringify({ displayName: form.display_name, members: memberIds.map((id) => ({ value: id })) }),
      });
      setForm({ display_name: "", member_ids: "" });
      setShowCreate(false);
      await load();
    } catch {
      setError("Failed to create group");
    } finally {
      setCreating(false);
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await apiFetch(`/api/v1/scim/Groups/${id}`, { method: "DELETE" });
      setConfirmDelete(null);
      setSelectedGroup(null);
      await load();
    } catch {
      setError("Failed to delete group");
    }
  };

  const handleRemoveMember = async (groupId: string, userId: string) => {
    try {
      await apiFetch(`/api/v1/scim/Groups/${groupId}/members/${userId}`, { method: "DELETE" });
      await load();
      if (selectedGroup?.id === groupId) {
        const updated = groups.find((g) => g.id === groupId);
        if (updated) setSelectedGroup(updated);
      }
    } catch {
      setError("Failed to remove member");
    }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <Users className="h-6 w-6 text-indigo-600" /> SCIM Groups
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">SCIM 2.0 group provisioning for automated user lifecycle management.</p>
        </div>
        <button onClick={() => setShowCreate(true)} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700"><Plus className="h-4 w-4" /> New Group</button>
      </div>

      {error && (
        <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      <div className="grid gap-6 lg:grid-cols-3">
        {/* Groups list */}
        <div className="lg:col-span-2">
          {loading ? (
            <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>
          ) : groups.length === 0 ? (
            <div className={cardCls}><div className="py-12 text-center"><Users className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No SCIM groups.</p></div></div>
          ) : (
            <div className="space-y-3">
              {groups.map((g) => (
                <div key={g.id} className={`${cardCls} cursor-pointer transition ${selectedGroup?.id === g.id ? "ring-2 ring-indigo-400" : ""}`} onClick={() => setSelectedGroup(g)}>
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <div className="rounded-lg bg-indigo-100 p-2 dark:bg-indigo-900/30"><Users className="h-4 w-4 text-indigo-600" /></div>
                      <div>
                        <p className="font-medium text-gray-800 dark:text-gray-200">{g.display_name}</p>
                        <p className="text-xs text-gray-400">{g.members.length} members</p>
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="text-xs text-gray-400">Created {new Date(g.meta?.created ?? Date.now()).toLocaleDateString()}</span>
                      <button onClick={(e) => { e.stopPropagation(); setConfirmDelete(g); }} className="rounded-lg p-1.5 text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20"><Trash2 className="h-4 w-4" /></button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Members panel */}
        <div>
          <h2 className="mb-3 text-sm font-semibold uppercase text-gray-500">Members</h2>
          {selectedGroup ? (
            <div className={cardCls}>
              <p className="mb-3 font-medium text-gray-800 dark:text-gray-200">{selectedGroup.display_name}</p>
              {selectedGroup.members.length === 0 ? (
                <p className="py-4 text-center text-xs text-gray-400">No members in this group.</p>
              ) : (
                <div className="space-y-2">
                  {selectedGroup.members.map((m) => (
                    <div key={m.value} className="flex items-center justify-between rounded-lg bg-gray-50 px-3 py-2 dark:bg-gray-900/30">
                      <span className="text-sm text-gray-600 dark:text-gray-300">{m.display || m.value}</span>
                      <button onClick={() => handleRemoveMember(selectedGroup.id, m.value)} className="rounded p-1 text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20"><UserMinus className="h-3.5 w-3.5" /></button>
                    </div>
                  ))}
                </div>
              )}
            </div>
          ) : (
            <div className={cardCls}><div className="py-8 text-center"><Users className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-xs text-gray-400">Select a group to view members.</p></div></div>
          )}
        </div>
      </div>

      {/* Create modal */}
      {showCreate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => !creating && setShowCreate(false)}>
          <div className="w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-semibold text-gray-900 dark:text-white">New SCIM Group</h2>
              <button onClick={() => setShowCreate(false)}><X className="h-5 w-5 text-gray-400" /></button>
            </div>
            <div className="mt-4 space-y-4">
              <div><label className="text-sm font-medium text-gray-700 dark:text-gray-300">Group Name</label><input value={form.display_name} onChange={(e) => setForm((p) => ({ ...p, display_name: e.target.value }))} placeholder="Engineering Team" className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" /></div>
              <div><label className="text-sm font-medium text-gray-700 dark:text-gray-300">Member IDs (comma-separated)</label><input value={form.member_ids} onChange={(e) => setForm((p) => ({ ...p, member_ids: e.target.value }))} placeholder="user-1, user-2" className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" /></div>
            </div>
            <div className="mt-5 flex justify-end gap-2">
              <button onClick={() => setShowCreate(false)} className="rounded-lg px-4 py-2 text-sm text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700">Cancel</button>
              <button onClick={handleCreate} disabled={!form.display_name.trim() || creating} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{creating ? <Loader2 className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />}Create</button>
            </div>
          </div>
        </div>
      )}

      {/* Delete confirm */}
      {confirmDelete && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setConfirmDelete(null)}>
          <div className="w-full max-w-sm rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center gap-3"><div className="rounded-full bg-red-100 p-2 dark:bg-red-900/30"><Trash2 className="h-5 w-5 text-red-600" /></div><div><h2 className="font-semibold text-gray-900 dark:text-white">Delete {confirmDelete.display_name}?</h2><p className="text-sm text-gray-500">All {confirmDelete.members.length} members will be removed from this group.</p></div></div>
            <div className="mt-5 flex justify-end gap-2"><button onClick={() => setConfirmDelete(null)} className="rounded-lg px-4 py-2 text-sm text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700">Cancel</button><button onClick={() => handleDelete(confirmDelete.id)} className="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700">Delete</button></div>
          </div>
        </div>
      )}
    </div>
  );
}

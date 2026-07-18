"use client";
import { useState, useEffect, useCallback } from "react";
import { Users, RefreshCw, Plus, X } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface GroupMap { id: string; external_group: string; local_role: string; auto_provision: boolean; sync_direction: "inbound" | "outbound" | "bidirectional"; last_sync: string; last_status: "success" | "failed" | "pending"; }

export default function ScimGroupMappingPage() {
  const t = useTranslations();
  const [mappings, setMappings] = useState<GroupMap[]>([]);
  const [loading, setLoading] = useState(false);
  const [showAdd, setShowAdd] = useState(false);
  const [form, setForm] = useState({ external_group: "", local_role: "", auto_provision: false, sync_direction: "inbound" as "inbound" | "outbound" | "bidirectional" });

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/identity/scim-group-mapping", { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const d = await res.json(); setMappings(d.mappings || d || []); }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const statusColors: Record<string, string> = { success: "text-green-600", failed: "text-red-600", pending: "text-yellow-600" };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold flex items-center gap-2"><Users className="w-6 h-6 text-teal-500" /> SCIM Group Mapping</h1><p className="text-sm text-gray-500 mt-1">Map external SCIM groups to local roles with auto-provisioning.</p></div>
        <button onClick={() => setShowAdd(true)} className="px-4 py-2 rounded-lg bg-teal-600 text-white text-sm font-medium flex items-center gap-2"><Plus className="w-4 h-4" /> Add Mapping</button>
      </div>

      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">External Group</th><th className="px-4 py-3 text-left font-medium">Local Role</th><th className="px-4 py-3 text-left font-medium">Auto Provision</th><th className="px-4 py-3 text-left font-medium">Sync Direction</th><th className="px-4 py-3 text-left font-medium">Last Sync</th><th className="px-4 py-3 text-left font-medium">Status</th></tr></thead>
          <tbody className="divide-y dark:divide-gray-800">{mappings.map((m: any) => (<tr key={m.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3 font-mono text-xs">{m.external_group}</td><td className="px-4 py-3"><span className="px-2 py-0.5 rounded text-xs bg-teal-50 dark:bg-teal-900/20 font-mono">{m.local_role}</span></td><td className="px-4 py-3">{m.auto_provision ? <span className="text-xs text-green-600">Yes</span> : <span className="text-xs text-gray-400">No</span>}</td><td className="px-4 py-3 text-xs text-gray-500">{m.sync_direction}</td><td className="px-4 py-3 text-xs text-gray-400">{m.last_sync}</td><td className="px-4 py-3"><span className={"text-xs font-medium " + statusColors[m.last_status]}>{m.last_status}</span></td></tr>))}{mappings.length === 0 && !loading && <tr><td colSpan={6} className="px-4 py-8 text-center text-gray-500">No group mappings.</td></tr>}</tbody>
        </table>
      </div>

      {showAdd && (<div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setShowAdd(false)}><div role="dialog" aria-modal="true" className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4" onClick={(e) => e.stopPropagation()}><div className="flex items-center justify-between px-6 py-4 border-b dark:border-gray-800"><h3 className="font-semibold">Add Group Mapping</h3><button onClick={() => setShowAdd(false)} aria-label="Close"><X className="w-5 h-5 text-gray-400" /></button></div><div className="px-6 py-4 space-y-3"><div><label className="text-sm font-medium">External Group</label><input type="text" value={form.external_group} onChange={(e) => setForm({ ...form, external_group: e.target.value })} placeholder="AzureAD_Developers" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div><div><label className="text-sm font-medium">Local Role</label><input type="text" value={form.local_role} onChange={(e) => setForm({ ...form, local_role: e.target.value })} placeholder="developer" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div><div><label className="text-sm font-medium">Sync Direction</label><select value={form.sync_direction} onChange={(e) => setForm({ ...form, sync_direction: e.target.value as "inbound" | "outbound" | "bidirectional" })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm"><option value="inbound">Inbound</option><option value="outbound">Outbound</option><option value="bidirectional">Bidirectional</option></select></div><label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={form.auto_provision} onChange={(e) => setForm({ ...form, auto_provision: e.target.checked })} className="rounded" /> Auto-provision new users</label></div><div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800"><button onClick={() => setShowAdd(false)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">Cancel</button><button onClick={() => setShowAdd(false)} disabled={!form.external_group || !form.local_role} className="px-4 py-2 rounded-lg bg-teal-600 text-white text-sm font-medium disabled:opacity-50">Save</button></div></div></div>)}
    </div>
  );
}

"use client";
import { useState, useEffect, useCallback } from "react";
import { Cloud, AlertTriangle, RefreshCw } from "lucide-react";
interface Mapping { scim_attr: string; local_attr: string; }
interface ErrorItem { id: string; user: string; error: string; timestamp: string; }
interface Config { endpoint_url: string; mappings: Mapping[]; rules: { create: boolean; update: boolean; deactivate: boolean; }; sync_direction: "inbound" | "outbound" | "bidirectional"; last_sync: string; last_status: "success" | "partial" | "failed"; error_queue: ErrorItem[]; }
export default function ScimProvisioningPage() {
  const [data, setData] = useState<Config | null>(null);
  const [loading, setLoading] = useState(false);
  const fetchData = useCallback(async () => {
    setLoading(true);
    try { const res = await fetch("/api/v1/identity/scim-provisioning", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); if (res.ok) setData(await res.json()); }
    catch { /* noop */ } finally { setLoading(false); }
  }, []);
  useEffect(() => { fetchData(); }, [fetchData]);
  const statusColor = data?.last_status === "success" ? "text-green-600" : data?.last_status === "partial" ? "text-yellow-600" : "text-red-600";
  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><Cloud className="w-6 h-6 text-blue-500" /> SCIM Provisioning</h1><p className="text-sm text-gray-500 mt-1">Manage SCIM endpoint mapping and sync configuration.</p></div>
      {data && (<>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div className="rounded-lg border p-4 dark:border-gray-800 col-span-2"><span className="text-sm text-gray-500">Endpoint URL</span><p className="font-mono text-xs mt-1">{data.endpoint_url}</p></div>
          <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Direction</span><p className="font-bold mt-1">{data.sync_direction}</p></div>
          <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Last Sync</span><p className={"font-bold mt-1 " + statusColor}>{data.last_status}</p><p className="text-xs text-gray-400">{data.last_sync}</p></div>
        </div>
        <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">Attribute Mappings</h3><table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-2 text-left font-medium">SCIM Attribute</th><th className="px-4 py-2 text-left font-medium">Local Attribute</th></tr></thead><tbody className="divide-y dark:divide-gray-800">{data.mappings.map((m, i) => (<tr key={i} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-2 font-mono text-xs">{m.scim_attr}</td><td className="px-4 py-2 font-mono text-xs text-blue-600">{m.local_attr}</td></tr>))}</tbody></table></div>
        <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">Provisioning Rules</h3><div className="flex gap-4">{(["create", "update", "deactivate"] as const).map((r) => (<label key={r} className="flex items-center gap-2 cursor-pointer"><input type="checkbox" checked={data.rules[r]} readOnly className="rounded" /><span className="text-sm capitalize">{r}</span></label>))}</div></div>
        {data.error_queue.length > 0 && (<div className="rounded-lg border border-red-300 dark:border-red-800 p-4"><h3 className="text-sm font-semibold text-red-600 mb-2 flex items-center gap-2"><AlertTriangle className="w-4 h-4" /> Error Queue ({data.error_queue.length})</h3><div className="space-y-1">{data.error_queue.map((e) => (<div key={e.id} className="flex items-center gap-2 text-xs"><span className="font-medium">{e.user}</span><span className="text-gray-500 flex-1">{e.error}</span><span className="text-gray-400">{e.timestamp}</span></div>))}</div></div>)}
      </>)}
      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
    </div>
  );
}

"use client";

import { useScimGroupMappingConfig } from "@ggid/sdk-react";
import { RefreshCw, Plus } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function ScimGroupMappingConfigPage() {
  const t = useTranslations();
  const { data, loading, error, refresh } = useScimGroupMappingConfig();
  if (loading) return <div className="p-8 text-gray-400">Loading SCIM group mapping...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div><h1 className="text-2xl font-bold">SCIM Group Mapping</h1><p className="text-sm text-gray-400 mt-1">Map external groups to local roles</p></div>
        <div className="flex gap-2"><button className="flex items-center gap-1 px-3 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg text-sm font-medium transition"><Plus className="w-4 h-4" /> Add Mapping</button><button onClick={refresh} className="flex items-center gap-1 px-3 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"><RefreshCw className="w-4 h-4" /> Sync</button></div>
      </div>

      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold mb-4">Group Mappings</h2>
        <table className="w-full text-sm"><thead><tr className="border-b border-gray-800 text-gray-400"><th className="text-left py-2">External Group</th><th className="text-left py-2">Local Role</th><th className="text-left py-2">Auto Provision</th><th className="text-left py-2">Direction</th></tr></thead>
          <tbody>{(data?.mappings ?? []).map((m) => (
            <tr key={m.id} className="border-b border-gray-800">
              <td className="py-2 font-mono text-xs text-blue-400">{m.external_group}</td>
              <td className="py-2 font-mono text-xs text-green-400">{m.local_role}</td>
              <td className="py-2"><span className={"text-xs px-2 py-0.5 rounded " + (m.auto_provision ? "bg-green-900 text-green-300" : "bg-gray-700")}>{m.auto_provision ? "ON" : "OFF"}</span></td>
              <td className="py-2"><span className="text-xs px-2 py-0.5 rounded bg-gray-700">{m.sync_direction}</span></td>
            </tr>
          ))}</tbody>
        </table>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="bg-gray-900 rounded-xl p-6"><h2 className="text-sm font-semibold mb-3">Per-App Override</h2><div className="space-y-2">{(data?.per_app ?? []).map((a) => (<div key={a.app} className="flex items-center gap-2 bg-gray-800 rounded p-2 text-xs"><span className="flex-1">{a.app}</span><span className="text-gray-400">{a.mapping_count} mappings</span></div>))}</div></div>
        <div className="bg-gray-900 rounded-xl p-6"><h2 className="text-sm font-semibold mb-3">Last Sync</h2><div className="space-y-1 text-xs text-gray-400"><div>Status: <span className="text-green-400">{data?.last_sync?.status}</span></div><div>Synced: {data?.last_sync?.synced_at}</div><div>Added: {data?.last_sync?.added} / Removed: {data?.last_sync?.removed} / Errors: {data?.last_sync?.errors}</div></div></div>
      </div>
    </div>
  );
}

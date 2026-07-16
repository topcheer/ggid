"use client";
import { useState, useEffect } from "react";
import { useTranslations } from "@/lib/i18n";

interface SyncHistory {
  timestamp: string;
  direction: "inbound" | "outbound";
  records: number;
  status: "success" | "partial" | "failed";
}

interface MappingRule {
  source_attr: string;
  ggid_field: string;
}

export default function UserProvisioningCenterPage() {
  const t = useTranslations();

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [sources, setSources] = useState<{ name: string; status: string }[]>([]);
  const [syncHistory, setSyncHistory] = useState<SyncHistory[]>([]);
  const [mappings, setMappings] = useState<MappingRule[]>([]);
  const [conflictPolicy, setConflictPolicy] = useState("skip");
  const [dryRun, setDryRun] = useState(false);

  useEffect(() => {
    fetch("/api/v1/users/bulk-provision", {
      headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
    })
      .then(res => { if (!res.ok) return null; return res.json(); })
      .then(data => {
        setSources(data.sources || []);
        setSyncHistory(data.syncHistory || data.sync_history || []);
        setMappings(data.mappings || []);
        setLoading(false);
      })
      .catch(err => { setError(err.message); setLoading(false); });
  }, []);

  const statusColors: Record<string, string> = { connected: "bg-green-100 text-green-700", active: "bg-green-100 text-green-700", disconnected: "bg-red-100 text-red-700", error: "bg-red-100 text-red-700", inactive: "bg-gray-100 text-gray-500" };
  const syncColors: Record<string, string> = { success: "bg-green-100 text-green-700", partial: "bg-yellow-100 text-yellow-700", failed: "bg-red-100 text-red-700" };

  if (loading) return (
    <div className="p-8"><h1 className="text-2xl font-bold mb-4">User Provisioning Center</h1><p>Loading...</p></div>
  );
  if (error) return (
    <div className="p-8"><h1 className="text-2xl font-bold mb-4">User Provisioning Center</h1><p className="text-red-600">Error: {error}</p></div>
  );
  return (
    <div className="p-8 space-y-6 max-w-5xl">
      <h1 className="text-2xl font-bold">User Provisioning Center</h1>
      <p className="text-gray-600">Manage provisioning sources, SCIM sync, attribute mapping, and conflict resolution.</p>

      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Provisioning Sources</h2><div className="grid grid-cols-4 gap-4">{sources.map((s, i) => (<div key={i} className="border rounded p-3"><div className="font-medium">{s.name}</div><span className={`mt-1 inline-block px-2 py-0.5 rounded text-xs ${statusColors[s.status] || ""}`}>{s.status}</span></div>))}</div></div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4"><h2 className="text-lg font-semibold">Mapping Rules</h2><table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Source Attribute</th><th scope="col">GGID Field</th></tr></thead><tbody>{mappings.map((m: MappingRule, i: number) => (<tr key={i} className="border-b"><td className="py-2 font-mono text-xs">{m.source_attr}</td><td><span className="text-gray-400 mx-2">{"->"}</span><span className="font-mono text-xs">{m.ggid_field}</span></td></tr>))}</tbody></table></div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4"><h2 className="text-lg font-semibold">Sync Settings</h2><div className="grid grid-cols-2 gap-4"><div><label className="block text-sm font-medium mb-1">Conflict Resolution</label><select value={conflictPolicy} onChange={(e) => setConflictPolicy(e.target.value)} className="border rounded px-3 py-2 w-full"><option value="skip">Skip</option><option value="overwrite">Overwrite</option><option value="merge">Merge</option></select></div><div className="flex items-center pt-6"><input type="checkbox" checked={dryRun} onChange={(e) => setDryRun(e.target.checked)} className="w-4 h-4 mr-2" /><label className="text-sm">Dry-Run Mode</label></div></div></div>

      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Sync History</h2><table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Timestamp</th><th scope="col">Direction</th><th>Records</th><th>Status</th></tr></thead><tbody>{syncHistory.map((h: SyncHistory, i: number) => (<tr key={i} className="border-b"><td className="py-2 text-xs text-gray-500">{h.timestamp}</td><td><span className="px-2 py-1 bg-blue-100 text-blue-700 rounded text-xs">{h.direction}</span></td><td>{h.records}</td><td><span className={`px-2 py-1 rounded text-xs ${syncColors[h.status] || ""}`}>{h.status}</span></td></tr>))}</tbody></table></div>
    </div>
  );
}

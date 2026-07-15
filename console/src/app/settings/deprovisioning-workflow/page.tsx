"use client";
import { useState, useEffect } from "react";
import { useTranslations } from "@/lib/i18n";

interface DeprovisionUser {
  user_id: string;
  username: string;
  department: string;
  stage: "notify" | "disable" | "revoke" | "archive" | "done";
  linked_accounts: number;
  grace_remaining_days: number;
}

const defaultQueue: DeprovisionUser[] = [];

const stages = ["notify", "disable", "revoke", "archive", "done"] as const;

export default function DeprovisioningWorkflowPage() {
  const t = useTranslations();

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [queue, setQueue] = useState<DeprovisionUser[]>(defaultQueue);
  const [gracePeriodDays, setGracePeriodDays] = useState(7);
  const [dryRun, setDryRun] = useState(false);
  const [cascadePreview, setCascadePreview] = useState<DeprovisionUser | null>(null);

  useEffect(() => {
    fetch("/api/v1/identity/deprovisioning/config", {
      headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
    })
      .then(res => { if (!res.ok) throw new Error(`HTTP ${res.status}`); return res.json(); })
      .then(data => { setQueue(data.queue || data.items || []); setLoading(false); })
      .catch(err => { setError(err.message); setLoading(false); });
  }, []);

  const stageColors: Record<string, string> = { notify: "bg-blue-100 text-blue-700", disable: "bg-yellow-100 text-yellow-700", revoke: "bg-orange-100 text-orange-700", archive: "bg-gray-100 text-gray-600", done: "bg-green-100 text-green-700" };
  const stageIndex = (s: string) => stages.indexOf(s as typeof stages[number]);

  if (loading) return (
    <div className="p-8"><h1 className="text-2xl font-bold mb-4">Deprovisioning Workflow</h1><p>Loading...</p></div>
  );
  if (error) return (
    <div className="p-8"><h1 className="text-2xl font-bold mb-4">Deprovisioning Workflow</h1><p className="text-red-600">Error: {error}</p></div>
  );
  return (
    <div className="p-8 space-y-6 max-w-5xl">
      <h1 className="text-2xl font-bold">Deprovisioning Workflow</h1>
      <p className="text-gray-600">Manage user deprovisioning with staged workflow, cascade preview, and bulk operations.</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Configuration</h2>
        <div className="grid grid-cols-2 gap-4">
          <div><label className="block text-sm font-medium mb-1">Grace Period (days)</label><input type="number" value={gracePeriodDays} onChange={(e) => setGracePeriodDays(parseInt(e.target.value) || 0)} className="border rounded px-3 py-2 w-32" /></div>
          <div className="flex items-center gap-3 pt-6"><input type="checkbox" checked={dryRun} onChange={(e) => setDryRun(e.target.checked)} className="w-4 h-4" /><label>Dry-Run Mode (no actual changes)</label></div>
        </div>
        <div className="border-2 border-dashed border-gray-300 rounded-lg p-4 text-center"><p className="text-sm text-gray-500">Bulk Deprovision: Drop CSV file here (user_id column required)</p><button className="mt-2 px-4 py-1 bg-gray-100 border rounded text-sm hover:bg-gray-200">Upload CSV</button></div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Workflow Steps</h2>
        <div className="flex items-center gap-2">
          {stages.map((s, i) => (
            <div key={s} className="flex items-center">
              <div className={`px-3 py-2 rounded text-xs font-medium ${stageColors[s]}`}>{s}</div>
              {i < stages.length - 1 && <span className="text-gray-400 mx-1">{"->"}</span>}
            </div>
          ))}
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Deprovisioning Queue</h2>
        <table className="w-full text-sm">
          <thead><tr className="border-b text-left"><th className="py-2">Username</th><th>Department</th><th>Stage</th><th>Progress</th><th>Linked Accounts</th><th>Grace Remaining</th><th>Cascade</th></tr></thead>
          <tbody>
            {queue.map((u: DeprovisionUser, i: number) => (
              <tr key={i} className="border-b hover:bg-gray-50">
                <td className="py-2 font-medium">{u.username}</td>
                <td>{u.department}</td>
                <td><span className={`px-2 py-1 rounded text-xs ${stageColors[u.stage] || ""}`}>{u.stage}</span></td>
                <td><div className="w-32 bg-gray-200 rounded-full h-2"><div className="bg-blue-600 h-2 rounded-full" style={{ width: `${(stageIndex(u.stage) + 1) / stages.length * 100}%` }} /></div></td>
                <td>{u.linked_accounts > 0 ? <span className="text-yellow-600 font-medium">{u.linked_accounts}</span> : "0"}</td>
                <td>{u.grace_remaining_days > 0 ? `${u.grace_remaining_days}d` : <span className="text-red-500">expired</span>}</td>
                <td><button onClick={() => setCascadePreview(u)} className="text-xs text-blue-600 hover:underline">Preview</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {cascadePreview && (
        <div className="bg-white rounded-lg p-6 shadow space-y-3">
          <div className="flex items-center justify-between"><h2 className="text-lg font-semibold">Cascade Preview: {cascadePreview.username}</h2><button onClick={() => setCascadePreview(null)} className="text-gray-400 hover:text-gray-600">Close</button></div>
          <div className="text-sm text-gray-600">Deprovisioning <strong>{cascadePreview.username}</strong> will affect <strong>{cascadePreview.linked_accounts}</strong> linked accounts:</div>
          <ul className="space-y-1 text-sm">
            {Array.from({ length: cascadePreview.linked_accounts }, (_, j) => (
              <li key={j} className="border-b py-1"><span className="font-mono text-xs">linked-account-{cascadePreview.user_id}-{j + 1}</span> <span className="ml-2 text-xs text-gray-400">{dryRun ? "[dry-run: will be disabled]" : "[will be disabled]"}</span></li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}

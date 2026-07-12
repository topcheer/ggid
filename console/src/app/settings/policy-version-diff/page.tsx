"use client";

import { useState, useCallback } from "react";
import { GitCompare, Plus, Minus, Edit3, AlertTriangle } from "lucide-react";

interface FieldChange {
  field: string;
  change_type: "added" | "removed" | "modified";
  old_value: string;
  new_value: string;
}

interface VersionDiff {
  version_a: string;
  version_b: string;
  field_changes: FieldChange[];
  impact_summary: { affected_users: number; affected_resources: number; rules_changed: number };
  breaking_changes: string[];
}

interface Policy { id: string; name: string; versions: string[] }

const changeConfig: Record<string, { icon: typeof Plus; color: string }> = {
  added: { icon: Plus, color: "text-green-600" },
  removed: { icon: Minus, color: "text-red-600" },
  modified: { icon: Edit3, color: "text-yellow-600" },
};

export default function PolicyVersionDiffPage() {
  const [policies] = useState<Policy[]>([{ id: "p1", name: "Data Access", versions: ["v1.0", "v1.1", "v2.0"] }, { id: "p2", name: "Admin Access", versions: ["v1.0", "v2.0"] }]);
  const [policyId, setPolicyId] = useState("");
  const [versionA, setVersionA] = useState("");
  const [versionB, setVersionB] = useState("");
  const [data, setData] = useState<VersionDiff | null>(null);
  const [loading, setLoading] = useState(false);

  const selectedPolicy = policies.find((p) => p.id === policyId);

  const diff = useCallback(async () => {
    if (!policyId || !versionA || !versionB) return;
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/policy/version-diff?id=${encodeURIComponent(policyId)}&a=${encodeURIComponent(versionA)}&b=${encodeURIComponent(versionB)}`, { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, [policyId, versionA, versionB]);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><GitCompare className="w-6 h-6 text-blue-500" /> Policy Version Diff</h1>
        <p className="text-sm text-gray-500 mt-1">Compare policy versions to identify field changes and breaking modifications.</p>
      </div>

      <div className="rounded-lg border dark:border-gray-800 p-4 space-y-3">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
          <div><label className="text-sm font-medium">Policy</label><select value={policyId} onChange={(e) => { setPolicyId(e.target.value); setVersionA(""); setVersionB(""); }} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"><option value="">Select</option>{policies.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}</select></div>
          <div><label className="text-sm font-medium">Version A</label><select value={versionA} onChange={(e) => setVersionA(e.target.value)} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"><option value="">Select</option>{selectedPolicy?.versions.map((v) => <option key={v} value={v}>{v}</option>)}</select></div>
          <div><label className="text-sm font-medium">Version B</label><select value={versionB} onChange={(e) => setVersionB(e.target.value)} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"><option value="">Select</option>{selectedPolicy?.versions.map((v) => <option key={v} value={v}>{v}</option>)}</select></div>
        </div>
        <button onClick={diff} disabled={loading || !policyId || !versionA || !versionB} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50 flex items-center gap-2"><GitCompare className="w-4 h-4" /> {loading ? "Comparing..." : "Compare Versions"}</button>
      </div>

      {data && (
        <>
          {data.breaking_changes.length > 0 && (
            <div className="rounded-lg border border-red-300 dark:border-red-800 bg-red-50 dark:bg-red-900/20 p-4"><div className="flex items-center gap-2 mb-2"><AlertTriangle className="w-5 h-5 text-red-500" /><span className="font-semibold text-red-700 dark:text-red-400">Breaking Changes Detected</span></div><ul className="space-y-1">{data.breaking_changes.map((b, i) => <li key={i} className="text-sm text-red-600">- {b}</li>)}</ul></div>
          )}

          <div className="grid grid-cols-3 gap-4">
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Affected Users</span><p className="text-xl font-bold mt-1">{data.impact_summary.affected_users}</p></div>
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Affected Resources</span><p className="text-xl font-bold mt-1">{data.impact_summary.affected_resources}</p></div>
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Rules Changed</span><p className="text-xl font-bold mt-1">{data.impact_summary.rules_changed}</p></div>
          </div>

          <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
            <table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Type</th><th className="px-4 py-3 text-left font-medium">Field</th><th className="px-4 py-3 text-left font-medium">Old Value</th><th className="px-4 py-3 text-left font-medium">New Value</th></tr></thead>
              <tbody className="divide-y dark:divide-gray-800">{data.field_changes.map((c, i) => { const cfg = changeConfig[c.change_type]; const Icon = cfg.icon; return (<tr key={i} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3"><span className={`flex items-center gap-1 text-xs ${cfg.color}`}><Icon className="w-3.5 h-3.5" /> {c.change_type}</span></td><td className="px-4 py-3 font-mono text-xs font-medium">{c.field}</td><td className="px-4 py-3 text-xs text-gray-500">{c.old_value || "-"}</td><td className="px-4 py-3 text-xs font-medium">{c.new_value || "-"}</td></tr>); })}{data.field_changes.length === 0 && <tr><td colSpan={4} className="px-4 py-8 text-center text-gray-500">No changes.</td></tr>}</tbody>
            </table>
          </div>
        </>
      )}
    </div>
  );
}

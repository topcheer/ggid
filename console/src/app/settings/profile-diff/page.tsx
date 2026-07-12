"use client";

import { useState, useCallback } from "react";
import { GitCompare, ArrowRight, Save, X } from "lucide-react";

interface DiffEntry {
  field: string;
  old_value: string;
  new_value: string;
  changed_by: string;
  changed_at: string;
}

interface DiffResult {
  user_id: string;
  username: string;
  version_a: string;
  version_b: string;
  diffs: DiffEntry[];
}

export default function ProfileDiffPage() {
  const [userId, setUserId] = useState("");
  const [versionA, setVersionA] = useState("");
  const [versionB, setVersionB] = useState("");
  const [result, setResult] = useState<DiffResult | null>(null);
  const [loading, setLoading] = useState(false);

  const diff = useCallback(async () => {
    if (!userId || !versionA || !versionB) return;
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/identity/profile-diff?user_id=${encodeURIComponent(userId)}&a=${encodeURIComponent(versionA)}&b=${encodeURIComponent(versionB)}`, { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setResult(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, [userId, versionA, versionB]);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><GitCompare className="w-6 h-6 text-purple-500" /> Profile Diff</h1>
        <p className="text-sm text-gray-500 mt-1">Compare two versions of a user profile side by side.</p>
      </div>

      <div className="rounded-lg border dark:border-gray-800 p-4 space-y-3">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
          <div><label className="text-sm font-medium">User ID</label><input type="text" value={userId} onChange={(e) => setUserId(e.target.value)} placeholder="usr-xxxx" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono" /></div>
          <div><label className="text-sm font-medium">Version A</label><input type="text" value={versionA} onChange={(e) => setVersionA(e.target.value)} placeholder="v1.0 or timestamp" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono" /></div>
          <div><label className="text-sm font-medium">Version B</label><input type="text" value={versionB} onChange={(e) => setVersionB(e.target.value)} placeholder="v2.0 or timestamp" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono" /></div>
        </div>
        <button onClick={diff} disabled={loading || !userId || !versionA || !versionB} className="px-4 py-2 rounded-lg bg-purple-600 text-white text-sm font-medium hover:bg-purple-700 disabled:opacity-50 flex items-center gap-2"><GitCompare className="w-4 h-4" /> {loading ? "Comparing..." : "Compare"}</button>
      </div>

      {result && (
        <>
          {result.diffs.length === 0 ? (
            <div className="rounded-lg border border-green-300 dark:border-green-800 bg-green-50 dark:bg-green-900/20 p-4 text-sm text-green-700 dark:text-green-400">No differences found between versions.</div>
          ) : (
            <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
              <table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Field</th><th className="px-4 py-3 text-left font-medium">Old Value</th><th className="px-4 py-3 text-left font-medium"></th><th className="px-4 py-3 text-left font-medium">New Value</th><th className="px-4 py-3 text-left font-medium">Changed By</th><th className="px-4 py-3 text-left font-medium">Changed At</th></tr></thead>
                <tbody className="divide-y dark:divide-gray-800">{result.diffs.map((d, i) => (<tr key={i} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3 font-mono text-xs font-medium">{d.field}</td><td className="px-4 py-3 text-xs text-gray-500 line-through">{d.old_value || "-"}</td><td className="px-4 py-3"><ArrowRight className="w-4 h-4 text-gray-400" /></td><td className="px-4 py-3 text-xs font-medium text-purple-600">{d.new_value || "-"}</td><td className="px-4 py-3 text-xs font-mono text-gray-500">{d.changed_by}</td><td className="px-4 py-3 text-xs text-gray-400">{d.changed_at}</td></tr>))}</tbody>
              </table>
            </div>
          )}
        </>
      )}
    </div>
  );
}

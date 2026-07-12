"use client";

import { useState, useEffect, useCallback } from "react";
import { ShieldAlert, Ban, Eye } from "lucide-react";

interface Mismatch {
  id: string;
  token_preview: string;
  expected_audience: string;
  actual_audience: string;
  resource: string;
  blocked: boolean;
  timestamp: string;
}

export default function AudienceMismatchesPage() {
  const [mismatches, setMismatches] = useState<Mismatch[]>([]);
  const [loading, setLoading] = useState(false);
  const [filterBlocked, setFilterBlocked] = useState<string>("");

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/oauth/audience-mismatches", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const data = await res.json(); setMismatches(data.mismatches || data || []); }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const filtered = filterBlocked === "" ? mismatches : filterBlocked === "blocked" ? mismatches.filter((m) => m.blocked) : mismatches.filter((m) => !m.blocked);
  const blockedCount = mismatches.filter((m) => m.blocked).length;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><ShieldAlert className="w-6 h-6 text-orange-500" /> Audience Mismatches</h1>
        <p className="text-sm text-gray-500 mt-1">Track tokens presented with incorrect audience claims.</p>
      </div>

      <div className="grid grid-cols-3 gap-4">
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Total Mismatches</span><p className="text-xl font-bold mt-1">{mismatches.length}</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Blocked</span><p className="text-xl font-bold text-red-600 mt-1">{blockedCount}</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Allowed (Warning)</span><p className="text-xl font-bold text-yellow-600 mt-1">{mismatches.length - blockedCount}</p></div>
      </div>

      <div className="flex items-center gap-2">
        <select value={filterBlocked} onChange={(e) => setFilterBlocked(e.target.value)} className="px-3 py-1.5 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"><option value="">All</option><option value="blocked">Blocked Only</option><option value="allowed">Allowed Only</option></select>
        <span className="text-sm text-gray-500">{filtered.length} entries</span>
      </div>

      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Token</th><th className="px-4 py-3 text-left font-medium">Expected</th><th className="px-4 py-3 text-left font-medium">Actual</th><th className="px-4 py-3 text-left font-medium">Resource</th><th className="px-4 py-3 text-left font-medium">Status</th><th className="px-4 py-3 text-left font-medium">Timestamp</th></tr></thead>
          <tbody className="divide-y dark:divide-gray-800">{filtered.map((m) => (<tr key={m.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3 font-mono text-xs text-gray-500">{m.token_preview}</td><td className="px-4 py-3 font-mono text-xs">{m.expected_audience}</td><td className="px-4 py-3 font-mono text-xs">{m.actual_audience}</td><td className="px-4 py-3 text-xs text-gray-500">{m.resource}</td><td className="px-4 py-3">{m.blocked ? <span className="flex items-center gap-1 text-xs text-red-600"><Ban className="w-3.5 h-3.5" /> Blocked</span> : <span className="flex items-center gap-1 text-xs text-yellow-600"><Eye className="w-3.5 h-3.5" /> Allowed</span>}</td><td className="px-4 py-3 text-xs text-gray-400">{m.timestamp}</td></tr>))}{filtered.length === 0 && !loading && <tr><td colSpan={6} className="px-4 py-8 text-center text-gray-500">No mismatches found.</td></tr>}</tbody>
        </table>
      </div>
    </div>
  );
}

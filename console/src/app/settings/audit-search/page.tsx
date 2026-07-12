"use client";

import { useState, useCallback } from "react";
import { Search, Download, Save, ChevronLeft, ChevronRight } from "lucide-react";

interface AuditEntry {
  id: string;
  timestamp: string;
  user: string;
  action: string;
  resource: string;
  severity: "info" | "warning" | "error" | "critical";
  detail: string;
}

const sevColors: Record<string, string> = {
  info: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-400",
  warning: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  error: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
  critical: "bg-red-200 text-red-900 dark:bg-red-900/50 dark:text-red-300",
};

export default function AuditSearchPage() {
  const [query, setQuery] = useState("");
  const [filters, setFilters] = useState({ user: "", action: "", resource: "", severity: "", start_date: "", end_date: "" });
  const [results, setResults] = useState<AuditEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [savedSearches, setSavedSearches] = useState<string[]>([]);

  const search = useCallback(async (p: number = 1) => {
    setLoading(true);
    try {
      const params = new URLSearchParams({ q: query, page: String(p), ...Object.fromEntries(Object.entries(filters).filter(([, v]) => v)) });
      const res = await fetch("/api/v1/audit/search?" + params, { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const d = await res.json(); setResults(d.entries || d.results || d || []); setTotalPages(d.total_pages || 1); setPage(p); }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, [query, filters]);

  const exportResults = () => { const csv = ["timestamp,user,action,resource,severity,detail", ...results.map((e) => [e.timestamp, e.user, e.action, e.resource, e.severity, JSON.stringify(e.detail)].join(","))].join("\n"); const blob = new Blob([csv], { type: "text/csv" }); const url = URL.createObjectURL(blob); const a = document.createElement("a"); a.href = url; a.download = "audit-export.csv"; a.click(); };

  const saveSearch = () => { if (query) { setSavedSearches([...savedSearches, query]); } };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Search className="w-6 h-6 text-blue-500" /> Audit Search</h1>
        <p className="text-sm text-gray-500 mt-1">Full-text search across audit events with advanced filters.</p>
      </div>

      <div className="rounded-lg border dark:border-gray-800 p-4 space-y-3">
        <div className="flex items-center gap-2"><div className="relative flex-1"><Search className="absolute left-2 top-2.5 w-4 h-4 text-gray-400" /><input type="text" value={query} onChange={(e) => setQuery(e.target.value)} onKeyDown={(e) => { if (e.key === "Enter") search(); }} placeholder="Search audit events..." className="w-full pl-8 pr-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div><button onClick={() => search()} disabled={loading} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50">Search</button></div>
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-2">
          <input type="text" placeholder="User" value={filters.user} onChange={(e) => setFilters({ ...filters, user: e.target.value })} className="px-2 py-1.5 rounded border dark:border-gray-700 dark:bg-gray-900 text-xs" />
          <input type="text" placeholder="Action" value={filters.action} onChange={(e) => setFilters({ ...filters, action: e.target.value })} className="px-2 py-1.5 rounded border dark:border-gray-700 dark:bg-gray-900 text-xs" />
          <input type="text" placeholder="Resource" value={filters.resource} onChange={(e) => setFilters({ ...filters, resource: e.target.value })} className="px-2 py-1.5 rounded border dark:border-gray-700 dark:bg-gray-900 text-xs" />
          <select value={filters.severity} onChange={(e) => setFilters({ ...filters, severity: e.target.value })} className="px-2 py-1.5 rounded border dark:border-gray-700 dark:bg-gray-900 text-xs"><option value="">Severity</option><option value="info">Info</option><option value="warning">Warning</option><option value="error">Error</option><option value="critical">Critical</option></select>
          <input type="date" value={filters.start_date} onChange={(e) => setFilters({ ...filters, start_date: e.target.value })} className="px-2 py-1.5 rounded border dark:border-gray-700 dark:bg-gray-900 text-xs" />
          <input type="date" value={filters.end_date} onChange={(e) => setFilters({ ...filters, end_date: e.target.value })} className="px-2 py-1.5 rounded border dark:border-gray-700 dark:bg-gray-900 text-xs" />
        </div>
        <div className="flex items-center gap-2"><button onClick={exportResults} className="text-xs font-medium text-green-600 hover:underline flex items-center gap-1"><Download className="w-3 h-3" /> Export CSV</button><button onClick={saveSearch} className="text-xs font-medium text-blue-600 hover:underline flex items-center gap-1"><Save className="w-3 h-3" /> Save Search</button>{savedSearches.length > 0 && <span className="text-xs text-gray-500">{savedSearches.length} saved</span>}</div>
      </div>

      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Timestamp</th><th className="px-4 py-3 text-left font-medium">User</th><th className="px-4 py-3 text-left font-medium">Action</th><th className="px-4 py-3 text-left font-medium">Resource</th><th className="px-4 py-3 text-left font-medium">Severity</th><th className="px-4 py-3 text-left font-medium">Detail</th></tr></thead>
          <tbody className="divide-y dark:divide-gray-800">{results.map((e) => (<tr key={e.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3 text-xs text-gray-400">{e.timestamp}</td><td className="px-4 py-3 text-xs font-mono">{e.user}</td><td className="px-4 py-3 text-xs">{e.action}</td><td className="px-4 py-3 text-xs font-mono">{e.resource}</td><td className="px-4 py-3"><span className={"px-2 py-0.5 rounded text-xs " + sevColors[e.severity]}>{e.severity}</span></td><td className="px-4 py-3 text-xs text-gray-500">{e.detail}</td></tr>))}{results.length === 0 && !loading && <tr><td colSpan={6} className="px-4 py-8 text-center text-gray-500">No results. Try searching.</td></tr>}</tbody>
        </table>
      </div>

      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-2"><button onClick={() => search(page - 1)} disabled={page <= 1} className="p-1.5 rounded border dark:border-gray-700 disabled:opacity-50"><ChevronLeft className="w-4 h-4" /></button><span className="text-sm text-gray-500">Page {page} of {totalPages}</span><button onClick={() => search(page + 1)} disabled={page >= totalPages} className="p-1.5 rounded border dark:border-gray-700 disabled:opacity-50"><ChevronRight className="w-4 h-4" /></button></div>
      )}
    </div>
  );
}

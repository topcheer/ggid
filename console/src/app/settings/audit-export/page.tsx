"use client";
import { useState, useEffect, useCallback } from "react";
import { Download, Calendar, Filter, FileDown } from "lucide-react";

interface ExportHistoryItem { id: string; filename: string; size_kb: number; rows: number; format: string; status: "completed" | "processing" | "failed"; created_at: string; }

export default function AuditExportPage() {
  const [format, setFormat] = useState("csv");
  const [startDate, setStartDate] = useState("");
  const [endDate, setEndDate] = useState("");
  const [filterUser, setFilterUser] = useState("");
  const [filterAction, setFilterAction] = useState("");
  const [filterSeverity, setFilterSeverity] = useState("");
  const [destination, setDestination] = useState("download");
  const [scheduled, setScheduled] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [history, setHistory] = useState<ExportHistoryItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadData = useCallback(async () => {
    setLoading(true); setError(null);
    try { const res = await fetch("/api/v1/audit/export/history", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); if (res.ok) { const d = await res.json(); setHistory(d.exports || d.history || []); } }
    catch (err) { setError(err instanceof Error ? err.message : "An error occurred"); } finally { setLoading(false); }
  }, []);
  useEffect(() => { loadData(); }, [loadData]);

  const doExport = async () => {
    setExporting(true);
    try { await fetch("/api/v1/audit/export", { method: "POST", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify({ format, start_date: startDate, end_date: endDate, user: filterUser, action: filterAction, severity: filterSeverity, destination, scheduled }) }); loadData(); }
    catch { /* noop */ }
    finally { setExporting(false); }
  };

  if (loading) return (<div className="p-8 flex items-center justify-center"><div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" /></div>);
  if (error) return (<div className="p-8"><div className="rounded-lg border border-red-300 bg-red-50 dark:bg-red-950 dark:border-red-800 p-4"><p className="text-red-700 dark:text-red-400 text-sm font-medium">Error: {error}</p><button onClick={loadData} className="mt-2 px-4 py-1.5 rounded-lg bg-red-600 text-white text-sm hover:bg-red-700">Retry</button></div></div>);

  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><Download className="w-6 h-6 text-blue-500" /> Audit Export</h1><p className="text-sm text-gray-500 mt-1">Export audit logs with filters in multiple formats.</p></div>

      <div className="rounded-lg border dark:border-gray-800 p-4 space-y-3">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
          <div><label className="text-xs font-medium text-gray-500">Format</label><select value={format} onChange={(e) => setFormat(e.target.value)} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"><option value="csv">CSV</option><option value="json">JSON</option><option value="parquet">Parquet</option></select></div>
          <div><label className="text-xs font-medium text-gray-500">Start Date</label><input type="date" value={startDate} onChange={(e) => setStartDate(e.target.value)} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div>
          <div><label className="text-xs font-medium text-gray-500">End Date</label><input type="date" value={endDate} onChange={(e) => setEndDate(e.target.value)} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div>
          <div><label className="text-xs font-medium text-gray-500">Destination</label><select value={destination} onChange={(e) => setDestination(e.target.value)} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"><option value="download">Download</option><option value="s3">S3</option><option value="https">HTTPS POST</option></select></div>
        </div>
        <div className="flex flex-wrap gap-2 items-center"><Filter className="w-4 h-4 text-gray-400" /><input type="text" value={filterUser} onChange={(e) => setFilterUser(e.target.value)} placeholder="User" className="px-3 py-1.5 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm w-32" /><input type="text" value={filterAction} onChange={(e) => setFilterAction(e.target.value)} placeholder="Action" className="px-3 py-1.5 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm w-32" /><select value={filterSeverity} onChange={(e) => setFilterSeverity(e.target.value)} className="px-3 py-1.5 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"><option value="">All Severity</option><option>low</option><option>medium</option><option>high</option><option>critical</option></select><label className="flex items-center gap-1 text-sm ml-2"><input type="checkbox" checked={scheduled} onChange={(e) => setScheduled(e.target.checked)} className="rounded" /> Scheduled</label></div>
        <button onClick={doExport} disabled={exporting} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium disabled:opacity-50 flex items-center gap-2"><FileDown className="w-4 h-4" /> {exporting ? "Exporting..." : "Export"}</button>
      </div>

      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Filename</th><th className="px-4 py-3 text-left font-medium">Size</th><th className="px-4 py-3 text-left font-medium">Rows</th><th className="px-4 py-3 text-left font-medium">Format</th><th className="px-4 py-3 text-left font-medium">Status</th><th className="px-4 py-3 text-left font-medium">Date</th></tr></thead>
          <tbody className="divide-y dark:divide-gray-800">{history.map((h) => (<tr key={h.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3 font-mono text-xs">{h.filename}</td><td className="px-4 py-3 text-xs text-gray-500">{h.size_kb} KB</td><td className="px-4 py-3 text-xs">{h.rows.toLocaleString()}</td><td className="px-4 py-3"><span className="px-2 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800 uppercase">{h.format}</span></td><td className="px-4 py-3"><span className={"text-xs " + (h.status === "completed" ? "text-green-600" : h.status === "failed" ? "text-red-600" : "text-yellow-600")}>{h.status}</span></td><td className="px-4 py-3 text-xs text-gray-400">{h.created_at}</td></tr>))}{history.length === 0 && <tr><td colSpan={6} className="px-4 py-8 text-center text-gray-500">No exports yet.</td></tr>}</tbody>
        </table>
      </div>
    </div>
  );
}

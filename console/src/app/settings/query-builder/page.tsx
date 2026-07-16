"use client";
import { useState } from "react";
import { Search, Play, Save, Download, Plus, X } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";
interface WhereClause { id: string; field: string; operator: string; value: string; }
interface QueryResult { timestamp: string; user: string; action: string; resource: string; result: string; }
const fields = ["timestamp", "user_id", "action", "resource", "result", "ip_address", "tenant_id"];
const operators = ["=", "!=", "LIKE", "IN", ">", "<", ">=", "<="];
export default function QueryBuilderPage() {
  const t = useTranslations();

  const [selectFields, setSelectFields] = useState<string[]>(["timestamp", "user_id", "action", "resource"]);
  const [whereClauses, setWhereClauses] = useState<WhereClause[]>([{ id: "w1", field: "action", operator: "=", value: "login" }]);
  const [groupBy, setGroupBy] = useState("");
  const [orderBy, setOrderBy] = useState("timestamp");
  const [limit, setLimit] = useState(50);
  const [results, setResults] = useState<QueryResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [history, setHistory] = useState<string[]>([]);
  const addWhere = () => { setWhereClauses([...whereClauses, { id: "w" + Date.now(), field: "action", operator: "=", value: "" }]); };
  const removeWhere = (id: string) => { setWhereClauses(whereClauses.filter((w) => w.id !== id)); };
  const updateWhere = (id: string, field: string, value: string) => { setWhereClauses(whereClauses.map((w) => w.id === id ? { ...w, [field]: value } : w)); };
  const execute = async () => { setLoading(true); try { const res = await fetch("/api/v1/audit/query", { method: "POST", headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify({ select: selectFields, where: whereClauses, group_by: groupBy, order_by: orderBy, limit }) }); if (res.ok) { const d = await res.json(); setResults(d.results || d || []); } setHistory([JSON.stringify({ select: selectFields, where: whereClauses, limit }), ...history].slice(0, 5)); } catch { /* noop */ } finally { setLoading(false); } };
  const exportData = (format: string) => { window.open("/api/v1/audit/query/export?format=" + format, "_blank"); };
  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><Search className="w-6 h-6 text-blue-500" /> {t("queryBuilder.title")}</h1><p className="text-sm text-gray-500 mt-1">Build and execute audit log queries.</p></div>
      <div className="rounded-lg border dark:border-gray-800 p-4 space-y-4">
        <div><label className="text-sm font-medium">SELECT</label><div className="flex flex-wrap gap-1 mt-1">{fields.map((f) => <label key={f} className="flex items-center gap-1 text-xs"><input aria-label="Select fields" type="checkbox" checked={selectFields.includes(f)} onChange={(e) => { if (e.target.checked) setSelectFields([...selectFields, f]); else setSelectFields(selectFields.filter((s) => s !== f)); }} /> {f}</label>)}</div></div>
        <div><label className="text-sm font-medium">WHERE</label><div className="space-y-2 mt-1">{whereClauses.map((w) => (<div key={w.id} className="flex items-center gap-2"><select value={w.field} onChange={(e) => updateWhere(w.id, "field", e.target.value)} className="px-2 py-1.5 rounded border dark:border-gray-700 dark:bg-gray-900 text-xs font-mono">{fields.map((f) => <option key={f} value={f}>{f}</option>)}</select><select value={w.operator} onChange={(e) => updateWhere(w.id, "operator", e.target.value)} className="px-2 py-1.5 rounded border dark:border-gray-700 dark:bg-gray-900 text-xs">{operators.map((o) => <option key={o} value={o}>{o}</option>)}</select><input aria-label="value" type="text" value={w.value} onChange={(e) => updateWhere(w.id, "value", e.target.value)} placeholder="value" className="flex-1 px-2 py-1.5 rounded border dark:border-gray-700 dark:bg-gray-900 text-xs font-mono" /><button onClick={() => removeWhere(w.id)} className="text-red-500"><X className="w-4 h-4" /></button></div>))}<button onClick={addWhere} className="text-xs text-blue-600 flex items-center gap-1"><Plus className="w-3 h-3" /> Add</button></div></div>
        <div className="grid grid-cols-3 gap-3"><div><label className="text-sm font-medium">GROUP BY</label><select value={groupBy} onChange={(e) => setGroupBy(e.target.value)} className="w-full mt-1 px-2 py-1.5 rounded border dark:border-gray-700 dark:bg-gray-900 text-xs"><option value="">None</option>{fields.map((f) => <option key={f} value={f}>{f}</option>)}</select></div><div><label className="text-sm font-medium">ORDER BY</label><select value={orderBy} onChange={(e) => setOrderBy(e.target.value)} className="w-full mt-1 px-2 py-1.5 rounded border dark:border-gray-700 dark:bg-gray-900 text-xs">{fields.map((f) => <option key={f} value={f}>{f}</option>)}</select></div><div><label className="text-sm font-medium">LIMIT</label><input aria-label="group By" type="number" value={limit} onChange={(e) => setLimit(parseInt(e.target.value) || 50)} className="w-full mt-1 px-2 py-1.5 rounded border dark:border-gray-700 dark:bg-gray-900 text-xs" /></div></div>
        <div className="flex gap-2"><button onClick={execute} disabled={loading} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium flex items-center gap-2"><Play className="w-4 h-4" /> Execute</button><button onClick={() => exportData("csv")} className="px-3 py-2 rounded-lg border dark:border-gray-700 text-sm flex items-center gap-1"><Download className="w-4 h-4" /> CSV</button><button onClick={() => exportData("json")} className="px-3 py-2 rounded-lg border dark:border-gray-700 text-sm flex items-center gap-1"><Download className="w-4 h-4" /> JSON</button><button className="px-3 py-2 rounded-lg border dark:border-gray-700 text-sm flex items-center gap-1"><Save className="w-4 h-4" /> Save</button></div>
      </div>
      {history.length > 0 && <div className="rounded-lg border dark:border-gray-800 p-3"><h4 className="text-xs font-semibold text-gray-500 mb-1">Recent Queries</h4><div className="space-y-1">{history.map((h, i) => <div key={i} className="text-xs font-mono text-gray-400 truncate">{h}</div>)}</div></div>}
      {results.length > 0 && <div className="overflow-x-auto rounded-lg border dark:border-gray-800"><table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr>{selectFields.map((f) => <th key={f} className="px-4 py-3 text-left font-medium">{f}</th>)}</tr></thead><tbody className="divide-y dark:divide-gray-800">{results.map((r, i) => (<tr key={i} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3 text-xs">{r.timestamp}</td><td className="px-4 py-3 text-xs font-mono">{r.user}</td><td className="px-4 py-3 text-xs">{r.action}</td><td className="px-4 py-3 text-xs">{r.resource}</td></tr>))}</tbody></table></div>}
    </div>
  );
}

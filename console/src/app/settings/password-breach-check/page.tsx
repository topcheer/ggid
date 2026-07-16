"use client";
import { useTranslations } from "@/lib/i18n";

import { useState, useEffect, useCallback } from "react";
import { ShieldAlert, ScanLine, AlertTriangle, RotateCcw } from "lucide-react";

interface BreachResult {
  id: string;
  username: string;
  breach_count: number;
  breach_sources: string[];
  last_checked: string;
  severity: "clean" | "low" | "medium" | "high" | "critical";
}

const sevColors: Record<string, string> = {
  clean: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400",
  low: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-400",
  medium: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  high: "bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400",
  critical: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
};

export default function PasswordBreachCheckPage() {
  const t = useTranslations();
  const [results, setResults] = useState<BreachResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [scanning, setScanning] = useState(false);
  const [frequency, setFrequency] = useState("weekly");

  const fetchData = useCallback(async () => {
    setLoading(true);
    try { const res = await fetch("/api/v1/auth/password-breach-check", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); if (res.ok) { const d = await res.json(); setResults(d.results || d || []); } }
    catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const scanNow = async () => {
    setScanning(true);
    try { await fetch("/api/v1/auth/password-breach-check/scan", { method: "POST", headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); fetchData(); }
    catch { /* noop */ }
    finally { setScanning(false); }
  };

  const forceReset = async (id: string) => {
    try { await fetch("/api/v1/auth/password-breach-check/" + id + "/force-reset", { method: "POST", headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); }
    catch { /* noop */ }
  };

  const compromised = results.filter((r) => r.severity === "critical" || r.severity === "high").length;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold flex items-center gap-2"><ShieldAlert className="w-6 h-6 text-red-500" />{t("passwordBreachCheck.title")}</h1><p className="text-sm text-gray-500 mt-1">Check passwords against known breach databases (HIBP).</p></div>
        <div className="flex items-center gap-2"><select aria-label="Frequency" value={frequency} onChange={(e) => setFrequency(e.target.value)} className="px-3 py-1.5 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"><option value="daily">Daily</option><option value="weekly">Weekly</option><option value="monthly">Monthly</option></select><button onClick={scanNow} disabled={scanning} className="px-4 py-2 rounded-lg bg-red-600 text-white text-sm font-medium hover:bg-red-700 disabled:opacity-50 flex items-center gap-2"><ScanLine className={"w-4 h-4 " + (scanning ? "animate-spin" : "")} /> {scanning ? "Scanning..." : "Scan Now"}</button></div>
      </div>

      {compromised > 0 && <div className="rounded-lg border border-red-300 dark:border-red-800 bg-red-50 dark:bg-red-900/20 p-4 flex items-center gap-3"><AlertTriangle className="w-8 h-8 text-red-500" /><div><span className="font-semibold text-red-700 dark:text-red-400">{compromised} compromised accounts detected</span><p className="text-sm text-gray-500">Force password reset immediately.</p></div></div>}

      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">User</th><th className="px-4 py-3 text-left font-medium">Breaches</th><th className="px-4 py-3 text-left font-medium">Sources</th><th className="px-4 py-3 text-left font-medium">Last Checked</th><th className="px-4 py-3 text-left font-medium">Severity</th><th className="px-4 py-3 text-left font-medium">Action</th></tr></thead>
          <tbody className="divide-y dark:divide-gray-800">{results.map((r) => (<tr key={r.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3 font-medium">{r.username}</td><td className="px-4 py-3"><span className={"font-bold " + (r.breach_count > 3 ? "text-red-600" : r.breach_count > 0 ? "text-yellow-600" : "text-green-600")}>{r.breach_count}</span></td><td className="px-4 py-3"><div className="flex flex-wrap gap-1">{r.breach_sources.slice(0, 3).map((s, i) => <span key={i} className="px-1.5 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800">{s}</span>)}{r.breach_sources.length > 3 && <span className="text-xs text-gray-400">+{r.breach_sources.length - 3}</span>}</div></td><td className="px-4 py-3 text-xs text-gray-400">{r.last_checked}</td><td className="px-4 py-3"><span className={"px-2 py-0.5 rounded text-xs " + sevColors[r.severity]}>{r.severity}</span></td><td className="px-4 py-3">{(r.severity === "critical" || r.severity === "high") && <button onClick={() => forceReset(r.id)} className="text-xs text-red-600 hover:underline flex items-center gap-1"><RotateCcw className="w-3 h-3" /> Force Reset</button>}</td></tr>))}{results.length === 0 && !loading && <tr><td colSpan={6} className="px-4 py-8 text-center text-gray-500">No results. Click Scan Now.</td></tr>}</tbody>
        </table>
      </div>
    </div>
  );
}

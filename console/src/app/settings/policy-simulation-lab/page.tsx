"use client";
import { useState } from "react";
import { FlaskConical, Play, Save, Check, X, AlertTriangle } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
interface SimResult { subject: string; resource: string; action: string; current_decision: string; proposed_decision: string; changed: boolean; }
export default function PolicySimulationLabPage() {
  const t = useTranslations();

  const [subject, setSubject] = useState("");
  const [resource, setResource] = useState("");
  const [action, setAction] = useState("");
  const [results, setResults] = useState<SimResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const simulate = async () => {
    if (!subject || !resource) return;
    setLoading(true);
    setError(null);
    try {
      const res = await fetch("/api/v1/policy/simulation-lab", { method: "POST", headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify({ subject, resource, action: action || "access" }) });
      if (!res.ok) return null;
      const d = await res.json();
      setResults(d.results || []);
    } catch (e) { setError(e instanceof Error ? e.message : "Failed to run simulation"); }
    finally { setLoading(false); }
  };
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold flex items-center gap-2"><FlaskConical className="w-6 h-6 text-purple-500" /> {t("policySimulationLab.title")}</h1><p className="text-sm text-gray-500 mt-1">Test policy changes in a sandbox with current vs proposed comparison.</p></div>
        <button aria-label="Save scenario" className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm flex items-center gap-2"><Save className="w-4 h-4" /> Save Scenario</button>
      </div>
      {error && <div className="rounded-lg border border-red-200 dark:border-red-900 bg-red-50 dark:bg-red-900/20 p-3 text-sm text-red-600 flex items-center justify-between"><span className="flex items-center gap-2"><AlertTriangle className="w-4 h-4" /> {error}</span><button onClick={() => setError(null)} className="text-xs underline hover:text-red-700">Dismiss</button></div>}
      <div className="rounded-lg border dark:border-gray-800 p-4 space-y-3">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
          <div><label className="text-sm font-medium">Subject</label><input type="text" value={subject} onChange={(e) => setSubject(e.target.value)} placeholder="user:alice" aria-label="Subject" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono" /></div>
          <div><label className="text-sm font-medium">Resource</label><input type="text" value={resource} onChange={(e) => setResource(e.target.value)} placeholder="doc:*" aria-label="Resource" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono" /></div>
          <div><label className="text-sm font-medium">Action</label><select value={action} onChange={(e) => setAction(e.target.value)} aria-label="Action" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"><option value="">access (default)</option><option value="read">read</option><option value="write">write</option><option value="delete">delete</option><option value="admin">admin</option></select></div>
        </div>
        <button onClick={simulate} disabled={loading || !subject || !resource} aria-label="Run simulation" className="px-4 py-2 rounded-lg bg-purple-600 text-white text-sm font-medium hover:bg-purple-700 disabled:opacity-50 flex items-center gap-2"><Play className="w-4 h-4" /> {loading ? "Simulating..." : "Run Simulation"}</button>
      </div>
      {results.length > 0 && (
        <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
          <table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Subject</th><th className="px-4 py-3 text-left font-medium">Resource</th><th className="px-4 py-3 text-left font-medium">Current</th><th className="px-4 py-3 text-left font-medium">Proposed</th><th className="px-4 py-3 text-left font-medium">Changed?</th></tr></thead>
            <tbody className="divide-y dark:divide-gray-800">{results.map((r, i) => (
              <tr key={i} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                <td className="px-4 py-3 font-mono text-xs">{r.subject}</td><td className="px-4 py-3 font-mono text-xs">{r.resource}</td>
                <td className="px-4 py-3"><span className={"text-xs font-bold " + (r.current_decision === "allow" ? "text-green-600" : "text-red-600")}>{r.current_decision}</span></td>
                <td className="px-4 py-3"><span className={"text-xs font-bold " + (r.proposed_decision === "allow" ? "text-green-600" : "text-red-600")}>{r.proposed_decision}</span></td>
                <td className="px-4 py-3">{r.changed ? <span className="text-xs text-orange-600 font-bold">CHANGED</span> : <Check className="w-4 h-4 text-gray-300" />}</td>
              </tr>
            ))}</tbody>
          </table>
        </div>
      )}
    </div>
  );
}

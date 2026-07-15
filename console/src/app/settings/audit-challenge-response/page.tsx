"use client";
import { useState, useEffect, useCallback } from "react";
import { Gavel, Upload, CheckCircle } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface Challenge { id: string; title: string; control: string; severity: string; status: "open" | "responded" | "resolved"; opened_at: string; compliance_impact: string; }
interface ResolvedChallenge extends Challenge { resolved_at: string; resolution: string; }

export default function AuditChallengeResponsePage() {
  const t = useTranslations();

  const [open, setOpen] = useState<Challenge[]>([]);
  const [resolved, setResolved] = useState<ResolvedChallenge[]>([]);
  const [loading, setLoading] = useState(false);
  const [respondId, setRespondId] = useState<string | null>(null);
  const [comment, setComment] = useState("");

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/audit/challenge-response", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const d = await res.json(); setOpen(d.open || []); setResolved(d.resolved || []); }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const respond = async (id: string) => {
    try { await fetch("/api/v1/audit/challenge-response/" + id + "/respond", { method: "POST", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify({ comment }) }); setRespondId(null); setComment(""); fetchData(); }
    catch { /* noop */ }
  };

  const sevColors: Record<string, string> = { critical: "bg-red-100 dark:bg-red-900/30 dark:text-red-400", high: "bg-orange-100 dark:bg-orange-900/30 dark:text-orange-400", medium: "bg-yellow-100 dark:bg-yellow-900/30 dark:text-yellow-400", low: "bg-gray-100 dark:bg-gray-800" };

  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><Gavel className="w-6 h-6 text-orange-500" /> {t("auditChallengeResponse.title")}</h1><p className="text-sm text-gray-500 mt-1">Respond to compliance challenges with evidence and resolution.</p></div>

      <div className="grid grid-cols-3 gap-4"><div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Open</span><p className="text-xl font-bold text-orange-600 mt-1">{open.length}</p></div><div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Resolved</span><p className="text-xl font-bold text-green-600 mt-1">{resolved.length}</p></div><div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Score Impact</span><p className="text-xl font-bold text-red-600 mt-1">-{open.length * 2}pts</p></div></div>

      <div className="space-y-3">{open.map((c) => (<div key={c.id} className="rounded-lg border dark:border-gray-800 p-4"><div className="flex items-center justify-between"><div><span className="font-semibold">{c.title}</span><p className="text-xs text-gray-400">Control: {c.control}</p></div><span className={"px-2 py-0.5 rounded text-xs " + sevColors[c.severity]}>{c.severity}</span></div><p className="text-xs text-gray-500 mt-2">Compliance Impact: {c.compliance_impact}</p>{respondId === c.id ? (<div className="mt-3 space-y-2"><textarea value={comment} onChange={(e) => setComment(e.target.value)} placeholder="Add your response..." className="w-full px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm h-20" /><div className="flex items-center gap-2"><button className="text-xs text-blue-600 flex items-center gap-1"><Upload className="w-3 h-3" /> Upload Evidence</button><button onClick={() => respond(c.id)} disabled={!comment} className="px-3 py-1 rounded-lg bg-blue-600 text-white text-xs font-medium disabled:opacity-50">Submit</button><button onClick={() => setRespondId(null)} className="px-3 py-1 rounded-lg border dark:border-gray-700 text-xs">Cancel</button></div></div>) : (<button onClick={() => setRespondId(c.id)} className="mt-2 text-xs text-blue-600 hover:underline">Respond</button>)}</div>))}{open.length === 0 && !loading && <p className="text-sm text-gray-500 text-center py-4">No open challenges.</p>}</div>

      {resolved.length > 0 && (<div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-2 flex items-center gap-2"><CheckCircle className="w-4 h-4 text-green-500" /> Resolved Challenges</h3><div className="space-y-1">{resolved.map((r) => (<div key={r.id} className="flex items-center justify-between text-sm py-1"><div><span className="font-medium">{r.title}</span><p className="text-xs text-gray-400">{r.resolution}</p></div><span className="text-xs text-gray-400">{r.resolved_at}</span></div>))}</div></div>)}
    </div>
  );
}

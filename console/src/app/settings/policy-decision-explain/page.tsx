"use client";
import { useTranslations } from "@/lib/i18n";
import { useState } from "react";
import { Brain, Play, CheckCircle, XCircle, ChevronRight } from "lucide-react";

interface ExplainResult { decision: "allow" | "deny" | "no_match"; confidence: number; matched_rules: { rule: string; effect: string; priority: number }[]; contributing_factors: string[]; alternatives: { policy: string; decision: string }[]; eval_path: string[]; }

export default function PolicyDecisionExplainPage() {
  const t = useTranslations();
  const [subject, setSubject] = useState("");
  const [resource, setResource] = useState("");
  const [action, setAction] = useState("");
  const [result, setResult] = useState<ExplainResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [expanded, setExpanded] = useState(false);

  const explain = async () => {
    if (!subject || !resource) return;
    setLoading(true);
    try {
      const res = await fetch("/api/v1/policy/decision-explain", { method: "POST", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify({ subject, resource, action: action || "access" }) });
      if (res.ok) setResult(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  };

  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><Brain className="w-6 h-6 text-purple-500" />{t("policyDecisionExplain.title")}</h1><p className="text-sm text-gray-500 mt-1">Understand why a policy decision was made with full evaluation path.</p></div>

      <div className="rounded-lg border dark:border-gray-800 p-4 space-y-3">
        <div className="grid grid-cols-3 gap-3">
          <input aria-label="user:alice" type="text" value={subject} onChange={(e) => setSubject(e.target.value)} placeholder="user:alice" className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono" />
          <input aria-label="doc:project" type="text" value={resource} onChange={(e) => setResource(e.target.value)} placeholder="doc:project" className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono" />
          <input aria-label="access" type="text" value={action} onChange={(e) => setAction(e.target.value)} placeholder="access" className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono" />
        </div>
        <button onClick={explain} disabled={loading || !subject || !resource} className="px-4 py-2 rounded-lg bg-purple-600 text-white text-sm font-medium disabled:opacity-50 flex items-center gap-2"><Play className="w-4 h-4" /> {loading ? "Analyzing..." : "Explain"}</button>
      </div>

      {result && (<>
        <div className={"rounded-lg border-2 p-4 flex items-center gap-3 " + (result.decision === "allow" ? "border-green-300 dark:border-green-800 bg-green-50 dark:bg-green-900/20" : "border-red-300 dark:border-red-800 bg-red-50 dark:bg-red-900/20")}>{result.decision === "allow" ? <CheckCircle className="w-8 h-8 text-green-500" /> : <XCircle className="w-8 h-8 text-red-500" />}<div><span className="text-sm text-gray-500">Decision</span><p className={"text-xl font-bold " + (result.decision === "allow" ? "text-green-600" : "text-red-600")}>{result.decision.toUpperCase()}</p></div><div className="ml-auto text-right"><span className="text-sm text-gray-500">Confidence</span><p className="text-xl font-bold">{result.confidence}%</p></div></div>

        <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">Matched Rules</h3><div className="space-y-2">{result.matched_rules.map((r, i) => (<div key={i} className="flex items-center gap-3 text-sm"><span className={"px-2 py-0.5 rounded text-xs font-medium " + (r.effect === "allow" ? "bg-green-100 dark:bg-green-900/30 dark:text-green-400" : "bg-red-100 dark:bg-red-900/30 dark:text-red-400")}>{r.effect}</span><span className="font-mono text-xs flex-1">{r.rule}</span><span className="text-xs text-gray-400">P{r.priority}</span></div>))}</div></div>

        <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-2">Contributing Factors</h3><div className="flex flex-wrap gap-2">{result.contributing_factors.map((f, i) => (<span key={i} className="px-2 py-1 rounded text-xs bg-gray-100 dark:bg-gray-800">{f}</span>))}</div></div>

        {result.alternatives.length > 0 && (<div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-2">Alternative Policies</h3><div className="space-y-1">{result.alternatives.map((a, i) => (<div key={i} className="flex items-center gap-2 text-sm"><span className="font-mono text-xs">{a.policy}</span><span className={"text-xs " + (a.decision === "allow" ? "text-green-600" : "text-red-600")}>{a.decision}</span></div>))}</div></div>)}

        <div className="rounded-lg border dark:border-gray-800 p-4"><button onClick={() => setExpanded(!expanded)} className="flex items-center gap-2 text-sm font-semibold"><ChevronRight className={"w-4 h-4 transition-transform " + (expanded ? "rotate-90" : "")} /> Evaluation Path</button>{expanded && (<div className="mt-2 space-y-1">{result.eval_path.map((p, i) => (<div key={i} className="text-xs font-mono text-gray-500 pl-6">{i + 1}. {p}</div>))}</div>)}</div>
      </>)}
    </div>
  );
}

"use client";
import { useTranslations } from "@/lib/i18n";
import { useState } from "react";
import { Play, ShieldCheck, ShieldX, MinusCircle, Clock, AlertTriangle } from "lucide-react";
interface DryRunResult { decision: "allow" | "deny" | "no_match"; matched_rules: { rule_id: string; rule_name: string; effect: string }[]; explanation: string; decision_time_ms: number; }
interface Policy { id: string; name: string; }
export default function PolicyDryRunPage() {
  const [policies] = useState<Policy[]>([{ id: "p1", name: "Data Access" }, { id: "p2", name: "Admin Access" }]);
  const [policyId, setPolicyId] = useState("");
  const [subject, setSubject] = useState("");
  const [resource, setResource] = useState("");
  const [action, setAction] = useState("");
  const [result, setResult] = useState<DryRunResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const t = useTranslations();
  const decisionConfig: Record<string, { color: string; icon: typeof ShieldCheck; label: string }> = {
    allow: { color: "text-green-600", icon: ShieldCheck, label: t("policyDryRun.allow") },
    deny: { color: "text-red-600", icon: ShieldX, label: t("policyDryRun.deny") },
    no_match: { color: "text-gray-500", icon: MinusCircle, label: t("policyDryRun.noMatch") },
  };
  const evaluate = async () => {
    if (!policyId || !subject || !resource) return;
    setLoading(true);
    setError(null);
    try {
      const res = await fetch("/api/v1/policy/dry-run", { method: "POST", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify({ policy_id: policyId, subject, resource, action: action || "access" }) });
      if (!res.ok) throw new Error(`Evaluation failed: HTTP ${res.status}`);
      setResult(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Failed to evaluate policy"); }
    finally { setLoading(false); }
  };
  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><Play className="w-6 h-6 text-blue-500" /> {t("policyDryRun.title")}</h1><p className="text-sm text-gray-500 mt-1">{t("policyDryRun.subtitle")}</p></div>
      {error && <div className="rounded-lg border border-red-200 dark:border-red-900 bg-red-50 dark:bg-red-900/20 p-3 text-sm text-red-600 flex items-center justify-between"><span className="flex items-center gap-2"><AlertTriangle className="w-4 h-4" /> {error}</span><button onClick={() => setError(null)} className="text-xs underline hover:text-red-700">{t("policyDryRun.dismiss")}</button></div>}
      <div className="rounded-lg border dark:border-gray-800 p-4 space-y-3">
        <div><label className="text-sm font-medium">{t("policyDryRun.policy")}</label><select value={policyId} onChange={(e) => setPolicyId(e.target.value)} aria-label="Select policy" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"><option value="">{t("policyDryRun.selectPolicy")}</option>{policies.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}</select></div>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
          <div><label className="text-sm font-medium">{t("policyDryRun.subject")}</label><input type="text" value={subject} onChange={(e) => setSubject(e.target.value)} placeholder="user:alice" aria-label="Subject" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono" /></div>
          <div><label className="text-sm font-medium">{t("policyDryRun.resource")}</label><input type="text" value={resource} onChange={(e) => setResource(e.target.value)} placeholder="doc:project-plan" aria-label="Resource" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono" /></div>
          <div><label className="text-sm font-medium">{t("policyDryRun.action")}</label><input type="text" value={action} onChange={(e) => setAction(e.target.value)} placeholder="access" aria-label="Action" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono" /></div>
        </div>
        <button onClick={evaluate} disabled={loading || !policyId || !subject || !resource} aria-label="Evaluate policy" className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50 flex items-center gap-2"><Play className="w-4 h-4" /> {loading ? t("policyDryRun.evaluating") : t("policyDryRun.evaluate")}</button>
      </div>
      {result && (() => {
        const cfg = decisionConfig[result.decision] || decisionConfig.no_match;
        const Icon = cfg.icon;
        return (
          <>
            <div className={"rounded-lg border-2 p-4 flex items-center gap-3 " + (result.decision === "allow" ? "border-green-300 dark:border-green-800 bg-green-50 dark:bg-green-900/20" : result.decision === "deny" ? "border-red-300 dark:border-red-800 bg-red-50 dark:bg-red-900/20" : "border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-900/20")}>
              <Icon className={"w-10 h-10 " + cfg.color} />
              <div><span className="text-sm text-gray-500">{t("policyDryRun.decision")}</span><p className={"text-2xl font-bold " + cfg.color}>{cfg.label}</p><p className="text-xs text-gray-400 flex items-center gap-1 mt-1"><Clock className="w-3 h-3" /> {result.decision_time_ms}ms</p></div>
            </div>
            {result.matched_rules.length > 0 && (
              <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">{t("policyDryRun.matchedRules")}</h3><div className="space-y-2">{result.matched_rules.map((r) => (<div key={r.rule_id} className="flex items-center gap-2"><span className="font-mono text-xs text-gray-500">{r.rule_id}</span><span className="text-sm flex-1">{r.rule_name}</span><span className={"px-2 py-0.5 rounded text-xs " + (r.effect === "allow" ? "bg-green-100 dark:bg-green-900/30 dark:text-green-400" : "bg-red-100 dark:bg-red-900/30 dark:text-red-400")}>{r.effect}</span></div>))}</div></div>
            )}
            <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-2">{t("policyDryRun.explanation")}</h3><p className="text-sm text-gray-600 dark:text-gray-400">{result.explanation}</p></div>
          </>
        );
      })()}
    </div>
  );
}

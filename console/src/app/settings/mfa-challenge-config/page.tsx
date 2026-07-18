"use client";
import { useState, useEffect } from "react";
import { Shield, Save } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";
const methods = ["totp", "sms", "email", "webauthn", "backup"];
const sensitiveActions = ["password_change", "email_change", "role_grant", "data_export", "admin_console", "api_key_create"];
export default function MfaChallengeConfigPage() {
  const t = useTranslations();

  const [priority, setPriority] = useState<string[]>([...methods]);
  const [stepUpActions, setStepUpActions] = useState<string[]>([]);
  const [frequency, setFrequency] = useState("once_per_session");
  const [fallback, setFallback] = useState(true);
  const [graceMin, setGraceMin] = useState(5);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch("/api/v1/auth/mfa/challenge-config", {
      headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
    })
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(data => {
        if (data) {
          if (data.priority) setPriority(data.priority);
          if (data.step_up_actions) setStepUpActions(data.step_up_actions);
          if (data.frequency) setFrequency(data.frequency);
          if (data.fallback !== undefined) setFallback(data.fallback);
          if (data.grace_min) setGraceMin(data.grace_min);
        }
        setLoading(false);
      })
      .catch(e => { setError(e.message); setLoading(false); });
  }, []);

  const move = (idx: number, dir: -1 | 1) => { const n = [...priority]; const t = idx + dir; if (t < 0 || t >= n.length) return; [n[idx], n[t]] = [n[t], n[idx]]; setPriority(n); };
  const toggleAction = (a: string) => setStepUpActions(stepUpActions.includes(a) ? stepUpActions.filter((x) => x !== a) : [...stepUpActions, a]);
  if (loading) return <div className="space-y-6"><p className="text-gray-500">Loading...</p></div>;
  if (error) return <div className="space-y-6 text-red-600">Error: {error}</div>;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between"><div><h1 className="text-2xl font-bold flex items-center gap-2"><Shield className="w-6 h-6 text-blue-500" /> {t("mfaChallengeConfig.title")}</h1><p className="text-sm text-gray-500 mt-1">Configure MFA method priority and step-up authentication.</p></div><button className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium flex items-center gap-2"><Save className="w-4 h-4" /> Save</button></div>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">Method Priority</h3><div className="space-y-2">{priority.map((m: any, i: number) => (<div key={m} className="flex items-center gap-2 p-2 rounded border dark:border-gray-800"><span className="text-xs text-gray-400 w-4">{i + 1}</span><span className="text-sm font-medium capitalize flex-1">{m}</span><button onClick={() => move(i, -1)} disabled={i === 0} className="text-xs px-2 py-0.5 rounded border dark:border-gray-700 disabled:opacity-30">\u2191</button><button onClick={() => move(i, 1)} disabled={i === priority.length - 1} className="text-xs px-2 py-0.5 rounded border dark:border-gray-700 disabled:opacity-30">\u2193</button></div>))}</div></div>
        <div className="space-y-4"><div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">Challenge Frequency</h3><select aria-label="Frequency" value={frequency} onChange={(e) => setFrequency(e.target.value)} className="w-full px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"><option value="always">Every request</option><option value="once_per_session">Once per session</option><option value="threshold_minutes">Threshold (minutes)</option></select>{frequency === "threshold_minutes" && <input type="number" value={graceMin} onChange={(e) => setGraceMin(parseInt(e.target.value))} className="w-24 mt-2 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" />}</div>
          <div className="rounded-lg border dark:border-gray-800 p-4"><label className="flex items-center gap-2 cursor-pointer"><input aria-label="Fallback" type="checkbox" checked={fallback} onChange={(e) => setFallback(e.target.checked)} className="rounded" /><span className="text-sm">Allow fallback method</span></label></div>
        </div>
      </div>
      <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">Require Step-Up For</h3><div className="grid grid-cols-2 md:grid-cols-3 gap-2">{sensitiveActions.map((a) => (<label key={a} className="flex items-center gap-2 cursor-pointer"><input aria-label="Step up actions" type="checkbox" checked={stepUpActions.includes(a)} onChange={() => toggleAction(a)} className="rounded" /><span className="text-sm font-mono">{a}</span></label>))}</div></div>
    </div>
  );
}

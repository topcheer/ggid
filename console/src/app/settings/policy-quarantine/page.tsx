"use client";
import { useTranslations } from "@/lib/i18n";

import { useState, useCallback } from "react";
import { ShieldOff, AlertTriangle, Undo, Clock } from "lucide-react";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface QuarantineResult {
  policy_id: string;
  policy_name: string;
  reason: string;
  duration_hours: number;
  affected_entities: { type: string; id: string; name: string }[];
  rollback_plan: { step: string; reversible: boolean }[];
  auto_reenable_at: string;
  quarantined: boolean;
}

interface Policy { id: string; name: string; }

export default function PolicyQuarantinePage() {
  const t = useTranslations();
  const [policies] = useState<Policy[]>([{ id: "p1", name: "Admin Access" }, { id: "p2", name: "Data Access" }, { id: "p3", name: "External Partner" }]);
  const [policyId, setPolicyId] = useState("");
  const [reason, setReason] = useState("");
  const [duration, setDuration] = useState(24);
  const [data, setData] = useState<QuarantineResult | null>(null);
  const [loading, setLoading] = useState(false);

  const quarantine = useCallback(async () => {
    if (!policyId) return;
    setLoading(true);
    try {
      const res = await fetch("/api/v1/policy/quarantine", { method: "POST", headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify({ policy_id: policyId, reason, duration_hours: duration }) });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, [policyId, reason, duration]);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><ShieldOff className="w-6 h-6 text-red-500" />{t("policyQuarantine.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Temporarily isolate a policy with automatic re-enable and rollback plan.</p>
      </div>

      <div className="rounded-lg border dark:border-gray-800 p-4 space-y-3">
        <div><label className="text-sm font-medium">Policy</label><select aria-label="Policy id" value={policyId} onChange={(e) => setPolicyId(e.target.value)} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"><option value="">Select Policy</option>{policies.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}</select></div>
        <div><label className="text-sm font-medium">Reason</label><input aria-label="Suspected misconfiguration" type="text" value={reason} onChange={(e) => setReason(e.target.value)} placeholder="Suspected misconfiguration" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div>
        <div><label className="text-sm font-medium">Duration: {duration} hours</label><input aria-label="duration" type="range" min={1} max={168} value={duration} onChange={(e) => setDuration(parseInt(e.target.value))} className="w-full mt-2 accent-red-500" /><div className="flex justify-between text-xs text-gray-400 mt-1"><span>1h</span><span>24h</span><span>168h (7d)</span></div></div>
        <button aria-label="action" onClick={quarantine} disabled={loading || !policyId} className="px-4 py-2 rounded-lg bg-red-600 text-white text-sm font-medium hover:bg-red-700 disabled:opacity-50 flex items-center gap-2"><ShieldOff className="w-4 h-4" /> {loading ? "Processing..." : "Quarantine Policy"}</button>
      </div>

      {data && (
        <>
          {data.affected_entities.length > 0 && (
            <div className="rounded-lg border border-orange-200 dark:border-orange-800 bg-orange-50 dark:bg-orange-900/20 p-4 flex items-center gap-2"><AlertTriangle className="w-5 h-5 text-orange-500" /><span className="font-semibold text-orange-700 dark:text-orange-400">{data.affected_entities.length} entities affected by quarantine</span></div>
          )}

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="text-sm font-semibold mb-3">Affected Entities</h3>
              <div className="space-y-1">{data.affected_entities.map((e, i) => (
                <div key={i} className="flex items-center gap-2 text-sm"><span className="px-1.5 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800">{e.type}</span><span className="flex-1">{e.name}</span><span className="text-xs font-mono text-gray-400">{e.id}</span></div>
              ))}{data.affected_entities.length === 0 && <p className="text-xs text-gray-400">No entities affected.</p>}</div>
            </div>

            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="text-sm font-semibold flex items-center gap-2 mb-3"><Undo className="w-4 h-4 text-gray-400" /> Rollback Plan</h3>
              <div className="space-y-2">{data.rollback_plan.map((s, i) => (
                <div key={i} className="flex items-center gap-2 text-sm"><span className="text-xs text-gray-400">{i + 1}.</span><span className="flex-1">{s.step}</span>{s.reversible ? <span className="text-xs text-green-600">reversible</span> : <span className="text-xs text-red-500">irreversible</span>}</div>
              ))}</div>
            </div>
          </div>

          <div className="rounded-lg border dark:border-gray-800 p-4 flex items-center gap-3"><Clock className="w-5 h-5 text-red-500" /><div><span className="text-sm text-gray-500">Auto Re-enable At:</span><span className="font-bold ml-2">{data.auto_reenable_at}</span></div></div>
        </>
      )}
    </div>
  );
}

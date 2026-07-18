"use client";
import { useState, useEffect } from "react";
import {
  KeyRound, Loader2, AlertCircle, X, RotateCw, Check, Clock,
  CheckCircle2, Activity, Shield, Zap, Save,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

type Tab = "active" | "history" | "schedule";

interface ActiveKey { id: string; type: string; key_id: string; created: string; next_rotation: string; days_left: number; status: "healthy" | "due" | "overdue"; }
interface RotationLog { id: string; key_type: string; old_id: string; new_id: string; timestamp: string; success: boolean; duration_ms: number; }

const KEYS: ActiveKey[] = [
  { id: "k1", type: "JWT Signing", key_id: "jwt-a1b2c3", created: "2025-01-08", next_rotation: "2025-01-15", days_left: 0, status: "due" },
  { id: "k2", type: "OAuth Client Secret", key_id: "oauth-x7y8z9", created: "2024-12-01", next_rotation: "2025-03-01", days_left: 45, status: "healthy" },
  { id: "k3", type: "TLS Certificate", key_id: "tls-cert-2025", created: "2025-01-01", next_rotation: "2025-04-01", days_left: 76, status: "healthy" },
  { id: "k4", type: "SCEP CA", key_id: "scep-ca-v3", created: "2024-10-15", next_rotation: "2025-01-10", days_left: -5, status: "overdue" },
];

const HISTORY: RotationLog[] = [
  { id: "h1", key_type: "JWT Signing", old_id: "jwt-d4e5f6", new_id: "jwt-a1b2c3", timestamp: new Date(Date.now() - 7 * 86400000).toISOString(), success: true, duration_ms: 340 },
  { id: "h2", key_type: "OAuth Client Secret", old_id: "oauth-p9q0r1", new_id: "oauth-x7y8z9", timestamp: new Date(Date.now() - 60 * 86400000).toISOString(), success: true, duration_ms: 180 },
  { id: "h3", key_type: "TLS Certificate", old_id: "tls-2024", new_id: "tls-cert-2025", timestamp: new Date(Date.now() - 14 * 86400000).toISOString(), success: true, duration_ms: 2400 },
];

const SCHEDULE = [
  { type: "JWT Signing", interval: 7, unit: "days" },
  { type: "OAuth Client Secret", interval: 90, unit: "days" },
  { type: "TLS Certificate", interval: 90, unit: "days" },
  { type: "SCEP CA", interval: 365, unit: "days" },
];

const STATUS_CFG: Record<string, { color: string; bg: string }> = {
  healthy: { color: "text-green-600", bg: "bg-green-100 dark:bg-green-900/30" },
  due: { color: "text-yellow-600", bg: "bg-yellow-100 dark:bg-yellow-900/30" },
  overdue: { color: "text-red-600", bg: "bg-red-100 dark:bg-red-900/30" },
};

export default function KeyRotationPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("active");
  const [loading, setLoading] = useState(true);
  const [rotating, setRotating] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  useEffect(() => { setLoading(false); }, []);

  const rotate = (id: string) => { setRotating(id); setTimeout(() => setRotating(null), 1500); };
  const saveSchedule = () => { setSaving(true); setTimeout(() => setSaving(false), 800); };

  return (
    <div className="space-y-6">
      <div><h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><KeyRound className="h-6 w-6 text-amber-500" /> {t("keyRotation.title")}</h1><p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("keyRotation.subtitle")}</p></div>

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([["active", `${t("keyRotation.activeKeys")} (${KEYS.length})`, KeyRound], ["history", t("keyRotation.history"), Clock], ["schedule", t("keyRotation.schedule"), Zap]] as const).map(([id, label, Icon]) => (
          <button key={id} onClick={() => setTab(id as Tab)} aria-pressed={tab === id} className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === id ? "border-amber-600 text-amber-600 dark:text-amber-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}><Icon className="h-4 w-4" /> {label}</button>
        ))}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-amber-500" /></div> : (<>

      {/* ACTIVE */}
      {tab === "active" && (
        <div className="space-y-3">{KEYS.map(k => { const cfg = STATUS_CFG[k.status]; return (
          <div key={k.id} className={`${card} flex items-center justify-between !p-3 ${k.status === "overdue" ? "border-red-200 dark:border-red-800" : ""}`}>
            <div className="flex items-center gap-3"><div className={`flex h-9 w-9 items-center justify-center rounded-lg ${cfg.bg}`}><KeyRound className={`h-4 w-4 ${cfg.color}`} /></div><div><div className="flex items-center gap-2"><span className="text-sm font-medium">{k.type}</span><code className="text-xs font-mono text-gray-500">{k.key_id}</code><span className={`px-1.5 py-0.5 rounded text-xs font-medium ${cfg.bg} ${cfg.color}`}>{k.status}</span></div><p className="text-xs text-gray-400">{t("keyRotation.created")} {new Date(k.created).toLocaleDateString()} · {t("keyRotation.next")} {new Date(k.next_rotation).toLocaleDateString()}</p></div></div>
            <div className="flex items-center gap-3"><span className={`text-xs font-bold ${k.days_left < 0 ? "text-red-600" : k.days_left === 0 ? "text-yellow-600" : "text-gray-400"}`}>{k.days_left < 0 ? `${Math.abs(k.days_left)}d ${t("keyRotation.overdue")}` : k.days_left === 0 ? t("keyRotation.dueToday") : `${k.days_left}d`}</span><button onClick={() => rotate(k.id)} disabled={rotating === k.id} aria-label={"Rotate " + k.type} className="flex items-center gap-1 rounded-lg bg-amber-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-amber-700 disabled:opacity-50">{rotating === k.id ? <Loader2 className="h-3 w-3 animate-spin" /> : <RotateCw className="h-3 w-3" />} {t("keyRotation.rotate")}</button></div>
          </div>
        );})}</div>
      )}

      {/* HISTORY */}
      {tab === "history" && (
        <div className="overflow-x-auto"><table className="w-full text-sm">
          <thead className="bg-gray-50 dark:bg-gray-800/50"><tr><th className="px-3 py-2 text-left text-xs text-gray-400">{t("keyRotation.keyType")}</th><th className="px-3 py-2 text-left text-xs text-gray-400">{t("keyRotation.oldKey")}</th><th className="px-3 py-2 text-left text-xs text-gray-400">{t("keyRotation.newKey")}</th><th className="px-3 py-2 text-center text-xs text-gray-400">{t("keyRotation.duration")}</th><th className="px-3 py-2 text-left text-xs text-gray-400">{t("keyRotation.timestamp")}</th><th className="px-3 py-2 text-center text-xs text-gray-400">{t("keyRotation.result")}</th></tr></thead>
          <tbody className="divide-y dark:divide-gray-800">{HISTORY.map(h => (
            <tr key={h.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-3 py-3 text-xs font-medium">{h.key_type}</td><td className="px-3 py-3"><code className="text-xs font-mono text-gray-400">{h.old_id}</code></td><td className="px-3 py-3"><code className="text-xs font-mono text-amber-500">{h.new_id}</code></td><td className="px-3 py-3 text-center text-xs font-mono">{h.duration_ms}ms</td><td className="px-3 py-3 text-xs text-gray-400">{new Date(h.timestamp).toLocaleString()}</td><td className="px-3 py-3 text-center">{h.success ? <CheckCircle2 className="mx-auto h-4 w-4 text-green-500" /> : <X className="mx-auto h-4 w-4 text-red-500" />}</td></tr>
          ))}</tbody>
        </table></div>
      )}

      {/* SCHEDULE */}
      {tab === "schedule" && (
        <div className="space-y-6">
          <div className={card}>
            <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Zap className="h-4 w-4" /> {t("keyRotation.intervals")}</h3>
            <div className="space-y-3">{SCHEDULE.map(s => (
              <div key={s.type} className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700">
                <span className="text-sm font-medium">{s.type}</span>
                <div className="flex items-center gap-2"><input type="number" defaultValue={s.interval} className="w-20 rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-2 py-1 text-sm text-center" /><span className="text-xs text-gray-400">{s.unit}</span></div>
              </div>
            ))}</div>
            <button onClick={saveSchedule} disabled={saving} className="mt-4 flex items-center gap-2 rounded-lg bg-amber-600 px-4 py-2 text-sm font-medium text-white hover:bg-amber-700 disabled:opacity-50">{saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />} {t("keyRotation.saveSchedule")}</button>
          </div>
          <div className={card}>
            <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Shield className="h-4 w-4" /> {t("keyRotation.rotationPolicy")}</h3>
            <div className="space-y-2 text-xs text-gray-500 dark:text-gray-400">
              <p className="flex items-start gap-2"><Check className="h-3.5 w-3.5 text-green-500 mt-0.5" /> {t("keyRotation.policy1")}</p>
              <p className="flex items-start gap-2"><Check className="h-3.5 w-3.5 text-green-500 mt-0.5" /> {t("keyRotation.policy2")}</p>
              <p className="flex items-start gap-2"><Check className="h-3.5 w-3.5 text-green-500 mt-0.5" /> {t("keyRotation.policy3")}</p>
            </div>
          </div>
        </div>
      )}

      </>)}
    </div>
  );
}

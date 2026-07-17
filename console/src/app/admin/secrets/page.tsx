"use client";
import { useState, useEffect } from "react";
import {
  KeyRound, Loader2, AlertCircle, X, RefreshCw, RotateCw, Check,
  CheckCircle2, XCircle, Clock, Lock, Zap, ChevronRight, Activity,
  Shield,
} from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";
import { useTranslations } from "@/lib/i18n";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface SecretRef { name: string; provider: string; path: string; last_rotated: string; interval_days: number; }
interface RotationHistory { id: string; secret: string; timestamp: string; success: boolean; duration_ms: number; }
interface ProviderHealth { name: string; status: string; latency_ms: number; last_check: string; }

type Tab = "references" | "rotation" | "health";

const SECRETS: SecretRef[] = [
  { name: "DATABASE_URL", provider: "vault", path: "secret/data/ggid/db", last_rotated: "2025-01-10T00:00:00Z", interval_days: 90 },
  { name: "REDIS_PASSWORD", provider: "vault", path: "secret/data/ggid/redis", last_rotated: "2024-12-01T00:00:00Z", interval_days: 30 },
  { name: "JWT_SIGNING_KEY", provider: "kms", path: "alias/ggid-jwt", last_rotated: "2025-01-12T00:00:00Z", interval_days: 7 },
  { name: "SMTP_PASSWORD", provider: "env", path: "SMTP_PASSWORD", last_rotated: "2024-11-15T00:00:00Z", interval_days: 180 },
  { name: "S3_ACCESS_KEY", provider: "kms", path: "alias/ggid-s3", last_rotated: "2025-01-08T00:00:00Z", interval_days: 60 },
  { name: "WEBHOOK_SECRET", provider: "env", path: "WEBHOOK_SIGNING_KEY", last_rotated: "2024-10-01T00:00:00Z", interval_days: 365 },
];

const PROVIDERS: ProviderHealth[] = [
  { name: "HashiCorp Vault", status: "healthy", latency_ms: 12, last_check: new Date().toISOString() },
  { name: "AWS KMS", status: "healthy", latency_ms: 45, last_check: new Date().toISOString() },
  { name: "Environment", status: "healthy", latency_ms: 0, last_check: new Date().toISOString() },
];

const ROTATION_HISTORY: RotationHistory[] = [
  { id: "rh-001", secret: "JWT_SIGNING_KEY", timestamp: new Date(Date.now() - 86400000).toISOString(), success: true, duration_ms: 340 },
  { id: "rh-002", secret: "DATABASE_URL", timestamp: new Date(Date.now() - 5 * 86400000).toISOString(), success: true, duration_ms: 1200 },
  { id: "rh-003", secret: "REDIS_PASSWORD", timestamp: new Date(Date.now() - 45 * 86400000).toISOString(), success: true, duration_ms: 180 },
];

const PROVIDER_COLORS: Record<string, string> = { vault: "text-purple-500", kms: "text-orange-500", env: "text-gray-500" };

export default function SecretsPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("references");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [rotating, setRotating] = useState<string | null>(null);

  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  useEffect(() => { setLoading(false); }, []);

  const rotateSecret = async (name: string) => {
    setRotating(name); setTimeout(() => setRotating(null), 1500);
  };

  const daysUntilRotation = (last: string, interval: number) => {
    const elapsed = Math.floor((Date.now() - new Date(last).getTime()) / 86400000);
    return interval - elapsed;
  };

  const upcoming = SECRETS.map(s => ({ ...s, daysLeft: daysUntilRotation(s.last_rotated, s.interval_days) })).sort((a, b) => a.daysLeft - b.daysLeft);

  return (
    <div className="space-y-6">
      <div><h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><KeyRound className="h-6 w-6 text-amber-500" /> {t("secrets.title")}</h1><p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("secrets.subtitle")}</p></div>

      {error && (<div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button></div>)}

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([["references", t("secrets.references"), KeyRound], ["rotation", t("secrets.rotation"), RotateCw], ["health", t("secrets.providerHealth"), Shield]] as const).map(([id, label, Icon]) => (
          <button key={id} onClick={() => setTab(id as Tab)} aria-pressed={tab === id} className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === id ? "border-amber-600 text-amber-600 dark:text-amber-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}><Icon className="h-4 w-4" /> {label}</button>
        ))}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-amber-500" /></div> : (<>

      {/* REFERENCES */}
      {tab === "references" && (
        <div className="overflow-x-auto"><table className="w-full text-sm">
          <thead className="bg-gray-50 dark:bg-gray-800/50"><tr><th className="px-3 py-2 text-left text-xs text-gray-400">{t("secrets.name")}</th><th className="px-3 py-2 text-center text-xs text-gray-400">{t("secrets.provider")}</th><th className="px-3 py-2 text-left text-xs text-gray-400">{t("secrets.path")}</th><th className="px-3 py-2 text-center text-xs text-gray-400">{t("secrets.lastRotated")}</th><th className="px-3 py-2 text-center text-xs text-gray-400">{t("secrets.interval")}</th><th className="px-3 py-2 text-right text-xs text-gray-400">{t("secrets.actions")}</th></tr></thead>
          <tbody className="divide-y dark:divide-gray-800">{SECRETS.map(s => {
            const days = daysUntilRotation(s.last_rotated, s.interval_days);
            return (
              <tr key={s.name} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                <td className="px-3 py-3"><code className="text-xs font-mono text-amber-500">{s.name}</code></td>
                <td className="px-3 py-3 text-center"><span className={`px-1.5 py-0.5 rounded text-xs font-mono bg-gray-100 dark:bg-gray-700 ${PROVIDER_COLORS[s.provider]}`}>{s.provider}</span></td>
                <td className="px-3 py-3"><code className="text-xs font-mono text-gray-500 truncate block max-w-xs">{s.path}</code></td>
                <td className="px-3 py-3 text-center text-xs">{new Date(s.last_rotated).toLocaleDateString()}</td>
                <td className="px-3 py-3 text-center"><span className={`text-xs font-mono ${days < 0 ? "text-red-600 font-bold" : days < 7 ? "text-orange-600" : "text-gray-400"}`}>{days < 0 ? `${Math.abs(days)}d overdue` : `${days}d left`}</span></td>
                <td className="px-3 py-3 text-right"><button onClick={() => rotateSecret(s.name)} disabled={rotating === s.name} aria-label={"Rotate " + s.name} className="rounded-lg p-1.5 text-amber-500 hover:bg-amber-50 dark:hover:bg-amber-900/20">{rotating === s.name ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <RotateCw className="h-3.5 w-3.5" />}</button></td>
              </tr>
            );
          })}</tbody>
        </table></div>
      )}

      {/* ROTATION */}
      {tab === "rotation" && (
        <div className="space-y-6">
          <div className={card}>
            <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Clock className="h-4 w-4" /> {t("secrets.upcomingRotations")}</h3>
            <div className="space-y-2">{upcoming.map(s => (
              <div key={s.name} className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700">
                <div className="flex items-center gap-3"><div className={`flex h-8 w-8 items-center justify-center rounded-lg ${s.daysLeft < 0 ? "bg-red-100 dark:bg-red-900/30" : s.daysLeft < 7 ? "bg-orange-100 dark:bg-orange-900/30" : "bg-green-100 dark:bg-green-900/30"}`}><Lock className={`h-4 w-4 ${s.daysLeft < 0 ? "text-red-500" : s.daysLeft < 7 ? "text-orange-500" : "text-green-500"}`} /></div><div><span className="text-sm font-medium">{s.name}</span><p className="text-xs text-gray-400">{t("secrets.interval")}: {s.interval_days}d · {s.provider}</p></div></div>
                <div className="text-right"><span className={`text-sm font-bold ${s.daysLeft < 0 ? "text-red-600" : s.daysLeft < 7 ? "text-orange-600" : "text-gray-500"}`}>{s.daysLeft < 0 ? `${Math.abs(s.daysLeft)}d overdue` : `${s.daysLeft}d`}</span><button onClick={() => rotateSecret(s.name)} disabled={rotating === s.name} className="ml-3 rounded-lg bg-amber-600 px-2 py-1 text-xs font-medium text-white hover:bg-amber-700">{rotating === s.name ? "..." : t("secrets.rotateNow")}</button></div>
              </div>
            ))}</div>
          </div>
          <div className={card}>
            <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Activity className="h-4 w-4" /> {t("secrets.rotationHistory")}</h3>
            <div className="space-y-2">{ROTATION_HISTORY.map(r => (
              <div key={r.id} className="flex items-center justify-between rounded-lg border p-2 dark:border-gray-700">
                <div className="flex items-center gap-3">{r.success ? <CheckCircle2 className="h-4 w-4 text-green-500" /> : <XCircle className="h-4 w-4 text-red-500" />}<div><code className="text-xs font-mono">{r.secret}</code><p className="text-xs text-gray-400">{new Date(r.timestamp).toLocaleString()} · {r.duration_ms}ms</p></div></div>
                <span className={`px-1.5 py-0.5 rounded text-xs ${r.success ? "bg-green-100 dark:bg-green-900/30 text-green-600" : "bg-red-100 dark:bg-red-900/30 text-red-600"}`}>{r.success ? "success" : "failed"}</span>
              </div>
            ))}</div>
          </div>
        </div>
      )}

      {/* HEALTH */}
      {tab === "health" && (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">{PROVIDERS.map(p => (
          <div key={p.name} className={card + " hover:shadow-md transition"}>
            <div className="flex items-center justify-between mb-3"><div className="flex h-10 w-10 items-center justify-center rounded-lg bg-amber-100 dark:bg-amber-900/30"><Shield className="h-5 w-5 text-amber-500" /></div><span className="flex items-center gap-1 px-2 py-0.5 rounded text-xs bg-green-100 dark:bg-green-900/30 text-green-600"><CheckCircle2 className="h-3 w-3" /> {p.status}</span></div>
            <h3 className="font-semibold text-sm">{p.name}</h3>
            <div className="mt-2 space-y-1 text-xs text-gray-400"><p>{t("secrets.latency")}: <span className="font-mono">{p.latency_ms}ms</span></p><p>{t("secrets.lastCheck")}: {new Date(p.last_check).toLocaleTimeString()}</p></div>
          </div>
        ))}</div>
      )}

      </>)}
    </div>
  );
}

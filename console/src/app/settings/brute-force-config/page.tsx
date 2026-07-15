"use client";

import { useState, useEffect, useCallback } from "react";
import { Lock, Save, Shield } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface EndpointOverride {
  endpoint: string;
  max_attempts: number;
  lockout_minutes: number;
}

interface Lockout {
  ip: string;
  username: string;
  locked_at: string;
  unlock_at: string;
  minutes_remaining: number;
}

interface Config {
  max_attempts: number;
  lockout_duration_minutes: number;
  progressive_delay: boolean;
  captcha_threshold: number;
  ip_allowlist: string[];
  endpoint_overrides: EndpointOverride[];
}

export default function BruteForceConfigPage() {
  const t = useTranslations();
  const [config, setConfig] = useState<Config>({ max_attempts: 5, lockout_duration_minutes: 15, progressive_delay: true, captcha_threshold: 3, ip_allowlist: ["127.0.0.1"], endpoint_overrides: [{ endpoint: "/api/v1/auth/login", max_attempts: 5, lockout_minutes: 15 }, { endpoint: "/api/v1/oauth/token", max_attempts: 10, lockout_minutes: 30 }] });
  const [lockouts, setLockouts] = useState<Lockout[]>([]);
  const [saving, setSaving] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch("/api/v1/auth/brute-force-config/lockouts", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (!res.ok) return null;
      const d = await res.json(); setLockouts(d.lockouts || []);
    } catch (err) { setError(err instanceof Error ? err.message : t("bruteForce.anError")); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const save = useCallback(async () => {
    setSaving(true);
    try { await fetch("/api/v1/auth/brute-force-config", { method: "PUT", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify(config) }); }
    catch { /* noop */ }
    finally { setSaving(false); }
  }, [config]);

  if (loading) return (<div className="p-8 flex items-center justify-center"><div className="animate-spin rounded-full h-8 w-8 border-b-2 border-red-600" /></div>);
  if (error) return (<div className="p-8"><div className="rounded-lg border border-red-300 bg-red-50 dark:bg-red-950 dark:border-red-800 p-4"><p className="text-red-700 dark:text-red-400 text-sm font-medium">Error: {error}</p><button onClick={loadData} className="mt-2 px-4 py-1.5 rounded-lg bg-red-600 text-white text-sm hover:bg-red-700">{t("common.refresh")}</button></div></div>);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold flex items-center gap-2"><Lock className="w-6 h-6 text-red-500" /> {t("bruteForce.title")}</h1><p className="text-sm text-gray-500 mt-1">{t("bruteForce.subtitle")}</p></div>
        <button onClick={save} disabled={saving} className="px-4 py-2 rounded-lg bg-red-600 text-white text-sm font-medium hover:bg-red-700 disabled:opacity-50 flex items-center gap-2"><Save className="w-4 h-4" /> {t("common.save")}</button>
      </div>

      <div className="rounded-lg border dark:border-gray-800 p-4 space-y-4 max-w-lg">
        <div><label className="text-sm font-medium">{t("bruteForce.maxAttempts")}</label><div className="flex items-center gap-3 mt-1"><input type="range" min={3} max={20} value={config.max_attempts} onChange={(e) => setConfig({ ...config, max_attempts: parseInt(e.target.value) })} className="flex-1" /><span className="font-bold text-sm w-10">{config.max_attempts}</span></div></div>
        <div><label className="text-sm font-medium">{t("bruteForce.lockoutDuration")}</label><div className="flex items-center gap-3 mt-1"><input type="range" min={5} max={120} value={config.lockout_duration_minutes} onChange={(e) => setConfig({ ...config, lockout_duration_minutes: parseInt(e.target.value) })} className="flex-1" /><span className="font-bold text-sm w-12">{config.lockout_duration_minutes}m</span></div></div>
        <div><label className="text-sm font-medium">{t("bruteForce.captchaThreshold")}</label><input type="number" min={1} max={10} value={config.captcha_threshold} onChange={(e) => setConfig({ ...config, captcha_threshold: parseInt(e.target.value) })} className="w-20 mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div>
        <label className="flex items-center gap-2 cursor-pointer"><input type="checkbox" checked={config.progressive_delay} onChange={(e) => setConfig({ ...config, progressive_delay: e.target.checked })} className="rounded" /><span className="text-sm">{t("bruteForce.progressiveDelay")}</span></label>
      </div>

      <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3 flex items-center gap-2"><Shield className="w-4 h-4 text-gray-400" /> {t("bruteForce.perEndpoint")}</h3><table className="w-full text-sm"><thead><tr><th className="px-4 py-2 text-left font-medium">{t("bruteForce.endpoint")}</th><th className="px-4 py-2 text-left font-medium">{t("bruteForce.maxAttempts")}</th><th className="px-4 py-2 text-left font-medium">{t("bruteForce.lockoutMin")}</th></tr></thead><tbody className="divide-y dark:divide-gray-800">{config.endpoint_overrides.map((e, i) => (<tr key={i}><td className="px-4 py-2 font-mono text-xs">{e.endpoint}</td><td className="px-4 py-2"><input type="number" value={e.max_attempts} onChange={(ev) => { const o = [...config.endpoint_overrides]; o[i] = { ...e, max_attempts: parseInt(ev.target.value) }; setConfig({ ...config, endpoint_overrides: o }); }} className="w-16 px-2 py-1 rounded border dark:border-gray-700 dark:bg-gray-900 text-sm" /></td><td className="px-4 py-2"><input type="number" value={e.lockout_minutes} onChange={(ev) => { const o = [...config.endpoint_overrides]; o[i] = { ...e, lockout_minutes: parseInt(ev.target.value) }; setConfig({ ...config, endpoint_overrides: o }); }} className="w-16 px-2 py-1 rounded border dark:border-gray-700 dark:bg-gray-900 text-sm" /></td></tr>))}</tbody></table></div>

      {lockouts.length > 0 && <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3 text-red-600">{t("bruteForce.currentLockouts")} ({lockouts.length})</h3><div className="space-y-1">{lockouts.map((l, i) => (<div key={i} className="flex items-center gap-2 text-sm"><span className="font-mono text-xs">{l.ip}</span><span className="text-xs">{l.username}</span><span className="text-xs text-gray-400 ml-auto">unlock in {l.minutes_remaining}m</span></div>))}</div></div>}
    </div>
  );
}

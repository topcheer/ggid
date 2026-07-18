"use client";
import { useTranslations } from "@/lib/i18n";

import React, { useEffect, useState } from "react";
import { useApi } from "@/lib/api";
import {
  Smartphone, Loader2, AlertCircle, X, ShieldCheck, Save, Lock, Unlock, TrendingUp,
} from "lucide-react";

interface DeviceTrustEntry {
  device_id: string;
  user_id: string;
  username: string;
  platform: string;
  os_version: string;
  managed: boolean;
  encrypted: boolean;
  jailbroken: boolean;
  last_seen: string;
  trust_score: number;
  enrolled_at: string;
}

interface PosturePolicy {
  min_os_version: Record<string, string>;
  require_encryption: boolean;
  block_jailbreak: boolean;
  require_managed: boolean;
  min_trust_score: number;
}

function trustColor(score: number): string {
  if (score >= 75) return "text-green-600";
  if (score >= 50) return "text-yellow-600";
  return "text-red-600";
}

export default function DeviceTrustPage() {
  const t = useTranslations();
  const { apiFetch } = useApi();
  const [devices, setDevices] = useState<DeviceTrustEntry[]>([]);
  const [config, setConfig] = useState<PosturePolicy | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    (async () => {
      try {
        const [d, c] = await Promise.all([
          apiFetch<DeviceTrustEntry[]>("/api/v1/auth/devices/trust").catch(() => []),
          apiFetch<PosturePolicy>("/api/v1/auth/devices/posture/config").catch(() => null),
        ]);
        setDevices(d); setConfig(c);
      } catch { setError("Failed to load device trust data"); }
      finally { setLoading(false); }
    })();
  }, []);

  const handleSaveConfig = async () => {
    if (!config) return;
    setSaving(true);
    try { await apiFetch("/api/v1/auth/devices/posture/config", { method: "PUT", body: JSON.stringify(config) }); }
    catch { setError("Save failed"); }
    finally { setSaving(false); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const untrusted = devices.filter((d: any) => d.trust_score < 50 || d.jailbroken || !d.encrypted);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Smartphone className="h-6 w-6 text-cyan-600" /> {t("backend.deviceTrust.title")}</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Device inventory with trust scoring and posture policy enforcement.</p>
      </div>

      {untrusted.length > 0 && (
        <div className="flex items-center gap-3 rounded-xl border border-orange-200 bg-orange-50 px-4 py-3 dark:border-orange-800 dark:bg-orange-900/20"><AlertCircle className="h-5 w-5 text-orange-600 shrink-0" /><span className="text-sm text-orange-700 dark:text-orange-400">{untrusted.length} device{untrusted.length > 1 ? "s" : ""} failing posture requirements.</span></div>
      )}

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-cyan-600" /></div>
      : (
        <>
          {/* Posture config */}
          {config && (
            <div className={cardCls}>
              <h3 className="mb-4 text-sm font-semibold text-gray-700 dark:text-gray-300">Posture Policy</h3>
              <div className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <button onClick={() => setConfig({ ...config, require_encryption: !config.require_encryption })} className={`flex items-center justify-between rounded-lg border px-4 py-3 text-sm ${config.require_encryption ? "border-green-300 bg-green-50 dark:border-green-800 dark:bg-green-900/20" : "border-gray-300 dark:border-gray-600"}`}><span className="flex items-center gap-2">{config.require_encryption ? <Lock className="h-4 w-4 text-green-600" /> : <Unlock className="h-4 w-4 text-gray-400" />}Require Encryption</span><span className={`text-xs font-medium ${config.require_encryption ? "text-green-600" : "text-gray-400"}`}>{config.require_encryption ? "On" : "Off"}</span></button>
                  <button onClick={() => setConfig({ ...config, block_jailbreak: !config.block_jailbreak })} className={`flex items-center justify-between rounded-lg border px-4 py-3 text-sm ${config.block_jailbreak ? "border-red-300 bg-red-50 dark:border-red-800 dark:bg-red-900/20" : "border-gray-300 dark:border-gray-600"}`}><span className="flex items-center gap-2"><ShieldCheck className="h-4 w-4" />{t("backend.deviceTrust.blockJailbroken")}</span><span className={`text-xs font-medium ${config.block_jailbreak ? "text-red-600" : "text-gray-400"}`}>{config.block_jailbreak ? "On" : "Off"}</span></button>
                  <button onClick={() => setConfig({ ...config, require_managed: !config.require_managed })} className={`flex items-center justify-between rounded-lg border px-4 py-3 text-sm ${config.require_managed ? "border-blue-300 bg-blue-50 dark:border-blue-800 dark:bg-blue-900/20" : "border-gray-300 dark:border-gray-600"}`}><span className="flex items-center gap-2"><ShieldCheck className="h-4 w-4" />Require MDM</span><span className={`text-xs font-medium ${config.require_managed ? "text-blue-600" : "text-gray-400"}`}>{config.require_managed ? "On" : "Off"}</span></button>
                  <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">{t("backend.deviceTrust.minTrustScore")}</label><input aria-label="config" type="number" value={config.min_trust_score} onChange={(e) => setConfig({ ...config, min_trust_score: parseInt(e.target.value) || 0 })} min={0} max={100} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
                </div>
                <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">{t("backend.deviceTrust.minOsVersions")}</label><div className="flex flex-wrap gap-2">{Object.entries(config.min_os_version).map(([k, v]: any[]) => <span key={k} className="rounded bg-gray-100 px-2 py-1 text-xs text-gray-600 dark:bg-gray-700 dark:text-gray-300">{k}: {v}</span>)}{Object.keys(config.min_os_version).length === 0 && <span className="text-xs text-gray-400">No minimums set</span>}</div></div>
                <button onClick={handleSaveConfig} disabled={saving} className="flex items-center gap-2 rounded-lg bg-cyan-600 px-4 py-2 text-sm font-medium text-white hover:bg-cyan-700 disabled:opacity-50">{saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}Save Policy</button>
              </div>
            </div>
          )}

          {/* Device table */}
          <div>
            <h2 className="mb-3 text-sm font-semibold uppercase text-gray-500">{t("backend.deviceTrust.deviceInventory")}</h2>
            {devices.length === 0 ? (
              <div className={cardCls}><div className="py-12 text-center"><Smartphone className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No enrolled devices.</p></div></div>
            ) : (
              <div className="overflow-x-auto rounded-xl border border-gray-200 dark:border-gray-700">
                <table className="w-full text-sm">
                  <thead className="bg-gray-50 dark:bg-gray-800"><tr>
                    <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">{t("backend.deviceTrust.device")}</th>
                    <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">User</th>
                    <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Platform</th>
                    <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">{t("backend.deviceTrust.flags")}</th>
                    <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Trust</th>
                    <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">{t("backend.deviceTrust.lastSeen")}</th>
                  </tr></thead>
                  <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                    {devices.map((d: any) => (
                      <tr key={d.device_id} className="bg-white dark:bg-gray-900">
                        <td className="px-4 py-3 font-mono text-xs text-gray-500">{d.device_id.slice(0, 16)}</td>
                        <td className="px-4 py-3 font-medium text-gray-900 dark:text-white">{d.username}</td>
                        <td className="px-4 py-3"><div className="text-gray-700 dark:text-gray-300">{d.platform}</div><div className="text-xs text-gray-400">{d.os_version}</div></td>
                        <td className="px-4 py-3"><div className="flex flex-wrap gap-1">
                          {d.managed && <span className="rounded bg-blue-100 px-1.5 py-0.5 text-xs text-blue-600 dark:bg-blue-900/30">{t("backend.deviceTrust.mdm")}</span>}
                          {d.encrypted ? <span className="rounded bg-green-100 px-1.5 py-0.5 text-xs text-green-600 dark:bg-green-900/30">{t("backend.deviceTrust.encrypted")}</span> : <span className="rounded bg-red-100 px-1.5 py-0.5 text-xs text-red-600 dark:bg-red-900/30">Unencrypted</span>}
                          {d.jailbroken && <span className="rounded bg-red-100 px-1.5 py-0.5 text-xs text-red-600 dark:bg-red-900/30">{t("backend.deviceTrust.jailbroken")}</span>}
                        </div></td>
                        <td className="px-4 py-3"><span className={`flex items-center gap-1 text-lg font-bold ${trustColor(d.trust_score)}`}><TrendingUp className="h-3 w-3" />{d.trust_score}</span></td>
                        <td className="px-4 py-3 text-gray-400">{d.last_seen ? new Date(d.last_seen).toLocaleDateString() : "—"}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </>
      )}
    </div>
  );
}

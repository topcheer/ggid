"use client";
import { useState, useCallback, useEffect } from "react";
import {
  Smartphone, Loader2, AlertCircle, X, RefreshCw, Shield, Check,
  CheckCircle2, XCircle, AlertTriangle, ChevronRight, Lock,
  Activity, Cpu, Settings as SettingsIcon, Gauge,
} from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";
import { useTranslations } from "@/lib/i18n";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface Device { id: string; user_id: string; os: string; browser: string; posture_score: number; compliant: boolean; last_seen: string; mdm_enrolled: boolean; hostname: string; }

type Tab = "devices" | "compliance" | "policies";

const SAMPLE_DEVICES: Device[] = [
  { id: "d1", user_id: "user:alice", os: "macOS 14.2", browser: "Chrome 121", posture_score: 92, compliant: true, last_seen: new Date(Date.now() - 300000).toISOString(), mdm_enrolled: true, hostname: "ALICE-MBP" },
  { id: "d2", user_id: "user:bob", os: "Windows 11", browser: "Edge 121", posture_score: 78, compliant: true, last_seen: new Date(Date.now() - 3600000).toISOString(), mdm_enrolled: true, hostname: "BOB-DESKTOP" },
  { id: "d3", user_id: "user:carol", os: "iOS 17.3", browser: "Safari Mobile", posture_score: 45, compliant: false, last_seen: new Date(Date.now() - 86400000).toISOString(), mdm_enrolled: false, hostname: "Carol-iPhone" },
  { id: "d4", user_id: "user:dave", os: "Android 14", browser: "Chrome Mobile", posture_score: 62, compliant: true, last_seen: new Date(Date.now() - 7200000).toISOString(), mdm_enrolled: false, hostname: "DAVE-PHONE" },
  { id: "d5", user_id: "user:eve", os: "Linux 6.5", browser: "Firefox 122", posture_score: 88, compliant: true, last_seen: new Date(Date.now() - 600000).toISOString(), mdm_enrolled: true, hostname: "eve-workstation" },
  { id: "d6", user_id: "user:frank", os: "macOS 13.6", browser: "Safari 17", posture_score: 35, compliant: false, last_seen: new Date(Date.now() - 172800000).toISOString(), mdm_enrolled: false, hostname: "Frank-Mac" },
];

const FAILURE_REASONS = [
  { reason: "OS out of date", count: 3, pct: 50 },
  { reason: "No disk encryption", count: 1, pct: 17 },
  { reason: "Jailbreak detected", count: 1, pct: 17 },
  { reason: "No MDM enrollment", count: 1, pct: 16 },
];

export default function DeviceFleetPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("devices");
  const [devices, setDevices] = useState<Device[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState("");
  const [minScore, setMinScore] = useState(60);

  const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/auth/devices/list", { headers: h }).catch(() => null);
      if (res?.ok) { const d = await res.json(); if (d.devices?.length) setDevices(d.devices); else setDevices(SAMPLE_DEVICES); }
      else setDevices(SAMPLE_DEVICES);
    } catch { setDevices(SAMPLE_DEVICES); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const filtered = search ? devices.filter(d => d.user_id.includes(search) || d.hostname.toLowerCase().includes(search.toLowerCase())) : devices;
  const compliant = devices.filter(d => d.compliant).length;
  const nonCompliant = devices.filter(d => !d.compliant).length;
  const mdmEnrolled = devices.filter(d => d.mdm_enrolled).length;
  const compliancePct = devices.length > 0 ? Math.round((compliant / devices.length) * 100) : 0;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Smartphone className="h-6 w-6 text-blue-500" /> {t("deviceFleet.title")}</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("deviceFleet.subtitle")}</p>
      </div>

      {error && (
        <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "devices" as Tab, label: `${t("deviceFleet.devices")} (${devices.length})`, icon: Smartphone },
          { id: "compliance" as Tab, label: t("deviceFleet.compliance"), icon: Shield },
          { id: "policies" as Tab, label: t("deviceFleet.policies"), icon: SettingsIcon },
        ]).map(tb => {
          const Icon = tb.icon;
          return (
            <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id}
              className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === tb.id ? "border-blue-600 text-blue-600 dark:text-blue-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}>
              <Icon className="h-4 w-4" /> {tb.label}
            </button>
          );
        })}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-blue-500" /></div> : (<>

      {/* DEVICES */}
      {tab === "devices" && (
        <div>
          <div className="mb-4"><div className="relative max-w-xs"><input type="text" value={search} onChange={e => setSearch(e.target.value)} placeholder={t("deviceFleet.search")} className="w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-1.5 text-sm" /></div></div>
          <div className="overflow-x-auto"><table className="w-full text-sm">
            <thead className="bg-gray-50 dark:bg-gray-800/50"><tr>
              <th className="px-3 py-2 text-left text-xs text-gray-400">{t("deviceFleet.user")}</th><th className="px-3 py-2 text-left text-xs text-gray-400">{t("deviceFleet.hostname")}</th><th className="px-3 py-2 text-left text-xs text-gray-400">{t("deviceFleet.os")}</th>
              <th className="px-3 py-2 text-center text-xs text-gray-400">{t("deviceFleet.posture")}</th><th className="px-3 py-2 text-center text-xs text-gray-400">{t("deviceFleet.compliant")}</th><th className="px-3 py-2 text-center text-xs text-gray-400">{t("deviceFleet.mdm")}</th><th className="px-3 py-2 text-left text-xs text-gray-400">{t("deviceFleet.lastSeen")}</th>
            </tr></thead>
            <tbody className="divide-y dark:divide-gray-800">
              {filtered.map(d => (
                <tr key={d.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                  <td className="px-3 py-3 text-xs font-mono">{d.user_id}</td>
                  <td className="px-3 py-3 text-xs">{d.hostname}</td>
                  <td className="px-3 py-3 text-xs">{d.os}</td>
                  <td className="px-3 py-3 text-center"><span className={`text-xs font-bold ${d.posture_score >= 80 ? "text-green-600" : d.posture_score >= 60 ? "text-yellow-600" : "text-red-600"}`}>{d.posture_score}</span></td>
                  <td className="px-3 py-3 text-center">{d.compliant ? <CheckCircle2 className="mx-auto h-4 w-4 text-green-500" /> : <XCircle className="mx-auto h-4 w-4 text-red-500" />}</td>
                  <td className="px-3 py-3 text-center">{d.mdm_enrolled ? <Check className="mx-auto h-3.5 w-3.5 text-green-500" /> : <span className="text-xs text-gray-300">—</span>}</td>
                  <td className="px-3 py-3 text-xs text-gray-400">{new Date(d.last_seen).toLocaleDateString()}</td>
                </tr>
              ))}
            </tbody>
          </table></div>
        </div>
      )}

      {/* COMPLIANCE */}
      {tab === "compliance" && (
        <div className="space-y-6">
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
            <div className={card + " text-center"}><Smartphone className="mx-auto h-5 w-5 text-blue-400" /><p className="mt-2 text-2xl font-bold">{devices.length}</p><p className="text-xs text-gray-400">{t("deviceFleet.totalDevices")}</p></div>
            <div className={card + " text-center"}><CheckCircle2 className="mx-auto h-5 w-5 text-green-400" /><p className="mt-2 text-2xl font-bold text-green-600">{compliant}</p><p className="text-xs text-gray-400">{t("deviceFleet.compliantDevices")}</p></div>
            <div className={card + " text-center"}><XCircle className="mx-auto h-5 w-5 text-red-400" /><p className="mt-2 text-2xl font-bold text-red-600">{nonCompliant}</p><p className="text-xs text-gray-400">{t("deviceFleet.nonCompliant")}</p></div>
            <div className={card + " text-center"}><Shield className="mx-auto h-5 w-5 text-purple-400" /><p className="mt-2 text-2xl font-bold">{compliancePct}%</p><p className="text-xs text-gray-400">{t("deviceFleet.complianceRate")}</p></div>
          </div>

          {/* SVG Donut */}
          <div className={card}>
            <div className="flex items-center gap-6">
              <svg width={120} height={120} viewBox="0 0 120 120">
                <circle cx="60" cy="60" r="48" fill="none" stroke="#22c55e" strokeWidth="16" strokeDasharray={`${(compliant / (devices.length || 1)) * 301.6} 301.6`} transform="rotate(-90 60 60)" />
                <circle cx="60" cy="60" r="48" fill="none" stroke="#ef4444" strokeWidth="16" strokeDasharray={`${(nonCompliant / (devices.length || 1)) * 301.6} 301.6`} strokeDashoffset={`-${(compliant / (devices.length || 1)) * 301.6}`} transform="rotate(-90 60 60)" />
                <text x="60" y="56" textAnchor="middle" className="fill-gray-900 dark:fill-white text-xl font-bold">{compliancePct}%</text>
                <text x="60" y="74" textAnchor="middle" className="fill-gray-400 text-xs">{t("deviceFleet.compliant")}</text>
              </svg>
              <div className="flex-1 space-y-3">
                <p className="text-sm font-semibold uppercase text-gray-400">{t("deviceFleet.failureReasons")}</p>
                {FAILURE_REASONS.map(f => (
                  <div key={f.reason} className="flex items-center gap-3">
                    <span className="w-40 text-xs text-gray-500">{f.reason}</span>
                    <div className="flex-1 h-4 overflow-hidden rounded-full bg-gray-100 dark:bg-gray-700"><div className="h-full rounded-full bg-red-500" style={{ width: `${f.pct}%` }} /></div>
                    <span className="w-6 text-right text-xs font-mono">{f.count}</span>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* POLICIES */}
      {tab === "policies" && (
        <div className="space-y-6">
          <div className={card}>
            <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Gauge className="h-4 w-4" /> {t("deviceFleet.postureThreshold")}</h3>
            <div className="flex items-center justify-between mb-2"><span className="text-sm font-medium">{t("deviceFleet.minPostureScore")}</span><span className="text-lg font-bold">{minScore}</span></div>
            <input type="range" min={0} max={100} value={minScore} onChange={e => setMinScore(parseInt(e.target.value))} className="w-full accent-blue-500" aria-label="Minimum posture score" />
            <p className="mt-2 text-xs text-gray-400">{t("deviceFleet.thresholdNote")}</p>
          </div>
          <div className={card}>
            <h3 className="mb-3 text-sm font-semibold uppercase text-gray-400">{t("deviceFleet.policyRules")}</h3>
            <div className="space-y-2">
              {[
                { rule: "Require OS version within 2 major releases", enabled: true },
                { rule: "Disk encryption mandatory", enabled: true },
                { rule: "Block jailbroken/rooted devices", enabled: true },
                { rule: "Require MDM enrollment for corporate devices", enabled: false },
                { rule: "Auto-quarantine devices below threshold", enabled: true },
              ].map((r: any, i: number) => (
                <div key={i} className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700">
                  <span className="text-sm">{r.rule}</span>
                  <span className={`px-1.5 py-0.5 rounded text-xs font-medium ${r.enabled ? "bg-green-100 dark:bg-green-900/30 text-green-600" : "bg-gray-100 dark:bg-gray-800 text-gray-400"}`}>{r.enabled ? "on" : "off"}</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      </>)}
    </div>
  );
}

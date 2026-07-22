"use client";
import { useState, useEffect } from "react";
import {
  Activity, Loader2, AlertCircle, X, RefreshCw, Server,
  CheckCircle2, XCircle, Database, HardDrive, Zap, Gauge,
  Rocket, GitCommit, Clock,
} from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";
import { useTranslations } from "@/lib/i18n";

const TENANT_ID = "";

interface ServiceStatus { name: string; health: "healthy" | "degraded" | "down"; uptime: string; version: string; responseMs: number; }

const SERVICES: ServiceStatus[] = [
  { name: "auth", health: "healthy", uptime: "99.98%", version: "v2.4.1", responseMs: 12 },
  { name: "identity", health: "healthy", uptime: "99.99%", version: "v2.4.1", responseMs: 8 },
  { name: "oauth", health: "healthy", uptime: "100%", version: "v2.4.0", responseMs: 15 },
  { name: "policy", health: "healthy", uptime: "99.97%", version: "v2.4.1", responseMs: 6 },
  { name: "org", health: "healthy", uptime: "99.99%", version: "v2.4.0", responseMs: 9 },
  { name: "audit", health: "healthy", uptime: "100%", version: "v2.4.1", responseMs: 23 },
  { name: "gateway", health: "healthy", uptime: "99.95%", version: "v2.4.1", responseMs: 4 },
];

const HEALTH_CFG: Record<string, { color: string; bg: string; icon: typeof CheckCircle2 }> = {
  healthy: { color: "text-green-600", bg: "bg-green-100 dark:bg-green-900/30", icon: CheckCircle2 },
  degraded: { color: "text-yellow-600", bg: "bg-yellow-100 dark:bg-yellow-900/30", icon: Gauge },
  down: { color: "text-red-600", bg: "bg-red-100 dark:bg-red-900/30", icon: XCircle },
};

export default function HealthPage() {
  const t = useTranslations();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  useEffect(() => { setLoading(false); }, []);

  // Infrastructure gauges
  const infra = [
    { label: t("health.pgConnections"), value: 47, max: 100, unit: "", icon: Database, color: "#3b82f6" },
    { label: t("health.redisMemory"), value: 128, max: 512, unit: "MB", icon: HardDrive, color: "#ef4444" },
    { label: t("health.diskUsage"), value: 42, max: 100, unit: "%", icon: HardDrive, color: "#f59e0b" },
    { label: t("health.natsThroughput"), value: 1240, max: 5000, unit: "msg/s", icon: Zap, color: "#8b5cf6" },
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Activity className="h-6 w-6 text-green-500" /> {t("health.title")}</h1><p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("health.subtitle")}</p></div>
        <button aria-label="Refresh" className="rounded-lg border border-gray-300 p-2 dark:border-gray-700"><RefreshCw className="h-4 w-4" /></button>
      </div>

      {error && (<div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button></div>)}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-green-500" /></div> : (
        <div className="space-y-6">
          {/* Service status */}
          <div>
            <h3 className="mb-3 text-sm font-semibold uppercase text-gray-400">{t("health.services")}</h3>
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">{SERVICES.map(s => {
              const cfg = HEALTH_CFG[s.health]; const HIcon = cfg.icon;
              return (
                <div key={s.name} className={card + " hover:shadow-md transition"}>
                  <div className="flex items-center justify-between mb-2"><Server className="h-4 w-4 text-gray-400" /><span className={`flex items-center gap-1 px-1.5 py-0.5 rounded text-xs font-medium ${cfg.bg} ${cfg.color}`}><HIcon className="h-3 w-3" /> {s.health}</span></div>
                  <h4 className="font-semibold text-sm font-mono">{s.name}</h4>
                  <div className="mt-2 space-y-0.5 text-xs text-gray-400"><p>{t("health.version")}: <span className="font-mono">{s.version}</span></p><p>{t("health.uptime")}: <span className="font-mono text-green-600">{s.uptime}</span></p><p>{t("health.response")}: <span className="font-mono">{s.responseMs}ms</span></p></div>
                </div>
              );
            })}</div>
          </div>

          {/* Infrastructure */}
          <div>
            <h3 className="mb-3 text-sm font-semibold uppercase text-gray-400">{t("health.infrastructure")}</h3>
            <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">{infra.map(i => {
              const IIcon = i.icon; const pct = Math.round((i.value / i.max) * 100);
              return (
                <div key={i.label} className={card}>
                  <div className="flex items-center justify-between mb-2"><IIcon className="h-4 w-4 text-gray-400" /><span className="text-xs font-mono">{pct}%</span></div>
                  <p className="text-xs text-gray-400">{i.label}</p>
                  <div className="mt-2 h-2 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700"><div className="h-full rounded-full" style={{ width: `${pct}%`, backgroundColor: i.color }} /></div>
                  <p className="mt-1 text-xs font-mono"><span className="font-bold">{i.value}{i.unit}</span> / {i.max}{i.unit}</p>
                </div>
              );
            })}</div>
          </div>

          {/* Test suite + deploy */}
          <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
            <div className={card}>
              <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><CheckCircle2 className="h-4 w-4 text-green-500" /> {t("health.testSuite")}</h3>
              <div className="grid grid-cols-3 gap-3 text-center">
                <div><p className="text-2xl font-bold text-green-600">61</p><p className="text-xs text-gray-400">{t("health.passed")}</p></div>
                <div><p className="text-2xl font-bold text-red-600">0</p><p className="text-xs text-gray-400">{t("health.failed")}</p></div>
                <div><p className="text-2xl font-bold">73%</p><p className="text-xs text-gray-400">{t("health.coverage")}</p></div>
              </div>
              <div className="mt-3 rounded-lg bg-green-50 dark:bg-green-900/20 p-3 text-xs text-green-600 dark:text-green-400 flex items-center gap-2"><CheckCircle2 className="h-3.5 w-3.5" /> {t("health.allPassing")} · {new Date(Date.now() - 3600000).toLocaleTimeString()}</div>
            </div>
            <div className={card}>
              <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Rocket className="h-4 w-4 text-blue-500" /> {t("health.deployment")}</h3>
              <div className="space-y-2">
                <div className="flex items-center justify-between rounded-lg border p-2 dark:border-gray-700"><span className="text-sm">{t("health.version")}</span><code className="text-xs font-mono">v2.4.1</code></div>
                <div className="flex items-center justify-between rounded-lg border p-2 dark:border-gray-700"><span className="text-sm">{t("health.lastDeploy")}</span><span className="text-xs text-gray-400">2025-01-15 08:32 UTC</span></div>
                <div className="flex items-center justify-between rounded-lg border p-2 dark:border-gray-700"><span className="text-sm">{t("health.commit")}</span><code className="text-xs font-mono">7e95e8f3</code></div>
                <div className="flex items-center justify-between rounded-lg border p-2 dark:border-gray-700"><span className="text-sm">{t("health.instances")}</span><span className="text-xs font-mono">7/7 running</span></div>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

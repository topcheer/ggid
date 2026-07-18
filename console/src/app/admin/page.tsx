"use client";
import { useState, useEffect } from "react";
import {
  Server, Loader2, AlertCircle, X, ChevronRight, Activity,
  Database, KeyRound, Flag, Globe, Power, CheckCircle2,
  Shield, Clock, Rocket, GitCommit, Cpu, HardDrive, Zap,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface QuickLink { href: string; icon: typeof Server; label: string; desc: string; color: string; }
interface AdminAction { id: string; action: string; admin: string; timestamp: string; }

const LINKS: QuickLink[] = [
  { href: "/admin/backup", icon: Database, label: "Backup & DR", desc: "Restore points, failover", color: "text-indigo-500" },
  { href: "/admin/secrets", icon: KeyRound, label: "Secrets", desc: "Rotation, provider health", color: "text-amber-500" },
  { href: "/admin/key-rotation", icon: KeyRound, label: "Key Rotation", desc: "JWT/TLS/CA lifecycle", color: "text-orange-500" },
  { href: "/admin/feature-flags", icon: Flag, label: "Feature Flags", desc: "Toggle experimental features", color: "text-violet-500" },
  { href: "/admin/settings", icon: Globe, label: "Admin Settings", desc: "CORS, shutdown, headers", color: "text-gray-500" },
  { href: "/admin/health", icon: Activity, label: "Platform Health", desc: "Services, infra metrics", color: "text-green-500" },
];

const ACTIONS: AdminAction[] = [
  { id: "a1", action: "Feature flag 'soar_engine' toggled on", admin: "arch@company.com", timestamp: new Date(Date.now() - 1800000).toISOString() },
  { id: "a2", action: "Manual backup triggered", admin: "ops@company.com", timestamp: new Date(Date.now() - 7200000).toISOString() },
  { id: "a3", action: "Secret 'JWT_SIGNING_KEY' rotated", admin: "system:rotator", timestamp: new Date(Date.now() - 86400000).toISOString() },
  { id: "a4", action: "Service 'gateway' restarted", admin: "ops@company.com", timestamp: new Date(Date.now() - 172800000).toISOString() },
];

export default function AdminDashboardPage() {
  const t = useTranslations();
  const [loading, setLoading] = useState(true);

  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  useEffect(() => { setLoading(false); }, []);

  return (
    <div className="space-y-6">
      <div><h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Shield className="h-6 w-6 text-gray-500" /> {t("adminDash.title")}</h1><p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("adminDash.subtitle")}</p></div>

      {/* System overview */}
      <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        <div className={card}><div className="flex items-center gap-2"><Rocket className="h-4 w-4 text-blue-500" /><span className="text-xs text-gray-400">{t("adminDash.version")}</span></div><p className="mt-1 text-lg font-bold font-mono">v2.4.1</p></div>
        <div className={card}><div className="flex items-center gap-2"><Activity className="h-4 w-4 text-green-500" /><span className="text-xs text-gray-400">{t("adminDash.uptime")}</span></div><p className="mt-1 text-lg font-bold text-green-600">99.98%</p></div>
        <div className={card}><div className="flex items-center gap-2"><Server className="h-4 w-4 text-gray-400" /><span className="text-xs text-gray-400">{t("adminDash.services")}</span></div><p className="mt-1 text-lg font-bold">7/7</p></div>
        <div className={card}><div className="flex items-center gap-2"><GitCommit className="h-4 w-4 text-purple-500" /><span className="text-xs text-gray-400">{t("adminDash.commit")}</span></div><p className="mt-1 text-lg font-bold font-mono text-sm">e318eabf</p></div>
      </div>

      {/* Quick links */}
      <div>
        <h2 className="mb-3 text-sm font-semibold uppercase text-gray-400">{t("adminDash.quickLinks")}</h2>
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">{LINKS.map(l => { const Icon = l.icon; return (
          <a key={l.href} href={l.href} className="group rounded-xl border border-gray-200 bg-white p-4 shadow-sm transition hover:shadow-md hover:border-gray-300 dark:border-gray-700 dark:bg-gray-800 dark:hover:border-gray-600">
            <div className="flex items-start justify-between mb-2"><div className="flex h-9 w-9 items-center justify-center rounded-lg bg-gray-100 dark:bg-gray-700"><Icon className={`h-4 w-4 ${l.color}`} /></div><ChevronRight className="h-4 w-4 text-gray-300 opacity-0 group-hover:opacity-100 transition" /></div>
            <h3 className="text-sm font-semibold group-hover:text-blue-600 dark:group-hover:text-blue-400">{l.label}</h3>
            <p className="text-xs text-gray-400">{l.desc}</p>
          </a>
        );})}</div>
      </div>

      {/* Recent admin actions */}
      <div className={card}>
        <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Clock className="h-4 w-4" /> {t("adminDash.recentActions")}</h3>
        <div className="space-y-2">{ACTIONS.map(a => (
          <div key={a.id} className="flex items-center justify-between rounded-lg border p-2 dark:border-gray-700">
            <div className="flex items-center gap-3"><div className="flex h-7 w-7 items-center justify-center rounded-lg bg-gray-100 dark:bg-gray-700"><Activity className="h-3.5 w-3.5 text-gray-400" /></div><div><p className="text-xs">{a.action}</p><p className="text-xs text-gray-400">{a.admin}</p></div></div>
            <span className="text-xs text-gray-400">{new Date(a.timestamp).toLocaleString()}</span>
          </div>
        ))}</div>
      </div>
    </div>
  );
}

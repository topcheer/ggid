"use client";
import { useState } from "react";
import {
  Bell, Loader2, AlertCircle, X, Plus, Check, Save, RefreshCw,
  Mail, MessageSquare, Phone, Radio, ChevronRight, Clock,
  CheckCircle2, XCircle, AlertTriangle, Settings, Zap,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

type Tab = "routing" | "channels" | "log";

interface RoutingRule { severity: string; channels: string[]; }
interface ChannelConfig { type: string; name: string; connected: boolean; config: Record<string, string>; }
interface NotificationLog { id: string; severity: string; channel: string; subject: string; status: "delivered" | "failed" | "pending"; timestamp: string; }

const SEVERITIES = ["critical", "high", "medium", "low", "info"];
const CHANNEL_TYPES = ["email", "slack", "teams", "pagerduty", "webhook", "sms"];

const CHANNEL_CFG: Record<string, { icon: typeof Mail; color: string }> = {
  email: { icon: Mail, color: "text-blue-500" }, slack: { icon: MessageSquare, color: "text-purple-500" },
  teams: { icon: MessageSquare, color: "text-indigo-500" }, pagerduty: { icon: Radio, color: "text-red-500" },
  webhook: { icon: Zap, color: "text-orange-500" }, sms: { icon: Phone, color: "text-green-500" },
};

const RULES: RoutingRule[] = [
  { severity: "critical", channels: ["pagerduty", "slack", "sms"] },
  { severity: "high", channels: ["slack", "email"] },
  { severity: "medium", channels: ["email"] },
  { severity: "low", channels: ["email"] },
  { severity: "info", channels: [] },
];

const CHANNELS: ChannelConfig[] = [
  { type: "email", name: "SMTP (ses.company.com)", connected: true, config: { from: "alerts@ggid.dev", encryption: "TLS" } },
  { type: "slack", name: "#security-alerts", connected: true, config: { webhook_url: "https://hooks.slack.com/..." } },
  { type: "teams", name: "Security Team", connected: false, config: {} },
  { type: "pagerduty", name: "GGID On-Call", connected: true, config: { routing_key: "pd-key-xxx", escalation: "30min" } },
];

const LOGS: NotificationLog[] = [
  { id: "n1", severity: "critical", channel: "pagerduty", subject: "MFA Fatigue Attack Detected", status: "delivered", timestamp: new Date(Date.now() - 300000).toISOString() },
  { id: "n2", severity: "high", channel: "slack", subject: "Impossible Travel: user:bob", status: "delivered", timestamp: new Date(Date.now() - 900000).toISOString() },
  { id: "n3", severity: "medium", channel: "email", subject: "Access Review Due", status: "delivered", timestamp: new Date(Date.now() - 3600000).toISOString() },
  { id: "n4", severity: "critical", channel: "sms", subject: "Database Failover Triggered", status: "failed", timestamp: new Date(Date.now() - 7200000).toISOString() },
];

const SEV_CFG: Record<string, string> = { critical: "bg-red-100 dark:bg-red-900/30 text-red-600", high: "bg-orange-100 dark:bg-orange-900/30 text-orange-600", medium: "bg-yellow-100 dark:bg-yellow-900/30 text-yellow-600", low: "bg-blue-100 dark:bg-blue-900/30 text-blue-600", info: "bg-gray-100 dark:bg-gray-800 text-gray-400" };

export default function NotificationsPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("routing");
  const [rules, setRules] = useState(RULES);
  const [saving, setSaving] = useState(false);

  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const toggleChannel = (sev: string, ch: string) => setRules(prev => prev.map(r => r.severity === sev ? { ...r, channels: r.channels.includes(ch) ? r.channels.filter(c => c !== ch) : [...r.channels, ch] } : r));
  const save = () => { setSaving(true); setTimeout(() => setSaving(false), 800); };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Bell className="h-6 w-6 text-yellow-500" /> {t("notif.title")}</h1><p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("notif.subtitle")}</p></div>
        <button onClick={save} disabled={saving} className="flex items-center gap-2 rounded-lg bg-yellow-600 px-4 py-2 text-sm font-medium text-white hover:bg-yellow-700 disabled:opacity-50">{saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />} {t("notif.save")}</button>
      </div>

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([["routing", t("notif.routing"), Settings], ["channels", t("notif.channels"), Mail], ["log", t("notif.log"), Clock]] as const).map(([id, label, Icon]) => (
          <button key={id} onClick={() => setTab(id as Tab)} aria-pressed={tab === id} className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === id ? "border-yellow-600 text-yellow-600 dark:text-yellow-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}><Icon className="h-4 w-4" /> {label}</button>
        ))}
      </div>

      {/* ROUTING */}
      {tab === "routing" && (
        <div className="space-y-3">{rules.map(r => (
          <div key={r.severity} className={card}>
            <div className="flex items-center gap-2 mb-3"><span className={`px-2 py-0.5 rounded text-sm font-bold ${SEV_CFG[r.severity]}`}>{r.severity}</span><span className="text-xs text-gray-400">→</span><div className="flex flex-wrap gap-1">{CHANNEL_TYPES.map(ch => { const CIcon = CHANNEL_CFG[ch]?.icon; return (
              <button key={ch} onClick={() => toggleChannel(r.severity, ch)} aria-pressed={r.channels.includes(ch)} className={`flex items-center gap-1 rounded-lg border px-2 py-1 text-xs transition ${r.channels.includes(ch) ? "border-yellow-500 bg-yellow-50 dark:bg-yellow-950/30 text-yellow-600" : "border-gray-300 dark:border-gray-700 text-gray-400"}`}>{CIcon && <CIcon className="h-3 w-3" />} {ch}</button>
            );})}</div></div>
          </div>
        ))}</div>
      )}

      {/* CHANNELS */}
      {tab === "channels" && (
        <div className="space-y-3">{CHANNELS.map(c => { const CIcon = CHANNEL_CFG[c.type]?.icon || Mail; return (
          <div key={c.type} className={`${card} flex items-center justify-between !p-3`}>
            <div className="flex items-center gap-3"><div className="flex h-9 w-9 items-center justify-center rounded-lg bg-gray-100 dark:bg-gray-700"><CIcon className={`h-4 w-4 ${CHANNEL_CFG[c.type]?.color}`} /></div><div><span className="text-sm font-medium">{c.name}</span><p className="text-xs text-gray-400">{c.type} · {Object.entries(c.config).slice(0, 2).map(([k, v]) => `${k}=${v.slice(0, 15)}`).join(", ")}</p></div></div>
            {c.connected ? <span className="flex items-center gap-1 px-2 py-0.5 rounded text-xs bg-green-100 dark:bg-green-900/30 text-green-600"><CheckCircle2 className="h-3 w-3" /> {t("notif.connected")}</span> : <button className="rounded-lg border border-gray-300 px-3 py-1 text-xs dark:border-gray-700">{t("notif.connect")}</button>}
          </div>
        );})}<button className="flex items-center gap-1 rounded-lg border border-dashed border-gray-300 px-4 py-2 text-sm text-gray-400 hover:border-gray-400 dark:border-gray-700"><Plus className="h-3.5 w-3.5" /> {t("notif.addChannel")}</button></div>
      )}

      {/* LOG */}
      {tab === "log" && (
        <div className="space-y-2">{LOGS.map(l => (
          <div key={l.id} className={`${card} flex items-center justify-between !p-3`}>
            <div className="flex items-center gap-3"><div className={`flex h-8 w-8 items-center justify-center rounded-lg ${SEV_CFG[l.severity]}`}><AlertTriangle className="h-4 w-4" /></div><div><div className="flex items-center gap-2"><span className={`px-1.5 py-0.5 rounded text-xs font-medium ${SEV_CFG[l.severity]}`}>{l.severity}</span><span className="text-sm font-medium">{l.subject}</span></div><p className="text-xs text-gray-400">{l.channel} · {new Date(l.timestamp).toLocaleString()}</p></div></div>
            <span className={`flex items-center gap-1 px-1.5 py-0.5 rounded text-xs font-medium ${l.status === "delivered" ? "bg-green-100 dark:bg-green-900/30 text-green-600" : l.status === "failed" ? "bg-red-100 dark:bg-red-900/30 text-red-600" : "bg-yellow-100 dark:bg-yellow-900/30 text-yellow-600"}`}>{l.status === "delivered" ? <CheckCircle2 className="h-3 w-3" /> : l.status === "failed" ? <XCircle className="h-3 w-3" /> : <Clock className="h-3 w-3" />} {l.status}</span>
          </div>
        ))}</div>
      )}
    </div>
  );
}

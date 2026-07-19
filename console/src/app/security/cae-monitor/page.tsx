"use client";

import { useState, useEffect, useCallback } from "react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";
import {
  Activity, ScrollText, Zap, Loader2, Plus, Trash2,
  Check, AlertCircle, Radio, Shield, TrendingDown,
} from "lucide-react";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "";
type TabId = "status" | "logs" | "triggers";

interface CAEEvent {
  id: string; timestamp: string; session_id: string; user: string;
  event: string; result: string; risk_delta: number;
}

interface Trigger {
  id: string; event: string; condition: string; action: string; enabled: boolean;
}

const TRIGGER_EVENTS = ["device_change", "ip_change", "risk_spike", "policy_match", "token_expiry"];
const TRIGGER_ACTIONS = ["revoke", "step_up", "challenge", "continue"];

export default function CAEMonitorPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<TabId>("status");

  const tabs: { id: TabId; label: string; icon: typeof Activity }[] = [
    { id: "status", label: t("caeMonitor.tabs.status"), icon: Activity },
    { id: "logs", label: t("caeMonitor.tabs.logs"), icon: ScrollText },
    { id: "triggers", label: t("caeMonitor.tabs.triggers"), icon: Zap },
  ];

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 p-4 md:p-8">
      <div className="max-w-5xl mx-auto">
        <div className="mb-6">
          <div className="flex items-center gap-3 mb-1">
            <Shield className="w-7 h-7 text-blue-600" />
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">{t("caeMonitor.title")}</h1>
          </div>
          <p className="text-gray-600 dark:text-gray-400 text-sm">{t("caeMonitor.description")}</p>
        </div>

        <div className="flex gap-1 mb-6 bg-gray-200 dark:bg-gray-800 rounded-lg p-1">
          {tabs.map(({ id, label, icon: Icon }) => (
            <button key={id} onClick={() => setTab(id)}
              className={`flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                tab === id ? "bg-white dark:bg-gray-700 text-blue-600 dark:text-blue-400 shadow-sm" : "text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white"
              }`}>
              <Icon className="w-4 h-4" />{label}
            </button>
          ))}
        </div>

        {tab === "status" && <StatusTab />}
        {tab === "logs" && <LogsTab />}
        {tab === "triggers" && <TriggersTab />}
      </div>
    </div>
  );
}

// ============ Status Tab ============

function StatusTab() {
  const t = useTranslations();
  const [stats, setStats] = useState<{ active: number; evaluated: number; revoked: number; avgMs: number; running: boolean; lastEval: string; streamActive: boolean } | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const load = async () => {
      try {
        const res = await fetch("/api/v1/security/cae/status", { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
        if (res.ok) {
          const data = await res.json();
          setStats({ active: data.active ?? 0, evaluated: data.evaluated ?? 0, revoked: data.revoked ?? 0, avgMs: data.avg_ms ?? 0, running: data.running ?? false, lastEval: data.last_eval ?? "", streamActive: data.stream_active ?? false });
        }
      } catch {}
      setLoading(false);
    };
    load();
  }, []);

  if (loading || !stats) return <Spinner />;

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <StatCard icon={Activity} label={t("caeMonitor.status.activeSessions")} value={stats.active} color="blue" />
        <StatCard icon={Check} label={t("caeMonitor.status.evaluated")} value={stats.evaluated} color="green" />
        <StatCard icon={TrendingDown} label={t("caeMonitor.status.revoked")} value={stats.revoked} color="red" />
        <StatCard icon={Zap} label={t("caeMonitor.status.avgEvalTime")} value={`${stats.avgMs}ms`} color="orange" />
      </div>

      <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-sm font-semibold text-gray-900 dark:text-white">{t("caeMonitor.status.evalEngine")}</h3>
          <div className="flex items-center gap-2">
            <div className={`w-2.5 h-2.5 rounded-full ${stats.running ? "bg-green-500 animate-pulse" : "bg-gray-400"}`} />
            <span className={`text-xs font-medium ${stats.running ? "text-green-600" : "text-gray-400"}`}>
              {stats.running ? t("caeMonitor.status.engineRunning") : t("caeMonitor.status.engineStopped")}
            </span>
          </div>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="p-3 rounded-lg bg-gray-50 dark:bg-gray-800/50">
            <div className="flex items-center gap-2 mb-1">
              <Radio className={`w-4 h-4 ${stats.streamActive ? "text-green-500" : "text-gray-400"}`} />
              <span className="text-xs text-gray-500">{stats.streamActive ? t("caeMonitor.status.streamActive") : t("caeMonitor.status.streamInactive")}</span>
            </div>
            <span className={`text-sm font-medium ${stats.streamActive ? "text-green-600" : "text-gray-400"}`}>{stats.streamActive ? "Connected" : "Disconnected"}</span>
          </div>
          <div className="p-3 rounded-lg bg-gray-50 dark:bg-gray-800/50">
            <span className="text-xs text-gray-500 block mb-1">{t("caeMonitor.status.lastEval")}</span>
            <span className="text-sm font-medium text-gray-900 dark:text-white">{new Date(stats.lastEval).toLocaleTimeString()}</span>
          </div>
        </div>
      </div>
    </div>
  );
}

// ============ Logs Tab ============

function LogsTab() {
  const t = useTranslations();
  const [logs, setLogs] = useState<CAEEvent[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const load = async () => {
      try {
        const res = await fetch("/api/v1/security/cae/events", { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
        if (res.ok) {
          const data = await res.json();
          setLogs(data.events || data.items || []);
        }
      } catch {}
      setLoading(false);
    };
    load();
  }, []);

  const resultColors: Record<string, string> = {
    revoke: "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300",
    step_up: "bg-blue-100 text-blue-700 dark:bg-blue-950 dark:text-blue-300",
    challenge: "bg-purple-100 text-purple-700 dark:bg-purple-950 dark:text-purple-300",
    continue: "bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-300",
  };

  if (loading) return <Spinner />;

  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
      <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-1">{t("caeMonitor.logs.title")}</h3>
      <p className="text-xs text-gray-500 dark:text-gray-400 mb-4">{t("caeMonitor.logs.description")}</p>
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead><tr className="border-b border-gray-200 dark:border-gray-800 text-left">
            <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400">{t("caeMonitor.logs.timestamp")}</th>
            <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400">{t("caeMonitor.logs.user")}</th>
            <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400">{t("caeMonitor.logs.event")}</th>
            <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400">{t("caeMonitor.logs.result")}</th>
            <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400 text-right">{t("caeMonitor.logs.riskDelta")}</th>
          </tr></thead>
          <tbody>
            {logs.map((l: any) => (
              <tr key={l.id} className="border-b border-gray-100 dark:border-gray-800/50">
                <td className="py-3 px-3 text-xs text-gray-500">{new Date(l.timestamp).toLocaleTimeString()}</td>
                <td className="py-3 px-3 text-gray-900 dark:text-white">{l.user}</td>
                <td className="py-3 px-3"><code className="text-xs text-gray-700 dark:text-gray-300">{l.event}</code></td>
                <td className="py-3 px-3">
                  <span className={`px-2 py-0.5 text-xs rounded-full ${resultColors[l.result]}`}>
                    {t(`caeMonitor.logs.result${l.result.replace(/_./g, (m: any) => m[1].toUpperCase()).replace(/^./, (m: any) => m.toUpperCase())}`)}
                  </span>
                </td>
                <td className="py-3 px-3 text-right">
                  <span className={`text-xs font-medium ${l.risk_delta > 0 ? "text-red-600" : "text-green-600"}`}>
                    {l.risk_delta > 0 ? "+" : ""}{l.risk_delta}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

// ============ Triggers Tab ============

function TriggersTab() {
  const t = useTranslations();
  const [triggers, setTriggers] = useState<Trigger[]>([]);
  const [showForm, setShowForm] = useState(false);

  const resultColors: Record<string, string> = {
    revoke: "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300",
    step_up: "bg-blue-100 text-blue-700 dark:bg-blue-950 dark:text-blue-300",
    challenge: "bg-purple-100 text-purple-700 dark:bg-purple-950 dark:text-purple-300",
    continue: "bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-300",
  };

  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
      <div className="flex items-center justify-between mb-4">
        <div>
          <h3 className="text-sm font-semibold text-gray-900 dark:text-white">{t("caeMonitor.triggers.title")}</h3>
          <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">{t("caeMonitor.triggers.description")}</p>
        </div>
        <button onClick={() => setShowForm(!showForm)} className="flex items-center gap-1.5 px-3 py-1.5 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-medium">
          <Plus className="w-4 h-4" />{t("caeMonitor.triggers.addTrigger")}
        </button>
      </div>

      {showForm && (
        <TriggerForm onAdd={(tr) => { setTriggers([...triggers, tr]); setShowForm(false); }} onCancel={() => setShowForm(false)} />
      )}

      <div className="space-y-2">
        {triggers.map((tr: any) => (
          <div key={tr.id} className="flex items-center gap-3 p-3 rounded-lg border border-gray-200 dark:border-gray-800">
            <div className="flex-1">
              <div className="flex items-center gap-2">
                <code className="text-xs font-medium text-gray-900 dark:text-white">{tr.event}</code>
                <span className="text-xs text-gray-400">{tr.condition}</span>
              </div>
            </div>
            <span className={`px-2 py-0.5 text-xs rounded-full ${resultColors[tr.action]}`}>{tr.action.replace(/_/g, " ")}</span>
            <button onClick={() => setTriggers(triggers.map((x: any) => x.id === tr.id ? { ...x, enabled: !x.enabled } : x))}
              className={`relative w-10 h-6 rounded-full transition-colors ${tr.enabled ? "bg-blue-600" : "bg-gray-300 dark:bg-gray-600"}`}>
              <span className={`absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full transition-transform ${tr.enabled ? "translate-x-4" : ""}`} />
            </button>
            <button onClick={() => { if (confirm(t("caeMonitor.triggers.confirmDelete"))) setTriggers(triggers.filter((x: any) => x.id !== tr.id)); }}
              className="p-1.5 hover:bg-red-50 dark:hover:bg-red-950 rounded"><Trash2 className="w-4 h-4 text-red-500" /></button>
          </div>
        ))}
      </div>
    </div>
  );
}

function TriggerForm({ onAdd, onCancel }: { onAdd: (t: Trigger) => void; onCancel: () => void }) {
  const t = useTranslations();
  const [event, setEvent] = useState("risk_spike");
  const [condition, setCondition] = useState("");
  const [action, setAction] = useState("step_up");

  return (
    <div className="border border-gray-200 dark:border-gray-700 rounded-lg p-4 space-y-3 bg-gray-50 dark:bg-gray-800/50">
      <div className="grid grid-cols-3 gap-3">
        <select value={event} onChange={(e) => setEvent(e.target.value)} className="px-2 py-1.5 rounded border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-xs text-gray-900 dark:text-white">
          {TRIGGER_EVENTS.map((e: any) => <option key={e} value={e}>{e}</option>)}
        </select>
        <input type="text" value={condition} onChange={(e) => setCondition(e.target.value)} placeholder="condition"
          className="px-2 py-1.5 rounded border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-xs text-gray-900 dark:text-white" />
        <select value={action} onChange={(e) => setAction(e.target.value)} className="px-2 py-1.5 rounded border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-xs text-gray-900 dark:text-white">
          {TRIGGER_ACTIONS.map((a: any) => <option key={a} value={a}>{a}</option>)}
        </select>
      </div>
      <div className="flex gap-2">
        <button onClick={() => onAdd({ id: `t${Date.now()}`, event, condition, action, enabled: true })}
          className="px-4 py-1.5 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-xs font-medium">{t("caeMonitor.triggers.addTrigger")}</button>
        <button onClick={onCancel} className="px-4 py-1.5 bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg text-xs font-medium">Cancel</button>
      </div>
    </div>
  );
}

// ============ Shared ============

function Spinner() { return <div className="flex justify-center py-20"><Loader2 className="w-8 h-8 animate-spin text-blue-600" /></div>; }

function StatCard({ icon: Icon, label, value, color }: { icon: typeof Activity; label: string; value: string | number; color: string }) {
  const colors: Record<string, string> = { blue: "text-blue-600", green: "text-green-600", orange: "text-orange-500", red: "text-red-500" };
  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-4">
      <div className="flex items-center gap-2 mb-2"><Icon className={`w-5 h-5 ${colors[color]}`} /><span className="text-xs text-gray-500 dark:text-gray-400">{label}</span></div>
      <div className="text-2xl font-bold text-gray-900 dark:text-white">{value}</div>
    </div>
  );
}

"use client";
import { useState, useEffect } from "react";
import {
  Puzzle, Loader2, AlertCircle, X, Upload, Trash2, Check, Play,
  ChevronRight, Zap, Code, BookOpen, Power, FileText, Lock,
  CheckCircle2, XCircle, Shield,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

type Tab = "installed" | "upload" | "hooks";

interface Plugin { id: string; name: string; version: string; hooks: string[]; status: "active" | "disabled"; size_kb: number; }
interface HookDef { name: string; desc: string; signature: string; }
interface HostFn { name: string; desc: string; }

const PLUGINS: Plugin[] = [
  { id: "p1", name: "ip-geolocator", version: "1.2.0", hooks: ["pre_auth", "post_auth"], status: "active", size_kb: 142 },
  { id: "p2", name: "audit-enricher", version: "0.9.1", hooks: ["post_auth", "post_policy"], status: "active", size_kb: 87 },
  { id: "p3", name: "custom-claims", version: "2.0.0", hooks: ["token_issue"], status: "disabled", size_kb: 210 },
];

const HOOKS: HookDef[] = [
  { name: "pre_auth", desc: "Called before authentication attempt", signature: "fn(user, ip, device) → Result<Allow, Deny>" },
  { name: "post_auth", desc: "Called after successful authentication", signature: "fn(user, session, risk) → Result<Continue, StepUp>" },
  { name: "pre_policy", desc: "Called before policy evaluation", signature: "fn(subject, resource, action) → Result<Eval, Allow>" },
  { name: "post_policy", desc: "Called after policy decision", signature: "fn(decision, context) → Result<Log, Alert>" },
  { name: "token_issue", desc: "Called during token issuance", signature: "fn(claims, scope) → Claims" },
  { name: "session_create", desc: "Called on new session creation", signature: "fn(session) → Result<Allow, Deny>" },
];

const HOST_FNS: HostFn[] = [
  { name: "log_event", desc: "Write to audit log" },
  { name: "get_user_attr", desc: "Read user attribute" },
  { name: "set_risk_score", desc: "Override risk score" },
  { name: "check_rate_limit", desc: "Query rate limiter" },
  { name: "http_fetch", desc: "External HTTP request (sandboxed)" },
  { name: "kv_get/set", desc: "Key-value store access" },
  { name: "emit_metric", desc: "Push to Prometheus metrics" },
];

export default function PluginsPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("installed");
  const [plugins, setPlugins] = useState<Plugin[]>([]);
  const [loading, setLoading] = useState(true);

  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  useEffect(() => { setPlugins(PLUGINS); setLoading(false); }, []);

  const togglePlugin = (id: string) => setPlugins(prev => prev.map(p => p.id === id ? { ...p, status: p.status === "active" ? "disabled" : "active" } : p));
  const deletePlugin = (id: string) => setPlugins(prev => prev.filter(p => p.id !== id));

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Puzzle className="h-6 w-6 text-violet-500" /> {t("plugins.title")}</h1><p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("plugins.subtitle")}</p></div>
        <button onClick={() => setTab("upload")} className="flex items-center gap-1 rounded-lg bg-violet-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-violet-700"><Upload className="h-3 w-3" /> {t("plugins.upload")}</button>
      </div>

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([["installed", `${t("plugins.installed")} (${plugins.length})`, Puzzle], ["upload", t("plugins.uploadTab"), Upload], ["hooks", t("plugins.hookRef"), BookOpen]] as const).map(([id, label, Icon]) => (
          <button key={id} onClick={() => setTab(id as Tab)} aria-pressed={tab === id} className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === id ? "border-violet-600 text-violet-600 dark:text-violet-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}><Icon className="h-4 w-4" /> {label}</button>
        ))}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-violet-500" /></div> : (<>

      {/* INSTALLED */}
      {tab === "installed" && (
        <div className="space-y-3">{plugins.map(p => (
          <div key={p.id} className={`${card} flex items-center justify-between !p-3`}>
            <div className="flex items-center gap-3"><div className="flex h-10 w-10 items-center justify-center rounded-lg bg-violet-100 dark:bg-violet-900/30"><Puzzle className="h-5 w-5 text-violet-500" /></div><div><div className="flex items-center gap-2"><span className="text-sm font-medium">{p.name}</span><code className="text-xs font-mono text-gray-400">v{p.version}</code><span className={`px-1.5 py-0.5 rounded text-xs font-medium ${p.status === "active" ? "bg-green-100 dark:bg-green-900/30 text-green-600" : "bg-gray-100 dark:bg-gray-800 text-gray-400"}`}>{p.status}</span></div><div className="flex flex-wrap gap-1 mt-0.5">{p.hooks.map(h => <span key={h} className="px-1 py-0.5 rounded bg-gray-100 dark:bg-gray-700 text-xs font-mono">{h}</span>)}</div><p className="text-xs text-gray-400">{p.size_kb} KB · WASM</p></div></div>
            <div className="flex items-center gap-2"><button onClick={() => togglePlugin(p.id)} aria-pressed={p.status === "active"} aria-label={"Toggle " + p.name} className={`relative h-6 w-11 rounded-full transition ${p.status === "active" ? "bg-green-500" : "bg-gray-300 dark:bg-gray-700"}`}><span className={`absolute top-0.5 h-5 w-5 rounded-full bg-white transition ${p.status === "active" ? "left-5" : "left-0.5"}`} /></button><button onClick={() => deletePlugin(p.id)} aria-label={"Delete " + p.name} className="rounded p-1.5 text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20"><Trash2 className="h-3.5 w-3.5" /></button></div>
          </div>
        ))}</div>
      )}

      {/* UPLOAD */}
      {tab === "upload" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={card}>
            <h3 className="mb-4 text-sm font-semibold uppercase text-gray-400">{t("plugins.uploadWasm")}</h3>
            <div className="rounded-xl border-2 border-dashed border-gray-300 p-8 text-center dark:border-gray-700"><Upload className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-2 text-sm text-gray-500">{t("plugins.dropWasm")}</p><p className="text-xs text-gray-400">.wasm · max 5MB</p><button className="mt-3 rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700">{t("plugins.browse")}</button></div>
          </div>
          <div className={card}>
            <h3 className="mb-3 text-sm font-semibold uppercase text-gray-400">{t("plugins.declareCapabilities")}</h3>
            <div className="space-y-2 text-xs">
              <div className="rounded-lg bg-gray-50 dark:bg-gray-900/50 p-3"><p className="font-semibold mb-1">{t("plugins.hooksToUse")}</p><div className="flex flex-wrap gap-1">{HOOKS.map(h => <label key={h.name} className="flex items-center gap-1"><input type="checkbox" className="rounded" /> <code className="font-mono">{h.name}</code></label>)}</div></div>
              <div className="rounded-lg bg-gray-50 dark:bg-gray-900/50 p-3"><p className="font-semibold mb-1">{t("plugins.hostFunctions")}</p><div className="flex flex-wrap gap-1">{HOST_FNS.map(f => <label key={f.name} className="flex items-center gap-1"><input type="checkbox" className="rounded" /> <code className="font-mono">{f.name}</code></label>)}</div></div>
              <div className="flex items-center gap-2 rounded-lg bg-yellow-50 dark:bg-yellow-900/20 p-2"><Lock className="h-3 w-3 text-yellow-500" /><span className="text-yellow-600">{t("plugins.sandboxNote")}</span></div>
              <button className="w-full rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700">{t("plugins.deploy")}</button>
            </div>
          </div>
        </div>
      )}

      {/* HOOKS */}
      {tab === "hooks" && (
        <div className="space-y-6">
          <div className={card}><h3 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Zap className="h-4 w-4" /> {t("plugins.lifecycleHooks")}</h3><div className="space-y-2">{HOOKS.map(h => (<div key={h.name} className="flex items-start gap-3 rounded-lg border p-3 dark:border-gray-700"><Zap className="h-4 w-4 text-violet-400 mt-0.5" /><div><code className="text-sm font-mono text-violet-500">{h.name}</code><p className="text-xs text-gray-400 mt-0.5">{h.desc}</p><code className="text-xs font-mono text-gray-500 block mt-1">{h.signature}</code></div></div>))}</div></div>
          <div className={card}><h3 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Code className="h-4 w-4" /> {t("plugins.hostFunctions")}</h3><div className="grid grid-cols-1 gap-2 sm:grid-cols-2">{HOST_FNS.map(f => (<div key={f.name} className="rounded-lg border p-2 dark:border-gray-700"><code className="text-xs font-mono text-blue-500">{f.name}</code><p className="text-xs text-gray-400">{f.desc}</p></div>))}</div></div>
        </div>
      )}

      </>)}
    </div>
  );
}

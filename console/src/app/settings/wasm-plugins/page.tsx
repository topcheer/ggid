"use client";
import { useState, useCallback, useEffect } from "react";
import { Boxes, Loader2, AlertCircle, X, RefreshCw, Plus, Trash2, Check, CheckCircle, XCircle, Upload, Settings, Activity, Zap, Code, Eye, ChevronRight, ArrowRight, RotateCcw, Cpu, Globe } from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface Plugin { id: string; name: string; version: string; status: "running" | "stopped" | "error"; hooks: string[]; memory_mb: number; calls_24h: number; avg_latency_ms: number; error_rate_pct: number; fuel_used: number; config: string; }
interface HookPoint { id: string; name: string; phase: string; description: string; plugins: string[]; }

const HOOK_POINTS: HookPoint[] = [
  { id: "gateway_pre", name: "Gateway Pre-Request", phase: "Gateway", description: "Before request enters GGID routing", plugins: [] },
  { id: "auth_pre", name: "Pre-Auth", phase: "Auth", description: "Before credential validation", plugins: [] },
  { id: "auth_post", name: "Post-Auth", phase: "Auth", description: "After successful authentication", plugins: [] },
  { id: "token_issue", name: "Token Issuance", phase: "Token", description: "Before JWT issuance, can inject claims", plugins: [] },
  { id: "token_validate", name: "Token Validation", phase: "Token", description: "During token verification", plugins: [] },
  { id: "policy_check", name: "Policy Evaluation", phase: "Policy", description: "Before ABAC/RBAC evaluation", plugins: [] },
  { id: "policy_post", name: "Policy Decision", phase: "Policy", description: "After policy decision, can override", plugins: [] },
  { id: "jit_provision", name: "JIT Provisioning", phase: "JIT", description: "During just-in-time user creation", plugins: [] },
  { id: "session_revoke", name: "Session Revocation", phase: "Session", description: "When session is revoked", plugins: [] },
  { id: "audit_log", name: "Audit Logging", phase: "Audit", description: "Before audit event is persisted", plugins: [] },
];

type Tab = "plugins" | "upload" | "hooks" | "config" | "metrics";

export default function WASMPluginsPage() {
  const [tab, setTab] = useState<Tab>("plugins");
  const [plugins, setPlugins] = useState<Plugin[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  // Upload
  const [fileName, setFileName] = useState("");
  const [upName, setUpName] = useState("");
  const [upHook, setUpHook] = useState("auth_post");
  const [upMemory, setUpMemory] = useState(64);
  const [upFuel, setUpFuel] = useState(100000);
  const [upTenant, setUpTenant] = useState("all");
  const [uploading, setUploading] = useState(false);
  // Config
  const [selectedPlugin, setSelectedPlugin] = useState<string | null>(null);
  const [configText, setConfigText] = useState("{}");
  const [saving, setSaving] = useState(false);
  // Actions
  const [togglingId, setTogglingId] = useState<string | null>(null);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/system/wasm-plugins", { headers: { ...authHeader(), "X-Tenant-ID": TENANT_ID } }).catch(() => null);
      if (res?.ok) { const d = await res.json(); setPlugins(d.plugins || d.items || []); }
    } catch { setError("Failed to load plugins"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const togglePlugin = async (id: string, status: string) => {
    setTogglingId(id);
    try {
      await fetch(`/api/v1/system/wasm-plugins/${id}`, { method: "PUT", headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID }, body: JSON.stringify({ status: status === "running" ? "stopped" : "running" }) });
      setPlugins(prev => prev.map(p => p.id === id ? { ...p, status: p.status === "running" ? "stopped" : "running" } : p));
    } catch { /* noop */ }
    finally { setTogglingId(null); }
  };

  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const selPlugin = plugins.find(p => p.id === selectedPlugin);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
          <Boxes className="h-6 w-6 text-indigo-500" /> WASM Plugin Console
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Upload and manage WebAssembly plugins across 10 identity lifecycle hook points.
        </p>
      </div>

      {error && (
        <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}
          <button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "plugins" as Tab, label: "Plugins", icon: Boxes },
          { id: "upload" as Tab, label: "Upload", icon: Upload },
          { id: "hooks" as Tab, label: "Hook Points", icon: Zap },
          { id: "config" as Tab, label: "Configuration", icon: Settings },
          { id: "metrics" as Tab, label: "Metrics", icon: Activity },
        ]).map(tb => {
          const Icon = tb.icon;
          return (
            <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id}
              className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === tb.id ? "border-indigo-600 text-indigo-600 dark:text-indigo-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}>
              <Icon className="h-4 w-4" /> {tb.label}
            </button>
          );
        })}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-500" /></div> : (<>

      {/* PLUGINS LIST */}
      {tab === "plugins" && (
        <div>
          {plugins.length === 0 ? (
            <div className={card}><div className="py-12 text-center"><Boxes className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No plugins installed. Upload a .wasm module to get started.</p><button onClick={() => setTab("upload")} className="mt-3 text-sm text-indigo-600 hover:underline">Upload plugin</button></div></div>
          ) : (
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">{plugins.map(p => (
              <div key={p.id} className={card + " hover:shadow-md transition"}>
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-3">
                    <div className={`h-2.5 w-2.5 rounded-full ${p.status === "running" ? "bg-green-500 animate-pulse" : p.status === "error" ? "bg-red-500" : "bg-gray-400"}`} />
                    <div><h3 className="font-semibold text-sm">{p.name}</h3><p className="text-xs text-gray-400">v{p.version}</p></div>
                  </div>
                  <span className={`px-1.5 py-0.5 rounded text-xs font-medium ${p.status === "running" ? "bg-green-100 dark:bg-green-900/30 text-green-600" : p.status === "error" ? "bg-red-100 dark:bg-red-900/30 text-red-600" : "bg-gray-100 dark:bg-gray-800 text-gray-400"}`}>{p.status}</span>
                </div>
                <div className="mt-3 flex flex-wrap gap-1">{p.hooks?.map(h => <span key={h} className="px-1.5 py-0.5 rounded bg-indigo-100 dark:bg-indigo-900/30 text-indigo-600 text-xs font-mono">{h}</span>)}</div>
                <div className="mt-3 grid grid-cols-2 gap-2 text-center">
                  <div><p className="text-xs text-gray-400">Calls 24h</p><p className="text-sm font-bold">{p.calls_24h}</p></div>
                  <div><p className="text-xs text-gray-400">Avg Latency</p><p className="text-sm font-bold">{p.avg_latency_ms}ms</p></div>
                  <div><p className="text-xs text-gray-400">Memory</p><p className="text-sm font-bold">{p.memory_mb}MB</p></div>
                  <div><p className="text-xs text-gray-400">Errors</p><p className={`text-sm font-bold ${p.error_rate_pct > 1 ? "text-red-600" : "text-green-600"}`}>{p.error_rate_pct}%</p></div>
                </div>
                <div className="mt-3 flex items-center justify-between">
                  <span className="text-xs text-gray-400">Fuel: {p.fuel_used.toLocaleString()}</span>
                  <div className="flex gap-1">
                    <button onClick={() => { setSelectedPlugin(p.id); setConfigText(p.config || "{}"); setTab("config"); }} className="rounded p-1 text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700"><Settings className="h-3.5 w-3.5" /></button>
                    <button onClick={() => togglePlugin(p.id, p.status)} disabled={togglingId === p.id} className="rounded-lg px-2 py-1 text-xs font-medium border dark:border-gray-700">{togglingId === p.id ? <Loader2 className="h-3 w-3 animate-spin" /> : p.status === "running" ? "Stop" : "Start"}</button>
                  </div>
                </div>
              </div>
            ))}</div>
          )}
        </div>
      )}

      {/* UPLOAD */}
      {tab === "upload" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Upload className="h-4 w-4" /> Upload .wasm Module</h2>
            <div className="rounded-xl border-2 border-dashed border-gray-300 p-6 text-center dark:border-gray-700">
              <Upload className="mx-auto h-10 w-10 text-gray-300" />
              <p className="mt-2 text-sm text-gray-500">Drop .wasm file or click to browse</p>
              <input type="file" accept=".wasm" className="hidden" onChange={e => setFileName(e.target.files?.[0]?.name || "")} />
              {fileName && <p className="mt-2 text-xs text-green-600">{fileName}</p>}
            </div>
          </div>
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Settings className="h-4 w-4" /> Plugin Metadata</h2>
            <div className="space-y-3">
              <div><label className="text-sm font-medium">Plugin Name</label><input type="text" value={upName} onChange={e => setUpName(e.target.value)} placeholder="custom-claim-injector" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>
              <div><label className="text-sm font-medium">Hook Point</label><select value={upHook} onChange={e => setUpHook(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">{HOOK_POINTS.map(h => <option key={h.id} value={h.id}>{h.name} ({h.phase})</option>)}</select></div>
              <div className="grid grid-cols-2 gap-3">
                <div><label className="text-sm font-medium">Memory (MB)</label><input type="number" min={16} max={512} value={upMemory} onChange={e => setUpMemory(parseInt(e.target.value) || 64)} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>
                <div><label className="text-sm font-medium">Fuel Limit</label><input type="number" value={upFuel} onChange={e => setUpFuel(parseInt(e.target.value) || 100000)} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>
              </div>
              <div><label className="text-sm font-medium">Tenant Scope</label><input type="text" value={upTenant} onChange={e => setUpTenant(e.target.value)} placeholder="all or tenant-id" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>
              <button disabled={!fileName || uploading} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{uploading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Upload className="h-4 w-4" />} Deploy Plugin</button>
            </div>
          </div>
        </div>
      )}

      {/* HOOK POINTS */}
      {tab === "hooks" && (
        <div className={card}>
          <h2 className="mb-6 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Zap className="h-4 w-4" /> Identity Lifecycle Hook Points</h2>
          <div className="space-y-2">{HOOK_POINTS.map(h => (
            <div key={h.id} className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700">
              <div className="flex items-center gap-3">
                <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-indigo-100 dark:bg-indigo-900/30"><Zap className="h-4 w-4 text-indigo-500" /></div>
                <div>
                  <div className="flex items-center gap-2"><span className="font-medium text-sm">{h.name}</span><span className="px-1.5 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-700">{h.phase}</span></div>
                  <p className="text-xs text-gray-400">{h.description}</p>
                </div>
              </div>
              <div className="flex items-center gap-2">
                {h.plugins.length > 0 ? h.plugins.map(p => <span key={p} className="px-1.5 py-0.5 rounded bg-green-100 dark:bg-green-900/30 text-green-600 text-xs font-mono">{p}</span>) : <span className="text-xs text-gray-300">no plugins</span>}
              </div>
            </div>
          ))}</div>
        </div>
      )}

      {/* CONFIG */}
      {tab === "config" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
          <div className={card + " lg:col-span-1"}>
            <h2 className="mb-3 text-sm font-semibold uppercase text-gray-400">Select Plugin</h2>
            <div className="space-y-1">{plugins.map(p => (
              <button key={p.id} onClick={() => { setSelectedPlugin(p.id); setConfigText(p.config || "{}"); }} aria-pressed={selectedPlugin === p.id}
                className={`flex w-full items-center justify-between rounded-lg border p-2 text-left ${selectedPlugin === p.id ? "border-indigo-500 bg-indigo-50 dark:bg-indigo-950/30" : "border-gray-200 dark:border-gray-700"}`}>
                <span className="text-sm font-medium">{p.name}</span>
                <span className="text-xs text-gray-400">v{p.version}</span>
              </button>
            ))}</div>
          </div>
          <div className={card + " lg:col-span-2"}>
            <h2 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Code className="h-4 w-4" /> Configuration {selPlugin && `- ${selPlugin.name}`}</h2>
            {selPlugin ? (
              <div>
                <textarea aria-label="Plugin config" value={configText} onChange={e => setConfigText(e.target.value)} rows={10} className="w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 font-mono text-xs" />
                <div className="mt-3 flex gap-2">
                  <button onClick={() => setSaving(true)} disabled={saving} className="flex items-center gap-1 rounded-lg bg-indigo-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{saving ? <Loader2 className="h-3 w-3 animate-spin" /> : <Check className="h-3 w-3" />} Save Config</button>
                  <button className="flex items-center gap-1 rounded-lg border border-gray-300 px-3 py-1.5 text-xs dark:border-gray-700"><RotateCcw className="h-3 w-3" /> Reset to Default</button>
                </div>
              </div>
            ) : <p className="text-sm text-gray-400">Select a plugin to edit configuration.</p>}
          </div>
        </div>
      )}

      {/* METRICS */}
      {tab === "metrics" && (
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
            <div className={card + " text-center"}><Cpu className="h-5 w-5 mx-auto text-indigo-400" /><p className="mt-2 text-2xl font-bold">{plugins.length}</p><p className="text-xs text-gray-400">Active Plugins</p></div>
            <div className={card + " text-center"}><Activity className="h-5 w-5 mx-auto text-blue-400" /><p className="mt-2 text-2xl font-bold">{plugins.reduce((a: any, p: any) => a + p.calls_24h, 0)}</p><p className="text-xs text-gray-400">Total Calls 24h</p></div>
            <div className={card + " text-center"}><Zap className="h-5 w-5 mx-auto text-yellow-400" /><p className="mt-2 text-2xl font-bold">{plugins.reduce((a: any, p: any) => a + p.fuel_used, 0).toLocaleString()}</p><p className="text-xs text-gray-400">Fuel Consumed</p></div>
            <div className={card + " text-center"}><Cpu className="h-5 w-5 mx-auto text-purple-400" /><p className="mt-2 text-2xl font-bold">{plugins.reduce((a: any, p: any) => a + p.memory_mb, 0)}MB</p><p className="text-xs text-gray-400">Total Memory</p></div>
          </div>
          {plugins.length === 0 ? <div className={card}><div className="py-8 text-center"><Activity className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No plugin metrics.</p></div></div> : (
            <div className={card}>
              <h3 className="mb-3 text-sm font-semibold uppercase text-gray-400">Per-Plugin Metrics</h3>
              <div className="space-y-2">{plugins.map(p => (
                <div key={p.id} className="rounded-lg border p-3 dark:border-gray-700">
                  <div className="flex items-center justify-between mb-2">
                    <span className="font-medium text-sm">{p.name}</span>
                    <span className={`text-xs ${p.error_rate_pct > 1 ? "text-red-600" : "text-green-600"}`}>{p.error_rate_pct}% errors</span>
                  </div>
                  <div className="grid grid-cols-3 gap-2">
                    <div className="flex items-center gap-2"><span className="text-xs text-gray-400 w-16">Calls</span><div className="flex-1 h-2 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700"><div className="h-full rounded-full bg-blue-500" style={{ width: `${Math.min(p.calls_24h / 10, 100)}%` }} /></div><span className="text-xs font-mono w-10 text-right">{p.calls_24h}</span></div>
                    <div className="flex items-center gap-2"><span className="text-xs text-gray-400 w-16">Latency</span><div className="flex-1 h-2 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700"><div className="h-full rounded-full bg-yellow-500" style={{ width: `${Math.min(p.avg_latency_ms, 100)}%` }} /></div><span className="text-xs font-mono w-10 text-right">{p.avg_latency_ms}ms</span></div>
                    <div className="flex items-center gap-2"><span className="text-xs text-gray-400 w-16">Memory</span><div className="flex-1 h-2 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700"><div className="h-full rounded-full bg-purple-500" style={{ width: `${Math.min(p.memory_mb / 2, 100)}%` }} /></div><span className="text-xs font-mono w-10 text-right">{p.memory_mb}MB</span></div>
                  </div>
                </div>
              ))}</div>
            </div>
          )}
        </div>
      )}

      </>)}
    </div>
  );
}

"use client";
import { useState } from "react";
import {
  Puzzle, Loader2, AlertCircle, X, Upload, Check, Play, Trash2,
  ChevronRight, Zap, Code, History, FileCheck, AlertTriangle,
  Lock, Activity, Plus,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

type Tab = "upload" | "versions";

const HOOKS = ["pre_auth", "post_auth", "pre_policy", "post_policy", "token_issue", "session_create"];
const HOST_FNS = ["log_event", "get_user_attr", "set_risk_score", "check_rate_limit", "http_fetch", "kv_getset", "emit_metric"];

interface PluginVersion { version: string; uploaded: string; size_kb: number; hash: string; }

const VERSIONS: PluginVersion[] = [
  { version: "1.2.0", uploaded: "2025-01-14T10:00:00Z", size_kb: 142, hash: "sha256:a1b2c3..." },
  { version: "1.1.0", uploaded: "2025-01-05T14:00:00Z", size_kb: 138, hash: "sha256:d4e5f6..." },
  { version: "1.0.0", uploaded: "2024-12-20T09:00:00Z", size_kb: 125, hash: "sha256:g7h8i9..." },
];

export default function PluginManagePage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("upload");
  const [selectedHooks, setSelectedHooks] = useState<string[]>([]);
  const [selectedFns, setSelectedFns] = useState<string[]>([]);
  const [uploading, setUploading] = useState(false);
  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState<string | null>(null);
  const [fileName, setFileName] = useState("");

  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const toggle = (list: string[], item: string, setter: (v: string[]) => void) => setter(list.includes(item) ? list.filter(x => x !== item) : [...list, item]);

  const upload = () => { if (!fileName) return; setUploading(true); setTimeout(() => setUploading(false), 1200); };
  const testInvoke = () => { setTesting(true); setTestResult(null); setTimeout(() => { setTesting(false); setTestResult("Plugin executed successfully in 4ms. Output: {\"result\": \"ok\"}"); }, 1000); };

  return (
    <div className="space-y-6">
      <div><h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Puzzle className="h-6 w-6 text-violet-500" /> {t("pluginMgr.title")}</h1><p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("pluginMgr.subtitle")}</p></div>

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([["upload", t("pluginMgr.uploadDeploy"), Upload], ["versions", t("pluginMgr.versions"), History]] as const).map(([id, label, Icon]) => (
          <button key={id} onClick={() => setTab(id as Tab)} aria-pressed={tab === id} className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === id ? "border-violet-600 text-violet-600 dark:text-violet-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}><Icon className="h-4 w-4" /> {label}</button>
        ))}
      </div>

      {/* UPLOAD */}
      {tab === "upload" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={card}>
            <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Upload className="h-4 w-4" /> {t("pluginMgr.wasmFile")}</h3>
            <div className="rounded-xl border-2 border-dashed border-gray-300 p-6 text-center dark:border-gray-700">
              <Upload className="mx-auto h-8 w-8 text-gray-300" />
              <p className="mt-2 text-sm text-gray-500">{fileName || t("pluginMgr.dropWasm")}</p>
              <p className="text-xs text-gray-400 mt-1">.wasm · max 16MB</p>
              <label className="mt-3 inline-flex items-center gap-1 rounded-lg bg-violet-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-violet-700 cursor-pointer"><Upload className="h-3 w-3" /> {t("pluginMgr.browse")}<input type="file" accept=".wasm" className="hidden" onChange={e => setFileName(e.target.files?.[0]?.name || "")} /></label>
            </div>
            {fileName && (
              <div className="mt-3 flex items-center gap-2 rounded-lg bg-green-50 dark:bg-green-900/20 p-2 text-xs text-green-600"><FileCheck className="h-3.5 w-3.5" /> {fileName} · {(142).toFixed(0)}KB</div>
            )}
          </div>
          <div className={card}>
            <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Zap className="h-4 w-4" /> {t("pluginMgr.hooksCaps")}</h3>
            <div className="space-y-3">
              <div><p className="text-xs font-semibold mb-1">{t("pluginMgr.registerHooks")}</p><div className="flex flex-wrap gap-1">{HOOKS.map(h => <button key={h} onClick={() => toggle(selectedHooks, h, setSelectedHooks)} aria-pressed={selectedHooks.includes(h)} className={`rounded px-2 py-1 text-xs font-mono ${selectedHooks.includes(h) ? "bg-violet-100 dark:bg-violet-900/30 text-violet-600" : "bg-gray-100 dark:bg-gray-700 text-gray-400"}`}>{h}</button>)}</div></div>
              <div><p className="text-xs font-semibold mb-1">{t("pluginMgr.hostFns")}</p><div className="flex flex-wrap gap-1">{HOST_FNS.map(f => <button key={f} onClick={() => toggle(selectedFns, f, setSelectedFns)} aria-pressed={selectedFns.includes(f)} className={`rounded px-2 py-1 text-xs font-mono ${selectedFns.includes(f) ? "bg-blue-100 dark:bg-blue-900/30 text-blue-600" : "bg-gray-100 dark:bg-gray-700 text-gray-400"}`}>{f}</button>)}</div></div>
              <div className="rounded-lg bg-yellow-50 dark:bg-yellow-900/20 p-2 text-xs text-yellow-600"><Lock className="inline h-3 w-3" /> {t("pluginMgr.sandboxNote")}</div>
            </div>
          </div>
          <div className="lg:col-span-2 flex items-center gap-3">
            <button onClick={upload} disabled={!fileName || uploading} className="flex items-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700 disabled:opacity-50">{uploading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Upload className="h-4 w-4" />} {t("pluginMgr.deploy")}</button>
            <button onClick={testInvoke} disabled={!fileName || testing} className="flex items-center gap-2 rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">{testing ? <Loader2 className="h-4 w-4 animate-spin" /> : <Play className="h-4 w-4" />} {t("pluginMgr.testInvoke")}</button>
          </div>
          {testResult && (
            <div className="lg:col-span-2 rounded-lg border border-green-300 bg-green-50 dark:border-green-700 dark:bg-green-950/30 p-3"><div className="flex items-center gap-2"><Check className="h-4 w-4 text-green-500" /><span className="text-sm font-medium text-green-700 dark:text-green-400">{t("pluginMgr.testSuccess")}</span></div><code className="mt-1 block text-xs font-mono text-green-600">{testResult}</code></div>
          )}
        </div>
      )}

      {/* VERSIONS */}
      {tab === "versions" && (
        <div className="space-y-2">
          {VERSIONS.map((v, i) => (
            <div key={v.version} className={`${card} flex items-center justify-between !p-3 ${i === 0 ? "border-violet-200 dark:border-violet-800" : ""}`}>
              <div className="flex items-center gap-3"><div className="flex h-9 w-9 items-center justify-center rounded-lg bg-violet-100 dark:bg-violet-900/30"><History className="h-4 w-4 text-violet-500" /></div><div><div className="flex items-center gap-2"><code className="text-sm font-mono font-bold">v{v.version}</code>{i === 0 && <span className="px-1.5 py-0.5 rounded text-xs bg-green-100 dark:bg-green-900/30 text-green-600">{t("pluginMgr.current")}</span>}</div><p className="text-xs text-gray-400">{new Date(v.uploaded).toLocaleString()} · {v.size_kb}KB · <code className="font-mono">{v.hash.slice(0, 20)}</code></p></div></div>
              <div className="flex gap-1"><button className="rounded-lg border border-gray-300 px-2 py-1 text-xs dark:border-gray-700">{t("pluginMgr.rollback")}</button>{i !== 0 && <button aria-label="Delete version" className="rounded p-1.5 text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20"><Trash2 className="h-3.5 w-3.5" /></button>}</div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

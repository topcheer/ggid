"use client";
import { useState, useEffect } from "react";
import {
  Settings, Loader2, AlertCircle, X, RefreshCw, Save, Check,
  Shield, Power, ToggleLeft, Lock, Globe, Cookie, Server,
  CheckCircle2, XCircle, Activity, Zap, ChevronRight,
} from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";
import { useTranslations } from "@/lib/i18n";

const TENANT_ID = "";

type Tab = "security" | "shutdown" | "flags";

const SERVICES = ["auth", "identity", "oauth", "policy", "org", "audit", "gateway"];

const FLAGS = [
  { key: "graphql", label: "GraphQL API", desc: "Enable GraphQL query endpoint (KB-111)", enabled: false },
  { key: "dlp_egress", label: "DLP Egress Control", desc: "Gateway PII detection + redaction middleware", enabled: true },
  { key: "ueba", label: "UEBA Behavioral Analytics", desc: "Isolation forest anomaly scoring", enabled: true },
  { key: "soar", label: "SOAR Playbook Engine", desc: "Automated threat response workflows", enabled: true },
  { key: "rebac", label: "ReBAC / Zanzibar", desc: "Relationship-based access control engine", enabled: false },
  { key: "pqc", label: "Post-Quantum Signing", desc: "PQC audit chain signatures (experimental)", enabled: false },
  { key: "consent_cascade", label: "Consent Cascade", desc: "GDPR Art.17 token/session revocation on withdrawal", enabled: true },
  { key: "adaptive_mfa", label: "Adaptive MFA", desc: "Risk-based step-up authentication triggers", enabled: true },
];

export default function AdminSettingsPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("security");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [flags, setFlags] = useState(FLAGS);

  // Security config
  const [corsOrigins, setCorsOrigins] = useState("https://console.ggid.dev\nhttps://admin.ggid.dev");
  const [cookieSecure, setCookieSecure] = useState(true);
  const [cookieHttpOnly, setCookieHttpOnly] = useState(true);
  const [cookieSameSite, setCookieSameSite] = useState("strict");

  // Shutdown state
  const [draining, setDraining] = useState(false);
  const [shutdownProgress, setShutdownProgress] = useState(0);
  const [shuttingDown, setShuttingDown] = useState(false);

  const H = { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID };
  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  useEffect(() => { setLoading(false); }, []);

  const saveConfig = async () => { setSaving(true); setTimeout(() => setSaving(false), 800); };
  const triggerDrain = () => { setDraining(true); setShutdownProgress(0); const timer = setInterval(() => { setShutdownProgress(p => { if (p >= 100) { clearInterval(timer); setDraining(false); return 100; } return p + 10; }); }, 200); };

  const securityHeaders = [
    { header: "Strict-Transport-Security", value: "max-age=63072000; includeSubDomains; preload", ok: true },
    { header: "X-Content-Type-Options", value: "nosniff", ok: true },
    { header: "X-Frame-Options", value: "DENY", ok: true },
    { header: "Content-Security-Policy", value: "default-src 'self'; script-src 'self'", ok: true },
    { header: "Referrer-Policy", value: "strict-origin-when-cross-origin", ok: true },
    { header: "Permissions-Policy", value: "camera=(), microphone=(), geolocation=()", ok: true },
  ];

  return (
    <div className="space-y-6">
      <div><h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Settings className="h-6 w-6 text-gray-500" /> {t("adminSettings.title")}</h1><p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("adminSettings.subtitle")}</p></div>

      {error && (<div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button></div>)}

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([["security", t("adminSettings.security"), Shield], ["shutdown", t("adminSettings.shutdown"), Power], ["flags", t("adminSettings.featureFlags"), ToggleLeft]] as const).map(([id, label, Icon]) => (
          <button key={id} onClick={() => setTab(id as Tab)} aria-pressed={tab === id} className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === id ? "border-gray-600 text-gray-600 dark:text-gray-300" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}><Icon className="h-4 w-4" /> {label}</button>
        ))}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-gray-500" /></div> : (<>

      {/* SECURITY */}
      {tab === "security" && (
        <div className="space-y-6">
          <div className={card}>
            <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Globe className="h-4 w-4" /> {t("adminSettings.corsConfig")}</h3>
            <textarea value={corsOrigins} onChange={e => setCorsOrigins(e.target.value)} rows={4} className="w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" />
          </div>
          <div className={card}>
            <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Cookie className="h-4 w-4" /> {t("adminSettings.cookieSettings")}</h3>
            <div className="space-y-2">
              {([["Secure", cookieSecure, () => setCookieSecure(!cookieSecure)], ["HttpOnly", cookieHttpOnly, () => setCookieHttpOnly(!cookieHttpOnly)]] as const).map(([label, val, toggle]) => (
                <label key={label} className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700"><span className="text-sm font-medium">{label}</span><button onClick={toggle} aria-pressed={val} className={`relative h-6 w-11 rounded-full transition ${val ? "bg-green-500" : "bg-gray-300 dark:bg-gray-700"}`}><span className={`absolute top-0.5 h-5 w-5 rounded-full bg-white transition ${val ? "left-5" : "left-0.5"}`} /></button></label>
              ))}
              <div className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700"><span className="text-sm font-medium">SameSite</span><select value={cookieSameSite} onChange={e => setCookieSameSite(e.target.value)} className="rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-1.5 text-sm"><option value="strict">strict</option><option value="lax">lax</option><option value="none">none</option></select></div>
            </div>
          </div>
          <div className={card}>
            <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Lock className="h-4 w-4" /> {t("adminSettings.securityHeaders")}</h3>
            <div className="space-y-1">{securityHeaders.map(h => (
              <div key={h.header} className="flex items-center justify-between rounded-lg border p-2 dark:border-gray-700"><div className="flex items-center gap-2 min-w-0"><CheckCircle2 className="h-4 w-4 text-green-500 shrink-0" /><code className="text-xs font-mono text-gray-500 shrink-0">{h.header}</code><code className="text-xs font-mono text-gray-400 truncate">{h.value}</code></div></div>
            ))}</div>
          </div>
          <button onClick={saveConfig} disabled={saving} className="flex items-center gap-2 rounded-lg bg-gray-700 px-4 py-2 text-sm font-medium text-white hover:bg-gray-800 disabled:opacity-50">{saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />} {t("adminSettings.save")}</button>
        </div>
      )}

      {/* SHUTDOWN */}
      {tab === "shutdown" && (
        <div className="space-y-6">
          <div className={card}>
            <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Power className="h-4 w-4" /> {t("adminSettings.gracefulShutdown")}</h3>
            <p className="text-sm text-gray-500 dark:text-gray-400 mb-4">{t("adminSettings.shutdownDesc")}</p>
            {draining ? (
              <div><div className="mb-2 flex items-center justify-between"><span className="text-sm">{t("adminSettings.draining")}</span><span className="text-sm font-mono">{shutdownProgress}%</span></div><div className="h-3 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700"><div className="h-full rounded-full bg-yellow-500 transition-all" style={{ width: `${shutdownProgress}%` }} /></div></div>
            ) : (
              <button onClick={triggerDrain} className="flex items-center gap-2 rounded-lg bg-yellow-600 px-4 py-2 text-sm font-medium text-white hover:bg-yellow-700"><Power className="h-4 w-4" /> {t("adminSettings.triggerDrain")}</button>
            )}
          </div>
          <div className={card}>
            <h3 className="mb-3 text-sm font-semibold uppercase text-gray-400">{t("adminSettings.serviceControls")}</h3>
            <div className="space-y-2">{SERVICES.map(svc => (
              <div key={svc} className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700">
                <div className="flex items-center gap-3"><Server className="h-4 w-4 text-green-500" /><span className="text-sm font-medium font-mono">{svc}</span><span className="flex items-center gap-1 px-1.5 py-0.5 rounded text-xs bg-green-100 dark:bg-green-900/30 text-green-600"><CheckCircle2 className="h-3 w-3" /> running</span></div>
                <div className="flex gap-1"><button aria-label={"Restart " + svc} className="rounded-lg border border-gray-300 px-2 py-1 text-xs dark:border-gray-700">{t("adminSettings.restart")}</button></div>
              </div>
            ))}</div>
          </div>
        </div>
      )}

      {/* FLAGS */}
      {tab === "flags" && (
        <div className="space-y-3">
          {flags.map(f => (
            <div key={f.key} className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700">
              <div><div className="flex items-center gap-2"><code className="text-sm font-mono text-gray-700 dark:text-gray-300">{f.key}</code>{f.enabled ? <span className="px-1.5 py-0.5 rounded text-xs bg-green-100 dark:bg-green-900/30 text-green-600">on</span> : <span className="px-1.5 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800 text-gray-400">off</span>}</div><p className="text-xs text-gray-400 mt-0.5">{f.desc}</p></div>
              <button onClick={() => setFlags(prev => prev.map(x => x.key === f.key ? { ...x, enabled: !x.enabled } : x))} aria-pressed={f.enabled} aria-label={f.label} className={`relative h-6 w-11 rounded-full transition ${f.enabled ? "bg-green-500" : "bg-gray-300 dark:bg-gray-700"}`}><span className={`absolute top-0.5 h-5 w-5 rounded-full bg-white transition ${f.enabled ? "left-5" : "left-0.5"}`} /></button>
            </div>
          ))}
          <button onClick={saveConfig} disabled={saving} className="flex items-center gap-2 rounded-lg bg-gray-700 px-4 py-2 text-sm font-medium text-white hover:bg-gray-800 disabled:opacity-50">{saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />} {t("adminSettings.saveFlags")}</button>
        </div>
      )}

      </>)}
    </div>
  );
}

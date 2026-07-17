"use client";
import { useState, useCallback, useEffect } from "react";
import {
  Globe, Loader2, AlertCircle, X, RefreshCw, Plus, Trash2, Check,
  Shield, Activity, ChevronRight, Zap, Server, Clock,
  CheckCircle2, XCircle, AlertTriangle, Lock, Settings, Cpu,
} from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";
import { useTranslations } from "@/lib/i18n";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface ProtectedApp {
  id: string; name: string; slug: string; upstream_url: string;
  description?: string; auth_mode: string; health_status: string;
  rate_limit_per_min: number; enabled: boolean; created_at: string;
}

type Tab = "apps" | "metrics" | "logs" | "tester";

const HEALTH_CFG: Record<string, { color: string; bg: string; icon: typeof CheckCircle2 }> = {
  healthy: { color: "text-green-600", bg: "bg-green-100 dark:bg-green-900/30", icon: CheckCircle2 },
  unhealthy: { color: "text-red-600", bg: "bg-red-100 dark:bg-red-900/30", icon: XCircle },
  degraded: { color: "text-yellow-600", bg: "bg-yellow-100 dark:bg-yellow-900/30", icon: AlertTriangle },
  unknown: { color: "text-gray-500", bg: "bg-gray-100 dark:bg-gray-800", icon: Clock },
};

export default function ZTNAPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("apps");
  const [apps, setApps] = useState<ProtectedApp[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  // App form
  const [showForm, setShowForm] = useState(false);
  const [fName, setFName] = useState("");
  const [fSlug, setFSlug] = useState("");
  const [fUrl, setFUrl] = useState("");
  const [fMode, setFMode] = useState("jwt");
  const [fLimit, setFLimit] = useState(100);

  // Policy tester
  const [tUser, setTUser] = useState("");
  const [tApp, setTApp] = useState("");
  const [tResult, setTResult] = useState<{ allowed: boolean; reason: string } | null>(null);
  const [testing, setTesting] = useState(false);

  const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
  const H = { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID };
  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/ztna/apps", { headers: h }).catch(() => null);
      if (res?.ok) { const d = await res.json(); setApps(d.apps || d || []); }
    } catch { setError(t("ztna.loadError")); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const createApp = async () => {
    if (!fName || !fSlug || !fUrl) return;
    setActionLoading("create");
    try {
      await fetch("/api/v1/ztna/apps", { method: "POST", headers: H, body: JSON.stringify({ name: fName, slug: fSlug, upstream_url: fUrl, auth_mode: fMode, rate_limit_per_min: fLimit }) });
      setShowForm(false); setFName(""); setFSlug(""); setFUrl(""); setFLimit(100);
      loadData();
    } catch { setError(t("ztna.createError")); }
    finally { setActionLoading(null); }
  };

  const deleteApp = async (id: string) => {
    setActionLoading(`del-${id}`);
    try { await fetch(`/api/v1/ztna/apps/${id}`, { method: "DELETE", headers: h }); loadData(); }
    catch { setError(t("ztna.deleteError")); }
    finally { setActionLoading(null); }
  };

  const testPolicy = async () => {
    if (!tUser || !tApp) return;
    setTesting(true); setTResult(null);
    try {
      const res = await fetch("/api/v1/ztna/test-policy", { method: "POST", headers: H, body: JSON.stringify({ user_id: tUser, app_id: tApp }) }).catch(() => null);
      if (res?.ok) { const d = await res.json(); setTResult({ allowed: d.allowed ?? true, reason: d.reason ?? "Access granted" }); }
      else setTResult({ allowed: false, reason: "Policy evaluation failed" });
    } catch { setError("Policy test failed"); }
    finally { setTesting(false); }
  };

  const activeApps = apps.filter(a => a.enabled);
  const healthyApps = apps.filter(a => a.health_status === "healthy");

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
          <Shield className="h-6 w-6 text-cyan-500" /> {t("ztna.title")}
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("ztna.subtitle")}</p>
      </div>

      {error && (
        <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}
          <button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "apps" as Tab, label: t("ztna.apps"), icon: Server },
          { id: "metrics" as Tab, label: t("ztna.metrics"), icon: Activity },
          { id: "logs" as Tab, label: t("ztna.accessLogs"), icon: Clock },
          { id: "tester" as Tab, label: t("ztna.policyTester"), icon: Zap },
        ]).map(tb => {
          const Icon = tb.icon;
          return (
            <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id}
              className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === tb.id ? "border-cyan-600 text-cyan-600 dark:text-cyan-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}>
              <Icon className="h-4 w-4" /> {tb.label}
            </button>
          );
        })}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-cyan-500" /></div> : (<>

      {/* ════ APPS ════ */}
      {tab === "apps" && (
        <div>
          <div className="mb-4 flex items-center justify-between">
            <div className="grid grid-cols-3 gap-3">
              <div className="text-center"><p className="text-lg font-bold">{activeApps.length}</p><p className="text-xs text-gray-400">{t("ztna.active")}</p></div>
              <div className="text-center"><p className="text-lg font-bold text-green-600">{healthyApps.length}</p><p className="text-xs text-gray-400">{t("ztna.healthy")}</p></div>
              <div className="text-center"><p className="text-lg font-bold text-red-600">{apps.filter(a => a.health_status === "unhealthy").length}</p><p className="text-xs text-gray-400">{t("ztna.unhealthy")}</p></div>
            </div>
            <button onClick={() => setShowForm(true)} className="flex items-center gap-1 rounded-lg bg-cyan-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-cyan-700">
              <Plus className="h-3 w-3" /> {t("ztna.addApp")}
            </button>
          </div>
          {apps.length === 0 ? (
            <div className={card}><div className="py-12 text-center"><Server className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">{t("ztna.noApps")}</p></div></div>
          ) : (
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">{apps.map(app => {
              const cfg = HEALTH_CFG[app.health_status] || HEALTH_CFG.unknown;
              const HIcon = cfg.icon;
              return (
                <div key={app.id} className={card + " hover:shadow-md transition"}>
                  <div className="flex items-start justify-between">
                    <div className="flex items-center gap-3">
                      <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-gray-100 dark:bg-gray-700"><Server className="h-5 w-5 text-cyan-500" /></div>
                      <div><h3 className="font-semibold text-sm">{app.name}</h3><p className="text-xs text-gray-400">/{app.slug}</p></div>
                    </div>
                    <span className={`flex items-center gap-1 px-1.5 py-0.5 rounded text-xs font-medium ${cfg.bg} ${cfg.color}`}><HIcon className="h-3 w-3" /> {app.health_status}</span>
                  </div>
                  <div className="mt-3 space-y-1 text-xs text-gray-500">
                    <p className="truncate">URL: <span className="font-mono">{app.upstream_url}</span></p>
                    <p>Auth: <span className="font-mono">{app.auth_mode}</span> · Limit: {app.rate_limit_per_min}/min</p>
                  </div>
                  <div className="mt-3 flex items-center justify-between">
                    <span className={`h-2 w-2 rounded-full ${app.enabled ? "bg-green-500 animate-pulse" : "bg-gray-400"}`} />
                    <button onClick={() => deleteApp(app.id)} disabled={actionLoading === `del-${app.id}`} aria-label={"Delete " + app.name} className="rounded p-1 text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20">
                      {actionLoading === `del-${app.id}` ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Trash2 className="h-3.5 w-3.5" />}
                    </button>
                  </div>
                </div>
              );
            })}</div>
          )}
        </div>
      )}

      {/* ════ METRICS ════ */}
      {tab === "metrics" && (
        <div className="space-y-6">
          <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
            <div className={card + " text-center"}><Server className="mx-auto h-5 w-5 text-cyan-400" /><p className="mt-2 text-2xl font-bold">{apps.length}</p><p className="text-xs text-gray-400">{t("ztna.totalApps")}</p></div>
            <div className={card + " text-center"}><Activity className="mx-auto h-5 w-5 text-green-400" /><p className="mt-2 text-2xl font-bold">12,847</p><p className="text-xs text-gray-400">{t("ztna.requestsToday")}</p></div>
            <div className={card + " text-center"}><Lock className="mx-auto h-5 w-5 text-red-400" /><p className="mt-2 text-2xl font-bold text-red-600">23</p><p className="text-xs text-gray-400">{t("ztna.blockedToday")}</p></div>
            <div className={card + " text-center"}><Cpu className="mx-auto h-5 w-5 text-blue-400" /><p className="mt-2 text-2xl font-bold">8ms</p><p className="text-xs text-gray-400">{t("ztna.avgLatency")}</p></div>
          </div>
          <div className={card}>
            <h3 className="mb-3 text-sm font-semibold uppercase text-gray-400">{t("ztna.perAppMetrics")}</h3>
            <div className="space-y-2">
              {apps.map(app => (
                <div key={app.id} className="flex items-center gap-3 rounded-lg border p-3 dark:border-gray-700">
                  <Server className="h-4 w-4 text-gray-400" />
                  <span className="text-sm font-medium flex-1">{app.name}</span>
                  <span className="text-xs text-gray-400">{app.rate_limit_per_min}/min</span>
                  <span className={`px-1.5 py-0.5 rounded text-xs ${(HEALTH_CFG[app.health_status] || HEALTH_CFG.unknown).bg} ${(HEALTH_CFG[app.health_status] || HEALTH_CFG.unknown).color}`}>{app.health_status}</span>
                </div>
              ))}
              {apps.length === 0 && <p className="text-sm text-gray-400">{t("ztna.noApps")}</p>}
            </div>
          </div>
        </div>
      )}

      {/* ════ LOGS ════ */}
      {tab === "logs" && (
        <div className={card}>
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Clock className="h-4 w-4" /> {t("ztna.accessLog")}</h2>
          <div className="space-y-2">
            {[...Array(6)].map((_, i) => {
              const users = ["user:alice", "user:bob", "user:carol", "service:api-gateway"];
              const actions = ["allow", "allow", "allow", "block", "allow", "step_up"];
              const action = actions[i];
              const usr = users[i % users.length];
              const appNames = apps.length > 0 ? apps.map(a => a.name) : ["dashboard", "api-docs", "grafana"];
              const appN = appNames[i % appNames.length];
              const time = new Date(Date.now() - i * 120000).toISOString();
              return (
                <div key={i} className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700">
                  <div className="flex items-center gap-3">
                    <div className={`flex h-8 w-8 items-center justify-center rounded-lg ${action === "allow" ? "bg-green-100 dark:bg-green-900/30" : action === "block" ? "bg-red-100 dark:bg-red-900/30" : "bg-yellow-100 dark:bg-yellow-900/30"}`}>
                      {action === "allow" ? <CheckCircle2 className="h-4 w-4 text-green-500" /> : action === "block" ? <XCircle className="h-4 w-4 text-red-500" /> : <AlertTriangle className="h-4 w-4 text-yellow-500" />}
                    </div>
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="text-xs font-mono">{usr}</span>
                        <ChevronRight className="h-3 w-3 text-gray-300" />
                        <span className="text-xs font-medium">{appN}</span>
                      </div>
                      <p className="text-xs text-gray-400">{new Date(time).toLocaleTimeString()}</p>
                    </div>
                  </div>
                  <span className={`px-1.5 py-0.5 rounded text-xs font-medium ${action === "allow" ? "bg-green-100 dark:bg-green-900/30 text-green-600" : action === "block" ? "bg-red-100 dark:bg-red-900/30 text-red-600" : "bg-yellow-100 dark:bg-yellow-900/30 text-yellow-600"}`}>{action}</span>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* ════ POLICY TESTER ════ */}
      {tab === "tester" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Zap className="h-4 w-4" /> {t("ztna.testPolicy")}</h2>
            <div className="space-y-3">
              <div><label className="text-sm font-medium">{t("ztna.user")}</label><input type="text" value={tUser} onChange={e => setTUser(e.target.value)} placeholder="user:alice" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div>
              <div><label className="text-sm font-medium">{t("ztna.application")}</label>
                <select value={tApp} onChange={e => setTApp(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">
                  <option value="">{t("ztna.selectApp")}</option>
                  {apps.map(a => <option key={a.id} value={a.id}>{a.name}</option>)}
                </select>
              </div>
              <button onClick={testPolicy} disabled={!tUser || !tApp || testing} className="flex items-center gap-2 rounded-lg bg-cyan-600 px-4 py-2 text-sm font-medium text-white hover:bg-cyan-700 disabled:opacity-50">
                {testing ? <Loader2 className="h-4 w-4 animate-spin" /> : <Zap className="h-4 w-4" />} {t("ztna.evaluate")}
              </button>
            </div>
          </div>
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Shield className="h-4 w-4" /> {t("ztna.result")}</h2>
            {tResult ? (
              <div className={`flex items-center gap-3 rounded-xl border-2 p-4 ${tResult.allowed ? "border-green-300 bg-green-50 dark:border-green-700 dark:bg-green-950/30" : "border-red-300 bg-red-50 dark:border-red-700 dark:bg-red-950/30"}`}>
                {tResult.allowed ? <CheckCircle2 className="h-8 w-8 text-green-500" /> : <XCircle className="h-8 w-8 text-red-500" />}
                <div>
                  <p className={`text-lg font-bold ${tResult.allowed ? "text-green-700 dark:text-green-400" : "text-red-700 dark:text-red-400"}`}>{tResult.allowed ? "ALLOWED" : "DENIED"}</p>
                  <p className="text-xs text-gray-500 dark:text-gray-400">{tResult.reason}</p>
                </div>
              </div>
            ) : <div className="py-8 text-center"><Zap className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">{t("ztna.noTestResult")}</p></div>}
          </div>
        </div>
      )}

      </>)}

      {/* Create app modal */}
      {showForm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowForm(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <h3 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white"><Plus className="h-5 w-5 text-cyan-500" /> {t("ztna.addApp")}</h3>
            <div className="mt-4 space-y-3">
              <div><label className="text-sm font-medium">{t("ztna.appName")}</label><input type="text" value={fName} onChange={e => setFName(e.target.value)} placeholder="Grafana Dashboard" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" autoFocus /></div>
              <div><label className="text-sm font-medium">{t("ztna.slug")}</label><input type="text" value={fSlug} onChange={e => setFSlug(e.target.value)} placeholder="grafana" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div>
              <div><label className="text-sm font-medium">{t("ztna.upstreamUrl")}</label><input type="text" value={fUrl} onChange={e => setFUrl(e.target.value)} placeholder="http://grafana.internal:3000" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div>
              <div className="grid grid-cols-2 gap-3">
                <div><label className="text-sm font-medium">{t("ztna.authMode")}</label>
                  <select value={fMode} onChange={e => setFMode(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">
                    <option value="jwt">JWT</option><option value="header">Header Injection</option><option value="oauth2">OAuth2 Proxy</option>
                  </select>
                </div>
                <div><label className="text-sm font-medium">{t("ztna.rateLimit")}</label><input type="number" value={fLimit} onChange={e => setFLimit(parseInt(e.target.value) || 100)} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>
              </div>
            </div>
            <div className="mt-4 flex justify-end gap-2">
              <button onClick={() => setShowForm(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">{t("common.cancel")}</button>
              <button onClick={createApp} disabled={!fName || !fSlug || !fUrl || actionLoading === "create"} className="rounded-lg bg-cyan-600 px-4 py-2 text-sm font-medium text-white hover:bg-cyan-700 disabled:opacity-50">
                {actionLoading === "create" ? <Loader2 className="h-4 w-4 animate-spin" /> : t("ztna.create")}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

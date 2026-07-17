"use client";

import { useState, useCallback, useEffect, useRef } from "react";
import {
  Shield, Loader2, AlertCircle, X, RefreshCw, Plus, Globe, Server,
  Activity, Zap, Clock, CheckCircle, XCircle, Filter, Search,
  ChevronRight, ChevronLeft, Terminal, Gauge, ShieldCheck, AlertTriangle,
  Code, Eye, ArrowRight, Settings, Trash2, Edit3,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

// ==================== Types ====================
interface ProtectedApp {
  id: string;
  name: string;
  domain: string;
  upstream_url: string;
  health: "healthy" | "degraded" | "down";
  qps: number;
  avg_latency_ms: number;
  error_rate_pct: number;
  policy_count: number;
  auth_mode: "oauth2" | "saml" | "oidc" | "basic";
  headers_injected: number;
  created_at: string;
}

interface AccessLogEntry {
  id: string;
  timestamp: string;
  app_name: string;
  user_id: string;
  method: string;
  path: string;
  status_code: number;
  decision: "allow" | "deny" | "stepup";
  latency_ms: number;
  reason: string;
}

interface MetricPoint {
  timestamp: string;
  qps: number;
  latency_ms: number;
  error_rate_pct: number;
}

interface PolicyCondition {
  id: string;
  attribute: string;
  operator: "eq" | "ne" | "in" | "contains" | "gt" | "lt";
  value: string;
}

interface Policy {
  id: string;
  name: string;
  effect: "allow" | "deny" | "stepup";
  conditions: PolicyCondition[];
  priority: number;
  enabled: boolean;
}

const healthConfig = {
  healthy: { color: "text-green-500", dot: "bg-green-500", label: "Healthy" },
  degraded: { color: "text-yellow-500", dot: "bg-yellow-500", label: "Degraded" },
  down: { color: "text-red-500", dot: "bg-red-500", label: "Down" },
};

const decisionConfig = {
  allow: { color: "text-green-600 bg-green-50 dark:bg-green-950/20", icon: CheckCircle },
  deny: { color: "text-red-600 bg-red-50 dark:bg-red-950/20", icon: XCircle },
  stepup: { color: "text-yellow-600 bg-yellow-50 dark:bg-yellow-950/20", icon: Shield },
};

type Tab = "apps" | "monitor" | "logs" | "tester";

export default function AccessBrokerPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("apps");
  const [apps, setApps] = useState<ProtectedApp[]>([]);
  const [logs, setLogs] = useState<AccessLogEntry[]>([]);
  const [metrics, setMetrics] = useState<MetricPoint[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  // Wizard
  const [showWizard, setShowWizard] = useState(false);
  const [wizardStep, setWizardStep] = useState(0);
  // Logs filter
  const [logFilter, setLogFilter] = useState("");
  const [logDecisionFilter, setLogDecisionFilter] = useState("all");
  // Policy tester
  const [testAppId, setTestAppId] = useState("");
  const [testUserId, setTestUserId] = useState("");
  const [testMethod, setTestMethod] = useState("GET");
  const [testPath, setTestPath] = useState("/");
  const [testContext, setTestContext] = useState('{"ip":"10.0.0.1","device":"managed","risk_score":20}');
  const [testResult, setTestResult] = useState<{ decision: string; reason: string; matched_policies: string[] } | null>(null);
  const [testing, setTesting] = useState(false);
  // Live monitoring
  const [liveMode, setLiveMode] = useState(false);
  const wsRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const headers = () => ({ ...authHeader(), "X-Tenant-ID": TENANT_ID });

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
      const [appsRes, logsRes] = await Promise.all([
        fetch("/api/v1/ztna/apps", { headers: h }).catch(() => null),
        fetch("/api/v1/ztna/access-logs?page_size=100", { headers: h }).catch(() => null),
      ]);
      if (appsRes?.ok) { const d = await appsRes.json(); setApps(d.apps || d.items || []); }
      if (logsRes?.ok) { const d = await logsRes.json(); setLogs(d.logs || d.items || []); }
    } catch { setError("Failed to load ZTNA data"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  // Live monitoring poll
  useEffect(() => {
    if (tab === "monitor" && liveMode) {
      wsRef.current = setInterval(async () => {
        const res = await fetch("/api/v1/ztna/metrics", { headers: { ...authHeader(), "X-Tenant-ID": TENANT_ID } }).catch(() => null);
        if (res?.ok) {
          const d = await res.json();
          setMetrics(prev => [...prev.slice(-29), ...(d.points || [])]);
        }
      }, 3000);
    }
    return () => { if (wsRef.current) clearInterval(wsRef.current); };
  }, [tab, liveMode]);

  const runPolicyTest = async () => {
    if (!testAppId || !testUserId) return;
    setTesting(true);
    setTestResult(null);
    try {
      const res = await fetch("/api/v1/ztna/test-policy", {
        method: "POST",
        headers: { ...headers(), "Content-Type": "application/json" },
        body: JSON.stringify({ app_id: testAppId, user_id: testUserId, method: testMethod, path: testPath, context: JSON.parse(testContext) }),
      });
      if (res.ok) setTestResult(await res.json());
      else setTestResult({ decision: "deny", reason: "Test failed", matched_policies: [] });
    } catch { setTestResult({ decision: "deny", reason: "Network error", matched_policies: [] }); }
    finally { setTesting(false); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const filteredLogs = logs.filter(l => {
    if (logDecisionFilter !== "all" && l.decision !== logDecisionFilter) return false;
    if (logFilter && !l.user_id.includes(logFilter) && !l.path.includes(logFilter) && !l.app_name.includes(logFilter)) return false;
    return true;
  });

  const wizardSteps = ["Basic Info", "Upstream", "Auth Policy", "Headers", "Health Check", "Review"];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <Shield className="h-6 w-6 text-indigo-500" />
            Access Broker / ZTNA Console
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Zero Trust Network Access — protect apps with identity-aware proxy, ABAC policies, and real-time monitoring.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button onClick={() => setShowWizard(true)} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-700">
            <Plus className="h-4 w-4" /> Register App
          </button>
          <button onClick={loadData} disabled={loading} aria-label="Refresh" className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800">
            <RefreshCw className={"h-4 w-4 " + (loading ? "animate-spin" : "")} /> Refresh
          </button>
        </div>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {/* Tabs */}
      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "apps" as Tab, label: "Applications", icon: Globe },
          { id: "monitor" as Tab, label: "Monitoring", icon: Gauge },
          { id: "logs" as Tab, label: "Access Logs", icon: Terminal },
          { id: "tester" as Tab, label: "Policy Tester", icon: Code },
        ]).map(tb => { const Icon = tb.icon; return (
          <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id}
            className={"flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap " +
              (tab === tb.id ? "border-indigo-600 text-indigo-600 dark:text-indigo-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300")}>
            <Icon className="h-4 w-4" /> {tb.label}
          </button>
        ); })}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-500" /></div> : (<>

      {/* APPLICATIONS TAB */}
      {tab === "apps" && (
        <>
          {apps.length === 0 ? (
            <div className={cardCls}><div className="py-12 text-center"><Globe className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No protected applications registered.</p><button onClick={() => setShowWizard(true)} className="mt-3 text-sm text-indigo-600 hover:underline">Register your first app</button></div></div>
          ) : (
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {apps.map(app => {
                const hc = healthConfig[app.health] || healthConfig.down;
                return (
                  <div key={app.id} className={cardCls + " transition hover:shadow-md"}>
                    <div className="flex items-start justify-between">
                      <div className="flex items-center gap-3">
                        <div className={"h-3 w-3 rounded-full " + hc.dot + (app.health === "degraded" ? " animate-pulse" : "")} />
                        <div>
                          <h3 className="font-semibold text-gray-900 dark:text-white">{app.name}</h3>
                          <p className="text-xs text-gray-400 font-mono">{app.domain}</p>
                        </div>
                      </div>
                      <div className="flex gap-1">
                        <button aria-label="Edit app" className="rounded p-1 text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700"><Edit3 className="h-3.5 w-3.5" /></button>
                        <button aria-label="Delete app" className="rounded p-1 text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20"><Trash2 className="h-3.5 w-3.5" /></button>
                      </div>
                    </div>
                    <div className="mt-3 flex items-center gap-2">
                      <span className="px-1.5 py-0.5 rounded text-xs font-mono bg-indigo-100 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400">{app.auth_mode}</span>
                      <span className={"px-1.5 py-0.5 rounded text-xs " + hc.color}>{hc.label}</span>
                      {app.headers_injected > 0 && <span className="px-1.5 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-700 text-gray-500">{app.headers_injected} headers</span>}
                    </div>
                    <div className="mt-3 grid grid-cols-3 gap-2 text-center">
                      <div><p className="text-xs text-gray-400">QPS</p><p className="text-sm font-bold text-blue-600">{app.qps}</p></div>
                      <div><p className="text-xs text-gray-400">Latency</p><p className={"text-sm font-bold " + (app.avg_latency_ms < 100 ? "text-green-600" : app.avg_latency_ms < 300 ? "text-yellow-600" : "text-red-600")}>{app.avg_latency_ms}ms</p></div>
                      <div><p className="text-xs text-gray-400">Errors</p><p className={"text-sm font-bold " + (app.error_rate_pct < 1 ? "text-green-600" : app.error_rate_pct < 5 ? "text-yellow-600" : "text-red-600")}>{app.error_rate_pct}%</p></div>
                    </div>
                    <div className="mt-3 flex items-center justify-between text-xs text-gray-400">
                      <span>{app.policy_count} policies</span>
                      <span className="font-mono truncate max-w-[150px]">{app.upstream_url}</span>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </>
      )}

      {/* MONITORING TAB */}
      {tab === "monitor" && (
        <>
          <div className="flex items-center justify-between">
            <h2 className="flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Gauge className="h-4 w-4" /> Real-Time Metrics</h2>
            <button onClick={() => setLiveMode(!liveMode)} aria-pressed={liveMode} className={"flex items-center gap-2 rounded-lg px-3 py-1.5 text-sm font-medium " + (liveMode ? "bg-green-50 text-green-700 dark:bg-green-950/30 dark:text-green-400" : "border border-gray-300 text-gray-600 dark:border-gray-700 dark:text-gray-300")}>
              <Activity className={"h-4 w-4 " + (liveMode ? "animate-pulse text-green-500" : "")} /> {liveMode ? "Live (3s)" : "Paused"}
            </button>
          </div>
          {/* Metrics sparkline chart */}
          {metrics.length > 0 ? (
            <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
              {([
                { label: "QPS", key: "qps" as const, color: "text-blue-600", bg: "bg-blue-500" },
                { label: "Latency (ms)", key: "latency_ms" as const, color: "text-purple-600", bg: "bg-purple-500" },
                { label: "Error Rate (%)", key: "error_rate_pct" as const, color: "text-red-600", bg: "bg-red-500" },
              ]).map(m => {
                const values = metrics.map(p => p[m.key]);
                const max = Math.max(...values, 1);
                const latest = values[values.length - 1] || 0;
                return (
                  <div key={m.key} className={cardCls}>
                    <div className="flex items-center justify-between">
                      <span className="text-xs font-semibold uppercase text-gray-400">{m.label}</span>
                      <span className={"text-lg font-bold " + m.color}>{typeof latest === "number" ? latest.toFixed(1) : latest}</span>
                    </div>
                    <div className="mt-3 flex items-end gap-0.5 h-20">
                      {values.map((v, i) => (
                        <div key={i} className={"flex-1 rounded-t " + m.bg} style={{ height: `${(v / max) * 100}%`, opacity: 0.3 + (i / values.length) * 0.7 }} />
                      ))}
                    </div>
                  </div>
                );
              })}
            </div>
          ) : (
            <div className={cardCls}><div className="py-8 text-center"><Gauge className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">{liveMode ? "Waiting for metrics..." : "Enable Live mode to see real-time data."}</p></div></div>
          )}
        </>
      )}

      {/* ACCESS LOGS TAB */}
      {tab === "logs" && (
        <div className={cardCls}>
          <div className="mb-4 flex flex-wrap items-center gap-2">
            <div className="relative flex-1 min-w-[200px]">
              <Search className="absolute left-2 top-2.5 h-4 w-4 text-gray-400" />
              <input aria-label="Search logs" type="text" value={logFilter} onChange={e => setLogFilter(e.target.value)} placeholder="Search app/user/path..." className="w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 pl-8 pr-3 py-1.5 text-sm" />
            </div>
            <select aria-label="Filter decision" value={logDecisionFilter} onChange={e => setLogDecisionFilter(e.target.value)} className="rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-2 py-1.5 text-sm">
              <option value="all">All Decisions</option>
              <option value="allow">Allow</option>
              <option value="deny">Deny</option>
              <option value="stepup">Step-up</option>
            </select>
            <span className="text-xs text-gray-400">{filteredLogs.length} entries</span>
          </div>
          {filteredLogs.length === 0 ? <div className="py-8 text-center"><Terminal className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No access logs.</p></div> : (
            <div className="overflow-x-auto max-h-[500px] overflow-y-auto">
              <table className="w-full text-sm">
                <thead className="sticky top-0 bg-gray-50 dark:bg-gray-900/80 backdrop-blur"><tr>
                  <th scope="col" className="px-3 py-2 text-left font-medium text-xs">Time</th>
                  <th scope="col" className="px-3 py-2 text-left font-medium text-xs">App</th>
                  <th scope="col" className="px-3 py-2 text-left font-medium text-xs">User</th>
                  <th scope="col" className="px-3 py-2 text-left font-medium text-xs">Method</th>
                  <th scope="col" className="px-3 py-2 text-left font-medium text-xs">Path</th>
                  <th scope="col" className="px-3 py-2 text-center font-medium text-xs">Status</th>
                  <th scope="col" className="px-3 py-2 text-center font-medium text-xs">Decision</th>
                  <th scope="col" className="px-3 py-2 text-right font-medium text-xs">Latency</th>
                </tr></thead>
                <tbody className="divide-y dark:divide-gray-800">
                  {filteredLogs.map(l => { const dc = decisionConfig[l.decision] || decisionConfig.deny; const DIcon = dc.icon; return (
                    <tr key={l.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                      <td className="px-3 py-2 text-xs text-gray-400 whitespace-nowrap">{new Date(l.timestamp).toLocaleTimeString()}</td>
                      <td className="px-3 py-2 text-xs font-medium">{l.app_name}</td>
                      <td className="px-3 py-2 text-xs font-mono">{l.user_id}</td>
                      <td className="px-3 py-2"><span className={"px-1.5 py-0.5 rounded text-xs font-mono " + (l.method === "GET" ? "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400" : l.method === "POST" ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400" : "bg-gray-100 dark:bg-gray-800")}>{l.method}</span></td>
                      <td className="px-3 py-2 text-xs font-mono text-gray-500 max-w-[200px] truncate">{l.path}</td>
                      <td className="px-3 py-2 text-center"><span className={"text-xs font-mono font-bold " + (l.status_code < 300 ? "text-green-600" : l.status_code < 400 ? "text-blue-600" : l.status_code < 500 ? "text-yellow-600" : "text-red-600")}>{l.status_code}</span></td>
                      <td className="px-3 py-2 text-center"><span className={"inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-xs font-medium " + dc.color}><DIcon className="h-3 w-3" /> {l.decision}</span></td>
                      <td className="px-3 py-2 text-right text-xs font-mono">{l.latency_ms}ms</td>
                    </tr>
                  ); })}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}

      {/* POLICY TESTER TAB */}
      {tab === "tester" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Code className="h-4 w-4" /> Test Configuration</h2>
            <div className="space-y-3">
              <div><label className="text-sm font-medium">Application</label>
                <select aria-label="Test application" value={testAppId} onChange={e => setTestAppId(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">
                  <option value="">Select app...</option>
                  {apps.map(a => <option key={a.id} value={a.id}>{a.name} ({a.domain})</option>)}
                </select>
              </div>
              <div><label className="text-sm font-medium">User ID</label><input aria-label="Test user" type="text" value={testUserId} onChange={e => setTestUserId(e.target.value)} placeholder="user:alice" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div>
              <div className="grid grid-cols-2 gap-3">
                <div><label className="text-sm font-medium">Method</label><select aria-label="Test method" value={testMethod} onChange={e => setTestMethod(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">{["GET","POST","PUT","DELETE"].map(m => <option key={m} value={m}>{m}</option>)}</select></div>
                <div><label className="text-sm font-medium">Path</label><input aria-label="Test path" type="text" value={testPath} onChange={e => setTestPath(e.target.value)} placeholder="/api/data" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div>
              </div>
              <div><label className="text-sm font-medium">Context (JSON)</label><textarea aria-label="Test context" value={testContext} onChange={e => setTestContext(e.target.value)} rows={4} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 font-mono text-xs" /></div>
              <button onClick={runPolicyTest} disabled={!testAppId || !testUserId || testing} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{testing ? <Loader2 className="h-4 w-4 animate-spin" /> : <Zap className="h-4 w-4" />} Evaluate Policy</button>
            </div>
          </div>
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Eye className="h-4 w-4" /> Result</h2>
            {testResult ? (
              <div>
                <div className={"flex items-center gap-3 rounded-xl border-2 p-4 " + (testResult.decision === "allow" ? "border-green-300 bg-green-50 dark:border-green-700 dark:bg-green-950/30" : testResult.decision === "stepup" ? "border-yellow-300 bg-yellow-50 dark:border-yellow-700 dark:bg-yellow-950/30" : "border-red-300 bg-red-50 dark:border-red-700 dark:bg-red-950/30")}>
                  {testResult.decision === "allow" ? <CheckCircle className="h-8 w-8 text-green-500" /> : testResult.decision === "stepup" ? <Shield className="h-8 w-8 text-yellow-500" /> : <XCircle className="h-8 w-8 text-red-500" />}
                  <div><p className={"text-xl font-bold capitalize " + (testResult.decision === "allow" ? "text-green-700 dark:text-green-400" : testResult.decision === "stepup" ? "text-yellow-700 dark:text-yellow-400" : "text-red-700 dark:text-red-400")}>{testResult.decision}</p><p className="text-sm text-gray-500">{testResult.reason}</p></div>
                </div>
                {testResult.matched_policies?.length > 0 && (
                  <div className="mt-4"><p className="text-xs font-semibold uppercase text-gray-400 mb-2">Matched Policies</p><div className="space-y-1">{testResult.matched_policies.map((p, i) => <div key={i} className="flex items-center gap-2 text-xs"><ChevronRight className="h-3 w-3 text-gray-400" /><span className="font-mono">{p}</span></div>)}</div></div>
                )}
              </div>
            ) : <div className="py-12 text-center"><Code className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">Configure a test case and click Evaluate.</p></div>}
          </div>
        </div>
      )}

      </>)}

      {/* REGISTER APP WIZARD */}
      {showWizard && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowWizard(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 max-h-[90vh] w-full max-w-2xl overflow-y-auto rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            {/* Step indicator */}
            <div className="mb-6 flex items-center justify-between">
              <h3 className="text-lg font-semibold text-gray-900 dark:text-white">Register Protected Application</h3>
              <button onClick={() => setShowWizard(false)} aria-label="Close"><X className="h-5 w-5 text-gray-400" /></button>
            </div>
            <div className="mb-6 flex items-center gap-1">
              {wizardSteps.map((s, i) => (
                <div key={i} className="flex items-center gap-1 flex-1">
                  <div className={"flex h-7 w-7 items-center justify-center rounded-full text-xs font-bold " + (i <= wizardStep ? "bg-indigo-600 text-white" : "bg-gray-200 dark:bg-gray-700 text-gray-400")}>{i + 1}</div>
                  {i < wizardSteps.length - 1 && <div className={"h-0.5 flex-1 " + (i < wizardStep ? "bg-indigo-600" : "bg-gray-200 dark:bg-gray-700")} />}
                </div>
              ))}
            </div>
            {/* Step content */}
            <div className="min-h-[200px]">
              {wizardStep === 0 && <div className="space-y-3"><div><label className="text-sm font-medium">Application Name *</label><input aria-label="App name" type="text" placeholder="Internal Dashboard" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" autoFocus /></div><div><label className="text-sm font-medium">Domain *</label><input aria-label="Domain" type="text" placeholder="dashboard.internal.corp" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div><div><label className="text-sm font-medium">Auth Mode</label><select aria-label="Auth mode" className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm"><option value="oidc">OIDC</option><option value="oauth2">OAuth 2.0</option><option value="saml">SAML</option><option value="basic">Basic Auth</option></select></div></div>}
              {wizardStep === 1 && <div className="space-y-3"><div><label className="text-sm font-medium">Upstream URL *</label><input aria-label="Upstream URL" type="text" placeholder="http://localhost:3000" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div><div><label className="text-sm font-medium">TLS Verify</label><select aria-label="TLS verify" className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm"><option value="true">Verify (recommended)</option><option value="false">Skip (dev only)</option></select></div><div><label className="text-sm font-medium">Timeout (seconds)</label><input aria-label="Timeout" type="number" defaultValue={30} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div></div>}
              {wizardStep === 2 && <div><p className="text-sm text-gray-500 mb-3">Define ABAC conditions for access control. CEL expression preview will appear below.</p><div className="rounded-lg border dark:border-gray-700 p-3"><div className="space-y-2"><div className="flex items-center gap-2"><select aria-label="Attribute" className="rounded border dark:border-gray-700 dark:bg-gray-900 px-2 py-1 text-xs"><option>user.role</option><option>user.department</option><option>device.trust_level</option><option>request.ip_range</option><option>time.hour</option></select><select aria-label="Operator" className="rounded border dark:border-gray-700 dark:bg-gray-900 px-2 py-1 text-xs"><option value="eq">==</option><option value="ne">!=</option><option value="in">in</option><option value="contains">contains</option></select><input aria-label="Value" type="text" placeholder="admin" className="flex-1 rounded border dark:border-gray-700 dark:bg-gray-900 px-2 py-1 text-xs font-mono" /><button className="text-red-400"><X className="h-3.5 w-3.5" /></button></div></div><button className="mt-2 flex items-center gap-1 text-xs text-indigo-600 hover:underline"><Plus className="h-3 w-3" /> Add Condition</button></div><div className="mt-3 rounded-lg bg-gray-900 p-3"><p className="text-xs text-green-400 font-mono">user.role == "admin" && device.trust_level {">="} "trusted"</p></div></div>}
              {wizardStep === 3 && <div className="space-y-3"><div className="rounded-lg border dark:border-gray-700 p-3"><div className="flex items-center gap-2"><input aria-label="Header name" type="text" placeholder="X-Forwarded-User" className="flex-1 rounded border dark:border-gray-700 dark:bg-gray-900 px-2 py-1 text-xs font-mono" /><span className="text-gray-400 text-xs">:</span><input aria-label="Header value" type="text" placeholder="$user.id" className="flex-1 rounded border dark:border-gray-700 dark:bg-gray-900 px-2 py-1 text-xs font-mono" /><button className="text-red-400"><X className="h-3.5 w-3.5" /></button></div></div><button className="flex items-center gap-1 text-xs text-indigo-600 hover:underline"><Plus className="h-3 w-3" /> Add Header</button><p className="text-xs text-gray-400">Headers are injected into upstream requests after authentication.</p></div>}
              {wizardStep === 4 && <div className="space-y-3"><div><label className="text-sm font-medium">Health Check Path</label><input aria-label="Health check path" type="text" placeholder="/healthz" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div><div><label className="text-sm font-medium">Health Check Interval (seconds)</label><input aria-label="Health interval" type="number" defaultValue={10} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div><div className="flex items-center gap-2"><button className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-700"><Activity className="h-4 w-4 text-green-500" /> Test Connection</button></div></div>}
              {wizardStep === 5 && <div className="space-y-2"><p className="text-sm text-gray-500">Review your configuration:</p><div className="rounded-lg border dark:border-gray-700 p-4 text-sm space-y-1"><div><span className="text-gray-400">Name:</span> Internal Dashboard</div><div><span className="text-gray-400">Domain:</span> dashboard.internal.corp</div><div><span className="text-gray-400">Upstream:</span> http://localhost:3000</div><div><span className="text-gray-400">Auth:</span> OIDC</div></div></div>}
            </div>
            {/* Navigation */}
            <div className="mt-6 flex justify-between">
              <button onClick={() => setWizardStep(Math.max(0, wizardStep - 1))} disabled={wizardStep === 0} className="flex items-center gap-1 rounded-lg border border-gray-300 px-4 py-2 text-sm disabled:opacity-30 dark:border-gray-700"><ChevronLeft className="h-4 w-4" /> Back</button>
              {wizardStep < wizardSteps.length - 1 ? (
                <button onClick={() => setWizardStep(wizardStep + 1)} className="flex items-center gap-1 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700">Next <ChevronRight className="h-4 w-4" /></button>
              ) : (
                <button onClick={() => setShowWizard(false)} className="flex items-center gap-1 rounded-lg bg-green-600 px-4 py-2 text-sm font-medium text-white hover:bg-green-700"><CheckCircle className="h-4 w-4" /> Register</button>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

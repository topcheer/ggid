"use client";
import { useState, useCallback, useEffect } from "react";
import { Gauge, Loader2, AlertCircle, X, RefreshCw, Plus, Trash2, Check, CheckCircle, XCircle, Activity, Zap, Settings, Eye, TrendingUp, Clock, AlertTriangle, Globe, TestTube } from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface TenantUsage { tenant_id: string; tenant_name: string; requests_24h: number; limit_24h: number; rate_429: number; trend: number[]; }
interface RateRule { id: string; endpoint_pattern: string; rps: number; burst: number; strategy: "token_bucket" | "sliding_window"; enabled: boolean; }
interface Error429 { id: string; tenant: string; ip: string; endpoint: string; timestamp: string; retry_after: number; }
interface QuotaStatus { remaining: number; limit: number; reset_at: string; current_rps: number; }

const TEMPLATES = [
  { id: "strict", name: "Strict", desc: "10 rps, 20 burst — for sensitive APIs", rps: 10, burst: 20 },
  { id: "standard", name: "Standard", desc: "100 rps, 200 burst — balanced default", rps: 100, burst: 200 },
  { id: "relaxed", name: "Relaxed", desc: "1000 rps, 2000 burst — high-throughput", rps: 1000, burst: 2000 },
  { id: "unlimited", name: "Unlimited", desc: "No rate limiting", rps: 0, burst: 0 },
];

type Tab = "dashboard" | "rules" | "errors" | "templates" | "tester";

export default function RateLimitingPage() {
  const [tab, setTab] = useState<Tab>("dashboard");
  const [usage, setUsage] = useState<TenantUsage[]>([]);
  const [rules, setRules] = useState<RateRule[]>([]);
  const [errors429, setErrors429] = useState<Error429[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  // Tester
  const [testEndpoint, setTestEndpoint] = useState("/api/v1/users");
  const [quotaStatus, setQuotaStatus] = useState<QuotaStatus | null>(null);
  const [testing, setTesting] = useState(false);
  // Actions
  const [togglingId, setTogglingId] = useState<string | null>(null);
  const [showRuleForm, setShowRuleForm] = useState(false);
  const [rulePattern, setRulePattern] = useState("/api/v1/*");
  const [ruleRps, setRuleRps] = useState(100);
  const [ruleBurst, setRuleBurst] = useState(200);
  const [ruleStrategy, setRuleStrategy] = useState<"token_bucket" | "sliding_window">("token_bucket");
  const [saving, setSaving] = useState(false);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
      const [uRes, rRes, eRes] = await Promise.all([
        fetch("/api/v1/identity/tenants/rate-limits/usage", { headers: h }).catch(() => null),
        fetch("/api/v1/identity/tenants/rate-limits/rules", { headers: h }).catch(() => null),
        fetch("/api/v1/identity/tenants/rate-limits/429-errors", { headers: h }).catch(() => null),
      ]);
      if (uRes?.ok) { const d = await uRes.json(); setUsage(d.tenants || d.items || []); }
      if (rRes?.ok) { const d = await rRes.json(); setRules(d.rules || d.items || []); }
      if (eRes?.ok) { const d = await eRes.json(); setErrors429(d.errors || d.items || []); }
    } catch { setError("Failed to load rate limiting data"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const testQuota = async () => {
    setTesting(true); setQuotaStatus(null);
    try {
      const res = await fetch(`/api/v1/identity/tenants/rate-limits/check?endpoint=${encodeURIComponent(testEndpoint)}`, { headers: { ...authHeader(), "X-Tenant-ID": TENANT_ID } });
      if (res.ok) setQuotaStatus(await res.json());
    } catch { setError("Check failed"); }
    finally { setTesting(false); }
  };

  const toggleRule = async (id: string, enabled: boolean) => {
    setTogglingId(id);
    try {
      await fetch(`/api/v1/identity/tenants/rate-limits/rules/${id}`, { method: "PUT", headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID }, body: JSON.stringify({ enabled: !enabled }) });
      setRules(prev => prev.map(r => r.id === id ? { ...r, enabled: !enabled } : r));
    } catch { /* noop */ }
    finally { setTogglingId(null); }
  };

  const saveRule = async () => {
    setSaving(true);
    try {
      await fetch("/api/v1/identity/tenants/rate-limits/rules", { method: "POST", headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID }, body: JSON.stringify({ endpoint_pattern: rulePattern, rps: ruleRps, burst: ruleBurst, strategy: ruleStrategy }) });
      setShowRuleForm(false); setRulePattern("/api/v1/*"); loadData();
    } catch { setError("Failed to save rule"); }
    finally { setSaving(false); }
  };

  const applyTemplate = async (tmpl: typeof TEMPLATES[0]) => {
    try {
      await fetch("/api/v1/identity/tenants/rate-limits/rules", { method: "POST", headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID }, body: JSON.stringify({ endpoint_pattern: "/api/v1/*", rps: tmpl.rps, burst: tmpl.burst, strategy: "token_bucket" }) });
      loadData();
    } catch { /* noop */ }
  };

  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
          <Gauge className="h-6 w-6 text-indigo-500" /> Per-Tenant Rate Limiting
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Token bucket + sliding window rate limiting with glob patterns, 429 monitoring, and templates.
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
          { id: "dashboard" as Tab, label: "Dashboard", icon: Gauge },
          { id: "rules" as Tab, label: "Rules", icon: Settings },
          { id: "errors" as Tab, label: "429 Monitor", icon: AlertTriangle },
          { id: "templates" as Tab, label: "Templates", icon: Zap },
          { id: "tester" as Tab, label: "Tester", icon: TestTube },
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

      {/* DASHBOARD */}
      {tab === "dashboard" && (
        <div className="space-y-4">
          {usage.length === 0 ? (
            <div className={card}><div className="py-8 text-center"><Gauge className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No usage data available.</p></div></div>
          ) : usage.map(u => {
            const pct = u.limit_24h ? Math.round((u.requests_24h / u.limit_24h) * 100) : 0;
            return (
              <div key={u.tenant_id} className={card}>
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <Globe className="h-5 w-5 text-gray-400" />
                    <div>
                      <p className="font-semibold text-sm">{u.tenant_name || u.tenant_id}</p>
                      <p className="text-xs text-gray-400">{u.requests_24h.toLocaleString()} / {u.limit_24h.toLocaleString()} requests (24h)</p>
                    </div>
                  </div>
                  <div className="flex items-center gap-3">
                    {u.rate_429 > 0 && <span className="px-2 py-0.5 rounded text-xs bg-red-100 dark:bg-red-900/30 text-red-600">{u.rate_429} throttled</span>}
                    <span className={`text-lg font-bold ${pct >= 90 ? "text-red-600" : pct >= 70 ? "text-yellow-600" : "text-green-600"}`}>{pct}%</span>
                  </div>
                </div>
                <div className="mt-2 h-2 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
                  <div className={`h-full rounded-full ${pct >= 90 ? "bg-red-500" : pct >= 70 ? "bg-yellow-500" : "bg-green-500"}`} style={{ width: `${Math.min(pct, 100)}%` }} />
                </div>
                {u.trend?.length > 0 && (
                  <div className="mt-3 flex items-end gap-0.5 h-16">
                    {u.trend.map((v: any, i: number) => <div key={i} className="flex-1 rounded-t bg-indigo-400 opacity-70" style={{ height: `${Math.min(v, 100)}%` }} />)}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}

      {/* RULES */}
      {tab === "rules" && (
        <div className={card}>
          <div className="mb-4 flex items-center justify-between">
            <h2 className="flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Settings className="h-4 w-4" /> Rate Limit Rules</h2>
            <button onClick={() => setShowRuleForm(true)} className="flex items-center gap-1 rounded-lg bg-indigo-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-indigo-700"><Plus className="h-3 w-3" /> Add Rule</button>
          </div>
          {rules.length === 0 ? (
            <div className="py-8 text-center"><Settings className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No rate limit rules configured.</p></div>
          ) : (
            <div className="space-y-2">{rules.map(r => (
              <div key={r.id} className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700">
                <div className="flex items-center gap-3">
                  <code className="text-xs font-mono text-blue-600 dark:text-blue-400">{r.endpoint_pattern}</code>
                  <span className="px-1.5 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-700">{r.strategy}</span>
                </div>
                <div className="flex items-center gap-3">
                  <span className="text-xs text-gray-400">{r.rps} rps / {r.burst} burst</span>
                  <button onClick={() => toggleRule(r.id, r.enabled)} disabled={togglingId === r.id} aria-pressed={r.enabled}
                    className={`rounded-lg px-2 py-1 text-xs font-medium ${r.enabled ? "bg-green-50 text-green-700 dark:bg-green-950/20" : "bg-gray-100 dark:bg-gray-800 text-gray-400"}`}>
                    {togglingId === r.id ? <Loader2 className="h-3 w-3 animate-spin" /> : r.enabled ? "Active" : "Disabled"}
                  </button>
                  <button className="text-red-400"><Trash2 className="h-3.5 w-3.5" /></button>
                </div>
              </div>
            ))}</div>
          )}
        </div>
      )}

      {/* 429 MONITOR */}
      {tab === "errors" && (
        <div className={card}>
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><AlertTriangle className="h-4 w-4" /> Throttled Requests (429)</h2>
          {errors429.length === 0 ? (
            <div className="py-8 text-center"><CheckCircle className="mx-auto h-10 w-10 text-green-300" /><p className="mt-3 text-sm text-gray-400">No throttled requests in the last 24h.</p></div>
          ) : (
            <div className="overflow-x-auto max-h-[400px] overflow-y-auto"><table className="w-full text-sm">
              <thead className="sticky top-0 bg-gray-50 dark:bg-gray-900/80"><tr>
                <th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">Tenant</th>
                <th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">IP</th>
                <th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">Endpoint</th>
                <th scope="col" className="px-3 py-2 text-right text-xs font-medium text-gray-400">Retry-After</th>
                <th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">Time</th>
              </tr></thead>
              <tbody className="divide-y dark:divide-gray-800">{errors429.map(e => (
                <tr key={e.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                  <td className="px-3 py-2 text-xs">{e.tenant}</td>
                  <td className="px-3 py-2 text-xs font-mono">{e.ip}</td>
                  <td className="px-3 py-2 text-xs font-mono text-blue-600 dark:text-blue-400">{e.endpoint}</td>
                  <td className="px-3 py-2 text-right text-xs">{e.retry_after}s</td>
                  <td className="px-3 py-2 text-xs text-gray-500">{new Date(e.timestamp).toLocaleTimeString()}</td>
                </tr>
              ))}</tbody>
            </table></div>
          )}
        </div>
      )}

      {/* TEMPLATES */}
      {tab === "templates" && (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          {TEMPLATES.map(tmpl => (
            <div key={tmpl.id} className={card + " hover:shadow-md transition"}>
              <div className="flex items-start justify-between">
                <div className="flex items-center gap-3">
                  <div className={`flex h-10 w-10 items-center justify-center rounded-lg ${tmpl.id === "strict" ? "bg-red-100 dark:bg-red-900/30" : tmpl.id === "unlimited" ? "bg-gray-100 dark:bg-gray-700" : "bg-indigo-100 dark:bg-indigo-900/30"}`}>
                    <Zap className={`h-5 w-5 ${tmpl.id === "strict" ? "text-red-500" : tmpl.id === "unlimited" ? "text-gray-400" : "text-indigo-500"}`} />
                  </div>
                  <div>
                    <h3 className="font-semibold text-sm">{tmpl.name}</h3>
                    <p className="text-xs text-gray-400">{tmpl.desc}</p>
                  </div>
                </div>
              </div>
              <div className="mt-3 grid grid-cols-2 gap-2 text-center">
                <div className="rounded-lg border p-2 dark:border-gray-700"><p className="text-xs text-gray-400">RPS</p><p className="text-lg font-bold">{tmpl.rps || "∞"}</p></div>
                <div className="rounded-lg border p-2 dark:border-gray-700"><p className="text-xs text-gray-400">Burst</p><p className="text-lg font-bold">{tmpl.burst || "∞"}</p></div>
              </div>
              <button onClick={() => applyTemplate(tmpl)} className="mt-3 w-full rounded-lg border border-indigo-200 px-3 py-2 text-xs font-medium text-indigo-700 hover:bg-indigo-50 dark:border-indigo-800 dark:text-indigo-400 dark:hover:bg-indigo-950/30">Apply to /api/v1/*</button>
            </div>
          ))}
        </div>
      )}

      {/* TESTER */}
      {tab === "tester" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><TestTube className="h-4 w-4" /> Quota Check</h2>
            <div className="space-y-3">
              <div>
                <label className="text-sm font-medium">Endpoint</label>
                <input aria-label="Endpoint" type="text" value={testEndpoint} onChange={e => setTestEndpoint(e.target.value)} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" />
              </div>
              <button onClick={testQuota} disabled={testing} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">
                {testing ? <Loader2 className="h-4 w-4 animate-spin" /> : <Eye className="h-4 w-4" />} Check Quota
              </button>
            </div>
          </div>
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Gauge className="h-4 w-4" /> Current Status</h2>
            {quotaStatus ? (
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <span className="text-sm text-gray-500">Remaining</span>
                  <span className="text-2xl font-bold text-indigo-600">{quotaStatus.remaining} / {quotaStatus.limit}</span>
                </div>
                <div className="h-3 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
                  <div className="h-full rounded-full bg-indigo-500" style={{ width: `${quotaStatus.limit ? (quotaStatus.remaining / quotaStatus.limit) * 100 : 0}%` }} />
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <div className="rounded-lg border p-3 dark:border-gray-700"><span className="text-xs text-gray-400">Current RPS</span><p className="text-lg font-bold">{quotaStatus.current_rps}</p></div>
                  <div className="rounded-lg border p-3 dark:border-gray-700"><span className="text-xs text-gray-400">Reset At</span><p className="text-sm">{new Date(quotaStatus.reset_at).toLocaleTimeString()}</p></div>
                </div>
              </div>
            ) : (
              <div className="py-8 text-center"><TestTube className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">Enter an endpoint to check quota.</p></div>
            )}
          </div>
        </div>
      )}

      </>)}

      {/* Rule form dialog */}
      {showRuleForm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowRuleForm(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <h3 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white"><Plus className="h-5 w-5 text-indigo-500" /> Add Rate Limit Rule</h3>
            <div className="mt-4 space-y-3">
              <div>
                <label className="text-sm font-medium">Endpoint Pattern (glob)</label>
                <input aria-label="Pattern" type="text" value={rulePattern} onChange={e => setRulePattern(e.target.value)} placeholder="/api/v1/users/*" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-sm font-medium">RPS</label>
                  <input aria-label="RPS" type="number" min={0} value={ruleRps} onChange={e => setRuleRps(parseInt(e.target.value) || 0)} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" />
                </div>
                <div>
                  <label className="text-sm font-medium">Burst</label>
                  <input aria-label="Burst" type="number" min={0} value={ruleBurst} onChange={e => setRuleBurst(parseInt(e.target.value) || 0)} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" />
                </div>
              </div>
              <div>
                <label className="text-sm font-medium">Strategy</label>
                <select aria-label="Strategy" value={ruleStrategy} onChange={e => setRuleStrategy(e.target.value as "token_bucket" | "sliding_window")} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">
                  <option value="token_bucket">Token Bucket</option>
                  <option value="sliding_window">Sliding Window</option>
                </select>
              </div>
            </div>
            <div className="mt-4 flex justify-end gap-2">
              <button onClick={() => setShowRuleForm(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">Cancel</button>
              <button onClick={saveRule} disabled={saving} className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{saving ? <Loader2 className="h-4 w-4 animate-spin" /> : "Save"}</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

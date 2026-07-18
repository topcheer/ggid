"use client";
import { useState, useCallback, useEffect } from "react";
import {
  ShieldAlert, Loader2, AlertCircle, X, RefreshCw, Plus, Trash2, Check,
  Globe, Search, Crosshair, Activity, Database, Zap, Eye, ChevronRight,
  TrendingUp, AlertTriangle, ExternalLink, Lock, Ban, Clock, Cpu,
  Server, Hash, Mail, Fingerprint, Radar,
} from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";
import { useTranslations } from "@/lib/i18n";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

/* ─── Types ─── */
interface IntelSource {
  source_name: string; source_type: string; status: string;
  last_sync: string; indicators_imported: number;
}
interface ThreatIndicator {
  type: string; value: string; confidence: number;
  tags: string[]; first_seen: string;
}
interface AutoBlockRule {
  rule_id: string; condition: string; action: string; enabled: boolean;
}
interface ThreatFeedResult {
  intel_sources: IntelSource[];
  indicators: ThreatIndicator[];
  auto_block_rules: AutoBlockRule[];
  total_indicators: number;
  generated_at: string;
}
interface ThreatEvent {
  event_type: string; severity: string; indicators: string[];
  source_ip: string; user_agent: string; tenant_id: string; timestamp: string;
}
interface ITDRIncident {
  id: string; name: string; severity: string; status: string;
  user_id: string; source: string; signals: string[];
  created_at: string; rule_ids: string[];
}

type Tab = "sources" | "indicators" | "checker" | "itdr" | "stats";

const SEVERITY_CFG: Record<string, { label: string; color: string; bg: string }> = {
  critical: { label: "Critical", color: "text-red-600", bg: "bg-red-100 dark:bg-red-900/30" },
  high: { label: "High", color: "text-orange-600", bg: "bg-orange-100 dark:bg-orange-900/30" },
  medium: { label: "Medium", color: "text-yellow-600", bg: "bg-yellow-100 dark:bg-yellow-900/30" },
  low: { label: "Low", color: "text-blue-600", bg: "bg-blue-100 dark:bg-blue-900/30" },
};

const INDICATOR_ICONS: Record<string, typeof Globe> = {
  ip: Globe, domain: Globe, hash: Hash, email: Mail,
  user_agent: Cpu, url: ExternalLink, file: Fingerprint,
};

const SOURCE_TYPE_LABELS: Record<string, string> = {
  threat_feed: "Threat Feed", ip_reputation: "IP Reputation",
  breach_data: "Breach Data", "stix/taxii": "STIX/TAXII", custom: "Custom",
};

export default function ThreatIntelPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("sources");
  const [feed, setFeed] = useState<ThreatFeedResult | null>(null);
  const [threatEvents, setThreatEvents] = useState<ThreatEvent[]>([]);
  const [itdrIncidents, setITDRIncidents] = useState<ITDRIncident[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Sources form
  const [showSourceForm, setShowSourceForm] = useState(false);
  const [srcName, setSrcName] = useState("");
  const [srcType, setSrcType] = useState("threat_feed");
  const [srcKey, setSrcKey] = useState("");
  const [srcInterval, setSrcInterval] = useState(60);

  // Indicators filter
  const [indFilter, setIndFilter] = useState("all");
  const [indSearch, setIndSearch] = useState("");

  // Checker
  const [chkType, setChkType] = useState("ip");
  const [chkValue, setChkValue] = useState("");
  const [chkResult, setChkResult] = useState<{ found: boolean; risk_score: number; matches: { source: string; confidence: number; tags: string[] }[] } | null>(null);
  const [checking, setChecking] = useState(false);

  const H = { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID };
  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
      const [fRes, tRes, iRes] = await Promise.all([
        fetch("/api/v1/auth/threat-intel/feed", { headers: h }).catch(() => null),
        fetch("/api/v1/audit/threat-feed", { headers: h }).catch(() => null),
        fetch("/api/v1/audit/itdr/incidents", { headers: h }).catch(() => null),
      ]);
      if (fRes?.ok) setFeed(await fRes.json());
      if (tRes?.ok) { const d = await tRes.json(); setThreatEvents(d.events || []); }
      if (iRes?.ok) { const d = await iRes.json(); setITDRIncidents(d.itdrIncidents || d.incidents || []); }
      setError(null);
    } catch { setError("Failed to load threat intelligence data"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const runCheck = async () => {
    if (!chkValue) return;
    setChecking(true); setChkResult(null);
    try {
      // Check against loaded indicators
      const matches: { source: string; confidence: number; tags: string[] }[] = [];
      const indicators = feed?.indicators || [];
      for (const ind of indicators) {
        if (ind.type === chkType && ind.value.toLowerCase().includes(chkValue.toLowerCase())) {
          matches.push({ source: "Local Feed", confidence: ind.confidence, tags: ind.tags });
        }
      }
      // Also try backend check endpoint
      const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
      const res = await fetch("/api/v1/audit/threat-feed?since=2000-01-01T00:00:00Z", { headers: h }).catch(() => null);
      if (res?.ok) {
        const d = await res.json();
        for (const ev of (d.events || []) as ThreatEvent[]) {
          if (ev.source_ip && ev.source_ip.includes(chkValue)) {
            matches.push({ source: "SIEM Feed", confidence: 0.85, tags: ev.indicators });
          }
        }
      }
      const riskScore = matches.length > 0
        ? Math.min(Math.round(matches.reduce((a: any, m: any) => a + m.confidence, 0) / matches.length * 100), 100)
        : 5;
      setChkResult({ found: matches.length > 0, risk_score: riskScore, matches });
    } catch { setError("Threat check failed"); }
    finally { setChecking(false); }
  };

  const filteredIndicators = (feed?.indicators || []).filter(ind => {
    if (indFilter !== "all" && ind.type !== indFilter) return false;
    if (indSearch && !ind.value.toLowerCase().includes(indSearch.toLowerCase()) && !ind.tags.some(t => t.includes(indSearch.toLowerCase()))) return false;
    return true;
  });

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
          <Radar className="h-6 w-6 text-rose-500" /> {t("threatIntel.title")}
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {t("threatIntel.subtitle")}
        </p>
      </div>

      {error && (
        <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}
          <button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      {/* Tabs */}
      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "sources" as Tab, label: t("threatIntel.sources"), icon: Database },
          { id: "indicators" as Tab, label: t("threatIntel.indicators"), icon: Crosshair },
          { id: "checker" as Tab, label: t("threatIntel.checker"), icon: Search },
          { id: "itdr" as Tab, label: t("threatIntel.itdr"), icon: Radar },
          { id: "stats" as Tab, label: t("threatIntel.statistics"), icon: TrendingUp },
        ]).map(tb => {
          const Icon = tb.icon;
          return (
            <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id}
              className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === tb.id ? "border-rose-600 text-rose-600 dark:text-rose-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}>
              <Icon className="h-4 w-4" /> {tb.label}
            </button>
          );
        })}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-rose-500" /></div> : (<>

      {/* ════ SOURCES ════ */}
      {tab === "sources" && (
        <div>
          <div className="mb-4 flex items-center justify-between">
            <h2 className="flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Database className="h-4 w-4" /> Intelligence Sources ({feed?.intel_sources?.length ?? 0})</h2>
            <button onClick={() => setShowSourceForm(true)} className="flex items-center gap-1 rounded-lg bg-rose-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-rose-700">
              <Plus className="h-3 w-3" /> Add Source
            </button>
          </div>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {feed?.intel_sources?.map(s => (
              <div key={s.source_name} className={card + " hover:shadow-md transition"}>
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-3">
                    <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-gray-100 dark:bg-gray-700"><Database className="h-5 w-5 text-rose-400" /></div>
                    <div>
                      <h3 className="font-semibold text-sm">{s.source_name}</h3>
                      <p className="text-xs text-gray-400">{SOURCE_TYPE_LABELS[s.source_type] || s.source_type}</p>
                    </div>
                  </div>
                  <span className={`flex items-center gap-1 px-1.5 py-0.5 rounded text-xs font-medium ${s.status === "active" ? "bg-green-100 dark:bg-green-900/30 text-green-600" : s.status === "syncing" ? "bg-blue-100 dark:bg-blue-900/30 text-blue-600 animate-pulse" : "bg-gray-100 dark:bg-gray-800 text-gray-400"}`}>
                    {s.status === "active" && <Check className="h-2.5 w-2.5" />} {s.status}
                  </span>
                </div>
                <div className="mt-3 grid grid-cols-2 gap-2 text-center">
                  <div><p className="text-xs text-gray-400">Indicators</p><p className="text-sm font-bold">{s.indicators_imported.toLocaleString()}</p></div>
                  <div><p className="text-xs text-gray-400">Last Sync</p><p className="text-xs font-mono">{s.last_sync ? new Date(s.last_sync).toLocaleTimeString() : "—"}</p></div>
                </div>
              </div>
            ))}
          </div>

          {/* Auto-block rules */}
          {feed?.auto_block_rules && feed.auto_block_rules.length > 0 && (
            <div className="mt-6">
              <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Ban className="h-4 w-4" /> Auto-Block Rules</h3>
              <div className="space-y-2">
                {feed.auto_block_rules.map(r => (
                  <div key={r.rule_id} className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700">
                    <div className="flex items-center gap-3">
                      <code className="text-xs font-mono text-gray-500">{r.rule_id}</code>
                      <span className="text-sm font-medium">{r.condition}</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="px-2 py-0.5 rounded text-xs bg-rose-100 dark:bg-rose-900/30 text-rose-600 font-mono">{r.action}</span>
                      <span className={`px-1.5 py-0.5 rounded text-xs ${r.enabled ? "bg-green-100 dark:bg-green-900/30 text-green-600" : "bg-gray-100 dark:bg-gray-800 text-gray-400"}`}>{r.enabled ? "on" : "off"}</span>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      {/* ════ INDICATORS ════ */}
      {tab === "indicators" && (
        <div className={card}>
          <div className="mb-4 flex items-center justify-between gap-4 flex-wrap">
            <h2 className="flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Crosshair className="h-4 w-4" /> Threat Indicators ({filteredIndicators.length})</h2>
            <div className="flex items-center gap-2">
              <select aria-label="Filter type" value={indFilter} onChange={e => setIndFilter(e.target.value)} className="rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-2 py-1.5 text-sm">
                <option value="all">All Types</option>
                <option value="ip">IP</option>
                <option value="domain">Domain</option>
                <option value="email">Email</option>
                <option value="hash">Hash</option>
                <option value="user_agent">User Agent</option>
                <option value="url">URL</option>
              </select>
              <div className="relative">
                <Search className="absolute left-2 top-2 h-4 w-4 text-gray-400" />
                <input type="text" value={indSearch} onChange={e => setIndSearch(e.target.value)} placeholder="Search..." className="rounded-lg border dark:border-gray-700 dark:bg-gray-900 pl-8 pr-3 py-1.5 text-sm w-48" />
              </div>
            </div>
          </div>
          {filteredIndicators.length === 0 ? (
            <div className="py-8 text-center"><Crosshair className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No indicators found.</p></div>
          ) : (
            <div className="overflow-x-auto max-h-[500px] overflow-y-auto">
              <table className="w-full text-sm">
                <thead className="sticky top-0 bg-gray-50 dark:bg-gray-900/80"><tr>
                  <th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">Type</th>
                  <th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">Value</th>
                  <th scope="col" className="px-3 py-2 text-center text-xs font-medium text-gray-400">Confidence</th>
                  <th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">Tags</th>
                  <th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">First Seen</th>
                </tr></thead>
                <tbody className="divide-y dark:divide-gray-800">
                  {filteredIndicators.map((ind: any, i: number) => {
                    const IIcon = INDICATOR_ICONS[ind.type] || Globe;
                    const pct = Math.round(ind.confidence * 100);
                    return (
                      <tr key={i} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                        <td className="px-3 py-2"><span className="flex items-center gap-1.5 text-xs"><IIcon className="h-3.5 w-3.5 text-gray-400" /> {ind.type}</span></td>
                        <td className="px-3 py-2 text-xs font-mono break-all max-w-xs">{ind.value}</td>
                        <td className="px-3 py-2 text-center">
                          <span className={`text-xs font-bold ${pct >= 90 ? "text-red-600" : pct >= 75 ? "text-orange-600" : pct >= 50 ? "text-yellow-600" : "text-blue-600"}`}>{pct}%</span>
                        </td>
                        <td className="px-3 py-2"><div className="flex flex-wrap gap-1">{ind.tags?.map(t => <span key={t} className="px-1.5 py-0.5 rounded bg-rose-100 dark:bg-rose-900/20 text-rose-600 text-xs font-mono">{t}</span>)}</div></td>
                        <td className="px-3 py-2 text-xs text-gray-400">{ind.first_seen ? new Date(ind.first_seen).toLocaleDateString() : "—"}</td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}

      {/* ════ CHECKER ════ */}
      {tab === "checker" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Search className="h-4 w-4" /> Threat Lookup</h2>
            <div className="space-y-3">
              <div>
                <label className="text-sm font-medium">Indicator Type</label>
                <select value={chkType} onChange={e => setChkType(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">
                  <option value="ip">IP Address</option>
                  <option value="domain">Domain</option>
                  <option value="email">Email</option>
                  <option value="hash">File Hash</option>
                  <option value="user_agent">User Agent</option>
                  <option value="url">URL</option>
                </select>
              </div>
              <div>
                <label className="text-sm font-medium">Value</label>
                <input type="text" value={chkValue} onChange={e => setChkValue(e.target.value)} placeholder="203.0.113.50" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" />
              </div>
              <button onClick={runCheck} disabled={!chkValue || checking}
                className="flex items-center gap-2 rounded-lg bg-rose-600 px-4 py-2 text-sm font-medium text-white hover:bg-rose-700 disabled:opacity-50">
                {checking ? <Loader2 className="h-4 w-4 animate-spin" /> : <Crosshair className="h-4 w-4" />} Check All Sources
              </button>
            </div>
          </div>
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Radar className="h-4 w-4" /> Result</h2>
            {chkResult ? (
              <div>
                <div className={`flex items-center gap-3 rounded-xl border-2 p-4 ${chkResult.risk_score >= 70 ? "border-red-300 bg-red-50 dark:border-red-700 dark:bg-red-950/30" : chkResult.risk_score >= 40 ? "border-yellow-300 bg-yellow-50 dark:border-yellow-700 dark:bg-yellow-950/30" : "border-green-300 bg-green-50 dark:border-green-700 dark:bg-green-950/30"}`}>
                  {chkResult.risk_score >= 70 ? <AlertTriangle className="h-8 w-8 text-red-500" /> : chkResult.risk_score >= 40 ? <AlertCircle className="h-8 w-8 text-yellow-500" /> : <Check className="h-8 w-8 text-green-500" />}
                  <div>
                    <p className={`text-lg font-bold ${chkResult.risk_score >= 70 ? "text-red-700 dark:text-red-400" : chkResult.risk_score >= 40 ? "text-yellow-700 dark:text-yellow-400" : "text-green-700 dark:text-green-400"}`}>
                      Risk Score: {chkResult.risk_score}/100
                    </p>
                    <p className="text-xs text-gray-500 dark:text-gray-400">
                      {chkResult.found ? `${chkResult.matches.length} source(s) matched` : "No matches across all sources"}
                    </p>
                  </div>
                </div>
                {chkResult.matches.length > 0 && (
                  <div className="mt-3 space-y-2">
                    {chkResult.matches.map((m: any, i: number) => (
                      <div key={i} className="rounded-lg border p-3 dark:border-gray-700">
                        <div className="flex items-center justify-between">
                          <span className="text-sm font-medium">{m.source}</span>
                          <span className={`text-sm font-bold ${m.confidence >= 0.9 ? "text-red-600" : m.confidence >= 0.7 ? "text-orange-600" : "text-yellow-600"}`}>{Math.round(m.confidence * 100)}%</span>
                        </div>
                        {m.tags?.length > 0 && (
                          <div className="mt-1 flex flex-wrap gap-1">{m.tags.map(t => <span key={t} className="px-1.5 py-0.5 rounded bg-rose-100 dark:bg-rose-900/20 text-rose-600 text-xs font-mono">{t}</span>)}</div>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            ) : (
              <div className="py-8 text-center"><Search className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">Enter an indicator to check across all sources.</p></div>
            )}
          </div>
        </div>
      )}

      {/* ════ ITDR CORRELATION ════ */}
      {tab === "itdr" && (
        <div className="space-y-6">
          {/* Threat Events from SIEM Feed */}
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Activity className="h-4 w-4" /> External Intel Hits (SIEM Feed)</h2>
            {threatEvents.length === 0 ? (
              <div className="py-6 text-center"><Activity className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No external threat events.</p></div>
            ) : (
              <div className="space-y-2">
                {threatEvents.map((ev: any, i: number) => {
                  const cfg = SEVERITY_CFG[ev.severity] || SEVERITY_CFG.medium;
                  return (
                    <div key={i} className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700">
                      <div className="flex items-center gap-3">
                        <div className={`flex h-8 w-8 items-center justify-center rounded-lg ${cfg.bg}`}><AlertTriangle className={`h-4 w-4 ${cfg.color}`} /></div>
                        <div>
                          <div className="flex items-center gap-2">
                            <span className="text-sm font-medium">{ev.event_type.replace(/_/g, " ")}</span>
                            <span className={`px-1.5 py-0.5 rounded text-xs font-medium ${cfg.bg} ${cfg.color}`}>{cfg.label}</span>
                          </div>
                          <p className="text-xs text-gray-400">
                            IP: <span className="font-mono">{ev.source_ip}</span> · UA: <span className="font-mono">{ev.user_agent}</span>
                          </p>
                        </div>
                      </div>
                      <div className="text-right">
                        <div className="flex flex-wrap gap-1 justify-end max-w-xs">
                          {ev.indicators?.map(ind => <span key={ind} className="px-1 py-0.5 rounded bg-rose-100 dark:bg-rose-900/20 text-rose-600 text-xs font-mono">{ind}</span>)}
                        </div>
                        <p className="text-xs text-gray-400 mt-1">{new Date(ev.timestamp).toLocaleString()}</p>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>

          {/* ITDR Incidents Cross-Reference */}
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Radar className="h-4 w-4" /> ITDR Incidents (Internal Detection × External Intel)</h2>
            {itdrIncidents.length === 0 ? (
              <div className="py-6 text-center"><Radar className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No active ITDR incidents.</p></div>
            ) : (
              <div className="space-y-2">
                {itdrIncidents.map(inc => {
                  const cfg = SEVERITY_CFG[inc.severity] || SEVERITY_CFG.medium;
                  return (
                    <div key={inc.id} className="rounded-lg border p-3 dark:border-gray-700">
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-3">
                          <div className={`flex h-8 w-8 items-center justify-center rounded-lg ${cfg.bg}`}><Radar className={`h-4 w-4 ${cfg.color}`} /></div>
                          <div>
                            <div className="flex items-center gap-2">
                              <span className="text-sm font-medium">{inc.name}</span>
                              <span className={`px-1.5 py-0.5 rounded text-xs font-medium ${cfg.bg} ${cfg.color}`}>{cfg.label}</span>
                              <span className="px-1.5 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-700">{inc.status}</span>
                            </div>
                            <p className="text-xs text-gray-400">User: <span className="font-mono">{inc.user_id}</span> · Source: {inc.source}</p>
                          </div>
                        </div>
                        <span className="text-xs text-gray-400">{new Date(inc.created_at).toLocaleString()}</span>
                      </div>
                      {inc.signals?.length > 0 && (
                        <div className="mt-2 flex flex-wrap gap-1">
                          {inc.signals.map(s => <span key={s} className="px-1.5 py-0.5 rounded bg-rose-100 dark:bg-rose-900/20 text-rose-600 text-xs font-mono">{s}</span>)}
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        </div>
      )}

      {/* ════ STATS ════ */}
      {tab === "stats" && (
        <div className="space-y-6">
          {/* KPI cards */}
          <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
            <div className={card + " text-center"}>
              <Database className="mx-auto h-5 w-5 text-rose-400" />
              <p className="mt-2 text-2xl font-bold">{feed?.total_indicators?.toLocaleString() ?? 0}</p>
              <p className="text-xs text-gray-400">Total Indicators</p>
            </div>
            <div className={card + " text-center"}>
              <Database className="mx-auto h-5 w-5 text-green-400" />
              <p className="mt-2 text-2xl font-bold">{feed?.intel_sources?.filter(s => s.status === "active").length ?? 0}</p>
              <p className="text-xs text-gray-400">Active Sources</p>
            </div>
            <div className={card + " text-center"}>
              <Activity className="mx-auto h-5 w-5 text-blue-400" />
              <p className="mt-2 text-2xl font-bold">{threatEvents.length}</p>
              <p className="text-xs text-gray-400">Threat Events 1h</p>
            </div>
            <div className={card + " text-center"}>
              <Radar className="mx-auto h-5 w-5 text-orange-400" />
              <p className="mt-2 text-2xl font-bold">{itdrIncidents.length}</p>
              <p className="text-xs text-gray-400">ITDR Incidents</p>
            </div>
          </div>

          {/* Source coverage */}
          <div className={card}>
            <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Database className="h-4 w-4" /> Source Coverage</h3>
            <div className="space-y-3">
              {feed?.intel_sources?.map(s => {
                const total = feed.total_indicators || 1;
                const pct = Math.round((s.indicators_imported / total) * 100);
                return (
                  <div key={s.source_name}>
                    <div className="flex items-center justify-between mb-1">
                      <span className="text-sm font-medium">{s.source_name}</span>
                      <span className="text-xs text-gray-400">{s.indicators_imported.toLocaleString()} ({pct}%)</span>
                    </div>
                    <div className="h-2 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
                      <div className={`h-full rounded-full ${s.status === "active" ? "bg-green-500" : s.status === "syncing" ? "bg-blue-500" : "bg-gray-400"}`} style={{ width: `${Math.max(pct, 2)}%` }} />
                    </div>
                  </div>
                );
              })}
            </div>
          </div>

          {/* Indicator type distribution */}
          <div className={card}>
            <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Crosshair className="h-4 w-4" /> Indicator Types</h3>
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
              {Object.entries(
                (feed?.indicators || []).reduce((acc: any, ind: any) => { acc[ind.type] = (acc[ind.type] || 0) + 1; return acc; }, {} as Record<string, number>)
              ).map(([type, count]: any[]) => {
                const IIcon = INDICATOR_ICONS[type] || Globe;
                return (
                  <div key={type} className="rounded-lg border p-3 text-center dark:border-gray-700">
                    <IIcon className="mx-auto h-4 w-4 text-gray-400" />
                    <p className="mt-1 text-lg font-bold">{count}</p>
                    <p className="text-xs text-gray-400">{type}</p>
                  </div>
                );
              })}
            </div>
          </div>

          {/* Top threats by confidence */}
          <div className={card}>
            <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><TrendingUp className="h-4 w-4" /> Top Threats (by Confidence)</h3>
            <div className="space-y-2">
              {(feed?.indicators || []).sort((a: any, b: any) => b.confidence - a.confidence).slice(0, 10).map((ind: any, i: number) => {
                const pct = Math.round(ind.confidence * 100);
                return (
                  <div key={i} className="flex items-center gap-3">
                    <span className="w-6 text-xs text-gray-400">#{i + 1}</span>
                    <span className="w-16 text-xs text-gray-400 font-mono">{ind.type}</span>
                    <code className="flex-1 text-xs font-mono truncate">{ind.value}</code>
                    <div className="w-24 h-2 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
                      <div className={`h-full rounded-full ${pct >= 90 ? "bg-red-500" : pct >= 75 ? "bg-orange-500" : "bg-yellow-500"}`} style={{ width: `${pct}%` }} />
                    </div>
                    <span className="w-10 text-right text-xs font-bold">{pct}%</span>
                  </div>
                );
              })}
            </div>
          </div>
        </div>
      )}

      </>)}

      {/* Source form dialog */}
      {showSourceForm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowSourceForm(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <h3 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white"><Plus className="h-5 w-5 text-rose-500" /> Add Intel Source</h3>
            <div className="mt-4 space-y-3">
              <div><label className="text-sm font-medium">Source Name</label><input type="text" value={srcName} onChange={e => setSrcName(e.target.value)} placeholder="AlienVault OTX" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" autoFocus /></div>
              <div><label className="text-sm font-medium">Source Type</label>
                <select value={srcType} onChange={e => setSrcType(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">
                  {Object.entries(SOURCE_TYPE_LABELS).map(([k, v]: any[]) => <option key={k} value={k}>{v}</option>)}
                </select>
              </div>
              <div><label className="text-sm font-medium">API Key</label><input type="password" value={srcKey} onChange={e => setSrcKey(e.target.value)} placeholder="••••••••••••" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div>
              <div><label className="text-sm font-medium">Sync Interval (minutes)</label><input type="number" min={5} value={srcInterval} onChange={e => setSrcInterval(parseInt(e.target.value) || 60)} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>
            </div>
            <div className="mt-4 flex justify-end gap-2">
              <button onClick={() => setShowSourceForm(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">Cancel</button>
              <button onClick={() => setShowSourceForm(false)} disabled={!srcName} className="rounded-lg bg-rose-600 px-4 py-2 text-sm font-medium text-white hover:bg-rose-700 disabled:opacity-50">Add Source</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

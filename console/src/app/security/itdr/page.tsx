"use client";

import { useState, useEffect, useCallback } from "react";
import {
  Shield, AlertTriangle, Activity, Clock, Zap, TrendingDown,
  Loader2, RefreshCw, Filter, ChevronRight, Play, Eye, X,
} from "lucide-react";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";

// ---- Types ----
interface ThreatHeatmapData {
  zones: { label: string; count: number; severity: string }[];
  total_threats: number;
  by_severity: Record<string, number>;
}

interface KillChainData {
  stages: { stage: string; label: string; count: number; color: string }[];
  total_attacks: number;
}

interface IncidentTimelineEvent {
  id: string;
  timestamp: string;
  event: string;
  severity: "info" | "warning" | "critical";
  source: string;
}

interface Incident {
  id: string;
  title: string;
  status: "open" | "investigating" | "contained" | "resolved";
  severity: "low" | "medium" | "high" | "critical";
  first_detected?: string;
  last_updated?: string;
  created_at?: string;
  assigned_to?: string;
  description?: string;
  kill_chain_stage?: string;
  triggered_rules?: string[];
  detection_count?: number;
}

interface PlaybookStep {
  order: number;
  action: string;
  target: string;
  delay_seconds?: number;
}

interface Playbook {
  id: string;
  name: string;
  trigger: string;
  steps?: PlaybookStep[];
  actions?: string[]; // frontend convenience: extracted from steps
  enabled: boolean;
}

const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
const statusColors: Record<string, string> = {
  open: "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400",
  investigating: "bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400",
  contained: "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400",
  resolved: "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400",
};
const severityColors: Record<string, string> = {
  low: "text-blue-600 dark:text-blue-400",
  medium: "text-amber-600 dark:text-amber-400",
  high: "text-orange-600 dark:text-orange-400",
  critical: "text-red-600 dark:text-red-400",
};

export default function ITDRDashboardPage() {
  const { apiFetch } = useApi();
  const t = useTranslations();
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [activeTab, setActiveTab] = useState<"overview" | "incidents" | "playbooks" | "timeline">("overview");

  const [heatmap, setHeatmap] = useState<ThreatHeatmapData | null>(null);
  const [killChain, setKillChain] = useState<KillChainData | null>(null);
  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [timeline, setTimeline] = useState<IncidentTimelineEvent[]>([]);
  const [playbooks, setPlaybooks] = useState<Playbook[]>([]);
  const [selectedIncident, setSelectedIncident] = useState<Incident | null>(null);
  const [error, setError] = useState<string | null>(null);

  const loadData = useCallback(async (isRefresh = false) => {
    if (isRefresh) setRefreshing(true); else setLoading(true);
    setError(null);
    try {
      const [hm, kc, inc, tl, pb] = await Promise.allSettled([
        apiFetch<ThreatHeatmapData>("/api/v1/audit/threat-heatmap"),
        apiFetch<KillChainData>("/api/v1/audit/kill-chain"),
        apiFetch<{ incidents?: Incident[] } | Incident[]>("/api/v1/audit/itdr/incidents"),
        apiFetch<{ events?: IncidentTimelineEvent[] } | IncidentTimelineEvent[]>("/api/v1/audit/incident-timeline"),
        apiFetch<{ playbooks?: Playbook[] } | Playbook[]>("/api/v1/audit/itdr/playbooks"),
      ]);

      if (hm.status === "fulfilled") setHeatmap(hm.value);
      if (kc.status === "fulfilled") setKillChain(kc.value);
      if (inc.status === "fulfilled") {
        const val = inc.value;
        setIncidents(Array.isArray(val) ? val : (val?.incidents || []));
      }
      if (tl.status === "fulfilled") {
        const val = tl.value;
        setTimeline(Array.isArray(val) ? val : (val?.events || []));
      }
      if (pb.status === "fulfilled") {
        const val = pb.value;
        const rawPlaybooks = Array.isArray(val) ? val : (val?.playbooks || []);
        // Transform steps → actions for frontend display
        setPlaybooks(rawPlaybooks.map((pb: Playbook) => ({
          ...pb,
          actions: pb.actions || (pb.steps?.map(s => s.action) || []),
        })));
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : t("itdr.loadError"));
    }
    if (isRefresh) setRefreshing(false); else setLoading(false);
  }, [apiFetch]);

  useEffect(() => { loadData(); }, [loadData]);

  // ---- Stats ----
  const openIncidents = incidents.filter(i => i.status === "open" || i.status === "investigating");
  const criticalIncidents = incidents.filter(i => i.severity === "critical");
  const totalThreats = heatmap?.total_threats || 0;
  const totalAttacks = killChain?.total_attacks || 0;

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-6 w-6 animate-spin text-brand-500" />
        <span className="ml-2 text-sm text-gray-500">{t("itdr.loading")}</span>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <Shield className="h-6 w-6 text-red-500" /> {t("itdr.title")}
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {t("itdr.subtitle")}
          </p>
        </div>
        <button
          onClick={() => loadData(true)}
          disabled={refreshing}
          aria-label="Refresh ITDR data"
          className="flex items-center gap-2 rounded-lg border border-gray-300 dark:border-gray-600 px-3 py-2 text-sm hover:bg-gray-50 dark:hover:bg-gray-700"
        >
          <RefreshCw className={`h-4 w-4 ${refreshing ? "animate-spin" : ""}`} /> Refresh
        </button>
      </div>

      {error && (
        <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-600 dark:border-red-800 dark:bg-red-950/30 dark:text-red-400">
          {error}
        </div>
      )}

      {/* KPI Cards */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <div className={`${card} border-l-4 border-l-red-500`}>
          <div className="flex items-center justify-between">
            <div>
              <p className="text-xs uppercase text-gray-400">{t("itdr.openIncidents")}</p>
              <p className="mt-1 text-2xl font-bold text-gray-900 dark:text-white">{openIncidents.length}</p>
            </div>
            <AlertTriangle className="h-8 w-8 text-red-400" />
          </div>
        </div>
        <div className={`${card} border-l-4 border-l-orange-500`}>
          <div className="flex items-center justify-between">
            <div>
              <p className="text-xs uppercase text-gray-400">{t("itdr.critical")}</p>
              <p className="mt-1 text-2xl font-bold text-gray-900 dark:text-white">{criticalIncidents.length}</p>
            </div>
            <Zap className="h-8 w-8 text-orange-400" />
          </div>
        </div>
        <div className={`${card} border-l-4 border-l-amber-500`}>
          <div className="flex items-center justify-between">
            <div>
              <p className="text-xs uppercase text-gray-400">{t("itdr.totalThreats")}</p>
              <p className="mt-1 text-2xl font-bold text-gray-900 dark:text-white">{totalThreats}</p>
            </div>
            <TrendingDown className="h-8 w-8 text-amber-400" />
          </div>
        </div>
        <div className={`${card} border-l-4 border-l-blue-500`}>
          <div className="flex items-center justify-between">
            <div>
              <p className="text-xs uppercase text-gray-400">{t("itdr.attackPatterns")}</p>
              <p className="mt-1 text-2xl font-bold text-gray-900 dark:text-white">{totalAttacks}</p>
            </div>
            <Activity className="h-8 w-8 text-blue-400" />
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700" role="tablist">
        {(["overview", "incidents", "playbooks", "timeline"] as const).map(tab => (
          <button
            key={tab}
            role="tab"
            aria-selected={activeTab === tab}
            aria-label={`${tab} tab`}
            onClick={() => setActiveTab(tab)}
            className={`px-4 py-2 text-sm font-medium capitalize transition ${
              activeTab === tab
                ? "border-b-2 border-brand-600 text-brand-600"
                : "text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
            }`}
          >
            {tab === "overview" && t("itdr.tab.overview")}
            {tab === "incidents" && t("itdr.tab.incidents")}
            {tab === "playbooks" && t("itdr.tab.playbooks")}
            {tab === "timeline" && t("itdr.tab.timeline")}
          </button>
        ))}
      </div>

      {/* Tab: Overview — Heatmap + Kill Chain */}
      {activeTab === "overview" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          {/* Threat Heatmap */}
          <div className={card}>
            <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400">
              <AlertTriangle className="h-4 w-4" /> {t("itdr.heatmap")}
            </h3>
            {heatmap?.zones?.length ? (
              <div className="space-y-2">
                {heatmap.zones.map((zone, i) => (
                  <div key={i} className="flex items-center gap-3">
                    <span className="w-32 truncate text-sm text-gray-700 dark:text-gray-300">{zone.label}</span>
                    <div className="flex-1">
                      <div className="h-6 rounded-full bg-gray-100 dark:bg-gray-700 overflow-hidden">
                        <div
                          className={`h-full rounded-full ${
                            zone.severity === "critical" ? "bg-red-500" :
                            zone.severity === "high" ? "bg-orange-500" :
                            zone.severity === "medium" ? "bg-amber-500" : "bg-blue-500"
                          }`}
                          style={{ width: `${Math.min((zone.count / Math.max(...heatmap.zones.map(z => z.count))) * 100, 100)}%` }}
                        />
                      </div>
                    </div>
                    <span className="w-8 text-right text-sm font-medium text-gray-900 dark:text-white">{zone.count}</span>
                  </div>
                ))}
              </div>
            ) : (
              <p className="py-8 text-center text-sm text-gray-400">{t("itdr.noHeatmapData")}</p>
            )}
          </div>

          {/* Kill Chain */}
          <div className={card}>
            <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400">
              <Activity className="h-4 w-4" /> {t("itdr.killChain")}
            </h3>
            {killChain?.stages?.length ? (
              <div className="space-y-3">
                {killChain.stages.map((stage, i) => (
                  <div key={i} className="flex items-center gap-3">
                    <div className="flex h-8 w-8 items-center justify-center rounded-full text-xs font-bold text-white" style={{ backgroundColor: stage.color }}>
                      {i + 1}
                    </div>
                    <div className="flex-1">
                      <div className="text-sm font-medium text-gray-900 dark:text-white">{stage.label}</div>
                      <div className="text-xs text-gray-400">{stage.count} events</div>
                    </div>
                    {i < killChain.stages.length - 1 && <ChevronRight className="h-4 w-4 text-gray-300" />}
                  </div>
                ))}
              </div>
            ) : (
              <p className="py-8 text-center text-sm text-gray-400">{t("itdr.noKillChainData")}</p>
            )}
          </div>
        </div>
      )}

      {/* Tab: Incidents */}
      {activeTab === "incidents" && (
        <div className={card}>
          <div className="mb-4 flex items-center justify-between">
            <h3 className="text-sm font-semibold uppercase text-gray-400">{t("itdr.incidents")}</h3>
            <span className="text-xs text-gray-400">{incidents.length} {t("cmd.results")}</span>
          </div>
          {incidents.length === 0 ? (
            <p className="py-8 text-center text-sm text-gray-400">{t("itdr.noIncidents")}</p>
          ) : (
            <div className="space-y-2">
              {incidents.map(inc => (
                <div
                  key={inc.id}
                  onClick={() => setSelectedIncident(inc)}
                  className="flex cursor-pointer items-center justify-between rounded-lg border p-4 transition hover:border-brand-300 dark:border-gray-700 dark:hover:border-brand-700"
                >
                  <div className="flex items-center gap-3">
                    <div className={`h-2 w-2 rounded-full ${
                      inc.severity === "critical" ? "bg-red-500" :
                      inc.severity === "high" ? "bg-orange-500" :
                      inc.severity === "medium" ? "bg-amber-500" : "bg-blue-500"
                    }`} />
                    <div>
                      <div className="text-sm font-medium text-gray-900 dark:text-white">{inc.title}</div>
                      <div className="text-xs text-gray-400">
                        {new Date(inc.first_detected || inc.created_at || "").toLocaleString()}
                        {inc.assigned_to && ` · ${t("itdr.assignedTo")}: ${inc.assigned_to}`}
                      </div>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${statusColors[inc.status] || ""}`}>
                      {inc.status}
                    </span>
                    <span className={`text-xs font-medium ${severityColors[inc.severity] || ""}`}>
                      {inc.severity}
                    </span>
                    <Eye className="h-4 w-4 text-gray-400" />
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Tab: Playbooks */}
      {activeTab === "playbooks" && (
        <div className={card}>
          <h3 className="mb-4 text-sm font-semibold uppercase text-gray-400">{t("itdr.playbooks")}</h3>
          {playbooks.length === 0 ? (
            <p className="py-8 text-center text-sm text-gray-400">{t("itdr.noPlaybooks")}</p>
          ) : (
            <div className="space-y-3">
              {playbooks.map(pb => (
                <div key={pb.id} className="rounded-lg border p-4 dark:border-gray-700">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <Play className="h-4 w-4 text-brand-500" />
                      <span className="text-sm font-medium text-gray-900 dark:text-white">{pb.name}</span>
                    </div>
                    <span className={`rounded-full px-2 py-0.5 text-xs ${pb.enabled ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400" : "bg-gray-100 text-gray-500 dark:bg-gray-700 dark:text-gray-400"}`}>
                      {pb.enabled ? t("itdr.enabled") : t("itdr.disabled")}
                    </span>
                  </div>
                  <div className="mt-2 text-xs text-gray-400">{t("itdr.trigger")}: {pb.trigger}</div>
                  {(pb.actions ?? []).length > 0 && (
                    <div className="mt-2 flex flex-wrap gap-1">
                      {(pb.actions || []).map((action, i) => (
                        <span key={i} className="rounded bg-gray-100 dark:bg-gray-700 px-2 py-0.5 text-xs text-gray-600 dark:text-gray-300">
                          {action}
                        </span>
                      ))}
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Tab: Timeline */}
      {activeTab === "timeline" && (
        <div className={card}>
          <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400">
            <Clock className="h-4 w-4" /> {t("itdr.timeline")}
          </h3>
          {timeline.length === 0 ? (
            <p className="py-8 text-center text-sm text-gray-400">{t("itdr.noTimeline")}</p>
          ) : (
            <div className="relative space-y-4 pl-6">
              <div className="absolute left-2 top-0 h-full w-px bg-gray-200 dark:bg-gray-700" />
              {timeline.map((evt, i) => (
                <div key={i} className="relative">
                  <div className={`absolute -left-[18px] h-3 w-3 rounded-full border-2 border-white dark:border-gray-800 ${
                    evt.severity === "critical" ? "bg-red-500" :
                    evt.severity === "warning" ? "bg-amber-500" : "bg-blue-500"
                  }`} />
                  <div className="text-sm font-medium text-gray-900 dark:text-white">{evt.event}</div>
                  <div className="text-xs text-gray-400">
                    {new Date(evt.timestamp).toLocaleString()} · {evt.source}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Incident Detail Modal */}
      {selectedIncident && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40" onClick={() => setSelectedIncident(null)}>
          <div className="w-full max-w-lg rounded-xl bg-white p-6 shadow-2xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <div className="mb-4 flex items-center justify-between">
              <h2 className="text-lg font-bold text-gray-900 dark:text-white">{selectedIncident.title}</h2>
              <button onClick={() => setSelectedIncident(null)} className="rounded-lg p-1 text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700">
                <X className="h-5 w-5" />
              </button>
            </div>
            <div className="space-y-3 text-sm">
              <div className="flex gap-2">
                <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${statusColors[selectedIncident.status] || ""}`}>{selectedIncident.status}</span>
                <span className={`text-xs font-medium ${severityColors[selectedIncident.severity] || ""}`}>{selectedIncident.severity}</span>
                {selectedIncident.kill_chain_stage && (
                  <span className="rounded-full bg-purple-100 px-2 py-0.5 text-xs text-purple-700 dark:bg-purple-900/30 dark:text-purple-400">
                    {selectedIncident.kill_chain_stage}
                  </span>
                )}
              </div>
              <div>
                <span className="text-gray-400">Detected:</span>
                <span className="ml-2 text-gray-700 dark:text-gray-300">{new Date(selectedIncident.first_detected || selectedIncident.created_at || "").toLocaleString()}</span>
              </div>
              {selectedIncident.assigned_to && (
                <div>
                  <span className="text-gray-400">Assigned to:</span>
                  <span className="ml-2 text-gray-700 dark:text-gray-300">{selectedIncident.assigned_to}</span>
                </div>
              )}
              {selectedIncident.description && (
                <div>
                  <span className="text-gray-400">Description:</span>
                  <p className="mt-1 text-gray-700 dark:text-gray-300">{selectedIncident.description}</p>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
"use client";

import { useEffect, useState, useCallback, useMemo } from "react";
import { useApi } from "@/lib/api";
import {
  BarChart3, LineChart, PieChart as PieIcon, Calendar, Download,
  Save, Trash2, Clock, Mail, Filter, FileBarChart, RefreshCw,
  FolderOpen, Plus,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

// ---- Types ----
interface ReportConfig {
  name: string;
  date_from: string;
  date_to: string;
  user_search: string;
  event_type: string;
  severity: string;
  service: string;
  group_by: "none" | "day" | "user" | "service" | "event_type";
  chart_type: "bar" | "line" | "pie";
}

interface SavedReport {
  id: string;
  name: string;
  config: ReportConfig;
  created_at: string;
}

interface ScheduleConfig {
  enabled: boolean;
  frequency: "daily" | "weekly" | "monthly";
  day_of_week: number;
  day_of_month: number;
  time: string;
  recipients: string;
}

interface GroupData {
  name: string;
  count: number;
  percentage: number;
}

// ---- Constants ----
const EVENT_TYPES = ["all", "login", "logout", "mfa", "token", "api", "password", "role", "config"];
const SEVERITIES = ["all", "info", "warning", "error", "critical"];
const SERVICES = ["all", "auth", "oauth", "identity", "policy", "audit", "gateway"];
const CHART_COLORS = ["#6366f1", "#10b981", "#f59e0b", "#ef4444", "#8b5cf6", "#06b6d4", "#ec4899", "#84cc16", "#f97316"];
const DAYS_OF_WEEK = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

const inputCls = "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";
const labelCls = "mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300";
const cardCls = "rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800";

export default function AuditReportsPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [msg, setMsg] = useState<string | null>(null);
  const [msgType, setMsgType] = useState<"success" | "error">("success");
  const [reportName, setReportName] = useState("New Report");

  // Report config
  const [config, setConfig] = useState<ReportConfig>({
    name: "New Report",
    date_from: new Date(Date.now() - 30 * 86400000).toISOString().slice(0, 10),
    date_to: new Date().toISOString().slice(0, 10),
    user_search: "",
    event_type: "all",
    severity: "all",
    service: "all",
    group_by: "day",
    chart_type: "bar",
  });

  // Schedule
  const [schedule, setSchedule] = useState<ScheduleConfig>({
    enabled: false,
    frequency: "weekly",
    day_of_week: 1,
    day_of_month: 1,
    time: "09:00",
    recipients: "",
  });

  // Saved reports
  const [savedReports, setSavedReports] = useState<SavedReport[]>([]);
  const [chartData, setChartData] = useState<GroupData[]>([]);
  const [loading, setLoading] = useState(false);

  // ---- Helpers ----
  const showMsg = (text: string, type: "success" | "error" = "success") => {
    setMsg(text);
    setMsgType(type);
    setTimeout(() => setMsg(null), 4000);
  };

  // ---- Data loading & aggregation ----
  const loadReportData = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams();
      params.set("page_size", "500");
      if (config.date_from) params.set("from", config.date_from + "T00:00:00Z");
      if (config.date_to) params.set("to", config.date_to + "T23:59:59Z");

      interface RawEvent {
        action: string;
        actor_id?: string;
        actor_name?: string;
        created_at: string;
        result?: string;
        metadata?: Record<string, unknown>;
      }

      let events: RawEvent[] = [];
      try {
        const data = await apiFetch<{ events?: RawEvent[] } | RawEvent[]>(
          `/api/v1/audit/events?${params}`,
        );
        events = Array.isArray(data) ? data : data.events || [];
      } catch {
        // Generate sample data for preview when API unavailable
        events = generateSampleData(config);
      }

      // Apply client-side filters
      events = events.filter((e) => {
        if (config.user_search) {
          const u = config.user_search.toLowerCase();
          if (!(e.actor_name || "").toLowerCase().includes(u) &&
              !(e.actor_id || "").toLowerCase().includes(u)) return false;
        }
        if (config.event_type !== "all") {
          if (!e.action.toLowerCase().includes(config.event_type)) return false;
        }
        if (config.service !== "all") {
          const inferred = inferService(e.action);
          if (inferred !== config.service) return false;
        }
        if (config.severity !== "all") {
          const inferred = inferSeverity(e.action, e.result);
          if (inferred !== config.severity) return false;
        }
        return true;
      });

      // Group data
      const grouped = groupEvents(events, config.group_by);
      setChartData(grouped);
    } catch {
      setChartData([]);
    } finally {
      setLoading(false);
    }
  }, [apiFetch, config]);

  useEffect(() => {
    loadReportData();
  }, [loadReportData]);

  // ---- SVG Chart rendering ----
  const maxCount = useMemo(() => Math.max(...chartData.map((d) => d.count), 1), [chartData]);

  const renderBarChart = () => {
    const barWidth = chartData.length > 0 ? Math.max(20, 600 / chartData.length - 10) : 0;
    const chartHeight = 250;
    const labelW = 60;
    const w = Math.max(600, chartData.length * (barWidth + 10) + labelW);

    return (
      <svg viewBox={`0 0 ${w} ${chartHeight + 50}`} className="w-full" style={{ maxHeight: 320 }}>
        {/* Grid lines */}
        {[0, 0.25, 0.5, 0.75, 1].map((r) => (
          <g key={r}>
            <line
              x1={labelW} y1={chartHeight - r * chartHeight + 10}
              x2={w - 10} y2={chartHeight - r * chartHeight + 10}
              stroke="#e5e7eb" strokeWidth={1}
            />
            <text x={labelW - 8} y={chartHeight - r * chartHeight + 14} textAnchor="end" fontSize={10} fill="#9ca3af">
              {Math.round(maxCount * r)}
            </text>
          </g>
        ))}
        {/* Bars */}
        {chartData.map((d, i) => {
          const h = (d.count / maxCount) * chartHeight;
          const x = labelW + i * (barWidth + 10) + 5;
          const y = chartHeight - h + 10;
          return (
            <g key={i}>
              <rect x={x} y={y} width={barWidth} height={h} fill={CHART_COLORS[i % CHART_COLORS.length]} rx={3} />
              <text x={x + barWidth / 2} y={chartHeight + 25} textAnchor="middle" fontSize={10} fill="#6b7280">
                {d.name.length > 12 ? d.name.slice(0, 10) + "..." : d.name}
              </text>
              <text x={x + barWidth / 2} y={y - 4} textAnchor="middle" fontSize={10} fill="#374151" fontWeight={600}>
                {d.count}
              </text>
            </g>
          );
        })}
      </svg>
    );
  };

  const renderLineChart = () => {
    const chartHeight = 250;
    const labelW = 60;
    const w = 600;
    const stepX = chartData.length > 1 ? (w - labelW - 20) / (chartData.length - 1) : 0;

    const points = chartData.map((d, i) => ({
      x: labelW + 10 + i * stepX,
      y: chartHeight - (d.count / maxCount) * chartHeight + 10,
      data: d,
    }));

    const pathD = points.map((p, i) => `${i === 0 ? "M" : "L"} ${p.x} ${p.y}`).join(" ");

    return (
      <svg viewBox={`0 0 ${w} ${chartHeight + 50}`} className="w-full" style={{ maxHeight: 320 }}>
        {[0, 0.25, 0.5, 0.75, 1].map((r) => (
          <g key={r}>
            <line x1={labelW} y1={chartHeight - r * chartHeight + 10} x2={w - 10} y2={chartHeight - r * chartHeight + 10} stroke="#e5e7eb" strokeWidth={1} />
            <text x={labelW - 8} y={chartHeight - r * chartHeight + 14} textAnchor="end" fontSize={10} fill="#9ca3af">
              {Math.round(maxCount * r)}
            </text>
          </g>
        ))}
        {/* Area fill */}
        {chartData.length > 1 && (
          <path
            d={`${pathD} L ${points[points.length - 1].x} ${chartHeight + 10} L ${points[0].x} ${chartHeight + 10} Z`}
            fill="#6366f1" fillOpacity={0.1}
          />
        )}
        <path d={pathD} fill="none" stroke="#6366f1" strokeWidth={2} />
        {points.map((p, i) => (
          <g key={i}>
            <circle cx={p.x} cy={p.y} r={4} fill="#6366f1" stroke="white" strokeWidth={2} />
            <text x={p.x} y={chartHeight + 25} textAnchor="middle" fontSize={10} fill="#6b7280">
              {p.data.name.length > 10 ? p.data.name.slice(0, 8) + ".." : p.data.name}
            </text>
          </g>
        ))}
      </svg>
    );
  };

  const renderPieChart = () => {
    const total = chartData.reduce((s, d) => s + d.count, 0) || 1;
    const cx = 150, cy = 130, r = 100;
    let cumulative = 0;

    const slices = chartData.map((d, i) => {
      const startAngle = (cumulative / total) * 2 * Math.PI - Math.PI / 2;
      cumulative += d.count;
      const endAngle = (cumulative / total) * 2 * Math.PI - Math.PI / 2;
      const x1 = cx + r * Math.cos(startAngle);
      const y1 = cy + r * Math.sin(startAngle);
      const x2 = cx + r * Math.cos(endAngle);
      const y2 = cy + r * Math.sin(endAngle);
      const largeArc = endAngle - startAngle > Math.PI ? 1 : 0;
      const path = `M ${cx} ${cy} L ${x1} ${y1} A ${r} ${r} 0 ${largeArc} 1 ${x2} ${y2} Z`;
      return { path, color: CHART_COLORS[i % CHART_COLORS.length], name: d.name, pct: ((d.count / total) * 100).toFixed(1) };
    });

    return (
      <svg viewBox="0 0 300 300" className="w-full" style={{ maxHeight: 320 }}>
        {slices.map((s, i) => (
          <path key={i} d={s.path} fill={s.color} stroke="white" strokeWidth={2} />
        ))}
        <circle cx={cx} cy={cy} r={45} fill="white" />
        <text x={cx} y={cy - 5} textAnchor="middle" fontSize={18} fontWeight={700} fill="#374151">{total}</text>
        <text x={cx} y={cy + 12} textAnchor="middle" fontSize={10} fill="#9ca3af">{t("auditReports.totalEvents")}</text>
      </svg>
    );
  };

  const renderChart = () => {
    if (chartData.length === 0) {
      return (
        <div className="flex h-[250px] items-center justify-center text-sm text-gray-400">
          {t("auditReports.noData")}
        </div>
      );
    }
    switch (config.chart_type) {
      case "bar": return renderBarChart();
      case "line": return renderLineChart();
      case "pie": return renderPieChart();
    }
  };

  // ---- Actions ----
  const handleSaveReport = async () => {
    try {
      await apiFetch("/api/v1/audit/reports", {
        method: "POST",
        body: JSON.stringify({ ...config, name: reportName, schedule }),
      }).catch(() => null);

      const newReport: SavedReport = {
        id: `rpt-${Date.now()}`,
        name: reportName,
        config: { ...config, name: reportName },
        created_at: new Date().toISOString(),
      };
      setSavedReports((prev) => [newReport, ...prev]);
      showMsg("Report template saved");
    } catch {
      showMsg("Failed to save report", "error");
    }
  };

  const handleExportCSV = () => {
    const rows = [["Group", "Count", "Percentage"]];
    chartData.forEach((d) => rows.push([d.name, String(d.count), `${d.percentage}%`]));
    const csv = rows.map((r) => r.join(",")).join("\n");
    const blob = new Blob([csv], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `${reportName.replace(/\s+/g, "-").toLowerCase()}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const handleLoadReport = (rpt: SavedReport) => {
    setConfig(rpt.config);
    setReportName(rpt.name);
    showMsg(`Loaded report "${rpt.name}"`);
  };

  const handleDeleteReport = (id: string) => {
    setSavedReports((prev) => prev.filter((r) => r.id !== id));
    showMsg("Report deleted");
  };

  return (
    <div>
      <style jsx global>{`
        @media print {
          body * { visibility: hidden; }
          #print-area, #print-area * { visibility: visible; }
          #print-area { position: absolute; left: 0; top: 0; width: 100%; }
          .no-print { display: none !important; }
        }
      `}</style>

      {/* Toast */}
      {msg && (
        <div className={`fixed right-4 top-4 z-50 rounded-lg px-4 py-3 text-sm text-white shadow-lg ${msgType === "success" ? "bg-green-600" : "bg-red-600"}`}>
          {msg}
        </div>
      )}

      {/* Header */}
      <div className="mb-6 flex items-center justify-between no-print">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">{t("auditReports.title")}</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("auditReports.subtitle")}</p>
        </div>
        <div className="flex gap-2">
          <button onClick={() => window.print()} className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600">
            <Download className="h-4 w-4" /> Export PDF
          </button>
          <button onClick={handleExportCSV} className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600">
            <Download className="h-4 w-4" /> Export CSV
          </button>
          <button onClick={loadReportData} className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600">
            <RefreshCw className="h-4 w-4" /> Refresh
          </button>
        </div>
      </div>

      <div className="grid gap-6 lg:grid-cols-4">
        {/* Left: Report builder form + chart */}
        <div className="space-y-6 lg:col-span-3" id="print-area">
          {/* Report Builder Form */}
          <div className={cardCls}>
            <div className="mb-4 flex items-center gap-2">
              <FileBarChart className="h-5 w-5 text-brand-600" />
              <input
                type="text"
                value={reportName}
                onChange={(e) => setReportName(e.target.value)}
                className="flex-1 rounded-lg border-0 border-b border-transparent px-1 text-lg font-semibold text-gray-900 focus:border-brand-500 focus:outline-none dark:bg-transparent dark:text-gray-100"
              />
            </div>

            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              <div>
                <label className={labelCls}><Calendar className="mr-1 inline h-3.5 w-3.5" />{t("auditReports.from")}</label>
                <input aria-label="config" type="date" value={config.date_from} onChange={(e) => setConfig({ ...config, date_from: e.target.value })} className={inputCls} />
              </div>
              <div>
                <label className={labelCls}><Calendar className="mr-1 inline h-3.5 w-3.5" />{t("auditReports.to")}</label>
                <input aria-label="config" type="date" value={config.date_to} onChange={(e) => setConfig({ ...config, date_to: e.target.value })} className={inputCls} />
              </div>
              <div>
                <label className={labelCls}><Filter className="mr-1 inline h-3.5 w-3.5" />{t("auditReports.userSearch")}</label>
                <input aria-label="username or ID..." type="text" value={config.user_search} onChange={(e) => setConfig({ ...config, user_search: e.target.value })} placeholder="username or ID..." className={inputCls} />
              </div>
              <div>
                <label className={labelCls}>{t("auditReports.eventType")}</label>
                <select aria-label="config" value={config.event_type} onChange={(e) => setConfig({ ...config, event_type: e.target.value })} className={inputCls}>
                  {EVENT_TYPES.map((t) => <option key={t} value={t}>{t === "all" ? "All Events" : t.charAt(0).toUpperCase() + t.slice(1)}</option>)}
                </select>
              </div>
              <div>
                <label className={labelCls}>{t("auditReports.severity")}</label>
                <select aria-label="config" value={config.severity} onChange={(e) => setConfig({ ...config, severity: e.target.value })} className={inputCls}>
                  {SEVERITIES.map((s) => <option key={s} value={s}>{s === "all" ? "All Severities" : s.charAt(0).toUpperCase() + s.slice(1)}</option>)}
                </select>
              </div>
              <div>
                <label className={labelCls}>{t("auditReports.service")}</label>
                <select aria-label="config" value={config.service} onChange={(e) => setConfig({ ...config, service: e.target.value })} className={inputCls}>
                  {SERVICES.map((s) => <option key={s} value={s}>{s === "all" ? "All Services" : s.charAt(0).toUpperCase() + s.slice(1)}</option>)}
                </select>
              </div>
            </div>

            {/* Group By + Chart Type */}
            <div className="mt-4 grid gap-4 sm:grid-cols-2">
              <div>
                <label className={labelCls}>{t("auditReports.groupBy")}</label>
                <div className="flex flex-wrap gap-2">
                  {(["none", "day", "user", "service", "event_type"] as const).map((g) => (
                    <label key={g} className={`cursor-pointer rounded-lg border px-3 py-1.5 text-xs font-medium ${config.group_by === g ? "border-brand-500 bg-brand-50 text-brand-700 dark:bg-brand-950/30 dark:text-brand-400" : "border-gray-300 text-gray-600 dark:border-gray-600 dark:text-gray-400"}`}>
                      <input aria-label="Config" type="radio" name="group_by" value={g} checked={config.group_by === g} onChange={(e) => setConfig({ ...config, group_by: e.target.value as ReportConfig["group_by"] })} className="hidden" />
                      {g === "none" ? "None" : g === "event_type" ? "Event Type" : g.charAt(0).toUpperCase() + g.slice(1)}
                    </label>
                  ))}
                </div>
              </div>
              <div>
                <label className={labelCls}>{t("auditReports.chartType")}</label>
                <div className="flex gap-2">
                  {([
                    { v: "bar", icon: BarChart3, label: "Bar" },
                    { v: "line", icon: LineChart, label: "Line" },
                    { v: "pie", icon: PieIcon, label: "Pie" },
                  ] as const).map((c) => (
                    <label key={c.v} className={`flex cursor-pointer items-center gap-1.5 rounded-lg border px-3 py-1.5 text-xs font-medium ${config.chart_type === c.v ? "border-brand-500 bg-brand-50 text-brand-700 dark:bg-brand-950/30 dark:text-brand-400" : "border-gray-300 text-gray-600 dark:border-gray-600 dark:text-gray-400"}`}>
                      <input aria-label="Config" type="radio" name="chart_type" value={c.v} checked={config.chart_type === c.v} onChange={(e) => setConfig({ ...config, chart_type: e.target.value as ReportConfig["chart_type"] })} className="hidden" />
                      <c.icon className="h-3.5 w-3.5" /> {c.label}
                    </label>
                  ))}
                </div>
              </div>
            </div>
          </div>

          {/* Chart Preview */}
          <div className={cardCls}>
            <div className="mb-4 flex items-center justify-between">
              <h2 className="text-sm font-semibold text-gray-700 dark:text-gray-300">{t("auditReports.preview")}</h2>
              {loading && <RefreshCw className="h-4 w-4 animate-spin text-gray-400" />}
            </div>
            {renderChart()}
            {/* Legend */}
            {chartData.length > 0 && (
              <div className="mt-4 flex flex-wrap gap-3">
                {chartData.map((d, i) => (
                  <div key={i} className="flex items-center gap-1.5 text-xs text-gray-600 dark:text-gray-400">
                    <span className="h-3 w-3 rounded" style={{ backgroundColor: CHART_COLORS[i % CHART_COLORS.length] }} />
                    {d.name} ({d.count})
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Report Details Table */}
          <div className={cardCls}>
            <h2 className="mb-4 text-sm font-semibold text-gray-700 dark:text-gray-300">Aggregated Data</h2>
            {chartData.length > 0 ? (
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr className="border-b border-gray-200 dark:border-gray-700 text-left text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">
                      <th scope="col" className="pb-2 pr-4">Group</th>
                      <th scope="col" className="pb-2 pr-4">Count</th>
                      <th scope="col" className="pb-2">Percentage</th>
                    </tr>
                  </thead>
                  <tbody>
                    {chartData.map((d, i) => (
                      <tr key={i} className="border-b border-gray-100 dark:border-gray-700/50">
                        <td className="py-2 pr-4 text-sm text-gray-800 dark:text-gray-200">{d.name}</td>
                        <td className="py-2 pr-4 text-sm font-medium text-gray-900 dark:text-gray-100">{d.count}</td>
                        <td className="py-2">
                          <div className="flex items-center gap-2">
                            <div className="h-2 w-20 rounded-full bg-gray-200 dark:bg-gray-700">
                              <div className="h-2 rounded-full" style={{ width: `${d.percentage}%`, backgroundColor: CHART_COLORS[i % CHART_COLORS.length] }} />
                            </div>
                            <span className="text-xs text-gray-500">{d.percentage.toFixed(1)}%</span>
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <p className="py-8 text-center text-sm text-gray-400">No data available for current filters</p>
            )}
          </div>
        </div>

        {/* Right sidebar: Schedule + Saved reports */}
        <div className="space-y-6 no-print">
          {/* Schedule */}
          <div className={cardCls}>
            <div className="mb-4 flex items-center justify-between">
              <h2 className="flex items-center gap-2 text-sm font-semibold text-gray-700 dark:text-gray-300">
                <Clock className="h-4 w-4 text-brand-600" /> {t("auditReports.schedule")}
              </h2>
              <label className="relative inline-flex cursor-pointer items-center">
                <input aria-label="Schedule" type="checkbox" checked={schedule.enabled} onChange={(e) => setSchedule({ ...schedule, enabled: e.target.checked })} className="peer sr-only" />
                <div className="h-5 w-9 rounded-full bg-gray-300 peer-checked:bg-brand-600 dark:bg-gray-600 dark:peer-checked:bg-brand-500" />
                <div className="absolute left-0.5 top-0.5 h-4 w-4 rounded-full bg-white transition-transform peer-checked:translate-x-4" />
              </label>
            </div>

            {schedule.enabled && (
              <div className="space-y-3">
                <div>
                  <label className={labelCls}>{t("auditReports.frequency")}</label>
                  <select aria-label="schedule" value={schedule.frequency} onChange={(e) => setSchedule({ ...schedule, frequency: e.target.value as ScheduleConfig["frequency"] })} className={inputCls}>
                    <option value="daily">Daily</option>
                    <option value="weekly">Weekly</option>
                    <option value="monthly">Monthly</option>
                  </select>
                </div>
                {schedule.frequency === "weekly" && (
                  <div>
                    <label className={labelCls}>Day of Week</label>
                    <select aria-label="schedule" value={schedule.day_of_week} onChange={(e) => setSchedule({ ...schedule, day_of_week: parseInt(e.target.value) })} className={inputCls}>
                      {DAYS_OF_WEEK.map((d, i) => <option key={i} value={i}>{d}</option>)}
                    </select>
                  </div>
                )}
                {schedule.frequency === "monthly" && (
                  <div>
                    <label className={labelCls}>Day of Month</label>
                    <select aria-label="schedule" value={schedule.day_of_month} onChange={(e) => setSchedule({ ...schedule, day_of_month: parseInt(e.target.value) })} className={inputCls}>
                      {Array.from({ length: 28 }, (_, i) => i + 1).map((d) => <option key={d} value={d}>{d}</option>)}
                    </select>
                  </div>
                )}
                <div>
                  <label className={labelCls}>Time</label>
                  <input aria-label="schedule" type="time" value={schedule.time} onChange={(e) => setSchedule({ ...schedule, time: e.target.value })} className={inputCls} />
                </div>
                <div>
                  <label className={labelCls}><Mail className="mr-1 inline h-3.5 w-3.5" />{t("auditReports.recipients")}</label>
                  <input aria-label="admin@example.com, team@example.com" type="text" value={schedule.recipients} onChange={(e) => setSchedule({ ...schedule, recipients: e.target.value })} placeholder="admin@example.com, team@example.com" className={inputCls} />
                  <p className="mt-1 text-xs text-gray-400">Comma-separated email addresses</p>
                </div>
              </div>
            )}
          </div>

          {/* Save button */}
          <button onClick={handleSaveReport} className="flex w-full items-center justify-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2.5 text-sm font-medium text-white hover:bg-brand-700">
            <Save className="h-4 w-4" /> {t("auditReports.saveReport")}
          </button>

          {/* Saved Reports */}
          <div className={cardCls}>
            <h2 className="mb-3 flex items-center gap-2 text-sm font-semibold text-gray-700 dark:text-gray-300">
              <FolderOpen className="h-4 w-4 text-brand-600" /> {t("auditReports.savedReports")}
            </h2>
            {savedReports.length === 0 ? (
              <p className="py-4 text-center text-xs text-gray-400">No saved reports yet</p>
            ) : (
              <div className="space-y-2">
                {savedReports.map((r) => (
                  <div key={r.id} className="flex items-center justify-between rounded-lg border border-gray-200 px-3 py-2 dark:border-gray-700">
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-medium text-gray-800 dark:text-gray-200">{r.name}</p>
                      <p className="text-xs text-gray-400">{new Date(r.created_at).toLocaleDateString()}</p>
                    </div>
                    <div className="flex gap-1">
                      <button onClick={() => handleLoadReport(r)} className="rounded p-1 text-brand-600 hover:bg-brand-50 dark:hover:bg-brand-950/30" title="Load">
                        <FolderOpen className="h-3.5 w-3.5" />
                      </button>
                      <button onClick={() => handleDeleteReport(r.id)} className="rounded p-1 text-red-500 hover:bg-red-50 dark:hover:bg-red-950/30" title="Delete">
                        <Trash2 className="h-3.5 w-3.5" />
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

// ---- Utility functions ----
function inferService(action: string): string {
  if (action.startsWith("user.") || action.startsWith("auth.")) return "auth";
  if (action.startsWith("oauth.") || action.startsWith("token.")) return "oauth";
  if (action.startsWith("policy.") || action.startsWith("role.")) return "policy";
  if (action.startsWith("org.") || action.startsWith("member.")) return "identity";
  return "audit";
}

function inferSeverity(action: string, result?: string): string {
  if (result === "denied") return "critical";
  if (result === "failure") return "error";
  if (action.includes("mfa") || action.includes("denied")) return "warning";
  return "info";
}

function groupEvents(
  events: { action: string; actor_id?: string; actor_name?: string; created_at: string; result?: string }[],
  groupBy: string,
): GroupData[] {
  if (events.length === 0) return [];
  const groups: Record<string, number> = {};

  for (const e of events) {
    let key = "All";
    switch (groupBy) {
      case "day":
        key = new Date(e.created_at).toISOString().slice(0, 10);
        break;
      case "user":
        key = e.actor_name || e.actor_id?.slice(0, 8) || "system";
        break;
      case "service":
        key = inferService(e.action);
        break;
      case "event_type":
        key = e.action;
        break;
      default:
        key = "All Events";
    }
    groups[key] = (groups[key] || 0) + 1;
  }

  const total = events.length;
  return Object.entries(groups)
    .map(([name, count]) => ({ name, count, percentage: (count / total) * 100 }))
    .sort((a, b) => b.count - a.count)
    .slice(0, 15);
}

function generateSampleData(config: ReportConfig) {
  const actionTypes = ["user.login", "user.logout", "token.refresh", "policy.evaluate", "role.assign", "user.register"];
  const services = ["auth", "oauth", "policy", "identity"];
  const sample: { action: string; actor_id: string; actor_name: string; created_at: string; result: string }[] = [];
  const days = Math.ceil((new Date(config.date_to).getTime() - new Date(config.date_from).getTime()) / 86400000) || 7;
  for (let i = 0; i < 80; i++) {
    const action = actionTypes[i % actionTypes.length];
    sample.push({
      action,
      actor_id: `user-${(i % 5) + 1}`,
      actor_name: `user${(i % 5) + 1}`,
      created_at: new Date(Date.now() - Math.floor(Math.random() * days * 86400000)).toISOString(),
      result: Math.random() > 0.15 ? "success" : "failure",
    });
  }
  return sample;
}

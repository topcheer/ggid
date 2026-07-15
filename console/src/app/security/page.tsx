"use client";

import { useEffect, useState, useCallback } from "react";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";
import {
  ShieldAlert,
  Lock,
  Ban,
  AlertTriangle,
  Activity,
  Plus,
  X,
  CheckCircle2,
  Circle,
  Clock,
  MapPin,
  ShieldCheck,
} from "lucide-react";

// --- Types ---

interface ThreatOverview {
  failed_logins_24h: number;
  locked_accounts: number;
  suspicious_ips: number;
  active_threats: number;
}

interface HeatmapCell {
  day: string;
  hour: number;
  count: number;
}

interface AnomalyAlert {
  id: string;
  type: "impossible_travel" | "brute_force" | "credential_stuffing";
  severity: "high" | "medium" | "low";
  description: string;
  timestamp: string;
  ip: string;
}

interface SecurityData {
  overview: ThreatOverview;
  heatmap: HeatmapCell[];
  anomalies: AnomalyAlert[];
  ip_allowlist: string[];
  ip_denylist: string[];
  recommendations: SecurityRecommendation[];
}

interface SecurityRecommendation {
  id: string;
  title: string;
  description: string;
  done: boolean;
}

// --- Component ---

export default function SecurityCenterPage() {
  const t = useTranslations();
  const { apiFetch } = useApi();
  const [data, setData] = useState<SecurityData | null>(null);
  const [loading, setLoading] = useState(true);
  const [dismissedAnomalies, setDismissedAnomalies] = useState<Set<string>>(new Set());
  const [allowInput, setAllowInput] = useState("");
  const [denyInput, setDenyInput] = useState("");
  const [msg, setMsg] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await apiFetch<SecurityData>("/api/v1/security/overview");
      setData(result);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load security overview");
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  useEffect(() => {
    if (msg) {
      const t = setTimeout(() => setMsg(null), 2500);
      return () => clearTimeout(t);
    }
  }, [msg]);

  if (error) {
    return (
      <div className="p-8">
        <div className="rounded-lg border border-red-300 bg-red-50 dark:bg-red-950 dark:border-red-800 p-4">
          <p className="text-red-700 dark:text-red-400 text-sm font-medium">Error: {error}</p>
          <button onClick={loadData} className="mt-2 px-4 py-1.5 rounded-lg bg-red-600 text-white text-sm hover:bg-red-700">{t("common.refresh")}</button>
        </div>
      </div>
    );
  }

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <Activity className="h-6 w-6 animate-pulse text-gray-400" />
        <span className="ml-2 text-gray-500">{t("security.loading")}</span>
      </div>
    );
  }

  const overview = data!.overview;
  const heatmap = data!.heatmap;
  const anomalies = data!.anomalies.filter((a) => !dismissedAnomalies.has(a.id));
  const allowlist = data!.ip_allowlist;
  const denylist = data!.ip_denylist;
  const recommendations = data!.recommendations;

  const cardCls =
    "rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const headingCls = "mb-4 text-lg font-semibold text-gray-900 dark:text-gray-100";
  const inputCls =
    "flex-1 rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";

  const severityColor = (severity: string) => {
    switch (severity) {
      case "high":
        return "bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-400";
      case "medium":
        return "bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-400";
      default:
        return "bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-400";
    }
  };

  const typeLabel = (type: string) =>
    type === "impossible_travel"
      ? "Impossible Travel"
      : type === "brute_force"
        ? "Brute Force"
        : "Credential Stuffing";

  // Heatmap helpers
  const days = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"];
  const hours = Array.from({ length: 24 }, (_, i) => i);
  const maxCount = Math.max(1, ...heatmap.map((c) => c.count));
  const heatColor = (count: number) => {
    const ratio = count / maxCount;
    if (ratio === 0) return "bg-gray-100 dark:bg-gray-700";
    if (ratio < 0.25) return "bg-green-200 dark:bg-green-900";
    if (ratio < 0.5) return "bg-amber-300 dark:bg-amber-800";
    if (ratio < 0.75) return "bg-orange-400 dark:bg-orange-700";
    return "bg-red-500 dark:bg-red-600";
  };
  const getCell = (day: string, hour: number) =>
    heatmap.find((c) => c.day === day && c.hour === hour)?.count ?? 0;

  const addAllow = () => {
    if (!allowInput.trim()) return;
    setData({ ...data!, ip_allowlist: [...allowlist, allowInput.trim()] });
    setAllowInput("");
    setMsg("Added to allowlist");
  };

  const removeAllow = (ip: string) => {
    setData({ ...data!, ip_allowlist: allowlist.filter((x) => x !== ip) });
  };

  const addDeny = () => {
    if (!denyInput.trim()) return;
    setData({ ...data!, ip_denylist: [...denylist, denyInput.trim()] });
    setDenyInput("");
    setMsg("Added to denylist");
  };

  const removeDeny = (ip: string) => {
    setData({ ...data!, ip_denylist: denylist.filter((x) => x !== ip) });
  };

  const dismissAnomaly = (id: string) => {
    setDismissedAnomalies((prev) => new Set(prev).add(id));
    setMsg("Anomaly dismissed");
  };

  const toggleRec = (id: string) => {
    setData({
      ...data!,
      recommendations: recommendations.map((r) =>
        r.id === id ? { ...r, done: !r.done } : r,
      ),
    });
  };

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">{t("security.title")}</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {t("security.subtitle")}
        </p>
      </div>

      {msg && (
        <div className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400">
          {msg}
        </div>
      )}

      {/* ===== Threat Overview Cards ===== */}
      <div className="mb-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <ThreatCard
          icon={AlertTriangle}
          label="Failed Logins (24h)"
          value={overview.failed_logins_24h}
          color="amber"
        />
        <ThreatCard
          icon={Lock}
          label="Locked Accounts"
          value={overview.locked_accounts}
          color="red"
        />
        <ThreatCard
          icon={Ban}
          label="Suspicious IPs"
          value={overview.suspicious_ips}
          color="orange"
        />
        <ThreatCard
          icon={ShieldAlert}
          label="Active Threats"
          value={overview.active_threats}
          color="red"
        />
      </div>

      {/* ===== Failed Login Heatmap ===== */}
      <div className="mb-6">
        <div className={cardCls}>
          <h2 className={`flex items-center gap-2 ${headingCls}`}>
            <Activity className="h-5 w-5 text-brand-600" /> Failed Login Heatmap (7 days)
          </h2>
          <div className="overflow-x-auto">
            <div className="min-w-[700px]">
              {/* Hour labels */}
              <div className="mb-1 flex">
                <div className="w-10 shrink-0" />
                {hours.map((h) => (
                  <div
                    key={h}
                    className="flex-1 text-center text-[10px] text-gray-400"
                  >
                    {h % 3 === 0 ? `${h}` : ""}
                  </div>
                ))}
              </div>
              {/* Day rows */}
              {days.map((day) => (
                <div key={day} className="mb-0.5 flex items-center">
                  <div className="w-10 shrink-0 text-xs font-medium text-gray-500">{day}</div>
                  {hours.map((h) => {
                    const count = getCell(day, h);
                    return (
                      <div
                        key={h}
                        className={`group relative mx-0.5 h-7 flex-1 rounded ${heatColor(count)} cursor-default`}
                        title={`${day} ${h}:00 — ${count} failed logins`}
                      />
                    );
                  })}
                </div>
              ))}
              {/* Legend */}
              <div className="mt-3 flex items-center gap-2 text-xs text-gray-400">
                <span>Low</span>
                <div className="h-3 w-20 rounded bg-gradient-to-r from-green-200 via-amber-300 via-orange-400 to-red-500" />
                <span>High</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* ===== Anomaly Detection Alerts ===== */}
      <div className="mb-6">
        <div className={cardCls}>
          <h2 className={`flex items-center gap-2 ${headingCls}`}>
            <ShieldAlert className="h-5 w-5 text-brand-600" /> Anomaly Detection
          </h2>
          {anomalies.length === 0 ? (
            <p className="py-6 text-center text-sm text-gray-400">
              No active anomalies detected. All clear.
            </p>
          ) : (
            <div className="space-y-3">
              {anomalies.map((alert) => (
                <div
                  key={alert.id}
                  className="flex items-start justify-between rounded-lg border border-gray-200 p-4 dark:border-gray-700"
                >
                  <div className="flex items-start gap-3">
                    <span
                      className={`mt-0.5 rounded-full px-2 py-0.5 text-xs font-medium ${severityColor(alert.severity)}`}
                    >
                      {alert.severity}
                    </span>
                    <div>
                      <p className="text-sm font-medium text-gray-900 dark:text-gray-100">
                        {typeLabel(alert.type)}
                      </p>
                      <p className="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                        {alert.description}
                      </p>
                      <div className="mt-1.5 flex items-center gap-3 text-xs text-gray-400">
                        <span className="flex items-center gap-1">
                          <Clock className="h-3 w-3" />
                          {new Date(alert.timestamp).toLocaleString()}
                        </span>
                        <span className="flex items-center gap-1 font-mono">
                          <MapPin className="h-3 w-3" />
                          {alert.ip}
                        </span>
                      </div>
                    </div>
                  </div>
                  <button
                    onClick={() => dismissAnomaly(alert.id)}
                    aria-label={`Dismiss anomaly ${alert.id}`}
                    className="ml-2 rounded-lg border border-gray-300 p-1.5 text-gray-400 hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700"
                    title="Dismiss"
                  >
                    <X className="h-4 w-4" />
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* ===== IP Allowlist / Denylist ===== */}
      <div className="mb-6 grid gap-6 lg:grid-cols-2">
        {/* Allowlist */}
        <div className={cardCls}>
          <h2 className={`flex items-center gap-2 ${headingCls}`}>
            <ShieldCheck className="h-5 w-5 text-green-600" /> IP Allowlist
          </h2>
          <div className="mb-3 flex gap-2">
            <input
              value={allowInput}
              onChange={(e) => setAllowInput(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && addAllow()}
              placeholder="e.g. 10.0.0.0/8"
              aria-label="Allowlist IP or CIDR"
              className={`${inputCls} font-mono`}
            />
            <button
              onClick={addAllow}
              aria-label="Add to allowlist"
              className="flex items-center gap-1 rounded-lg bg-green-600 px-3 py-2 text-sm text-white hover:bg-green-700"
            >
              <Plus className="h-4 w-4" /> Allow
            </button>
          </div>
          <div className="space-y-2">
            {allowlist.length === 0 ? (
              <p className="text-sm text-gray-400">No IPs on allowlist</p>
            ) : (
              allowlist.map((ip) => (
                <div
                  key={ip}
                  className="flex items-center justify-between rounded-lg border border-green-200 bg-green-50 px-3 py-2 dark:border-green-800 dark:bg-green-950/40"
                >
                  <span className="font-mono text-sm text-gray-700 dark:text-gray-300">{ip}</span>
                  <button
                    onClick={() => removeAllow(ip)}
                    aria-label={`Remove ${ip} from allowlist`}
                    className="text-gray-400 hover:text-red-500"
                  >
                    <X className="h-4 w-4" />
                  </button>
                </div>
              ))
            )}
          </div>
        </div>

        {/* Denylist */}
        <div className={cardCls}>
          <h2 className={`flex items-center gap-2 ${headingCls}`}>
            <Ban className="h-5 w-5 text-red-600" /> IP Denylist
          </h2>
          <div className="mb-3 flex gap-2">
            <input
              value={denyInput}
              onChange={(e) => setDenyInput(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && addDeny()}
              placeholder="e.g. 198.51.100.10"
              aria-label="Denylist IP or CIDR"
              className={`${inputCls} font-mono`}
            />
            <button
              onClick={addDeny}
              aria-label="Add to denylist"
              className="flex items-center gap-1 rounded-lg bg-red-600 px-3 py-2 text-sm text-white hover:bg-red-700"
            >
              <Plus className="h-4 w-4" /> Deny
            </button>
          </div>
          <div className="space-y-2">
            {denylist.length === 0 ? (
              <p className="text-sm text-gray-400">No IPs on denylist</p>
            ) : (
              denylist.map((ip) => (
                <div
                  key={ip}
                  className="flex items-center justify-between rounded-lg border border-red-200 bg-red-50 px-3 py-2 dark:border-red-800 dark:bg-red-950/40"
                >
                  <span className="font-mono text-sm text-gray-700 dark:text-gray-300">{ip}</span>
                  <button
                    onClick={() => removeDeny(ip)}
                    aria-label={`Remove ${ip} from denylist`}
                    className="text-gray-400 hover:text-red-500"
                  >
                    <X className="h-4 w-4" />
                  </button>
                </div>
              ))
            )}
          </div>
        </div>
      </div>

      {/* ===== Security Recommendations ===== */}
      <div className={cardCls}>
        <h2 className={`flex items-center gap-2 ${headingCls}`}>
          <ShieldCheck className="h-5 w-5 text-brand-600" /> Security Recommendations
        </h2>
        <div className="space-y-2">
          {recommendations.map((rec) => (
            <div
              key={rec.id}
              className="flex items-start justify-between rounded-lg border border-gray-200 p-4 dark:border-gray-700"
            >
              <div className="flex items-start gap-3">
                <button onClick={() => toggleRec(rec.id)} aria-label={`Toggle recommendation ${rec.title}`} className="mt-0.5 shrink-0">
                  {rec.done ? (
                    <CheckCircle2 className="h-5 w-5 text-green-500" />
                  ) : (
                    <Circle className="h-5 w-5 text-gray-300 dark:text-gray-600" />
                  )}
                </button>
                <div>
                  <p
                    className={`text-sm font-medium ${
                      rec.done
                        ? "text-gray-400 line-through"
                        : "text-gray-900 dark:text-gray-100"
                    }`}
                  >
                    {rec.title}
                  </p>
                  <p className="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                    {rec.description}
                  </p>
                </div>
              </div>
              <span
                className={`ml-2 shrink-0 rounded-full px-2 py-0.5 text-xs font-medium ${
                  rec.done
                    ? "bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-400"
                    : "bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-400"
                }`}
              >
                {rec.done ? "Done" : "Action Needed"}
              </span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

// --- Threat Card sub-component ---

function ThreatCard({
  icon: Icon,
  label,
  value,
  color,
}: {
  icon: React.ElementType;
  label: string;
  value: number;
  color: "amber" | "red" | "orange";
}) {
  const colorMap = {
    amber: { bg: "bg-amber-100 dark:bg-amber-900/30", text: "text-amber-600 dark:text-amber-400" },
    red: { bg: "bg-red-100 dark:bg-red-900/30", text: "text-red-600 dark:text-red-400" },
    orange: { bg: "bg-orange-100 dark:bg-orange-900/30", text: "text-orange-600 dark:text-orange-400" },
  };
  const c = colorMap[color];
  return (
    <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800">
      <div className="flex items-center gap-3">
        <div className={`flex h-10 w-10 items-center justify-center rounded-lg ${c.bg}`}>
          <Icon className={`h-5 w-5 ${c.text}`} />
        </div>
        <div>
          <p className="text-2xl font-bold dark:text-gray-100">{value.toLocaleString()}</p>
          <p className="text-xs text-gray-500">{label}</p>
        </div>
      </div>
    </div>
  );
}

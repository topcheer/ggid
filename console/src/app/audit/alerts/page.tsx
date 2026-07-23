"use client";

import { useState, useEffect, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  Bell,
  Plus,
  Trash2,
  Loader2,
  CheckCircle2,
  XCircle,
  Clock,
  Filter,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface AlertHistoryEntry {
  id: string;
  rule_name: string;
  triggered_at: string;
  metric: string;
  value: number;
  threshold: number;
  action: string;
  status: "sent" | "failed";
  message: string;
}

interface AlertRule {
  id: string;
  name: string;
  enabled: boolean;
  metric: string;
  condition: string;
  threshold: number;
  last_triggered?: string;
  trigger_count: number;
}

export default function AuditAlertsPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [rules, setRules] = useState<AlertRule[]>([]);
  const [history, setHistory] = useState<AlertHistoryEntry[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [tab, setTab] = useState<"rules" | "history">("rules");
  const [statusFilter, setStatusFilter] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [rulesRes, historyRes] = await Promise.all([
        apiFetch<{ rules?: AlertRule[] }>("/api/v1/settings/alerting/rules").catch(() => ({ rules: [] })),
        apiFetch<{ alerts?: AlertHistoryEntry[] }>("/api/v1/audit/alerts").catch(() => ({ alerts: [] })),
      ]);
      setRules(rulesRes.rules ?? []);
      setHistory(historyRes.alerts ?? []);
    } catch {
      setError("Failed to load alert rules");
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => { load(); }, [load]);

  const handleToggle = async (id: string) => {
    const rule = rules.find((r: any) => r.id === id);
    if (!rule) return;
    setRules(rules.map((r: any) => (r.id === id ? { ...r, enabled: !r.enabled } : r)));
    try {
      await apiFetch(`/api/v1/settings/alerting/rules/${id}`, {
        method: "PATCH",
        body: JSON.stringify({ enabled: !rule.enabled }),
      });
    } catch { /* optimistic */ }
  };

  const handleDelete = async (id: string) => {
    setRules(rules.filter((r: any) => r.id !== id));
    try {
      await apiFetch(`/api/v1/settings/alerting/rules/${id}`, { method: "DELETE" });
    } catch { /* optimistic */ }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const filteredHistory = statusFilter ? history.filter((h: any) => h.status === statusFilter) : history;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
          <Bell className="h-7 w-7 text-indigo-600" />
          Audit Alerts
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Manage alerting rules and view triggered alert history.
        </p>
      </div>

      {/* Tabs */}
      <div className="flex gap-2 border-b border-gray-200 dark:border-gray-700">
        {(["rules", "history"] as const).map((t: any) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={`px-4 py-2 text-sm font-medium capitalize ${
              tab === t
                ? "border-b-2 border-indigo-600 text-indigo-600"
                : "text-gray-500 hover:text-gray-700 dark:text-gray-400"
            }`}
          >
            {t} {t === "rules" ? `(${rules.length})` : `(${history.length})`}
          </button>
        ))}
      </div>

      {loading ? (
        <div className="flex justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-indigo-600" />
        </div>
      ) : tab === "rules" ? (
        <div className="space-y-3">
          {rules.length === 0 ? (
            <div className={`${cardCls} text-center`}>
              <Bell className="mx-auto mb-3 h-12 w-12 text-gray-300" />
              <p className="text-gray-500 dark:text-gray-400">No alert rules configured.</p>
            </div>
          ) : (
            rules.map((rule: any) => (
              <div key={rule.id} className={cardCls}>
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <button
                      onClick={() => handleToggle(rule.id)}
                      className={`flex h-6 w-11 items-center rounded-full transition-colors ${
                        rule.enabled ? "bg-indigo-600" : "bg-gray-300 dark:bg-gray-600"
                      }`}
                    >
                      <span className={`h-5 w-5 transform rounded-full bg-white shadow transition-transform ${
                        rule.enabled ? "translate-x-5" : "translate-x-0.5"
                      }`} />
                    </button>
                    <div>
                      <span className="font-semibold text-gray-900 dark:text-white">{rule.name}</span>
                      <div className="mt-1 flex flex-wrap gap-3 text-xs text-gray-400">
                        <code className="rounded bg-gray-100 px-1.5 py-0.5 dark:bg-gray-700">{rule.metric}</code>
                        <span>{rule.condition} {rule.threshold}</span>
                        <span className="flex items-center gap-1"><Clock className="h-3 w-3" /> Triggered {rule.trigger_count}x</span>
                        {rule.last_triggered && (
                          <span>Last: {new Date(rule.last_triggered).toLocaleString()}</span>
                        )}
                      </div>
                    </div>
                  </div>
                  <button
                    onClick={() => handleDelete(rule.id)}
                    className="rounded-lg p-2 text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
              </div>
            ))
          )}
        </div>
      ) : (
        <div className="space-y-4">
          {/* Filter */}
          <div className="flex items-center gap-3">
            <Filter className="h-4 w-4 text-gray-400" />
            <select
              className="rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value)}
            >
              <option value="">All Statuses</option>
              <option value="sent">Sent</option>
              <option value="failed">Failed</option>
            </select>
          </div>

          {filteredHistory.length === 0 ? (
            <div className={`${cardCls} text-center`}>
              <Bell className="mx-auto mb-3 h-12 w-12 text-gray-300" />
              <p className="text-gray-500 dark:text-gray-400">No alert history.</p>
            </div>
          ) : (
            <div className={`${cardCls} overflow-hidden p-0`}>
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-gray-200 text-left text-xs uppercase text-gray-400 dark:border-gray-700">
                      <th scope="col" className="px-4 py-3">Rule</th>
                      <th scope="col" className="px-4 py-3">Metric</th>
                      <th scope="col" className="px-4 py-3">Value</th>
                      <th scope="col" className="px-4 py-3">Threshold</th>
                      <th scope="col" className="px-4 py-3">Action</th>
                      <th scope="col" className="px-4 py-3">Status</th>
                      <th scope="col" className="px-4 py-3">Time</th>
                    </tr>
                  </thead>
                  <tbody>
                    {filteredHistory.length === 0 ? (
                      <tr><td colSpan={7} className="px-4 py-8 text-center text-sm text-gray-400">No alert history yet. Alerts will appear here when rules trigger.</td></tr>
                    ) : filteredHistory.map((entry: any) => (
                      <tr key={entry.id} className="border-b border-gray-100 dark:border-gray-700/50">
                        <td className="px-4 py-3 font-medium text-gray-800 dark:text-gray-200">{entry.rule_name}</td>
                        <td className="px-4 py-3"><code className="rounded bg-gray-100 px-1.5 py-0.5 text-xs dark:bg-gray-700">{entry.metric}</code></td>
                        <td className="px-4 py-3 text-gray-600 dark:text-gray-300">{entry.value}</td>
                        <td className="px-4 py-3 text-gray-600 dark:text-gray-300">{entry.threshold}</td>
                        <td className="px-4 py-3 capitalize text-gray-600 dark:text-gray-300">{entry.action}</td>
                        <td className="px-4 py-3">
                          {entry.status === "sent" ? (
                            <span className="flex items-center gap-1 text-xs text-green-600"><CheckCircle2 className="h-3 w-3" /> Sent</span>
                          ) : (
                            <span className="flex items-center gap-1 text-xs text-red-600"><XCircle className="h-3 w-3" /> Failed</span>
                          )}
                        </td>
                        <td className="px-4 py-3 text-xs text-gray-400">{new Date(entry.triggered_at).toLocaleString()}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

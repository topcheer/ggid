"use client";
import { useEffect, useState } from "react";
import { useAuditRealtimeAlerts, ActiveAlert } from "@ggid/sdk-react";
import {
  Bell,
  BellOff,
  AlertTriangle,
  CheckCircle,
  ArrowUp,
  Zap,
  RefreshCw,
} from "lucide-react";

export default function AuditRealTimeAlertsPage() {
  const { data, loading, error, refresh, acknowledgeAlert, testAlert } = useAuditRealtimeAlerts();

  const [ack, setAck] = useState<Record<string, boolean>>({});

  useEffect(() => { refresh(); }, [refresh]);

  const severityColors: Record<string, string> = {
    critical: "bg-red-900/30 text-red-300 border-red-800",
    high: "bg-orange-900/30 text-orange-300 border-orange-800",
    medium: "bg-yellow-900/30 text-yellow-300 border-yellow-800",
    low: "bg-blue-900/30 text-blue-300 border-blue-800",
  };

  if (loading) return <div className="p-8 text-gray-400">Loading alerts...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const alerts = data?.active_alerts ?? [];

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Real-Time Alerts</h1>
          <p className="text-sm text-gray-400 mt-1">Live audit alert feed with escalation policies</p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => testAlert()}
            aria-label="Send test alert"
            className="flex items-center gap-1 px-3 py-2 bg-purple-600 hover:bg-purple-700 rounded-lg text-sm font-medium transition"
          >
            <Zap className="w-4 h-4" />
            Test Alert
          </button>
          <button
            onClick={refresh}
            aria-label="Refresh alerts"
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
          >
            <RefreshCw className="w-4 h-4 inline mr-1" /> Refresh
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Active Alerts Feed (2 cols) */}
        <div className="lg:col-span-2 bg-gray-900 rounded-xl p-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold flex items-center gap-2">
              <Bell className="w-5 h-5 text-yellow-400" />
              Active Alerts
            </h2>
            <span className="text-xs text-gray-400 auto-refresh-badge">Auto-refresh: 30s</span>
          </div>
          <div className="space-y-2 max-h-96 overflow-y-auto">
            {alerts.map((alert: ActiveAlert, i) => {
              const severity = alert.severity || "low";
              return (
                <div key={alert.id || i} className={"rounded-lg p-3 border " + (severityColors[severity] || severityColors.low)}>
                  <div className="flex items-start justify-between mb-1">
                    <div className="flex items-center gap-2">
                      <AlertTriangle className="w-4 h-4" />
                      <p className="text-sm font-medium">{alert.rule_name}</p>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="text-xs px-2 py-0.5 rounded bg-black/30 uppercase">{severity}</span>
                      <button
                        onClick={() => { acknowledgeAlert(alert.id); setAck({ ...ack, [alert.id]: true }); }}
                        disabled={ack[alert.id]}
                        aria-label={`Acknowledge alert ${alert.rule_name}`}
                        className="text-xs px-2 py-1 bg-gray-700 hover:bg-gray-600 rounded disabled:opacity-50"
                      >
                        <CheckCircle className="w-3 h-3 inline" /> Ack
                      </button>
                    </div>
                  </div>
                  <p className="text-xs opacity-80">{alert.message}</p>
                  <div className="flex items-center gap-3 mt-1 text-xs opacity-60">
                    <span>Channel: {alert.channel}</span>
                    <span>Triggered: {alert.triggered_at}</span>
                  </div>
                </div>
              );
            })}
            {alerts.length === 0 && (
              <p className="text-sm text-gray-500 text-center py-8">No active alerts.</p>
            )}
          </div>
        </div>

        {/* Sidebar: Rules + Escalation + Suppression */}
        <div className="space-y-6">
          {/* Alert Rules */}
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-sm font-semibold mb-3">Alert Rules</h2>
            <div className="space-y-2">
              {(data?.alert_rules ?? []).map((r) => (
                <div key={r.rule_name} className="bg-gray-800 rounded-lg p-2">
                  <div className="flex items-center justify-between mb-1">
                    <p className="text-xs font-medium">{r.rule_name}</p>
                    <span className={"text-xs px-1.5 py-0.5 rounded uppercase " + (
                      r.severity === "critical" ? "bg-red-900 text-red-300" :
                      r.severity === "high" ? "bg-orange-900 text-orange-300" :
                      "bg-yellow-900 text-yellow-300"
                    )}>
                      {r.severity}
                    </span>
                  </div>
                  <p className="text-xs text-gray-400">{r.condition}</p>
                  <p className="text-xs text-gray-500 mt-0.5">Channel: {r.channel}</p>
                </div>
              ))}
            </div>
          </div>

          {/* Suppression Rules */}
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-sm font-semibold flex items-center gap-2 mb-3">
              <BellOff className="w-4 h-4 text-gray-400" />
              Suppression Rules
            </h2>
            <div className="space-y-1">
              {(data?.alert_suppression_rules ?? []).map((s, i) => (
                <div key={i} className="flex items-center justify-between bg-gray-800 rounded p-2">
                  <span className="text-xs font-mono text-gray-300">{s.dedup_key}</span>
                  <span className="text-xs text-gray-400">{s.suppress_minutes}m</span>
                </div>
              ))}
            </div>
          </div>

          {/* Escalation Policy */}
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-sm font-semibold flex items-center gap-2 mb-3">
              <ArrowUp className="w-4 h-4 text-purple-400" />
              Escalation Policy
            </h2>
            <div className="space-y-2">
              {(data?.escalation_policy ?? []).map((e, i) => (
                <div key={i} className="bg-gray-800 rounded p-2">
                  <div className="flex items-center justify-between">
                    <span className="text-xs text-gray-300">Notify after {e.notify_after_minutes}m</span>
                    <span className="text-xs text-purple-400">{e.escalate_to}</span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

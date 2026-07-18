"use client";

import { useState, useEffect } from "react";
import { useApi } from "@/lib/api";
import {
  Bell,
  Plus,
  Trash2,
  Save,
  Loader2,
  AlertTriangle,
  BellRing,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface AlertRule {
  id: string;
  name: string;
  description: string;
  enabled: boolean;
  metric: string;
  condition: ">" | "<" | "=" | ">=" | "<=";
  threshold: number;
  window: number; // minutes
  action: "email" | "webhook" | "slack" | "pagerduty";
  target: string; // email address, webhook URL, slack channel, PD service
  cooldown: number; // minutes between repeat alerts
  lastTriggered?: string;
}

const METRICS = [
  { value: "failed_logins", label: "Failed Login Attempts" },
  { value: "successful_logins", label: "Successful Logins" },
  { value: "mfa_enrollment", label: "MFA Enrollment Rate" },
  { value: "new_users", label: "New User Registrations" },
  { value: "suspended_accounts", label: "Suspended Accounts" },
  { value: "api_errors", label: "API Error Rate" },
  { value: "audit_events", label: "Audit Events Volume" },
];

const ACTIONS = [
  { value: "email", label: "Send Email" },
  { value: "webhook", label: "Webhook" },
  { value: "slack", label: "Slack Notification" },
  { value: "pagerduty", label: "PagerDuty Alert" },
];

const CONDITIONS = [">", "<", "=", ">=", "<="];

const STORAGE_KEY = "ggid_alert_rules";

const defaultRule: Omit<AlertRule, "id"> = {
  name: "",
  description: "",
  enabled: true,
  metric: "failed_logins",
  condition: ">",
  threshold: 10,
  window: 5,
  action: "email",
  target: "",
  cooldown: 30,
};

export default function AlertingRulesPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [rules, setRules] = useState<AlertRule[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [msg, setMsg] = useState<string | null>(null);
  const [showAddForm, setShowAddForm] = useState(false);

  useEffect(() => {
    const load = async () => {
      try {
        const data = await apiFetch<{ rules?: AlertRule[] }>("/api/v1/settings/alerting/rules");
        setRules(data.rules ?? []);
      } catch {
        // Fall back to localStorage
        const stored = localStorage.getItem(STORAGE_KEY);
        if (stored) {
          setRules(JSON.parse(stored));
        }
      } finally {
        setLoading(false);
      }
    };
    load();
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const persistRules = async (updated: AlertRule[]) => {
    setRules(updated);
    setSaving(true);
    try {
      await apiFetch("/api/v1/settings/alerting/rules", {
        method: "PUT",
        body: JSON.stringify({ rules: updated }),
      });
      setMsg("Alert rules saved");
    } catch {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(updated));
      setMsg("Alert rules saved (offline mode)");
    } finally {
      setSaving(false);
      setTimeout(() => setMsg(null), 4000);
    }
  };

  const handleAddRule = () => {
    const newRule: AlertRule = {
      ...defaultRule,
      id: `rule-${Date.now()}`,
      name: `Rule ${rules.length + 1}`,
    };
    persistRules([...rules, newRule]);
    setShowAddForm(false);
  };

  const handleDeleteRule = (id: string) => {
    persistRules(rules.filter((r: any) => r.id !== id));
  };

  const handleToggleRule = (id: string) => {
    persistRules(
      rules.map((r: any) => (r.id === id ? { ...r, enabled: !r.enabled } : r))
    );
  };

  const handleUpdateRule = (id: string, field: keyof AlertRule, value: string | number | boolean) => {
    setRules(
      rules.map((r: any) =>
        r.id === id ? { ...r, [field]: value } : r
      )
    );
  };

  const handleSaveAll = () => {
    persistRules(rules);
  };

  const inputCls =
    "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";
  const smallInputCls =
    "rounded-lg border border-gray-300 px-2 py-1.5 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";
  const cardCls =
    "rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <Bell className="h-7 w-7 text-indigo-600" />
            Alerting Rules
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Configure automated alerts based on security metrics and thresholds.
          </p>
        </div>
        <div className="flex items-center gap-2">
          {msg && (
            <span className="text-sm text-green-600">{msg}</span>
          )}
          <button
            onClick={handleSaveAll}
            disabled={saving}
            className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
          >
            {saving ? (
              <Loader2 className="mr-1 inline h-4 w-4 animate-spin" />
            ) : (
              <Save className="mr-1 inline h-4 w-4" />
            )}
            Save All
          </button>
          <button
            onClick={() => setShowAddForm(true)}
            className="rounded-lg border border-indigo-300 px-4 py-2 text-sm font-medium text-indigo-600 hover:bg-indigo-50 dark:border-indigo-700 dark:text-indigo-400 dark:hover:bg-indigo-900/20"
          >
            <Plus className="mr-1 inline h-4 w-4" />
            Add Rule
          </button>
        </div>
      </div>

      {/* Add form */}
      {showAddForm && (
        <div className={`${cardCls} border-indigo-300 dark:border-indigo-700`}>
          <h3 className="mb-4 text-sm font-semibold text-gray-700 dark:text-gray-300">
            New Alert Rule
          </h3>
          <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
            <input
              className={inputCls}
              placeholder="Rule name"
              defaultValue=""
              onChange={(e) => (defaultRule.name = e.target.value)}
            />
            <select aria-label="Select option" className={inputCls} defaultValue={defaultRule.metric}>
              {METRICS.map((m: any) => (
                <option key={m.value} value={m.value}>{m.label}</option>
              ))}
            </select>
            <select aria-label="Select option" className={inputCls} defaultValue={defaultRule.action}>
              {ACTIONS.map((a: any) => (
                <option key={a.value} value={a.value}>{a.label}</option>
              ))}
            </select>
          </div>
          <div className="mt-3 flex gap-2">
            <button
              onClick={() => {
                defaultRule.name = defaultRule.name || `Rule ${rules.length + 1}`;
                handleAddRule();
              }}
              className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700"
            >
              Create Rule
            </button>
            <button
              onClick={() => setShowAddForm(false)}
              className="rounded-lg border border-gray-300 px-4 py-2 text-sm text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {/* Rules list */}
      {loading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-indigo-600" />
        </div>
      ) : rules.length === 0 ? (
        <div className={`${cardCls} text-center`}>
          <AlertTriangle className="mx-auto mb-3 h-12 w-12 text-gray-300" />
          <p className="text-gray-500 dark:text-gray-400">
            No alert rules configured. Click "Add Rule" to create one.
          </p>
        </div>
      ) : (
        <div className="space-y-4">
          {rules.map((rule: any) => (
            <div key={rule.id} className={cardCls}>
              <div className="flex items-start justify-between gap-4">
                {/* Left: name + description */}
                <div className="flex items-start gap-3">
                  <button
                    onClick={() => handleToggleRule(rule.id)}
                    className={`mt-0.5 flex h-6 w-11 items-center rounded-full transition-colors ${
                      rule.enabled ? "bg-indigo-600" : "bg-gray-300 dark:bg-gray-600"
                    }`}
                  >
                    <span
                      className={`h-5 w-5 transform rounded-full bg-white shadow transition-transform ${
                        rule.enabled ? "translate-x-5" : "translate-x-0.5"
                      }`}
                    />
                  </button>
                  <div>
                    <input
                      className="border-none bg-transparent text-base font-semibold text-gray-900 outline-none dark:text-white"
                      value={rule.name}
                      onChange={(e) => handleUpdateRule(rule.id, "name", e.target.value)}
                    />
                    <input
                      className="w-full border-none bg-transparent text-xs text-gray-400 outline-none"
                      placeholder="Add description..."
                      value={rule.description}
                      onChange={(e) => handleUpdateRule(rule.id, "description", e.target.value)}
                    />
                  </div>
                </div>

                {/* Right: delete */}
                <button
                  onClick={() => handleDeleteRule(rule.id)}
                  className="rounded-lg p-2 text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20"
                >
                  <Trash2 className="h-4 w-4" />
                </button>
              </div>

              {/* Condition row */}
              <div className="mt-4 flex flex-wrap items-center gap-3 text-sm text-gray-600 dark:text-gray-300">
                <span>When</span>
                <select
                  className={smallInputCls}
                  value={rule.metric}
                  onChange={(e) => handleUpdateRule(rule.id, "metric", e.target.value)}
                >
                  {METRICS.map((m: any) => (
                    <option key={m.value} value={m.value}>{m.label}</option>
                  ))}
                </select>
                <select
                  className={smallInputCls}
                  value={rule.condition}
                  onChange={(e) => handleUpdateRule(rule.id, "condition", e.target.value)}
                >
                  {CONDITIONS.map((c: any) => (
                    <option key={c} value={c}>{c}</option>
                  ))}
                </select>
                <input
                  type="number"
                  className={`${smallInputCls} w-20`}
                  value={rule.threshold}
                  onChange={(e) => handleUpdateRule(rule.id, "threshold", Number(e.target.value))}
                />
                <span>in the last</span>
                <input
                  type="number"
                  className={`${smallInputCls} w-16`}
                  value={rule.window}
                  onChange={(e) => handleUpdateRule(rule.id, "window", Number(e.target.value))}
                />
                <span>min</span>
              </div>

              {/* Action row */}
              <div className="mt-3 flex flex-wrap items-center gap-3 text-sm text-gray-600 dark:text-gray-300">
                <BellRing className="h-4 w-4 text-gray-400" />
                <select
                  className={smallInputCls}
                  value={rule.action}
                  onChange={(e) => handleUpdateRule(rule.id, "action", e.target.value)}
                >
                  {ACTIONS.map((a: any) => (
                    <option key={a.value} value={a.value}>{a.label}</option>
                  ))}
                </select>
                <input
                  className={`${smallInputCls} flex-1`}
                  placeholder={
                    rule.action === "email" ? "admin@example.com" :
                    rule.action === "webhook" ? "https://hooks.example.com/alert" :
                    rule.action === "slack" ? "#security-alerts" :
                    "service-id"
                  }
                  value={rule.target}
                  onChange={(e) => handleUpdateRule(rule.id, "target", e.target.value)}
                />
                <span>Cooldown:</span>
                <input
                  type="number"
                  className={`${smallInputCls} w-16`}
                  value={rule.cooldown}
                  onChange={(e) => handleUpdateRule(rule.id, "cooldown", Number(e.target.value))}
                />
                <span>min</span>
              </div>

              {/* Last triggered */}
              {rule.lastTriggered && (
                <div className="mt-2 text-xs text-gray-400">
                  Last triggered: {rule.lastTriggered}
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

"use client";

import { useState, useEffect, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  ShieldAlert, Plus, Trash2, X, AlertCircle, Loader2, Check,
  ShieldX, ShieldCheck, AlertTriangle,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface SoDRule {
  id: string;
  roles: string[];
  description: string;
  severity: "critical" | "high" | "medium";
  enabled: boolean;
  created_at: string;
}

interface SoDViolation {
  id: string;
  user_id: string;
  user_name: string;
  conflicting_roles: string[];
  rule_description: string;
  severity: "critical" | "high" | "medium";
  detected_at: string;
}

const SEVERITY_COLOR = {
  critical: { bg: "bg-red-100 dark:bg-red-900/30", text: "text-red-700 dark:text-red-400", border: "border-red-200 dark:border-red-800" },
  high: { bg: "bg-orange-100 dark:bg-orange-900/30", text: "text-orange-700 dark:text-orange-400", border: "border-orange-200 dark:border-orange-800" },
  medium: { bg: "bg-yellow-100 dark:bg-yellow-900/30", text: "text-yellow-700 dark:text-yellow-400", border: "border-yellow-200 dark:border-yellow-800" },
};

export default function SoDPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [rules, setRules] = useState<SoDRule[]>([]);
  const [violations, setViolations] = useState<SoDViolation[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [tab, setTab] = useState<"rules" | "violations">("rules");
  const [showAdd, setShowAdd] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState<SoDRule | null>(null);

  // Form
  const [form, setForm] = useState({
    roles: "",
    description: "",
    severity: "high" as SoDRule["severity"],
  });
  const [creating, setCreating] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [rulesRes, violRes] = await Promise.all([
        apiFetch<{ rules?: SoDRule[]; items?: SoDRule[] }>("/api/v1/policy/sod/rules").catch(() => ({ rules: [] as SoDRule[], items: [] as SoDRule[] })),
        apiFetch<{ violations?: SoDViolation[]; items?: SoDViolation[] }>("/api/v1/policy/sod/violations").catch(() => ({ violations: [] as SoDViolation[], items: [] as SoDViolation[] })),
      ]);
      setRules(rulesRes.rules ?? rulesRes.items ?? []);
      setViolations(violRes.violations ?? violRes.items ?? []);
    } catch {
      setError("Failed to load SoD data");
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => { load(); }, [load]);

  const handleCreate = async () => {
    const roleList = form.roles.split(",").map((r) => r.trim()).filter(Boolean);
    if (roleList.length < 2 || !form.description.trim()) return;
    setCreating(true);
    try {
      await apiFetch("/api/v1/policy/sod/rules", {
        method: "POST",
        body: JSON.stringify({ roles: roleList, description: form.description, severity: form.severity }),
      });
      setForm({ roles: "", description: "", severity: "high" });
      setShowAdd(false);
      await load();
    } catch {
      setError("Failed to create SoD rule");
    } finally {
      setCreating(false);
    }
  };

  const handleToggle = async (rule: SoDRule) => {
    try {
      await apiFetch(`/api/v1/policy/sod/rules/${rule.id}`, {
        method: "PATCH", body: JSON.stringify({ enabled: !rule.enabled }),
      });
      setRules((prev) => prev.map((r) => r.id === rule.id ? { ...r, enabled: !r.enabled } : r));
    } catch {
      setError("Failed to toggle rule");
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await apiFetch(`/api/v1/policy/sod/rules/${id}`, { method: "DELETE" });
      setConfirmDelete(null);
      await load();
    } catch {
      setError("Failed to delete rule");
    }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const violCount = violations.filter((v) => v.severity === "critical").length;
  const activeRules = rules.filter((r) => r.enabled).length;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <ShieldAlert className="h-6 w-6 text-indigo-600" /> {t("sod.title")}
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Define mutually exclusive roles and detect violations.
          </p>
        </div>
        <button onClick={() => setShowAdd(true)} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700">
          <Plus className="h-4 w-4" /> Add Rule
        </button>
      </div>

      {/* Summary cards */}
      <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
        <div className={cardCls}>
          <p className="text-xs font-medium text-gray-400">{t("backend3.sod.activeRules")}</p>
          <p className="mt-1 text-2xl font-bold text-indigo-600">{activeRules}</p>
        </div>
        <div className={cardCls}>
          <p className="text-xs font-medium text-gray-400">{t("backend3.sod.totalViolations")}</p>
          <p className="mt-1 text-2xl font-bold text-red-600">{violations.length}</p>
        </div>
        <div className={cardCls}>
          <p className="text-xs font-medium text-gray-400">{t("backend3.sod.critical")}</p>
          <p className="mt-1 text-2xl font-bold text-red-600">{violCount}</p>
        </div>
        <div className={cardCls}>
          <p className="text-xs font-medium text-gray-400">{t("backend3.sod.cleanUsers")}</p>
          <p className="mt-1 text-2xl font-bold text-green-600">
            {violations.length === 0 ? "All" : "Review"}
          </p>
        </div>
      </div>

      {error && (
        <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}
          <button onClick={() => setError(null)} className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      {/* Tabs */}
      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700">
        <button
          onClick={() => setTab("rules")}
          className={`flex items-center gap-2 border-b-2 px-4 py-2 text-sm font-medium ${tab === "rules" ? "border-indigo-600 text-indigo-600" : "border-transparent text-gray-400 hover:text-gray-600"}`}
        >
          <ShieldCheck className="h-4 w-4" /> Rules ({rules.length})
        </button>
        <button
          onClick={() => setTab("violations")}
          className={`flex items-center gap-2 border-b-2 px-4 py-2 text-sm font-medium ${tab === "violations" ? "border-indigo-600 text-indigo-600" : "border-transparent text-gray-400 hover:text-gray-600"}`}
        >
          <ShieldX className="h-4 w-4" /> Violations ({violations.length})
        </button>
      </div>

      {loading ? (
        <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>
      ) : tab === "rules" ? (
        rules.length === 0 ? (
          <div className={cardCls}>
            <div className="py-12 text-center">
              <ShieldAlert className="mx-auto h-12 w-12 text-gray-300" />
              <p className="mt-4 text-sm text-gray-400">No SoD rules configured. Add one to enforce role separation.</p>
            </div>
          </div>
        ) : (
          <div className="space-y-3">
            {rules.map((rule) => {
              const colors = SEVERITY_COLOR[rule.severity];
              return (
                <div key={rule.id} className={`${cardCls} ${rule.enabled ? "" : "opacity-60"}`}>
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="flex items-center gap-2">
                        <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${colors.bg} ${colors.text}`}>{rule.severity}</span>
                        <span className="text-sm font-medium text-gray-800 dark:text-gray-200">{rule.description}</span>
                      </div>
                      <div className="mt-2 flex items-center gap-2">
                        <div className="flex flex-wrap gap-1">
                          {rule.roles.map((role, i) => (
                            <span key={role} className="flex items-center gap-1">
                              <span className="rounded-lg bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-600 dark:bg-gray-700 dark:text-gray-300">{role}</span>
                              {i < rule.roles.length - 1 && <AlertTriangle className="h-3 w-3 text-orange-400" />}
                            </span>
                          ))}
                        </div>
                      </div>
                      <p className="mt-1 text-xs text-gray-400">Created: {new Date(rule.created_at).toLocaleDateString()}</p>
                    </div>
                    <div className="flex items-center gap-2">
                      <label className="relative inline-flex cursor-pointer items-center">
                        <input type="checkbox" checked={rule.enabled} onChange={() => handleToggle(rule)} className="peer sr-only" />
                        <div className="h-5 w-9 rounded-full bg-gray-200 after:absolute after:left-[2px] after:top-[2px] after:h-4 after:w-4 after:rounded-full after:border after:transition-all peer-checked:bg-indigo-600 peer-checked:after:translate-x-full dark:bg-gray-700" />
                      </label>
                      <button onClick={() => setConfirmDelete(rule)} className="rounded-lg p-1.5 text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20">
                        <Trash2 className="h-4 w-4" />
                      </button>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        )
      ) : (
        violations.length === 0 ? (
          <div className={cardCls}>
            <div className="py-12 text-center">
              <ShieldCheck className="mx-auto h-12 w-12 text-green-300" />
              <p className="mt-4 text-sm text-gray-400">No SoD violations detected. All users comply with separation rules.</p>
            </div>
          </div>
        ) : (
          <div className="space-y-3">
            {violations.map((v) => {
              const colors = SEVERITY_COLOR[v.severity];
              return (
                <div key={v.id} className={`${cardCls} ${colors.border}`}>
                  <div className="flex items-start gap-3">
                    <div className={`rounded-lg p-2 ${colors.bg}`}>
                      <ShieldX className={`h-5 w-5 ${colors.text}`} />
                    </div>
                    <div className="flex-1">
                      <div className="flex items-center gap-2">
                        <span className="font-medium text-gray-800 dark:text-gray-200">{v.user_name}</span>
                        <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${colors.bg} ${colors.text}`}>{v.severity}</span>
                      </div>
                      <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{v.rule_description}</p>
                      <div className="mt-2 flex flex-wrap gap-1">
                        {v.conflicting_roles.map((role, i) => (
                          <span key={role} className="flex items-center gap-1">
                            <span className="rounded-lg bg-red-50 px-2 py-0.5 text-xs font-medium text-red-600 dark:bg-red-900/20 dark:text-red-400">{role}</span>
                            {i < v.conflicting_roles.length - 1 && <span className="text-xs text-gray-400">+</span>}
                          </span>
                        ))}
                      </div>
                      <p className="mt-1 text-xs text-gray-400">Detected: {new Date(v.detected_at).toLocaleString()}</p>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        )
      )}

      {/* Add rule modal */}
      {showAdd && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowAdd(false)}>
          <div className="w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-semibold text-gray-900 dark:text-white">{t("backend3.sod.addRule")}</h2>
              <button onClick={() => setShowAdd(false)}><X className="h-5 w-5 text-gray-400" /></button>
            </div>
            <div className="mt-4 space-y-4">
              <div>
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300">Conflicting Roles (comma-separated, min 2)</label>
                <input value={form.roles} onChange={(e) => setForm((p) => ({ ...p, roles: e.target.value }))} placeholder="admin, auditor" className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" />
                <p className="mt-1 text-xs text-gray-400">These roles will be mutually exclusive — a user cannot hold all of them simultaneously.</p>
              </div>
              <div>
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300">{t("backend3.sod.description")}</label>
                <input value={form.description} onChange={(e) => setForm((p) => ({ ...p, description: e.target.value }))} placeholder="Admin and auditor roles are mutually exclusive" className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" />
              </div>
              <div>
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300">{t("backend3.sod.severity")}</label>
                <div className="mt-2 flex gap-2">
                  {(["critical", "high", "medium"] as const).map((s) => {
                    const colors = SEVERITY_COLOR[s];
                    return (
                      <button key={s} onClick={() => setForm((p) => ({ ...p, severity: s }))}
                        className={`flex-1 rounded-lg border px-3 py-2 text-sm font-medium capitalize ${form.severity === s ? `${colors.border} ${colors.bg} ${colors.text}` : "border-gray-300 text-gray-500 dark:border-gray-600"}`}>
                        {s}
                      </button>
                    );
                  })}
                </div>
              </div>
            </div>
            <div className="mt-6 flex justify-end gap-2">
              <button onClick={() => setShowAdd(false)} className="rounded-lg px-4 py-2 text-sm text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700">{t("backend3.sod.cancel")}</button>
              <button
                onClick={handleCreate}
                disabled={form.roles.split(",").filter((r) => r.trim()).length < 2 || !form.description.trim() || creating}
                className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
              >
                {creating ? <Loader2 className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />} Create Rule
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Delete confirmation */}
      {confirmDelete && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setConfirmDelete(null)}>
          <div className="w-full max-w-sm rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center gap-3">
              <div className="rounded-full bg-red-100 p-2 dark:bg-red-900/30"><Trash2 className="h-5 w-5 text-red-600" /></div>
              <div>
                <h2 className="font-semibold text-gray-900 dark:text-white">Delete SoD Rule?</h2>
                <p className="text-sm text-gray-500"><strong>{confirmDelete.description}</strong> will be removed. Future violations of this rule will not be detected.</p>
              </div>
            </div>
            <div className="mt-5 flex justify-end gap-2">
              <button onClick={() => setConfirmDelete(null)} className="rounded-lg px-4 py-2 text-sm text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700">{t("backend3.sod.cancel")}</button>
              <button onClick={() => handleDelete(confirmDelete.id)} className="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700">{t("backend3.sod.delete")}</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

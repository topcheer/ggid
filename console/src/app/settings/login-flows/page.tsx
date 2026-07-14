"use client";

import { useState, useEffect } from "react";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";
import { Workflow, Save, Loader2, ChevronUp, ChevronDown, Plus, Trash2, GripVertical, ArrowRight, Power } from "lucide-react";

interface FlowCondition {
  ip_range: string;
  risk_threshold: number;
  user_role: "admin" | "user" | "service" | "any";
}

interface FlowStep {
  id: string;
  type: "password" | "totp" | "webauthn" | "magic_link";
  label: string;
  condition: FlowCondition;
  enabled: boolean;
}

interface FlowConfig {
  name: string;
  enabled: boolean;
  steps: FlowStep[];
}

const STORAGE_KEY = "ggid_login_flow_config";

const STEP_TYPES: Record<string, string> = {
  password: "Password",
  totp: "OTP (TOTP)",
  webauthn: "WebAuthn",
  magic_link: "Email Magic Link",
};

const ROLE_OPTIONS: Array<{ value: string; label: string }> = [
  { value: "any", label: "Any role" },
  { value: "admin", label: "Admin" },
  { value: "user", label: "User" },
  { value: "service", label: "Service" },
];

const defaultFlow: FlowConfig = {
  name: "Default Login Flow",
  enabled: true,
  steps: [
    {
      id: crypto.randomUUID(),
      type: "password",
      label: STEP_TYPES.password,
      condition: { ip_range: "", risk_threshold: 0, user_role: "any" },
      enabled: true,
    },
    {
      id: crypto.randomUUID(),
      type: "totp",
      label: STEP_TYPES.totp,
      condition: { ip_range: "", risk_threshold: 50, user_role: "any" },
      enabled: true,
    },
  ],
};

function buildPreview(steps: FlowStep[]): string {
  const active = steps.filter((s) => s.enabled);
  if (active.length === 0) return "No active steps";
  return active.map((s, i) => {
    let suffix = "";
    if (s.condition.user_role !== "any") suffix += ` if role = ${s.condition.user_role}`;
    if (s.condition.risk_threshold > 0) suffix += ` if risk > ${s.condition.risk_threshold}`;
    if (s.condition.ip_range) suffix += ` if IP in ${s.condition.ip_range}`;
    return `Step ${i + 1}: ${s.label}${suffix}`;
  }).join(" → ");
}

export default function LoginFlowsPage() {
  const { apiFetch } = useApi();
  const t = useTranslations();
  const [flow, setFlow] = useState<FlowConfig>(defaultFlow);
  const [msg, setMsg] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [expandedStep, setExpandedStep] = useState<string | null>(null);
  const [dragIndex, setDragIndex] = useState<number | null>(null);

  const loadFlow = async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch("/api/v1/settings/login-flows", { headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const data = await res.json(); if (data) setFlow({ ...defaultFlow, ...data }); }
    } catch (err: any) { setError(err.message); }
    finally { setLoading(false); }
  };

  useEffect(() => { loadFlow(); }, []);

  useEffect(() => {
    if (msg) {
      const t = setTimeout(() => setMsg(null), 3000);
      return () => clearTimeout(t);
    }
  }, [msg]);

  const handleSave = async () => {
    setSaving(true);
    try {
      await apiFetch("/api/v1/settings/login-flows", {
        method: "POST",
        body: JSON.stringify(flow),
      });
      setMsg(t("flows.flowSavedServer"));
    } catch {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(flow));
      setMsg(t("flows.flowSavedLocal"));
    } finally {
      setSaving(false);
    }
  };

  const moveStep = (index: number, dir: "up" | "down") => {
    const target = dir === "up" ? index - 1 : index + 1;
    if (target < 0 || target >= flow.steps.length) return;
    const newSteps = [...flow.steps];
    [newSteps[index], newSteps[target]] = [newSteps[target], newSteps[index]];
    setFlow({ ...flow, steps: newSteps });
  };

  const removeStep = (id: string) => {
    setFlow({ ...flow, steps: flow.steps.filter((s) => s.id !== id) });
  };

  const addStep = (type: FlowStep["type"]) => {
    const newStep: FlowStep = {
      id: crypto.randomUUID(),
      type,
      label: STEP_TYPES[type],
      condition: { ip_range: "", risk_threshold: 0, user_role: "any" },
      enabled: true,
    };
    setFlow({ ...flow, steps: [...flow.steps, newStep] });
  };

  const updateStep = (id: string, updates: Partial<FlowStep>) => {
    setFlow({
      ...flow,
      steps: flow.steps.map((s) => (s.id === id ? { ...s, ...updates } : s)),
    });
  };

  const updateCondition = (id: string, updates: Partial<FlowCondition>) => {
    setFlow({
      ...flow,
      steps: flow.steps.map((s) =>
        s.id === id ? { ...s, condition: { ...s.condition, ...updates } } : s,
      ),
    });
  };

  // Drag and drop handlers
  const handleDragStart = (index: number) => {
    setDragIndex(index);
  };

  const handleDragOver = (e: React.DragEvent, index: number) => {
    e.preventDefault();
    if (dragIndex === null || dragIndex === index) return;
    const newSteps = [...flow.steps];
    const draggedItem = newSteps[dragIndex];
    newSteps.splice(dragIndex, 1);
    newSteps.splice(index, 0, draggedItem);
    setDragIndex(index);
    setFlow({ ...flow, steps: newSteps });
  };

  const handleDragEnd = () => {
    setDragIndex(null);
  };

  if (loading) return (
    <div className="p-8 flex items-center justify-center">
      <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-600" />
    </div>
  );

  if (error) return (
    <div className="p-8">
      <div className="rounded-lg border border-red-300 bg-red-50 dark:bg-red-950 dark:border-red-800 p-4">
        <p className="text-red-700 dark:text-red-400 text-sm font-medium">Error: {error}</p>
        <button onClick={loadFlow} className="mt-2 px-4 py-1.5 rounded-lg bg-red-600 text-white text-sm hover:bg-red-700">Retry</button>
      </div>
    </div>
  );

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-2xl font-bold dark:text-gray-100">
          <Workflow className="h-6 w-6 text-brand-600" /> {t("flows.builder")}
        </h1>
        <button
          onClick={handleSave}
          disabled={saving}
          className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
        >
          {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />} {t("flows.saveFlow")}
        </button>
      </div>

      {msg && (
        <div className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400">
          {msg}
        </div>
      )}

      {/* Flow-level enable toggle */}
      <div className="mb-4 flex items-center gap-3 rounded-xl border border-gray-200 bg-white px-4 py-3 shadow-sm dark:border-gray-700 dark:bg-gray-800">
        <button
          onClick={() => setFlow({ ...flow, enabled: !flow.enabled })}
          className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${flow.enabled ? "bg-brand-600" : "bg-gray-300 dark:bg-gray-600"}`}
        >
          <span className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${flow.enabled ? "translate-x-6" : "translate-x-1"}`} />
        </button>
        <div className="flex-1">
          <input
            value={flow.name}
            onChange={(e) => setFlow({ ...flow, name: e.target.value })}
            className="text-sm font-semibold bg-transparent text-gray-900 dark:text-gray-100 outline-none border-b border-transparent focus:border-brand-500"
          />
          <p className="text-xs text-gray-500">
            {flow.enabled ? t("flows.flowActive") : t("flows.flowDisabled")}
          </p>
        </div>
      </div>

      {/* Flow Preview */}
      <div className="mb-4 rounded-xl border border-blue-200 bg-blue-50 p-4 dark:border-blue-800 dark:bg-blue-950/30">
        <h3 className="mb-2 text-xs font-semibold uppercase text-blue-700 dark:text-blue-400">{t("flows.flowPreview")}</h3>
        <div className="flex flex-wrap items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
          {flow.steps.filter((s) => s.enabled).length === 0 ? (
            <span className="text-gray-400">{t("flows.noActiveSteps")}</span>
          ) : (
            flow.steps.filter((s) => s.enabled).map((step, i, arr) => (
              <span key={step.id} className="flex items-center gap-2">
                <span className="inline-flex items-center gap-1.5 rounded-lg bg-white px-3 py-1.5 text-xs font-medium shadow-sm dark:bg-gray-800">
                  <span className="flex h-5 w-5 items-center justify-center rounded-full bg-brand-100 text-brand-700 dark:bg-brand-900 dark:text-brand-300 text-xs font-bold">
                    {i + 1}
                  </span>
                  {step.label}
                  {step.condition.user_role !== "any" && (
                    <span className="text-xs text-purple-600 dark:text-purple-400">[{step.condition.user_role}]</span>
                  )}
                  {step.condition.risk_threshold > 0 && (
                    <span className="text-xs text-orange-600 dark:text-orange-400">[risk&gt;{step.condition.risk_threshold}]</span>
                  )}
                </span>
                {i < arr.length - 1 && <ArrowRight className="h-3.5 w-3.5 text-gray-400" />}
              </span>
            ))
          )}
        </div>
        <p className="mt-2 text-xs text-gray-500 font-mono break-all">{buildPreview(flow.steps)}</p>
      </div>

      {/* Steps */}
      <div className="space-y-3">
        {flow.steps.map((step, index) => (
          <div
            key={step.id}
            draggable
            onDragStart={() => handleDragStart(index)}
            onDragOver={(e) => handleDragOver(e, index)}
            onDragEnd={handleDragEnd}
            className={`rounded-xl border bg-white shadow-sm transition-shadow dark:bg-gray-800 ${
              dragIndex === index ? "border-brand-400 opacity-50" : "border-gray-200 dark:border-gray-700"
            } ${!step.enabled ? "opacity-60" : ""}`}
          >
            <div className="flex items-center gap-3 p-4">
              {/* Drag handle */}
              <GripVertical className="h-5 w-5 text-gray-300 cursor-grab active:cursor-grabbing" />

              {/* Step number */}
              <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-brand-100 text-sm font-bold text-brand-700 dark:bg-brand-900 dark:text-brand-300">
                {index + 1}
              </span>

              {/* Label */}
              <div className="flex-1">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-semibold text-gray-900 dark:text-gray-100">{step.label}</span>
                  {step.condition.user_role !== "any" && (
                    <span className="rounded bg-purple-100 px-1.5 py-0.5 text-xs text-purple-700 dark:bg-purple-900 dark:text-purple-300">
                      role={step.condition.user_role}
                    </span>
                  )}
                  {step.condition.risk_threshold > 0 && (
                    <span className="rounded bg-orange-100 px-1.5 py-0.5 text-xs text-orange-700 dark:bg-orange-900 dark:text-orange-300">
                      risk&gt;{step.condition.risk_threshold}
                    </span>
                  )}
                  {step.condition.ip_range && (
                    <span className="rounded bg-blue-100 px-1.5 py-0.5 text-xs text-blue-700 dark:bg-blue-900 dark:text-blue-300">
                      IP: {step.condition.ip_range}
                    </span>
                  )}
                </div>
              </div>

              {/* Enable/disable toggle */}
              <button
                onClick={() => updateStep(step.id, { enabled: !step.enabled })}
                title={step.enabled ? t("flows.disableStep") : t("flows.enableStep")}
                className={`relative inline-flex h-5 w-9 items-center rounded-full transition-colors ${step.enabled ? "bg-brand-600" : "bg-gray-300 dark:bg-gray-600"}`}
              >
                <span className={`inline-block h-3.5 w-3.5 transform rounded-full bg-white transition-transform ${step.enabled ? "translate-x-5" : "translate-x-1"}`} />
              </button>

              {/* Move up/down */}
              <div className="flex flex-col">
                <button
                  onClick={() => moveStep(index, "up")}
                  disabled={index === 0}
                  className="text-gray-400 hover:text-brand-600 disabled:opacity-30"
                >
                  <ChevronUp className="h-4 w-4" />
                </button>
                <button
                  onClick={() => moveStep(index, "down")}
                  disabled={index === flow.steps.length - 1}
                  className="text-gray-400 hover:text-brand-600 disabled:opacity-30"
                >
                  <ChevronDown className="h-4 w-4" />
                </button>
              </div>

              {/* Expand conditions */}
              <button
                onClick={() => setExpandedStep(expandedStep === step.id ? null : step.id)}
                className="rounded-lg border border-gray-300 px-2.5 py-1 text-xs font-medium text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
              >
                {expandedStep === step.id ? t("flows.hideConditions") : t("flows.showConditions")}
              </button>

              {/* Delete */}
              <button
                onClick={() => removeStep(step.id)}
                className="rounded p-1 text-gray-400 hover:bg-red-50 hover:text-red-600"
              >
                <Trash2 className="h-4 w-4" />
              </button>
            </div>

            {/* Conditions panel */}
            {expandedStep === step.id && (
              <div className="border-t border-gray-100 bg-gray-50/50 p-4 dark:border-gray-700 dark:bg-gray-900/30">
                <h4 className="mb-3 text-xs font-semibold uppercase text-gray-500">Conditions for this step</h4>
                <div className="grid gap-4 sm:grid-cols-3">
                  {/* IP range */}
                  <div>
                    <label className="mb-1 block text-xs font-medium text-gray-500">{t("flows.ipRangeCidr")}</label>
                    <input
                      value={step.condition.ip_range}
                      onChange={(e) => updateCondition(step.id, { ip_range: e.target.value })}
                      placeholder="10.0.0.0/8"
                      className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm font-mono dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
                    />
                    <p className="mt-1 text-xs text-gray-400">Only apply if user IP is in this range</p>
                  </div>
                  {/* Risk threshold */}
                  <div>
                    <label className="mb-1 block text-xs font-medium text-gray-500">
                      {t("flows.riskThreshold")}: {step.condition.risk_threshold}
                    </label>
                    <input
                      type="range"
                      min={0}
                      max={100}
                      value={step.condition.risk_threshold}
                      onChange={(e) => updateCondition(step.id, { risk_threshold: parseInt(e.target.value) })}
                      className="w-full accent-brand-600"
                    />
                    <p className="mt-1 text-xs text-gray-400">Apply this step if risk &gt; {step.condition.risk_threshold}</p>
                  </div>
                  {/* User role */}
                  <div>
                    <label className="mb-1 block text-xs font-medium text-gray-500">{t("flows.userRole")}</label>
                    <select
                      value={step.condition.user_role}
                      onChange={(e) =>
                        updateCondition(step.id, { user_role: e.target.value as FlowCondition["user_role"] })
                      }
                      className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
                    >
                      {ROLE_OPTIONS.map((opt) => (
                        <option key={opt.value} value={opt.value}>
                          {opt.label}
                        </option>
                      ))}
                    </select>
                    <p className="mt-1 text-xs text-gray-400">Only apply for specific role</p>
                  </div>
                </div>
              </div>
            )}
          </div>
        ))}
      </div>

      {/* Add step buttons */}
      <div className="mt-4">
        <h3 className="mb-2 text-xs font-semibold uppercase text-gray-500">{t("flows.addStep")}</h3>
        <div className="flex flex-wrap gap-2">
          {(Object.keys(STEP_TYPES) as FlowStep["type"][]).map((type) => (
            <button
              key={type}
              onClick={() => addStep(type)}
              className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
            >
              <Plus className="h-4 w-4" /> {STEP_TYPES[type]}
            </button>
          ))}
        </div>
      </div>

      {flow.steps.length === 0 && (
        <div className="mt-4 rounded-xl border border-dashed border-gray-300 p-8 text-center dark:border-gray-600">
          <Power className="mx-auto mb-2 h-8 w-8 text-gray-300" />
          <p className="text-sm text-gray-400">{t("flows.noSteps")}</p>
        </div>
      )}
    </div>
  );
}

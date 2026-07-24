"use client";

import { useState, useEffect, useCallback } from "react";
import { useTranslations } from "@/lib/i18n";
import { useConfirm } from "@/components/ConfirmDialog";
import { authHeader } from "@/lib/auth-helpers";
import {
  Shield, KeyRound, Lock, Save, Loader2, Plus, Trash2, Edit2,
  Check, X, AlertCircle, Eye, EyeOff,
} from "lucide-react";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "";

// ============ Types ============

interface PasswordPolicy {
  min_length: number;
  require_uppercase: boolean;
  require_lowercase: boolean;
  require_digit: boolean;
  require_special: boolean;
  prevent_username: boolean;
  prevent_common: boolean;
  history_count: number;
  expiry_days: number;
  hibp_check: boolean;
}

interface LockoutPolicy {
  max_attempts: number;
  lockout_duration: number;
  ip_max_attempts: number;
  ip_lockout_duration: number;
}

interface MethodPolicy {
  id: string;
  group: string;
  required_methods: string[];
  forbidden_methods: string[];
}

type TabId = "methodPolicies" | "passwordPolicy" | "lockoutPolicy";

const AUTH_METHODS = ["password", "webauthn", "totp", "sms", "email", "social", "saml"];

const DEFAULT_PASSWORD: PasswordPolicy = {
  min_length: 12,
  require_uppercase: true,
  require_lowercase: true,
  require_digit: true,
  require_special: true,
  prevent_username: true,
  prevent_common: true,
  history_count: 5,
  expiry_days: 90,
  hibp_check: true,
};

const DEFAULT_LOCKOUT: LockoutPolicy = {
  max_attempts: 5,
  lockout_duration: 900,
  ip_max_attempts: 20,
  ip_lockout_duration: 3600,
};

// ============ Page ============

export default function SecurityPolicyPage() {
  const t = useTranslations();
  const { confirm: showConfirm } = useConfirm();
  const [activeTab, setActiveTab] = useState<TabId>("passwordPolicy");

  const tabs: { id: TabId; label: string; icon: typeof Shield }[] = [
    { id: "methodPolicies", label: t("securityPolicy.tabs.methodPolicies"), icon: Shield },
    { id: "passwordPolicy", label: t("securityPolicy.tabs.passwordPolicy"), icon: KeyRound },
    { id: "lockoutPolicy", label: t("securityPolicy.tabs.lockoutPolicy"), icon: Lock },
  ];

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-800 dark:bg-gray-950 p-4 md:p-8">
      <div className="max-w-5xl mx-auto">
        {/* Header */}
        <div className="mb-6">
          <div className="flex items-center gap-3 mb-2">
            <Shield className="w-7 h-7 text-blue-600" />
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white dark:text-white">
              {t("securityPolicy.title")}
            </h1>
          </div>
          <p className="text-gray-600 dark:text-gray-400 dark:text-gray-400 text-sm">
            {t("securityPolicy.description")}
          </p>
        </div>

        {/* Tabs */}
        <div className="flex gap-1 mb-6 bg-gray-200 dark:bg-gray-800 rounded-lg p-1">
          {tabs.map((tab: any) => {
            const Icon = tab.icon;
            return (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                  activeTab === tab.id
                    ? "bg-white dark:bg-gray-700 text-blue-600 dark:text-blue-400 shadow-sm"
                    : "text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white"
                }`}
              >
                <Icon className="w-4 h-4" />
                {tab.label}
              </button>
            );
          })}
        </div>

        {/* Tab Content */}
        {activeTab === "passwordPolicy" && <PasswordPolicyTab />}
        {activeTab === "lockoutPolicy" && <LockoutPolicyTab />}
        {activeTab === "methodPolicies" && <MethodPoliciesTab />}
      </div>
    </div>
  );
}

// ============ Password Policy Tab ============

function PasswordPolicyTab() {
  const t = useTranslations();
  const [config, setConfig] = useState<PasswordPolicy>(DEFAULT_PASSWORD);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [msg, setMsg] = useState<{ type: "success" | "error"; text: string } | null>(null);
  const [testPw, setTestPw] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch(`${API_BASE}/api/v1/auth/password/policy`, {
        headers: { ...authHeader() },
      });
      if (res.ok) {
        const data = await res.json();
        setConfig({ ...DEFAULT_PASSWORD, ...data });
      }
    } catch {
      // Use defaults
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  const save = async () => {
    setSaving(true);
    try {
      const res = await fetch(`${API_BASE}/api/v1/auth/password/policy`, {
        method: "PUT",
        headers: { "Content-Type": "application/json", ...authHeader() },
        body: JSON.stringify(config),
      });
      if (!res.ok) throw new Error("save failed");
      setMsg({ type: "success", text: t("securityPolicy.passwordPolicy.policySaved") });
    } catch {
      setMsg({ type: "error", text: t("settings.endpointUnavailable") });
    } finally {
      setSaving(false);
      setTimeout(() => setMsg(null), 3000);
    }
  };

  const toggle = (key: keyof PasswordPolicy) =>
    setConfig((c) => ({ ...c, [key]: !c[key] }));

  const testPasswordStrength = (): { score: number; checks: { label: string; pass: boolean }[] } => {
    const checks = [
      { label: t("securityPolicy.passwordPolicy.minLength"), pass: testPw.length >= config.min_length },
      { label: t("securityPolicy.passwordPolicy.requireUppercase"), pass: !config.require_uppercase || /[A-Z]/.test(testPw) },
      { label: t("securityPolicy.passwordPolicy.requireLowercase"), pass: !config.require_lowercase || /[a-z]/.test(testPw) },
      { label: t("securityPolicy.passwordPolicy.requireDigit"), pass: !config.require_digit || /\d/.test(testPw) },
      { label: t("securityPolicy.passwordPolicy.requireSpecial"), pass: !config.require_special || /[!@#$%^&*(),.?":{}|<>]/.test(testPw) },
    ];
    const score = checks.filter((c: any) => c.pass).length;
    return { score, checks };
  };

  if (loading) {
    return (
      <div className="flex justify-center py-20">
        <Loader2 className="w-8 h-8 animate-spin text-blue-600" />
      </div>
    );
  }

  const { score, checks } = testPasswordStrength();

  return (
    <div className="bg-white dark:bg-gray-800 dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-700 dark:border-gray-800 p-6 space-y-6">
      {/* Config Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Number fields */}
        <div className="space-y-4">
          <NumberField
            label={t("securityPolicy.passwordPolicy.minLength")}
            value={config.min_length}
            onChange={(v) => setConfig({ ...config, min_length: v })}
            min={4}
            max={128}
          />
          <NumberField
            label={t("securityPolicy.passwordPolicy.historyCount")}
            value={config.history_count}
            onChange={(v) => setConfig({ ...config, history_count: v })}
            min={0}
            max={24}
          />
          <NumberField
            label={t("securityPolicy.passwordPolicy.expiryDays")}
            value={config.expiry_days}
            onChange={(v) => setConfig({ ...config, expiry_days: v })}
            min={0}
            max={365}
          />
        </div>

        {/* Toggle fields */}
        <div className="space-y-3">
          <ToggleRow label={t("securityPolicy.passwordPolicy.requireUppercase")} checked={config.require_uppercase} onChange={() => toggle("require_uppercase")} />
          <ToggleRow label={t("securityPolicy.passwordPolicy.requireLowercase")} checked={config.require_lowercase} onChange={() => toggle("require_lowercase")} />
          <ToggleRow label={t("securityPolicy.passwordPolicy.requireDigit")} checked={config.require_digit} onChange={() => toggle("require_digit")} />
          <ToggleRow label={t("securityPolicy.passwordPolicy.requireSpecial")} checked={config.require_special} onChange={() => toggle("require_special")} />
          <ToggleRow label={t("securityPolicy.passwordPolicy.preventUsername")} checked={config.prevent_username} onChange={() => toggle("prevent_username")} />
          <ToggleRow label={t("securityPolicy.passwordPolicy.preventCommon")} checked={config.prevent_common} onChange={() => toggle("prevent_common")} />
          <ToggleRow label={t("securityPolicy.passwordPolicy.hibpCheck")} checked={config.hibp_check} onChange={() => toggle("hibp_check")} />
        </div>
      </div>

      {/* Password Strength Preview */}
      <div className="border-t border-gray-200 dark:border-gray-700 dark:border-gray-800 pt-4">
        <h3 className="text-sm font-semibold text-gray-700 dark:text-gray-300 dark:text-gray-300 mb-2">
          {t("securityPolicy.passwordPolicy.preview")}
        </h3>
        <input
          type="text"
          value={testPw}
          onChange={(e) => setTestPw(e.target.value)}
          placeholder={t("securityPolicy.passwordPolicy.testPassword")}
          className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 dark:bg-gray-800 text-sm text-gray-900 dark:text-white dark:text-white mb-3"
        />
        {testPw && (
          <div className="space-y-1">
            <div className="flex gap-1">
              {[0, 1, 2, 3, 4].map((i: any) => (
                <div
                  key={i}
                  className={`h-1.5 flex-1 rounded-full ${
                    i < score
                      ? score >= 4 ? "bg-green-500" : score >= 3 ? "bg-yellow-500" : "bg-red-500"
                      : "bg-gray-200 dark:bg-gray-700"
                  }`}
                />
              ))}
            </div>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-1">
              {checks.map((c: any, i: number) => (
                <div key={i} className="flex items-center gap-1 text-xs">
                  {c.pass ? <Check className="w-3 h-3 text-green-500" /> : <X className="w-3 h-3 text-red-500" />}
                  <span className={c.pass ? "text-green-700 dark:text-green-400" : "text-red-700 dark:text-red-400"}>
                    {c.label}
                  </span>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* Actions */}
      {msg && (
        <div className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm ${
          msg.type === "success" ? "bg-green-50 text-green-700 dark:bg-green-950 dark:text-green-300" : "bg-red-50 text-red-700 dark:bg-red-950 dark:text-red-300"
        }`}>
          {msg.type === "success" ? <Check className="w-4 h-4" /> : <AlertCircle className="w-4 h-4" />}
          {msg.text}
        </div>
      )}
      <button
        onClick={save}
        disabled={saving}
        className="flex items-center gap-2 px-6 py-2.5 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg font-medium text-sm transition-colors"
      >
        {saving ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
        {t("securityPolicy.passwordPolicy.save")}
      </button>
    </div>
  );
}

// ============ Lockout Policy Tab ============

function LockoutPolicyTab() {
  const t = useTranslations();
  const [config, setConfig] = useState<LockoutPolicy>(DEFAULT_LOCKOUT);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [msg, setMsg] = useState<{ type: "success" | "error"; text: string } | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch(`${API_BASE}/api/v1/auth/lockout-policy`, {
        headers: { ...authHeader() },
      });
      if (res.ok) {
        const data = await res.json();
        setConfig({ ...DEFAULT_LOCKOUT, ...data });
      }
    } catch {
      // Use defaults
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  const save = async () => {
    setSaving(true);
    try {
      const res = await fetch(`${API_BASE}/api/v1/auth/lockout-policy`, {
        method: "PUT",
        headers: { "Content-Type": "application/json", ...authHeader() },
        body: JSON.stringify(config),
      });
      if (!res.ok) throw new Error("save failed");
      setMsg({ type: "success", text: t("securityPolicy.lockoutPolicy.policySaved") });
    } catch {
      setMsg({ type: "error", text: t("settings.endpointUnavailable") });
    } finally {
      setSaving(false);
      setTimeout(() => setMsg(null), 3000);
    }
  };

  if (loading) {
    return (
      <div className="flex justify-center py-20">
        <Loader2 className="w-8 h-8 animate-spin text-blue-600" />
      </div>
    );
  }

  return (
    <div className="bg-white dark:bg-gray-800 dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-700 dark:border-gray-800 p-6 space-y-6">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="space-y-4">
          <div className="p-4 rounded-lg bg-blue-50 dark:bg-blue-950/30 border border-blue-200 dark:border-blue-900">
            <div className="flex items-center gap-2 mb-3">
              <Lock className="w-5 h-5 text-blue-600" />
              <h3 className="text-sm font-semibold text-gray-900 dark:text-white dark:text-white">
                {t("securityPolicy.lockoutPolicy.title")}
              </h3>
            </div>
            <NumberField
              label={t("securityPolicy.lockoutPolicy.maxAttempts")}
              value={config.max_attempts}
              onChange={(v) => setConfig({ ...config, max_attempts: v })}
              min={1}
              max={20}
            />
            <div className="mt-3">
              <NumberField
                label={t("securityPolicy.lockoutPolicy.lockoutDuration")}
                value={config.lockout_duration}
                onChange={(v) => setConfig({ ...config, lockout_duration: v })}
                min={30}
                max={86400}
              />
            </div>
          </div>
        </div>

        <div className="space-y-4">
          <div className="p-4 rounded-lg bg-orange-50 dark:bg-orange-950/30 border border-orange-200 dark:border-orange-900">
            <div className="flex items-center gap-2 mb-3">
              <Shield className="w-5 h-5 text-orange-600" />
              <h3 className="text-sm font-semibold text-gray-900 dark:text-white dark:text-white">
                IP-Level
              </h3>
            </div>
            <NumberField
              label={t("securityPolicy.lockoutPolicy.ipMaxAttempts")}
              value={config.ip_max_attempts}
              onChange={(v) => setConfig({ ...config, ip_max_attempts: v })}
              min={1}
              max={100}
            />
            <div className="mt-3">
              <NumberField
                label={t("securityPolicy.lockoutPolicy.ipLockoutDuration")}
                value={config.ip_lockout_duration}
                onChange={(v) => setConfig({ ...config, ip_lockout_duration: v })}
                min={60}
                max={86400}
              />
            </div>
          </div>
        </div>
      </div>

      {msg && (
        <div className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm ${
          msg.type === "success" ? "bg-green-50 text-green-700 dark:bg-green-950 dark:text-green-300" : "bg-red-50 text-red-700 dark:bg-red-950 dark:text-red-300"
        }`}>
          {msg.type === "success" ? <Check className="w-4 h-4" /> : <AlertCircle className="w-4 h-4" />}
          {msg.text}
        </div>
      )}
      <button
        onClick={save}
        disabled={saving}
        className="flex items-center gap-2 px-6 py-2.5 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg font-medium text-sm transition-colors"
      >
        {saving ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
        {t("securityPolicy.lockoutPolicy.save")}
      </button>
    </div>
  );
}

// ============ Method Policies Tab ============

function MethodPoliciesTab() {
  const t = useTranslations();
  const [policies, setPolicies] = useState<MethodPolicy[]>([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [editing, setEditing] = useState<MethodPolicy | null>(null);
  const [msg, setMsg] = useState<{ type: "success" | "error"; text: string } | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch(`${API_BASE}/api/v1/auth/method-policies`, {
        headers: { ...authHeader() },
      });
      if (res.ok) {
        const data = await res.json();
        setPolicies(Array.isArray(data) ? data : (data.policies || []));
      }
    } catch {
      // No policies yet
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  const handleDelete = async (id: string) => {
    showConfirm({
      title: t("securityPolicy.methodPolicies.deletePolicy"),
      description: t("securityPolicy.methodPolicies.confirmDelete"),
      variant: "danger" as const,
      onConfirm: async () => {
        await fetch(`${API_BASE}/api/v1/auth/method-policies/${id}`, {
          method: "DELETE",
          headers: { ...authHeader() },
        });
        setPolicies(policies.filter((p: any) => p.id !== id));
        setMsg({ type: "success", text: t("securityPolicy.methodPolicies.policyDeleted") });
      },
    });
  };

  const handleSaved = () => {
    setShowForm(false);
    setEditing(null);
    load();
    setMsg({ type: "success", text: t("securityPolicy.methodPolicies.policySaved") });
    setTimeout(() => setMsg(null), 3000);
  };

  if (loading) {
    return (
      <div className="flex justify-center py-20">
        <Loader2 className="w-8 h-8 animate-spin text-blue-600" />
      </div>
    );
  }

  return (
    <div className="bg-white dark:bg-gray-800 dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-700 dark:border-gray-800 p-6 space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-sm font-semibold text-gray-900 dark:text-white dark:text-white">
            {t("securityPolicy.methodPolicies.title")}
          </h3>
          <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
            {t("securityPolicy.methodPolicies.description")}
          </p>
        </div>
        <button
          onClick={() => { setEditing(null); setShowForm(true); }}
          className="flex items-center gap-1.5 px-3 py-1.5 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-medium"
        >
          <Plus className="w-4 h-4" />
          {t("securityPolicy.methodPolicies.addPolicy")}
        </button>
      </div>

      {msg && (
        <div className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm ${
          msg.type === "success" ? "bg-green-50 text-green-700 dark:bg-green-950 dark:text-green-300" : "bg-red-50 text-red-700 dark:bg-red-950 dark:text-red-300"
        }`}>
          {msg.type === "success" ? <Check className="w-4 h-4" /> : <AlertCircle className="w-4 h-4" />}
          {msg.text}
        </div>
      )}

      {showForm && (
        <MethodPolicyForm
          editing={editing}
          onSaved={handleSaved}
          onCancel={() => { setShowForm(false); setEditing(null); }}
        />
      )}

      {/* Policies Table */}
      {policies.length === 0 && !showForm ? (
        <div className="text-center py-12 text-gray-500 dark:text-gray-400">
          <Shield className="w-12 h-12 mx-auto mb-3 opacity-30" />
          <p className="text-sm">{t("securityPolicy.methodPolicies.noPolicies")}</p>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-200 dark:border-gray-700 dark:border-gray-800 text-left">
                <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400 dark:text-gray-400">
                  {t("securityPolicy.methodPolicies.group")}
                </th>
                <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400 dark:text-gray-400">
                  {t("securityPolicy.methodPolicies.requiredMethods")}
                </th>
                <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400 dark:text-gray-400">
                  {t("securityPolicy.methodPolicies.forbiddenMethods")}
                </th>
                <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400 dark:text-gray-400 text-right">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody>
              {policies.map((p: any) => (
                <tr key={p.id} className="border-b border-gray-100 dark:border-gray-800/50">
                  <td className="py-3 px-3 font-medium text-gray-900 dark:text-white dark:text-white">{p.group}</td>
                  <td className="py-3 px-3">
                    <div className="flex flex-wrap gap-1">
                      {p.required_methods.map((m: any) => (
                        <span key={m} className="px-2 py-0.5 text-xs bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-300 rounded">
                          {t(`securityPolicy.methodPolicies.${m}`)}
                        </span>
                      ))}
                    </div>
                  </td>
                  <td className="py-3 px-3">
                    <div className="flex flex-wrap gap-1">
                      {p.forbidden_methods.map((m: any) => (
                        <span key={m} className="px-2 py-0.5 text-xs bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300 rounded">
                          {t(`securityPolicy.methodPolicies.${m}`)}
                        </span>
                      ))}
                    </div>
                  </td>
                  <td className="py-3 px-3 text-right">
                    <button
                      onClick={() => { setEditing(p); setShowForm(true); }}
                      className="p-1.5 hover:bg-gray-100 dark:hover:bg-gray-700 dark:bg-gray-700 dark:hover:bg-gray-800 rounded"
                    >
                      <Edit2 className="w-4 h-4 text-gray-500" />
                    </button>
                    <button
                      onClick={() => handleDelete(p.id)}
                      className="p-1.5 hover:bg-red-50 dark:hover:bg-red-950 rounded"
                    >
                      <Trash2 className="w-4 h-4 text-red-500" />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

function MethodPolicyForm({ editing, onSaved, onCancel }: {
  editing: MethodPolicy | null;
  onSaved: () => void;
  onCancel: () => void;
}) {
  const t = useTranslations();
  const [group, setGroup] = useState(editing?.group || "");
  const [required, setRequired] = useState<Set<string>>(new Set(editing?.required_methods || []));
  const [forbidden, setForbidden] = useState<Set<string>>(new Set(editing?.forbidden_methods || []));
  const [saving, setSaving] = useState(false);

  const toggleMethod = (method: string, set: Set<string>, setter: (s: Set<string>) => void) => {
    const next = new Set(set);
    if (next.has(method)) next.delete(method);
    else next.add(method);
    setter(next);
  };

  const save = async () => {
    setSaving(true);
    try {
      const url = editing
        ? `${API_BASE}/api/v1/auth/method-policies/${editing.id}`
        : `${API_BASE}/api/v1/auth/method-policies`;
      const method = editing ? "PUT" : "POST";
      await fetch(url, {
        method,
        headers: { "Content-Type": "application/json", ...authHeader() },
        body: JSON.stringify({
          group,
          required_methods: [...required],
          forbidden_methods: [...forbidden],
        }),
      });
      onSaved();
    } catch {
      // Error
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="border border-gray-200 dark:border-gray-700 dark:border-gray-700 rounded-lg p-4 space-y-4 bg-gray-50 dark:bg-gray-800 dark:bg-gray-800/50">
      <div>
        <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 dark:text-gray-400 mb-1">
          {t("securityPolicy.methodPolicies.group")}
        </label>
        <input
          type="text"
          value={group}
          onChange={(e) => setGroup(e.target.value)}
          placeholder={t("securityPolicy.methodPolicies.selectGroup")}
          className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 dark:border-gray-700 bg-white dark:bg-gray-800 dark:bg-gray-800 text-sm text-gray-900 dark:text-white dark:text-white"
        />
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div>
          <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 dark:text-gray-400 mb-2">
            {t("securityPolicy.methodPolicies.requiredMethods")}
          </label>
          <div className="space-y-1">
            {AUTH_METHODS.map((m: any) => (
              <label key={m} className="flex items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  checked={required.has(m)}
                  onChange={() => toggleMethod(m, required, setRequired)}
                  className="rounded"
                />
                {t(`securityPolicy.methodPolicies.${m}`)}
              </label>
            ))}
          </div>
        </div>
        <div>
          <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 dark:text-gray-400 mb-2">
            {t("securityPolicy.methodPolicies.forbiddenMethods")}
          </label>
          <div className="space-y-1">
            {AUTH_METHODS.map((m: any) => (
              <label key={m} className="flex items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  checked={forbidden.has(m)}
                  onChange={() => toggleMethod(m, forbidden, setForbidden)}
                  className="rounded"
                />
                {t(`securityPolicy.methodPolicies.${m}`)}
              </label>
            ))}
          </div>
        </div>
      </div>

      <div className="flex gap-2">
        <button
          onClick={save}
          disabled={saving || !group}
          className="flex items-center gap-1.5 px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg text-sm font-medium"
        >
          {saving ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
          Save
        </button>
        <button
          onClick={onCancel}
          className="px-4 py-2 bg-gray-200 dark:bg-gray-700 hover:bg-gray-300 dark:hover:bg-gray-600 text-gray-700 dark:text-gray-300 dark:text-gray-300 rounded-lg text-sm font-medium"
        >
          Cancel
        </button>
      </div>
    </div>
  );
}

// ============ Shared Components ============

function NumberField({ label, value, onChange, min, max }: {
  label: string;
  value: number;
  onChange: (v: number) => void;
  min: number;
  max: number;
}) {
  return (
    <div>
      <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 dark:text-gray-400 mb-1">{label}</label>
      <input
        type="number"
        value={value}
        onChange={(e) => {
          const v = parseInt(e.target.value) || 0;
          onChange(Math.max(min, Math.min(max, v)));
        }}
        min={min}
        max={max}
        className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 dark:bg-gray-800 text-sm text-gray-900 dark:text-white dark:text-white"
      />
    </div>
  );
}

function ToggleRow({ label, checked, onChange }: {
  label: string;
  checked: boolean;
  onChange: () => void;
}) {
  return (
    <label className="flex items-center justify-between cursor-pointer py-1">
      <span className="text-sm text-gray-700 dark:text-gray-300 dark:text-gray-300">{label}</span>
      <button
        onClick={onChange}
        className={`relative w-10 h-6 rounded-full transition-colors ${
          checked ? "bg-blue-600" : "bg-gray-300 dark:bg-gray-600"
        }`}
      >
        <span className={`absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full transition-transform ${
          checked ? "translate-x-4" : ""
        }`} />
      </button>
    </label>
  );
}

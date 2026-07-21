"use client";

import { useState, useEffect, useCallback } from "react";
import { useApi } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import {
  Save,
  Loader2,
  ShieldCheck,
  Check,
  X,
  KeyRound,
  History,
  Clock,
} from "lucide-react";

interface PasswordPolicyConfig {
  min_length: number;
  require_uppercase: boolean;
  require_lowercase: boolean;
  require_digit: boolean;
  require_special: boolean;
  prevent_username: boolean;
  prevent_common: boolean;
  history_count: number;
  expiry_days: number;
}

const STORAGE_KEY = "ggid_password_policy";

const defaultConfig: PasswordPolicyConfig = {
  min_length: 12,
  require_uppercase: true,
  require_lowercase: true,
  require_digit: true,
  require_special: true,
  prevent_username: true,
  prevent_common: false,
  history_count: 5,
  expiry_days: 90,
};

const COMMON_PASSWORDS = [
  "password", "12345678", "qwerty", "abc123", "letmein",
  "admin", "welcome", "monkey", "dragon", "master",
  "sunshine", "iloveyou", "princess", "passw0rd", "football",
  "baseball", "superman", "trustno1", "hello123", "charlie",
];

function Toggle({
  checked,
  onChange,
  label,
  description,
}: {
  checked: boolean;
  onChange: (v: boolean) => void;
  label: string;
  description?: string;
}) {
  return (
    <label className="flex cursor-pointer items-center justify-between py-2">
      <div>
        <span className="text-sm font-medium text-gray-700 dark:text-gray-300 dark:text-gray-300">{label}</span>
        {description && (
          <p className="text-xs text-gray-400">{description}</p>
        )}
      </div>
      <button
        type="button"
        onClick={() => onChange(!checked)}
        className={`relative inline-flex h-6 w-11 shrink-0 items-center rounded-full transition-colors ${
          checked ? "bg-brand-600" : "bg-gray-300 dark:bg-gray-600"
        }`}
      >
        <span
          className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
            checked ? "translate-x-6" : "translate-x-1"
          }`}
        />
      </button>
    </label>
  );
}

function RuleCheck({ passes, label }: { passes: boolean; label: string }) {
  return (
    <div className="flex items-center gap-2 py-1">
      {passes ? (
        <Check className="h-4 w-4 text-green-500" />
      ) : (
        <X className="h-4 w-4 text-red-400" />
      )}
      <span className={passes ? "text-sm text-green-600 dark:text-green-400" : "text-sm text-gray-400 line-through"}>
        {label}
      </span>
    </div>
  );
}

export default function PasswordPolicyPage() {
  const { apiFetch, TENANT_ID } = useApi();
  const { t } = useI18n();
  const [config, setConfig] = useState<PasswordPolicyConfig>(defaultConfig);
  const [msg, setMsg] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [testPw, setTestPw] = useState("");
  const [testUsername, setTestUsername] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Load from API (fallback to localStorage for offline)
  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const data = await apiFetch<any>(`/api/v1/auth/password/policy`);
        if (!cancelled && data) {
          // Map backend fields to frontend config
          setConfig({
            min_length: data.min_length ?? data.MinLength ?? defaultConfig.min_length,
            require_uppercase: data.require_uppercase ?? data.require_upper ?? data.RequireUpper ?? defaultConfig.require_uppercase,
            require_lowercase: data.require_lowercase ?? data.require_lower ?? data.RequireLower ?? defaultConfig.require_lowercase,
            require_digit: data.require_digit ?? data.RequireDigit ?? defaultConfig.require_digit,
            require_special: data.require_special ?? data.RequireSpecial ?? defaultConfig.require_special,
            prevent_username: data.prevent_username ?? defaultConfig.prevent_username,
            prevent_common: data.prevent_common ?? (data.Blacklist?.length > 0) ?? defaultConfig.prevent_common,
            history_count: data.history_count ?? data.HistoryCount ?? defaultConfig.history_count,
            expiry_days: data.expiry_days ?? data.max_age_days ?? data.MaxAgeDays ?? defaultConfig.expiry_days,
          });
        }
      } catch {
        // Fallback to localStorage if API unavailable
        const stored = typeof window !== "undefined" ? localStorage.getItem(STORAGE_KEY) : null;
        if (stored) {
          try { setConfig({ ...defaultConfig, ...JSON.parse(stored) }); } catch { /* ignore */ }
        }
      }
      if (!cancelled) setLoading(false);
    })();
    return () => { cancelled = true; };
  }, []);

  useEffect(() => {
    if (msg) {
      const t = setTimeout(() => setMsg(null), 3000);
      return () => clearTimeout(t);
    }
  }, [msg]);

  const handleSave = async () => {
    setSaving(true);
    try {
      // Use the password-policy config endpoint (POST /api/v1/auth/password-policy)
      await apiFetch(`/api/v1/auth/password-policy`, {
        method: "POST",
        body: JSON.stringify({
          min_length: config.min_length,
          require_uppercase: config.require_uppercase,
          require_lowercase: config.require_lowercase,
          require_digit: config.require_digit,
          require_special: config.require_special,
          blacklist: config.prevent_common ? COMMON_PASSWORDS : [],
        }),
      });
      setMsg(t("passwordPolicy.policySaved"));
    } catch {
      // Fallback: also try the security/password-policy PUT endpoint
      try {
        await apiFetch(`/api/v1/security/password-policy`, {
          method: "PUT",
          body: JSON.stringify({
            min_length: config.min_length,
            require_upper: config.require_uppercase,
            require_lower: config.require_lowercase,
            require_digit: config.require_digit,
            require_special: config.require_special,
            history_count: config.history_count,
            blacklist: config.prevent_common ? COMMON_PASSWORDS : [],
          }),
        });
        setMsg(t("passwordPolicy.policySaved"));
      } catch {
        // Last resort: save to localStorage
        localStorage.setItem(STORAGE_KEY, JSON.stringify(config));
        setMsg(t("settings.endpointUnavailable"));
      }
    } finally {
      setSaving(false);
    }
  };

  // Live validation against current policy
  const checkRule = useCallback(
    (rule: keyof PasswordPolicyConfig): boolean => {
      if (!testPw) return false;
      switch (rule) {
        case "min_length":
          return testPw.length >= config.min_length;
        case "require_uppercase":
          return /[A-Z]/.test(testPw);
        case "require_lowercase":
          return /[a-z]/.test(testPw);
        case "require_digit":
          return /[0-9]/.test(testPw);
        case "require_special":
          return /[^A-Za-z0-9]/.test(testPw);
        case "prevent_username":
          if (!config.prevent_username || !testUsername) return true;
          return !testPw.toLowerCase().includes(testUsername.toLowerCase());
        case "prevent_common":
          if (!config.prevent_common) return true;
          return !COMMON_PASSWORDS.includes(testPw.toLowerCase());
        default:
          return false;
      }
    },
    [testPw, testUsername, config],
  );

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-6 w-6 animate-spin text-brand-500" />
        <span className="ml-2 text-sm text-gray-500">Loading password policy...</span>
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-lg border border-red-200 bg-red-50 dark:border-red-900 dark:bg-red-950/30 p-4">
        <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
        <button onClick={() => window.location.reload()} aria-label="Retry loading password policy" className="mt-2 text-xs text-red-600 underline">Retry</button>
      </div>
    );
  }

  return (
    <div className="max-w-3xl">
      <div className="mb-6 flex items-center gap-3">
        <ShieldCheck className="h-7 w-7 text-brand-600" />
        <div>
          <h1 className="text-2xl font-bold dark:text-gray-100">{t("passwordPolicy.title")}</h1>
          <p className="text-sm text-gray-500">
            Configure password complexity rules, history, and expiry
          </p>
        </div>
      </div>

      {msg && (
        <div role="status" className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400">
          {msg}
        </div>
      )}

      <div className="space-y-6">
        {/* Min Length */}
        <div className="rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold text-gray-700 dark:text-gray-300 dark:text-gray-300">
            <KeyRound className="h-4 w-4 text-brand-600" /> Minimum Length
          </h2>
          <div className="flex items-center gap-4">
            <input
              type="number"
              min={8}
              max={128}
              value={config.min_length}
              onChange={(e) => {
                const v = Math.max(8, Math.min(128, Number(e.target.value) || 8));
                setConfig({ ...config, min_length: v });
              }}
              className="w-24 rounded-lg border border-gray-300 dark:border-gray-600 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100"
            />
            <span className="text-sm text-gray-400">characters</span>
          </div>
          <input
            type="range"
            min={8}
            max={128}
            value={config.min_length}
            onChange={(e) => setConfig({ ...config, min_length: Number(e.target.value) })}
            className="mt-3 w-full accent-brand-600"
          />
          <div className="mt-1 flex justify-between text-xs text-gray-400">
            <span>8</span>
            <span>128</span>
          </div>
        </div>

        {/* Complexity Rules */}
        <div className="rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <h2 className="mb-2 text-sm font-semibold text-gray-700 dark:text-gray-300 dark:text-gray-300">
            Complexity Rules
          </h2>
          <div className="divide-y divide-gray-100 dark:divide-gray-700">
            <Toggle
              checked={config.require_uppercase}
              onChange={(v) => setConfig({ ...config, require_uppercase: v })}
              label={t("passwordPolicy.requireUppercase")}
              description={t("settings.atLeastUpper")}
            />
            <Toggle
              checked={config.require_lowercase}
              onChange={(v) => setConfig({ ...config, require_lowercase: v })}
              label={t("passwordPolicy.requireLowercase")}
              description={t("settings.atLeastLower")}
            />
            <Toggle
              checked={config.require_digit}
              onChange={(v) => setConfig({ ...config, require_digit: v })}
              label={t("passwordPolicy.requireDigits")}
              description={t("settings.atLeastDigit")}
            />
            <Toggle
              checked={config.require_special}
              onChange={(v) => setConfig({ ...config, require_special: v })}
              label={t("passwordPolicy.requireSpecial")}
              description={t("settings.atLeastSpecial")}
            />
            <Toggle
              checked={config.prevent_username}
              onChange={(v) => setConfig({ ...config, prevent_username: v })}
              label={t("passwordPolicy.preventUsername")}
              description={t("settings.noUsername")}
            />
            <Toggle
              checked={config.prevent_common}
              onChange={(v) => setConfig({ ...config, prevent_common: v })}
              label={t("passwordPolicy.preventCommon")}
              description={t("settings.dictPassword")}
            />
          </div>
        </div>

        {/* Password History */}
        <div className="rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold text-gray-700 dark:text-gray-300 dark:text-gray-300">
            <History className="h-4 w-4 text-brand-600" /> Password History
          </h2>
          <label className="mb-1 block text-xs font-medium text-gray-500">
            Prevent reuse of last N passwords (0 = disabled)
          </label>
          <div className="flex items-center gap-4">
            <input
              type="number"
              min={0}
              max={24}
              value={config.history_count}
              onChange={(e) => {
                const v = Math.max(0, Math.min(24, Number(e.target.value) || 0));
                setConfig({ ...config, history_count: v });
              }}
              className="w-24 rounded-lg border border-gray-300 dark:border-gray-600 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100"
            />
            <span className="text-sm text-gray-400">passwords</span>
          </div>
          <input
            type="range"
            min={0}
            max={24}
            value={config.history_count}
            onChange={(e) => setConfig({ ...config, history_count: Number(e.target.value) })}
            className="mt-3 w-full accent-brand-600"
          />
          <div className="mt-1 flex justify-between text-xs text-gray-400">
            <span>0</span>
            <span>24</span>
          </div>
        </div>

        {/* Password Expiry */}
        <div className="rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold text-gray-700 dark:text-gray-300 dark:text-gray-300">
            <Clock className="h-4 w-4 text-brand-600" /> Password Expiry
          </h2>
          <label className="mb-1 block text-xs font-medium text-gray-500">
            Days until forced change (0 = never)
          </label>
          <div className="flex items-center gap-4">
            <input
              type="number"
              min={0}
              max={365}
              value={config.expiry_days}
              onChange={(e) => {
                const v = Math.max(0, Math.min(365, Number(e.target.value) || 0));
                setConfig({ ...config, expiry_days: v });
              }}
              className="w-24 rounded-lg border border-gray-300 dark:border-gray-600 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100"
            />
            <span className="text-sm text-gray-400">days</span>
          </div>
          <input
            type="range"
            min={0}
            max={365}
            value={config.expiry_days}
            onChange={(e) => setConfig({ ...config, expiry_days: Number(e.target.value) })}
            className="mt-3 w-full accent-brand-600"
          />
          <div className="mt-1 flex justify-between text-xs text-gray-400">
            <span>0 (never)</span>
            <span>365</span>
          </div>
        </div>

        {/* Live Preview */}
        <div className="rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold text-gray-700 dark:text-gray-300 dark:text-gray-300">
            <ShieldCheck className="h-4 w-4 text-brand-600" /> Live Password Preview
          </h2>
          <p className="mb-3 text-xs text-gray-400">
            Type a test password to see which rules it passes or fails against the current policy.
          </p>
          {config.prevent_username && (
            <input
              type="text"
              value={testUsername}
              onChange={(e) => setTestUsername(e.target.value)}
              placeholder={t("settings.testUsername")}
              className="mb-2 w-full rounded-lg border border-gray-300 dark:border-gray-600 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100"
            />
          )}
          <input
            type="text"
            value={testPw}
            onChange={(e) => setTestPw(e.target.value)}
            placeholder={t("settings.testPassword")}
            className="w-full rounded-lg border border-gray-300 dark:border-gray-600 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100"
          />
          {testPw && (
            <div className="mt-3 space-y-0.5">
              <RuleCheck passes={checkRule("min_length")} label={`At least ${config.min_length} characters`} />
              <RuleCheck
                passes={checkRule("require_uppercase") || !config.require_uppercase}
                label="Contains uppercase letter"
              />
              <RuleCheck
                passes={checkRule("require_lowercase") || !config.require_lowercase}
                label="Contains lowercase letter"
              />
              <RuleCheck
                passes={checkRule("require_digit") || !config.require_digit}
                label={t("settings.containsDigit")}
              />
              <RuleCheck
                passes={checkRule("require_special") || !config.require_special}
                label="Contains special character"
              />
              {config.prevent_username && (
                <RuleCheck passes={checkRule("prevent_username")} label="Does not contain username" />
              )}
              {config.prevent_common && (
                <RuleCheck passes={checkRule("prevent_common")} label="Not a common password" />
              )}
            </div>
          )}
        </div>

        {/* Save */}
        <div className="flex justify-end">
          <button
            onClick={handleSave}
            disabled={saving}
            className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-6 py-2.5 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
          >
            {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
            Save Policy
          </button>
        </div>
      </div>
    </div>
  );
}

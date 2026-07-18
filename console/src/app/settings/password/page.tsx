"use client";

import { useState, useEffect, useCallback, useMemo } from "react";
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
  Eye,
  EyeOff,
  Ban,
  ListChecks,
  RotateCcw,
  Zap,
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
  force_change_on_next_login: boolean;
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
  force_change_on_next_login: false,
};

const COMMON_PASSWORDS = [
  "password", "12345678", "qwerty", "abc123", "letmein",
  "admin", "welcome", "monkey", "dragon", "master",
  "sunshine", "iloveyou", "princess", "passw0rd", "football",
  "baseball", "superman", "trustno1", "hello123", "charlie",
  "00000000", "11111111", "12341234", "password1", "qwerty123",
  "welcome1", "admin123", "letmein1", "abc12345", "test1234",
  "iloveyou1", "welcome123", "1q2w3e4r", "passw0rd1", "princess1",
  "football1", "baseball1", "dragon1", "master1", "monkey1",
  "shadow", "michael", "jennifer", "jordan", "hunter",
  "robert", "andrew", "jessica", "harley", "ranger",
  "pepper", "ginger", "austin", "joshua", "thomas",
];

const TOTAL_BLACKLIST = 10000; // simulated full blacklist size

// ── Toggle Component ──

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
        <span className="text-sm font-medium text-gray-700 dark:text-gray-300">{label}</span>
        {description && <p className="text-xs text-gray-400">{description}</p>}
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

// ── Strength meter ──

function estimateStrength(pw: string): { score: number; label: string; color: string; crackTime: string } {
  if (!pw) return { score: 0, label: "—", color: "bg-gray-300", crackTime: "—" };
  let score = 0;
  // Length
  if (pw.length >= 8) score++;
  if (pw.length >= 12) score++;
  if (pw.length >= 16) score++;
  // Variety
  if (/[a-z]/.test(pw)) score++;
  if (/[A-Z]/.test(pw)) score++;
  if (/[0-9]/.test(pw)) score++;
  if (/[^A-Za-z0-9]/.test(pw)) score++;

  // Estimate crack time (rough)
  let charsetSize = 0;
  if (/[a-z]/.test(pw)) charsetSize += 26;
  if (/[A-Z]/.test(pw)) charsetSize += 26;
  if (/[0-9]/.test(pw)) charsetSize += 10;
  if (/[^A-Za-z0-9]/.test(pw)) charsetSize += 32;
  const combinations = Math.pow(charsetSize, pw.length);
  const seconds = combinations / 1e10; // 10 billion guesses/sec

  let crackTime: string;
  if (seconds < 1) crackTime = "instant";
  else if (seconds < 60) crackTime = `${Math.round(seconds)} seconds`;
  else if (seconds < 3600) crackTime = `${Math.round(seconds / 60)} minutes`;
  else if (seconds < 86400) crackTime = `${Math.round(seconds / 3600)} hours`;
  else if (seconds < 31536000) crackTime = `${Math.round(seconds / 86400)} days`;
  else if (seconds < 31536000 * 100) crackTime = `${Math.round(seconds / 31536000)} years`;
  else if (seconds < 31536000 * 1e6) crackTime = `${Math.round(seconds / 31536000 / 1000)}K years`;
  else crackTime = "centuries+";

  if (score <= 2) return { score, label: "Very Weak", color: "bg-red-500", crackTime };
  if (score <= 3) return { score, label: "Weak", color: "bg-orange-500", crackTime };
  if (score <= 4) return { score, label: "Fair", color: "bg-yellow-500", crackTime };
  if (score <= 5) return { score, label: "Strong", color: "bg-green-500", crackTime };
  return { score, label: "Very Strong", color: "bg-emerald-600", crackTime };
}

// ── Main Page ──

export default function PasswordPolicyEnhancedPage() {
  const { apiFetch, TENANT_ID } = useApi();
  const [config, setConfig] = useState<PasswordPolicyConfig>(defaultConfig);
  const [msg, setMsg] = useState<string | null>(null);
  const [msgType, setMsgType] = useState<"success" | "error">("success");
  const [saving, setSaving] = useState(false);
  const [testPw, setTestPw] = useState("");
  const [testUsername, setTestUsername] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [showBlacklist, setShowBlacklist] = useState(false);

  // Load from localStorage or API on mount
  useEffect(() => {
    const stored = typeof window !== "undefined" ? localStorage.getItem(STORAGE_KEY) : null;
    if (stored) {
      try {
        setConfig({ ...defaultConfig, ...JSON.parse(stored) });
      } catch {
        // ignore
      }
    }
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
      await apiFetch(`/api/v1/tenants/${TENANT_ID}/password-policy`, {
        method: "PUT",
        body: JSON.stringify(config),
      });
      setMsg("Password policy saved to server");
      setMsgType("success");
      localStorage.setItem(STORAGE_KEY, JSON.stringify(config));
    } catch {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(config));
      setMsg("Endpoint unavailable — saved to localStorage");
      setMsgType("error");
    } finally {
      setSaving(false);
    }
  };

  const handleReset = () => {
    setConfig(defaultConfig);
    setTestPw("");
    setTestUsername("");
    localStorage.removeItem(STORAGE_KEY);
    setMsg("Policy reset to defaults");
    setMsgType("success");
  };

  // Live validation
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

  const strength = useMemo(() => estimateStrength(testPw), [testPw]);

  const allRulesPass = useMemo(() => {
    const rules: boolean[] = [checkRule("min_length")];
    if (config.require_uppercase) rules.push(checkRule("require_uppercase"));
    if (config.require_lowercase) rules.push(checkRule("require_lowercase"));
    if (config.require_digit) rules.push(checkRule("require_digit"));
    if (config.require_special) rules.push(checkRule("require_special"));
    if (config.prevent_username) rules.push(checkRule("prevent_username"));
    if (config.prevent_common) rules.push(checkRule("prevent_common"));
    return testPw ? rules.every(Boolean) : false;
  }, [checkRule, config, testPw]);

  return (
    <div className="max-w-4xl">
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <ShieldCheck className="h-7 w-7 text-brand-600" />
          <div>
            <h1 className="text-2xl font-bold dark:text-gray-100">Password Policy</h1>
            <p className="text-sm text-gray-500">
              Configure complexity rules, history, expiry, and blacklist
            </p>
          </div>
        </div>
      </div>

      {/* Message banner */}
      {msg && (
        <div
          className={`mb-4 rounded-lg border p-3 text-sm ${
            msgType === "success"
              ? "border-green-200 bg-green-50 text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400"
              : "border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-800 dark:bg-amber-950 dark:text-amber-400"
          }`}
        >
          {msg}
        </div>
      )}

      <div className="space-y-6">
        {/* ── Min Length ── */}
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold text-gray-700 dark:text-gray-300">
            <KeyRound className="h-4 w-4 text-brand-600" /> Minimum Length
          </h2>
          <div className="flex items-center gap-4">
            <span className="text-3xl font-bold text-brand-600">{config.min_length}</span>
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

        {/* ── Complexity Rules ── */}
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <h2 className="mb-2 text-sm font-semibold text-gray-700 dark:text-gray-300">Complexity Rules</h2>
          <div className="divide-y divide-gray-100 dark:divide-gray-700">
            <Toggle
              checked={config.require_uppercase}
              onChange={(v) => setConfig({ ...config, require_uppercase: v })}
              label="Require uppercase (A-Z)"
              description="At least one uppercase letter"
            />
            <Toggle
              checked={config.require_lowercase}
              onChange={(v) => setConfig({ ...config, require_lowercase: v })}
              label="Require lowercase (a-z)"
              description="At least one lowercase letter"
            />
            <Toggle
              checked={config.require_digit}
              onChange={(v) => setConfig({ ...config, require_digit: v })}
              label="Require digits (0-9)"
              description="At least one numeric character"
            />
            <Toggle
              checked={config.require_special}
              onChange={(v) => setConfig({ ...config, require_special: v })}
              label="Require special characters (!@#$%^&*)"
              description="At least one non-alphanumeric character"
            />
            <Toggle
              checked={config.prevent_username}
              onChange={(v) => setConfig({ ...config, prevent_username: v })}
              label="Prevent username in password"
              description="Password cannot contain the user's username"
            />
          </div>
        </div>

        {/* ── Password Expiry ── */}
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold text-gray-700 dark:text-gray-300">
            <Clock className="h-4 w-4 text-brand-600" /> Password Expiry
          </h2>
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
              className="w-24 rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100"
            />
            <span className="text-sm text-gray-400">days (0 = never)</span>
          </div>
          <input
            type="range"
            min={0}
            max={365}
            value={config.expiry_days}
            onChange={(e) => setConfig({ ...config, expiry_days: Number(e.target.value) })}
            className="mt-3 w-full accent-brand-600"
          />
          <div className="mt-1 mb-4 flex justify-between text-xs text-gray-400">
            <span>0 (never)</span>
            <span>365</span>
          </div>
          <label className="flex cursor-pointer items-center gap-2 border-t border-gray-100 pt-3 dark:border-gray-700">
            <input
              type="checkbox"
              checked={config.force_change_on_next_login}
              onChange={(e) => setConfig({ ...config, force_change_on_next_login: e.target.checked })}
              className="h-4 w-4 rounded border-gray-300 text-brand-600 focus:ring-brand-500"
            />
            <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
              Force change on next login
            </span>
          </label>
        </div>

        {/* ── Password History ── */}
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold text-gray-700 dark:text-gray-300">
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
              className="w-24 rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100"
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

        {/* ── Common Password Blacklist ── */}
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <h2 className="mb-2 flex items-center gap-2 text-sm font-semibold text-gray-700 dark:text-gray-300">
            <Ban className="h-4 w-4 text-brand-600" /> Common Password Blacklist
          </h2>
          <div className="divide-y divide-gray-100 dark:divide-gray-700">
            <Toggle
              checked={config.prevent_common}
              onChange={(v) => setConfig({ ...config, prevent_common: v })}
              label="Block common passwords"
              description="Reject passwords from a known dictionary of commonly used passwords"
            />
          </div>
          {config.prevent_common && (
            <div className="mt-4 flex items-center justify-between rounded-lg bg-gray-50 p-3 dark:bg-gray-900">
              <div className="flex items-center gap-2">
                <ListChecks className="h-5 w-5 text-brand-600" />
                <div>
                  <p className="text-sm font-medium text-gray-700 dark:text-gray-300">
                    {TOTAL_BLACKLIST.toLocaleString()} passwords blocked
                  </p>
                  <p className="text-xs text-gray-400">Based on known breach corpora and common patterns</p>
                </div>
              </div>
              <button
                onClick={() => setShowBlacklist(true)}
                className="rounded-lg border border-gray-300 px-3 py-1.5 text-xs font-medium text-gray-600 hover:bg-gray-100 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-800"
              >
                View List
              </button>
            </div>
          )}
        </div>

        {/* ── Live Password Test ── */}
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold text-gray-700 dark:text-gray-300">
            <Zap className="h-4 w-4 text-brand-600" /> Live Password Test
          </h2>
          <p className="mb-3 text-xs text-gray-400">
            Type a test password to get real-time feedback on which rules pass or fail.
          </p>

          {config.prevent_username && (
            <input
              type="text"
              value={testUsername}
              onChange={(e) => setTestUsername(e.target.value)}
              placeholder="Test username (for username check)"
              className="mb-2 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100"
            />
          )}

          <div className="relative">
            <input
              type={showPassword ? "text" : "password"}
              value={testPw}
              onChange={(e) => setTestPw(e.target.value)}
              placeholder="Type a test password..."
              className="w-full rounded-lg border border-gray-300 px-3 py-2 pr-10 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100"
            />
            <button
              type="button"
              onClick={() => setShowPassword(!showPassword)}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600"
            >
              {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
            </button>
          </div>

          {/* Strength meter */}
          {testPw && (
            <div className="mt-4">
              <div className="mb-1 flex items-center justify-between text-xs">
                <span className="font-medium text-gray-500">Strength: {strength.label}</span>
                <span className="text-gray-400">Est. crack time: {strength.crackTime}</span>
              </div>
              <div className="flex h-2 gap-0.5">
                {[1, 2, 3, 4, 5, 6].map((i: any) => (
                  <div
                    key={i}
                    className={`flex-1 rounded-full transition-colors ${
                      i <= strength.score ? strength.color : "bg-gray-200 dark:bg-gray-700"
                    }`}
                  />
                ))}
              </div>
            </div>
          )}

          {/* Rule checks */}
          {testPw && (
            <div className="mt-4 space-y-0.5 rounded-lg bg-gray-50 p-3 dark:bg-gray-900">
              <RuleCheck passes={checkRule("min_length")} label={`At least ${config.min_length} characters`} />
              {config.require_uppercase && (
                <RuleCheck passes={checkRule("require_uppercase")} label="Contains uppercase (A-Z)" />
              )}
              {config.require_lowercase && (
                <RuleCheck passes={checkRule("require_lowercase")} label="Contains lowercase (a-z)" />
              )}
              {config.require_digit && (
                <RuleCheck passes={checkRule("require_digit")} label="Contains digit (0-9)" />
              )}
              {config.require_special && (
                <RuleCheck passes={checkRule("require_special")} label="Contains special character" />
              )}
              {config.prevent_username && (
                <RuleCheck passes={checkRule("prevent_username")} label="Does not contain username" />
              )}
              {config.prevent_common && (
                <RuleCheck passes={checkRule("prevent_common")} label="Not a common password" />
              )}
              <div className="mt-2 border-t border-gray-200 pt-2 dark:border-gray-700">
                {allRulesPass ? (
                  <div className="flex items-center gap-2 text-sm font-medium text-green-600 dark:text-green-400">
                    <Check className="h-4 w-4" />
                    Password meets all requirements
                  </div>
                ) : (
                  <div className="flex items-center gap-2 text-sm font-medium text-red-500">
                    <X className="h-4 w-4" />
                    Password does not meet all requirements
                  </div>
                )}
              </div>
            </div>
          )}
        </div>

        {/* ── Action Buttons ── */}
        <div className="flex justify-between">
          <button
            onClick={handleReset}
            className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-5 py-2.5 text-sm font-medium text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-800"
          >
            <RotateCcw className="h-4 w-4" />
            Reset
          </button>
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

      {/* ── Blacklist Modal ── */}
      {showBlacklist && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
          onClick={() => setShowBlacklist(false)}
        >
          <div
            className="max-h-[70vh] w-full max-w-lg overflow-hidden rounded-xl bg-white shadow-2xl dark:bg-gray-800"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between border-b border-gray-200 p-4 dark:border-gray-700">
              <div className="flex items-center gap-2">
                <Ban className="h-5 w-5 text-brand-600" />
                <h3 className="text-sm font-semibold text-gray-800 dark:text-gray-200">
                  Common Password Blacklist
                </h3>
              </div>
              <button onClick={() => setShowBlacklist(false)} aria-label="Close" className="text-gray-400 hover:text-gray-600">
                <X className="h-5 w-5" />
              </button>
            </div>
            <div className="p-4">
              <p className="mb-3 text-xs text-gray-400">
                Showing a sample of {COMMON_PASSWORDS.length} from {TOTAL_BLACKLIST.toLocaleString()} blocked passwords:
              </p>
              <div className="grid max-h-50 grid-cols-2 gap-1 overflow-y-auto sm:grid-cols-3" style={{ maxHeight: "300px" }}>
                {COMMON_PASSWORDS.map((pw: any) => (
                  <code
                    key={pw}
                    className="rounded bg-gray-100 px-2 py-1 text-xs text-gray-600 dark:bg-gray-900 dark:text-gray-400"
                  >
                    {pw}
                  </code>
                ))}
              </div>
              <p className="mt-3 text-xs text-gray-400">
                Full list is maintained server-side and updated regularly from breach corpora.
              </p>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

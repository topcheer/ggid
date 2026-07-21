"use client";

import { useState, useCallback, useMemo } from "react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";
import {
  KeyRound, Shield, Check, X, Save, Loader2, AlertCircle,
  Eye, EyeOff, Zap, Clock, Hash, Lightbulb,
} from "lucide-react";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "";

// ============ Lightweight zxcvbn-like scoring ============

const COMMON = ["password", "123456", "qwerty", "admin", "welcome", "letmein", "monkey", "dragon", "master", "login"];
const SEQUENCES = ["abcdefghijklmnopqrstuvwxyz", "0123456789", "qwertyuiop", "asdfghjkl"];

function scorePassword(pw: string): {
  score: number; guesses: number; crackTime: string; entropy: number;
  warning: string; suggestions: string[]; patterns: string[];
} {
  if (!pw) return { score: 0, guesses: 0, crackTime: "—", entropy: 0, warning: "", suggestions: [], patterns: [] };

  const len = pw.length;
  let charsetSize = 0;
  if (/[a-z]/.test(pw)) charsetSize += 26;
  if (/[A-Z]/.test(pw)) charsetSize += 26;
  if (/[0-9]/.test(pw)) charsetSize += 10;
  if (/[^a-zA-Z0-9]/.test(pw)) charsetSize += 32;
  const entropy = Math.round(len * Math.log2(Math.max(charsetSize, 1)));

  // Pattern detection
  const patterns: string[] = [];
  const lower = pw.toLowerCase();
  for (const seq of SEQUENCES) {
    for (let i = 0; i <= seq.length - 3; i++) {
      if (lower.includes(seq.substring(i, i + 4))) { patterns.push("sequence"); break; }
    }
  }
  if (/(.)\1{2,}/.test(pw)) patterns.push("repeat");
  for (const c of COMMON) { if (lower.includes(c)) { patterns.push("common"); break; } }

  // Score calculation
  let score = 0;
  if (len >= 8) score++;
  if (len >= 12 && charsetSize >= 36) score++;
  if (len >= 16 && charsetSize >= 62) score++;
  if (patterns.length === 0 && entropy >= 60) score++;
  if (patterns.includes("common") || len < 6) score = 0;
  if (patterns.length > 1 && score > 1) score = 1;
  score = Math.min(4, Math.max(0, score));

  // Guesses estimation
  const guesses = Math.pow(Math.max(charsetSize, 1), len);
  // Crack time at 10 billion guesses/sec
  const seconds = guesses / 1e10;
  let crackTime: string;
  if (seconds < 1) crackTime = "instant";
  else if (seconds < 60) crackTime = `${Math.round(seconds)} seconds`;
  else if (seconds < 3600) crackTime = `${Math.round(seconds / 60)} minutes`;
  else if (seconds < 86400) crackTime = `${Math.round(seconds / 3600)} hours`;
  else if (seconds < 31536000) crackTime = `${Math.round(seconds / 86400)} days`;
  else if (seconds < 31536000 * 1000) crackTime = `${Math.round(seconds / 31536000)} years`;
  else if (seconds < 31536000 * 1e9) crackTime = `${Math.round(seconds / 31536000 / 1000)} thousand years`;
  else crackTime = "centuries";

  // Warning + suggestions
  let warning = "";
  const suggestions: string[] = [];
  if (patterns.includes("common")) { warning = "Contains a common password"; suggestions.push("Avoid common words and patterns"); }
  if (patterns.includes("sequence")) { warning = warning || "Contains a sequence"; suggestions.push("Avoid sequences (abc, 123, qwerty)"); }
  if (patterns.includes("repeat")) { warning = warning || "Contains repeated characters"; suggestions.push("Avoid repeated characters (aaa, 111)"); }
  if (len < 12) suggestions.push("Use 12+ characters");
  if (!/[A-Z]/.test(pw)) suggestions.push("Add uppercase letters");
  if (!/[0-9]/.test(pw)) suggestions.push("Add numbers");
  if (!/[^a-zA-Z0-9]/.test(pw)) suggestions.push("Add symbols (!@#$)");
  if (score >= 4 && suggestions.length === 0) suggestions.push("");

  return { score, guesses, crackTime, entropy, warning, suggestions: suggestions.filter(Boolean), patterns };
}

// ============ Page ============

type TabId = "tester" | "config";

export default function PasswordStrengthPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<TabId>("tester");

  const tabs: { id: TabId; label: string; icon: typeof KeyRound }[] = [
    { id: "tester", label: t("passwordStrength.tabs.tester"), icon: KeyRound },
    { id: "config", label: t("passwordStrength.tabs.config"), icon: Shield },
  ];

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-800 dark:bg-gray-950 p-4 md:p-8">
      <div className="max-w-4xl mx-auto">
        <div className="mb-6">
          <div className="flex items-center gap-3 mb-1">
            <KeyRound className="w-7 h-7 text-blue-600" />
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white dark:text-white">{t("passwordStrength.title")}</h1>
          </div>
          <p className="text-gray-600 dark:text-gray-400 dark:text-gray-400 text-sm">{t("passwordStrength.description")}</p>
        </div>

        <div className="flex gap-1 mb-6 bg-gray-200 dark:bg-gray-800 rounded-lg p-1">
          {tabs.map(({ id, label, icon: Icon }) => (
            <button key={id} onClick={() => setTab(id)}
              className={`flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                tab === id ? "bg-white dark:bg-gray-700 text-blue-600 dark:text-blue-400 shadow-sm" : "text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white"
              }`}>
              <Icon className="w-4 h-4" />{label}
            </button>
          ))}
        </div>

        {tab === "tester" && <TesterTab />}
        {tab === "config" && <ConfigTab />}
      </div>
    </div>
  );
}

// ============ Tester Tab ============

function TesterTab() {
  const t = useTranslations();
  const [pw, setPw] = useState("");
  const [show, setShow] = useState(false);

  const result = useMemo(() => scorePassword(pw), [pw]);

  const scoreColors = ["bg-red-500", "bg-orange-500", "bg-yellow-500", "bg-lime-500", "bg-green-500"];
  const scoreTextColors = ["text-red-600", "text-orange-600", "text-yellow-600", "text-lime-600", "text-green-600"];
  const iconMap = [X, X, AlertCircle, Check, Check];

  return (
    <div className="space-y-4">
      <div className="bg-white dark:bg-gray-800 dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-700 dark:border-gray-800 p-6">
        <h3 className="text-sm font-semibold text-gray-900 dark:text-white dark:text-white mb-3">{t("passwordStrength.tester.title")}</h3>

        {/* Input */}
        <div className="relative">
          <input
            type={show ? "text" : "password"}
            value={pw}
            onChange={(e) => setPw(e.target.value)}
            placeholder={t("passwordStrength.tester.enterPassword")}
            className="w-full px-4 py-3 pr-10 rounded-lg border border-gray-300 dark:border-gray-600 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 dark:bg-gray-800 text-sm text-gray-900 dark:text-white dark:text-white font-mono"
            autoFocus
          />
          <button onClick={() => setShow(!show)} className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 dark:text-gray-400">
            {show ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
          </button>
        </div>

        {/* Score Bar */}
        {pw && (
          <div className="mt-4">
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs text-gray-500">{t("passwordStrength.tester.score")}</span>
              <span className={`text-sm font-bold ${scoreTextColors[result.score]}`}>
                {t(`passwordStrength.tester.score${result.score}`)}
              </span>
            </div>
            <div className="flex gap-1">
              {[0, 1, 2, 3, 4].map((i: any) => (
                <div key={i} className={`h-2 flex-1 rounded-full transition-all ${i <= result.score ? scoreColors[result.score] : "bg-gray-200 dark:bg-gray-700"}`} />
              ))}
            </div>
          </div>
        )}

        {/* Stats */}
        {pw && (
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mt-4">
            <StatBox icon={Clock} label={t("passwordStrength.tester.crackTime")} value={result.crackTime} />
            <StatBox icon={Hash} label={t("passwordStrength.tester.entropy")} value={`${result.entropy} bits`} />
            <StatBox icon={Zap} label={t("passwordStrength.tester.guesses")} value={result.guesses > 1e6 ? `${(result.guesses / 1e6).toFixed(1)}M` : result.guesses > 1e3 ? `${(result.guesses / 1e3).toFixed(0)}K` : String(result.guesses)} />
            <StatBox icon={KeyRound} label={t("passwordStrength.tester.score")} value={`${result.score}/4`} />
          </div>
        )}

        {/* Warning */}
        {pw && result.warning && (
          <div className="mt-4 flex items-center gap-2 px-4 py-2 rounded-lg bg-orange-50 dark:bg-orange-950/30 text-orange-700 dark:text-orange-300 text-sm">
            <AlertCircle className="w-4 h-4" />
            <span className="font-medium">{t("passwordStrength.tester.warning")}:</span>
            <span>{result.warning}</span>
          </div>
        )}

        {/* Patterns */}
        {pw && result.patterns.length > 0 && (
          <div className="mt-3 flex items-center gap-2 flex-wrap">
            <span className="text-xs text-gray-500">{t("passwordStrength.tester.patternDetected")}:</span>
            {result.patterns.map((p: any, i: number) => (
              <span key={i} className="px-2 py-0.5 text-xs bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300 rounded">{p}</span>
            ))}
          </div>
        )}

        {/* Suggestions */}
        {pw && (
          <div className="mt-4">
            <div className="flex items-center gap-2 mb-2">
              <Lightbulb className="w-4 h-4 text-yellow-500" />
              <span className="text-xs font-medium text-gray-600 dark:text-gray-400 dark:text-gray-400">{t("passwordStrength.tester.suggestions")}</span>
            </div>
            {result.suggestions.length === 0 ? (
              <div className="flex items-center gap-1 text-sm text-green-600">
                <Check className="w-4 h-4" />{t("passwordStrength.tester.noSuggestions")}
              </div>
            ) : (
              <ul className="space-y-1">
                {result.suggestions.map((s: any, i: number) => (
                  <li key={i} className="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300 dark:text-gray-300">
                    <X className="w-3 h-3 text-red-400" />{s}
                  </li>
                ))}
              </ul>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

// ============ Config Tab ============

function ConfigTab() {
  const t = useTranslations();
  const [minScore, setMinScore] = useState(3);
  const [dictCheck, setDictCheck] = useState(true);
  const [breachCheck, setBreachCheck] = useState(true);
  const [blocklist, setBlocklist] = useState("company\npassword\nadmin\nwelcome\nggid");
  const [saving, setSaving] = useState(false);
  const [msg, setMsg] = useState<string | null>(null);

  const save = async () => {
    setSaving(true);
    try {
      await fetch(`${API_BASE}/api/v1/auth/password-policy/check`, {
        method: "PUT", headers: { "Content-Type": "application/json", ...authHeader() },
        body: JSON.stringify({ min_score: minScore, dictionary_check: dictCheck, breach_check: breachCheck, custom_blocklist: blocklist.split("\n").filter(Boolean) }),
      });
    } catch { /* ok */ }
    setSaving(false);
    setMsg(t("passwordStrength.config.saved"));
    setTimeout(() => setMsg(null), 3000);
  };

  return (
    <div className="bg-white dark:bg-gray-800 dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-700 dark:border-gray-800 p-6 space-y-5">
      <h3 className="text-sm font-semibold text-gray-900 dark:text-white dark:text-white">{t("passwordStrength.config.title")}</h3>

      {/* Min Score */}
      <div>
        <label className="block text-sm font-medium text-gray-900 dark:text-white dark:text-white mb-1">{t("passwordStrength.config.minScore")}</label>
        <p className="text-xs text-gray-500 dark:text-gray-400 mb-3">{t("passwordStrength.config.minScoreDesc")}</p>
        <div className="flex gap-2">
          {[0, 1, 2, 3, 4].map((s: any) => {
            const colors = ["bg-red-500", "bg-orange-500", "bg-yellow-500", "bg-lime-500", "bg-green-500"];
            return (
              <button key={s} onClick={() => setMinScore(s)}
                className={`flex-1 py-2 rounded-lg border-2 text-sm font-medium transition-all ${
                  minScore === s ? `border-blue-500 text-white ${colors[s]}` : "border-gray-200 dark:border-gray-700 text-gray-500"
                }`}>
                {s} — {t(`passwordStrength.tester.score${s}`)}
              </button>
            );
          })}
        </div>
      </div>

      {/* Toggles */}
      <div className="space-y-3">
        <ToggleRow label={t("passwordStrength.config.dictionaryCheck")} desc={t("passwordStrength.config.dictionaryCheckDesc")} checked={dictCheck} onChange={() => setDictCheck(!dictCheck)} />
        <ToggleRow label={t("passwordStrength.config.breachCheck")} desc={t("passwordStrength.config.breachCheckDesc")} checked={breachCheck} onChange={() => setBreachCheck(!breachCheck)} />
      </div>

      {/* Custom Blocklist */}
      <div>
        <label className="block text-sm font-medium text-gray-900 dark:text-white dark:text-white mb-1">{t("passwordStrength.config.customBlocklist")}</label>
        <p className="text-xs text-gray-500 dark:text-gray-400 mb-2">{t("passwordStrength.config.customBlocklistDesc")}</p>
        <textarea value={blocklist} onChange={(e) => setBlocklist(e.target.value)} rows={6}
          placeholder={t("passwordStrength.config.customBlocklistPlaceholder")}
          className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 dark:bg-gray-800 text-sm font-mono text-gray-900 dark:text-white dark:text-white" />
        <p className="text-xs text-gray-400 mt-1">{blocklist.split("\n").filter(Boolean).length} words in blocklist</p>
      </div>

      {msg && (
        <div className="flex items-center gap-2 px-4 py-2 rounded-lg bg-green-50 text-green-700 dark:bg-green-950 dark:text-green-300 text-sm">
          <Check className="w-4 h-4" />{msg}
        </div>
      )}

      <button onClick={save} disabled={saving}
        className="flex items-center gap-2 px-6 py-2.5 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg font-medium text-sm">
        {saving ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
        {t("passwordStrength.config.save")}
      </button>
    </div>
  );
}

// ============ Shared ============

function StatBox({ icon: Icon, label, value }: { icon: typeof Clock; label: string; value: string }) {
  return (
    <div className="p-3 rounded-lg bg-gray-50 dark:bg-gray-800 dark:bg-gray-800/50">
      <div className="flex items-center gap-1 mb-1">
        <Icon className="w-3.5 h-3.5 text-gray-400" />
        <span className="text-xs text-gray-500">{label}</span>
      </div>
      <div className="text-sm font-bold text-gray-900 dark:text-white dark:text-white truncate">{value}</div>
    </div>
  );
}

function ToggleRow({ label, desc, checked, onChange }: { label: string; desc: string; checked: boolean; onChange: () => void }) {
  return (
    <label className="flex items-center justify-between cursor-pointer py-1">
      <div>
        <span className="text-sm text-gray-700 dark:text-gray-300 dark:text-gray-300">{label}</span>
        <p className="text-xs text-gray-400">{desc}</p>
      </div>
      <button onClick={onChange}
        className={`relative w-10 h-6 rounded-full transition-colors flex-shrink-0 ml-3 ${checked ? "bg-blue-600" : "bg-gray-300 dark:bg-gray-600"}`}>
        <span className={`absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full transition-transform ${checked ? "translate-x-4" : ""}`} />
      </button>
    </label>
  );
}

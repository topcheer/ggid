"use client";

import React, { useState } from "react";
import { useApi } from "@/lib/api";
import {
  ShieldCheck, Loader2, AlertCircle, X, TrendingUp, KeyRound, Users, Lightbulb, ArrowRight,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface Recommendation {
  id: string;
  category: string;
  title: string;
  description: string;
  severity: "low" | "medium" | "high" | "critical";
  impact: number;
  action_url: string;
}

interface Posture {
  score: number;
  grade: string;
  mfa_adoption_pct: number;
  weak_password_count: number;
  total_users: number;
  active_sessions: number;
  expired_sessions: number;
  failed_logins_24h: number;
  recommendations: Recommendation[];
  last_calculated: string;
}

const sevColors: Record<string, string> = {
  critical: "text-red-600 bg-red-100 dark:bg-red-900/30 dark:text-red-400",
  high: "text-orange-600 bg-orange-100 dark:bg-orange-900/30 dark:text-orange-400",
  medium: "text-yellow-600 bg-yellow-100 dark:bg-yellow-900/30 dark:text-yellow-400",
  low: "text-blue-600 bg-blue-100 dark:bg-blue-900/30 dark:text-blue-400",
};

function PostureGauge({ score, grade }: { score: number; grade: string }) {
  const clamped = Math.min(100, Math.max(0, score));
  const color = clamped >= 80 ? "#16a34a" : clamped >= 60 ? "#eab308" : clamped >= 40 ? "#f97316" : "#dc2626";
  const gradeColor = grade === "A" ? "text-green-600" : grade === "B" ? "text-lime-600" : grade === "C" ? "text-yellow-600" : grade === "D" ? "text-orange-600" : "text-red-600";
  return (
    <div className="relative flex flex-col items-center">
      <svg width="160" height="90" viewBox="0 0 160 90">
        <path d="M 10 80 A 70 70 0 0 1 150 80" fill="none" stroke="#e5e7eb" strokeWidth="10" strokeLinecap="round" />
        <path d="M 10 80 A 70 70 0 0 1 150 80" fill="none" stroke={color} strokeWidth="10" strokeLinecap="round" strokeDasharray={`${(clamped / 100) * 220} 220`} />
      </svg>
      <div className="-mt-8 flex flex-col items-center">
        <div className="text-3xl font-bold" style={{ color }}>{clamped}</div>
        <div className={`text-2xl font-bold ${gradeColor}`}>{grade}</div>
      </div>
    </div>
  );
}

export default function SecurityPosturePage() {
  const t = useTranslations();
  const { apiFetch } = useApi();
  const [posture, setPosture] = useState<Posture | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useState(() => {
    (async () => {
      try { setPosture(await apiFetch<Posture>("/api/v1/audit/security-posture").catch(() => null)); }
      catch { setError("Failed to load posture"); }
      finally { setLoading(false); }
    })();
  });

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><ShieldCheck className="h-6 w-6 text-emerald-600" /> {t("securityPosture.title")}</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Overall security score based on MFA adoption, password strength, and session hygiene.</p>
      </div>

      {error && <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-emerald-600" /></div>
      : posture ? (
        <>
          <div className="grid grid-cols-3 gap-6">
            {/* Score gauge */}
            <div className={`${cardCls} flex flex-col items-center justify-center`}><PostureGauge score={posture.score} grade={posture.grade} /><div className="mt-2 text-xs uppercase text-gray-400">Posture Score</div></div>

            {/* MFA + weak passwords */}
            <div className={cardCls}><div className="mb-4 flex items-center gap-2"><Users className="h-4 w-4 text-indigo-500" /><span className="text-xs font-semibold uppercase text-gray-400">MFA Adoption</span></div>
              <div className="flex items-center gap-3"><div className="text-3xl font-bold text-indigo-600">{posture.mfa_adoption_pct}%</div><div className="flex-1"><div className="h-3 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700"><div className="h-full rounded-full bg-indigo-500" style={{ width: `${posture.mfa_adoption_pct}%` }} /></div></div></div>
              <div className="mt-3 flex items-center gap-2 text-sm"><KeyRound className="h-4 w-4 text-orange-500" /><span className="text-gray-500">Weak passwords:</span><span className="font-bold text-orange-600">{posture.weak_password_count}</span><span className="text-gray-400">/ {posture.total_users}</span></div>
            </div>

            {/* Session + login stats */}
            <div className={cardCls}><div className="mb-4 flex items-center gap-2"><TrendingUp className="h-4 w-4 text-blue-500" /><span className="text-xs font-semibold uppercase text-gray-400">Session Stats</span></div>
              <div className="space-y-2 text-sm">
                <div className="flex justify-between"><span className="text-gray-500">Active sessions</span><span className="font-medium text-gray-900 dark:text-white">{posture.active_sessions}</span></div>
                <div className="flex justify-between"><span className="text-gray-500">Expired (not cleared)</span><span className="font-medium text-orange-600">{posture.expired_sessions}</span></div>
                <div className="flex justify-between"><span className="text-gray-500">Failed logins (24h)</span><span className="font-medium text-red-600">{posture.failed_logins_24h}</span></div>
              </div>
            </div>
          </div>

          {/* Recommendations */}
          <div>
            <h2 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-500"><Lightbulb className="h-4 w-4" /> Recommendations</h2>
            {posture.recommendations.length === 0 ? (
              <div className={cardCls}><div className="py-8 text-center"><ShieldCheck className="mx-auto h-10 w-10 text-green-300" /><p className="mt-3 text-sm text-gray-400">No recommendations. Your posture is excellent.</p></div></div>
            ) : (
              <div className="space-y-2">
                {posture.recommendations.map((r) => (
                  <div key={r.id} className={`${cardCls} flex items-center justify-between py-3`}>
                    <div className="flex items-center gap-3">
                      <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${sevColors[r.severity] || ""}`}>{r.severity}</span>
                      <div><div className="font-medium text-gray-900 dark:text-white">{r.title}</div><div className="text-xs text-gray-400">{r.description}</div></div>
                    </div>
                    <div className="flex items-center gap-3"><span className="text-xs text-gray-400">+{r.impact} pts</span><ArrowRight className="h-4 w-4 text-gray-300" /></div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </>
      ) : <div className={cardCls}><div className="py-12 text-center"><ShieldCheck className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No posture data available.</p></div></div>}
    </div>
  );
}

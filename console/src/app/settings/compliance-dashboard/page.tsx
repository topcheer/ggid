"use client";

import React, { useState } from "react";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";
import {
  PieChart, Loader2, AlertCircle, X, CheckCircle, Clock, XCircle, ChevronRight,
} from "lucide-react";

interface FrameworkSummary {
  framework: string;
  coverage_pct: number;
  total_controls: number;
  compliant: number;
  partial: number;
  missing: number;
  gap_count: number;
  last_assessed: string;
}

function Donut({ pct, color }: { pct: number; color: string }) {
  const r = 32; const c = 2 * Math.PI * r; const offset = c - (pct / 100) * c;
  return (
    <svg width="80" height="80" viewBox="0 0 80 80">
      <circle cx="40" cy="40" r={r} fill="none" stroke="#e5e7eb" strokeWidth="8" />
      <circle cx="40" cy="40" r={r} fill="none" stroke={color} strokeWidth="8" strokeDasharray={c} strokeDashoffset={offset} strokeLinecap="round" transform="rotate(-90 40 40)" />
      <text x="40" y="44" textAnchor="middle" className="fill-gray-700 dark:fill-gray-200 text-sm font-bold">{pct}%</text>
    </svg>
  );
}

export default function ComplianceDashboardPage() {
  const t = useTranslations();
  const { apiFetch } = useApi();
  const [frameworks, setFrameworks] = useState<FrameworkSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selected, setSelected] = useState<string | null>(null);

  useState(() => {
    (async () => {
      try { setFrameworks(await apiFetch<FrameworkSummary[]>("/api/v1/audit/compliance-dashboard").catch(() => [])); }
      catch { setError(t("complianceDashboard.failedLoad")); }
      finally { setLoading(false); }
    })();
  });

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const totalGaps = frameworks.reduce((s, f) => s + f.gap_count, 0);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><PieChart className="h-6 w-6 text-teal-600" /> {t("complianceDashboard.title")}</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("complianceDashboard.subtitle")}</p>
      </div>

      {error && <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {totalGaps > 0 && <div className="flex items-center gap-3 rounded-xl border border-orange-200 bg-orange-50 px-4 py-3 dark:border-orange-800 dark:bg-orange-900/20"><AlertCircle className="h-5 w-5 text-orange-600 shrink-0" /><span className="text-sm text-orange-700 dark:text-orange-400">{totalGaps} compliance gap{totalGaps > 1 ? "s" : ""} across all frameworks.</span></div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-teal-600" /></div>
      : frameworks.length === 0 ? (
        <div className={cardCls}><div className="py-12 text-center"><PieChart className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">{t("complianceDashboard.noData")}</p></div></div>
      ) : (
        <div className="grid grid-cols-3 gap-4">
          {frameworks.map((f) => {
            const color = f.coverage_pct >= 80 ? "#16a34a" : f.coverage_pct >= 60 ? "#eab308" : "#dc2626";
            const isExpanded = selected === f.framework;
            return (
              <div key={f.framework} className={`${cardCls} ${isExpanded ? "col-span-3" : ""}`}>
                <div className="flex items-center gap-4">
                  <Donut pct={f.coverage_pct} color={color} />
                  <div className="flex-1">
                    <h3 className="text-lg font-bold text-gray-900 dark:text-white">{f.framework}</h3>
                    <div className="mt-1 space-y-0.5 text-sm">
                      <div className="flex items-center gap-2"><CheckCircle className="h-3 w-3 text-green-500" /><span className="text-gray-500">{f.compliant}/{f.total_controls} compliant</span></div>
                      {f.partial > 0 && <div className="flex items-center gap-2"><Clock className="h-3 w-3 text-yellow-500" /><span className="text-gray-500">{f.partial} partial</span></div>}
                      {f.missing > 0 && <div className="flex items-center gap-2"><XCircle className="h-3 w-3 text-red-500" /><span className="text-gray-500">{f.missing} missing</span></div>}
                      {f.gap_count > 0 && <div className="text-xs text-orange-600">{f.gap_count} gaps</div>}
                    </div>
                    <div className="mt-1 text-xs text-gray-400">Last assessed: {f.last_assessed ? new Date(f.last_assessed).toLocaleDateString() : "never"}</div>
                  </div>
                  <button onClick={() => setSelected(isExpanded ? null : f.framework)} className="self-start text-xs text-teal-600 hover:underline">{isExpanded ? "Collapse" : "Details"}</button>
                </div>
                {isExpanded && (
                  <div className="mt-4 border-t border-gray-200 pt-4 dark:border-gray-700">
                    <div className="grid grid-cols-3 gap-4 text-center">
                      <div><div className="text-xs font-semibold uppercase text-green-500">Compliant</div><p className="text-2xl font-bold text-green-600">{f.compliant}</p></div>
                      <div><div className="text-xs font-semibold uppercase text-yellow-500">Partial</div><p className="text-2xl font-bold text-yellow-600">{f.partial}</p></div>
                      <div><div className="text-xs font-semibold uppercase text-red-500">Missing</div><p className="text-2xl font-bold text-red-600">{f.missing}</p></div>
                    </div>
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

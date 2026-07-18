"use client";

import { useState, useEffect, useCallback } from "react";
import { ShieldCheck, AlertCircle } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface FrameworkInfo {
  framework: string;
  total_controls: number;
  covered: number;
  gaps: string[];
  coverage_pct: number;
}

interface CoverageData {
  frameworks: FrameworkInfo[];
}

const frameworks = ["SOC 2", "HIPAA", "ISO 27001", "GDPR"];

export default function FrameworkCoveragePage() {
  const t = useTranslations();

  const [activeTab, setActiveTab] = useState("SOC 2");
  const [data, setData] = useState<CoverageData | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/audit/framework-coverage", { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const current = data?.frameworks.find((f: any) => f.framework === activeTab);
  const barColor = (pct: number) => pct >= 80 ? "#10b981" : pct >= 50 ? "#f59e0b" : "#ef4444";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><ShieldCheck className="w-6 h-6 text-green-500" /> {t("frameworkCoverage.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Compliance control coverage across regulatory frameworks.</p>
      </div>

      <div className="flex items-center gap-1 border-b dark:border-gray-800">
        {frameworks.map((f: any) => (
          <button key={f} onClick={() => setActiveTab(f)} className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${activeTab === f ? "border-blue-600 text-blue-600" : "border-transparent text-gray-500 hover:text-gray-700"}`}>{f}</button>
        ))}
      </div>

      {current && (
        <>
          <div className="grid grid-cols-3 gap-4">
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">{t("backend3.frameworkCoverage.totalControls")}</span><p className="text-2xl font-bold mt-1">{current.total_controls}</p></div>
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">{t("backend3.frameworkCoverage.covered")}</span><p className="text-2xl font-bold text-green-600 mt-1">{current.covered}</p></div>
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">{t("backend3.frameworkCoverage.gaps")}</span><p className="text-2xl font-bold text-red-600 mt-1">{current.gaps.length}</p></div>
          </div>

          <div className="rounded-lg border dark:border-gray-800 p-4">
            <div className="flex items-center justify-between mb-2"><span className="text-sm font-semibold">Coverage: {current.coverage_pct.toFixed(1)}%</span></div>
            <div className="w-full bg-gray-100 dark:bg-gray-800 rounded-full h-4 overflow-hidden"><div className="h-full rounded-full transition-all" style={{ width: `${current.coverage_pct}%`, background: barColor(current.coverage_pct) }} /></div>
          </div>

          {current.gaps.length > 0 && (
            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="text-sm font-semibold flex items-center gap-2 mb-3"><AlertCircle className="w-4 h-4 text-red-500" /> Uncovered Controls ({current.gaps.length})</h3>
              <div className="space-y-1">{current.gaps.map((g: any, i: number) => (
                <div key={i} className="flex items-center gap-2 text-sm"><span className="w-2 h-2 rounded-full bg-red-500" /><span className="font-mono text-xs">{g}</span></div>
              ))}</div>
            </div>
          )}

          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            {data!.frameworks.map((f: any) => (
              <div key={f.framework} className="rounded-lg border dark:border-gray-800 p-3 cursor-pointer" onClick={() => setActiveTab(f.framework)}>
                <span className="text-xs font-medium text-gray-500">{f.framework}</span>
                <p className="text-lg font-bold mt-1" style={{ color: barColor(f.coverage_pct) }}>{f.coverage_pct.toFixed(0)}%</p>
                <div className="w-full bg-gray-100 dark:bg-gray-800 rounded-full h-1.5 overflow-hidden mt-1"><div className="h-full rounded-full" style={{ width: `${f.coverage_pct}%`, background: barColor(f.coverage_pct) }} /></div>
              </div>
            ))}
          </div>
        </>
      )}
      {!current && !loading && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
    </div>
  );
}

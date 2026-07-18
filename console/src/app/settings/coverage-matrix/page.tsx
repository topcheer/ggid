"use client";

import { useState, useEffect, useCallback } from "react";
import { Grid3x3, AlertTriangle, Layers } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface MatrixData {
  subjects: string[];
  resources: string[];
  cells: { subject: string; resource: string; coverage_pct: number; policies: number }[];
  uncovered: { subject: string; resource: string }[];
  redundant: { subject: string; resource: string; count: number }[];
  gaps_count: number;
}

function cellColor(pct: number) {
  const t = useTranslations();

  if (pct >= 100) return "bg-green-500";
  if (pct >= 50) return "bg-yellow-500";
  if (pct > 0) return "bg-orange-500";
  return "bg-red-200 dark:bg-red-900/30";
}

export default function CoverageMatrixPage() {
  const t = useTranslations();
  const [data, setData] = useState<MatrixData | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/policy/coverage-matrix", { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Grid3x3 className="w-6 h-6 text-indigo-500" /> {t("coverageMatrix.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Policy coverage across subject-resource combinations with gap detection.</p>
      </div>

      {data && (
        <>
          <div className="grid grid-cols-3 gap-4">
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">{t("backend3.coverageMatrix.subjects")}</span><p className="text-xl font-bold mt-1">{data.subjects.length}</p></div>
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">{t("backend3.coverageMatrix.resources")}</span><p className="text-xl font-bold mt-1">{data.resources.length}</p></div>
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">{t("backend3.coverageMatrix.gaps")}</span><p className="text-xl font-bold text-red-600 mt-1">{data.gaps_count}</p></div>
          </div>

          {data.subjects.length > 0 && data.resources.length > 0 && (
            <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
              <table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-3 py-2 text-left font-medium sticky">{t("backend3.coverageMatrix.subject")}</th>{data.resources.map((r) => <th key={r} className="px-2 py-2 text-center text-xs font-mono">{r}</th>)}</tr></thead>
                <tbody className="divide-y dark:divide-gray-800">{data.subjects.map((subj) => (
                  <tr key={subj} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-3 py-2 font-mono text-xs">{subj}</td>{data.resources.map((res) => { const cell = data.cells.find((c) => c.subject === subj && c.resource === res); const pct = cell?.coverage_pct ?? 0; return (<td key={res} className="px-2 py-2 text-center"><div className={`inline-block w-10 h-7 rounded ${cellColor(pct)} flex items-center justify-center text-xs font-bold text-white`}>{pct > 0 ? pct + "%" : "-"}</div></td>); })}</tr>
                ))}</tbody>
              </table>
            </div>
          )}

          {data.uncovered.length > 0 && (
            <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold flex items-center gap-2 mb-3"><AlertTriangle className="w-4 h-4 text-red-500" /> Uncovered Combinations</h3><div className="flex flex-wrap gap-2">{data.uncovered.map((u: any, i: number) => <span key={i} className="px-2 py-1 rounded text-xs bg-red-50 dark:bg-red-900/20 text-red-600 font-mono">{u.subject} x {u.resource}</span>)}</div></div>
          )}

          {data.redundant.length > 0 && (
            <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold flex items-center gap-2 mb-3"><Layers className="w-4 h-4 text-yellow-500" /> Redundant Policies</h3><div className="space-y-1">{data.redundant.map((r: any, i: number) => <div key={i} className="flex items-center gap-2 text-sm"><span className="font-mono text-xs">{r.subject} x {r.resource}</span><span className="px-2 py-0.5 rounded text-xs bg-yellow-100 dark:bg-yellow-900/30 dark:text-yellow-400">{r.count} policies</span></div>)}</div></div>
          )}
        </>
      )}
      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
    </div>
  );
}

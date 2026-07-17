"use client";

import React, { useEffect, useState } from "react";
import { useApi } from "@/lib/api";
import {
  Grid3x3, Loader2, AlertCircle, X, XCircle, Info,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface SoDRule {
  id: string;
  role_a: string;
  role_b: string;
  reason: string;
  created_at: string;
}

export default function SoDMatrixPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [roles, setRoles] = useState<string[]>([]);
  const [matrix, setMatrix] = useState<boolean[][]>([]);
  const [rules, setRules] = useState<SoDRule[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [toggling, setToggling] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      try {
        const data = await apiFetch<{ roles: string[]; matrix: boolean[][]; rules: SoDRule[] }>("/api/v1/policy/sod-matrix").catch(() => ({ roles: [], matrix: [], rules: [] }));
        setRoles(data.roles); setMatrix(data.matrix); setRules(data.rules);
      } catch { setError("Failed to load SoD matrix"); }
      finally { setLoading(false); }
    })();
  }, []);

  const handleToggle = async (i: number, j: number) => {
    if (i === j) return;
    const key = `${i}-${j}`;
    setToggling(key);
    try { await apiFetch("/api/v1/policy/sod-matrix/toggle", { method: "POST", body: JSON.stringify({ role_a: roles[i], role_b: roles[j] }) }); setMatrix((prev) => { const n = prev.map((r) => [...r]); n[i][j] = !n[i][j]; n[j][i] = !n[j][i]; return n; }); }
    catch { setError("Toggle failed"); }
    finally { setToggling(null); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Grid3x3 className="h-6 w-6 text-red-600" /> {t("securitySodMatrix.title")}</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Role exclusion grid preventing conflicts of interest.</p>
      </div>

      {/* Legend */}
      <div className="flex items-center gap-6 text-sm">
        <div className="flex items-center gap-2"><XCircle className="h-4 w-4 text-red-500" /><span className="text-gray-600 dark:text-gray-300">Exclusive (cannot co-assign)</span></div>
        <div className="flex items-center gap-2"><div className="h-4 w-4 rounded border border-gray-300 dark:border-gray-600" /><span className="text-gray-600 dark:text-gray-300">Compatible</span></div>
        <div className="flex items-center gap-2"><Info className="h-4 w-4 text-gray-400" /><span className="text-gray-500">{rules.length} rule{rules.length !== 1 ? "s" : ""}</span></div>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-red-600" /></div>
      : roles.length === 0 ? (
        <div className={cardCls}><div className="py-12 text-center"><Grid3x3 className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No roles available for matrix.</p></div></div>
      ) : (
        <div className="overflow-x-auto rounded-xl border border-gray-200 dark:border-gray-700">
          <table className="w-full text-sm">
            <thead><tr>
              <th scope="col" className="sticky left-0 z-10 border-b border-r border-gray-200 bg-gray-50 px-3 py-2 text-xs font-semibold text-gray-500 dark:border-gray-700 dark:bg-gray-800">Role</th>
              {roles.map((r) => <th scope="col" key={r} className="border-b border-r border-gray-200 bg-gray-50 px-2 py-2 text-xs font-semibold text-gray-500 dark:border-gray-700 dark:bg-gray-800"><div className="max-w-[80px] truncate" title={r}>{r}</div></th>)}
            </tr></thead>
            <tbody>
              {roles.map((roleA, i) => (
                <tr key={roleA}>
                  <td className="sticky left-0 z-10 border-b border-r border-gray-200 bg-gray-50 px-3 py-2 text-xs font-semibold text-gray-600 dark:border-gray-700 dark:bg-gray-800" title={roleA}><div className="max-w-[100px] truncate">{roleA}</div></td>
                  {roles.map((_, j) => (
                    <td key={j} className="border-b border-r border-gray-200 text-center dark:border-gray-700">
                      {i === j ? <div className="flex h-7 items-center justify-center"><div className="h-1 w-1 rounded-full bg-gray-300" /></div>
                      : <button onClick={() => handleToggle(i, j)} disabled={toggling === `${i}-${j}`} className="flex h-7 w-full items-center justify-center hover:bg-gray-50 dark:hover:bg-gray-800">
                        {matrix[i] && matrix[i][j] ? <XCircle className="h-4 w-4 text-red-500" /> : <div className="h-4 w-4 rounded border border-gray-200 dark:border-gray-600" />}
                      </button>}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Rules list */}
      {rules.length > 0 && (
        <div>
          <h2 className="mb-3 text-sm font-semibold uppercase text-gray-500">Active Rules</h2>
          <div className="flex flex-wrap gap-2">
            {rules.map((r) => (
              <div key={r.id} className="flex items-center gap-2 rounded-lg border border-gray-200 px-3 py-1.5 text-xs dark:border-gray-700">
                <span className="font-medium text-gray-700 dark:text-gray-300">{r.role_a}</span>
                <XCircle className="h-3 w-3 text-red-400" />
                <span className="font-medium text-gray-700 dark:text-gray-300">{r.role_b}</span>
                {r.reason && <span className="text-gray-400">— {r.reason}</span>}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

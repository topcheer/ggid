"use client";
import { useTranslations } from "@/lib/i18n";
import { useState, useEffect, useCallback } from "react";
import { Grid3x3, Shield } from "lucide-react";

interface ScopeMatrix { scopes: string[]; consent_levels: ("none" | "implicit" | "explicit" | "admin")[]; assignments: Record<string, "none" | "implicit" | "explicit" | "admin">; risk_levels: Record<string, string>; }

const levelColors: Record<string, string> = { none: "bg-gray-300 dark:bg-gray-700", implicit: "bg-green-500", explicit: "bg-yellow-500", admin: "bg-red-500" };
const riskColors: Record<string, string> = { low: "text-green-600", medium: "text-yellow-600", high: "text-orange-600", critical: "text-red-600" };

export default function OAuthScopeConsentMatrixPage() {
  const t = useTranslations();
  const [data, setData] = useState<ScopeMatrix | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/oauth/scope-consent-matrix", { headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  if (!data) return <p className="text-sm text-gray-500 text-center py-8">Loading...</p>;

  const cycleLevel = (scope: string) => {
    const current = data.assignments[scope] || "none";
    const idx = data.consent_levels.indexOf(current);
    const next = data.consent_levels[(idx + 1) % data.consent_levels.length];
    setData({ ...data, assignments: { ...data.assignments, [scope]: next } });
  };

  const summary = { none: 0, implicit: 0, explicit: 0, admin: 0 };
  Object.values(data.assignments).forEach((v) => { summary[v]++; });

  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><Grid3x3 className="w-6 h-6 text-blue-500" />{t("oauthScopeConsentMatrix.title")}</h1><p className="text-sm text-gray-500 mt-1">Configure required consent levels per OAuth scope.</p></div>

      <div className="grid grid-cols-4 gap-4">{Object.entries(summary).map(([level, count]) => (<div key={level} className="rounded-lg border p-4 dark:border-gray-800"><div className="flex items-center gap-2"><span className={"w-3 h-3 rounded " + levelColors[level]} /><span className="text-sm capitalize text-gray-500">{level}</span></div><p className="text-xl font-bold mt-1">{count}</p></div>))}</div>

      <div className="overflow-x-auto rounded-lg border dark:border-gray-800"><table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Scope</th><th className="px-4 py-3 text-left font-medium">Risk</th><th className="px-4 py-3 text-left font-medium">Required Consent</th></tr></thead><tbody className="divide-y dark:divide-gray-800">{data.scopes.map((scope) => { const level = data.assignments[scope] || "none"; const risk = data.risk_levels[scope] || "low"; return (<tr key={scope} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3 font-mono text-xs font-medium">{scope}</td><td className="px-4 py-3"><span className={"flex items-center gap-1 text-xs font-medium " + riskColors[risk]}><Shield className="w-3 h-3" /> {risk}</span></td><td className="px-4 py-3"><button onClick={() => cycleLevel(scope)} className={"px-3 py-1 rounded text-xs font-medium text-white " + levelColors[level]}>{level}</button></td></tr>); })}</tbody></table></div>

      <div className="flex items-center gap-4 text-xs">{Object.entries(levelColors).map(([level, color]) => (<div key={level} className="flex items-center gap-1"><span className={"w-4 h-4 rounded " + color} /> <span className="capitalize">{level}</span></div>))}</div>
    </div>
  );
}

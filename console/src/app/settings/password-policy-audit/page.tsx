"use client";

import { useState, useEffect, useCallback } from "react";
import { ShieldCheck, AlertTriangle, Filter } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface Violation {
  id: string;
  user_id: string;
  username: string;
  rule_type: string;
  detail: string;
  detected_at: string;
}

const ruleTypes = ["min_length", "complexity", "history", "expiry", "reused", "common", "breached"];

export default function PasswordPolicyAuditPage() {
  const t = useTranslations();

  const [violations, setViolations] = useState<Violation[]>([]);
  const [loading, setLoading] = useState(false);
  const [filterRule, setFilterRule] = useState("");

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/auth/password-policy-audit", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const data = await res.json(); setViolations(data.violations || data || []); }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const filtered = filterRule ? violations.filter((v) => v.rule_type === filterRule) : violations;
  const totalUsers = new Set(violations.map((v) => v.user_id)).size;
  const complianceRate = violations.length === 0 ? 100 : Math.max(0, Math.round(100 - (violations.length / Math.max(totalUsers * 3, 1)) * 100));
  const gaugeColor = complianceRate >= 90 ? "#10b981" : complianceRate >= 70 ? "#f59e0b" : "#ef4444";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><ShieldCheck className="w-6 h-6 text-green-500" /> {t("passwordPolicyAudit.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Audit password policy compliance across all users.</p>
      </div>

      <div className="flex items-center gap-6">
        <div className="relative w-28 h-28">
          <svg viewBox="0 0 64 64" className="w-full h-full"><circle cx={32} cy={32} r={28} fill="none" stroke="currentColor" strokeWidth={6} className="text-gray-200 dark:text-gray-800" /><circle cx={32} cy={32} r={28} fill="none" stroke={gaugeColor} strokeWidth={6} strokeDasharray={`${(complianceRate / 100) * 176} 176`} strokeLinecap="round" transform="rotate(-90 32 32)" /></svg>
          <div className="absolute inset-0 flex flex-col items-center justify-center"><span className="text-2xl font-bold" style={{ color: gaugeColor }}>{complianceRate}%</span><span className="text-[10px] text-gray-400">compliant</span></div>
        </div>
        <div className="space-y-1">
          <div className="flex items-center gap-2"><span className="text-sm text-gray-500">Total Violations</span><span className="font-bold text-red-600">{violations.length}</span></div>
          <div className="flex items-center gap-2"><span className="text-sm text-gray-500">Affected Users</span><span className="font-bold">{totalUsers}</span></div>
          {complianceRate < 90 && <div className="flex items-center gap-1 text-xs text-orange-600"><AlertTriangle className="w-3 h-3" /> Below 90% compliance threshold</div>}
        </div>
      </div>

      <div className="flex items-center gap-2">
        <Filter className="w-4 h-4 text-gray-400" />
        <select value={filterRule} onChange={(e) => setFilterRule(e.target.value)} className="px-3 py-1.5 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm">
          <option value="">All Rules</option>
          {ruleTypes.map((r) => <option key={r} value={r}>{r}</option>)}
        </select>
        <span className="text-sm text-gray-500">{filtered.length} violations</span>
      </div>

      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">User</th><th className="px-4 py-3 text-left font-medium">Rule</th><th className="px-4 py-3 text-left font-medium">Detail</th><th className="px-4 py-3 text-left font-medium">Detected</th></tr></thead>
          <tbody className="divide-y dark:divide-gray-800">
            {filtered.map((v) => (
              <tr key={v.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                <td className="px-4 py-3"><span className="font-medium">{v.username}</span><p className="text-xs text-gray-400 font-mono">{v.user_id}</p></td>
                <td className="px-4 py-3"><span className="px-2 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800 font-mono">{v.rule_type}</span></td>
                <td className="px-4 py-3 text-gray-600 dark:text-gray-400">{v.detail}</td>
                <td className="px-4 py-3 text-xs text-gray-500">{v.detected_at}</td>
              </tr>
            ))}
            {filtered.length === 0 && !loading && <tr><td colSpan={4} className="px-4 py-8 text-center text-gray-500">No violations found.</td></tr>}
          </tbody>
        </table>
      </div>
    </div>
  );
}

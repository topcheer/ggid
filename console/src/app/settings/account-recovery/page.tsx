"use client";
import { useTranslations } from "@/lib/i18n";
import { useState, useEffect, useCallback } from "react";
import { LifeBuoy, ShieldCheck, Ban, AlertTriangle, RotateCcw } from "lucide-react";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";
interface RecoveryCode { user_id: string; username: string; total: number; used: number; remaining: number; generated_at: string; }
interface RecoveryConfig { methods: string[]; verification_steps: string[]; enabled: boolean; }
interface AuditEntry { id: string; user: string; action: string; timestamp: string; ip: string; }
export default function AccountRecoveryPage() {
  const t = useTranslations();
  const [config, setConfig] = useState<RecoveryConfig | null>(null);
  const [codes, setCodes] = useState<RecoveryCode[]>([]);
  const [audit, setAudit] = useState<AuditEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch("/api/v1/auth/account-recovery", { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (!res.ok) return null;
      const d = await res.json();
      setConfig(d.config);
      setCodes(d.recovery_codes || []);
      setAudit(d.audit_trail || []);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load account recovery data");
    } finally { setLoading(false); }
  }, []);
  useEffect(() => { fetchData(); }, [fetchData]);
  if (loading) return (
    <div className="p-8 text-center">
      <div className="inline-block w-5 h-5 border-2 border-current border-t-transparent rounded-full animate-spin text-blue-600 mb-2" />
      <div className="text-sm text-gray-500">Loading account recovery...</div>
    </div>
  );
  if (error) return (
    <div className="p-8">
      <div className="rounded-lg border border-red-300 bg-red-50 dark:bg-red-950 dark:border-red-800 p-4">
        <p className="text-red-700 dark:text-red-400 text-sm font-medium flex items-center gap-2"><AlertTriangle className="w-4 h-4" /> Error: {error}</p>
        <button onClick={fetchData} className="mt-2 px-4 py-1.5 rounded-lg bg-red-600 text-white text-sm hover:bg-red-700">Retry</button>
      </div>
    </div>
  );
  if (!config) return <p className="text-sm text-gray-500 text-center py-8">No recovery data available.</p>;
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2"><LifeBuoy className="w-6 h-6 text-blue-500" />{t("accountRecovery.title")}</h1>
          <p className="text-sm text-gray-500 mt-1">Configure recovery methods, identity verification, and recovery codes.</p>
        </div>
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" checked={config.enabled} onChange={(e) => setConfig({ ...config, enabled: e.target.checked })} aria-label="Enable account recovery" className="rounded" /> Enabled
        </label>
      </div>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div className="rounded-lg border dark:border-gray-800 p-4">
          <h3 className="text-sm font-semibold mb-3">Recovery Methods</h3>
          <div className="space-y-2">{config.methods.map((m: any) => (<div key={m} className="flex items-center gap-2 text-sm"><span className="w-2 h-2 rounded-full bg-green-500" />{m}</div>
          ))}</div>
        </div>
        <div className="rounded-lg border dark:border-gray-800 p-4">
          <h3 className="text-sm font-semibold mb-3 flex items-center gap-2"><ShieldCheck className="w-4 h-4 text-gray-400" /> Identity Verification Steps</h3>
          <div className="space-y-2">{config.verification_steps.map((s: any, i: number) => (<div key={i} className="flex items-center gap-2 text-sm">
              <span className="w-5 h-5 rounded-full bg-blue-100 dark:bg-blue-900/30 text-blue-600 text-xs flex items-center justify-center font-bold">{i + 1}</span>{s}
            </div>
          ))}</div>
        </div>
      </div>
      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 dark:bg-gray-900/50">
            <tr>
              <th scope="col" className="px-4 py-3 text-left font-medium">User</th>
              <th scope="col" className="px-4 py-3 text-left font-medium">Total</th>
              <th scope="col" className="px-4 py-3 text-left font-medium">Used</th>
              <th scope="col" className="px-4 py-3 text-left font-medium">Remaining</th>
              <th scope="col" className="px-4 py-3 text-left font-medium">Generated</th>
            </tr>
          </thead>
          <tbody className="divide-y dark:divide-gray-800">
            {codes.map((c: any) => (
              <tr key={c.user_id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                <td className="px-4 py-3 font-medium">{c.username}</td>
                <td className="px-4 py-3 text-xs">{c.total}</td>
                <td className="px-4 py-3 text-xs text-red-600">{c.used}</td>
                <td className="px-4 py-3">
                  <span className={"text-xs font-bold " + (c.remaining === 0 ? "text-red-600" : c.remaining <= 2 ? "text-yellow-600" : "text-green-600")}>{c.remaining}</span>
                </td>
                <td className="px-4 py-3 text-xs text-gray-400">{c.generated_at}</td>
              </tr>
            ))}
            {codes.length === 0 && <tr><td colSpan={5} className="px-4 py-8 text-center text-gray-500">No recovery codes.</td></tr>}
          </tbody>
        </table>
      </div>
      <div className="rounded-lg border dark:border-gray-800 p-4">
        <h3 className="text-sm font-semibold mb-3 flex items-center gap-2"><Ban className="w-4 h-4 text-gray-400" /> Recovery Audit Trail</h3>
        <div className="space-y-1">
          {audit.slice(0, 10).map((a: any) => (
            <div key={a.id} className="flex items-center justify-between text-sm py-1">
              <div>
                <span className="font-medium">{a.user}</span>
                <span className="text-gray-500"> {a.action}</span>
              </div>
              <div className="text-right">
                <span className="text-xs font-mono text-gray-400">{a.ip}</span>
                <span className="text-xs text-gray-400 ml-2">{a.timestamp}</span>
              </div>
            </div>
          ))}
          {audit.length === 0 && <p className="text-xs text-gray-500">No recovery events.</p>}
        </div>
      </div>
    </div>
  );
}

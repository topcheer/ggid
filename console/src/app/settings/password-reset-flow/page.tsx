"use client";
import { useState, useEffect, useCallback } from "react";
import { KeyRound, Save, Mail, MessageSquare, ShieldQuestion, UserCog } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface ResetConfig { methods: { email_link: boolean; sms_code: boolean; security_questions: boolean; admin_reset: boolean }; token_expiry_minutes: number; require_mfa: boolean; reset_after_failed_attempts: number; notify_on_reset: boolean; }
interface ResetEvent { id: string; user: string; method: string; requested_at: string; completed: boolean; ip: string; }

const methodIcons: Record<string, typeof Mail> = { email_link: Mail, sms_code: MessageSquare, security_questions: ShieldQuestion, admin_reset: UserCog };

export default function PasswordResetFlowPage() {
  const t = useTranslations();

  const [config, setConfig] = useState<ResetConfig | null>(null);
  const [history, setHistory] = useState<ResetEvent[]>([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/auth/password-reset-config", { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const d = await res.json(); setConfig(d.config || d); setHistory(d.history || []); }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const save = async () => {
    if (!config) return;
    setSaving(true);
    try { await fetch("/api/v1/auth/password-reset-config", { method: "PUT", headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify(config) }); }
    catch { /* noop */ }
    finally { setSaving(false); }
  };

  if (!config) return <p className="text-sm text-gray-500 text-center py-8">Loading...</p>;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold flex items-center gap-2"><KeyRound className="w-6 h-6 text-orange-500" /> {t("passwordResetFlow.title")}</h1><p className="text-sm text-gray-500 mt-1">Configure password reset methods and security policies.</p></div>
        <button onClick={save} disabled={saving} className="px-4 py-2 rounded-lg bg-orange-600 text-white text-sm font-medium disabled:opacity-50 flex items-center gap-2"><Save className="w-4 h-4" /> Save</button>
      </div>

      <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">Reset Methods</h3><div className="grid grid-cols-2 gap-3">{Object.entries(config.methods).map(([key, val]) => { const Icon = methodIcons[key] || KeyRound; return (<label key={key} className="flex items-center gap-2 text-sm rounded-lg border dark:border-gray-700 p-3"><input aria-label="Val" type="checkbox" checked={val} onChange={(e) => setConfig({ ...config, methods: { ...config.methods, [key]: e.target.checked } })} className="rounded" /> <Icon className="w-4 h-4 text-gray-400" /> {key.replace(/_/g, " ")}</label>); })}</div></div>

      <div className="grid grid-cols-2 gap-4">
        <div className="rounded-lg border dark:border-gray-800 p-4"><label className="text-sm font-medium">Token Expiry (minutes)</label><input aria-label="config" type="number" value={config.token_expiry_minutes} onChange={(e) => setConfig({ ...config, token_expiry_minutes: parseInt(e.target.value) || 0 })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div>
        <div className="rounded-lg border dark:border-gray-800 p-4"><label className="text-sm font-medium">Reset After Failed Attempts</label><input aria-label="config" type="number" value={config.reset_after_failed_attempts} onChange={(e) => setConfig({ ...config, reset_after_failed_attempts: parseInt(e.target.value) || 0 })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div>
      </div>

      <div className="flex items-center gap-6"><label className="flex items-center gap-2 text-sm"><input aria-label="Config" type="checkbox" checked={config.require_mfa} onChange={(e) => setConfig({ ...config, require_mfa: e.target.checked })} className="rounded" /> Require MFA for reset</label><label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={config.notify_on_reset} onChange={(e) => setConfig({ ...config, notify_on_reset: e.target.checked })} className="rounded" /> Notify user on reset</label></div>

      <div className="overflow-x-auto rounded-lg border dark:border-gray-800"><table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">User</th><th className="px-4 py-3 text-left font-medium">Method</th><th className="px-4 py-3 text-left font-medium">Completed</th><th className="px-4 py-3 text-left font-medium">IP</th><th className="px-4 py-3 text-left font-medium">Time</th></tr></thead><tbody className="divide-y dark:divide-gray-800">{history.map((e) => (<tr key={e.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3 font-medium">{e.user}</td><td className="px-4 py-3 text-xs">{e.method}</td><td className="px-4 py-3">{e.completed ? <span className="text-xs text-green-600">Yes</span> : <span className="text-xs text-red-600">No</span>}</td><td className="px-4 py-3 text-xs font-mono text-gray-500">{e.ip}</td><td className="px-4 py-3 text-xs text-gray-400">{e.requested_at}</td></tr>))}{history.length === 0 && <tr><td colSpan={5} className="px-4 py-8 text-center text-gray-500">No reset events.</td></tr>}</tbody></table></div>
    </div>
  );
}

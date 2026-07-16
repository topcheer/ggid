"use client";
import { useTranslations } from "@/lib/i18n";
import { useState, useEffect, useCallback } from "react";
import { Lock, Save } from "lucide-react";
interface PolicyConfig { min_length: number; require_uppercase: boolean; require_lowercase: boolean; require_digit: boolean; require_special: boolean; check_dictionary: boolean; check_breach: boolean; expiry_days: number; history_count: number; per_role: { role: string; min_length: number; expiry_days: number }[]; }
export default function PasswordPolicyConfigPage() {
  const t = useTranslations();
  const [config, setConfig] = useState<PolicyConfig>({ min_length: 12, require_uppercase: true, require_lowercase: true, require_digit: true, require_special: true, check_dictionary: true, check_breach: true, expiry_days: 90, history_count: 5, per_role: [{ role: "admin", min_length: 16, expiry_days: 30 }] });
  const [saving, setSaving] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch("/api/v1/auth/password-policy-config", { headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (!res.ok) return null;
      const data = await res.json();
      if (data) setConfig(prev => ({ ...prev, ...data }));
    } catch (err) { setError(err instanceof Error ? err.message : "An error occurred"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadConfig(); }, [loadConfig]);

  const save = async () => { setSaving(true); try { await fetch("/api/v1/auth/password-policy-config", { method: "PUT", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify(config) }); } catch { /* noop */ } finally { setSaving(false); } };

  if (loading) return (
    <div className="p-8 flex items-center justify-center">
      <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
    </div>
  );

  if (error) return (
    <div className="p-8">
      <div className="rounded-lg border border-red-300 bg-red-50 dark:bg-red-950 dark:border-red-800 p-4">
        <p className="text-red-700 dark:text-red-400 text-sm font-medium">Error: {error}</p>
        <button onClick={loadConfig} className="mt-2 px-4 py-1.5 rounded-lg bg-red-600 text-white text-sm hover:bg-red-700">Retry</button>
      </div>
    </div>
  );

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between"><div><h1 className="text-2xl font-bold flex items-center gap-2"><Lock className="w-6 h-6 text-blue-500" />{t("passwordPolicyConfig.title")}</h1><p className="text-sm text-gray-500 mt-1">Configure password complexity and lifecycle rules.</p></div><button aria-label="Save" onClick={save} disabled={saving} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium flex items-center gap-2"><Save className="w-4 h-4" /> {saving ? "Saving..." : "Save"}</button></div>
      <div className="rounded-lg border dark:border-gray-800 p-4 space-y-4 max-w-lg"><div><label className="text-sm font-medium">Minimum Length</label><div className="flex items-center gap-3 mt-1"><input aria-label="Config" type="range" min={8} max={32} value={config.min_length} onChange={(e) => setConfig({ ...config, min_length: parseInt(e.target.value) })} className="flex-1" /><span className="text-lg font-bold w-10">{config.min_length}</span></div></div><div className="space-y-2"><label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={config.require_uppercase} onChange={(e) => setConfig({ ...config, require_uppercase: e.target.checked })} className="rounded" /> Require Uppercase (A-Z)</label><label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={config.require_lowercase} onChange={(e) => setConfig({ ...config, require_lowercase: e.target.checked })} className="rounded" /> Require Lowercase (a-z)</label><label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={config.require_digit} onChange={(e) => setConfig({ ...config, require_digit: e.target.checked })} className="rounded" /> Require Digit (0-9)</label><label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={config.require_special} onChange={(e) => setConfig({ ...config, require_special: e.target.checked })} className="rounded" /> Require Special Character</label><label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={config.check_dictionary} onChange={(e) => setConfig({ ...config, check_dictionary: e.target.checked })} className="rounded" /> Dictionary Check</label><label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={config.check_breach} onChange={(e) => setConfig({ ...config, check_breach: e.target.checked })} className="rounded" /> Breach Database Check (HIBP)</label></div><div className="grid grid-cols-2 gap-3"><div><label className="text-sm font-medium">Expiry (days)</label><input type="number" min={0} value={config.expiry_days} onChange={(e) => setConfig({ ...config, expiry_days: parseInt(e.target.value) || 0 })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div><div><label className="text-sm font-medium">History Count</label><input type="number" min={0} value={config.history_count} onChange={(e) => setConfig({ ...config, history_count: parseInt(e.target.value) || 0 })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div></div></div>
      <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">Per-Role Overrides</h3><div className="overflow-x-auto"><table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-3 py-2 text-left font-medium">Role</th><th className="px-3 py-2 text-left font-medium">Min Length</th><th className="px-3 py-2 text-left font-medium">Expiry (days)</th></tr></thead><tbody className="divide-y dark:divide-gray-800">{config.per_role.map((r, i) => (<tr key={i}><td className="px-3 py-2"><span className="font-mono text-xs">{r.role}</span></td><td className="px-3 py-2"><input aria-label="r" type="number" value={r.min_length} onChange={(e) => { const o = [...config.per_role]; o[i] = { ...r, min_length: parseInt(e.target.value) || 0 }; setConfig({ ...config, per_role: o }); }} className="w-16 px-2 py-1 rounded border dark:border-gray-700 dark:bg-gray-900 text-xs" /></td><td className="px-3 py-2"><input type="number" value={r.expiry_days} onChange={(e) => { const o = [...config.per_role]; o[i] = { ...r, expiry_days: parseInt(e.target.value) || 0 }; setConfig({ ...config, per_role: o }); }} className="w-16 px-2 py-1 rounded border dark:border-gray-700 dark:bg-gray-900 text-xs" /></td></tr>))}</tbody></table></div></div>
    </div>
  );
}

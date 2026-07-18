"use client";
import { useTranslations } from "@/lib/i18n";

import { useState, useEffect, useCallback } from "react";
import { KeyRound, Save, ToggleLeft, ToggleRight, Clock } from "lucide-react";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface PasswordPolicy {
  max_age_days: number;
  expiry_warning_days: number;
  enforce_after: boolean;
  per_org_override: { org_id: string; org_name: string; max_age_days: number; enabled: boolean }[];
  upcoming_expiry: { user_id: string; username: string; org: string; expires_in_days: number }[];
}

export default function PasswordAgePolicyPage() {
  const t = useTranslations();
  const [policy, setPolicy] = useState<PasswordPolicy | null>(null);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/auth/password-age-policy", { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setPolicy(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const save = async () => {
    if (!policy) return;
    setSaving(true);
    try { await fetch("/api/v1/auth/password-age-policy", { method: "PUT", headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify(policy) }); setSaved(true); setTimeout(() => setSaved(false), 2000); }
    catch { /* noop */ }
    finally { setSaving(false); }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><KeyRound className="w-6 h-6 text-orange-500" />{t("passwordAgePolicy.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Configure password expiration policies with per-organization overrides.</p>
      </div>

      {policy && (
        <>
          <div className="rounded-lg border dark:border-gray-800 p-6 space-y-4 max-w-lg">
            <div><label className="text-sm font-medium">Max Age (days)</label><input aria-label="policy" type="number" min={1} value={policy.max_age_days} onChange={(e) => setPolicy({ ...policy, max_age_days: parseInt(e.target.value) || 0 })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div>
            <div><label className="text-sm font-medium">Expiry Warning (days before)</label><input aria-label="policy" type="number" min={1} value={policy.expiry_warning_days} onChange={(e) => setPolicy({ ...policy, expiry_warning_days: parseInt(e.target.value) || 0 })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div>
            <button onClick={() => setPolicy({ ...policy, enforce_after: !policy.enforce_after })} className="flex items-center gap-2 text-sm">{policy.enforce_after ? <ToggleRight className="w-8 h-8 text-green-500" /> : <ToggleLeft className="w-8 h-8 text-gray-400" />}<span className={policy.enforce_after ? "text-green-600" : "text-gray-500"}>Enforce expiration (lock account after max age)</span></button>
            <div className="flex items-center gap-2"><button aria-label="Save" onClick={save} disabled={saving} className="px-4 py-2 rounded-lg bg-orange-600 text-white text-sm font-medium hover:bg-orange-700 disabled:opacity-50 flex items-center gap-2"><Save className="w-4 h-4" /> {saving ? "Saving..." : "Save"}</button>{saved && <span className="text-sm text-green-600">Saved!</span>}</div>
          </div>

          {policy.per_org_override.length > 0 && (
            <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
              <table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Organization</th><th className="px-4 py-3 text-left font-medium">Max Age (days)</th><th className="px-4 py-3 text-left font-medium">Enabled</th></tr></thead>
                <tbody className="divide-y dark:divide-gray-800">{policy.per_org_override.map((o: any, i: number) => (
                  <tr key={o.org_id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3"><span className="font-medium">{o.org_name}</span><p className="text-xs text-gray-400 font-mono">{o.org_id}</p></td><td className="px-4 py-3"><input aria-label="Input field" type="number" value={o.max_age_days} onChange={(e) => { const next = [...policy.per_org_override]; next[i] = { ...o, max_age_days: parseInt(e.target.value) || 0 }; setPolicy({ ...policy, per_org_override: next }); }} className="w-20 px-2 py-1 rounded border dark:border-gray-700 dark:bg-gray-900 text-sm" /></td><td className="px-4 py-3"><button onClick={() => { const next = [...policy.per_org_override]; next[i] = { ...o, enabled: !o.enabled }; setPolicy({ ...policy, per_org_override: next }); }} className="text-sm">{o.enabled ? <ToggleRight className="w-6 h-6 text-green-500" /> : <ToggleLeft className="w-6 h-6 text-gray-400" />}</button></td></tr>
                ))}</tbody>
              </table>
            </div>
          )}

          {policy.upcoming_expiry.length > 0 && (
            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="text-sm font-semibold flex items-center gap-2 mb-3"><Clock className="w-4 h-4 text-orange-500" /> Upcoming Expirations</h3>
              <div className="space-y-1">{policy.upcoming_expiry.map((u) => (
                <div key={u.user_id} className="flex items-center gap-2 text-sm"><span className="font-medium">{u.username}</span><span className="text-xs text-gray-400">{u.org}</span><span className={`ml-auto px-2 py-0.5 rounded text-xs font-bold ${u.expires_in_days <= 3 ? "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400" : u.expires_in_days <= 7 ? "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400" : "bg-gray-100 dark:bg-gray-800"}`}>{u.expires_in_days}d left</span></div>
              ))}</div>
            </div>
          )}
        </>
      )}
      {!policy && !loading && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
    </div>
  );
}

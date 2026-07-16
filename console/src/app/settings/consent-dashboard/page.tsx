"use client";

import { useState, useEffect, useCallback } from "react";
import { ShieldCheck, Ban, Clock } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface Consent {
  client_id: string;
  client_name: string;
  user_count: number;
  scopes: string[];
  last_granted: string;
  consent_rate: number;
}

interface Dashboard {
  active_consents: Consent[];
  revocation_trend: { day: string; count: number }[];
  pending_expiry: { client_name: string; user: string; expires_at: string; days_left: number }[];
}

export default function ConsentDashboardPage() {
  const t = useTranslations();

  const [data, setData] = useState<Dashboard | null>(null);
  const [loading, setLoading] = useState(false);
  const [selected, setSelected] = useState<string[]>([]);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try { const res = await fetch("/api/v1/oauth/consent-dashboard", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); if (res.ok) setData(await res.json()); }
    catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const bulkRevoke = async () => {
    for (const id of selected) { try { await fetch("/api/v1/oauth/consent-dashboard/" + id + "/revoke", { method: "POST", headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); } catch { /* noop */ } }
    setSelected([]); fetchData();
  };

  const toggleSelect = (id: string) => { setSelected(selected.includes(id) ? selected.filter((s) => s !== id) : [...selected, id]); };
  const maxRev = Math.max(...(data?.revocation_trend.map((t) => t.count) || [1]), 1);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold flex items-center gap-2"><ShieldCheck className="w-6 h-6 text-green-500" /> {t("consentDashboard.title")}</h1><p className="text-sm text-gray-500 mt-1">Monitor active consent grants across all OAuth clients.</p></div>
        {selected.length > 0 && <button onClick={bulkRevoke} className="px-4 py-2 rounded-lg bg-red-600 text-white text-sm font-medium hover:bg-red-700 flex items-center gap-2"><Ban className="w-4 h-4" /> Revoke ({selected.length})</button>}
      </div>

      {data && (
        <>
          <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
            <table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium w-8"></th><th className="px-4 py-3 text-left font-medium">Client</th><th className="px-4 py-3 text-left font-medium">Users</th><th className="px-4 py-3 text-left font-medium">Scopes</th><th className="px-4 py-3 text-left font-medium">Consent Rate</th><th className="px-4 py-3 text-left font-medium">Last Granted</th></tr></thead>
              <tbody className="divide-y dark:divide-gray-800">{data.active_consents.map((c) => (<tr key={c.client_id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3"><input aria-label="Toggle" type="checkbox" checked={selected.includes(c.client_id)} onChange={() => toggleSelect(c.client_id)} className="rounded" /></td><td className="px-4 py-3 font-medium">{c.client_name}</td><td className="px-4 py-3 font-bold">{c.user_count}</td><td className="px-4 py-3"><div className="flex flex-wrap gap-1">{c.scopes.slice(0, 3).map((s, i) => <span key={i} className="px-1.5 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800 font-mono">{s}</span>)}{c.scopes.length > 3 && <span className="text-xs text-gray-400">+{c.scopes.length - 3}</span>}</div></td><td className="px-4 py-3"><span className={"font-bold text-sm " + (c.consent_rate >= 80 ? "text-green-600" : c.consent_rate >= 50 ? "text-yellow-600" : "text-red-600")}>{c.consent_rate.toFixed(0)}%</span></td><td className="px-4 py-3 text-xs text-gray-400">{c.last_granted}</td></tr>))}</tbody>
            </table>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">Revocation Trend (30d)</h3><div className="flex items-end gap-0.5 h-16">{data.revocation_trend.map((t, i) => <div key={i} className="flex-1 bg-red-400 dark:bg-red-500 rounded-t" style={{ height: (t.count / maxRev) * 100 + "%", minHeight: "2px" }} title={t.day + ": " + t.count} />)}</div></div>
            <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">Pending Expiry</h3><div className="space-y-2">{data.pending_expiry.map((e, i) => <div key={i} className="flex items-center gap-2 text-sm"><Clock className="w-3.5 h-3.5 text-orange-500" /><span className="text-xs flex-1">{e.user} - {e.client_name}</span><span className={"text-xs font-bold " + (e.days_left <= 7 ? "text-red-600" : "text-orange-600")}>{e.days_left}d</span></div>)}{data.pending_expiry.length === 0 && <p className="text-xs text-gray-400">None expiring soon.</p>}</div></div>
          </div>
        </>
      )}
      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
    </div>
  );
}

"use client";
import { useState, useEffect, useCallback } from "react";
import { KeyRound, Search, AlertTriangle } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";
interface GrantEvent { id: string; client_name: string; user_id: string; username: string; scopes: string[]; granted_at: string; expires_at: string; revoked_at: string | null; grant_type: string; }
export default function GrantHistoryPage() {
  const t = useTranslations();

  const [events, setEvents] = useState<GrantEvent[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState("");
  const [filterType, setFilterType] = useState("");
  const [showEvidence, setShowEvidence] = useState(false);
  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch("/api/v1/oauth/grant-history", { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (!res.ok) return null;
      const d = await res.json();
      setEvents(d.events || d || []);
    } catch (e) { setError(e instanceof Error ? e.message : "Failed to load grant history"); }
    finally { setLoading(false); }
  }, []);
  useEffect(() => { fetchData(); }, [fetchData]);
  const filtered = events.filter((e: any) => { if (filterType && e.grant_type !== filterType) return false; if (search) { const q = search.toLowerCase(); return e.username.toLowerCase().includes(q) || e.client_name.toLowerCase().includes(q); } return true; });
  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><KeyRound className="w-6 h-6 text-blue-500" /> {t("big1.grantHistory.title")}</h1><p className="text-sm text-gray-500 mt-1">{t("big1.grantHistory.trackOAuthGrantEventsOverTime")}</p></div>
      {error && <div className="rounded-lg border border-red-200 dark:border-red-900 bg-red-50 dark:bg-red-900/20 p-3 text-sm text-red-600 flex items-center justify-between"><span className="flex items-center gap-2"><AlertTriangle className="w-4 h-4" /> {error}</span><button onClick={fetchData} aria-label="Retry loading grant history" className="text-xs underline hover:text-red-700">{t("big1.grantHistory.retry")}</button></div>}
      <div className="flex items-center gap-3">
        <div className="relative flex-1 max-w-xs">
          <Search className="absolute left-2 top-2.5 w-4 h-4 text-gray-400" />
          <input type="text" value={search} onChange={(e) => setSearch(e.target.value)} placeholder="Search client/user..." aria-label="Search grant history" className="w-full pl-8 pr-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" />
        </div>
        <select value={filterType} onChange={(e) => setFilterType(e.target.value)} aria-label="Filter by grant type" className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm">
          <option value="">{t("big1.grantHistory.allGrantTypes")}</option>
          <option value="authorization_code">{t("big1.grantHistory.authorizationCode")}</option>
          <option value="client_credentials">{t("big1.grantHistory.clientCredentials")}</option>
          <option value="refresh_token">{t("big1.grantHistory.refreshToken")}</option>
          <option value="device_code">{t("big1.grantHistory.deviceCode")}</option>
        </select>
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" checked={showEvidence} onChange={(e) => setShowEvidence(e.target.checked)} aria-label="Show consent evidence" className="rounded" />{t("big1.grantHistory.consentEvidence")}</label>
      </div>
      {loading && <div className="rounded-lg border dark:border-gray-800 p-8 text-center"><div className="inline-block w-5 h-5 border-2 border-current border-t-transparent rounded-full animate-spin text-blue-600 mb-2" /><div className="text-sm text-gray-500">{t("big1.grantHistory.loadingGrantHistory")}</div></div>}
      <div className="relative pl-8">
        <div className="absolute left-3 top-0 bottom-0 w-px bg-gray-200 dark:bg-gray-800" />
        <div className="space-y-3">
          {filtered.map((e: any) => (
            <div key={e.id} className="relative">
              <div className={"absolute -left-5 w-4 h-4 rounded-full border-2 " + (e.revoked_at ? "bg-red-500 border-red-200" : "bg-green-500 border-green-200")} />
              <div className="rounded-lg border dark:border-gray-800 p-3 ml-2">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium">{e.client_name}</span>
                    <span className="text-xs text-gray-400">{e.grant_type}</span>
                  </div>
                  <span className="text-xs text-gray-400">{e.granted_at}</span>
                </div>
                <div className="mt-1 text-sm"><span className="text-gray-500">{t("big1.grantHistory.user")}</span><span className="font-medium">{e.username}</span></div>
                <div className="mt-1 flex flex-wrap gap-1">{e.scopes.map((s: any, i: number) => <span key={i} className="px-1.5 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800 font-mono">{s}</span>)}</div>
                <div className="mt-1 text-xs text-gray-400">{t("big1.grantHistory.expires")}{e.expires_at}{e.revoked_at && <span className="text-red-500 ml-2">{t("big1.grantHistory.revoked")}{e.revoked_at}</span>}</div>
                {showEvidence && <div className="mt-2 border-t dark:border-gray-800 pt-1 text-xs text-gray-400">{t("big1.grantHistory.consentIP1921681100EvidenceHash0xabc123")}</div>}
              </div>
            </div>
          ))}
          {filtered.length === 0 && !loading && <p className="text-sm text-gray-500 py-4 ml-2">{t("big1.grantHistory.noGrantEvents")}</p>}
        </div>
      </div>
    </div>
  );
}

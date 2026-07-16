"use client";
import { useState, useCallback } from "react";
import { Search, Monitor, Ban, ShieldCheck, Info, AlertTriangle } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
interface Session { id: string; device: string; ip_address: string; location: string; created_at: string; last_active: string; mfa_verified: boolean; scopes: string[]; expires_at: string; }
export default function SessionInspectorPage() {
  const t = useTranslations();
  const [search, setSearch] = useState("");
  const [sessions, setSessions] = useState<Session[]>([]);
  const [selected, setSelected] = useState<Session | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const searchUser = useCallback(async () => {
    if (!search) return;
    setLoading(true);
    setError(null);
    try {
      const res = await fetch("/api/v1/auth/session-inspector?user=" + encodeURIComponent(search), { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (!res.ok) return null;
      const d = await res.json();
      setSessions(d.sessions || d || []);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to search sessions");
    } finally { setLoading(false); }
  }, [search]);
  const revoke = async (id: string) => {
    try {
      const res = await fetch("/api/v1/auth/session-inspector/" + id, { method: "DELETE", headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (!res.ok) return null;
      setSessions(sessions.filter((s) => s.id !== id));
      if (selected?.id === id) setSelected(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to revoke session");
    }
  };
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Monitor className="w-6 h-6 text-blue-500" /> Session Inspector</h1>
        <p className="text-sm text-gray-500 mt-1">{t("sessionInspector.subtitle")}</p>
      </div>
      {error && <div className="rounded-lg border border-red-200 dark:border-red-900 bg-red-50 dark:bg-red-900/20 p-3 text-sm text-red-600 flex items-center justify-between"><span className="flex items-center gap-2"><AlertTriangle className="w-4 h-4" /> {error}</span><button onClick={() => setError(null)} className="text-xs underline hover:text-red-700">{t("sessionInspector.dismiss")}</button></div>}
      <div className="flex items-center gap-2">
        <div className="relative flex-1 max-w-md">
          <Search className="absolute left-2 top-2.5 w-4 h-4 text-gray-400" />
          <input type="text" value={search} onChange={(e) => setSearch(e.target.value)} onKeyDown={(e) => { if (e.key === "Enter") searchUser(); }} placeholder={t("sessionInspector.searchUser")} aria-label={t("sessionInspector.searchSessions")} className="w-full pl-8 pr-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" />
        </div>
        <button onClick={searchUser} disabled={loading || !search} aria-label={t("sessionInspector.searchSessions")} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium disabled:opacity-50">{loading ? t("sessionInspector.searching") : t("sessionInspector.search")}</button>
      </div>
      {loading && <div className="rounded-lg border dark:border-gray-800 p-8 text-center"><div className="inline-block w-5 h-5 border-2 border-current border-t-transparent rounded-full animate-spin text-blue-600 mb-2" /><div className="text-sm text-gray-500">{t("sessionInspector.searchingSessions")}</div></div>}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2 space-y-2">
          {sessions.map((s) => (
            <div key={s.id} className={"rounded-lg border p-3 cursor-pointer " + (selected?.id === s.id ? "border-blue-500" : "dark:border-gray-800")} onClick={() => setSelected(s)}>
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Monitor className="w-4 h-4 text-gray-400" />
                  <span className="text-sm font-medium">{s.device}</span>
                  {s.mfa_verified && <span className="flex items-center gap-1 text-xs text-green-600"><ShieldCheck className="w-3 h-3" /> MFA</span>}
                </div>
                <button onClick={(e) => { e.stopPropagation(); revoke(s.id); }} aria-label={`Revoke session ${s.id}`} className="text-xs text-red-600 hover:underline flex items-center gap-1"><Ban className="w-3 h-3" /> Revoke</button>
              </div>
              <div className="flex items-center gap-3 mt-1 text-xs text-gray-500">
                <span className="font-mono">{s.ip_address}</span>
                <span>{s.location}</span>
                <span>{t("sessionInspector.last")} {s.last_active}</span>
              </div>
            </div>
          ))}
          {sessions.length === 0 && !loading && <p className="text-sm text-gray-500 text-center py-8">{t("sessionInspector.noSessions")}</p>}
        </div>
        {selected && (
          <div className="rounded-lg border dark:border-gray-800 p-4 space-y-3">
            <div className="flex items-center gap-2"><Info className="w-4 h-4 text-blue-500" /><h3 className="text-sm font-semibold">{t("sessionInspector.sessionDetails")}</h3></div>
            <div className="text-xs space-y-1">
              <div><span className="text-gray-500">{t("sessionInspector.sessionId")}</span> <span className="font-mono">{selected.id}</span></div>
              <div><span className="text-gray-500">{t("sessionInspector.device")}</span> {selected.device}</div>
              <div><span className="text-gray-500">{t("sessionInspector.ip")}</span> <span className="font-mono">{selected.ip_address}</span></div>
              <div><span className="text-gray-500">{t("sessionInspector.location")}</span> {selected.location}</div>
              <div><span className="text-gray-500">{t("sessionInspector.created")}</span> {selected.created_at}</div>
              <div><span className="text-gray-500">Expires:</span> {selected.expires_at}</div>
              <div><span className="text-gray-500">MFA:</span> {selected.mfa_verified ? "Verified" : "Not verified"}</div>
            </div>
            <div><span className="text-xs text-gray-500">Scopes:</span><div className="flex flex-wrap gap-1 mt-1">{selected.scopes.map((sc, i) => <span key={i} className="px-1.5 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800 font-mono">{sc}</span>)}</div></div>
            <button onClick={() => revoke(selected.id)} aria-label="Revoke selected session" className="w-full px-3 py-1.5 rounded-lg bg-red-600 text-white text-xs font-medium">Revoke Session</button>
          </div>
        )}
      </div>
    </div>
  );
}

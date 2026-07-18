"use client";
import { useState, useEffect, useCallback } from "react";
import { Settings2, Save, X } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface ClaimMap { id: string; claim_name: string; source: "user_attr" | "group" | "static"; source_value: string; transform: string; }
interface ClientOverride { client_id: string; client_name: string; extra_claims: number; }
const scopesList = ["openid", "profile", "email", "groups", "offline_access"];

export default function OidcClaimMappingPage() {
  const [mappings, setMappings] = useState<ClaimMap[]>([]);
  const [overrides, setOverrides] = useState<ClientOverride[]>([]);
  const [scopeMatrix, setScopeMatrix] = useState<Record<string, string[]>>({});
  const [tokenType, setTokenType] = useState("id_token");
  const [loading, setLoading] = useState(false);
  const [showAdd, setShowAdd] = useState(false);
  const [form, setForm] = useState<{ claim_name: string; source: "user_attr" | "group" | "static"; source_value: string; transform: string }>({ claim_name: "", source: "user_attr", source_value: "", transform: "" });
  const t = useTranslations();

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/oauth/claim-mapping", { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const d = await res.json(); setMappings(d.mappings || []); setOverrides(d.client_overrides || []); setScopeMatrix(d.scope_claims || {}); setTokenType(d.token_type || "id_token"); }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold flex items-center gap-2"><Settings2 className="w-6 h-6 text-blue-500" /> {t("oidcClaimMapping.title")}</h1><p className="text-sm text-gray-500 mt-1">{t("oidcClaimMapping.subtitle")}</p></div>
        <button onClick={() => setShowAdd(true)} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium">{t("oidcClaimMapping.addClaim")}</button>
      </div>

      <div className="flex items-center gap-3"><label className="text-sm font-medium">{t("oidcClaimMapping.tokenType")}</label><select aria-label="Token type" value={tokenType} onChange={(e) => setTokenType(e.target.value)} className="px-3 py-1.5 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"><option value="id_token">{t("oidcClaimMapping.idToken")}</option><option value="access_token">{t("oidcClaimMapping.accessToken")}</option><option value="userinfo">{t("oidcClaimConfig.userInfo")}</option></select></div>

      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">{t("oidcClaimMapping.claim")}</th><th className="px-4 py-3 text-left font-medium">{t("oidcClaimMapping.source")}</th><th className="px-4 py-3 text-left font-medium">{t("oidcClaimMapping.sourceValue")}</th><th className="px-4 py-3 text-left font-medium">{t("oidcClaimMappingConfig.transform")}</th></tr></thead>
          <tbody className="divide-y dark:divide-gray-800">{mappings.map((m: any) => (<tr key={m.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3 font-mono text-xs font-medium">{m.claim_name}</td><td className="px-4 py-3"><span className="px-2 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800">{m.source}</span></td><td className="px-4 py-3 text-xs text-gray-500">{m.source_value || "-"}</td><td className="px-4 py-3 text-xs text-gray-500">{m.transform || "-"}</td></tr>))}{mappings.length === 0 && !loading && <tr><td colSpan={4} className="px-4 py-8 text-center text-gray-500">{t("oidcClaimMapping.noMappings")}</td></tr>}</tbody>
        </table>
      </div>

      <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">{t("oidcClaimMapping.scopeToClaims")}</h3><div className="overflow-x-auto"><table className="w-full text-sm"><thead><tr><th className="px-3 py-2 text-left text-xs font-medium text-gray-500">{t("oidcClaimMapping.scope")}</th><th className="px-3 py-2 text-left text-xs font-medium text-gray-500">{t("oidcClaimMapping.claims")}</th></tr></thead><tbody>{scopesList.map((s: any) => (<tr key={s} className="border-t dark:border-gray-800"><td className="px-3 py-2 font-mono text-xs font-medium">{s}</td><td className="px-3 py-2"><div className="flex flex-wrap gap-1">{(scopeMatrix[s] || []).map((c: any) => (<span key={c} className="px-1.5 py-0.5 rounded text-xs bg-blue-50 dark:bg-blue-900/20 font-mono">{c}</span>))}{(scopeMatrix[s] || []).length === 0 && <span className="text-xs text-gray-400">{t("common.none")}</span>}</div></td></tr>))}</tbody></table></div></div>

      {overrides.length > 0 && (<div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-2">{t("oidcClaimMapping.perClient")}</h3>{overrides.map((o: any) => (<div key={o.client_id} className="flex items-center justify-between text-sm py-1"><span className="font-medium">{o.client_name}</span><span className="text-xs text-gray-500">{o.extra_claims} extra claim(s)</span></div>))}</div>)}

      {showAdd && (<div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setShowAdd(false)}><div role="dialog" aria-modal="true" className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4" onClick={(e) => e.stopPropagation()}><div className="flex items-center justify-between px-6 py-4 border-b dark:border-gray-800"><h3 className="font-semibold">{t("oidcClaimMapping.addMapping")}</h3><button onClick={() => setShowAdd(false)} aria-label="Close"><X className="w-5 h-5 text-gray-400" /></button></div><div className="px-6 py-4 space-y-3"><div><label className="text-sm font-medium">{t("oidcClaimMapping.claimName")}</label><input type="text" value={form.claim_name} onChange={(e) => setForm({ ...form, claim_name: e.target.value })} placeholder="department" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div><div><label className="text-sm font-medium">{t("oidcClaimMapping.source")}</label><select value={form.source} onChange={(e) => setForm({ ...form, source: e.target.value as "user_attr" | "group" | "static" })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm"><option value="user_attr">User Attribute</option><option value="group">{t("oidcClaimMapping.group")}</option><option value="static">{t("oidcClaimMapping.static")}</option></select></div><div><label className="text-sm font-medium">{t("oidcClaimMapping.sourceValue")}</label><input type="text" value={form.source_value} onChange={(e) => setForm({ ...form, source_value: e.target.value })} placeholder="user.department" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div></div><div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800"><button onClick={() => setShowAdd(false)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">{t("oidcClaimMapping.cancel")}</button><button onClick={() => setShowAdd(false)} disabled={!form.claim_name} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium disabled:opacity-50">{t("oidcClaimMapping.save")}</button></div></div></div>)}
    </div>
  );
}

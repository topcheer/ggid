"use client";
import { useState, useEffect } from "react";
import { Settings2, Plus, X, Save, Loader2 } from "lucide-react";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";
interface CustomClaim { name: string; source: string; value: string; }
interface ClaimConfig { standard_claims: Record<string, boolean>; custom_claims: CustomClaim[]; scope_mappings: Record<string, string[]>; token_type: string; }
const standardClaims = ["sub", "name", "email", "email_verified", "given_name", "family_name", "middle_name", "nickname", "preferred_username", "profile", "picture", "website", "gender", "birthdate", "zoneinfo", "locale", "phone_number", "address", "updated_at"];
const scopes = ["openid", "profile", "email", "address", "phone", "offline_access"];
const defaultConfig: ClaimConfig = { standard_claims: { sub: true, name: true, email: true, email_verified: true }, custom_claims: [{ name: "department", source: "ldap", value: "ou" }], scope_mappings: { openid: ["sub"], profile: ["name", "family_name"], email: ["email", "email_verified"] }, token_type: "id_token" };
export default function OidcClaimConfigPage() {
  const { apiFetch, TENANT_ID } = useApi();
  const [config, setConfig] = useState<ClaimConfig>(defaultConfig);
  const [saving, setSaving] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showAdd, setShowAdd] = useState(false);
  const [form, setForm] = useState({ name: "", source: "", value: "" });
  const t = useTranslations();

  useEffect(() => {
    const loadConfig = async () => {
      setLoading(true);
      setError(null);
      try {
        const data = await apiFetch<ClaimConfig>("/api/v1/oauth/oidc-claim-config");
        if (data) setConfig(data);
      } catch {
        // Use defaults if API unavailable
      } finally {
        setLoading(false);
      }
    };
    loadConfig();
  }, [apiFetch]);

  const save = async () => { setSaving(true); try { await apiFetch("/api/v1/oauth/oidc-claim-config", { method: "PUT", body: JSON.stringify(config) }); } catch (err) { setError("Failed to save OIDC claim config"); } finally { setSaving(false); } };
  const toggleClaim = (c: string) => setConfig({ ...config, standard_claims: { ...config.standard_claims, [c]: !config.standard_claims[c] } });
  const addCustom = () => { if (!form.name) return; setConfig({ ...config, custom_claims: [...config.custom_claims, { ...form }] }); setShowAdd(false); setForm({ name: "", source: "", value: "" }); };
  const removeCustom = (i: number) => setConfig({ ...config, custom_claims: config.custom_claims.filter((_, idx) => idx !== i) });
  const toggleMapping = (scope: string, claim: string) => { const cur = config.scope_mappings[scope] || []; const next = cur.includes(claim) ? cur.filter((c) => c !== claim) : [...cur, claim]; setConfig({ ...config, scope_mappings: { ...config.scope_mappings, [scope]: next } }); };
  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-6 w-6 animate-spin text-blue-500" />
        <span className="ml-2 text-sm text-gray-500">{t("oidcClaimConfig.loading")}</span>
      </div>
    );
  }
  return (
    <div className="space-y-6">
      {error && (
        <div className="rounded-lg border border-red-200 bg-red-50 dark:border-red-900 dark:bg-red-950/30 p-3">
          <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
        </div>
      )}
      <div className="flex items-center justify-between"><div><h1 className="text-2xl font-bold flex items-center gap-2"><Settings2 className="w-6 h-6 text-blue-500" /> {t("oidcClaimConfig.title")}</h1><p className="text-sm text-gray-500 mt-1">{t("oidcClaimConfig.subtitle")}</p></div><button onClick={save} disabled={saving} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium flex items-center gap-2"><Save className="w-4 h-4" /> {saving ? t("common.loading") : t("common.save")}</button></div>
      <div className="rounded-lg border dark:border-gray-800 p-4"><div className="flex items-center gap-3 mb-3"><label className="text-sm font-medium">{t("oidcClaimConfig.tokenType")}</label><select value={config.token_type} onChange={(e) => setConfig({ ...config, token_type: e.target.value })} className="px-2 py-1.5 rounded border dark:border-gray-700 dark:bg-gray-900 text-xs"><option value="id_token">{t("oidcClaimConfig.idToken")}</option><option value="access_token">{t("oidcClaimConfig.accessToken")}</option><option value="userinfo">{t("oidcClaimConfig.userInfo")}</option></select></div><h3 className="text-sm font-semibold mb-2">{t("oidcClaimConfig.standardClaims")}</h3><div className="grid grid-cols-3 md:grid-cols-4 gap-2">{standardClaims.map((c) => <label key={c} className="flex items-center gap-1.5 text-xs"><input type="checkbox" checked={config.standard_claims[c] || false} onChange={() => toggleClaim(c)} className="rounded" /> {c}</label>)}</div></div>
      <div className="rounded-lg border dark:border-gray-800 p-4"><div className="flex items-center justify-between mb-3"><h3 className="text-sm font-semibold">{t("oidcClaimConfig.customClaims")}</h3><button onClick={() => setShowAdd(true)} className="text-xs text-blue-600 flex items-center gap-1"><Plus className="w-3 h-3" /> {t("oidcClaimConfig.add")}</button></div><div className="overflow-x-auto"><table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-3 py-2 text-left font-medium">{t("oidcClaimConfig.name")}</th><th className="px-3 py-2 text-left font-medium">{t("oidcClaimConfig.source")}</th><th className="px-3 py-2 text-left font-medium">{t("oidcClaimConfig.value")}</th><th className="px-3 py-2 text-left font-medium"></th></tr></thead><tbody className="divide-y dark:divide-gray-800">{config.custom_claims.map((c, i) => (<tr key={i} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-3 py-2 font-mono text-xs">{c.name}</td><td className="px-3 py-2 text-xs text-gray-500">{c.source}</td><td className="px-3 py-2 text-xs font-mono">{c.value}</td><td className="px-3 py-2"><button onClick={() => removeCustom(i)} className="text-red-500"><X className="w-3.5 h-3.5" /></button></td></tr>))}</tbody></table></div></div>
      <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">{t("oidcClaimConfig.scopeToClaim")}</h3><div className="overflow-x-auto"><table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-3 py-2 text-left font-medium">{t("oidcClaimConfig.scope")}</th><th className="px-3 py-2 text-left font-medium">{t("oidcClaimConfig.claims")}</th></tr></thead><tbody className="divide-y dark:divide-gray-800">{scopes.map((scope) => (<tr key={scope} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-3 py-2 font-mono text-xs font-medium">{scope}</td><td className="px-3 py-2"><div className="flex flex-wrap gap-1">{standardClaims.filter((c) => config.standard_claims[c]).map((c) => { const active = (config.scope_mappings[scope] || []).includes(c); return (<button key={c} onClick={() => toggleMapping(scope, c)} className={"px-1.5 py-0.5 rounded text-xs font-mono " + (active ? "bg-blue-100 dark:bg-blue-900/30 dark:text-blue-400" : "bg-gray-100 dark:bg-gray-800 text-gray-400")}>{c}</button>); })}</div></td></tr>))}</tbody></table></div></div>
      {showAdd && (<div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setShowAdd(false)}><div className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4" onClick={(e) => e.stopPropagation()}><div className="flex items-center justify-between px-6 py-4 border-b dark:border-gray-800"><h3 className="font-semibold">{t("oidcClaimConfig.addCustomClaim")}</h3><button onClick={() => setShowAdd(false)}><X className="w-5 h-5 text-gray-400" /></button></div><div className="px-6 py-4 space-y-3"><div><label className="text-sm font-medium">{t("oidcClaimConfig.name")}</label><input type="text" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm font-mono" /></div><div><label className="text-sm font-medium">{t("oidcClaimConfig.source")}</label><input type="text" value={form.source} onChange={(e) => setForm({ ...form, source: e.target.value })} placeholder="ldap / db / header" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div><div><label className="text-sm font-medium">{t("oidcClaimConfig.value")}</label><input type="text" value={form.value} onChange={(e) => setForm({ ...form, value: e.target.value })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm font-mono" /></div></div><div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800"><button onClick={() => setShowAdd(false)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">{t("oidcClaimConfig.cancel")}</button><button onClick={addCustom} disabled={!form.name} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium disabled:opacity-50">{t("oidcClaimConfig.add")}</button></div></div></div>)}
    </div>
  );
}

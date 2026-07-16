"use client";
import { useEffect, useState } from "react";
import { useOAuthTokenExchangeConfig, OAuthTokenExchangeConfig, PerClientScopes } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

export default function OAuthTokenExchangeConfigPage() {
  const { config, loading, error, fetchConfig, updateConfig } = useOAuthTokenExchangeConfig();
  const [form, setForm] = useState<OAuthTokenExchangeConfig | null>(null);
  const [saving, setSaving] = useState(false);
  const t = useTranslations();

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };

  if (loading && !form) return <div className="p-8">{t("oauthTokenExchange.loading")}</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">{t("oauthTokenExchange.noData")}</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">{t("oauthTokenExchange.title")}</h1>
      <p className="text-gray-600">{t("oauthTokenExchange.subtitle")}</p>

      <div className="flex items-center gap-3 bg-white rounded-lg p-4 shadow">
        <input type="checkbox" checked={form.enabled} onChange={(e) => setForm({ ...form, enabled: e.target.checked })} className="w-5 h-5" />
        <label className="font-medium">{t("oauthTokenExchange.enable")}</label>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">{t("oauthTokenExchange.tokenTypes")}</h2>
        <div>
          <label className="block text-sm font-medium mb-1">{t("oauthTokenExchange.allowedSubjectTokens")}</label>
          <div className="text-sm text-gray-600">{form.allowed_subject_token_types.join(", ")}</div>
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">{t("oauthTokenExchange.allowedActorTokens")}</label>
          <div className="text-sm text-gray-600">{form.allowed_actor_token_types.join(", ")}</div>
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">{t("oauthTokenExchange.policySettings")}</h2>
        <div>
          <label className="block text-sm font-medium mb-1">{t("oauthTokenExchange.audienceRestriction")}</label>
          <select value={form.audience_restriction_policy} onChange={(e) => setForm({ ...form, audience_restriction_policy: e.target.value as OAuthTokenExchangeConfig["audience_restriction_policy"] })} className="border rounded px-3 py-2">
            <option value="strict">{t("oauthTokenExchange.strict")}</option><option value="permissive">{t("oauthTokenExchange.permissive")}</option><option value="none">{t("oauthTokenExchange.none")}</option>
          </select>
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">{t("oauthTokenExchange.maxDelegationDepth")}</label>
          <input type="number" value={form.max_delegation_depth} onChange={(e) => setForm({ ...form, max_delegation_depth: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-32" />
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">{t("oauthTokenExchange.perClientScopes")}</h2>
        <table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">{t("oauthTokenExchange.client")}</th><th scope="col">{t("oauthTokenExchange.allowedScopes")}</th></tr></thead><tbody>
          {form.per_client_allowed_scopes.map((c: PerClientScopes, i: number) => (
            <tr key={i} className="border-b"><td className="py-2"><span className="font-medium">{c.client_name}</span><div className="text-xs text-gray-400">{c.client_id}</div></td><td>{c.allowed_scopes.join(", ")}</td></tr>
          ))}
        </tbody></table>
      </div>

      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}

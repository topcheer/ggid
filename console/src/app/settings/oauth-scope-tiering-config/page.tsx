"use client";
import { useEffect, useState } from "react";
import { useTranslations } from "@/lib/i18n";
import { useOAuthScopeTieringConfig, OAuthScopeTieringConfig, TierDefinition, ScopePackage, ScopeInheritanceRule } from "@ggid/sdk-react";

export default function OAuthScopeTieringConfigPage() {
  const t = useTranslations();
  const { config, loading, error, fetchConfig, updateConfig } = useOAuthScopeTieringConfig();
  const [form, setForm] = useState<OAuthScopeTieringConfig | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => {
    if (!form) return;
    setSaving(true);
    await updateConfig(form);
    setSaving(false);
  };

  if (loading && !form) return <div className="p-8">{t("oauthScopeTiering.loading")}</div>;
  if (error) return <div className="p-8 text-red-600">{t("common.error")}: {error}</div>;
  if (!form) return <div className="p-8">{t("oauthScopeTiering.noData")}</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">{t("oauthScopeTiering.title")}</h1>
      <p className="text-gray-600">{t("oauthScopeTiering.subtitle")}</p>
      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">{t("oauthScopeTiering.general")}</h2>
        <div className="flex items-center gap-3">
          <input type="checkbox" checked={form.least_privilege_defaults} onChange={(e) => setForm({ ...form, least_privilege_defaults: e.target.checked })} className="w-4 h-4" />
          <label>{t("oauthScopeTiering.leastPrivilege")}</label>
        </div>
        <div className="flex items-center gap-3">
          <input type="checkbox" checked={form.migration_from_flat_scopes} onChange={(e) => setForm({ ...form, migration_from_flat_scopes: e.target.checked })} className="w-4 h-4" />
          <label>{t("oauthScopeTiering.migration")}</label>
        </div>
      </div>
      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">{t("oauthScopeTiering.tierDefinitions")}</h2>
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b text-left">
              <th className="py-2">{t("oauthScopeTiering.tier")}</th>
              <th>{t("oauthScopeTiering.consentPolicy")}</th>
            </tr>
          </thead>
          <tbody>
            {form.tier_definitions.map((td: TierDefinition, i: number) => (
              <tr key={i} className="border-b">
                <td className="py-2 font-medium">{td.tier}</td>
                <td><span className="px-2 py-1 bg-blue-100 text-blue-700 rounded text-xs">{td.consent_policy}</span></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">{t("oauthScopeTiering.scopePackages")}</h2>
        <div className="space-y-2">
          {form.scope_packages.map((p: ScopePackage, i: number) => (
            <div key={i} className="border-b pb-2">
              <div className="font-medium">{p.name}</div>
              <div className="text-sm text-gray-500 font-mono">{p.scopes.join(", ")}</div>
            </div>
          ))}
        </div>
      </div>
      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">{t("oauthScopeTiering.scopeInheritance")}</h2>
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b text-left">
              <th className="py-2">{t("oauthScopeTiering.parentScope")}</th>
              <th>{t("oauthScopeTiering.childScopes")}</th>
            </tr>
          </thead>
          <tbody>
            {form.scope_inheritance_rules.map((r: ScopeInheritanceRule, i: number) => (
              <tr key={i} className="border-b">
                <td className="py-2 font-mono">{r.parent_scope}</td>
                <td className="text-xs font-mono">{r.child_scopes.join(", ")}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">
        {saving ? t("common.loading") : t("common.save")}
      </button>
    </div>
  );
}

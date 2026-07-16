"use client";
import { useEffect, useState } from "react";
import { useFederationPatternsConfig, FederationPatternsConfig, TrustLifecycleRule, MemberProvider } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

export default function FederationPatternsConfigPage() {
  const t = useTranslations();

  const { config, loading, error, fetchConfig, updateConfig } = useFederationPatternsConfig();
  const [form, setForm] = useState<FederationPatternsConfig | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };

  if (loading && !form) return <div className="p-8">{t("big1.federationPatternsConfig.loading")}</div>;
  if (error) return <div className="p-8 text-red-600">{t("big1.federationPatternsConfig.error")}{error}</div>;
  if (!form) return <div className="p-8">{t("big1.federationPatternsConfig.noData")}</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">{t("big1.federationPatternsConfig.title")}</h1>
      <p className="text-gray-600">{t("big1.federationPatternsConfig.configureFederationTopologyTrustLifecycleAndMemberProviders")}</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">{t("big1.federationPatternsConfig.federationTopology")}</h2>
        <div><label className="block text-sm font-medium mb-1">{t("big1.federationPatternsConfig.pattern")}</label><select aria-label="Select option" value={form.pattern} onChange={(e) => setForm({ ...form, pattern: e.target.value as FederationPatternsConfig["pattern"] })} className="border rounded px-3 py-2"><option value="hub_spoke">{t("big1.federationPatternsConfig.hubSpoke")}</option><option value="bilateral">{t("big1.federationPatternsConfig.bilateral")}</option><option value="multi_party">{t("big1.federationPatternsConfig.multiParty")}</option></select></div>
        <div><label className="block text-sm font-medium mb-1">{t("big1.federationPatternsConfig.discoveryMethod")}</label><select value={form.discovery_method} onChange={(e) => setForm({ ...form, discovery_method: e.target.value as FederationPatternsConfig["discovery_method"] })} className="border rounded px-3 py-2"><option value="metadata_url">{t("big1.federationPatternsConfig.metadataUrl")}</option><option value="webfinger">{t("big1.federationPatternsConfig.webfinger")}</option><option value="dns">{t("big1.federationPatternsConfig.dns")}</option><option value="manual">{t("big1.federationPatternsConfig.manual")}</option></select></div>
        <div><label className="block text-sm font-medium mb-1">{t("big1.federationPatternsConfig.attributeMappingPolicy")}</label><input type="text" value={form.attribute_mapping_policy} onChange={(e) => setForm({ ...form, attribute_mapping_policy: e.target.value })} className="border rounded px-3 py-2 w-full" /></div>
        <div className="flex items-center gap-3"><input type="checkbox" checked={form.slo_propagation} onChange={(e) => setForm({ ...form, slo_propagation: e.target.checked })} className="w-4 h-4" /><label>{t("big1.federationPatternsConfig.sloPropagation")}</label></div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">{t("big1.federationPatternsConfig.trustLifecycleRules")}</h2>
        <table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">{t("big1.federationPatternsConfig.name")}</th><th scope="col">{t("big1.federationPatternsConfig.description")}</th><th>{t("big1.federationPatternsConfig.autoRevokeDays")}</th></tr></thead><tbody>
          {form.trust_lifecycle_rules.map((r: TrustLifecycleRule, i: number) => (
            <tr key={i} className="border-b"><td className="py-2 font-medium">{r.name}</td><td className="text-xs">{r.description}</td><td>{r.auto_revoke_after_days}</td></tr>
          ))}
        </tbody></table>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">{t("big1.federationPatternsConfig.memberProviders")}</h2>
        <div className="space-y-2">
          {form.member_providers.map((p: MemberProvider, i: number) => (
            <div key={i} className="flex items-center justify-between border-b py-2">
              <div><span className="font-medium">{p.name}</span><span className="ml-2 text-xs text-gray-400">{p.entity_id}</span></div>
              <div className="flex items-center gap-3">
                <span className="text-xs text-gray-500">{t("big1.federationPatternsConfig.joined")}{p.joined}</span>
                <span className={`px-2 py-1 rounded text-xs ${p.status === "active" ? "bg-green-100 text-green-700" : "bg-gray-100 text-gray-500"}`}>{p.status}</span>
              </div>
            </div>
          ))}
        </div>
      </div>

      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? t("big1.federationPatternsConfig.saving") : t("big1.federationPatternsConfig.saveChanges")}</button>
    </div>
  );
}

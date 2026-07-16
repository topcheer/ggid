"use client";
import { useEffect, useState } from "react";
import { useIdentityAttributeSchemaConfig, IdentityAttributeSchemaConfig, CustomAttribute } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

export default function IdentityAttributeSchemaConfigPage() {
  const { config, loading, error, fetchConfig, updateConfig } = useIdentityAttributeSchemaConfig();
  const [form, setForm] = useState<IdentityAttributeSchemaConfig | null>(null);
  const [saving, setSaving] = useState(false);
  useEffect(() => { fetchConfig(); }, [fetchConfig]); useEffect(() => { if (config) setForm(config); }, [config]);
  const t = useTranslations();
  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };
  if (loading && !form) return <div className="p-8">{t("idAttributeSchema.loading")}</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">{t("idAttributeSchema.noData")}</div>;
  const privColors: Record<string, string> = { public: "bg-green-100 text-green-700", internal: "bg-blue-100 text-blue-700", confidential: "bg-yellow-100 text-yellow-700", restricted: "bg-red-100 text-red-700" };

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">{t("idAttributeSchema.title")}</h1>
      <p className="text-gray-600">{t("idAttributeSchema.subtitle")}</p>
      <div className="bg-white rounded-lg p-6 shadow space-y-3"><h2 className="text-lg font-semibold">{t("idAttributeSchema.standardAttrs")}</h2><div className="text-sm text-gray-600">{form.standard_attributes.join(", ")}</div><div className="flex items-center gap-3"><input aria-label="Form" type="checkbox" checked={form.schema_extension} onChange={(e) => setForm({ ...form, schema_extension: e.target.checked })} className="w-4 h-4" /><label>{t("idAttributeSchema.allowExtension")}</label></div></div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">{t("idAttributeSchema.customAttrs")}</h2><table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">{t("idAttributeSchema.name")}</th><th className="py-2">{t("idAttributeSchema.type")}</th><th className="py-2">{t("idAttributeSchema.multi")}</th><th className="py-2">{t("idAttributeSchema.required")}</th><th className="py-2">{t("idAttributeSchema.validation")}</th><th className="py-2">{t("idAttributeSchema.privacy")}</th></tr></thead><tbody>{form.custom_attributes.map((a: CustomAttribute, i: number) => (<tr key={i} className="border-b"><td className="py-2 font-medium">{a.name}</td><td>{a.type}</td><td>{a.multi_valued ? "Yes" : "No"}</td><td>{a.required ? "Yes" : "No"}</td><td className="text-xs font-mono">{a.validation_rule}</td><td><span className={`px-2 py-1 rounded text-xs ${privColors[a.privacy_classification] || ""}`}>{a.privacy_classification}</span></td></tr>))}</tbody></table></div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">{t("idAttributeSchema.perAttrMasking")}</h2><div className="space-y-1">{form.per_attribute_masking.map((m: { attribute: string; masked: boolean }, i: number) => (<div key={i} className="flex items-center justify-between border-b py-1"><span className="text-sm">{m.attribute}</span><span className={`px-2 py-1 rounded text-xs ${m.masked ? "bg-blue-100 text-blue-700" : "bg-gray-100 text-gray-500"}`}>{m.masked ? "Masked" : "Visible"}</span></div>))}</div></div>
      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}

"use client";
import { useEffect, useState } from "react";
import { useSamlSpConfig, SamlSpConfig, AttributeConsumingService } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

export default function SamlSpConfigPage() {
  const t = useTranslations();
  const { config, loading, error, fetchConfig, updateConfig } = useSamlSpConfig();
  const [form, setForm] = useState<SamlSpConfig | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };

  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">SAML SP Configuration</h1>
      <p className="text-gray-600">Configure SAML Service Provider settings.</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">SP Endpoints</h2>
        <div>
          <label className="block text-sm font-medium mb-1">Entity ID</label>
          <input aria-label="form" type="text" value={form.entity_id}
            onChange={(e) => setForm({ ...form, entity_id: e.target.value })}
            className="border rounded px-3 py-2 w-full" />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">ACS URL</label>
          <input aria-label="form" type="text" value={form.acs_url}
            onChange={(e) => setForm({ ...form, acs_url: e.target.value })}
            className="border rounded px-3 py-2 w-full" />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">SLO URL</label>
          <input aria-label="form" type="text" value={form.slo_url}
            onChange={(e) => setForm({ ...form, slo_url: e.target.value })}
            className="border rounded px-3 py-2 w-full" />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">Metadata URL</label>
          <input aria-label="form" type="text" value={form.metadata_url}
            onChange={(e) => setForm({ ...form, metadata_url: e.target.value })}
            className="border rounded px-3 py-2 w-full" />
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Security</h2>
        <div>
          <label className="block text-sm font-medium mb-1">Signature Algorithm</label>
          <select aria-label="form" value={form.signature_algorithm}
            onChange={(e) => setForm({ ...form, signature_algorithm: e.target.value as SamlSpConfig["signature_algorithm"] })}
            className="border rounded px-3 py-2">
            <option value="RSA-SHA256">RSA-SHA256</option>
            <option value="RSA-SHA1">RSA-SHA1</option>
            <option value="ECDSA-SHA256">ECDSA-SHA256</option>
          </select>
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">Name ID Format</label>
          <select aria-label="form" value={form.name_id_format}
            onChange={(e) => setForm({ ...form, name_id_format: e.target.value as SamlSpConfig["name_id_format"] })}
            className="border rounded px-3 py-2">
            <option value="unspecified">Unspecified</option>
            <option value="emailAddress">Email Address</option>
            <option value="persistent">Persistent</option>
            <option value="transient">Transient</option>
          </select>
        </div>
        <div className="flex items-center gap-3">
          <input aria-label="Toggle" type="checkbox" checked={form.want_signed}
            onChange={(e) => setForm({ ...form, want_signed: e.target.checked })}
            className="w-4 h-4" />
          <label>Require Signed Responses</label>
        </div>
        <div className="flex items-center gap-3">
          <input aria-label="Toggle" type="checkbox" checked={form.want_encrypted}
            onChange={(e) => setForm({ ...form, want_encrypted: e.target.checked })}
            className="w-4 h-4" />
          <label>Require Encrypted Assertions</label>
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Attribute Consuming Services</h2>
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b text-left">
              <th scope="col" className="py-2">Index</th>
              <th scope="col">Service Name</th>
              <th scope="col">Requested Attributes</th>
            </tr>
          </thead>
          <tbody>
            {form.attribute_consuming_service.map((acs: AttributeConsumingService, i: number) => (
              <tr key={i} className="border-b">
                <td className="py-2">{acs.index}</td>
                <td>{acs.service_name}</td>
                <td>{acs.requested_attributes.join(", ")}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <button aria-label="action" onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}

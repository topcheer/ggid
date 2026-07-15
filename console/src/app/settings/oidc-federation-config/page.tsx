"use client";
import { useEffect, useState } from "react";
import { useOidcFederationConfig, OidcFederationConfig } from "@ggid/sdk-react";

interface LocalTrustAnchor {
  issuer: string;
  jwks_uri: string;
  trust_mark: string;
}

interface LocalFederatedProvider {
  name: string;
  issuer: string;
  status: "active" | "inactive";
}

interface LocalEntityCategoryRequirement {
  category: string;
  required_claims: string[];
}

interface LocalOidcFederationConfig extends OidcFederationConfig {
  trust_anchors: LocalTrustAnchor[];
  federated_providers: LocalFederatedProvider[];
  entity_category_requirements: LocalEntityCategoryRequirement[];
}

export default function OidcFederationConfigPage() {
  const { config, loading, error, fetchConfig, updateConfig } = useOidcFederationConfig();
  const [form, setForm] = useState<LocalOidcFederationConfig | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config as unknown as LocalOidcFederationConfig); }, [config]);

  const handleSave = async () => {
    if (!form) return;
    setSaving(true);
    await updateConfig(form as unknown as Parameters<typeof updateConfig>[0]);
    setSaving(false);
  };

  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">OIDC Federation Configuration</h1>
      <p className="text-gray-600">Configure OpenID Connect Federation (RFC 8411 / Federation Entities).</p>

      {/* Auto Discovery */}
      <div className="flex items-center gap-3 bg-white rounded-lg p-4 shadow">
        <input
          type="checkbox"
          checked={form.auto_discovery}
          onChange={(e) => setForm({ ...form, auto_discovery: e.target.checked })}
          className="w-5 h-5"
        />
        <label className="font-medium">Auto Discovery</label>
      </div>

      {/* Trust Resolution Policy */}
      <div className="bg-white rounded-lg p-6 shadow">
        <label className="block text-sm font-medium mb-2">Trust Resolution Policy</label>
        <select
          value={form.trust_resolution_policy}
          onChange={(e) => setForm({ ...form, trust_resolution_policy: e.target.value as LocalOidcFederationConfig["trust_resolution_policy"] })}
          className="border rounded px-3 py-2"
        >
          <option value="tree">Tree</option>
          <option value="path">Path</option>
          <option value="graph">Graph</option>
        </select>
      </div>

      {/* Trust Anchors */}
      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Trust Anchors</h2>
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b text-left">
              <th className="py-2">Issuer</th>
              <th>JWKS URI</th>
              <th>Trust Mark</th>
            </tr>
          </thead>
          <tbody>
            {form.trust_anchors.map((a, i) => (
              <tr key={i} className="border-b">
                <td className="py-2">{a.issuer}</td>
                <td className="break-all">{a.jwks_uri}</td>
                <td className="break-all">{a.trust_mark}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Federated Providers */}
      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Federated Providers</h2>
        <div className="space-y-2">
          {form.federated_providers.map((p, i) => (
            <div key={i} className="flex items-center justify-between border-b py-2">
              <div>
                <span className="font-medium">{p.name}</span>
                <span className="ml-2 text-gray-500">{p.issuer}</span>
              </div>
              <span className={`px-2 py-1 rounded text-xs ${p.status === "active" ? "bg-green-100 text-green-700" : "bg-gray-100 text-gray-500"}`}>
                {p.status}
              </span>
            </div>
          ))}
        </div>
      </div>

      {/* Entity Category Requirements */}
      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Entity Category Requirements</h2>
        <div className="space-y-3">
          {form.entity_category_requirements.map((ecr, i) => (
            <div key={i} className="border-b pb-2">
              <div className="font-medium">{ecr.category}</div>
              <div className="text-sm text-gray-500">Required Claims: {ecr.required_claims.join(", ")}</div>
            </div>
          ))}
        </div>
      </div>

      <button onClick={handleSave} disabled={saving} aria-label="Save OIDC federation config" className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}

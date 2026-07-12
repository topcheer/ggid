"use client";
import { useEffect, useState } from "react";
import { useOAuthScopeTieringConfig, OAuthScopeTieringConfig, TierDefinition, ScopePackage, ScopeInheritanceRule } from "@ggid/sdk-react";

export default function OAuthScopeTieringConfigPage() {
  const { config, loading, error, fetchConfig, updateConfig } = useOAuthScopeTieringConfig();
  const [form, setForm] = useState<OAuthScopeTieringConfig | null>(null);
  const [saving, setSaving] = useState(false);
  useEffect(() => { fetchConfig(); }, [fetchConfig]); useEffect(() => { if (config) setForm(config); }, [config]);
  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };
  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">OAuth Scope Tiering Configuration</h1>
      <p className="text-gray-600">Configure scope tiers, packages, and inheritance for least-privilege access.</p>
      <div className="bg-white rounded-lg p-6 shadow space-y-4"><h2 className="text-lg font-semibold">General</h2><div className="flex items-center gap-3"><input type="checkbox" checked={form.least_privilege_defaults} onChange={(e) => setForm({ ...form, least_privilege_defaults: e.target.checked })} className="w-4 h-4" /><label>Least Privilege Defaults</label></div><div className="flex items-center gap-3"><input type="checkbox" checked={form.migration_from_flat_scopes} onChange={(e) => setForm({ ...form, migration_from_flat_scopes: e.target.checked })} className="w-4 h-4" /><label>Migration from Flat Scopes</label></div></div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Tier Definitions</h2><table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Tier</th><th>Consent Policy</th></tr></thead><tbody>{form.tier_definitions.map((t: TierDefinition, i: number) => (<tr key={i} className="border-b"><td className="py-2 font-medium">{t.tier}</td><td><span className="px-2 py-1 bg-blue-100 text-blue-700 rounded text-xs">{t.consent_policy}</span></td></tr>))}</tbody></table></div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Scope Packages</h2><div className="space-y-2">{form.scope_packages.map((p: ScopePackage, i: number) => (<div key={i} className="border-b pb-2"><div className="font-medium">{p.name}</div><div className="text-sm text-gray-500 font-mono">{p.scopes.join(", ")}</div></div>))}</div></div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Scope Inheritance Rules</h2><table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Parent Scope</th><th>Child Scopes</th></tr></thead><tbody>{form.scope_inheritance_rules.map((r: ScopeInheritanceRule, i: number) => (<tr key={i} className="border-b"><td className="py-2 font-mono">{r.parent_scope}</td><td className="text-xs font-mono">{r.child_scopes.join(", ")}</td></tr>))}</tbody></table></div>
      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}

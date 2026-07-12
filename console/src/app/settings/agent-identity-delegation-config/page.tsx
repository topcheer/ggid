"use client";
import { useEffect, useState } from "react";
import { useAgentIdentityDelegationConfig, AgentIdentityDelegationConfig, ScopeNarrowingRule, PerAgentTrust } from "@ggid/sdk-react";

export default function AgentIdentityDelegationConfigPage() {
  const { config, loading, error, fetchConfig, updateConfig } = useAgentIdentityDelegationConfig();
  const [form, setForm] = useState<AgentIdentityDelegationConfig | null>(null);
  const [saving, setSaving] = useState(false);
  useEffect(() => { fetchConfig(); }, [fetchConfig]); useEffect(() => { if (config) setForm(config); }, [config]);
  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };
  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;
  const trustColors: Record<string, string> = { low: "bg-gray-100 text-gray-600", medium: "bg-yellow-100 text-yellow-700", high: "bg-green-100 text-green-700" };

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">Agent Identity Delegation Configuration</h1>
      <p className="text-gray-600">Configure AI agent delegation depth, scope narrowing, and trust levels.</p>
      <div className="bg-white rounded-lg p-6 shadow space-y-4"><h2 className="text-lg font-semibold">Delegation Policy</h2><div><label className="block text-sm font-medium mb-1">Max Delegation Depth</label><input type="number" value={form.max_delegation_depth} onChange={(e) => setForm({ ...form, max_delegation_depth: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-32" /></div><div><label className="block text-sm font-medium mb-1">Token Exchange Policy</label><select value={form.token_exchange_policy} onChange={(e) => setForm({ ...form, token_exchange_policy: e.target.value as AgentIdentityDelegationConfig["token_exchange_policy"] })} className="border rounded px-3 py-2"><option value="strict">Strict</option><option value="permissive">Permissive</option></select></div><div className="flex items-center gap-3"><input type="checkbox" checked={form.revocation_propagation} onChange={(e) => setForm({ ...form, revocation_propagation: e.target.checked })} className="w-4 h-4" /><label>Revocation Propagation</label></div></div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Scope Narrowing Rules</h2><table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Parent Scope</th><th>Allowed Child Scopes</th></tr></thead><tbody>{form.scope_narrowing_rules.map((r: ScopeNarrowingRule, i: number) => (<tr key={i} className="border-b"><td className="py-2 font-mono">{r.parent_scope}</td><td className="text-xs">{r.allowed_child_scopes.join(", ")}</td></tr>))}</tbody></table></div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Per-Agent Trust Levels</h2><table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Agent</th><th>Trust Level</th></tr></thead><tbody>{form.per_agent_trust_level.map((a: PerAgentTrust, i: number) => (<tr key={i} className="border-b"><td className="py-2"><span className="font-medium">{a.agent_name}</span><div className="text-xs text-gray-400">{a.agent_id}</div></td><td><span className={`px-2 py-1 rounded text-xs ${trustColors[a.trust_level] || ""}`}>{a.trust_level}</span></td></tr>))}</tbody></table></div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Delegation Chain Preview</h2><pre className="bg-gray-50 rounded p-4 text-xs overflow-x-auto whitespace-pre-wrap">{form.delegation_chain_preview}</pre></div>
      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}

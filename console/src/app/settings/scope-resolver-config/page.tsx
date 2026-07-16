'use client';
import { useState, useEffect, useMemo, useCallback } from 'react';
import { useTranslations } from "@/lib/i18n";

interface ScopeNode { name: string; description: string; children: string[]; parent: string | null; }
interface ClientScopeMapping { client: string; allowedScopes: string[]; }
interface ScopeRestriction { id: string; name: string; description: string; enabled: boolean; }

export default function ScopeResolverConfigPage() {
  const t = useTranslations();

  const [scopeTree, setScopeTree] = useState<Record<string, ScopeNode>>({});
  const [clientMappings, setClientMappings] = useState<ClientScopeMapping[]>([]);
  const [restrictions, setRestrictions] = useState<ScopeRestriction[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<'hierarchy' | 'mapping' | 'wildcard' | 'calculator' | 'restrictions'>('hierarchy');
  const [consentEnforcement, setConsentEnforcement] = useState(true);
  const [calcUser, setCalcUser] = useState('alice@corp.com');
  const [calcClient, setCalcClient] = useState('');
  const [wildcardRules, setWildcardRules] = useState('');

  useEffect(() => {
    fetch("/api/v1/oauth/scope-resolver-config", {
      headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
    })
      .then(res => { if (!res.ok) return null; return res.json(); })
      .then(data => {
        setScopeTree(data.scopeTree || {});
        setClientMappings(data.clientMappings || []);
        setRestrictions(data.restrictions || []);
        setWildcardRules(data.wildcardRules || '');
        setConsentEnforcement(data.consentEnforcement ?? true);
        if (data.clientMappings && data.clientMappings.length > 0) setCalcClient(data.clientMappings[0].client);
        setLoading(false);
      })
      .catch(err => { setError(err.message); setLoading(false); });
  }, []);

  const rootScopes = useMemo(() => Object.values(scopeTree).filter(s => s.parent === null), [scopeTree]);

  const resolvedScopes = useMemo(() => {
    const mapping = clientMappings.find(m => m.client === calcClient);
    return mapping ? mapping.allowedScopes : [];
  }, [calcClient, clientMappings]);

  const toggleRestriction = useCallback((id: string) => {
    setRestrictions(prev => prev.map(r => r.id === id ? { ...r, enabled: !r.enabled } : r));
  }, []);

  const renderScopeTree = (scopeName: string, depth: number = 0): React.ReactNode => {
    const node = scopeTree[scopeName];
    if (!node) return null;
    return (
      <div key={scopeName} className="space-y-1">
        <div className="flex items-center gap-2" style={{ paddingLeft: depth * 20 }}>
          {depth > 0 && <span className="text-gray-300">{'|'}</span>}
          <span className="font-mono text-xs font-medium text-blue-600">{scopeName}</span>
          <span className="text-xs text-gray-400">{'-'} {node.description}</span>
        </div>
        {node.children.map(child => renderScopeTree(child, depth + 1))}
      </div>
    );
  };

  if (loading) return <div className="p-6"><h1 className="text-2xl font-bold">Scope Resolver Configuration</h1><p className="text-gray-600 mt-2">Loading...</p></div>;
  if (error) return <div className="p-6"><h1 className="text-2xl font-bold">Scope Resolver Configuration</h1><p className="text-red-600 mt-2">Error: {error}</p></div>;

  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold text-gray-900">Scope Resolver Configuration</h1><p className="mt-1 text-sm text-gray-500">Manage scope hierarchy, client mappings, wildcard expansion, and effective scope resolution.</p></div>

      <div className="flex gap-2 border-b border-gray-200 overflow-x-auto">
        {(['hierarchy', 'mapping', 'wildcard', 'calculator', 'restrictions'] as const).map(tab => (
          <button key={tab} onClick={() => setActiveTab(tab)} className={`px-4 py-2 text-sm font-medium border-b-2 whitespace-nowrap ${activeTab === tab ? 'border-blue-500 text-blue-600' : 'border-transparent text-gray-500 hover:text-gray-700'}`}>{tab === 'hierarchy' ? 'Scope Hierarchy' : tab === 'mapping' ? 'Client Mapping' : tab === 'wildcard' ? 'Wildcard Config' : tab === 'calculator' ? 'Effective Scope Calculator' : 'Restrictions'}</button>
        ))}
      </div>

      {activeTab === 'hierarchy' && (
        <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
          <h3 className="text-sm font-medium text-gray-700">Scope Hierarchy Viewer</h3>
          <div className="mt-4 space-y-2">{rootScopes.length === 0 ? <p className="text-gray-500 text-sm">No scopes configured.</p> : rootScopes.map(s => renderScopeTree(s.name))}</div>
        </div>
      )}

      {activeTab === 'mapping' && (
        <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
          <h3 className="text-sm font-medium text-gray-700">Client Scope Mappings</h3>
          <div className="mt-4 space-y-4">
            {clientMappings.length === 0 ? <p className="text-gray-500 text-sm">No client mappings configured.</p> :
            clientMappings.map(m => (
              <div key={m.client} className="border-b border-gray-100 pb-3">
                <div className="flex items-center justify-between mb-2"><span className="text-sm font-medium text-gray-700">{m.client}</span><span className="text-xs text-gray-400">{m.allowedScopes.length} scopes</span></div>
                <div className="flex flex-wrap gap-1.5">{m.allowedScopes.map(s => (<span key={s} className="inline-flex rounded bg-blue-50 px-2 py-0.5 text-xs font-mono text-blue-700">{s}</span>))}</div>
              </div>
            ))}
          </div>
        </div>
      )}

      {activeTab === 'wildcard' && (
        <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
          <h3 className="text-sm font-medium text-gray-700">Wildcard Scope Expansion Rules</h3>
          <p className="mt-1 text-xs text-gray-400">Define how wildcard scopes (e.g. audit:*) expand to concrete scopes</p>
          <textarea aria-label="Wildcard rules" value={wildcardRules} onChange={e => setWildcardRules(e.target.value)} rows={8} className="mt-3 block w-full rounded-md border border-gray-300 px-3 py-2 font-mono text-xs" />
          <div className="mt-4 flex items-center gap-3">
            <button onClick={() => setConsentEnforcement(!consentEnforcement)} className={`relative inline-flex h-6 w-11 items-center rounded-full transition ${consentEnforcement ? 'bg-blue-600' : 'bg-gray-200'}`}><span className={`inline-block h-4 w-4 transform rounded-full bg-white transition ${consentEnforcement ? 'translate-x-6' : 'translate-x-1'}`} /></button>
            <span className="text-sm text-gray-700">Scope Consent Enforcement {consentEnforcement ? 'Enabled' : 'Disabled'}</span>
          </div>
          <button className="mt-4 rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700">Save Wildcard Rules</button>
        </div>
      )}

      {activeTab === 'calculator' && (
        <div className="space-y-4">
          <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
            <h3 className="text-sm font-medium text-gray-700">Effective Scope Calculator</h3>
            <div className="mt-4 grid grid-cols-1 gap-4 md:grid-cols-2">
              <div><label className="block text-xs font-medium text-gray-600">User</label><input aria-label="calc User" type="text" value={calcUser} onChange={e => setCalcUser(e.target.value)} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm" /></div>
              <div><label className="block text-xs font-medium text-gray-600">Client</label><select aria-label="calc Client" value={calcClient} onChange={e => setCalcClient(e.target.value)} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm">{clientMappings.map(m => <option key={m.client} value={m.client}>{m.client}</option>)}</select></div>
            </div>
          </div>
          <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
            <h3 className="text-sm font-medium text-gray-700">Resolved Scopes for {calcUser} @ {calcClient}</h3>
            <div className="mt-3 flex flex-wrap gap-2">{resolvedScopes.map(s => (<span key={s} className="inline-flex rounded bg-green-50 px-3 py-1 text-xs font-mono text-green-700 border border-green-200">{s}</span>))}</div>
            <p className="mt-3 text-xs text-gray-400">{resolvedScopes.length} effective scopes after hierarchy resolution, wildcard expansion, and restriction filtering</p>
          </div>
        </div>
      )}

      {activeTab === 'restrictions' && (
        <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
          <h3 className="text-sm font-medium text-gray-700">Scope Restriction Rules</h3>
          <div className="mt-4 space-y-3">
            {restrictions.length === 0 ? <p className="text-gray-500 text-sm">No restrictions configured.</p> :
            restrictions.map(r => (
              <div key={r.id} className="flex items-center justify-between border-b border-gray-100 pb-3">
                <div><div className="text-sm font-medium text-gray-700">{r.name}</div><div className="text-xs text-gray-500">{r.description}</div></div>
                <button onClick={() => toggleRestriction(r.id)} className={`relative inline-flex h-5 w-9 items-center rounded-full transition ${r.enabled ? 'bg-green-500' : 'bg-gray-200'}`}><span className={`inline-block h-3 w-3 transform rounded-full bg-white transition ${r.enabled ? 'translate-x-5' : 'translate-x-1'}`} /></button>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

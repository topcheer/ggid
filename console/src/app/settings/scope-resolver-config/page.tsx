'use client';

import { useState, useMemo, useCallback } from 'react';

interface ScopeNode {
  name: string;
  description: string;
  children: string[];
  parent: string | null;
}

interface ClientScopeMapping {
  client: string;
  allowedScopes: string[];
}

interface ScopeRestriction {
  id: string;
  name: string;
  description: string;
  enabled: boolean;
}

const SCOPE_TREE: Record<string, ScopeNode> = {
  openid: { name: 'openid', description: 'OpenID Connect scope', children: ['profile', 'email'], parent: null },
  profile: { name: 'profile', description: 'User profile information', children: [], parent: 'openid' },
  email: { name: 'email', description: 'Email address access', children: [], parent: 'openid' },
  audit: { name: 'audit', description: 'Audit log access', children: ['audit:read', 'audit:export'], parent: null },
  'audit:read': { name: 'audit:read', description: 'Read audit logs', children: [], parent: 'audit' },
  'audit:export': { name: 'audit:export', description: 'Export audit logs', children: [], parent: 'audit' },
  users: { name: 'users', description: 'User management', children: ['users:read', 'users:write'], parent: null },
  'users:read': { name: 'users:read', description: 'Read user profiles', children: [], parent: 'users' },
  'users:write': { name: 'users:write', description: 'Modify user profiles', children: [], parent: 'users' },
  roles: { name: 'roles', description: 'Role management', children: ['roles:read', 'roles:manage'], parent: null },
  'roles:read': { name: 'roles:read', description: 'Read roles', children: [], parent: 'roles' },
  'roles:manage': { name: 'roles:manage', description: 'Create/modify/delete roles', children: [], parent: 'roles' },
  deploy: { name: 'deploy', description: 'Deployment operations', children: ['deploy:write'], parent: null },
  'deploy:write': { name: 'deploy:write', description: 'Trigger deployments', children: [], parent: 'deploy' },
  admin: { name: 'admin', description: 'Admin operations', children: ['admin:all', 'admin:config'], parent: null },
  'admin:all': { name: 'admin:all', description: 'Full admin access', children: [], parent: 'admin' },
  'admin:config': { name: 'admin:config', description: 'System configuration', children: [], parent: 'admin' },
};

const CLIENT_MAPPINGS: ClientScopeMapping[] = [
  { client: 'web-console', allowedScopes: ['openid', 'profile', 'email', 'audit:read', 'users:read', 'users:write', 'roles:read', 'roles:manage'] },
  { client: 'mobile-app', allowedScopes: ['openid', 'profile', 'email'] },
  { client: 'analytics-tool', allowedScopes: ['openid', 'audit:read', 'audit:export'] },
  { client: 'ci-cd-tool', allowedScopes: ['openid', 'deploy:write'] },
  { client: 'admin-panel', allowedScopes: ['openid', 'profile', 'admin:all', 'admin:config'] },
];

const RESTRICTIONS: ScopeRestriction[] = [
  { id: 'r1', name: 'Wildcard Expansion Limit', description: 'Wildcard scopes (*) only expand to direct children, not nested', enabled: true },
  { id: 'r2', name: 'Consent Required for Sensitive Scopes', description: 'Scopes marked as sensitive require explicit user consent', enabled: true },
  { id: 'r3', name: 'Deny Cross-Domain Scopes', description: 'Prevent granting scopes from different domain trees', enabled: false },
  { id: 'r4', name: 'Max Scopes per Token', description: 'Limit maximum scopes per access token to 20', enabled: true },
];

export default function ScopeResolverConfigPage() {
  const [activeTab, setActiveTab] = useState<'hierarchy' | 'mapping' | 'wildcard' | 'calculator' | 'restrictions'>('hierarchy');
  const [consentEnforcement, setConsentEnforcement] = useState(true);
  const [restrictions, setRestrictions] = useState<ScopeRestriction[]>(RESTRICTIONS);
  const [calcUser, setCalcUser] = useState('alice@corp.com');
  const [calcClient, setCalcClient] = useState('web-console');
  const [wildcardRules, setWildcardRules] = useState('audit:* -> audit:read, audit:export\nusers:* -> users:read, users:write\nroles:* -> roles:read, roles:manage');

  const rootScopes = useMemo(() => Object.values(SCOPE_TREE).filter(s => s.parent === null), []);

  const resolvedScopes = useMemo(() => {
    const mapping = CLIENT_MAPPINGS.find(m => m.client === calcClient);
    return mapping ? mapping.allowedScopes : [];
  }, [calcClient]);

  const toggleRestriction = useCallback((id: string) => {
    setRestrictions(restrictions.map(r => r.id === id ? { ...r, enabled: !r.enabled } : r));
  }, [restrictions]);

  const renderScopeTree = (scopeName: string, depth: number = 0): React.ReactNode => {
    const node = SCOPE_TREE[scopeName];
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

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Scope Resolver Configuration</h1>
        <p className="mt-1 text-sm text-gray-500">Manage scope hierarchy, client mappings, wildcard expansion, and effective scope resolution.</p>
      </div>

      <div className="flex gap-2 border-b border-gray-200 overflow-x-auto">
        {(['hierarchy', 'mapping', 'wildcard', 'calculator', 'restrictions'] as const).map(tab => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`px-4 py-2 text-sm font-medium border-b-2 whitespace-nowrap ${
              activeTab === tab ? 'border-blue-500 text-blue-600' : 'border-transparent text-gray-500 hover:text-gray-700'
            }`}
          >
            {tab === 'hierarchy' ? 'Scope Hierarchy' : tab === 'mapping' ? 'Client Mapping' : tab === 'wildcard' ? 'Wildcard Config' : tab === 'calculator' ? 'Effective Scope Calculator' : 'Restrictions'}
          </button>
        ))}
      </div>

      {activeTab === 'hierarchy' && (
        <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
          <h3 className="text-sm font-medium text-gray-700">Scope Hierarchy Viewer</h3>
          <div className="mt-4 space-y-2">
            {rootScopes.map(s => renderScopeTree(s.name))}
          </div>
        </div>
      )}

      {activeTab === 'mapping' && (
        <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
          <h3 className="text-sm font-medium text-gray-700">Client Scope Mappings</h3>
          <div className="mt-4 space-y-4">
            {CLIENT_MAPPINGS.map(m => (
              <div key={m.client} className="border-b border-gray-100 pb-3">
                <div className="flex items-center justify-between mb-2">
                  <span className="text-sm font-medium text-gray-700">{m.client}</span>
                  <span className="text-xs text-gray-400">{m.allowedScopes.length} scopes</span>
                </div>
                <div className="flex flex-wrap gap-1.5">
                  {m.allowedScopes.map(s => (
                    <span key={s} className="inline-flex rounded bg-blue-50 px-2 py-0.5 text-xs font-mono text-blue-700">{s}</span>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {activeTab === 'wildcard' && (
        <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
          <h3 className="text-sm font-medium text-gray-700">Wildcard Scope Expansion Rules</h3>
          <p className="mt-1 text-xs text-gray-400">Define how wildcard scopes (e.g. audit:*) expand to concrete scopes</p>
          <textarea
            value={wildcardRules}
            onChange={e => setWildcardRules(e.target.value)}
            rows={8}
            className="mt-3 block w-full rounded-md border border-gray-300 px-3 py-2 font-mono text-xs"
          />
          <div className="mt-4 flex items-center gap-3">
            <button onClick={() => setConsentEnforcement(!consentEnforcement)} className={`relative inline-flex h-6 w-11 items-center rounded-full transition ${consentEnforcement ? 'bg-blue-600' : 'bg-gray-200'}`}>
              <span className={`inline-block h-4 w-4 transform rounded-full bg-white transition ${consentEnforcement ? 'translate-x-6' : 'translate-x-1'}`} />
            </button>
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
              <div>
                <label className="block text-xs font-medium text-gray-600">User</label>
                <select value={calcUser} onChange={e => setCalcUser(e.target.value)} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm">
                  <option value="alice@corp.com">alice@corp.com</option>
                  <option value="bob@corp.com">bob@corp.com</option>
                  <option value="charlie@corp.com">charlie@corp.com</option>
                  <option value="admin@corp.com">admin@corp.com</option>
                </select>
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-600">Client</label>
                <select value={calcClient} onChange={e => setCalcClient(e.target.value)} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm">
                  {CLIENT_MAPPINGS.map(m => <option key={m.client} value={m.client}>{m.client}</option>)}
                </select>
              </div>
            </div>
          </div>
          <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
            <h3 className="text-sm font-medium text-gray-700">Resolved Scopes for {calcUser} @ {calcClient}</h3>
            <div className="mt-3 flex flex-wrap gap-2">
              {resolvedScopes.map(s => (
                <span key={s} className="inline-flex rounded bg-green-50 px-3 py-1 text-xs font-mono text-green-700 border border-green-200">{s}</span>
              ))}
            </div>
            <p className="mt-3 text-xs text-gray-400">{resolvedScopes.length} effective scopes after hierarchy resolution, wildcard expansion, and restriction filtering</p>
          </div>
        </div>
      )}

      {activeTab === 'restrictions' && (
        <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
          <h3 className="text-sm font-medium text-gray-700">Scope Restriction Rules</h3>
          <div className="mt-4 space-y-3">
            {restrictions.map(r => (
              <div key={r.id} className="flex items-center justify-between border-b border-gray-100 pb-3">
                <div>
                  <span className="text-sm font-medium text-gray-700">{r.name}</span>
                  <p className="text-xs text-gray-400">{r.description}</p>
                </div>
                <button onClick={() => toggleRestriction(r.id)} className={`relative inline-flex h-5 w-9 items-center rounded-full transition ${r.enabled ? 'bg-green-500' : 'bg-gray-200'}`}>
                  <span className={`inline-block h-3 w-3 transform rounded-full bg-white transition ${r.enabled ? 'translate-x-5' : 'translate-x-1'}`} />
                </button>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
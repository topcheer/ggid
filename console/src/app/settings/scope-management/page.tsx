'use client';
import { useState } from 'react';

interface Scope {
  id: string;
  name: string;
  description: string;
  isSystem: boolean;
  defaultForClients: boolean;
  parent: string;
}

export default function ScopeManagementPage() {
  const [scopes, setScopes] = useState<Scope[]>([
    { id: 'sc1', name: 'openid', description: 'OpenID Connect scope', isSystem: true, defaultForClients: true, parent: '-' },
    { id: 'sc2', name: 'profile', description: 'User profile claims', isSystem: true, defaultForClients: true, parent: 'openid' },
    { id: 'sc3', name: 'email', description: 'Email address claim', isSystem: true, defaultForClients: true, parent: 'openid' },
    { id: 'sc4', name: 'read:users', description: 'Read user profiles', isSystem: false, defaultForClients: false, parent: '-' },
    { id: 'sc5', name: 'write:users', description: 'Create/update users', isSystem: false, defaultForClients: false, parent: 'read:users' },
    { id: 'sc6', name: 'admin:all', description: 'Full admin access', isSystem: false, defaultForClients: false, parent: '-' },
    { id: 'sc7', name: 'read:audit', description: 'Read audit logs', isSystem: false, defaultForClients: false, parent: '-' },
  ]);

  const [showForm, setShowForm] = useState(false);
  const [newScope, setNewScope] = useState({ name: '', description: '', isSystem: false, defaultForClients: false, parent: '-' });
  const [clients] = useState([
    { name: 'web-app', scopes: ['openid', 'profile', 'read:users'] },
    { name: 'mobile-app', scopes: ['openid', 'profile', 'email'] },
    { name: 'admin-cli', scopes: ['openid', 'admin:all', 'read:audit'] },
  ]);

  const addScope = () => {
    setScopes(prev => [...prev, { id: `sc${prev.length + 1}`, ...newScope }]);
    setShowForm(false);
    setNewScope({ name: '', description: '', isSystem: false, defaultForClients: false, parent: '-' });
  };

  const deleteScope = (id: string) => {
    setScopes(prev => prev.filter(s => s.id !== id && !s.isSystem));
  };

  const allScopeNames = scopes.map(s => s.name);

  const renderTree = (parent: string, depth: number): React.ReactNode => {
    const children = scopes.filter(s => s.parent === parent);
    if (children.length === 0) return null;
    return (
      <ul className="ml-4 border-l border-gray-200 pl-3 space-y-1">
        {children.map(s => (
          <li key={s.id}>
            <div className="flex items-center gap-2">
              <span className={`px-2 py-0.5 rounded text-xs font-mono ${s.isSystem ? 'bg-purple-100 text-purple-700' : 'bg-blue-100 text-blue-700'}`}>{s.name}</span>
              {s.defaultForClients && <span className="text-xs text-gray-400">(default)</span>}
              <span className="text-xs text-gray-500">{s.description}</span>
            </div>
            {renderTree(s.name, depth + 1)}
          </li>
        ))}
      </ul>
    );
  };

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Scope Management</h1>
          <p className="text-gray-600">Manage OAuth/OIDC scopes, hierarchy, and client assignments.</p>
        </div>
        <button onClick={() => setShowForm(!showForm)} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">{showForm ? 'Cancel' : 'Create Scope'}</button>
      </div>

      {showForm && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Create Scope</h2>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="text-sm font-medium">Name</label>
              <input type="text" placeholder="read:reports" value={newScope.name} onChange={e => setNewScope(prev => ({ ...prev, name: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1 font-mono" />
            </div>
            <div>
              <label className="text-sm font-medium">Parent Scope</label>
              <select value={newScope.parent} onChange={e => setNewScope(prev => ({ ...prev, parent: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1">
                <option value="-">(none)</option>
                {allScopeNames.map(n => <option key={n} value={n}>{n}</option>)}
              </select>
            </div>
          </div>
          <div>
            <label className="text-sm font-medium">Description</label>
            <input type="text" value={newScope.description} onChange={e => setNewScope(prev => ({ ...prev, description: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" />
          </div>
          <div className="flex gap-4">
            <label className="flex items-center gap-2 text-sm">
              <input type="checkbox" checked={newScope.defaultForClients} onChange={e => setNewScope(prev => ({ ...prev, defaultForClients: e.target.checked }))} className="rounded" />
              Default for new clients
            </label>
          </div>
          <button onClick={addScope} disabled={!newScope.name} className="px-4 py-2 bg-blue-600 text-white rounded text-sm disabled:opacity-50">Add Scope</button>
        </section>
      )}

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Scope List</h2>
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th className="p-3">Name</th>
              <th className="p-3">Description</th>
              <th className="p-3">Type</th>
              <th className="p-3">Default</th>
              <th className="p-3">Actions</th>
            </tr>
          </thead>
          <tbody>
            {scopes.map(s => (
              <tr key={s.id} className="border-b">
                <td className="p-3 font-mono text-xs">{s.name}</td>
                <td className="p-3 text-gray-600">{s.description}</td>
                <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs ${s.isSystem ? 'bg-purple-100 text-purple-700' : 'bg-blue-100 text-blue-700'}`}>{s.isSystem ? 'system' : 'custom'}</span></td>
                <td className="p-3">{s.defaultForClients ? <span className="text-green-600 text-xs">yes</span> : <span className="text-gray-400 text-xs">no</span>}</td>
                <td className="p-3">{!s.isSystem && <button onClick={() => deleteScope(s.id)} className="text-red-600 text-xs hover:underline">Delete</button>}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Scope Hierarchy</h2>
        {renderTree('-', 0)}
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Scope-to-Client Assignment Matrix</h2>
        <table className="w-full text-xs">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th className="p-2">Client</th>
              {scopes.map(s => <th key={s.id} className="p-2">{s.name}</th>)}
            </tr>
          </thead>
          <tbody>
            {clients.map(c => (
              <tr key={c.name} className="border-b">
                <td className="p-2 font-medium">{c.name}</td>
                {scopes.map(s => (
                  <td key={s.id} className="p-2 text-center">
                    {c.scopes.includes(s.name) ? <span className="text-green-600">✓</span> : <span className="text-gray-300">-</span>}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </div>
  );
}
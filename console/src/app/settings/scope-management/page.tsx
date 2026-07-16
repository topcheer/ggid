'use client';
import { useTranslations } from "@/lib/i18n";
import { useState, useEffect } from 'react';
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface Scope {
  id: string;
  name: string;
  description: string;
  isSystem: boolean;
  defaultForClients: boolean;
  parent: string;
}

export default function ScopeManagementPage() {
  const t = useTranslations();
  const [scopes, setScopes] = useState<Scope[]>([]);
  const [clients, setClients] = useState<{ name: string; scopes: string[] }[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [newScope, setNewScope] = useState({ name: '', description: '', isSystem: false, defaultForClients: false, parent: '-' });

  useEffect(() => {
    fetch("/api/v1/oauth/scopes", {
      headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
    })
      .then(res => { if (!res.ok) return null; return res.json(); })
      .then(data => {
        setScopes(data.scopes || data.items || []);
        setClients(data.clients || []);
        setLoading(false);
      })
      .catch(err => { setError(err.message); setLoading(false); });
  }, []);

  const addScope = () => {
    fetch("/api/v1/oauth/scopes", {
      method: "POST",
      headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
      body: JSON.stringify(newScope),
    })
      .then(res => { if (!res.ok) return null; return res.json(); })
      .then(data => { setScopes(prev => [...prev, data]); setShowForm(false); setNewScope({ name: '', description: '', isSystem: false, defaultForClients: false, parent: '-' }); })
      .catch(err => setError(err.message));
  };

  const deleteScope = (id: string) => {
    fetch(`/api/v1/oauth/scopes/${id}`, { method: "DELETE", headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } })
      .then(res => { if (!res.ok) return null; setScopes(prev => prev.filter(s => s.id !== id)); })
      .catch(err => setError(err.message));
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

  if (loading) return <div className="p-6"><h1 className="text-2xl font-bold">Scope Management</h1><p className="text-gray-600 mt-2">Loading...</p></div>;
  if (error) return <div className="p-6"><h1 className="text-2xl font-bold">Scope Management</h1><p className="text-red-600 mt-2">Error: {error}</p></div>;

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold">Scope Management</h1><p className="text-gray-600">Manage OAuth/OIDC scopes, hierarchy, and client assignments.</p></div>
        <button onClick={() => setShowForm(!showForm)} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">{showForm ? 'Cancel' : 'Create Scope'}</button>
      </div>

      {showForm && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Create Scope</h2>
          <div className="grid grid-cols-2 gap-4">
            <div><label className="text-sm font-medium">Name</label><input aria-label="read:reports" type="text" placeholder="read:reports" value={newScope.name} onChange={e => setNewScope(prev => ({ ...prev, name: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1 font-mono" /></div>
            <div><label className="text-sm font-medium">Parent Scope</label><select aria-label="new Scope" value={newScope.parent} onChange={e => setNewScope(prev => ({ ...prev, parent: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1"><option value="-">(none)</option>{allScopeNames.map(n => <option key={n} value={n}>{n}</option>)}</select></div>
          </div>
          <div><label className="text-sm font-medium">Description</label><input aria-label="new Scope" type="text" value={newScope.description} onChange={e => setNewScope(prev => ({ ...prev, description: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
          <div className="flex gap-4"><label className="flex items-center gap-2 text-sm"><input aria-label="New scope" type="checkbox" checked={newScope.defaultForClients} onChange={e => setNewScope(prev => ({ ...prev, defaultForClients: e.target.checked }))} className="rounded" />Default for new clients</label></div>
          <button onClick={addScope} disabled={!newScope.name} className="px-4 py-2 bg-blue-600 text-white rounded text-sm disabled:opacity-50">Add Scope</button>
        </section>
      )}

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Scope List</h2>
        {scopes.length === 0 ? <p className="text-gray-500 text-sm">No scopes configured.</p> :
        <table className="w-full text-sm">
          <thead className="bg-gray-50"><tr className="text-left"><th className="p-3">Name</th><th className="p-3">Description</th><th className="p-3">Type</th><th className="p-3">Default</th><th className="p-3">Actions</th></tr></thead>
          <tbody>{scopes.map(s => (<tr key={s.id} className="border-b"><td className="p-3 font-mono text-xs">{s.name}</td><td className="p-3 text-gray-600">{s.description}</td><td className="p-3"><span className={`px-2 py-0.5 rounded text-xs ${s.isSystem ? 'bg-purple-100 text-purple-700' : 'bg-blue-100 text-blue-700'}`}>{s.isSystem ? 'system' : 'custom'}</span></td><td className="p-3">{s.defaultForClients ? <span className="text-green-600 text-xs">yes</span> : <span className="text-gray-400 text-xs">no</span>}</td><td className="p-3">{!s.isSystem && <button onClick={() => deleteScope(s.id)} className="text-red-600 text-xs hover:underline">Delete</button>}</td></tr>))}</tbody>
        </table>}
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Scope Hierarchy</h2>
        {renderTree('-', 0)}
      </section>

      {clients.length > 0 && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Scope-to-Client Assignment Matrix</h2>
          <table className="w-full text-xs">
            <thead className="bg-gray-50"><tr className="text-left"><th className="p-2">Client</th>{scopes.map(s => <th key={s.id} className="p-2">{s.name}</th>)}</tr></thead>
            <tbody>{clients.map(c => (<tr key={c.name} className="border-b"><td className="p-2 font-medium">{c.name}</td>{scopes.map(s => (<td key={s.id} className="p-2 text-center">{c.scopes.includes(s.name) ? <span className="text-green-600">V</span> : <span className="text-gray-300">-</span>}</td>))}</tr>))}</tbody>
          </table>
        </section>
      )}
    </div>
  );
}

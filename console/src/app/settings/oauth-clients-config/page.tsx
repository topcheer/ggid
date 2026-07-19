'use client';
import { useState, useEffect } from 'react';
import { Loader2 } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface Client {
  id: string;
  clientId: string;
  name: string;
  grantTypes: string[];
  scopes: string[];
  redirectUris: string[];
  status: string;
  tokenLifetime: number;
  logoUri: string;
  policyUri: string;
}

export default function OauthClientsConfigPage() {
  const t = useTranslations();


  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const res = await fetch("/api/v1/oauth/clients", {
          method: "GET",
          headers: { ...authHeader(),
            "Content-Type": "application/json",
            "X-Tenant-ID": "00000000-0000-0000-0000-000000000001",
          },
        });
        if (!res.ok) return null;
        const json = await res.json();
      } catch (e) {
        setError(e instanceof Error ? e.message : "Failed to load");
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, []);

  const [showForm, setShowForm] = useState(false);
  const [editing, setEditing] = useState<Client | null>(null);
  const [showSecret, setShowSecret] = useState(false);
  const [newClient, setNewClient] = useState({ name: '', clientId: '', grantTypes: [] as string[], scopes: [] as string[], redirectUris: '' });const [clients, setClients] = useState<Client[]>([
    { id: 'c1', clientId: 'web-app', name: 'Web Application', grantTypes: ['authorization_code', 'refresh_token'], scopes: ['openid', 'profile'], redirectUris: ['https://app.ggid.io/callback'], status: 'active', tokenLifetime: 3600, logoUri: 'https://app.ggid.io/logo.png', policyUri: 'https://app.ggid.io/privacy' },
    { id: 'c2', clientId: 'mobile-app', name: 'Mobile App', grantTypes: ['authorization_code', 'refresh_token'], scopes: ['openid', 'profile', 'offline_access'], redirectUris: ['com.ggid.app://callback'], status: 'active', tokenLifetime: 7200, logoUri: '', policyUri: '' },
    { id: 'c3', clientId: 'admin-cli', name: 'Admin CLI', grantTypes: ['client_credentials'], scopes: ['admin:all'], redirectUris: [], status: 'active', tokenLifetime: 1800, logoUri: '', policyUri: '' },
    { id: 'c4', clientId: 'legacy-svc', name: 'Legacy Service', grantTypes: ['password'], scopes: ['read:users'], redirectUris: ['https://legacy.ggid.io/cb'], status: 'disabled', tokenLifetime: 3600, logoUri: '', policyUri: '' },
  ]);

  if (loading) return <div className="flex items-center justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-blue-500" /></div>;
  if (error) return <div className="p-8 text-red-500">Error: {error}</div>;
  

  const allGrants = ['authorization_code', 'refresh_token', 'client_credentials', 'password', 'implicit', 'urn:ietf:params:oauth:grant-type:device_code'];
  const allScopes = ['openid', 'profile', 'email', 'offline_access', 'read:users', 'write:users', 'admin:all'];

  const toggleGrant = (g: string) => setNewClient(prev => ({ ...prev, grantTypes: prev.grantTypes.includes(g) ? prev.grantTypes.filter(x => x !== g) : [...prev.grantTypes, g] }));
  const toggleScope = (s: string) => setNewClient(prev => ({ ...prev, scopes: prev.scopes.includes(s) ? prev.scopes.filter(x => x !== s) : [...prev.scopes, s] }));

  const createClient = () => {
    const id = `c${clients.length + 1}`;
    setClients(prev => [...prev, { id, clientId: newClient.clientId || `client-${Date.now()}`, name: newClient.name, grantTypes: newClient.grantTypes, scopes: newClient.scopes, redirectUris: newClient.redirectUris.split('\n').filter(Boolean), status: 'active', tokenLifetime: 3600, logoUri: '', policyUri: '' }]);
    setShowForm(false); setNewClient({ name: '', clientId: '', grantTypes: [], scopes: [], redirectUris: '' });
    setShowSecret(true);
  };

  const deleteClient = (id: string) => setClients(prev => prev.filter(c => c.id !== id));
  const toggleStatus = (id: string) => setClients(prev => prev.map(c => c.id === id ? { ...c, status: c.status === 'active' ? 'disabled' : 'active' } : c));

  const statusColor = (s: string) => s === 'active' ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700';

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold">OAuth Clients Configuration</h1><p className="text-gray-600">Manage OAuth 2.0/OIDC client registrations, metadata, and token lifetimes.</p></div>
        <button onClick={() => setShowForm(!showForm)} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">{showForm ? 'Cancel' : 'Create Client'}</button>
      </div>

      {showSecret && (
        <div className="bg-amber-50 border border-amber-200 rounded p-4 text-sm space-y-2">
          <div className="font-medium text-amber-800">Client Secret (shown once):</div>
          <div className="font-mono text-xs bg-white rounded p-2">{createdSecret || "(API response)"}</div>
          <button onClick={() => setShowSecret(false)} className="text-xs text-blue-600">Dismiss</button>
        </div>
      )}

      {showForm && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Create OAuth Client</h2>
          <div className="grid grid-cols-2 gap-4">
            <div><label className="text-sm font-medium">Client Name</label><input aria-label="new Client" type="text" value={newClient.name} onChange={e => setNewClient(prev => ({ ...prev, name: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
            <div><label className="text-sm font-medium">Client ID (auto if empty)</label><input aria-label="new Client" type="text" value={newClient.clientId} onChange={e => setNewClient(prev => ({ ...prev, clientId: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
          </div>
          <div><label className="text-sm font-medium">Grant Types</label><div className="flex flex-wrap gap-2 mt-2">{allGrants.map(g => <label key={g} className="flex items-center gap-1 text-xs"><input aria-label="New client" type="checkbox" checked={newClient.grantTypes.includes(g)} onChange={() => toggleGrant(g)} className="rounded" /><span className="font-mono">{g}</span></label>)}</div></div>
          <div><label className="text-sm font-medium">Scopes</label><div className="flex flex-wrap gap-2 mt-2">{allScopes.map(s => <label key={s} className="flex items-center gap-1 text-xs"><input aria-label="New client" type="checkbox" checked={newClient.scopes.includes(s)} onChange={() => toggleScope(s)} className="rounded" /><span className="font-mono">{s}</span></label>)}</div></div>
          <div><label className="text-sm font-medium">Redirect URIs (one per line)</label><textarea aria-label="Text input" value={newClient.redirectUris} onChange={e => setNewClient(prev => ({ ...prev, redirectUris: e.target.value }))} rows={3} className="w-full border rounded px-3 py-2 text-sm mt-1 font-mono" /></div>
          <button onClick={createClient} disabled={!newClient.name} className="px-4 py-2 bg-blue-600 text-white rounded text-sm disabled:opacity-50">Create</button>
        </section>
      )}

      <section className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50"><tr className="text-left"><th className="p-3">Client ID</th><th className="p-3">Name</th><th className="p-3">Grants</th><th className="p-3">Scopes</th><th className="p-3">Token TTL</th><th className="p-3">Status</th><th className="p-3">Actions</th></tr></thead>
          <tbody>
            {clients.map(c => (
              <tr key={c.id} className="border-b">
                <td className="p-3 font-mono text-xs">{c.clientId}</td>
                <td className="p-3 font-medium">{c.name}</td>
                <td className="p-3"><div className="flex flex-wrap gap-1">{c.grantTypes.map(g => <span key={g} className="px-1 py-0.5 bg-purple-100 text-purple-700 rounded text-xs">{g}</span>)}</div></td>
                <td className="p-3"><div className="flex flex-wrap gap-1">{c.scopes.map(s => <span key={s} className="px-1 py-0.5 bg-blue-100 text-blue-700 rounded text-xs">{s}</span>)}</div></td>
                <td className="p-3 text-xs">{c.tokenLifetime}s</td>
                <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs ${statusColor(c.status)}`}>{c.status}</span></td>
                <td className="p-3"><div className="flex gap-2"><button onClick={() => toggleStatus(c.id)} className="text-blue-600 text-xs hover:underline">{c.status === 'active' ? 'Disable' : 'Enable'}</button><button onClick={() => deleteClient(c.id)} className="text-red-600 text-xs hover:underline">Delete</button></div></td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </div>
  );
}
'use client';
import { useTranslations } from "@/lib/i18n";
import { useState, useEffect } from 'react';
import { Loader2 } from "lucide-react";

interface Client {
  id: string;
  clientId: string;
  name: string;
  status: string;
  created: string;
  lastUsed: string;
  grantTypes: string[];
  redirectUris: string[];
  secretRotatedAt: string;
}

export default function ClientLifecyclePage() {

  const t = useTranslations();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const res = await fetch("/api/v1/oauth/clients", {
          method: "GET",
          headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`,
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
  const [deleteTarget, setDeleteTarget] = useState<Client | null>(null);
  const [newClient, setNewClient] = useState({ name: '', grantTypes: ['authorization_code'], redirectUris: '' });const [clients, setClients] = useState<Client[]>([
    { id: 'c1', clientId: 'web-app-prod', name: 'Web App Production', status: 'active', created: '2026-01-15', lastUsed: '2026-07-12', grantTypes: ['authorization_code', 'refresh_token'], redirectUris: ['https://app.ggid.io/callback'], secretRotatedAt: '2026-06-01' },
    { id: 'c2', clientId: 'mobile-app', name: 'Mobile App', status: 'active', created: '2026-03-01', lastUsed: '2026-07-11', grantTypes: ['authorization_code', 'refresh_token', 'pkce'], redirectUris: ['com.ggid.app://callback'], secretRotatedAt: '2026-05-15' },
    { id: 'c3', clientId: 'batch-service', name: 'Batch Processing Service', status: 'active', created: '2025-11-01', lastUsed: '2026-07-12', grantTypes: ['client_credentials'], redirectUris: [], secretRotatedAt: '2026-04-01' },
    { id: 'c4', clientId: 'legacy-portal', name: 'Legacy Portal', status: 'inactive', created: '2024-06-01', lastUsed: '2026-02-28', grantTypes: ['authorization_code'], redirectUris: ['https://legacy.ggid.io/cb'], secretRotatedAt: '2025-06-01' },
  ]);

  if (loading) return <div className="flex items-center justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-blue-500" /></div>;
  if (error) return <div className="p-8 text-red-500">Error: {error}</div>;
  

  const statusColor = (s: string): string =>
    s === 'active' ? 'bg-green-100 text-green-700' : 'bg-gray-200 text-gray-600';

  const allGrantTypes = ['authorization_code', 'refresh_token', 'client_credentials', 'pkce', 'password', 'device_code'];

  const toggleGrantType = (gt: string) => {
    setNewClient(prev => ({
      ...prev,
      grantTypes: prev.grantTypes.includes(gt) ? prev.grantTypes.filter(g => g !== gt) : [...prev.grantTypes, gt],
    }));
  };

  const registerClient = () => {
    const clientId = newClient.name.toLowerCase().replace(/\s+/g, '-');
    setClients(prev => [...prev, {
      id: `c${prev.length + 1}`,
      clientId,
      name: newClient.name || `Client ${prev.length + 1}`,
      status: 'active',
      created: new Date().toISOString().slice(0, 10),
      lastUsed: new Date().toISOString().slice(0, 10),
      grantTypes: newClient.grantTypes,
      redirectUris: newClient.redirectUris ? newClient.redirectUris.split(',').map(u => u.trim()) : [],
      secretRotatedAt: new Date().toISOString().slice(0, 10),
    }]);
    setShowForm(false);
    setNewClient({ name: '', grantTypes: ['authorization_code'], redirectUris: '' });
  };

  const toggleStatus = (id: string) => {
    setClients(prev => prev.map(c => c.id === id ? { ...c, status: c.status === 'active' ? 'inactive' : 'active' } : c));
  };

  const rotateSecret = (id: string) => {
    setClients(prev => prev.map(c => c.id === id ? { ...c, secretRotatedAt: new Date().toISOString().slice(0, 10) } : c));
  };

  const confirmDelete = () => {
    if (deleteTarget) setClients(prev => prev.filter(c => c.id !== deleteTarget.id));
    setDeleteTarget(null);
  };

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t("backend.clientLifecycle.title")}</h1>
          <p className="text-gray-600">Register, manage, and deactivate OAuth clients (RFC 7591/7592).</p>
        </div>
        <button onClick={() => setShowForm(!showForm)} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">
          {showForm ? 'Cancel' : 'Register Client'}
        </button>
      </div>

      {showForm && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Register OAuth Client (RFC 7591)</h2>
          <div>
            <label className="text-sm font-medium">{t("backend.clientLifecycle.clientName")}</label>
            <input aria-label="My Application" type="text" placeholder="My Application" value={newClient.name} onChange={e => setNewClient(prev => ({ ...prev, name: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" />
          </div>
          <div>
            <label className="text-sm font-medium">{t("backend.clientLifecycle.grantTypes")}</label>
            <div className="flex flex-wrap gap-2 mt-2">
              {allGrantTypes.map(gt => (
                <label key={gt} className="flex items-center gap-1 text-sm">
                  <input aria-label="New client" type="checkbox" checked={newClient.grantTypes.includes(gt)} onChange={() => toggleGrantType(gt)} className="rounded" />
                  {gt}
                </label>
              ))}
            </div>
          </div>
          <div>
            <label className="text-sm font-medium">Redirect URIs (comma-separated)</label>
            <input aria-label="https://app.example.com/callback, com.example.app://cb" type="text" placeholder="https://app.example.com/callback, com.example.app://cb" value={newClient.redirectUris} onChange={e => setNewClient(prev => ({ ...prev, redirectUris: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" />
          </div>
          <button onClick={registerClient} disabled={!newClient.name} className="px-4 py-2 bg-blue-600 text-white rounded text-sm disabled:opacity-50">Register</button>
        </section>
      )}

      <div className="grid grid-cols-3 gap-4">
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold">{clients.length}</div>
          <div className="text-sm text-gray-500">Total Clients</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold text-green-600">{clients.filter(c => c.status === 'active').length}</div>
          <div className="text-sm text-gray-500">{t("backend.clientLifecycle.active")}</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold text-gray-500">{clients.filter(c => c.status === 'inactive').length}</div>
          <div className="text-sm text-gray-500">Inactive</div>
        </div>
      </div>

      <section className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th scope="col" className="p-3">{t("backend.clientLifecycle.clientId")}</th>
              <th scope="col" className="p-3">{t("backend.clientLifecycle.clientName")}</th>
              <th scope="col" className="p-3">Status</th>
              <th scope="col" className="p-3">{t("backend.clientLifecycle.grantTypes")}</th>
              <th scope="col" className="p-3">Redirect URIs</th>
              <th scope="col" className="p-3">{t("backend.clientLifecycle.created")}</th>
              <th scope="col" className="p-3">Secret Rotated</th>
              <th scope="col" className="p-3">{t("backend.clientLifecycle.actions")}</th>
            </tr>
          </thead>
          <tbody>
            {clients.map(c => (
              <tr key={c.id} className="border-b hover:bg-gray-50">
                <td className="p-3 font-mono text-xs">{c.clientId}</td>
                <td className="p-3 font-medium">{c.name}</td>
                <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs ${statusColor(c.status)}`}>{c.status}</span></td>
                <td className="p-3"><div className="flex flex-wrap gap-1">{c.grantTypes.map(g => <span key={g} className="px-1.5 py-0.5 bg-gray-100 rounded text-xs">{g}</span>)}</div></td>
                <td className="p-3 text-xs text-gray-500">{c.redirectUris.length} URI(s)</td>
                <td className="p-3 text-gray-500">{c.created}</td>
                <td className="p-3 text-gray-500">{c.secretRotatedAt}</td>
                <td className="p-3">
                  <div className="flex gap-2">
                    <button onClick={() => rotateSecret(c.id)} className="text-blue-600 text-xs hover:underline">Rotate Secret</button>
                    <button onClick={() => toggleStatus(c.id)} className="text-amber-600 text-xs hover:underline">{c.status === 'active' ? 'Deactivate' : 'Activate'}</button>
                    <button onClick={() => setDeleteTarget(c)} className="text-red-600 text-xs hover:underline">{t("backend.clientLifecycle.delete")}</button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      {deleteTarget && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 max-w-md w-full mx-4 space-y-4">
            <h2 className="text-lg font-semibold">{t("backend.clientLifecycle.deleteClient")}</h2>
            <p className="text-sm text-gray-600">Permanently delete <strong>{deleteTarget.name}</strong> ({deleteTarget.clientId})? All tokens will be revoked immediately.</p>
            <div className="flex justify-end gap-3">
              <button onClick={() => setDeleteTarget(null)} className="px-4 py-2 border rounded text-sm">{t("backend.clientLifecycle.cancel")}</button>
              <button aria-label="action" onClick={confirmDelete} className="px-4 py-2 bg-red-600 text-white rounded text-sm">{t("backend.clientLifecycle.confirmDelete")}</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
'use client';
import { useState, useEffect } from 'react';
import { Loader2 } from 'lucide-react';
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface DynamicClient {
  id: string;
  clientId: string;
  created: string;
  grantTypes: string[];
  scopes: string[];
  softwareStatement: boolean;
  status: string;
}

export default function DynamicClientRegistrationPage() {
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
  const [autoApprove, setAutoApprove] = useState(true);
  const [openRegistration, setOpenRegistration] = useState(true);
  const [validateSoftware, setValidateSoftware] = useState(true);
  const [validateUri, setValidateUri] = useState(true);
  const [validateLogo, setValidateLogo] = useState(false);
  const [maxRedirectUris, setMaxRedirectUris] = useState(5);
  const [form, setForm] = useState({ clientName: '', redirectUris: '', grantTypes: '', scopes: '', softwareStatement: false });const [clients, setClients] = useState<DynamicClient[]>([
    { id: 'dc1', clientId: 'dyn-client-001', created: '2026-07-01', grantTypes: ['authorization_code'], scopes: ['openid', 'profile'], softwareStatement: true, status: 'active' },
    { id: 'dc2', clientId: 'dyn-client-002', created: '2026-06-15', grantTypes: ['client_credentials'], scopes: ['read:users'], softwareStatement: false, status: 'active' },
    { id: 'dc3', clientId: 'dyn-client-003', created: '2026-05-20', grantTypes: ['authorization_code', 'refresh_token'], scopes: ['openid', 'email'], softwareStatement: true, status: 'disabled' },
  ]);

  if (loading) return <div className="flex items-center justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-blue-500" /></div>;
  if (error) return <div className="p-8 text-red-500">Error: {error}</div>;
  


  const register = () => {
    const newClient: DynamicClient = {
      id: `dc${clients.length + 1}`,
      clientId: `dyn-client-${String(clients.length + 1).padStart(3, '0')}`,
      created: new Date().toISOString().slice(0, 10),
      grantTypes: form.grantTypes.split(',').map(s => s.trim()).filter(Boolean),
      scopes: form.scopes.split(' ').map(s => s.trim()).filter(Boolean),
      softwareStatement: form.softwareStatement,
      status: autoApprove ? 'active' : 'pending',
    };
    setClients(prev => [newClient, ...prev]);
    setShowForm(false);
    setForm({ clientName: '', redirectUris: '', grantTypes: '', scopes: '', softwareStatement: false });
  };

  const statusColor = (s: string): string =>
    s === 'active' ? 'bg-green-100 text-green-700' : s === 'pending' ? 'bg-amber-100 text-amber-700' : 'bg-red-100 text-red-700';

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Dynamic Client Registration</h1>
        <p className="text-gray-600">RFC 7591 dynamic OAuth client registration and management.</p>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <label className="flex items-center justify-between bg-white rounded-lg shadow p-4">
          <span className="text-sm font-medium">Open Registration</span>
          <input aria-label="Open registration" type="checkbox" checked={openRegistration} onChange={e => setOpenRegistration(e.target.checked)} className="rounded" />
        </label>
        <label className="flex items-center justify-between bg-white rounded-lg shadow p-4">
          <span className="text-sm font-medium">Auto-Approve New Clients</span>
          <input aria-label="Auto approve" type="checkbox" checked={autoApprove} onChange={e => setAutoApprove(e.target.checked)} className="rounded" />
        </label>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Validation Settings</h2>
        <div className="grid grid-cols-2 gap-4">
          <label className="flex items-center justify-between"><span className="text-sm">Validate Software Statements</span><input aria-label="Validate software" type="checkbox" checked={validateSoftware} onChange={e => setValidateSoftware(e.target.checked)} className="rounded" /></label>
          <label className="flex items-center justify-between"><span className="text-sm">Validate Client URIs</span><input aria-label="Validate uri" type="checkbox" checked={validateUri} onChange={e => setValidateUri(e.target.checked)} className="rounded" /></label>
          <label className="flex items-center justify-between"><span className="text-sm">Validate Logo URIs</span><input aria-label="Validate logo" type="checkbox" checked={validateLogo} onChange={e => setValidateLogo(e.target.checked)} className="rounded" /></label>
          <div className="flex items-center gap-3">
            <label className="text-sm">Max Redirect URIs:</label>
            <input aria-label="max Redirect Uris" type="number" min={1} max={20} value={maxRedirectUris} onChange={e => setMaxRedirectUris(parseInt(e.target.value) || 5)} className="w-16 border rounded px-2 py-1 text-sm" />
          </div>
        </div>
      </section>

      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Registered Clients ({clients.length})</h2>
        <button onClick={() => setShowForm(!showForm)} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">{showForm ? 'Cancel' : 'Register Client'}</button>
      </div>

      {showForm && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Register New Client (RFC 7591)</h2>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="text-sm font-medium">Client Name</label>
              <input aria-label="form" type="text" value={form.clientName} onChange={e => setForm(prev => ({ ...prev, clientName: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" />
            </div>
            <div>
              <label className="text-sm font-medium">Redirect URIs (comma-separated)</label>
              <input aria-label="https://app.example.com/callback" type="text" placeholder="https://app.example.com/callback" value={form.redirectUris} onChange={e => setForm(prev => ({ ...prev, redirectUris: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" />
            </div>
            <div>
              <label className="text-sm font-medium">Grant Types (comma-separated)</label>
              <input aria-label="authorization_code, refresh_token" type="text" placeholder="authorization_code, refresh_token" value={form.grantTypes} onChange={e => setForm(prev => ({ ...prev, grantTypes: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" />
            </div>
            <div>
              <label className="text-sm font-medium">Scopes (space-separated)</label>
              <input aria-label="openid profile email" type="text" placeholder="openid profile email" value={form.scopes} onChange={e => setForm(prev => ({ ...prev, scopes: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" />
            </div>
          </div>
          <label className="flex items-center gap-2 text-sm">
            <input aria-label="Form" type="checkbox" checked={form.softwareStatement} onChange={e => setForm(prev => ({ ...prev, softwareStatement: e.target.checked }))} className="rounded" />
            Include Software Statement (JWT)
          </label>
          <button onClick={register} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">Register</button>
        </section>
      )}

      <section className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th scope="col" className="p-3">Client ID</th>
              <th scope="col" className="p-3">Created</th>
              <th scope="col" className="p-3">Grant Types</th>
              <th scope="col" className="p-3">Scopes</th>
              <th scope="col" className="p-3">Software Statement</th>
              <th scope="col" className="p-3">Status</th>
            </tr>
          </thead>
          <tbody>
            {clients.map(c => (
              <tr key={c.id} className="border-b">
                <td className="p-3 font-mono text-xs">{c.clientId}</td>
                <td className="p-3 text-gray-500">{c.created}</td>
                <td className="p-3"><div className="flex flex-wrap gap-1">{c.grantTypes.map(g => <span key={g} className="px-1.5 py-0.5 bg-blue-100 text-blue-700 rounded text-xs">{g}</span>)}</div></td>
                <td className="p-3"><div className="flex flex-wrap gap-1">{c.scopes.map(s => <span key={s} className="px-1.5 py-0.5 bg-gray-100 rounded text-xs">{s}</span>)}</div></td>
                <td className="p-3">{c.softwareStatement ? <span className="text-green-600 text-xs">yes</span> : <span className="text-gray-400 text-xs">no</span>}</td>
                <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs ${statusColor(c.status)}`}>{c.status}</span></td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </div>
  );
}
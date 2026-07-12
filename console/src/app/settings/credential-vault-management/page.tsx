'use client';
import { useState } from 'react';

interface Credential {
  id: string;
  type: string;
  name: string;
  created: string;
  lastUsed: string;
  lastRotated: string;
  status: string;
  encrypted: boolean;
  rotationDays: number;
}

export default function CredentialVaultManagementPage() {
  const [credentials, setCredentials] = useState<Credential[]>([
    { id: 'c1', type: 'API Key', name: 'gateway-prod-key', created: '2026-01-15', lastUsed: '2026-07-12', lastRotated: '2026-06-01', status: 'active', encrypted: true, rotationDays: 90 },
    { id: 'c2', type: 'OAuth Token', name: 'azure-ad-service-token', created: '2026-03-01', lastUsed: '2026-07-11', lastRotated: '2026-05-01', status: 'active', encrypted: true, rotationDays: 60 },
    { id: 'c3', type: 'SSH Key', name: 'deploy-key-prod', created: '2025-08-01', lastUsed: '2026-07-10', lastRotated: '2025-08-01', status: 'active', encrypted: true, rotationDays: 365 },
    { id: 'c4', type: 'Password', name: 'ldap-bind-pw', created: '2025-12-01', lastUsed: '2026-07-12', lastRotated: '2026-03-01', status: 'active', encrypted: true, rotationDays: 90 },
    { id: 'c5', type: 'Certificate', name: 'auth-tls-cert', created: '2025-06-01', lastUsed: '2026-07-12', lastRotated: '2025-06-01', status: 'expired', encrypted: true, rotationDays: 365 },
  ]);

  const [showForm, setShowForm] = useState(false);
  const [newCred, setNewCred] = useState({ type: 'API Key', name: '', rotationDays: 90 });
  const [auditLog, setAuditLog] = useState([
    { timestamp: '2026-07-12 14:30', action: 'Rotated', credential: 'gateway-prod-key', actor: 'admin@ggid.io' },
    { timestamp: '2026-07-11 09:15', action: 'Accessed', credential: 'azure-ad-service-token', actor: 'auth-service' },
    { timestamp: '2026-07-10 16:00', action: 'Created', credential: 'deploy-key-prod', actor: 'devops@ggid.io' },
    { timestamp: '2026-07-09 11:20', action: 'Expired', credential: 'auth-tls-cert', actor: 'system' },
  ]);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());

  const types = ['API Key', 'OAuth Token', 'SSH Key', 'Password', 'Certificate'];

  const statusColor = (s: string): string =>
    s === 'active' ? 'bg-green-100 text-green-700' :
    s === 'expired' ? 'bg-red-100 text-red-700' :
    'bg-gray-200 text-gray-600';

  const isExpired = (c: Credential): boolean => {
    const days = Math.floor((Date.now() - new Date(c.lastRotated).getTime()) / 86400000);
    return days > c.rotationDays;
  };

  const addCredential = () => {
    const today = new Date().toISOString().slice(0, 10);
    setCredentials(prev => [...prev, {
      id: `c${prev.length + 1}`,
      type: newCred.type,
      name: newCred.name || `credential-${prev.length + 1}`,
      created: today,
      lastUsed: today,
      lastRotated: today,
      status: 'active',
      encrypted: true,
      rotationDays: newCred.rotationDays,
    }]);
    setAuditLog(prev => [{ timestamp: today + ' ' + new Date().toTimeString().slice(0, 5), action: 'Created', credential: newCred.name || `credential-${credentials.length + 1}`, actor: 'current-user@ggid.io' }, ...prev]);
    setShowForm(false);
    setNewCred({ type: 'API Key', name: '', rotationDays: 90 });
  };

  const toggleSelect = (id: string) => {
    setSelectedIds(prev => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id); else next.add(id);
      return next;
    });
  };

  const bulkRotate = () => {
    const now = new Date().toISOString().slice(0, 10);
    setCredentials(prev => prev.map(c => selectedIds.has(c.id) ? { ...c, lastRotated: now, status: 'active' } : c));
    selectedIds.forEach(id => {
      const cred = credentials.find(c => c.id === id);
      if (cred) setAuditLog(prev => [{ timestamp: now + ' ' + new Date().toTimeString().slice(0, 5), action: 'Rotated', credential: cred.name, actor: 'current-user@ggid.io' }, ...prev]);
    });
    setSelectedIds(new Set());
  };

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Credential Vault Management</h1>
          <p className="text-gray-600">Secure storage and rotation of all organization credentials.</p>
        </div>
        <button onClick={() => setShowForm(!showForm)} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">
          {showForm ? 'Cancel' : 'Add Credential'}
        </button>
      </div>

      {showForm && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Add Credential</h2>
          <div>
            <label className="text-sm font-medium">Type</label>
            <select value={newCred.type} onChange={e => setNewCred(prev => ({ ...prev, type: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1">
              {types.map(t => <option key={t} value={t}>{t}</option>)}
            </select>
          </div>
          <div>
            <label className="text-sm font-medium">Name</label>
            <input type="text" placeholder="Credential name" value={newCred.name} onChange={e => setNewCred(prev => ({ ...prev, name: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" />
          </div>
          <div>
            <label className="text-sm font-medium">Rotation Policy (days)</label>
            <input type="number" min={1} max={365} value={newCred.rotationDays} onChange={e => setNewCred(prev => ({ ...prev, rotationDays: parseInt(e.target.value) || 90 }))} className="w-24 border rounded px-3 py-2 text-sm mt-1" />
          </div>
          <button onClick={addCredential} disabled={!newCred.name} className="px-4 py-2 bg-blue-600 text-white rounded text-sm disabled:opacity-50">Add to Vault</button>
        </section>
      )}

      <div className="grid grid-cols-4 gap-4">
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold">{credentials.length}</div>
          <div className="text-sm text-gray-500">Total</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold text-green-600">{credentials.filter(c => c.encrypted).length}</div>
          <div className="text-sm text-gray-500">Encrypted</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold text-red-600">{credentials.filter(c => c.status === 'expired' || isExpired(c)).length}</div>
          <div className="text-sm text-gray-500">Expired</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold text-amber-600">{credentials.filter(c => c.type === 'API Key').length}</div>
          <div className="text-sm text-gray-500">API Keys</div>
        </div>
      </div>

      {selectedIds.size > 0 && (
        <div className="flex items-center gap-3 bg-blue-50 rounded p-3">
          <span className="text-sm">{selectedIds.size} selected</span>
          <button onClick={bulkRotate} className="px-3 py-1 bg-blue-600 text-white rounded text-sm">Bulk Rotate</button>
          <button onClick={() => setSelectedIds(new Set())} className="px-3 py-1 border rounded text-sm">Clear</button>
        </div>
      )}

      <section className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th className="p-3"></th>
              <th className="p-3">Name</th>
              <th className="p-3">Type</th>
              <th className="p-3">Created</th>
              <th className="p-3">Last Used</th>
              <th className="p-3">Last Rotated</th>
              <th className="p-3">Rotation</th>
              <th className="p-3">Encrypted</th>
              <th className="p-3">Status</th>
            </tr>
          </thead>
          <tbody>
            {credentials.map(c => (
              <tr key={c.id} className="border-b hover:bg-gray-50">
                <td className="p-3"><input type="checkbox" checked={selectedIds.has(c.id)} onChange={() => toggleSelect(c.id)} className="rounded" /></td>
                <td className="p-3 font-medium">{c.name}</td>
                <td className="p-3"><span className="px-2 py-0.5 bg-gray-100 rounded text-xs">{c.type}</span></td>
                <td className="p-3 text-gray-500">{c.created}</td>
                <td className="p-3 text-gray-500">{c.lastUsed}</td>
                <td className="p-3 text-gray-500">{c.lastRotated}</td>
                <td className="p-3 text-gray-500">{c.rotationDays}d</td>
                <td className="p-3">{c.encrypted ? <span className="text-green-600 text-xs">AES-256</span> : <span className="text-red-600 text-xs">No</span>}</td>
                <td className="p-3">
                  <span className={`px-2 py-0.5 rounded text-xs ${statusColor(c.status === 'active' && isExpired(c) ? 'expired' : c.status)}`}>
                    {c.status === 'active' && isExpired(c) ? 'expired' : c.status}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Access Audit Log</h2>
        <div className="space-y-2">
          {auditLog.map((entry, idx) => (
            <div key={idx} className="flex items-center gap-3 text-sm border-b pb-2">
              <span className="text-xs text-gray-400 w-32">{entry.timestamp}</span>
              <span className={`px-2 py-0.5 rounded text-xs ${
                entry.action === 'Rotated' ? 'bg-blue-100 text-blue-700' :
                entry.action === 'Created' ? 'bg-green-100 text-green-700' :
                entry.action === 'Expired' ? 'bg-red-100 text-red-700' :
                'bg-gray-100 text-gray-600'
              }`}>{entry.action}</span>
              <span className="font-mono text-xs flex-1">{entry.credential}</span>
              <span className="text-xs text-gray-500">{entry.actor}</span>
            </div>
          ))}
        </div>
      </section>
    </div>
  );
}
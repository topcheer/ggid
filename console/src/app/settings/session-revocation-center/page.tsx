'use client';
import { useTranslations } from "@/lib/i18n";
import { useState, useEffect } from 'react';

interface Session {
  id: string;
  userId: string;
  device: string;
  ip: string;
  created: string;
  lastActive: string;
  tenant: string;
}

export default function SessionRevocationCenterPage() {
  const t = useTranslations();
  const [sessions, setSessions] = useState<Session[]>([
    { id: 's1', userId: 'alice@ggid.io', device: 'MacBook Pro', ip: '10.0.0.15', created: '2026-07-12 08:00', lastActive: '2026-07-12 14:30', tenant: 'default' },
    { id: 's2', userId: 'bob@ggid.io', device: 'iPhone 15', ip: '10.0.0.22', created: '2026-07-11 20:00', lastActive: '2026-07-12 13:45', tenant: 'default' },
    { id: 's3', userId: 'alice@ggid.io', device: 'Chrome/Windows', ip: '192.168.1.50', created: '2026-07-10 14:00', lastActive: '2026-07-11 18:00', tenant: 'default' },
    { id: 's4', userId: 'carol@ggid.io', device: 'iPad', ip: '10.0.0.33', created: '2026-07-12 09:00', lastActive: '2026-07-12 14:00', tenant: 'acme' },
  ]);

  const [reason, setReason] = useState('');
  const [bulkConfirm, setBulkConfirm] = useState(false);
  const [auditLog, setAuditLog] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch('/api/v1/auth/sessions/revoke', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': '00000000-0000-0000-0000-000000000001' },
      body: JSON.stringify({ action: 'list' }),
    })
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(data => {
        if (data && data.sessions) setSessions(data.sessions);
        else if (Array.isArray(data)) setSessions(data);
        setLoading(false);
      })
      .catch(e => { setError(e.message); setLoading(false); });
  }, []);

  const revoke = (id: string, userId: string) => {
    setSessions(prev => prev.filter(s => s.id !== id));
    setAuditLog(prev => [`Revoked session ${id} for ${userId}${reason ? `: ${reason}` : ''} at ${new Date().toISOString().slice(0, 16)}`, ...prev]);
  };

  const revokeAllForUser = (userId: string) => {
    const count = sessions.filter(s => s.userId === userId).length;
    setSessions(prev => prev.filter(s => s.userId !== userId));
    setAuditLog(prev => [`Bulk revoked ${count} sessions for ${userId}${reason ? `: ${reason}` : ''}`, ...prev]);
  };

  const revokeByTenant = (tenant: string) => {
    const count = sessions.filter(s => s.tenant === tenant).length;
    setSessions(prev => prev.filter(s => s.tenant !== tenant));
    setAuditLog(prev => [`Revoked ${count} sessions for tenant ${tenant}${reason ? `: ${reason}` : ''}`, ...prev]);
  };

  const bulkRevoke = () => {
    setSessions([]);
    setAuditLog(prev => [`BULK REVOKE: All ${sessions.length} sessions revoked${reason ? `: ${reason}` : ''}`, ...prev]);
    setBulkConfirm(false);
  };

  if (loading) return <div className="p-6"><p>Loading...</p></div>;
  if (error) return <div className="p-6 text-red-600">Error: {error}</div>;

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Session Revocation Center</h1>
        <p className="text-gray-600">Revoke active user sessions by user, tenant, or in bulk.</p>
      </div>

      <div className="grid grid-cols-3 gap-4">
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold">{sessions.length}</div>
          <div className="text-sm text-gray-500">Active Sessions</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold">{new Set(sessions.map(s => s.userId)).size}</div>
          <div className="text-sm text-gray-500">Users Online</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold">{new Set(sessions.map(s => s.tenant)).size}</div>
          <div className="text-sm text-gray-500">Tenants</div>
        </div>
      </div>

      <section className="bg-white rounded-lg shadow p-4">
        <input type="text" placeholder="Revocation reason (optional)" value={reason} onChange={e => setReason(e.target.value)} className="w-full border rounded px-3 py-2 text-sm" />
      </section>

      {bulkConfirm ? (
        <div className="flex items-center gap-3 bg-red-50 rounded p-3">
          <span className="text-sm">Revoke ALL {sessions.length} sessions?</span>
          <button onClick={bulkRevoke} className="px-3 py-1 bg-red-600 text-white rounded text-sm">Confirm</button>
          <button onClick={() => setBulkConfirm(false)} className="px-3 py-1 border rounded text-sm">Cancel</button>
        </div>
      ) : (
        <button onClick={() => setBulkConfirm(true)} disabled={sessions.length === 0} className="px-4 py-2 bg-red-600 text-white rounded text-sm disabled:opacity-50">Bulk Revoke All</button>
      )}

      <section className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th className="p-3">User</th>
              <th className="p-3">Device</th>
              <th className="p-3">IP</th>
              <th className="p-3">Created</th>
              <th className="p-3">Last Active</th>
              <th className="p-3">Tenant</th>
              <th className="p-3">Actions</th>
            </tr>
          </thead>
          <tbody>
            {sessions.map(s => (
              <tr key={s.id} className="border-b hover:bg-gray-50">
                <td className="p-3 font-medium">{s.userId}</td>
                <td className="p-3 text-gray-600">{s.device}</td>
                <td className="p-3 font-mono text-xs">{s.ip}</td>
                <td className="p-3 text-gray-500">{s.created}</td>
                <td className="p-3 text-gray-500">{s.lastActive}</td>
                <td className="p-3"><span className="px-2 py-0.5 bg-blue-100 text-blue-700 rounded text-xs">{s.tenant}</span></td>
                <td className="p-3">
                  <div className="flex gap-2">
                    <button onClick={() => revoke(s.id, s.userId)} aria-label={`Revoke session for ${s.userId}`} className="text-red-600 text-xs hover:underline">Revoke</button>
                    <button onClick={() => revokeAllForUser(s.userId)} aria-label={`Revoke all sessions for ${s.userId}`} className="text-amber-600 text-xs hover:underline">Revoke All User</button>
                    <button onClick={() => revokeByTenant(s.tenant)} aria-label={`Revoke all sessions for tenant ${s.tenant}`} className="text-purple-600 text-xs hover:underline">Revoke Tenant</button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Revocation Audit Log</h2>
        {auditLog.length === 0 ? (
          <p className="text-sm text-gray-400">No revocations yet.</p>
        ) : (
          <div className="space-y-2">
            {auditLog.map((log, idx) => (
              <div key={idx} className="text-sm border-l-2 border-red-400 pl-3 py-1">{log}</div>
            ))}
          </div>
        )}
      </section>
    </div>
  );
}
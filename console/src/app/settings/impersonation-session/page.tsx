'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";

interface ImpSession { id: string; impersonator: string; target: string; startedAt: string; expires: string; reason: string; }

export default function ImpersonationSessionPage() {
  const t = useTranslations();

  const [sessions, setSessions] = useState<ImpSession[]>([]);
  const [showForm, setShowForm] = useState(false);
  const [newSession, setNewSession] = useState({ target: '', reason: '', duration: 30 });
  const [auditLog, setAuditLog] = useState([] as { time: string; action: string; target: string }[]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch('/api/v1/auth/impersonate', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': '00000000-0000-0000-0000-000000000001' },
      body: JSON.stringify({ action: 'list' }),
    })
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(data => {
        if (data) {
          if (data.sessions) setSessions(data.sessions);
          if (data.audit_log) setAuditLog(data.audit_log);
        }
        setLoading(false);
      })
      .catch(e => { setError(e.message); setLoading(false); });
  }, []);

  const start = () => {
    setSessions(prev => [...prev, { id: `s${prev.length + 1}`, impersonator: 'admin@ggid.io', target: newSession.target, startedAt: new Date().toISOString().slice(11, 16), expires: `${new Date(Date.now() + newSession.duration * 60000).toISOString().slice(11, 16)}`, reason: newSession.reason }]);
    setShowForm(false); setNewSession({ target: '', reason: '', duration: 30 });
  };
  const endSession = (id: string) => setSessions(prev => prev.filter(s => s.id !== id));

  if (loading) return <div className="p-6"><p>Loading...</p></div>;
  if (error) return <div className="p-6 text-red-600">Error: {error}</div>;

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold">Impersonation Sessions</h1><p className="text-gray-600">Monitor and manage active admin impersonation sessions.</p></div>
        <button onClick={() => setShowForm(!showForm)} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">{showForm ? 'Cancel' : 'Start Session'}</button>
      </div>

      <section className="bg-amber-50 border border-amber-200 rounded p-4 space-y-2">
        <h2 className="text-sm font-semibold text-amber-800">Safety Guardrails</h2>
        <div className="grid grid-cols-2 gap-2 text-xs text-amber-700">
          <div>Max duration: 60 minutes</div><div>Reason required: yes</div>
          <div>Audit level: full</div><div>Auto-revoke on idle: 15min</div>
          <div>Restricted to roles: admin, security-admin</div><div>Target consent: optional</div>
        </div>
      </section>

      {showForm && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Start Impersonation</h2>
          <div><label className="text-sm font-medium">Target User</label><input type="text" placeholder="user@ggid.io" value={newSession.target} onChange={e => setNewSession(prev => ({ ...prev, target: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
          <div><label className="text-sm font-medium">Reason</label><input type="text" placeholder="Why are you impersonating?" value={newSession.reason} onChange={e => setNewSession(prev => ({ ...prev, reason: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
          <div><label className="text-sm font-medium">Duration (minutes)</label><input type="number" min={5} max={60} value={newSession.duration} onChange={e => setNewSession(prev => ({ ...prev, duration: parseInt(e.target.value) || 30 }))} className="w-24 border rounded px-2 py-1 text-sm mt-1" /></div>
          <button onClick={start} disabled={!newSession.target || !newSession.reason} className="px-4 py-2 bg-blue-600 text-white rounded text-sm disabled:opacity-50">Start</button>
        </section>
      )}

      <section className="bg-white rounded-lg shadow overflow-hidden">
        <h2 className="text-lg font-semibold p-6 pb-4">Active Sessions</h2>
        {sessions.length === 0 ? <p className="text-sm text-gray-400 px-6 pb-6">No active impersonation sessions.</p> : (
          <table className="w-full text-sm"><thead className="bg-gray-50"><tr className="text-left"><th className="p-3">Impersonator</th><th className="p-3">Target</th><th className="p-3">Started</th><th className="p-3">Expires</th><th className="p-3">Reason</th><th className="p-3">Action</th></tr></thead>
            <tbody>{sessions.map(s => (
              <tr key={s.id} className="border-b"><td className="p-3 font-medium">{s.impersonator}</td><td className="p-3">{s.target}</td><td className="p-3 text-gray-500">{s.startedAt}</td><td className="p-3 text-gray-500">{s.expires}</td><td className="p-3 text-gray-600">{s.reason}</td><td className="p-3"><button onClick={() => endSession(s.id)} className="text-red-600 text-xs hover:underline">End</button></td></tr>
            ))}</tbody></table>
        )}
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Session Audit Log</h2>
        <div className="space-y-2">{auditLog.map((l, i) => <div key={i} className="flex items-center gap-3 text-sm border-b pb-2"><span className="text-xs text-gray-500">{l.time}</span><span className="font-mono text-xs">{l.action}</span><span className="text-xs text-gray-500">{l.target}</span></div>)}</div>
      </section>
    </div>
  );
}
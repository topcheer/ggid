'use client';
import { useState, useEffect } from 'react';

interface ImpersonationRecord {
  id: string;
  impersonator: string;
  target: string;
  reason: string;
  startedAt: string;
  duration: string;
  status: string;
}

export default function ImpersonationConfigPage() {
  const [allowedImpersonators, setAllowedImpersonators] = useState<string[]>([]);
  const [requireReason, setRequireReason] = useState(true);
  const [maxDuration, setMaxDuration] = useState(60);
  const [auditLevel, setAuditLevel] = useState('full');
  const [requireTargetConsent, setRequireTargetConsent] = useState(false);
  const [autoRevokeOnIdle, setAutoRevokeOnIdle] = useState(true);
  const [restrictToRoles, setRestrictToRoles] = useState<string[]>([]);
  const [newImpersonator, setNewImpersonator] = useState('');
  const [newRole, setNewRole] = useState('');
  const [history, setHistory] = useState<ImpersonationRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch('/api/v1/auth/impersonation/config', {
      headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': '00000000-0000-0000-0000-000000000001' },
    })
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(data => {
        if (data) {
          if (data.allowed_impersonators) setAllowedImpersonators(data.allowed_impersonators);
          if (data.require_reason !== undefined) setRequireReason(data.require_reason);
          if (data.max_duration) setMaxDuration(data.max_duration);
          if (data.audit_level) setAuditLevel(data.audit_level);
          if (data.require_target_consent !== undefined) setRequireTargetConsent(data.require_target_consent);
          if (data.auto_revoke_on_idle !== undefined) setAutoRevokeOnIdle(data.auto_revoke_on_idle);
          if (data.restrict_to_roles) setRestrictToRoles(data.restrict_to_roles);
          if (data.history) setHistory(data.history);
        }
        setLoading(false);
      })
      .catch(e => { setError(e.message); setLoading(false); });
  }, []);

  const addImpersonator = () => { if (newImpersonator) { setAllowedImpersonators(prev => [...prev, newImpersonator]); setNewImpersonator(''); } };
  const removeImpersonator = (u: string) => setAllowedImpersonators(prev => prev.filter(x => x !== u));
  const addRole = () => { if (newRole) { setRestrictToRoles(prev => [...prev, newRole]); setNewRole(''); } };
  const removeRole = (r: string) => setRestrictToRoles(prev => prev.filter(x => x !== r));

  const statusColor = (s: string): string =>
    s === 'active' ? 'bg-green-100 text-green-700' : 'bg-gray-200 text-gray-600';

  if (loading) return <div className="p-6"><p>Loading...</p></div>;
  if (error) return <div className="p-6 text-red-600">Error: {error}</div>;

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Impersonation Configuration</h1>
        <p className="text-gray-600">Configure admin impersonation policies, restrictions, and audit trail.</p>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <label className="flex items-center justify-between bg-white rounded-lg shadow p-4">
          <span className="text-sm font-medium">Require Reason</span>
          <input type="checkbox" checked={requireReason} onChange={e => setRequireReason(e.target.checked)} className="rounded" />
        </label>
        <label className="flex items-center justify-between bg-white rounded-lg shadow p-4">
          <span className="text-sm font-medium">Require Target Consent</span>
          <input type="checkbox" checked={requireTargetConsent} onChange={e => setRequireTargetConsent(e.target.checked)} className="rounded" />
        </label>
        <label className="flex items-center justify-between bg-white rounded-lg shadow p-4">
          <span className="text-sm font-medium">Auto-Revoke on Idle</span>
          <input type="checkbox" checked={autoRevokeOnIdle} onChange={e => setAutoRevokeOnIdle(e.target.checked)} className="rounded" />
        </label>
        <div className="bg-white rounded-lg shadow p-4">
          <label className="text-sm font-medium">Audit Level</label>
          <select value={auditLevel} onChange={e => setAuditLevel(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1">
            <option value="full">Full (all actions logged)</option>
            <option value="summary">Summary (start/end only)</option>
            <option value="disabled">Disabled (not recommended)</option>
          </select>
        </div>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Max Duration: {maxDuration} minutes</h2>
        <input type="range" min={5} max={480} step={5} value={maxDuration} onChange={e => setMaxDuration(parseInt(e.target.value))} className="w-full" />
        <div className="flex justify-between text-xs text-gray-400"><span>5min</span><span>8h</span></div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Allowed Impersonators</h2>
        <div className="space-y-2">
          {allowedImpersonators.map(u => (
            <div key={u} className="flex items-center gap-2"><span className="font-mono text-sm flex-1">{u}</span><button onClick={() => removeImpersonator(u)}aria-label={"Remove impersonator " + u} className="text-red-600 text-xs">Remove</button></div>
          ))}
        </div>
        <div className="flex gap-2">
          <input type="text" placeholder="user@ggid.io" value={newImpersonator} onChange={e => setNewImpersonator(e.target.value)} className="flex-1 border rounded px-2 py-1 text-sm" />
          <button onClick={addImpersonator} aria-label="Add impersonator" className="px-3 py-1 bg-blue-600 text-white rounded text-sm">Add</button>
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Restricted to Roles</h2>
        <p className="text-sm text-gray-500">Only users with these roles can be impersonated.</p>
        <div className="flex flex-wrap gap-2">
          {restrictToRoles.map(r => (
            <div key={r} className="flex items-center gap-1"><span className="px-2 py-1 bg-purple-100 text-purple-700 rounded text-xs">{r}</span><button onClick={() => removeRole(r)}aria-label={"Remove role " + r} className="text-red-600 text-xs">x</button></div>
          ))}
        </div>
        <div className="flex gap-2">
          <input type="text" placeholder="role-name" value={newRole} onChange={e => setNewRole(e.target.value)} className="flex-1 border rounded px-2 py-1 text-sm" />
          <button onClick={addRole} aria-label="Add restricted role" className="px-3 py-1 bg-blue-600 text-white rounded text-sm">Add</button>
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Impersonation History</h2>
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th className="p-3">Impersonator</th><th className="p-3">Target</th><th className="p-3">Reason</th><th className="p-3">Started</th><th className="p-3">Duration</th><th className="p-3">Status</th>
            </tr>
          </thead>
          <tbody>
            {history.map(r => (
              <tr key={r.id} className="border-b">
                <td className="p-3 font-medium">{r.impersonator}</td>
                <td className="p-3">{r.target}</td>
                <td className="p-3 text-gray-600">{r.reason}</td>
                <td className="p-3 text-gray-500">{r.startedAt}</td>
                <td className="p-3 text-gray-500">{r.duration}</td>
                <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs ${statusColor(r.status)}`}>{r.status}</span></td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </div>
  );
}
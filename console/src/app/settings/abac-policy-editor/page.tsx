'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";

interface Policy { id: string; name: string; subject: string; resource: string; action: string; effect: string; priority: number; enabled: boolean; }

export default function AbacPolicyEditorPage() {
  const t = useTranslations();

  const [policies, setPolicies] = useState<Policy[]>([]);
  const [showForm, setShowForm] = useState(false);
  const [newPolicy, setNewPolicy] = useState({ name: '', subject: '', resource: '', action: '', effect: 'allow' });
  const [simSubject, setSimSubject] = useState('role=admin');
  const [simResource, setSimResource] = useState('user/alice');
  const [simAction, setSimAction] = useState('read');
  const [simResult, setSimResult] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch('/api/v1/policies/abac/groups', {
      headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': '00000000-0000-0000-0000-000000000001' },
    })
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(data => {
        if (data && data.policies) setPolicies(data.policies);
        else if (Array.isArray(data)) setPolicies(data);
        setLoading(false);
      })
      .catch(e => { setError(e.message); setLoading(false); });
  }, []);

  const addPolicy = () => {
    setPolicies(prev => [...prev, { id: `p${prev.length + 1}`, name: newPolicy.name, subject: newPolicy.subject, resource: newPolicy.resource, action: newPolicy.action, effect: newPolicy.effect, priority: prev.length + 1, enabled: true }]);
    setShowForm(false); setNewPolicy({ name: '', subject: '', resource: '', action: '', effect: 'allow' });
  };
  const togglePolicy = (id: string) => setPolicies(prev => prev.map(p => p.id === id ? { ...p, enabled: !p.enabled } : p));
  const movePriority = (id: string, dir: 'up' | 'down') => {
    setPolicies(prev => {
      const idx = prev.findIndex(p => p.id === id);
      const target = dir === 'up' ? idx - 1 : idx + 1;
      if (target < 0 || target >= prev.length) return prev;
      const next = [...prev];
      [next[idx].priority, next[target].priority] = [next[target].priority, next[idx].priority];
      return next.sort((a, b) => a.priority - b.priority);
    });
  };
  const runSim = () => {
    const match = policies.find(p => p.enabled && (p.subject === simSubject || p.subject === '*') && (p.resource === simResource || p.resource === '*' || simResource.startsWith(p.resource.replace('*', ''))));
    setSimResult(match ? `${match.effect.toUpperCase()} — matched policy: ${match.name}` : 'DENY — no matching policy');
  };

  const effectColor = (e: string) => e === 'allow' ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700';

  if (loading) return <div className="p-6"><p>Loading...</p></div>;
  if (error) return <div className="p-6 text-red-600">Error: {error}</div>;

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold">ABAC Policy Editor</h1><p className="text-gray-600">Attribute-based access control policies with condition builder and simulator.</p></div>
        <button onClick={() => setShowForm(!showForm)} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">{showForm ? 'Cancel' : 'Create Policy'}</button>
      </div>

      {showForm && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Create Policy</h2>
          <div><label className="text-sm font-medium">Policy Name</label><input aria-label="new Policy" type="text" value={newPolicy.name} onChange={e => setNewPolicy(prev => ({ ...prev, name: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
          <div className="grid grid-cols-2 gap-4">
            <div><label className="text-sm font-medium">Subject (attribute=value)</label><input aria-label="role=admin" type="text" placeholder="role=admin" value={newPolicy.subject} onChange={e => setNewPolicy(prev => ({ ...prev, subject: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1 font-mono" /></div>
            <div><label className="text-sm font-medium">Resource</label><input aria-label="user/*" type="text" placeholder="user/*" value={newPolicy.resource} onChange={e => setNewPolicy(prev => ({ ...prev, resource: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1 font-mono" /></div>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div><label className="text-sm font-medium">Action</label><input aria-label="read" type="text" placeholder="read" value={newPolicy.action} onChange={e => setNewPolicy(prev => ({ ...prev, action: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1 font-mono" /></div>
            <div><label className="text-sm font-medium">Effect</label><select aria-label="new Policy" value={newPolicy.effect} onChange={e => setNewPolicy(prev => ({ ...prev, effect: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1"><option value="allow">Allow</option><option value="deny">Deny</option></select></div>
          </div>
          <button onClick={addPolicy} disabled={!newPolicy.name} className="px-4 py-2 bg-blue-600 text-white rounded text-sm disabled:opacity-50">Create</button>
        </section>
      )}

      <section className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50"><tr className="text-left"><th className="p-3">Priority</th><th className="p-3">Name</th><th className="p-3">Subject</th><th className="p-3">Resource</th><th className="p-3">Action</th><th className="p-3">Effect</th><th className="p-3">Enabled</th><th className="p-3">Order</th></tr></thead>
          <tbody>
            {policies.map(p => (
              <tr key={p.id} className="border-b">
                <td className="p-3">{p.priority}</td>
                <td className="p-3 font-medium">{p.name}</td>
                <td className="p-3 font-mono text-xs">{p.subject}</td>
                <td className="p-3 font-mono text-xs">{p.resource}</td>
                <td className="p-3 font-mono text-xs">{p.action}</td>
                <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs ${effectColor(p.effect)}`}>{p.effect}</span></td>
                <td className="p-3"><input aria-label="Toggle" type="checkbox" checked={p.enabled} onChange={() => togglePolicy(p.id)} className="rounded" /></td>
                <td className="p-3"><div className="flex gap-1"><button onClick={() => movePriority(p.id, 'up')} className="text-xs text-gray-400">up</button><button onClick={() => movePriority(p.id, 'down')} className="text-xs text-gray-400">down</button></div></td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Policy Simulator</h2>
        <div className="grid grid-cols-3 gap-4">
          <div><label className="text-sm font-medium">Subject</label><input aria-label="sim Subject" type="text" value={simSubject} onChange={e => setSimSubject(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1 font-mono" /></div>
          <div><label className="text-sm font-medium">Resource</label><input aria-label="sim Resource" type="text" value={simResource} onChange={e => setSimResource(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1 font-mono" /></div>
          <div><label className="text-sm font-medium">Action</label><input aria-label="sim Action" type="text" value={simAction} onChange={e => setSimAction(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1 font-mono" /></div>
        </div>
        <button onClick={runSim} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">Evaluate</button>
        {simResult && <div className={`text-sm p-3 rounded ${simResult.startsWith('ALLOW') ? 'bg-green-50 text-green-700' : 'bg-red-50 text-red-700'}`}>{simResult}</div>}
      </section>
    </div>
  );
}
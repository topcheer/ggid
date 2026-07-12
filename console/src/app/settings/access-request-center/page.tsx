'use client';
import { useState, useEffect } from 'react';

interface AccessRequest { id: string; requester: string; resource: string; role: string; justification: string; status: string; submittedAt: string; sla: string; }

export default function AccessRequestCenterPage() {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [requests, setRequests] = useState<AccessRequest[]>([]);
  const [showForm, setShowForm] = useState(false);
  const [newReq, setNewReq] = useState({ resource: '', role: '', justification: '', duration: 24 });
  const [autoApproveLow, setAutoApproveLow] = useState(true);
  const [slaHours, setSlaHours] = useState(48);

  useEffect(() => {
    fetch("/api/v1/policies/access-requests", {
      headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
    })
      .then(res => { if (!res.ok) throw new Error(`HTTP ${res.status}`); return res.json(); })
      .then(data => { setRequests(Array.isArray(data) ? data : (data.requests || data.items || [])); setLoading(false); })
      .catch(err => { setError(err.message); setLoading(false); });
  }, []);

  const submit = () => {
    setRequests(prev => [...prev, { id: `ar${prev.length + 1}`, requester: 'current@ggid.io', resource: newReq.resource, role: newReq.role, justification: newReq.justification, status: 'pending', submittedAt: new Date().toISOString().slice(0, 16).replace('T', ' '), sla: `${slaHours}h` }]);
    setShowForm(false); setNewReq({ resource: '', role: '', justification: '', duration: 24 });
  };
  const approve = (id: string) => setRequests(prev => prev.map(r => r.id === id ? { ...r, status: 'approved' } : r));
  const reject = (id: string) => setRequests(prev => prev.map(r => r.id === id ? { ...r, status: 'rejected' } : r));

  const statusColor = (s: string) => s === 'approved' ? 'bg-green-100 text-green-700' : s === 'rejected' ? 'bg-red-100 text-red-700' : s === 'pending' ? 'bg-amber-100 text-amber-700' : 'bg-gray-100 text-gray-600';

  if (loading) return (
    <div className="p-6"><h1 className="text-2xl font-bold mb-4">Access Request Center</h1><p>Loading...</p></div>
  );
  if (error) return (
    <div className="p-6"><h1 className="text-2xl font-bold mb-4">Access Request Center</h1><p className="text-red-600">Error: {error}</p></div>
  );
  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold">Access Request Center</h1><p className="text-gray-600">Submit, approve, and track access requests with SLA monitoring.</p></div>
        <button onClick={() => setShowForm(!showForm)} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">{showForm ? 'Cancel' : 'Submit Request'}</button>
      </div>

      {showForm && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Submit Access Request</h2>
          <div className="grid grid-cols-2 gap-4">
            <div><label className="text-sm font-medium">Resource</label><input type="text" placeholder="production-db" value={newReq.resource} onChange={e => setNewReq(prev => ({ ...prev, resource: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
            <div><label className="text-sm font-medium">Role</label><input type="text" placeholder="read-only" value={newReq.role} onChange={e => setNewReq(prev => ({ ...prev, role: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
          </div>
          <div><label className="text-sm font-medium">Justification</label><textarea value={newReq.justification} onChange={e => setNewReq(prev => ({ ...prev, justification: e.target.value }))} rows={2} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
          <div><label className="text-sm font-medium">Duration (hours)</label><input type="number" min={1} max={720} value={newReq.duration} onChange={e => setNewReq(prev => ({ ...prev, duration: parseInt(e.target.value) || 24 }))} className="w-24 border rounded px-2 py-1 text-sm mt-1" /></div>
          <button onClick={submit} disabled={!newReq.resource || !newReq.justification} className="px-4 py-2 bg-blue-600 text-white rounded text-sm disabled:opacity-50">Submit</button>
        </section>
      )}

      <div className="grid grid-cols-2 gap-4">
        <label className="flex items-center justify-between bg-white rounded-lg shadow p-4"><span className="text-sm font-medium">Auto-approve low-risk requests</span><input type="checkbox" checked={autoApproveLow} onChange={e => setAutoApproveLow(e.target.checked)} className="rounded" /></label>
        <div className="bg-white rounded-lg shadow p-4"><label className="text-sm font-medium">SLA Target (hours)</label><input type="number" min={1} max={168} value={slaHours} onChange={e => setSlaHours(parseInt(e.target.value) || 48)} className="w-20 border rounded px-2 py-1 text-sm mt-1" /></div>
      </div>

      <section className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full text-sm"><thead className="bg-gray-50"><tr className="text-left"><th className="p-3">Requester</th><th className="p-3">Resource</th><th className="p-3">Role</th><th className="p-3">Justification</th><th className="p-3">Status</th><th className="p-3">SLA</th><th className="p-3">Actions</th></tr></thead>
          <tbody>{requests.map(r => (
            <tr key={r.id} className="border-b">
              <td className="p-3 font-medium">{r.requester}</td><td className="p-3 font-mono text-xs">{r.resource}</td><td className="p-3 text-xs">{r.role}</td><td className="p-3 text-gray-600 text-xs max-w-xs truncate">{r.justification}</td>
              <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs ${statusColor(r.status)}`}>{r.status}</span></td><td className="p-3 text-xs text-gray-500">{r.sla}</td>
              <td className="p-3">{r.status === 'pending' && <div className="flex gap-2"><button onClick={() => approve(r.id)} className="text-green-600 text-xs hover:underline">Approve</button><button onClick={() => reject(r.id)} className="text-red-600 text-xs hover:underline">Reject</button></div>}</td>
            </tr>))}</tbody></table>
      </section>
    </div>
  );
}
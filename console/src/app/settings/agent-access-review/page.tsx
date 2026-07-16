'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface Review {
  id: string;
  agentId: string;
  agentName: string;
  reviewer: string;
  decision: string;
  timestamp: string;
  scopes: string[];
  comment: string;
  drift: boolean;
}

export default function AgentAccessReviewPage() {
  const t = useTranslations();

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [reviews, setReviews] = useState<Review[]>([]);

  useEffect(() => {
    fetch("/api/v1/identity/nhi", {
      headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
    })
      .then(res => { if (!res.ok) return null; return res.json(); })
      .then(data => { setReviews(Array.isArray(data) ? data : (data.reviews || data.items || [])); setLoading(false); })
      .catch(err => { setError(err.message); setLoading(false); });
  }, []);

  const [showForm, setShowForm] = useState(false);
  const [selectedAgent, setSelectedAgent] = useState('');
  const [selectedScopes, setSelectedScopes] = useState<string[]>([]);
  const [decision, setDecision] = useState('approve');
  const [comment, setComment] = useState('');
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());

  const allScopes = ['read:users', 'write:users', 'read:orgs', 'write:orgs', 'read:audit', 'write:audit', 'admin:all', 'read:policy'];

  const toggleScope = (scope: string) => {
    setSelectedScopes(prev => prev.includes(scope) ? prev.filter(s => s !== scope) : [...prev, scope]);
  };

  const submitReview = () => {
    const newReview: Review = {
      id: `r${reviews.length + 1}`,
      agentId: selectedAgent || 'agent-new',
      agentName: `Agent ${selectedAgent || 'New'}`,
      reviewer: 'current-user@ggid.io',
      decision,
      timestamp: new Date().toISOString().slice(0, 16).replace('T', ' '),
      scopes: selectedScopes,
      comment,
      drift: false,
    };
    setReviews(prev => [newReview, ...prev]);
    setShowForm(false);
    setSelectedAgent('');
    setSelectedScopes([]);
    setDecision('approve');
    setComment('');
  };

  const toggleSelect = (id: string) => {
    setSelectedIds(prev => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id); else next.add(id);
      return next;
    });
  };

  const bulkApprove = () => {
    setReviews(prev => prev.map(r => selectedIds.has(r.id) ? { ...r, decision: 'approved' } : r));
    setSelectedIds(new Set());
  };

  const decisionColor = (d: string) =>
    d === 'approved' ? 'bg-green-100 text-green-700' :
    d === 'rejected' ? 'bg-red-100 text-red-700' :
    'bg-amber-100 text-amber-700';

  if (loading) return (
    <div className="p-6"><h1 className="text-2xl font-bold mb-4">Agent Access Review</h1><p>Loading...</p></div>
  );
  if (error) return (
    <div className="p-6"><h1 className="text-2xl font-bold mb-4">Agent Access Review</h1><p className="text-red-600">Error: {error}</p></div>
  );
  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Agent Access Review</h1>
          <p className="text-gray-600">Review and manage AI agent access scopes and permissions.</p>
        </div>
        <button onClick={() => setShowForm(!showForm)} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">
          {showForm ? 'Cancel' : 'New Review'}
        </button>
      </div>

      {showForm && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Create Review</h2>
          <div>
            <label className="text-sm font-medium">Agent</label>
            <input
              type="text"
              placeholder="Agent ID or name"
              value={selectedAgent}
              onChange={e => setSelectedAgent(e.target.value)}
              className="w-full border rounded px-3 py-2 text-sm mt-1"
            />
          </div>
          <div>
            <label className="text-sm font-medium">Scopes</label>
            <div className="flex flex-wrap gap-2 mt-2">
              {allScopes.map(s => (
                <label key={s} className="flex items-center gap-1 text-sm">
                  <input aria-label="Selected scopes" type="checkbox" checked={selectedScopes.includes(s)} onChange={() => toggleScope(s)} className="rounded" />
                  {s}
                </label>
              ))}
            </div>
          </div>
          <div>
            <label className="text-sm font-medium">Decision</label>
            <div className="flex gap-4 mt-2">
              {['approve', 'reject', 'revoke'].map(d => (
                <label key={d} className="flex items-center gap-2 text-sm">
                  <input aria-label="Decision" type="radio" checked={decision === d} onChange={() => setDecision(d)} />
                  <span className="capitalize">{d}</span>
                </label>
              ))}
            </div>
          </div>
          <div>
            <label className="text-sm font-medium">Comment</label>
            <textarea
              value={comment}
              onChange={e => setComment(e.target.value)}
              rows={2}
              className="w-full border rounded px-3 py-2 text-sm mt-1"
            />
          </div>
          <button onClick={submitReview} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">Submit Review</button>
        </section>
      )}

      {selectedIds.size > 0 && (
        <div className="flex items-center gap-3 bg-blue-50 rounded p-3">
          <span className="text-sm">{selectedIds.size} selected</span>
          <button onClick={bulkApprove} className="px-3 py-1 bg-green-600 text-white rounded text-sm">Bulk Approve</button>
          <button onClick={() => setSelectedIds(new Set())} className="px-3 py-1 border rounded text-sm">Clear</button>
        </div>
      )}

      <section className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th scope="col" className="p-3"></th>
              <th scope="col" className="p-3">Agent</th>
              <th scope="col" className="p-3">Reviewer</th>
              <th scope="col" className="p-3">Decision</th>
              <th scope="col" className="p-3">Scopes</th>
              <th scope="col" className="p-3">Timestamp</th>
              <th scope="col" className="p-3">Drift</th>
            </tr>
          </thead>
          <tbody>
            {reviews.map(r => (
              <tr key={r.id} className="border-b hover:bg-gray-50">
                <td className="p-3"><input aria-label="Toggle" type="checkbox" checked={selectedIds.has(r.id)} onChange={() => toggleSelect(r.id)} className="rounded" /></td>
                <td className="p-3">
                  <div className="font-mono text-xs text-gray-500">{r.agentId}</div>
                  <div className="font-medium">{r.agentName}</div>
                </td>
                <td className="p-3 text-gray-600">{r.reviewer}</td>
                <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs capitalize ${decisionColor(r.decision)}`}>{r.decision}</span></td>
                <td className="p-3"><div className="flex flex-wrap gap-1">{r.scopes.map(s => <span key={s} className="px-1.5 py-0.5 bg-gray-100 rounded text-xs">{s}</span>)}</div></td>
                <td className="p-3 text-gray-500 text-xs">{r.timestamp}</td>
                <td className="p-3">{r.drift && <span className="px-2 py-0.5 bg-red-100 text-red-700 rounded text-xs">Drift</span>}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <section className="bg-white rounded-lg shadow p-6">
        <h2 className="text-lg font-semibold mb-4">Review History Timeline</h2>
        <div className="space-y-3">
          {reviews.map(r => (
            <div key={r.id} className="flex gap-4 items-start">
              <div className={`w-3 h-3 rounded-full mt-1.5 ${r.decision === 'approved' ? 'bg-green-500' : r.decision === 'rejected' ? 'bg-red-500' : 'bg-amber-500'}`} />
              <div className="flex-1">
                <div className="text-sm font-medium">{r.agentName} <span className={`px-2 py-0.5 rounded text-xs capitalize ${decisionColor(r.decision)}`}>{r.decision}</span></div>
                <div className="text-xs text-gray-500">{r.reviewer} - {r.timestamp}</div>
                {r.comment && <div className="text-sm text-gray-600 mt-1">{r.comment}</div>}
              </div>
            </div>
          ))}
        </div>
      </section>
    </div>
  );
}

'use client';
import { useState } from 'react';

interface Review {
  id: string;
  user: string;
  reviewer: string;
  roles: string[];
  scopes: string[];
  decision: string;
  timestamp: string;
  overdue: boolean;
}

export default function AccessReviewCenterPage() {
  const [reviews, setReviews] = useState<Review[]>([
    { id: 'r1', user: 'alice@ggid.io', reviewer: 'manager1@ggid.io', roles: ['admin', 'developer'], scopes: ['admin:all', 'read:audit'], decision: 'pending', timestamp: '2026-07-01', overdue: true },
    { id: 'r2', user: 'bob@ggid.io', reviewer: 'manager2@ggid.io', roles: ['developer'], scopes: ['write:users'], decision: 'approved', timestamp: '2026-07-05', overdue: false },
    { id: 'r3', user: 'carol@ggid.io', reviewer: 'manager1@ggid.io', roles: ['auditor'], scopes: ['read:audit', 'read:users'], decision: 'pending', timestamp: '2026-07-08', overdue: false },
    { id: 'r4', user: 'dave@ggid.io', reviewer: 'manager3@ggid.io', roles: ['finance', 'operations'], scopes: ['write:orgs'], decision: 'rejected', timestamp: '2026-06-28', overdue: false },
  ]);

  const [showForm, setShowForm] = useState(false);
  const [scheduledReview, setScheduledReview] = useState(true);
  const [reviewFrequency, setReviewFrequency] = useState('quarterly');
  const [newReview, setNewReview] = useState({ user: '', reviewer: '', scopes: [] as string[] });
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());

  const allScopes = ['admin:all', 'read:users', 'write:users', 'read:orgs', 'write:orgs', 'read:audit', 'write:audit', 'read:policy'];

  const decisionColor = (d: string): string =>
    d === 'approved' ? 'bg-green-100 text-green-700' :
    d === 'rejected' ? 'bg-red-100 text-red-700' :
    d === 'revoked' ? 'bg-amber-100 text-amber-700' :
    'bg-blue-100 text-blue-700';

  const toggleScope = (scope: string) => {
    setNewReview(prev => ({ ...prev, scopes: prev.scopes.includes(scope) ? prev.scopes.filter(s => s !== scope) : [...prev.scopes, scope] }));
  };

  const createReview = () => {
    setReviews(prev => [...prev, {
      id: `r${prev.length + 1}`,
      user: newReview.user || 'unknown@ggid.io',
      reviewer: newReview.reviewer || 'admin@ggid.io',
      roles: [],
      scopes: newReview.scopes,
      decision: 'pending',
      timestamp: new Date().toISOString().slice(0, 10),
      overdue: false,
    }]);
    setShowForm(false);
    setNewReview({ user: '', reviewer: '', scopes: [] });
  };

  const setDecision = (id: string, decision: string) => {
    setReviews(prev => prev.map(r => r.id === id ? { ...r, decision } : r));
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

  const pendingCount = reviews.filter(r => r.decision === 'pending').length;
  const overdueCount = reviews.filter(r => r.overdue).length;

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Access Review Center</h1>
          <p className="text-gray-600">Certify user access, manage review schedules, and track overdue reviews.</p>
        </div>
        <button onClick={() => setShowForm(!showForm)} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">
          {showForm ? 'Cancel' : 'Create Review'}
        </button>
      </div>

      {showForm && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Create Access Review</h2>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="text-sm font-medium">User</label>
              <input type="text" placeholder="user@ggid.io" value={newReview.user} onChange={e => setNewReview(prev => ({ ...prev, user: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" />
            </div>
            <div>
              <label className="text-sm font-medium">Reviewer</label>
              <input type="text" placeholder="reviewer@ggid.io" value={newReview.reviewer} onChange={e => setNewReview(prev => ({ ...prev, reviewer: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" />
            </div>
          </div>
          <div>
            <label className="text-sm font-medium">Scopes to Review</label>
            <div className="flex flex-wrap gap-2 mt-2">
              {allScopes.map(s => (
                <label key={s} className="flex items-center gap-1 text-sm">
                  <input type="checkbox" checked={newReview.scopes.includes(s)} onChange={() => toggleScope(s)} className="rounded" />
                  {s}
                </label>
              ))}
            </div>
          </div>
          <button onClick={createReview} disabled={!newReview.user} className="px-4 py-2 bg-blue-600 text-white rounded text-sm disabled:opacity-50">Create Review</button>
        </section>
      )}

      <div className="grid grid-cols-3 gap-4">
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold">{reviews.length}</div>
          <div className="text-sm text-gray-500">Total Reviews</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold text-blue-600">{pendingCount}</div>
          <div className="text-sm text-gray-500">Pending</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold text-red-600">{overdueCount}</div>
          <div className="text-sm text-gray-500">Overdue</div>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <label className="flex items-center justify-between bg-white rounded-lg shadow p-4">
          <span className="text-sm font-medium">Scheduled Reviews</span>
          <input type="checkbox" checked={scheduledReview} onChange={e => setScheduledReview(e.target.checked)} className="rounded" />
        </label>
        <section className="bg-white rounded-lg shadow p-4">
          <label className="text-sm font-medium">Review Frequency</label>
          <select value={reviewFrequency} onChange={e => setReviewFrequency(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1">
            <option value="monthly">Monthly</option>
            <option value="quarterly">Quarterly</option>
            <option value="semiannual">Semi-Annual</option>
            <option value="annual">Annual</option>
          </select>
        </section>
      </div>

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
              <th className="p-3"></th>
              <th className="p-3">User</th>
              <th className="p-3">Reviewer</th>
              <th className="p-3">Scopes</th>
              <th className="p-3">Decision</th>
              <th className="p-3">Date</th>
              <th className="p-3">Actions</th>
            </tr>
          </thead>
          <tbody>
            {reviews.map(r => (
              <tr key={r.id} className="border-b hover:bg-gray-50">
                <td className="p-3"><input type="checkbox" checked={selectedIds.has(r.id)} onChange={() => toggleSelect(r.id)} className="rounded" /></td>
                <td className="p-3 font-medium">
                  {r.user}
                  {r.overdue && <span className="ml-1 px-1.5 py-0.5 bg-red-100 text-red-700 rounded text-xs">Overdue</span>}
                </td>
                <td className="p-3 text-gray-600">{r.reviewer}</td>
                <td className="p-3"><div className="flex flex-wrap gap-1">{r.scopes.map(s => <span key={s} className="px-1.5 py-0.5 bg-gray-100 rounded text-xs">{s}</span>)}</div></td>
                <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs capitalize ${decisionColor(r.decision)}`}>{r.decision}</span></td>
                <td className="p-3 text-gray-500">{r.timestamp}</td>
                <td className="p-3">
                  {r.decision === 'pending' && (
                    <div className="flex gap-1">
                      <button onClick={() => setDecision(r.id, 'approved')} className="text-green-600 text-xs hover:underline">Approve</button>
                      <button onClick={() => setDecision(r.id, 'rejected')} className="text-red-600 text-xs hover:underline">Reject</button>
                      <button onClick={() => setDecision(r.id, 'revoked')} className="text-amber-600 text-xs hover:underline">Revoke</button>
                    </div>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </div>
  );
}
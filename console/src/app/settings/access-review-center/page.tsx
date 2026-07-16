'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";

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
  const t = useTranslations();

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [reviews, setReviews] = useState<Review[]>([]);
  const [showForm, setShowForm] = useState(false);
  const [scheduledReview, setScheduledReview] = useState(true);
  const [reviewFrequency, setReviewFrequency] = useState('quarterly');
  const [newReview, setNewReview] = useState({ user: '', reviewer: '', scopes: [] as string[] });
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());

  useEffect(() => {
    fetch("/api/v1/policies/access-reviews/campaigns", {
      headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
    })
      .then(res => { if (!res.ok) return null; return res.json(); })
      .then(data => { setReviews(Array.isArray(data) ? data : (data.reviews || data.campaigns || [])); setLoading(false); })
      .catch(err => { setError(err.message); setLoading(false); });
  }, []);

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

  if (loading) return (
    <div className="p-6"><h1 className="text-2xl font-bold mb-4"> {t("backend3.accessReviewCenter.title")}</h1><p>Loading...</p></div>
  );
  if (error) return (
    <div className="p-6"><h1 className="text-2xl font-bold mb-4"> {t("backend3.accessReviewCenter.title")}</h1><p className="text-red-600">Error: {error}</p></div>
  );
  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold"> {t("backend3.accessReviewCenter.title")}</h1>
          <p className="text-gray-600">Certify user access, manage review schedules, and track overdue reviews.</p>
        </div>
        <button onClick={() => setShowForm(!showForm)} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">
          {showForm ? 'Cancel' : t("backend3.accessReviewCenter.createReview")}
        </button>
      </div>

      {showForm && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">{t("backend3.accessReviewCenter.createAccessReview")}</h2>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="text-sm font-medium">User</label>
              <input aria-label="user@ggid.io" type="text" placeholder="user@ggid.io" value={newReview.user} onChange={e => setNewReview(prev => ({ ...prev, user: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" />
            </div>
            <div>
              <label className="text-sm font-medium">Reviewer</label>
              <input aria-label="reviewer@ggid.io" type="text" placeholder="reviewer@ggid.io" value={newReview.reviewer} onChange={e => setNewReview(prev => ({ ...prev, reviewer: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" />
            </div>
          </div>
          <div>
            <label className="text-sm font-medium">Scopes to Review</label>
            <div className="flex flex-wrap gap-2 mt-2">
              {allScopes.map(s => (
                <label key={s} className="flex items-center gap-1 text-sm">
                  <input aria-label="New review" type="checkbox" checked={newReview.scopes.includes(s)} onChange={() => toggleScope(s)} className="rounded" />
                  {s}
                </label>
              ))}
            </div>
          </div>
          <button aria-label="action" onClick={createReview} disabled={!newReview.user} className="px-4 py-2 bg-blue-600 text-white rounded text-sm disabled:opacity-50">{t("backend3.accessReviewCenter.createReview")}</button>
        </section>
      )}

      <div className="grid grid-cols-3 gap-4">
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold">{reviews.length}</div>
          <div className="text-sm text-gray-500">Total Reviews</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold text-blue-600">{pendingCount}</div>
          <div className="text-sm text-gray-500">{t("backend3.accessReviewCenter.pending")}</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold text-red-600">{overdueCount}</div>
          <div className="text-sm text-gray-500">{t("backend3.accessReviewCenter.overdue")}</div>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <label className="flex items-center justify-between bg-white rounded-lg shadow p-4">
          <span className="text-sm font-medium">Scheduled Reviews</span>
          <input aria-label="Scheduled review" type="checkbox" checked={scheduledReview} onChange={e => setScheduledReview(e.target.checked)} className="rounded" />
        </label>
        <section className="bg-white rounded-lg shadow p-4">
          <label className="text-sm font-medium">Review Frequency</label>
          <select aria-label="review Frequency" value={reviewFrequency} onChange={e => setReviewFrequency(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1">
            <option value="monthly">{t("backend3.accessReviewCenter.monthly")}</option>
            <option value="quarterly">{t("backend3.accessReviewCenter.quarterly")}</option>
            <option value="semiannual">Semi-Annual</option>
            <option value="annual">{t("backend3.accessReviewCenter.annual")}</option>
          </select>
        </section>
      </div>

      {selectedIds.size > 0 && (
        <div className="flex items-center gap-3 bg-blue-50 rounded p-3">
          <span className="text-sm">{selectedIds.size} selected</span>
          <button aria-label="action" onClick={bulkApprove} className="px-3 py-1 bg-green-600 text-white rounded text-sm">{t("backend3.accessReviewCenter.bulkApprove")}</button>
          <button onClick={() => setSelectedIds(new Set())} className="px-3 py-1 border rounded text-sm">{t("backend3.accessReviewCenter.clear")}</button>
        </div>
      )}

      <section className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th scope="col" className="p-3"></th>
              <th scope="col" className="p-3">User</th>
              <th scope="col" className="p-3">Reviewer</th>
              <th scope="col" className="p-3">Scopes</th>
              <th scope="col" className="p-3">{t("backend3.accessReviewCenter.decision")}</th>
              <th scope="col" className="p-3">{t("backend3.accessReviewCenter.date")}</th>
              <th scope="col" className="p-3">{t("backend3.accessReviewCenter.actions")}</th>
            </tr>
          </thead>
          <tbody>
            {reviews.map(r => (
              <tr key={r.id} className="border-b hover:bg-gray-50">
                <td className="p-3"><input aria-label="Toggle" type="checkbox" checked={selectedIds.has(r.id)} onChange={() => toggleSelect(r.id)} className="rounded" /></td>
                <td className="p-3 font-medium">
                  {r.user}
                  {r.overdue && <span className="ml-1 px-1.5 py-0.5 bg-red-100 text-red-700 rounded text-xs">{t("backend3.accessReviewCenter.overdue")}</span>}
                </td>
                <td className="p-3 text-gray-600">{r.reviewer}</td>
                <td className="p-3"><div className="flex flex-wrap gap-1">{r.scopes.map(s => <span key={s} className="px-1.5 py-0.5 bg-gray-100 rounded text-xs">{s}</span>)}</div></td>
                <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs capitalize ${decisionColor(r.decision)}`}>{r.decision}</span></td>
                <td className="p-3 text-gray-500">{r.timestamp}</td>
                <td className="p-3">
                  {r.decision === 'pending' && (
                    <div className="flex gap-1">
                      <button onClick={() => setDecision(r.id, 'approved')} className="text-green-600 text-xs hover:underline">{t("backend3.accessReviewCenter.approve")}</button>
                      <button onClick={() => setDecision(r.id, 'rejected')} className="text-red-600 text-xs hover:underline">{t("backend3.accessReviewCenter.reject")}</button>
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
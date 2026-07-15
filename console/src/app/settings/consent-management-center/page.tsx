'use client';
import { useState, useEffect } from 'react';
import { Loader2 } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface Consent {
  id: string;
  user: string;
  purpose: string;
  grantedAt: string;
  expires: string;
  status: string;
}

export default function ConsentManagementCenterPage() {
  const t = useTranslations();


  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const res = await fetch("/api/v1/oauth/consent", {
          method: "GET",
          headers: {
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

  const [filterPurpose, setFilterPurpose] = useState('all');
  const [filterStatus, setFilterStatus] = useState('all');
  const [showReceipt, setShowReceipt] = useState<Consent | null>(null);const [consents, setConsents] = useState<Consent[]>([
    { id: 'cn1', user: 'alice@ggid.io', purpose: 'marketing', grantedAt: '2026-06-01', expires: '2027-06-01', status: 'active' },
    { id: 'cn2', user: 'bob@ggid.io', purpose: 'data-sharing', grantedAt: '2026-05-15', expires: '2026-11-15', status: 'active' },
    { id: 'cn3', user: 'carol@ggid.io', purpose: 'marketing', grantedAt: '2026-01-01', expires: '2026-07-01', status: 'expired' },
    { id: 'cn4', user: 'dave@ggid.io', purpose: 'analytics', grantedAt: '2026-07-01', expires: '2027-07-01', status: 'active' },
    { id: 'cn5', user: 'alice@ggid.io', purpose: 'analytics', grantedAt: '2026-04-01', expires: '2026-10-01', status: 'revoked' },
  ]);
const [purposes] = useState([
    { name: 'marketing', count: 2, description: 'Marketing communications' },
    { name: 'data-sharing', count: 1, description: 'Share data with partners' },
    { name: 'analytics', count: 2, description: 'Usage analytics tracking' },
  ]);

  if (loading) return <div className="flex items-center justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-blue-500" /></div>;
  if (error) return <div className="p-8 text-red-500">Error: {error}</div>;
  
  
  const statusColor = (s: string): string =>
    s === 'active' ? 'bg-green-100 text-green-700' : s === 'revoked' ? 'bg-red-100 text-red-700' : 'bg-gray-200 text-gray-600';

  const filtered = consents.filter(c =>
    (filterPurpose === 'all' || c.purpose === filterPurpose) &&
    (filterStatus === 'all' || c.status === filterStatus)
  );

  const revokeConsent = (id: string) => {
    setConsents(prev => prev.map(c => c.id === id ? { ...c, status: 'revoked' } : c));
  };

  const activeCount = consents.filter(c => c.status === 'active').length;
  const revokedCount = consents.filter(c => c.status === 'revoked').length;
  const grantRate = Math.round((consents.length / (consents.length + revokedCount)) * 100);

  const generateReceipt = (c: Consent): string => {
    return btoa(JSON.stringify({
      iss: 'ggid.io', sub: c.user, purpose: c.purpose,
      iat: c.grantedAt, exp: c.expires, status: c.status, jti: c.id,
    }));
  };

  const exportReport = () => {
    const csv = ['id,user,purpose,granted_at,expires,status', ...consents.map(c => `${c.id},${c.user},${c.purpose},${c.grantedAt},${c.expires},${c.status}`)].join('\n');
    const blob = new Blob([csv], { type: 'text/csv' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url; a.download = 'consent-report.csv'; a.click();
  };

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Consent Management Center</h1>
          <p className="text-gray-600">Manage user consent records, receipts, and analytics.</p>
        </div>
        <button onClick={exportReport} className="px-4 py-2 border rounded text-sm">Export Report</button>
      </div>

      <div className="grid grid-cols-4 gap-4">
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-2xl font-bold">{consents.length}</div><div className="text-sm text-gray-500">Total Records</div></div>
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-2xl font-bold text-green-600">{activeCount}</div><div className="text-sm text-gray-500">Active</div></div>
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-2xl font-bold text-red-600">{revokedCount}</div><div className="text-sm text-gray-500">Revoked</div></div>
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-2xl font-bold">{grantRate}%</div><div className="text-sm text-gray-500">Grant Rate</div></div>
      </div>

      <div className="flex gap-4">
        <select value={filterPurpose} onChange={e => setFilterPurpose(e.target.value)} className="border rounded px-3 py-2 text-sm">
          <option value="all">All Purposes</option>
          {purposes.map(p => <option key={p.name} value={p.name}>{p.name} ({p.count})</option>)}
        </select>
        <select value={filterStatus} onChange={e => setFilterStatus(e.target.value)} className="border rounded px-3 py-2 text-sm">
          <option value="all">All Statuses</option>
          <option value="active">Active</option>
          <option value="revoked">Revoked</option>
          <option value="expired">Expired</option>
        </select>
      </div>

      <section className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th className="p-3">User</th>
              <th className="p-3">Purpose</th>
              <th className="p-3">Granted</th>
              <th className="p-3">Expires</th>
              <th className="p-3">Status</th>
              <th className="p-3">Actions</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map(c => (
              <tr key={c.id} className="border-b hover:bg-gray-50">
                <td className="p-3 font-medium">{c.user}</td>
                <td className="p-3"><span className="px-2 py-0.5 bg-blue-100 text-blue-700 rounded text-xs">{c.purpose}</span></td>
                <td className="p-3 text-gray-500">{c.grantedAt}</td>
                <td className="p-3 text-gray-500">{c.expires}</td>
                <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs ${statusColor(c.status)}`}>{c.status}</span></td>
                <td className="p-3">
                  <div className="flex gap-2">
                    <button onClick={() => setShowReceipt(c)} className="text-blue-600 text-xs hover:underline">Receipt</button>
                    {c.status === 'active' && <button onClick={() => revokeConsent(c.id)} className="text-red-600 text-xs hover:underline">Revoke</button>}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      {showReceipt && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 max-w-lg w-full mx-4 space-y-4">
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-semibold">Consent Receipt (JWT)</h2>
              <button onClick={() => setShowReceipt(null)} className="text-gray-400 text-sm">Close</button>
            </div>
            <div className="space-y-2 text-sm">
              <div><span className="text-gray-500">User:</span> {showReceipt.user}</div>
              <div><span className="text-gray-500">Purpose:</span> {showReceipt.purpose}</div>
              <div><span className="text-gray-500">Granted:</span> {showReceipt.grantedAt}</div>
              <div><span className="text-gray-500">Expires:</span> {showReceipt.expires}</div>
              <div><span className="text-gray-500">Status:</span> {showReceipt.status}</div>
            </div>
            <div>
              <div className="text-xs text-gray-500 mb-1">JWT Token:</div>
              <pre className="bg-gray-900 text-green-400 rounded p-3 text-xs overflow-x-auto break-all">{generateReceipt(showReceipt)}</pre>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
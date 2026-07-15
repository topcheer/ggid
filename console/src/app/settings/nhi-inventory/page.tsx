'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";

interface NHIEntry {
  id: string;
  name: string;
  type: string;
  status: string;
  created: string;
  lastUsed: string;
  owner: string;
  riskScore: number;
}

interface DecommissionModal {
  open: boolean;
  entry: NHIEntry | null;
  reason: string;
}

export default function NHIInventoryPage() {
  const t = useTranslations();

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [entries, setEntries] = useState<NHIEntry[]>([]);

  useEffect(() => {
    fetch("/api/v1/identity/nhi", {
      headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
    })
      .then(res => { if (!res.ok) return null; return res.json(); })
      .then(data => { setEntries(Array.isArray(data) ? data : (data.entries || data.items || [])); setLoading(false); })
      .catch(err => { setError(err.message); setLoading(false); });
  }, []);

  const [filter, setFilter] = useState('all');
  const [modal, setModal] = useState<DecommissionModal>({ open: false, entry: null, reason: '' });

  const isOrphan = (lastUsed: string): boolean => {
    const days = Math.floor((Date.now() - new Date(lastUsed).getTime()) / 86400000);
    return days > 90;
  };

  const riskColor = (score: number): string =>
    score >= 75 ? 'text-red-600 bg-red-50' : score >= 50 ? 'text-amber-600 bg-amber-50' : 'text-green-600 bg-green-50';

  const filtered = filter === 'all' ? entries : entries.filter(e => e.type === filter);

  const openDecommission = (entry: NHIEntry) => setModal({ open: true, entry, reason: '' });

  const confirmDecommission = () => {
    if (modal.entry) {
      setEntries(prev => prev.map(e => e.id === modal.entry!.id ? { ...e, status: 'decommissioned' } : e));
    }
    setModal({ open: false, entry: null, reason: '' });
  };

  const types = ['all', 'service-account', 'api-key', 'ai-agent', 'iot', 'oauth-client'];

  if (loading) return (
    <div className="p-6"><h1 className="text-2xl font-bold mb-4">NHI Inventory</h1><p>Loading...</p></div>
  );
  if (error) return (
    <div className="p-6"><h1 className="text-2xl font-bold mb-4">NHI Inventory</h1><p className="text-red-600">Error: {error}</p></div>
  );
  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Non-Human Identity Inventory</h1>
        <p className="text-gray-600">Track and manage all non-human identities across the organization.</p>
      </div>

      <div className="flex gap-2 flex-wrap">
        {types.map(t => (
          <button
            key={t}
            onClick={() => setFilter(t)}
            className={`px-3 py-1.5 rounded text-sm capitalize ${filter === t ? 'bg-blue-600 text-white' : 'bg-gray-100 text-gray-600'}`}
          >
            {t === 'all' ? 'All' : t.replace('-', ' ')}
          </button>
        ))}
      </div>

      <div className="grid grid-cols-4 gap-4">
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold">{entries.length}</div>
          <div className="text-sm text-gray-500">Total NHIs</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold text-amber-600">{entries.filter(e => isOrphan(e.lastUsed)).length}</div>
          <div className="text-sm text-gray-500">Orphaned</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold text-red-600">{entries.filter(e => e.riskScore >= 75).length}</div>
          <div className="text-sm text-gray-500">High Risk</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold text-green-600">{entries.filter(e => e.riskScore < 50).length}</div>
          <div className="text-sm text-gray-500">Low Risk</div>
        </div>
      </div>

      <section className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th className="p-3">Name</th>
              <th className="p-3">Type</th>
              <th className="p-3">Status</th>
              <th className="p-3">Created</th>
              <th className="p-3">Last Used</th>
              <th className="p-3">Owner</th>
              <th className="p-3">Risk</th>
              <th className="p-3">Action</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map(e => (
              <tr key={e.id} className="border-b hover:bg-gray-50">
                <td className="p-3 font-medium">{e.name}</td>
                <td className="p-3 capitalize text-gray-600">{e.type.replace('-', ' ')}</td>
                <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs ${e.status === 'active' ? 'bg-green-100 text-green-700' : 'bg-gray-200 text-gray-600'}`}>{e.status}</span></td>
                <td className="p-3 text-gray-500">{e.created}</td>
                <td className="p-3 text-gray-500">
                  {e.lastUsed}
                  {isOrphan(e.lastUsed) && <span className="ml-1 px-1.5 py-0.5 bg-amber-100 text-amber-700 rounded text-xs">Orphan</span>}
                </td>
                <td className="p-3 text-gray-600">{e.owner}</td>
                <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs font-mono ${riskColor(e.riskScore)}`}>{e.riskScore}</span></td>
                <td className="p-3">
                  {e.status === 'active' && (
                    <button onClick={() => openDecommission(e)} className="text-red-600 text-xs hover:underline">Decommission</button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      {modal.open && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 max-w-md w-full mx-4 space-y-4">
            <h2 className="text-lg font-semibold">Decommission NHI</h2>
            <p className="text-sm text-gray-600">You are about to decommission <strong>{modal.entry?.name}</strong>. This action will revoke all credentials and disable access.</p>
            <div>
              <label className="text-sm font-medium">Reason</label>
              <select value={modal.reason} onChange={e => setModal(prev => ({ ...prev, reason: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1">
                <option value="">Select reason...</option>
                <option value="orphaned">Orphaned - No recent usage</option>
                <option value="replaced">Replaced by new identity</option>
                <option value="security">Security concern</option>
                <option value="end-of-life">End of lifecycle</option>
              </select>
            </div>
            <div className="flex justify-end gap-3">
              <button onClick={() => setModal({ open: false, entry: null, reason: '' })} className="px-4 py-2 border rounded text-sm">Cancel</button>
              <button onClick={confirmDecommission} disabled={!modal.reason} className="px-4 py-2 bg-red-600 text-white rounded text-sm disabled:opacity-50">Confirm Decommission</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

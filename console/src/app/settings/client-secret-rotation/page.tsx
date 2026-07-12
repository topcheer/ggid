'use client';
import { useState, useEffect } from 'react';

interface ClientSecret {
  id: string;
  clientId: string;
  clientName: string;
  lastRotated: string;
  nextRotation: string;
  ageDays: number;
  autoRotate: boolean;
  intervalDays: number;
  dualSecret: boolean;
  dualPeriodDays: number;
}

export default function ClientSecretRotationPage() {

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [data, setData] = useState<any[]>([]);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const res = await fetch("/api/v1/oauth/clients", {
          method: "GET",
          headers: {
            "Content-Type": "application/json",
            "X-Tenant-ID": "00000000-0000-0000-0000-000000000001",
          },
        });
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const json = await res.json();
        setData(Array.isArray(json) ? json : [json]);
      } catch (e) {
        setError(e instanceof Error ? e.message : "Failed to load");
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, []);

  if (loading) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-500">Error: {error}</div>;
  if (!data || data.length === 0) return <div className="p-8 text-gray-500">No data available</div>;
  const [clients, setClients] = useState<ClientSecret[]>([
    { id: 'c1', clientId: 'web-app', clientName: 'Web Application', lastRotated: '2026-06-01', nextRotation: '2026-09-01', ageDays: 42, autoRotate: true, intervalDays: 90, dualSecret: true, dualPeriodDays: 7 },
    { id: 'c2', clientId: 'mobile-app', clientName: 'Mobile App', lastRotated: '2026-04-15', nextRotation: '2026-07-15', ageDays: 89, autoRotate: false, intervalDays: 90, dualSecret: false, dualPeriodDays: 0 },
    { id: 'c3', clientId: 'admin-cli', clientName: 'Admin CLI', lastRotated: '2025-12-01', nextRotation: '2026-06-01', ageDays: 224, autoRotate: true, intervalDays: 180, dualSecret: true, dualPeriodDays: 14 },
    { id: 'c4', clientId: 'api-gateway', clientName: 'API Gateway', lastRotated: '2026-07-01', nextRotation: '2027-01-01', ageDays: 12, autoRotate: true, intervalDays: 180, dualSecret: false, dualPeriodDays: 0 },
  ]);

  const [rotateTarget, setRotateTarget] = useState<ClientSecret | null>(null);
  const [newSecret, setNewSecret] = useState('');
  const [history] = useState([
    { clientId: 'web-app', rotatedAt: '2026-06-01', rotatedBy: 'admin@ggid.io' },
    { clientId: 'mobile-app', rotatedAt: '2026-04-15', rotatedBy: 'dev-team@ggid.io' },
    { clientId: 'admin-cli', rotatedAt: '2025-12-01', rotatedBy: 'admin@ggid.io' },
    { clientId: 'api-gateway', rotatedAt: '2026-07-01', rotatedBy: 'infra@ggid.io' },
  ]);

  const isOverdue = (c: ClientSecret) => c.ageDays > c.intervalDays;
  const isDueSoon = (c: ClientSecret) => c.ageDays > c.intervalDays - 14 && c.ageDays <= c.intervalDays;

  const ageBadge = (c: ClientSecret) => {
    if (isOverdue(c)) return <span className="px-2 py-0.5 bg-red-100 text-red-700 rounded text-xs">Overdue</span>;
    if (isDueSoon(c)) return <span className="px-2 py-0.5 bg-amber-100 text-amber-700 rounded text-xs">Due soon</span>;
    return <span className="px-2 py-0.5 bg-green-100 text-green-700 rounded text-xs">OK</span>;
  };

  const rotate = () => {
    if (rotateTarget) {
      const generated = Array.from({ length: 32 }, () => Math.floor(Math.random() * 16).toString(16)).join('');
      setNewSecret(generated);
      setClients(prev => prev.map(c => c.id === rotateTarget.id ? {
        ...c,
        lastRotated: new Date().toISOString().slice(0, 10),
        nextRotation: new Date(Date.now() + c.intervalDays * 86400000).toISOString().slice(0, 10),
        ageDays: 0,
      } : c));
    }
  };

  const toggleAutoRotate = (id: string) => {
    setClients(prev => prev.map(c => c.id === id ? { ...c, autoRotate: !c.autoRotate } : c));
  };

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Client Secret Rotation</h1>
        <p className="text-gray-600">Manage OAuth client secret rotation schedules and dual-secret periods.</p>
      </div>

      <div className="grid grid-cols-3 gap-4">
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold">{clients.length}</div>
          <div className="text-sm text-gray-500">Total Clients</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold text-red-600">{clients.filter(isOverdue).length}</div>
          <div className="text-sm text-gray-500">Overdue</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold text-amber-600">{clients.filter(isDueSoon).length}</div>
          <div className="text-sm text-gray-500">Due Soon</div>
        </div>
      </div>

      <section className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th className="p-3">Client</th>
              <th className="p-3">Last Rotated</th>
              <th className="p-3">Next Rotation</th>
              <th className="p-3">Age</th>
              <th className="p-3">Status</th>
              <th className="p-3">Auto</th>
              <th className="p-3">Dual Secret</th>
              <th className="p-3">Action</th>
            </tr>
          </thead>
          <tbody>
            {clients.map(c => (
              <tr key={c.id} className="border-b hover:bg-gray-50">
                <td className="p-3"><div className="font-medium">{c.clientName}</div><div className="font-mono text-xs text-gray-500">{c.clientId}</div></td>
                <td className="p-3 text-gray-500">{c.lastRotated}</td>
                <td className="p-3 text-gray-500">{c.nextRotation}</td>
                <td className="p-3">{c.ageDays}d</td>
                <td className="p-3">{ageBadge(c)}</td>
                <td className="p-3"><label className="flex items-center"><input type="checkbox" checked={c.autoRotate} onChange={() => toggleAutoRotate(c.id)} className="rounded" /></label></td>
                <td className="p-3">{c.dualSecret ? <span className="text-xs text-blue-600">{c.dualPeriodDays}d period</span> : <span className="text-xs text-gray-400">no</span>}</td>
                <td className="p-3"><button onClick={() => { setRotateTarget(c); setNewSecret(''); }} className="text-blue-600 text-xs hover:underline">Rotate</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      {rotateTarget && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 max-w-md w-full mx-4 space-y-4">
            <h2 className="text-lg font-semibold">Rotate Secret: {rotateTarget.clientName}</h2>
            {newSecret ? (
              <>
                <p className="text-sm text-amber-700 bg-amber-50 rounded p-3">Copy this secret now. It will only be shown once.</p>
                <pre className="bg-gray-900 text-green-400 rounded p-3 text-xs overflow-x-auto break-all">{newSecret}</pre>
                <div className="flex justify-end gap-3">
                  <button onClick={() => { navigator.clipboard.writeText(newSecret); }} className="px-4 py-2 border rounded text-sm">Copy</button>
                  <button onClick={() => setRotateTarget(null)} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">Done</button>
                </div>
              </>
            ) : (
              <>
                <p className="text-sm text-gray-600">You are about to rotate the secret for <strong>{rotateTarget.clientName}</strong> ({rotateTarget.clientId}).{rotateTarget.dualSecret ? ` The old secret will remain valid for ${rotateTarget.dualPeriodDays} days (dual-secret period).` : ' The old secret will be revoked immediately.'}</p>
                <div className="flex justify-end gap-3">
                  <button onClick={() => setRotateTarget(null)} className="px-4 py-2 border rounded text-sm">Cancel</button>
                  <button onClick={rotate} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">Confirm Rotation</button>
                </div>
              </>
            )}
          </div>
        </div>
      )}

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Secret Rotation History</h2>
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th className="p-3">Client ID</th>
              <th className="p-3">Rotated At</th>
              <th className="p-3">Rotated By</th>
            </tr>
          </thead>
          <tbody>
            {history.map((h, idx) => (
              <tr key={idx} className="border-b">
                <td className="p-3 font-mono text-xs">{h.clientId}</td>
                <td className="p-3 text-gray-500">{h.rotatedAt}</td>
                <td className="p-3 text-gray-600">{h.rotatedBy}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </div>
  );
}
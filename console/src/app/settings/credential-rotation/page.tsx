'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface RotationSchedule {
  id: string;
  credential: string;
  type: string;
  intervalDays: number;
  nextRotation: string;
  status: string;
  autoRotate: boolean;
  notifyBeforeDays: number;
}

interface NewSchedule {
  credential: string;
  intervalDays: number;
  autoRotate: boolean;
  notifyBeforeDays: number;
}

export default function CredentialRotationPage() {
  const t = useTranslations();

  const [schedules, setSchedules] = useState<RotationSchedule[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch('/api/v1/auth/credentials/rotation/due', {
      headers: { ...authHeader(), 'Content-Type': 'application/json', 'X-Tenant-ID': '00000000-0000-0000-0000-000000000001' },
    })
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(data => {
        if (data && data.schedules) setSchedules(data.schedules);
        else if (Array.isArray(data)) setSchedules(data);
        setLoading(false);
      })
      .catch(e => { setError(e.message); setLoading(false); });
  }, []);

  const [showForm, setShowForm] = useState(false);
  const [newSchedule, setNewSchedule] = useState<NewSchedule>({ credential: '', intervalDays: 90, autoRotate: true, notifyBeforeDays: 7 });
  const [executeTarget, setExecuteTarget] = useState<RotationSchedule | null>(null);

  const statusBadge = (status: string): string =>
    status === 'overdue' ? 'bg-red-100 text-red-700' :
    status === 'due' ? 'bg-amber-100 text-amber-700' :
    'bg-green-100 text-green-700';

  const addSchedule = () => {
    const next = new Date();
    next.setDate(next.getDate() + newSchedule.intervalDays);
    setSchedules(prev => [...prev, {
      id: `s${prev.length + 1}`,
      credential: newSchedule.credential,
      type: 'custom',
      intervalDays: newSchedule.intervalDays,
      nextRotation: next.toISOString().slice(0, 10),
      status: 'scheduled',
      autoRotate: newSchedule.autoRotate,
      notifyBeforeDays: newSchedule.notifyBeforeDays,
    }]);
    setShowForm(false);
    setNewSchedule({ credential: '', intervalDays: 90, autoRotate: true, notifyBeforeDays: 7 });
  };

  const executeRotation = () => {
    if (executeTarget) {
      const next = new Date();
      next.setDate(next.getDate() + executeTarget.intervalDays);
      setSchedules(prev => prev.map(s => s.id === executeTarget.id ? { ...s, status: 'scheduled', nextRotation: next.toISOString().slice(0, 10) } : s));
    }
    setExecuteTarget(null);
  };

  if (loading) return <div className="p-6"><p>Loading...</p></div>;
  if (error) return <div className="p-6 text-red-600">Error: {error}</div>;

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t("credentialRotation.title")}</h1>
          <p className="text-gray-600">Schedule and execute credential rotation across the organization.</p>
        </div>
        <button onClick={() => setShowForm(!showForm)} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">
          {showForm ? 'Cancel' : 'Schedule Rotation'}
        </button>
      </div>

      {showForm && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">New Rotation Schedule</h2>
          <div>
            <label className="text-sm font-medium">Credential</label>
            <input aria-label="Credential name" type="text" placeholder="Credential name" value={newSchedule.credential} onChange={e => setNewSchedule(prev => ({ ...prev, credential: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="text-sm font-medium">Interval (days)</label>
              <input aria-label="new Schedule" type="number" min={1} max={365} value={newSchedule.intervalDays} onChange={e => setNewSchedule(prev => ({ ...prev, intervalDays: parseInt(e.target.value) || 90 }))} className="w-full border rounded px-3 py-2 text-sm mt-1" />
            </div>
            <div>
              <label className="text-sm font-medium">Notify Before (days)</label>
              <input aria-label="new Schedule" type="number" min={1} max={30} value={newSchedule.notifyBeforeDays} onChange={e => setNewSchedule(prev => ({ ...prev, notifyBeforeDays: parseInt(e.target.value) || 7 }))} className="w-full border rounded px-3 py-2 text-sm mt-1" />
            </div>
          </div>
          <label className="flex items-center gap-2">
            <input aria-label="New schedule" type="checkbox" checked={newSchedule.autoRotate} onChange={e => setNewSchedule(prev => ({ ...prev, autoRotate: e.target.checked }))} className="rounded" />
            <span className="text-sm">Auto-rotate when due</span>
          </label>
          <button onClick={addSchedule} disabled={!newSchedule.credential} className="px-4 py-2 bg-blue-600 text-white rounded text-sm disabled:opacity-50">Add Schedule</button>
        </section>
      )}

      <div className="grid grid-cols-3 gap-4">
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold">{schedules.length}</div>
          <div className="text-sm text-gray-500">Total Schedules</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold text-amber-600">{schedules.filter(s => s.status === 'due').length}</div>
          <div className="text-sm text-gray-500">Due</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold text-red-600">{schedules.filter(s => s.status === 'overdue').length}</div>
          <div className="text-sm text-gray-500">Overdue</div>
        </div>
      </div>

      <section className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th scope="col" className="p-3">Credential</th>
              <th scope="col" className="p-3">Type</th>
              <th scope="col" className="p-3">Interval</th>
              <th scope="col" className="p-3">Next Rotation</th>
              <th scope="col" className="p-3">Status</th>
              <th scope="col" className="p-3">Auto</th>
              <th scope="col" className="p-3">Action</th>
            </tr>
          </thead>
          <tbody>
            {schedules.length === 0 ? (
              <tr><td colSpan={7} className="p-3 text-center text-gray-400">No data available</td></tr>
            ) : schedules.map(s => (
              <tr key={s.id} className="border-b hover:bg-gray-50">
                <td className="p-3 font-medium">{s.credential}</td>
                <td className="p-3 text-gray-600 capitalize">{s.type.replace('-', ' ')}</td>
                <td className="p-3 text-gray-500">{s.intervalDays}d</td>
                <td className="p-3 text-gray-500">{s.nextRotation}</td>
                <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs capitalize ${statusBadge(s.status)}`}>{s.status}</span></td>
                <td className="p-3">{s.autoRotate ? <span className="text-green-600 text-xs">Yes</span> : <span className="text-gray-400 text-xs">No</span>}</td>
                <td className="p-3">
                  <button onClick={() => setExecuteTarget(s)} className="text-blue-600 text-xs hover:underline">Rotate Now</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      {executeTarget && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 max-w-md w-full mx-4 space-y-4">
            <h2 className="text-lg font-semibold">Execute Rotation</h2>
            <p className="text-sm text-gray-600">You are about to rotate <strong>{executeTarget.credential}</strong>. The old credential will be revoked immediately and a new one will be generated.</p>
            <div className="text-xs text-gray-400">
              <div>Type: {executeTarget.type}</div>
              <div>Current status: {executeTarget.status}</div>
              <div>Next rotation: {executeTarget.nextRotation}</div>
            </div>
            <div className="flex justify-end gap-3">
              <button onClick={() => setExecuteTarget(null)} className="px-4 py-2 border rounded text-sm">Cancel</button>
              <button onClick={executeRotation} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">Confirm Rotation</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

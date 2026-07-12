'use client';
import { useState } from 'react';

interface Device {
  id: string;
  deviceName: string;
  platform: string;
  fingerprint: string;
  boundAt: string;
  lastSeen: string;
  trustScore: number;
}

export default function DeviceBindingConfigPage() {
  const [devices, setDevices] = useState<Device[]>([
    { id: 'd1', deviceName: 'MacBook Pro', platform: 'macOS', fingerprint: 'fp_a1b2c3d4e5', boundAt: '2026-05-01', lastSeen: '2026-07-12', trustScore: 85 },
    { id: 'd2', deviceName: 'iPhone 15', platform: 'iOS', fingerprint: 'fp_f6e5d4c3b2', boundAt: '2026-04-15', lastSeen: '2026-07-11', trustScore: 90 },
    { id: 'd3', deviceName: 'Work Desktop', platform: 'Windows', fingerprint: 'fp_a1b2c3f6e5', boundAt: '2026-03-01', lastSeen: '2026-07-10', trustScore: 70 },
    { id: 'd4', deviceName: 'Unknown Device', platform: 'Android', fingerprint: 'fp_z9y8x7w6v5', boundAt: '2026-07-05', lastSeen: '2026-06-28', trustScore: 35 },
    { id: 'd5', deviceName: 'Linux Server', platform: 'Linux', fingerprint: 'fp_q1w2e3r4t5', boundAt: '2026-02-10', lastSeen: '2026-07-12', trustScore: 80 },
  ]);

  const [platformFilter, setPlatformFilter] = useState('all');
  const [unbindTarget, setUnbindTarget] = useState<Device | null>(null);
  const [thresholds, setThresholds] = useState({ trusted: 70, suspicious: 40 });

  const platforms = ['all', 'iOS', 'Android', 'Windows', 'macOS', 'Linux'];

  const trustColor = (score: number): string =>
    score >= thresholds.trusted ? 'bg-green-100 text-green-700' :
    score >= thresholds.suspicious ? 'bg-amber-100 text-amber-700' :
    'bg-red-100 text-red-700';

  const trustLabel = (score: number): string =>
    score >= thresholds.trusted ? 'trusted' :
    score >= thresholds.suspicious ? 'suspicious' :
    'untrusted';

  const filtered = platformFilter === 'all' ? devices : devices.filter(d => d.platform === platformFilter);

  const confirmUnbind = () => {
    if (unbindTarget) setDevices(prev => prev.filter(d => d.id !== unbindTarget.id));
    setUnbindTarget(null);
  };

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Device Binding Configuration</h1>
        <p className="text-gray-600">Manage bound devices, trust scores, and platform-specific policies.</p>
      </div>

      <div className="grid grid-cols-4 gap-4">
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold">{devices.length}</div>
          <div className="text-sm text-gray-500">Bound Devices</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold text-green-600">{devices.filter(d => d.trustScore >= thresholds.trusted).length}</div>
          <div className="text-sm text-gray-500">Trusted</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold text-amber-600">{devices.filter(d => d.trustScore >= thresholds.suspicious && d.trustScore < thresholds.trusted).length}</div>
          <div className="text-sm text-gray-500">Suspicious</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold text-red-600">{devices.filter(d => d.trustScore < thresholds.suspicious).length}</div>
          <div className="text-sm text-gray-500">Untrusted</div>
        </div>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Trust Score Thresholds</h2>
        <div className="flex items-center gap-6">
          <div>
            <label className="text-sm font-medium">Trusted ({'>='})</label>
            <input type="number" min={0} max={100} value={thresholds.trusted} onChange={e => setThresholds(prev => ({ ...prev, trusted: parseInt(e.target.value) || 70 }))} className="w-20 border rounded px-2 py-1 text-sm mt-1" />
          </div>
          <div>
            <label className="text-sm font-medium">Suspicious ({'>='})</label>
            <input type="number" min={0} max={100} value={thresholds.suspicious} onChange={e => setThresholds(prev => ({ ...prev, suspicious: parseInt(e.target.value) || 40 }))} className="w-20 border rounded px-2 py-1 text-sm mt-1" />
          </div>
        </div>
      </section>

      <div className="flex gap-2 flex-wrap">
        {platforms.map(p => (
          <button key={p} onClick={() => setPlatformFilter(p)} className={`px-3 py-1.5 rounded text-sm ${platformFilter === p ? 'bg-blue-600 text-white' : 'bg-gray-100 text-gray-600'}`}>
            {p === 'all' ? 'All Platforms' : p}
          </button>
        ))}
      </div>

      <section className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th className="p-3">Device</th>
              <th className="p-3">Platform</th>
              <th className="p-3">Fingerprint</th>
              <th className="p-3">Bound At</th>
              <th className="p-3">Last Seen</th>
              <th className="p-3">Trust Score</th>
              <th className="p-3">Action</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map(d => (
              <tr key={d.id} className="border-b hover:bg-gray-50">
                <td className="p-3 font-medium">{d.deviceName}</td>
                <td className="p-3"><span className="px-2 py-0.5 bg-gray-100 rounded text-xs">{d.platform}</span></td>
                <td className="p-3 font-mono text-xs text-gray-500">{d.fingerprint}</td>
                <td className="p-3 text-gray-500">{d.boundAt}</td>
                <td className="p-3 text-gray-500">{d.lastSeen}</td>
                <td className="p-3">
                  <span className={`px-2 py-0.5 rounded text-xs font-mono ${trustColor(d.trustScore)}`}>{d.trustScore} ({trustLabel(d.trustScore)})</span>
                </td>
                <td className="p-3"><button onClick={() => setUnbindTarget(d)} className="text-red-600 text-xs hover:underline">Unbind</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      {unbindTarget && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 max-w-md w-full mx-4 space-y-4">
            <h2 className="text-lg font-semibold">Unbind Device</h2>
            <p className="text-sm text-gray-600">Unbind <strong>{unbindTarget.deviceName}</strong> ({unbindTarget.platform})? The user will need to re-bind this device for authentication.</p>
            <div className="flex justify-end gap-3">
              <button onClick={() => setUnbindTarget(null)} className="px-4 py-2 border rounded text-sm">Cancel</button>
              <button onClick={confirmUnbind} className="px-4 py-2 bg-red-600 text-white rounded text-sm">Confirm Unbind</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
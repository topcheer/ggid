'use client';
import { useState } from 'react';

interface Passkey {
  id: string;
  deviceName: string;
  platform: string;
  created: string;
  lastUsed: string;
  transports: string[];
  syncStatus: string;
  backupEligible: boolean;
}

export default function PasskeyManagementPage() {
  const [passkeys, setPasskeys] = useState<Passkey[]>([
    { id: 'pk1', deviceName: 'MacBook Pro', platform: 'Apple', created: '2026-06-01', lastUsed: '2026-07-12', transports: ['internal', 'hybrid'], syncStatus: 'synced', backupEligible: true },
    { id: 'pk2', deviceName: 'iPhone 15', platform: 'Apple', created: '2026-05-15', lastUsed: '2026-07-11', transports: ['internal'], syncStatus: 'synced', backupEligible: true },
    { id: 'pk3', deviceName: 'Chrome on Windows', platform: 'Google', created: '2026-04-20', lastUsed: '2026-06-28', transports: ['internal', 'hybrid'], syncStatus: 'synced', backupEligible: true },
    { id: 'pk4', deviceName: 'Security Key FIDO2', platform: 'Hardware', created: '2026-03-10', lastUsed: '2026-07-10', transports: ['usb', 'nfc'], syncStatus: 'not-applicable', backupEligible: false },
  ]);

  const [showEnroll, setShowEnroll] = useState(false);
  const [enrollDevice, setEnrollDevice] = useState('');
  const [revokeTarget, setRevokeTarget] = useState<Passkey | null>(null);

  const platformColor = (p: string): string =>
    p === 'Apple' ? 'bg-gray-100 text-gray-700' :
    p === 'Google' ? 'bg-blue-100 text-blue-700' :
    p === 'Microsoft' ? 'bg-cyan-100 text-cyan-700' :
    'bg-amber-100 text-amber-700';

  const syncColor = (s: string): string =>
    s === 'synced' ? 'bg-green-100 text-green-700' :
    s === 'pending' ? 'bg-amber-100 text-amber-700' :
    'bg-gray-100 text-gray-500';

  const enrollPasskey = () => {
    const platforms = ['Apple', 'Google', 'Microsoft', 'Hardware'];
    const newPk: Passkey = {
      id: `pk${passkeys.length + 1}`,
      deviceName: enrollDevice || 'New Device',
      platform: platforms[Math.floor(Math.random() * platforms.length)],
      created: new Date().toISOString().slice(0, 10),
      lastUsed: new Date().toISOString().slice(0, 10),
      transports: ['internal', 'hybrid'],
      syncStatus: 'pending',
      backupEligible: true,
    };
    setPasskeys(prev => [...prev, newPk]);
    setShowEnroll(false);
    setEnrollDevice('');
  };

  const confirmRevoke = () => {
    if (revokeTarget) {
      setPasskeys(prev => prev.filter(p => p.id !== revokeTarget.id));
    }
    setRevokeTarget(null);
  };

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Passkey Management</h1>
          <p className="text-gray-600">Manage registered passkeys and enroll new devices.</p>
        </div>
        <button onClick={() => setShowEnroll(!showEnroll)} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">
          {showEnroll ? 'Cancel' : 'Enroll Passkey'}
        </button>
      </div>

      {showEnroll && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">One-Tap Passkey Enrollment</h2>
          <p className="text-sm text-gray-500">Enter a device name to begin the WebAuthn enrollment flow. The browser will prompt for platform authenticator registration.</p>
          <div>
            <label className="text-sm font-medium">Device Name</label>
            <input
              type="text"
              placeholder="e.g. Work Laptop, Personal Phone"
              value={enrollDevice}
              onChange={e => setEnrollDevice(e.target.value)}
              className="w-full border rounded px-3 py-2 text-sm mt-1"
            />
          </div>
          <button onClick={enrollPasskey} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">
            Start Enrollment
          </button>
        </section>
      )}

      <div className="grid grid-cols-4 gap-4">
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold">{passkeys.length}</div>
          <div className="text-sm text-gray-500">Registered</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold text-green-600">{passkeys.filter(p => p.syncStatus === 'synced').length}</div>
          <div className="text-sm text-gray-500">Synced</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold text-blue-600">{passkeys.filter(p => p.backupEligible).length}</div>
          <div className="text-sm text-gray-500">Backup Eligible</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold">{new Set(passkeys.map(p => p.platform)).size}</div>
          <div className="text-sm text-gray-500">Platforms</div>
        </div>
      </div>

      <section className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th className="p-3">Device</th>
              <th className="p-3">Platform</th>
              <th className="p-3">Created</th>
              <th className="p-3">Last Used</th>
              <th className="p-3">Transports</th>
              <th className="p-3">Sync</th>
              <th className="p-3">Backup</th>
              <th className="p-3">Action</th>
            </tr>
          </thead>
          <tbody>
            {passkeys.map(p => (
              <tr key={p.id} className="border-b hover:bg-gray-50">
                <td className="p-3 font-medium">{p.deviceName}</td>
                <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs ${platformColor(p.platform)}`}>{p.platform}</span></td>
                <td className="p-3 text-gray-500">{p.created}</td>
                <td className="p-3 text-gray-500">{p.lastUsed}</td>
                <td className="p-3"><div className="flex flex-wrap gap-1">{p.transports.map(t => <span key={t} className="px-1.5 py-0.5 bg-gray-100 rounded text-xs font-mono">{t}</span>)}</div></td>
                <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs capitalize ${syncColor(p.syncStatus)}`}>{p.syncStatus.replace('-', ' ')}</span></td>
                <td className="p-3">{p.backupEligible ? <span className="text-green-600 text-xs">Eligible</span> : <span className="text-gray-400 text-xs">N/A</span>}</td>
                <td className="p-3"><button onClick={() => setRevokeTarget(p)} className="text-red-600 text-xs hover:underline">Revoke</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-3">
        <h2 className="text-lg font-semibold">Platform Authenticator Info</h2>
        <div className="grid grid-cols-3 gap-4">
          <div className="border rounded p-4">
            <div className="font-medium text-sm">Apple iCloud Keychain</div>
            <div className="text-xs text-gray-500 mt-1">Touch ID / Face ID. Synced across Apple devices via iCloud Keychain. Backup eligible.</div>
          </div>
          <div className="border rounded p-4">
            <div className="font-medium text-sm">Google Password Manager</div>
            <div className="text-xs text-gray-500 mt-1">Android biometrics + Chrome. Synced via Google Account. Backup eligible on supported devices.</div>
          </div>
          <div className="border rounded p-4">
            <div className="font-medium text-sm">Microsoft Windows Hello</div>
            <div className="text-xs text-gray-500 mt-1">Windows Hello (PIN/biometric). Device-bound, no cloud sync. Not backup eligible.</div>
          </div>
        </div>
      </section>

      {revokeTarget && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 max-w-md w-full mx-4 space-y-4">
            <h2 className="text-lg font-semibold">Revoke Passkey</h2>
            <p className="text-sm text-gray-600">You are about to revoke the passkey for <strong>{revokeTarget.deviceName}</strong> ({revokeTarget.platform}). The user will need to re-enroll this device to use passkey authentication again.</p>
            <div className="flex justify-end gap-3">
              <button onClick={() => setRevokeTarget(null)} className="px-4 py-2 border rounded text-sm">Cancel</button>
              <button onClick={confirmRevoke} className="px-4 py-2 bg-red-600 text-white rounded text-sm">Confirm Revoke</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

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
  const t = useTranslations();

  const [passkeys, setPasskeys] = useState<Passkey[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

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

  const enrollPasskey = async () => {
    try {
      const beginResp = await fetch('/api/v1/auth/webauthn/register/begin', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...authHeader() },
        body: JSON.stringify({ device_name: enrollDevice || 'New Device' }),
      });
      if (!beginResp.ok) throw new Error(`Registration begin failed: ${beginResp.status}`);
      const publicKey = await beginResp.json();

      // Use WebAuthn API to create credential
      const credential = await navigator.credentials.create({ publicKey });
      if (!credential) throw new Error('No credential returned');

      // Send credential to finish endpoint
      const finishResp = await fetch('/api/v1/auth/webauthn/register/finish', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...authHeader() },
        body: JSON.stringify(credential),
      });
      if (!finishResp.ok) throw new Error(`Registration finish failed: ${finishResp.status}`);

      setShowEnroll(false);
      setEnrollDevice('');
      // Reload passkeys from API
      const statusResp = await fetch('/api/v1/auth/passkeys/status', {
        headers: { ...authHeader(), 'Content-Type': 'application/json' },
      });
      if (statusResp.ok) {
        const data = await statusResp.json();
        setPasskeys(data.passkeys || (Array.isArray(data) ? data : []));
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Passkey enrollment failed');
    }
  };

  useEffect(() => {
    fetch('/api/v1/auth/passkeys/status', {
      headers: { ...authHeader(), 'Content-Type': 'application/json', 'X-Tenant-ID': '00000000-0000-0000-0000-000000000001' },
    })
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(data => {
        if (data && data.passkeys) setPasskeys(data.passkeys);
        else if (Array.isArray(data)) setPasskeys(data);
        setLoading(false);
      })
      .catch(e => { setError(e.message); setLoading(false); });
  }, []);

  const confirmRevoke = () => {
    if (revokeTarget) {
      setPasskeys(prev => prev.filter(p => p.id !== revokeTarget.id));
    }
    setRevokeTarget(null);
  };

  if (loading) return <div className="p-6"><p>{t("common.loading")}</p></div>;
  if (error) return <div className="p-6 text-red-600">Error: {error}</div>;

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t("passkeyMgmt.title")}</h1>
          <p className="text-gray-600">{t("passkeyMgmt.subtitle")}</p>
        </div>
        <button onClick={() => setShowEnroll(!showEnroll)} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">
          {showEnroll ? t("common.cancel") : t("passkeyMgmt.enrollPasskey")}
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
              <th scope="col" className="p-3">Device</th>
              <th scope="col" className="p-3">Platform</th>
              <th scope="col" className="p-3">Created</th>
              <th scope="col" className="p-3">Last Used</th>
              <th scope="col" className="p-3">Transports</th>
              <th scope="col" className="p-3">Sync</th>
              <th scope="col" className="p-3">Backup</th>
              <th scope="col" className="p-3">Action</th>
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
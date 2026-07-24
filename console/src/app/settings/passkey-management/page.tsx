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
      const rawKey = await beginResp.json();

      // Decode base64url fields to ArrayBuffer for WebAuthn API
      const b64urlToBuf = (val: unknown): ArrayBuffer => {
        if (val instanceof ArrayBuffer || val instanceof Uint8Array) return val as ArrayBuffer;
        if (typeof val !== 'string') return new ArrayBuffer(0);
        const b64 = val.replace(/-/g, '+').replace(/_/g, '/');
        const padded = b64 + '='.repeat((4 - b64.length % 4) % 4);
        const bin = atob(padded);
        const buf = new Uint8Array(bin.length);
        for (let i = 0; i < bin.length; i++) buf[i] = bin.charCodeAt(i);
        return buf.buffer;
      };

      const publicKey: PublicKeyCredentialCreationOptions = {
        ...rawKey,
        challenge: b64urlToBuf(rawKey.challenge),
        user: rawKey.user ? { ...rawKey.user, id: b64urlToBuf(rawKey.user.id) } : undefined,
        excludeCredentials: Array.isArray(rawKey.excludeCredentials)
          ? rawKey.excludeCredentials.map((c: any) => ({ ...c, id: b64urlToBuf(c.id) }))
          : [],
      };

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
      headers: { ...authHeader(), 'Content-Type': 'application/json', 'X-Tenant-ID': localStorage.getItem('ggid_tenant_id') || '' },
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
          <p className="text-gray-600 dark:text-gray-400">{t("passkeyMgmt.subtitle")}</p>
        </div>
        <button onClick={() => setShowEnroll(!showEnroll)} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">
          {showEnroll ? t("common.cancel") : t("passkeyMgmt.enrollPasskey")}
        </button>
      </div>

      {showEnroll && (
        <section className="bg-white dark:bg-gray-800 rounded-lg shadow p-6 space-y-4">
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
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold">{passkeys.length}</div>
          <div className="text-sm text-gray-500">Registered</div>
        </div>
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold text-green-600">{passkeys.filter(p => p.syncStatus === 'synced').length}</div>
          <div className="text-sm text-gray-500">Synced</div>
        </div>
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold text-blue-600">{passkeys.filter(p => p.backupEligible).length}</div>
          <div className="text-sm text-gray-500">Backup Eligible</div>
        </div>
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold">{new Set(passkeys.map(p => p.platform)).size}</div>
          <div className="text-sm text-gray-500">Platforms</div>
        </div>
      </div>

      <section className="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 dark:bg-gray-800">
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
            {passkeys.length === 0 ? (
              <tr>
                <td colSpan={8} className="p-8 text-center text-gray-500">
                  <div className="flex flex-col items-center gap-2">
                    <svg className="w-12 h-12 text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M15 7a4 4 0 11-8 0 4 4 0 018 0zM12 14v7m-3-3h6" /></svg>
                    <p className="text-sm font-medium">No passkeys registered yet</p>
                    <p className="text-xs text-gray-400">Click "Register Passkey" above to add your first passkey device.</p>
                  </div>
                </td>
              </tr>
            ) : passkeys.map(p => (
              <tr key={p.id} className="border-b hover:bg-gray-50 dark:hover:bg-gray-700 dark:bg-gray-800">
                <td className="p-3 font-medium">{p.deviceName}</td>
                <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs ${platformColor(p.platform)}`}>{p.platform}</span></td>
                <td className="p-3 text-gray-500">{p.created}</td>
                <td className="p-3 text-gray-500">{p.lastUsed}</td>
                <td className="p-3"><div className="flex flex-wrap gap-1">{p.transports.map(t => <span key={t} className="px-1.5 py-0.5 bg-gray-100 dark:bg-gray-700 rounded text-xs font-mono">{t}</span>)}</div></td>
                <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs capitalize ${syncColor(p.syncStatus)}`}>{p.syncStatus.replace('-', ' ')}</span></td>
                <td className="p-3">{p.backupEligible ? <span className="text-green-600 text-xs">Eligible</span> : <span className="text-gray-400 text-xs">N/A</span>}</td>
                <td className="p-3"><button onClick={() => setRevokeTarget(p)} className="text-red-600 text-xs hover:underline">Revoke</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <section className="bg-white dark:bg-gray-800 rounded-lg shadow p-6 space-y-3">
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
          <div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-md w-full mx-4 space-y-4">
            <h2 className="text-lg font-semibold">Revoke Passkey</h2>
            <p className="text-sm text-gray-600 dark:text-gray-400">You are about to revoke the passkey for <strong>{revokeTarget.deviceName}</strong> ({revokeTarget.platform}). The user will need to re-enroll this device to use passkey authentication again.</p>
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
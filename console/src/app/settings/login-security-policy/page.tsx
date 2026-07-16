'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface LockoutRecord {
  user: string;
  attempts: number;
  lockedAt: string;
  reason: string;
}

export default function LoginSecurityPolicyPage() {
  const t = useTranslations();

  const [maxAttempts, setMaxAttempts] = useState(5);
  const [lockoutDuration, setLockoutDuration] = useState(900);
  const [captchaAfter, setCaptchaAfter] = useState(3);
  const [enforceMfaAdmin, setEnforceMfaAdmin] = useState(true);
  const [anomalyAlert, setAnomalyAlert] = useState(true);
  const [ipAllowlist, setIpAllowlist] = useState<string[]>([]);
  const [ipBlocklist, setIpBlocklist] = useState<string[]>([]);
  const [newIp, setNewIp] = useState('');
  const [ipMode, setIpMode] = useState<'allow' | 'block'>('allow');
  const [lockouts, setLockouts] = useState<LockoutRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const addIp = () => {
    if (ipMode === 'allow') setIpAllowlist(prev => [...prev, newIp]);
    else setIpBlocklist(prev => [...prev, newIp]);
    setNewIp('');
  };

  const removeIp = (ip: string, mode: 'allow' | 'block') => {
    if (mode === 'allow') setIpAllowlist(prev => prev.filter(i => i !== ip));
    else setIpBlocklist(prev => prev.filter(i => i !== ip));
  };

  useEffect(() => {
    fetch('/api/v1/auth/password-policy/config', {
      headers: { ...authHeader(), 'Content-Type': 'application/json', 'X-Tenant-ID': '00000000-0000-0000-0000-000000000001' },
    })
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(data => {
        if (data) {
          if (data.max_attempts) setMaxAttempts(data.max_attempts);
          if (data.lockout_duration) setLockoutDuration(data.lockout_duration);
          if (data.captcha_after) setCaptchaAfter(data.captcha_after);
          if (data.enforce_mfa_admin !== undefined) setEnforceMfaAdmin(data.enforce_mfa_admin);
          if (data.anomaly_alert !== undefined) setAnomalyAlert(data.anomaly_alert);
          if (data.ip_allowlist) setIpAllowlist(data.ip_allowlist);
          if (data.ip_blocklist) setIpBlocklist(data.ip_blocklist);
          if (data.lockouts) setLockouts(data.lockouts);
        }
        setLoading(false);
      })
      .catch(e => { setError(e.message); setLoading(false); });
  }, []);

  if (loading) return <div className="p-6"><p>Loading...</p></div>;
  if (error) return <div className="p-6 text-red-600">Error: {error}</div>;

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Login Security Policy</h1>
        <p className="text-gray-600">Configure lockout, captcha, IP restrictions, and admin MFA enforcement.</p>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Lockout Settings</h2>
        <div className="grid grid-cols-3 gap-4">
          <div>
            <label className="text-sm font-medium">Max Attempts</label>
            <input aria-label="max Attempts" type="number" min={1} max={20} value={maxAttempts} onChange={e => setMaxAttempts(parseInt(e.target.value) || 5)} className="w-full border rounded px-2 py-1 text-sm mt-1" />
          </div>
          <div>
            <label className="text-sm font-medium">Lockout Duration: {lockoutDuration}s</label>
            <input aria-label="lockout Duration" type="range" min={60} max={3600} step={60} value={lockoutDuration} onChange={e => setLockoutDuration(parseInt(e.target.value))} className="w-full mt-2" />
          </div>
          <div>
            <label className="text-sm font-medium">Captcha After: {captchaAfter} attempts</label>
            <input aria-label="captcha After" type="range" min={1} max={10} value={captchaAfter} onChange={e => setCaptchaAfter(parseInt(e.target.value))} className="w-full mt-2" />
          </div>
        </div>
      </section>

      <div className="grid grid-cols-2 gap-4">
        <label className="flex items-center justify-between bg-white rounded-lg shadow p-4">
          <span className="text-sm font-medium">Enforce MFA for Admins</span>
          <input aria-label="Enforce mfa admin" type="checkbox" checked={enforceMfaAdmin} onChange={e => setEnforceMfaAdmin(e.target.checked)} className="rounded" />
        </label>
        <label className="flex items-center justify-between bg-white rounded-lg shadow p-4">
          <span className="text-sm font-medium">Login Anomaly Alerts</span>
          <input aria-label="Anomaly alert" type="checkbox" checked={anomalyAlert} onChange={e => setAnomalyAlert(e.target.checked)} className="rounded" />
        </label>
      </div>

      <div className="grid grid-cols-2 gap-6">
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">IP Allowlist</h2>
          <div className="space-y-2">
            {ipAllowlist.map(ip => (
              <div key={ip} className="flex items-center gap-2">
                <span className="font-mono text-xs flex-1">{ip}</span>
                <button onClick={() => removeIp(ip, 'allow')} className="text-red-600 text-xs">Remove</button>
              </div>
            ))}
          </div>
          <div className="flex gap-2">
            <input aria-label="CIDR" type="text" placeholder="CIDR" value={ipMode === 'allow' ? newIp : ''} onChange={e => { setNewIp(e.target.value); setIpMode('allow'); }} className="flex-1 border rounded px-2 py-1 text-sm font-mono" />
            <button onClick={() => { setIpMode('allow'); addIp(); }} className="px-3 py-1 bg-blue-600 text-white rounded text-sm">Add</button>
          </div>
        </section>

        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">IP Blocklist</h2>
          <div className="space-y-2">
            {ipBlocklist.map(ip => (
              <div key={ip} className="flex items-center gap-2">
                <span className="font-mono text-xs flex-1">{ip}</span>
                <button onClick={() => removeIp(ip, 'block')} className="text-red-600 text-xs">Remove</button>
              </div>
            ))}
          </div>
          <div className="flex gap-2">
            <input aria-label="CIDR" type="text" placeholder="CIDR" value={ipMode === 'block' ? newIp : ''} onChange={e => { setNewIp(e.target.value); setIpMode('block'); }} className="flex-1 border rounded px-2 py-1 text-sm font-mono" />
            <button onClick={() => { setIpMode('block'); addIp(); }} className="px-3 py-1 bg-red-600 text-white rounded text-sm">Add</button>
          </div>
        </section>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Account Lockout History</h2>
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th scope="col" className="p-3">User</th>
              <th scope="col" className="p-3">Attempts</th>
              <th scope="col" className="p-3">Locked At</th>
              <th scope="col" className="p-3">Reason</th>
              <th scope="col" className="p-3">Action</th>
            </tr>
          </thead>
          <tbody>
            {lockouts.map((l, idx) => (
              <tr key={idx} className="border-b">
                <td className="p-3 font-medium">{l.user}</td>
                <td className="p-3">{l.attempts}</td>
                <td className="p-3 text-gray-500">{l.lockedAt}</td>
                <td className="p-3 text-gray-600">{l.reason}</td>
                <td className="p-3"><button className="text-blue-600 text-xs hover:underline">Unlock</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </div>
  );
}

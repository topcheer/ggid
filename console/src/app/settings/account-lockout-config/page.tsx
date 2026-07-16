'use client';
import { useTranslations } from "@/lib/i18n";
import { useState, useEffect } from 'react';

interface LockoutRecord {
  user: string;
  attempts: number;
  lockedAt: string;
  unlockedAt: string;
  reason: string;
}

export default function AccountLockoutConfigPage() {
  const t = useTranslations();
  const [maxAttempts, setMaxAttempts] = useState(5);
  const [windowMinutes, setWindowMinutes] = useState(15);
  const [lockoutDuration, setLockoutDuration] = useState(30);
  const [captchaThreshold, setCaptchaThreshold] = useState(3);
  const [autoUnlock, setAutoUnlock] = useState(true);
  const [autoUnlockAfter, setAutoUnlockAfter] = useState(60);
  const [perIpTracking, setPerIpTracking] = useState(true);
  const [perUserTracking, setPerUserTracking] = useState(true);

  const [lockouts, setLockouts] = useState<LockoutRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch('/api/v1/auth/lockout-policy/config', {
      headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, 'Content-Type': 'application/json', 'X-Tenant-ID': '00000000-0000-0000-0000-000000000001' },
    })
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(data => {
        if (data) {
          if (data.max_attempts) setMaxAttempts(data.max_attempts);
          if (data.window_minutes) setWindowMinutes(data.window_minutes);
          if (data.lockout_duration) setLockoutDuration(data.lockout_duration);
          if (data.captcha_threshold) setCaptchaThreshold(data.captcha_threshold);
          if (data.auto_unlock !== undefined) setAutoUnlock(data.auto_unlock);
          if (data.auto_unlock_after) setAutoUnlockAfter(data.auto_unlock_after);
          if (data.per_ip_tracking !== undefined) setPerIpTracking(data.per_ip_tracking);
          if (data.per_user_tracking !== undefined) setPerUserTracking(data.per_user_tracking);
          if (data.lockouts) setLockouts(data.lockouts);
        }
        setLoading(false);
      })
      .catch(e => { setError(e.message); setLoading(false); });
  }, []);

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      {loading && <p className="text-gray-500">Loading...</p>}
      {error && <p className="text-red-600">Error: {error}</p>}
      <div>
        <h1 className="text-2xl font-bold">Account Lockout Configuration</h1>
        <p className="text-gray-600">Configure brute-force protection with per-IP and per-user tracking.</p>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Lockout Policy</h2>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="text-sm font-medium">Max Attempts</label>
            <input aria-label="max Attempts" type="number" min={1} max={20} value={maxAttempts} onChange={e => setMaxAttempts(parseInt(e.target.value) || 5)} className="w-full border rounded px-3 py-2 text-sm mt-1" />
          </div>
          <div>
            <label className="text-sm font-medium">Window (minutes)</label>
            <input aria-label="window Minutes" type="number" min={1} max={120} value={windowMinutes} onChange={e => setWindowMinutes(parseInt(e.target.value) || 15)} className="w-full border rounded px-3 py-2 text-sm mt-1" />
          </div>
        </div>

        <div>
          <label className="text-sm font-medium">Lockout Duration: {lockoutDuration}min</label>
          <input aria-label="lockout Duration" type="range" min={5} max={60} value={lockoutDuration} onChange={e => setLockoutDuration(parseInt(e.target.value))} className="w-full mt-2" />
          <div className="flex justify-between text-xs text-gray-400"><span>5min</span><span>60min</span></div>
        </div>

        <div>
          <label className="text-sm font-medium">Captcha After: {captchaThreshold} attempts</label>
          <input aria-label="captcha Threshold" type="number" min={1} max={10} value={captchaThreshold} onChange={e => setCaptchaThreshold(parseInt(e.target.value) || 3)} className="w-24 border rounded px-2 py-1 text-sm mt-1" />
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Auto-Unlock</h2>
        <label className="flex items-center justify-between">
          <span className="text-sm">Auto-unlock after duration</span>
          <input aria-label="Auto unlock" type="checkbox" checked={autoUnlock} onChange={e => setAutoUnlock(e.target.checked)} className="rounded" />
        </label>
        {autoUnlock && (
          <div className="flex items-center gap-3">
            <label className="text-sm">Unlock after:</label>
            <input aria-label="auto Unlock After" type="number" min={5} max={1440} value={autoUnlockAfter} onChange={e => setAutoUnlockAfter(parseInt(e.target.value) || 60)} className="w-24 border rounded px-2 py-1 text-sm" />
            <span className="text-sm text-gray-500">minutes</span>
          </div>
        )}
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Tracking Mode</h2>
        <label className="flex items-center justify-between">
          <span className="text-sm">Per-IP Tracking</span>
          <input aria-label="Per ip tracking" type="checkbox" checked={perIpTracking} onChange={e => setPerIpTracking(e.target.checked)} className="rounded" />
        </label>
        <label className="flex items-center justify-between">
          <span className="text-sm">Per-User Tracking</span>
          <input aria-label="Per user tracking" type="checkbox" checked={perUserTracking} onChange={e => setPerUserTracking(e.target.checked)} className="rounded" />
        </label>
        <p className="text-xs text-gray-400">Per-IP tracks attempts by source IP. Per-User tracks by username. Enable both for maximum protection.</p>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Lockout History</h2>
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th scope="col" className="p-3">User</th>
              <th scope="col" className="p-3">Attempts</th>
              <th scope="col" className="p-3">Locked At</th>
              <th scope="col" className="p-3">Unlocked At</th>
              <th scope="col" className="p-3">Reason</th>
              <th scope="col" className="p-3">Action</th>
            </tr>
          </thead>
          <tbody>
            {lockouts.length === 0 ? (
              <tr><td colSpan={6} className="p-3 text-center text-gray-400">No data available</td></tr>
            ) : lockouts.map((l, idx) => (
              <tr key={idx} className="border-b">
                <td className="p-3 font-medium">{l.user}</td>
                <td className="p-3">{l.attempts}</td>
                <td className="p-3 text-gray-500">{l.lockedAt}</td>
                <td className="p-3 text-gray-500">{l.unlockedAt}</td>
                <td className="p-3 text-gray-600">{l.reason}</td>
                <td className="p-3">{l.unlockedAt === '-' && <button className="text-blue-600 text-xs hover:underline">Unlock</button>}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </div>
  );
}

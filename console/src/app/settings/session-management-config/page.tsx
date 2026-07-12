'use client';
import { useState } from 'react';

export default function SessionManagementConfigPage() {
  const [idleTimeout, setIdleTimeout] = useState(30);
  const [absoluteTimeout, setAbsoluteTimeout] = useState(480);
  const [maxConcurrent, setMaxConcurrent] = useState(3);
  const [fixationPrevention, setFixationPrevention] = useState(true);
  const [bindIp, setBindIp] = useState(true);
  const [bindDevice, setBindDevice] = useState(false);
  const [bindGeo, setBindGeo] = useState(false);
  const [stepUpTimeout, setStepUpTimeout] = useState(300);
  const [storage, setStorage] = useState('redis');
  const [logoutBehavior, setLogoutBehavior] = useState('all_sessions');

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      <div><h1 className="text-2xl font-bold">Session Management Configuration</h1><p className="text-gray-600">Configure session lifetimes, concurrency, binding, and storage.</p></div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Session Lifetime</h2>
        <div><label className="text-sm font-medium">Idle Timeout: {idleTimeout}min</label><input type="range" min={5} max={120} value={idleTimeout} onChange={e => setIdleTimeout(parseInt(e.target.value))} className="w-full mt-2" /><div className="flex justify-between text-xs text-gray-400"><span>5min</span><span>2h</span></div></div>
        <div><label className="text-sm font-medium">Absolute Timeout: {absoluteTimeout}min ({Math.round(absoluteTimeout / 60)}h)</label><input type="range" min={60} max={1440} step={30} value={absoluteTimeout} onChange={e => setAbsoluteTimeout(parseInt(e.target.value))} className="w-full mt-2" /><div className="flex justify-between text-xs text-gray-400"><span>1h</span><span>24h</span></div></div>
        <div><label className="text-sm font-medium">Max Concurrent Sessions</label><input type="number" min={1} max={20} value={maxConcurrent} onChange={e => setMaxConcurrent(parseInt(e.target.value) || 3)} className="w-24 border rounded px-2 py-1 text-sm mt-1" /></div>
      </section>

      <div className="grid grid-cols-2 gap-4">
        <label className="flex items-center justify-between bg-white rounded-lg shadow p-4"><span className="text-sm font-medium">Session Fixation Prevention</span><input type="checkbox" checked={fixationPrevention} onChange={e => setFixationPrevention(e.target.checked)} className="rounded" /></label>
        <div className="bg-white rounded-lg shadow p-4"><label className="text-sm font-medium">Step-up Auth Timeout: {stepUpTimeout}s</label><input type="range" min={60} max={1800} step={60} value={stepUpTimeout} onChange={e => setStepUpTimeout(parseInt(e.target.value))} className="w-full mt-2" /></div>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Session Binding</h2>
        <p className="text-sm text-gray-500">Bind sessions to client characteristics to prevent session hijacking.</p>
        <div className="space-y-2">
          <label className="flex items-center justify-between"><span className="text-sm">Bind to IP address</span><input type="checkbox" checked={bindIp} onChange={e => setBindIp(e.target.checked)} className="rounded" /></label>
          <label className="flex items-center justify-between"><span className="text-sm">Bind to device fingerprint</span><input type="checkbox" checked={bindDevice} onChange={e => setBindDevice(e.target.checked)} className="rounded" /></label>
          <label className="flex items-center justify-between"><span className="text-sm">Bind to geo-location</span><input type="checkbox" checked={bindGeo} onChange={e => setBindGeo(e.target.checked)} className="rounded" /></label>
        </div>
        {(bindIp || bindDevice || bindGeo) && <p className="text-xs text-amber-600">Warning: strict binding may cause session invalidation when users change networks/devices.</p>}
      </section>

      <div className="grid grid-cols-2 gap-4">
        <div className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Session Storage</h2>
          <select value={storage} onChange={e => setStorage(e.target.value)} className="w-full border rounded px-3 py-2 text-sm">
            <option value="redis">Redis (server-side, revocable)</option>
            <option value="jwt">JWT (stateless, not revocable)</option>
            <option value="hybrid">Hybrid (JWT + Redis blacklist)</option>
          </select>
          {storage === 'jwt' && <p className="text-xs text-amber-600">JWT sessions cannot be revoked before expiry.</p>}
        </div>
        <div className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Logout Behavior</h2>
          <select value={logoutBehavior} onChange={e => setLogoutBehavior(e.target.value)} className="w-full border rounded px-3 py-2 text-sm">
            <option value="current">Current session only</option>
            <option value="all_sessions">All sessions for user</option>
            <option value="all_devices">All sessions across all devices</option>
          </select>
        </div>
      </div>
    </div>
  );
}
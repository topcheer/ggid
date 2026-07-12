'use client';
import { useState } from 'react';

export default function UserActivityDashboardPage() {
  const [period, setPeriod] = useState('24h');
  const [stats] = useState({ activeUsers: 142, totalLogins: 893, failedAttempts: 23, avgSessionMin: 47 });
  const [topUsers] = useState([
    { user: 'alice@ggid.io', logins: 45, lastActive: '14:30' },
    { user: 'bob@ggid.io', logins: 32, lastActive: '14:15' },
    { user: 'carol@ggid.io', logins: 28, lastActive: '13:45' },
    { user: 'dave@ggid.io', logins: 19, lastActive: '12:00' },
  ]);
  const [loginMethods] = useState([
    { method: 'Password', count: 520, pct: 58 },
    { method: 'SSO/SAML', count: 210, pct: 24 },
    { method: 'OIDC', count: 98, pct: 11 },
    { method: 'WebAuthn', count: 65, pct: 7 },
  ]);
  const [deviceBreakdown] = useState([
    { device: 'Desktop', count: 612, pct: 68 },
    { device: 'Mobile', count: 215, pct: 24 },
    { device: 'Tablet', count: 66, pct: 8 },
  ]);

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold">User Activity Dashboard</h1><p className="text-gray-600">Monitor user activity, login trends, and device breakdown.</p></div>
        <select value={period} onChange={e => setPeriod(e.target.value)} className="border rounded px-3 py-2 text-sm"><option value="24h">Last 24 hours</option><option value="7d">Last 7 days</option><option value="30d">Last 30 days</option></select>
      </div>

      <div className="grid grid-cols-4 gap-4">
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-3xl font-bold text-green-600">{stats.activeUsers}</div><div className="text-sm text-gray-500">Active Users</div></div>
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-3xl font-bold">{stats.totalLogins}</div><div className="text-sm text-gray-500">Total Logins</div></div>
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-3xl font-bold text-red-600">{stats.failedAttempts}</div><div className="text-sm text-gray-500">Failed Attempts</div></div>
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-3xl font-bold">{stats.avgSessionMin}min</div><div className="text-sm text-gray-500">Avg Session</div></div>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Login Trends</h2>
        <div className="flex items-end gap-2 h-32">
          {[45, 52, 38, 61, 73, 48, 89, 67, 54, 72, 81, 63].map((v, i) => (
            <div key={i} className="flex-1 flex flex-col items-center"><div className="w-full bg-blue-500 rounded-t" style={{ height: `${v}px` }} /><div className="text-xs text-gray-400 mt-1">{i * 2}h</div></div>
          ))}
        </div>
      </section>

      <div className="grid grid-cols-2 gap-6">
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Login Methods</h2>
          <div className="space-y-3">{loginMethods.map(m => (
            <div key={m.method} className="flex items-center gap-3"><span className="text-sm w-24">{m.method}</span><div className="flex-1 bg-gray-200 rounded-full h-4 overflow-hidden"><div className="h-4 bg-blue-500 rounded-full" style={{ width: `${m.pct}%` }} /></div><span className="text-xs text-gray-500 w-16">{m.count}</span></div>
          ))}</div>
        </section>
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Device Breakdown</h2>
          <div className="space-y-3">{deviceBreakdown.map(d => (
            <div key={d.device} className="flex items-center gap-3"><span className="text-sm w-20">{d.device}</span><div className="flex-1 bg-gray-200 rounded-full h-4 overflow-hidden"><div className="h-4 bg-purple-500 rounded-full" style={{ width: `${d.pct}%` }} /></div><span className="text-xs text-gray-500 w-16">{d.count}</span></div>
          ))}</div>
        </section>
      </div>

      <div className="grid grid-cols-2 gap-6">
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Top Active Users</h2>
          <table className="w-full text-sm"><thead className="bg-gray-50"><tr className="text-left"><th className="p-2">User</th><th className="p-2">Logins</th><th className="p-2">Last Active</th></tr></thead>
            <tbody>{topUsers.map(u => <tr key={u.user} className="border-b"><td className="p-2 font-medium">{u.user}</td><td className="p-2">{u.logins}</td><td className="p-2 text-gray-500">{u.lastActive}</td></tr>)}</tbody></table>
        </section>
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Failed Attempts</h2>
          <div className="flex items-end gap-2 h-32">
            {[3, 1, 5, 2, 8, 4, 1, 0, 2, 6, 3, 1].map((v, i) => (
              <div key={i} className="flex-1 flex flex-col items-center"><div className={`w-full rounded-t ${v > 5 ? 'bg-red-500' : v > 2 ? 'bg-amber-500' : 'bg-green-500'}`} style={{ height: `${v * 8}px` }} /></div>
            ))}
          </div>
        </section>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Risk Score Distribution</h2>
        <div className="flex items-end gap-2 h-24">
          {[85, 60, 35, 20, 12, 8, 5, 3, 2, 1].map((v, i) => (
            <div key={i} className="flex-1 flex flex-col items-center"><div className={`w-full rounded-t ${i < 2 ? 'bg-red-500' : i < 4 ? 'bg-amber-500' : 'bg-green-500'}`} style={{ height: `${v}px` }} /><div className="text-xs text-gray-400">{i * 10}</div></div>
          ))}
        </div>
      </section>
    </div>
  );
}
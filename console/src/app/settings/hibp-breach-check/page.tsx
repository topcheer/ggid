'use client';
import { useState } from 'react';

interface BreachRecord {
  id: string;
  user: string;
  breachName: string;
  date: string;
  dataClasses: string[];
}

export default function HibpBreachCheckPage() {
  const [enabled, setEnabled] = useState(true);
  const [apiKey, setApiKey] = useState('');
  const [checkOnLogin, setCheckOnLogin] = useState(true);
  const [checkOnPasswordChange, setCheckOnPasswordChange] = useState(true);
  const [checkOnRegister, setCheckOnRegister] = useState(true);
  const [autoForceReset, setAutoForceReset] = useState(false);
  const [notifyUser, setNotifyUser] = useState(true);
  const [notifyAdmin, setNotifyAdmin] = useState(true);
  const [lastCheck, setLastCheck] = useState('2026-07-12 14:00');

  const [breaches] = useState<BreachRecord[]>([
    { id: 'b1', user: 'alice@ggid.io', breachName: 'LinkedIn 2021', date: '2026-07-10', dataClasses: ['Emails', 'Passwords'] },
    { id: 'b2', user: 'bob@ggid.io', breachName: 'Adobe 2013', date: '2026-07-08', dataClasses: ['Emails', 'Passwords', 'Hints'] },
    { id: 'b3', user: 'carol@ggid.io', breachName: 'Collection #1', date: '2026-07-05', dataClasses: ['Emails', 'Passwords'] },
  ]);

  const [compromisedPasswords] = useState(['password123', 'admin', '123456', 'qwerty', 'letmein', 'welcome1']);

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">HIBP Breach Check</h1>
        <p className="text-gray-600">Have I Been Pwned integration for credential breach monitoring.</p>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">API Configuration</h2>
        <label className="flex items-center justify-between">
          <span className="text-sm font-medium">Enable HIBP Breach Check</span>
          <input type="checkbox" checked={enabled} onChange={e => setEnabled(e.target.checked)} className="rounded" />
        </label>
        <div>
          <label className="text-sm font-medium">HIBP API Key</label>
          <input type="password" placeholder="Enter HIBP API key" value={apiKey} onChange={e => setApiKey(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1" />
          <p className="text-xs text-gray-400 mt-1">Get your API key at https://haveibeenpwned.com/API/Key</p>
        </div>
        <div className="text-sm text-gray-500">Last check: {lastCheck}</div>
        <button onClick={() => setLastCheck(new Date().toISOString().slice(0, 16).replace('T', ' '))} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">Run Check Now</button>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Check Triggers</h2>
        <div className="space-y-2">
          <label className="flex items-center justify-between"><span className="text-sm">Check on Login</span><input type="checkbox" checked={checkOnLogin} onChange={e => setCheckOnLogin(e.target.checked)} className="rounded" /></label>
          <label className="flex items-center justify-between"><span className="text-sm">Check on Password Change</span><input type="checkbox" checked={checkOnPasswordChange} onChange={e => setCheckOnPasswordChange(e.target.checked)} className="rounded" /></label>
          <label className="flex items-center justify-between"><span className="text-sm">Check on Register</span><input type="checkbox" checked={checkOnRegister} onChange={e => setCheckOnRegister(e.target.checked)} className="rounded" /></label>
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Response Actions</h2>
        <div className="space-y-2">
          <label className="flex items-center justify-between"><span className="text-sm">Auto-Force Password Reset on Breach</span><input type="checkbox" checked={autoForceReset} onChange={e => setAutoForceReset(e.target.checked)} className="rounded" /></label>
          <label className="flex items-center justify-between"><span className="text-sm">Notify User</span><input type="checkbox" checked={notifyUser} onChange={e => setNotifyUser(e.target.checked)} className="rounded" /></label>
          <label className="flex items-center justify-between"><span className="text-sm">Notify Admin</span><input type="checkbox" checked={notifyAdmin} onChange={e => setNotifyAdmin(e.target.checked)} className="rounded" /></label>
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Breach History</h2>
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th className="p-3">User</th>
              <th className="p-3">Breach</th>
              <th className="p-3">Date</th>
              <th className="p-3">Data Classes</th>
            </tr>
          </thead>
          <tbody>
            {breaches.map(b => (
              <tr key={b.id} className="border-b">
                <td className="p-3 font-medium">{b.user}</td>
                <td className="p-3"><span className="px-2 py-0.5 bg-red-100 text-red-700 rounded text-xs">{b.breachName}</span></td>
                <td className="p-3 text-gray-500">{b.date}</td>
                <td className="p-3"><div className="flex flex-wrap gap-1">{b.dataClasses.map(d => <span key={d} className="px-1.5 py-0.5 bg-amber-100 text-amber-700 rounded text-xs">{d}</span>)}</div></td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Compromised Password Blocklist ({compromisedPasswords.length})</h2>
        <div className="flex flex-wrap gap-2">
          {compromisedPasswords.map(p => (
            <span key={p} className="px-2 py-1 bg-red-50 text-red-700 rounded text-xs font-mono">{p}</span>
          ))}
        </div>
        <p className="text-xs text-gray-400">Passwords matching these entries are rejected during registration and password change.</p>
      </section>
    </div>
  );
}
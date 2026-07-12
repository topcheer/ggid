'use client';
import { useState } from 'react';

export default function IpReputationConfigPage() {
  const [enabled, setEnabled] = useState(true);
  const [provider, setProvider] = useState('internal');
  const [checkInterval, setCheckInterval] = useState(300);
  const [blockThreshold, setBlockThreshold] = useState(80);
  const [suspiciousThreshold, setSuspiciousThreshold] = useState(50);
  const [allowlist, setAllowlist] = useState(['10.0.0.0/8', '172.16.0.0/12']);
  const [blocklist, setBlocklist] = useState(['203.0.113.0/24']);
  const [blockedCountries, setBlockedCountries] = useState<string[]>(['CN', 'RU']);
  const [asnBlocklist, setAsnBlocklist] = useState<string[]>(['AS12345', 'AS67890']);
  const [newIp, setNewIp] = useState('');
  const [newCountry, setNewCountry] = useState('');
  const [newAsn, setNewAsn] = useState('');
  const [listMode, setListMode] = useState<'allow' | 'block'>('allow');

  const countries = ['CN', 'RU', 'KP', 'IR', 'SY', 'CU', 'VN', 'TR', 'BR', 'IN'];
  const [stats] = useState({ checkedToday: 15420, blockedToday: 89, suspiciousToday: 234, avgScore: 12.5 });

  const addIp = () => {
    if (listMode === 'allow') setAllowlist(prev => [...prev, newIp]);
    else setBlocklist(prev => [...prev, newIp]);
    setNewIp('');
  };
  const removeIp = (ip: string, mode: 'allow' | 'block') => {
    if (mode === 'allow') setAllowlist(prev => prev.filter(i => i !== ip));
    else setBlocklist(prev => prev.filter(i => i !== ip));
  };
  const toggleCountry = (c: string) => setBlockedCountries(prev => prev.includes(c) ? prev.filter(x => x !== c) : [...prev, c]);
  const addAsn = () => { if (newAsn) { setAsnBlocklist(prev => [...prev, newAsn]); setNewAsn(''); } };
  const removeAsn = (a: string) => setAsnBlocklist(prev => prev.filter(x => x !== a));

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">IP Reputation Configuration</h1>
        <p className="text-gray-600">Configure IP reputation scoring, geo-blocking, and ASN filtering.</p>
      </div>

      <div className="grid grid-cols-4 gap-4">
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-2xl font-bold">{stats.checkedToday.toLocaleString()}</div><div className="text-sm text-gray-500">Checked (24h)</div></div>
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-2xl font-bold text-red-600">{stats.blockedToday}</div><div className="text-sm text-gray-500">Blocked (24h)</div></div>
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-2xl font-bold text-amber-600">{stats.suspiciousToday}</div><div className="text-sm text-gray-500">Suspicious (24h)</div></div>
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-2xl font-bold">{stats.avgScore}</div><div className="text-sm text-gray-500">Avg Score</div></div>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">General Settings</h2>
        <label className="flex items-center justify-between"><span className="text-sm font-medium">Enable IP Reputation</span><input type="checkbox" checked={enabled} onChange={e => setEnabled(e.target.checked)} className="rounded" /></label>
        <div className="grid grid-cols-2 gap-4">
          <div><label className="text-sm font-medium">Reputation Provider</label><select value={provider} onChange={e => setProvider(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1"><option value="internal">Internal (built-in database)</option><option value="external">External API (AbuseIPDB)</option><option value="hybrid">Hybrid (internal + external)</option></select></div>
          <div><label className="text-sm font-medium">Check Interval (s)</label><input type="number" min={60} value={checkInterval} onChange={e => setCheckInterval(parseInt(e.target.value) || 300)} className="w-full border rounded px-2 py-1 text-sm mt-1" /></div>
        </div>
        <div className="grid grid-cols-2 gap-4">
          <div><label className="text-sm font-medium">Block Threshold: {blockThreshold}</label><input type="range" min={0} max={100} value={blockThreshold} onChange={e => setBlockThreshold(parseInt(e.target.value))} className="w-full mt-2" /></div>
          <div><label className="text-sm font-medium">Suspicious Threshold: {suspiciousThreshold}</label><input type="range" min={0} max={100} value={suspiciousThreshold} onChange={e => setSuspiciousThreshold(parseInt(e.target.value))} className="w-full mt-2" /></div>
        </div>
      </section>

      <div className="grid grid-cols-2 gap-6">
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Allowlist</h2>
          <div className="space-y-2">{allowlist.map(ip => <div key={ip} className="flex items-center gap-2"><span className="font-mono text-xs flex-1">{ip}</span><button onClick={() => removeIp(ip, 'allow')} className="text-red-600 text-xs">Remove</button></div>)}</div>
          <div className="flex gap-2"><input type="text" placeholder="CIDR" value={listMode === 'allow' ? newIp : ''} onChange={e => { setNewIp(e.target.value); setListMode('allow'); }} className="flex-1 border rounded px-2 py-1 text-sm font-mono" /><button onClick={() => { setListMode('allow'); addIp(); }} className="px-3 py-1 bg-green-600 text-white rounded text-sm">Add</button></div>
        </section>
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Blocklist</h2>
          <div className="space-y-2">{blocklist.map(ip => <div key={ip} className="flex items-center gap-2"><span className="font-mono text-xs flex-1">{ip}</span><button onClick={() => removeIp(ip, 'block')} className="text-red-600 text-xs">Remove</button></div>)}</div>
          <div className="flex gap-2"><input type="text" placeholder="CIDR" value={listMode === 'block' ? newIp : ''} onChange={e => { setNewIp(e.target.value); setListMode('block'); }} className="flex-1 border rounded px-2 py-1 text-sm font-mono" /><button onClick={() => { setListMode('block'); addIp(); }} className="px-3 py-1 bg-red-600 text-white rounded text-sm">Add</button></div>
        </section>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Geo-Block (Country Level)</h2>
        <div className="flex flex-wrap gap-2">
          {countries.map(c => (
            <label key={c} className={`px-3 py-1 rounded text-sm cursor-pointer ${blockedCountries.includes(c) ? 'bg-red-100 text-red-700' : 'bg-gray-100 text-gray-600'}`}>
              <input type="checkbox" checked={blockedCountries.includes(c)} onChange={() => toggleCountry(c)} className="hidden" />{c}
            </label>
          ))}
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">ASN Blocklist</h2>
        <div className="flex flex-wrap gap-2">
          {asnBlocklist.map(a => <div key={a} className="flex items-center gap-1"><span className="px-2 py-1 bg-red-50 text-red-700 rounded text-xs font-mono">{a}</span><button onClick={() => removeAsn(a)} className="text-red-600 text-xs">x</button></div>)}
        </div>
        <div className="flex gap-2"><input type="text" placeholder="AS12345" value={newAsn} onChange={e => setNewAsn(e.target.value)} className="flex-1 border rounded px-2 py-1 text-sm font-mono" /><button onClick={addAsn} className="px-3 py-1 bg-red-600 text-white rounded text-sm">Add</button></div>
      </section>
    </div>
  );
}
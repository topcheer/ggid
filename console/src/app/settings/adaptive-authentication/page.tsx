'use client';
import { useState } from 'react';

interface RiskRule {
  id: string;
  condition: string;
  action: string;
  enabled: boolean;
}

interface DeviceTrust {
  id: string;
  deviceId: string;
  name: string;
  trustLevel: string;
  lastSeen: string;
}

export default function AdaptiveAuthenticationPage() {
  const [rules, setRules] = useState<RiskRule[]>([
    { id: 'r1', condition: 'New device login', action: 'step_up_mfa', enabled: true },
    { id: 'r2', condition: 'Login from new country', action: 'block', enabled: true },
    { id: 'r3', condition: 'Login outside business hours', action: 'step_up_mfa', enabled: true },
    { id: 'r4', condition: 'Impossible travel detected', action: 'block', enabled: true },
    { id: 'r5', condition: 'Failed password 3+ times', action: 'lock_account', enabled: true },
  ]);

  const [thresholds, setThresholds] = useState([
    { level: 'low', minScore: 0, maxScore: 25, action: 'allow' },
    { level: 'medium', minScore: 25, maxScore: 50, action: 'step_up' },
    { level: 'high', minScore: 50, maxScore: 75, action: 'challenge_mfa' },
    { level: 'critical', minScore: 75, maxScore: 100, action: 'block' },
  ]);

  const [ipAllowlist, setIpAllowlist] = useState(['10.0.0.0/8', '172.16.0.0/12', '192.168.1.0/24']);
  const [ipBlocklist, setIpBlocklist] = useState(['203.0.113.0/24', '198.51.100.50']);
  const [newIp, setNewIp] = useState('');
  const [ipMode, setIpMode] = useState<'allow' | 'block'>('allow');

  const [devices, setDevices] = useState<DeviceTrust[]>([
    { id: 'd1', deviceId: 'dev-001', name: 'MacBook Pro', trustLevel: 'trusted', lastSeen: '2026-07-12' },
    { id: 'd2', deviceId: 'dev-002', name: 'iPhone 15', trustLevel: 'trusted', lastSeen: '2026-07-11' },
    { id: 'd3', deviceId: 'dev-003', name: 'Unknown Device', trustLevel: 'untrusted', lastSeen: '2026-07-10' },
  ]);

  const [realTimeEval, setRealTimeEval] = useState(true);
  const [adaptiveMfa, setAdaptiveMfa] = useState(true);
  const [showAddRule, setShowAddRule] = useState(false);
  const [newRule, setNewRule] = useState({ condition: '', action: 'step_up_mfa' });

  const actionColor = (a: string): string =>
    a === 'block' ? 'bg-red-100 text-red-700' :
    a === 'lock_account' ? 'bg-red-100 text-red-700' :
    a === 'challenge_mfa' ? 'bg-amber-100 text-amber-700' :
    a === 'step_up_mfa' || a === 'step_up' ? 'bg-blue-100 text-blue-700' :
    'bg-green-100 text-green-700';

  const toggleRule = (id: string) => {
    setRules(prev => prev.map(r => r.id === id ? { ...r, enabled: !r.enabled } : r));
  };

  const addRule = () => {
    setRules(prev => [...prev, { id: `r${prev.length + 1}`, condition: newRule.condition, action: newRule.action, enabled: true }]);
    setShowAddRule(false);
    setNewRule({ condition: '', action: 'step_up_mfa' });
  };

  const addIp = () => {
    if (ipMode === 'allow') setIpAllowlist(prev => [...prev, newIp]);
    else setIpBlocklist(prev => [...prev, newIp]);
    setNewIp('');
  };

  const removeIp = (ip: string, mode: 'allow' | 'block') => {
    if (mode === 'allow') setIpAllowlist(prev => prev.filter(i => i !== ip));
    else setIpBlocklist(prev => prev.filter(i => i !== ip));
  };

  const trustColor = (t: string): string =>
    t === 'trusted' ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700';

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Adaptive Authentication</h1>
        <p className="text-gray-600">Risk-based authentication rules, step-up triggers, and adaptive MFA policies.</p>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <label className="flex items-center justify-between bg-white rounded-lg shadow p-4">
          <span className="text-sm font-medium">Real-time Risk Evaluation</span>
          <input type="checkbox" checked={realTimeEval} onChange={e => setRealTimeEval(e.target.checked)} className="rounded" />
        </label>
        <label className="flex items-center justify-between bg-white rounded-lg shadow p-4">
          <span className="text-sm font-medium">Adaptive MFA Policy</span>
          <input type="checkbox" checked={adaptiveMfa} onChange={e => setAdaptiveMfa(e.target.checked)} className="rounded" />
        </label>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">Risk-Based Auth Rules</h2>
          <button onClick={() => setShowAddRule(!showAddRule)} className="px-3 py-1 bg-blue-600 text-white rounded text-sm">
            {showAddRule ? 'Cancel' : 'Add Rule'}
          </button>
        </div>

        {showAddRule && (
          <div className="flex gap-3 border rounded p-3">
            <input type="text" placeholder="Condition (e.g. Login from TOR exit node)" value={newRule.condition} onChange={e => setNewRule(prev => ({ ...prev, condition: e.target.value }))} className="flex-1 border rounded px-3 py-1.5 text-sm" />
            <select value={newRule.action} onChange={e => setNewRule(prev => ({ ...prev, action: e.target.value }))} className="border rounded px-3 py-1.5 text-sm">
              <option value="allow">Allow</option>
              <option value="step_up_mfa">Step-up MFA</option>
              <option value="challenge_mfa">Challenge MFA</option>
              <option value="block">Block</option>
              <option value="lock_account">Lock Account</option>
            </select>
            <button onClick={addRule} disabled={!newRule.condition} className="px-3 py-1.5 bg-blue-600 text-white rounded text-sm disabled:opacity-50">Add</button>
          </div>
        )}

        <div className="space-y-2">
          {rules.map(r => (
            <div key={r.id} className="flex items-center gap-3 border-b pb-2">
              <label className="flex items-center gap-2">
                <input type="checkbox" checked={r.enabled} onChange={() => toggleRule(r.id)} className="rounded" />
                <span className={`text-sm ${r.enabled ? '' : 'text-gray-400'}`}>{r.condition}</span>
              </label>
              <span className="text-gray-300">{'->'}</span>
              <span className={`px-2 py-0.5 rounded text-xs ${actionColor(r.action)}`}>{r.action.replace('_', ' ')}</span>
            </div>
          ))}
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Risk Score Thresholds</h2>
        <div className="space-y-3">
          {thresholds.map(t => (
            <div key={t.level} className="flex items-center gap-4">
              <span className={`px-2 py-0.5 rounded text-xs capitalize ${
                t.level === 'critical' ? 'bg-red-100 text-red-700' :
                t.level === 'high' ? 'bg-amber-100 text-amber-700' :
                t.level === 'medium' ? 'bg-yellow-100 text-yellow-700' :
                'bg-green-100 text-green-700'
              }`}>{t.level}</span>
              <span className="text-sm text-gray-500">{t.minScore}-{t.maxScore}</span>
              <span className="text-gray-300">{'->'}</span>
              <span className={`px-2 py-0.5 rounded text-xs ${actionColor(t.action)}`}>{t.action.replace('_', ' ')}</span>
            </div>
          ))}
        </div>
      </section>

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
            <input type="text" placeholder="CIDR (e.g. 10.0.0.0/8)" value={ipMode === 'allow' ? newIp : ''} onChange={e => setNewIp(e.target.value)} className="flex-1 border rounded px-2 py-1 text-sm font-mono" />
            <button onClick={addIp} className="px-3 py-1 bg-blue-600 text-white rounded text-sm">Add</button>
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
            <input type="text" placeholder="IP or CIDR" value={ipMode === 'block' ? newIp : ''} onChange={e => { setNewIp(e.target.value); setIpMode('block'); }} className="flex-1 border rounded px-2 py-1 text-sm font-mono" />
            <button onClick={() => { setIpMode('block'); addIp(); }} className="px-3 py-1 bg-red-600 text-white rounded text-sm">Add</button>
          </div>
        </section>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Device Trust List</h2>
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th className="p-3">Device</th>
              <th className="p-3">Device ID</th>
              <th className="p-3">Trust Level</th>
              <th className="p-3">Last Seen</th>
            </tr>
          </thead>
          <tbody>
            {devices.map(d => (
              <tr key={d.id} className="border-b">
                <td className="p-3 font-medium">{d.name}</td>
                <td className="p-3 font-mono text-xs text-gray-500">{d.deviceId}</td>
                <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs capitalize ${trustColor(d.trustLevel)}`}>{d.trustLevel}</span></td>
                <td className="p-3 text-gray-500">{d.lastSeen}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </div>
  );
}
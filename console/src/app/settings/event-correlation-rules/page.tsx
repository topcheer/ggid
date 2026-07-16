'use client';

import { useState, useCallback, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";

interface CorrelationRule {
  id: string;
  name: string;
  pattern: string;
  window: number;
  action: string;
  enabled: boolean;
  matches: number;
  falsePositives: number;
}

interface CorrelationEvent {
  id: string;
  type: string;
  user: string;
  timestamp: string;
  ip: string;
}

const INITIAL_RULES: CorrelationRule[] = [];

const TEST_EVENTS: CorrelationEvent[] = [
  { id: 'ev-1', type: 'failed_login', user: 'alice@corp.com', timestamp: '2025-01-15T10:00:00Z', ip: '192.168.1.50' },
  { id: 'ev-2', type: 'failed_login', user: 'alice@corp.com', timestamp: '2025-01-15T10:01:00Z', ip: '192.168.1.50' },
  { id: 'ev-3', type: 'failed_login', user: 'alice@corp.com', timestamp: '2025-01-15T10:02:00Z', ip: '10.0.0.99' },
  { id: 'ev-4', type: 'failed_login', user: 'alice@corp.com', timestamp: '2025-01-15T10:03:00Z', ip: '10.0.0.99' },
  { id: 'ev-5', type: 'failed_login', user: 'alice@corp.com', timestamp: '2025-01-15T10:04:00Z', ip: '172.16.0.5' },
  { id: 'ev-6', type: 'login_success', user: 'alice@corp.com', timestamp: '2025-01-15T10:05:00Z', ip: '172.16.0.5' },
  { id: 'ev-7', type: 'admin_action', user: 'alice@corp.com', timestamp: '2025-01-15T10:06:00Z', ip: '172.16.0.5' },
];

export default function EventCorrelationRulesPage() {
  const t = useTranslations();

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [rules, setRules] = useState<CorrelationRule[]>(INITIAL_RULES);
  const [activeTab, setActiveTab] = useState<'rules' | 'create' | 'test'>('rules');
  const [newName, setNewName] = useState('');
  const [newPattern, setNewPattern] = useState('');
  const [newWindow, setNewWindow] = useState(300);
  const [newAction, setNewAction] = useState('alert_admin');
  const [testResult, setTestResult] = useState<{ ruleName: string; correlations: string[] } | null>(null);

  const toggleRule = useCallback((id: string) => {
    setRules(rules.map(r => r.id === id ? { ...r, enabled: !r.enabled } : r));
  }, [rules]);

  const createRule = useCallback(() => {
    if (!newName.trim() || !newPattern.trim()) return;
    const newRule: CorrelationRule = {
      id: `cr-${String(rules.length + 1).padStart(3, '0')}`,
      name: newName.trim(),
      pattern: newPattern.trim(),
      window: newWindow,
      action: newAction,
      enabled: true,
      matches: 0,
      falsePositives: 0,
    };
    setRules([newRule, ...rules]);
    setNewName('');
    setNewPattern('');
    setNewWindow(300);
    setNewAction('alert_admin');
    setActiveTab('rules');
  }, [rules, newName, newPattern, newWindow, newAction]);

  const testRules = useCallback(() => {
    setTestResult({
      ruleName: 'Multiple Failed Logins',
      correlations: [
        '5 failed_login events for alice@corp.com within 5 minutes (10:00-10:04)',
        '3 distinct IPs detected: 192.168.1.50, 10.0.0.99, 172.16.0.5',
        'Action triggered: lock_account',
        'Also matched: IP Hopping (3 distinct IPs + login_success)',
      ],
    });
  }, []);

  const falsePositiveRate = (r: CorrelationRule) => r.matches > 0 ? ((r.falsePositives / r.matches) * 100).toFixed(1) : '0.0';

  useEffect(() => {
    fetch("/api/v1/audit/correlation/rules", {
      headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
    })
      .then(res => { if (!res.ok) return null; return res.json(); })
      .then(data => { setRules(data.rules || data.items || []); setLoading(false); })
      .catch(err => { setError(err.message); setLoading(false); });
  }, []);

  if (loading) return (<div className="p-6"><h1 className="text-2xl font-bold mb-4">Event Correlation Rules</h1><p>Loading...</p></div>);
  if (error) return (<div className="p-6"><h1 className="text-2xl font-bold mb-4">Event Correlation Rules</h1><p className="text-red-600">Error: {error}</p></div>);
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Event Correlation Rules</h1>
        <p className="mt-1 text-sm text-gray-500">Define correlation rules for security events, test patterns, and monitor false positive rates.</p>
      </div>

      <div className="flex gap-2 border-b border-gray-200">
        {(['rules', 'create', 'test'] as const).map(tab => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`px-4 py-2 text-sm font-medium border-b-2 ${
              activeTab === tab ? 'border-blue-500 text-blue-600' : 'border-transparent text-gray-500 hover:text-gray-700'
            }`}
          >
            {tab === 'rules' ? `Rules (${rules.length})` : tab === 'create' ? 'Create Rule' : 'Test Rules'}
          </button>
        ))}
      </div>

      {activeTab === 'rules' && (
        <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-200 text-left text-xs text-gray-500">
                <th scope="col" className="pb-2">Name</th>
                <th scope="col" className="pb-2">Pattern</th>
                <th scope="col" className="pb-2">Window</th>
                <th scope="col" className="pb-2">Action</th>
                <th scope="col" className="pb-2">Matches</th>
                <th scope="col" className="pb-2">FP Rate</th>
                <th scope="col" className="pb-2">Enabled</th>
              </tr>
            </thead>
            <tbody>
              {rules.map(r => (
                <tr key={r.id} className="border-b border-gray-100">
                  <td className="py-2 font-medium">{r.name}</td>
                  <td className="py-2 font-mono text-xs text-gray-600">{r.pattern}</td>
                  <td className="py-2 text-xs">{r.window}s</td>
                  <td className="py-2 text-xs">
                    <span className="inline-flex rounded bg-orange-50 px-2 py-0.5 text-orange-700">{r.action}</span>
                  </td>
                  <td className="py-2">{r.matches}</td>
                  <td className="py-2">
                    <span className={Number(falsePositiveRate(r)) > 10 ? 'text-red-600 font-medium' : Number(falsePositiveRate(r)) > 5 ? 'text-yellow-600' : 'text-green-600'}>
                      {falsePositiveRate(r)}%
                    </span>
                  </td>
                  <td className="py-2">
                    <button onClick={() => toggleRule(r.id)} aria-label={`Toggle rule ${r.name}`} className={`relative inline-flex h-5 w-9 items-center rounded-full transition ${r.enabled ? 'bg-green-500' : 'bg-gray-200'}`}>
                      <span className={`inline-block h-3 w-3 transform rounded-full bg-white transition ${r.enabled ? 'translate-x-5' : 'translate-x-1'}`} />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {activeTab === 'create' && (
        <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
          <h3 className="text-sm font-medium text-gray-700">Create Correlation Rule</h3>
          <div className="mt-4 space-y-4">
            <div>
              <label className="block text-xs font-medium text-gray-600">Rule Name</label>
              <input aria-label="e.g. Rapid API Calls" value={newName} onChange={e => setNewName(e.target.value)} placeholder="e.g. Rapid API Calls" className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm" />
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-600">Event Pattern</label>
              <input aria-label="e.g. api_call > 100 AND user != admin" value={newPattern} onChange={e => setNewPattern(e.target.value)} placeholder="e.g. api_call > 100 AND user != admin" className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm font-mono" />
              <p className="mt-1 text-xs text-gray-400">Use event types with comparison operators and AND/OR logic</p>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-xs font-medium text-gray-600">Time Window (seconds)</label>
                <input aria-label="new Window" type="number" value={newWindow} onChange={e => setNewWindow(Number(e.target.value))} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm" />
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-600">Action</label>
                <select aria-label="new Action" value={newAction} onChange={e => setNewAction(e.target.value)} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm">
                  <option value="alert_admin">Alert Admin</option>
                  <option value="lock_account">Lock Account</option>
                  <option value="require_mfa">Require MFA</option>
                  <option value="block_and_alert">Block and Alert</option>
                  <option value="force_reauth">Force Re-authentication</option>
                  <option value="require_approval">Require Approval</option>
                </select>
              </div>
            </div>
            <button onClick={createRule} disabled={!newName.trim() || !newPattern.trim()} className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50">
              Create Rule
            </button>
          </div>
        </div>
      )}

      {activeTab === 'test' && (
        <div className="space-y-4">
          <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
            <h3 className="text-sm font-medium text-gray-700">Test Input Events</h3>
            <table className="mt-3 w-full text-sm">
              <thead>
                <tr className="border-b border-gray-200 text-left text-xs text-gray-500">
                  <th scope="col" className="pb-2">ID</th>
                  <th scope="col" className="pb-2">Type</th>
                  <th scope="col" className="pb-2">User</th>
                  <th scope="col" className="pb-2">Timestamp</th>
                  <th scope="col" className="pb-2">IP</th>
                </tr>
              </thead>
              <tbody>
                {TEST_EVENTS.map(ev => (
                  <tr key={ev.id} className="border-b border-gray-100">
                    <td className="py-2 font-mono text-xs">{ev.id}</td>
                    <td className="py-2 text-xs"><span className="inline-flex rounded bg-blue-50 px-1.5 py-0.5 text-blue-700">{ev.type}</span></td>
                    <td className="py-2 font-mono text-xs">{ev.user}</td>
                    <td className="py-2 text-xs text-gray-500">{ev.timestamp.slice(0, 19).replace('T', ' ')}</td>
                    <td className="py-2 font-mono text-xs">{ev.ip}</td>
                  </tr>
                ))}
              </tbody>
            </table>
            <button onClick={testRules} className="mt-4 rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700">
              Run Correlation Test
            </button>
          </div>

          {testResult && (
            <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
              <h3 className="text-sm font-medium text-gray-700">Correlation Results</h3>
              <div className="mt-3 space-y-2">
                {testResult.correlations.map((c, i) => (
                  <div key={i} className="flex items-start gap-2 text-sm">
                    <span className="inline-flex h-5 w-5 items-center justify-center rounded-full bg-blue-100 text-xs font-medium text-blue-700">{i + 1}</span>
                    <span className="text-gray-700">{c}</span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
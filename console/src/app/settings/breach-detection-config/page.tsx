'use client';

import { useState, useCallback } from 'react';

interface BreachRecord {
  id: string;
  user: string;
  breachName: string;
  breachDate: string;
  dataClasses: string[];
  compromisedPassword: boolean;
  resolved: boolean;
}

interface CompromisedPassword {
  hash: string;
  count: number;
  lastSeen: string;
}

const INITIAL_BREACHES: BreachRecord[] = [
  { id: 'br-001', user: 'alice@corp.com', breachName: 'Adobe', breachDate: '2013-10-04', dataClasses: ['Emails', 'Password Hints', 'Encrypted Passwords'], compromisedPassword: true, resolved: true },
  { id: 'br-002', user: 'alice@corp.com', breachName: 'LinkedIn', breachDate: '2016-05-18', dataClasses: ['Emails', 'Passwords'], compromisedPassword: true, resolved: true },
  { id: 'br-003', user: 'bob@corp.com', breachName: 'Collection #1', breachDate: '2019-01-07', dataClasses: ['Emails', 'Passwords'], compromisedPassword: true, resolved: false },
  { id: 'br-004', user: 'charlie@corp.com', breachName: 'Dropbox', breachDate: '2012-07-01', dataClasses: ['Emails', 'Passwords'], compromisedPassword: false, resolved: true },
  { id: 'br-005', user: 'bob@corp.com', breachName: 'MyFitnessPal', breachDate: '2018-02-01', dataClasses: ['Emails', 'Usernames'], compromisedPassword: false, resolved: true },
];

const COMPROMISED_PASSWORDS: CompromisedPassword[] = [
  { hash: '5f4dcc3b5aa765d6...', count: 2340822, lastSeen: '2025-01-10T08:00:00Z' },
  { hash: '098f6bcd4621d373...', count: 892441, lastSeen: '2025-01-12T08:00:00Z' },
  { hash: 'e10adc3949ba59ab...', count: 1567432, lastSeen: '2025-01-08T08:00:00Z' },
  { hash: '25d55ad283aa400a...', count: 423871, lastSeen: '2025-01-14T08:00:00Z' },
];

export default function BreachDetectionConfigPage() {
  const [hibpEnabled, setHibpEnabled] = useState(true);
  const [apiKey, setApiKey] = useState('********-****-****-****-************');
  const [checkOnLogin, setCheckOnLogin] = useState(true);
  const [checkOnPasswordChange, setCheckOnPasswordChange] = useState(true);
  const [checkOnRegister, setCheckOnRegister] = useState(true);
  const [autoForceReset, setAutoForceReset] = useState(true);
  const [notifyEmail, setNotifyEmail] = useState(true);
  const [notifyAdmin, setNotifyAdmin] = useState(true);
  const [breaches, setBreaches] = useState<BreachRecord[]>(INITIAL_BREACHES);
  const [filterUser, setFilterUser] = useState('all');
  const [activeTab, setActiveTab] = useState<'config' | 'history' | 'passwords'>('config');

  const lastCheck = '2025-01-15T08:00:00Z';
  const rateLimitRemaining = 42;
  const rateLimitTotal = 50;

  const filteredBreaches = filterUser === 'all' ? breaches : breaches.filter(b => b.user === filterUser);

  const resolveBreach = useCallback((id: string) => {
    setBreaches(breaches.map(b => b.id === id ? { ...b, resolved: true } : b));
  }, [breaches]);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Breach Detection Configuration</h1>
        <p className="mt-1 text-sm text-gray-500">Configure HIBP integration, breach checking policies, and compromised password detection.</p>
      </div>

      <div className="flex gap-2 border-b border-gray-200">
        {(['config', 'history', 'passwords'] as const).map(tab => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`px-4 py-2 text-sm font-medium border-b-2 ${
              activeTab === tab ? 'border-blue-500 text-blue-600' : 'border-transparent text-gray-500 hover:text-gray-700'
            }`}
          >
            {tab === 'config' ? 'Configuration' : tab === 'history' ? 'Breach History' : 'Compromised Passwords'}
          </button>
        ))}
      </div>

      {activeTab === 'config' && (
        <div className="space-y-4">
          <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
            <h3 className="text-sm font-medium text-gray-700">HIBP Integration</h3>
            <div className="mt-4 space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <span className="text-sm font-medium text-gray-700">HIBP API Enabled</span>
                  <p className="text-xs text-gray-400">Enable breach checking via Have I Been Pwned API</p>
                </div>
                <button onClick={() => setHibpEnabled(!hibpEnabled)} className={`relative inline-flex h-6 w-11 items-center rounded-full transition ${hibpEnabled ? 'bg-blue-600' : 'bg-gray-200'}`}>
                  <span className={`inline-block h-4 w-4 transform rounded-full bg-white transition ${hibpEnabled ? 'translate-x-6' : 'translate-x-1'}`} />
                </button>
              </div>

              <div>
                <label className="block text-xs font-medium text-gray-600">HIBP API Key</label>
                <input
                  type="password"
                  value={apiKey}
                  onChange={e => setApiKey(e.target.value)}
                  className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm font-mono"
                />
              </div>

              <div className="flex items-center gap-4 text-xs text-gray-500">
                <span>Last Check: {lastCheck.slice(0, 19).replace('T', ' ')}</span>
                <span>Rate Limit: {rateLimitRemaining}/{rateLimitTotal} requests remaining</span>
              </div>
            </div>
          </div>

          <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
            <h3 className="text-sm font-medium text-gray-700">Check Triggers</h3>
            <div className="mt-4 space-y-3">
              {[
                { label: 'Check on Login', desc: 'Check for breaches when user logs in', value: checkOnLogin, setter: setCheckOnLogin },
                { label: 'Check on Password Change', desc: 'Check when user changes password', value: checkOnPasswordChange, setter: setCheckOnPasswordChange },
                { label: 'Check on Register', desc: 'Check when new user registers', value: checkOnRegister, setter: setCheckOnRegister },
                { label: 'Auto Force Password Reset', desc: 'Automatically force password reset for compromised accounts', value: autoForceReset, setter: setAutoForceReset },
                { label: 'Notify User by Email', desc: 'Send email notification to affected users', value: notifyEmail, setter: setNotifyEmail },
                { label: 'Notify Admin', desc: 'Send notification to administrators', value: notifyAdmin, setter: setNotifyAdmin },
              ].map(item => (
                <div key={item.label} className="flex items-center justify-between">
                  <div>
                    <span className="text-sm text-gray-700">{item.label}</span>
                    <p className="text-xs text-gray-400">{item.desc}</p>
                  </div>
                  <button onClick={() => item.setter(!item.value)} className={`relative inline-flex h-5 w-9 items-center rounded-full transition ${item.value ? 'bg-green-500' : 'bg-gray-200'}`}>
                    <span className={`inline-block h-3 w-3 transform rounded-full bg-white transition ${item.value ? 'translate-x-5' : 'translate-x-1'}`} />
                  </button>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {activeTab === 'history' && (
        <div className="space-y-4">
          <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
            <div className="flex items-center gap-4 mb-4">
              <label className="text-sm font-medium text-gray-700">Filter by User:</label>
              <select value={filterUser} onChange={e => setFilterUser(e.target.value)} className="rounded-md border border-gray-300 px-3 py-1.5 text-sm">
                <option value="all">All Users</option>
                <option value="alice@corp.com">alice@corp.com</option>
                <option value="bob@corp.com">bob@corp.com</option>
                <option value="charlie@corp.com">charlie@corp.com</option>
              </select>
            </div>

            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-200 text-left text-xs text-gray-500">
                  <th className="pb-2">User</th>
                  <th className="pb-2">Breach</th>
                  <th className="pb-2">Date</th>
                  <th className="pb-2">Data Classes</th>
                  <th className="pb-2">Password Compromised</th>
                  <th className="pb-2">Status</th>
                  <th className="pb-2">Action</th>
                </tr>
              </thead>
              <tbody>
                {filteredBreaches.map(b => (
                  <tr key={b.id} className="border-b border-gray-100">
                    <td className="py-2 text-xs font-mono">{b.user}</td>
                    <td className="py-2 font-medium">{b.breachName}</td>
                    <td className="py-2 text-xs text-gray-500">{b.breachDate}</td>
                    <td className="py-2">
                      <div className="flex flex-wrap gap-1">
                        {b.dataClasses.map(dc => <span key={dc} className="inline-flex rounded bg-gray-100 px-1.5 py-0.5 text-[10px] text-gray-600">{dc}</span>)}
                      </div>
                    </td>
                    <td className="py-2">
                      {b.compromisedPassword ? <span className="text-red-600 text-xs font-medium">Yes</span> : <span className="text-gray-400 text-xs">No</span>}
                    </td>
                    <td className="py-2">
                      <span className={`inline-flex rounded px-2 py-0.5 text-xs ${b.resolved ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'}`}>{b.resolved ? 'Resolved' : 'Unresolved'}</span>
                    </td>
                    <td className="py-2">
                      {!b.resolved && <button onClick={() => resolveBreach(b.id)} className="text-xs text-blue-600 hover:underline">Resolve</button>}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {activeTab === 'passwords' && (
        <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
          <h3 className="text-sm font-medium text-gray-700">Compromised Password List ({COMPROMISED_PASSWORDS.length})</h3>
          <table className="mt-2 w-full text-sm">
            <thead>
              <tr className="border-b border-gray-200 text-left text-xs text-gray-500">
                <th className="pb-2">Hash (SHA1)</th>
                <th className="pb-2">Occurrence Count</th>
                <th className="pb-2">Last Seen</th>
              </tr>
            </thead>
            <tbody>
              {COMPROMISED_PASSWORDS.map((p, i) => (
                <tr key={i} className="border-b border-gray-100">
                  <td className="py-2 font-mono text-xs">{p.hash}</td>
                  <td className="py-2">{p.count.toLocaleString()}</td>
                  <td className="py-2 text-xs text-gray-500">{p.lastSeen.slice(0, 10)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
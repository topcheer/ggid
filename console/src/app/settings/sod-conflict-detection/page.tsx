'use client';
import { useState } from 'react';

interface SodRule {
  id: string;
  ruleName: string;
  roleA: string;
  roleB: string;
  conflictLevel: string;
}

interface Violation {
  id: string;
  user: string;
  rule: string;
  date: string;
  status: string;
}

const roles = ['admin', 'developer', 'auditor', 'finance', 'security', 'operations', 'support'];

export default function SodConflictDetectionPage() {
  const [rules, setRules] = useState<SodRule[]>([
    { id: 'r1', ruleName: 'No Admin + Auditor', roleA: 'admin', roleB: 'auditor', conflictLevel: 'critical' },
    { id: 'r2', ruleName: 'No Finance + Admin', roleA: 'finance', roleB: 'admin', conflictLevel: 'high' },
    { id: 'r3', ruleName: 'No Developer + Security', roleA: 'developer', roleB: 'security', conflictLevel: 'medium' },
    { id: 'r4', ruleName: 'No Operations + Auditor', roleA: 'operations', roleB: 'auditor', conflictLevel: 'high' },
  ]);

  const [violations, setViolations] = useState<Violation[]>([
    { id: 'v1', user: 'alice@ggid.io', rule: 'No Admin + Auditor', date: '2026-07-10', status: 'resolved' },
    { id: 'v2', user: 'bob@ggid.io', rule: 'No Finance + Admin', date: '2026-07-08', status: 'open' },
    { id: 'v3', user: 'carol@ggid.io', rule: 'No Developer + Security', date: '2026-07-05', status: 'remediated' },
  ]);

  const [sensitivity, setSensitivity] = useState('moderate');
  const [autoRemediate, setAutoRemediate] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [newRule, setNewRule] = useState({ ruleName: '', roleA: 'admin', roleB: 'auditor', conflictLevel: 'medium' });

  const levelColor = (l: string): string =>
    l === 'critical' ? 'bg-red-100 text-red-700' :
    l === 'high' ? 'bg-amber-100 text-amber-700' :
    'bg-yellow-100 text-yellow-700';

  const statusColor = (s: string): string =>
    s === 'resolved' ? 'bg-green-100 text-green-700' :
    s === 'open' ? 'bg-red-100 text-red-700' :
    'bg-blue-100 text-blue-700';

  const addRule = () => {
    setRules(prev => [...prev, { id: `r${prev.length + 1}`, ...newRule }]);
    setShowForm(false);
    setNewRule({ ruleName: '', roleA: 'admin', roleB: 'auditor', conflictLevel: 'medium' });
  };

  // Build conflict matrix
  const matrix: number[][] = roles.map((_, i) => roles.map((_, j) => {
    if (i === j) return 0;
    const r = rules.find(rule => (rule.roleA === roles[i] && rule.roleB === roles[j]) || (rule.roleA === roles[j] && rule.roleB === roles[i]));
    return r ? (r.conflictLevel === 'critical' ? 3 : r.conflictLevel === 'high' ? 2 : 1) : 0;
  }));

  const cellColor = (v: number): string =>
    v === 3 ? 'bg-red-500 text-white' :
    v === 2 ? 'bg-amber-400 text-white' :
    v === 1 ? 'bg-yellow-300 text-gray-700' :
    'bg-gray-50 text-gray-300';

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Separation of Duties Conflict Detection</h1>
        <p className="text-gray-600">Detect and prevent conflicting role assignments across the organization.</p>
      </div>

      <div className="grid grid-cols-3 gap-4">
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold">{rules.length}</div>
          <div className="text-sm text-gray-500">SOD Rules</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold text-red-600">{violations.filter(v => v.status === 'open').length}</div>
          <div className="text-sm text-gray-500">Open Violations</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold text-green-600">{violations.filter(v => v.status === 'resolved' || v.status === 'remediated').length}</div>
          <div className="text-sm text-gray-500">Resolved</div>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Sensitivity Level</h2>
          <select value={sensitivity} onChange={e => setSensitivity(e.target.value)} className="w-full border rounded px-3 py-2 text-sm">
            <option value="strict">Strict - All conflict levels trigger violation</option>
            <option value="moderate">Moderate - High and critical trigger violation</option>
            <option value="relaxed">Relaxed - Only critical triggers violation</option>
          </select>
        </section>

        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Auto-Remediation</h2>
          <label className="flex items-center justify-between">
            <span className="text-sm">Automatically remove conflicting roles</span>
            <input type="checkbox" checked={autoRemediate} onChange={e => setAutoRemediate(e.target.checked)} className="rounded" />
          </label>
          <p className="text-xs text-gray-400">When enabled, the system automatically removes the less recently assigned role upon conflict detection.</p>
        </section>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">SOD Rules</h2>
          <button onClick={() => setShowForm(!showForm)} className="px-3 py-1 bg-blue-600 text-white rounded text-sm">
            {showForm ? 'Cancel' : 'Add Rule'}
          </button>
        </div>

        {showForm && (
          <div className="grid grid-cols-4 gap-3 border rounded p-3">
            <input type="text" placeholder="Rule name" value={newRule.ruleName} onChange={e => setNewRule(prev => ({ ...prev, ruleName: e.target.value }))} className="border rounded px-2 py-1.5 text-sm" />
            <select value={newRule.roleA} onChange={e => setNewRule(prev => ({ ...prev, roleA: e.target.value }))} className="border rounded px-2 py-1.5 text-sm">
              {roles.map(r => <option key={r} value={r}>{r}</option>)}
            </select>
            <select value={newRule.roleB} onChange={e => setNewRule(prev => ({ ...prev, roleB: e.target.value }))} className="border rounded px-2 py-1.5 text-sm">
              {roles.map(r => <option key={r} value={r}>{r}</option>)}
            </select>
            <select value={newRule.conflictLevel} onChange={e => setNewRule(prev => ({ ...prev, conflictLevel: e.target.value }))} className="border rounded px-2 py-1.5 text-sm">
              <option value="critical">Critical</option>
              <option value="high">High</option>
              <option value="medium">Medium</option>
            </select>
            <button onClick={addRule} disabled={!newRule.ruleName} className="col-span-4 px-3 py-1.5 bg-blue-600 text-white rounded text-sm disabled:opacity-50">Add Rule</button>
          </div>
        )}

        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th className="p-3">Rule</th>
              <th className="p-3">Role A</th>
              <th className="p-3">Role B</th>
              <th className="p-3">Conflict Level</th>
            </tr>
          </thead>
          <tbody>
            {rules.map(r => (
              <tr key={r.id} className="border-b hover:bg-gray-50">
                <td className="p-3 font-medium">{r.ruleName}</td>
                <td className="p-3"><span className="px-2 py-0.5 bg-gray-100 rounded text-xs">{r.roleA}</span></td>
                <td className="p-3"><span className="px-2 py-0.5 bg-gray-100 rounded text-xs">{r.roleB}</span></td>
                <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs capitalize ${levelColor(r.conflictLevel)}`}>{r.conflictLevel}</span></td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Conflict Matrix Heatmap</h2>
        <table className="text-xs">
          <thead>
            <tr>
              <th className="p-2"></th>
              {roles.map(r => <th key={r} className="p-2 text-gray-500 capitalize">{r}</th>)}
            </tr>
          </thead>
          <tbody>
            {roles.map((role, i) => (
              <tr key={role}>
                <td className="p-2 font-medium capitalize text-gray-500">{role}</td>
                {matrix[i].map((v, j) => (
                  <td key={j} className={`p-2 text-center ${cellColor(v)}`}>
                    {v > 0 ? v : '-'}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
        <div className="flex items-center gap-4 text-xs text-gray-500">
          <span className="flex items-center gap-1"><span className="w-3 h-3 bg-red-500 rounded" /> Critical</span>
          <span className="flex items-center gap-1"><span className="w-3 h-3 bg-amber-400 rounded" /> High</span>
          <span className="flex items-center gap-1"><span className="w-3 h-3 bg-yellow-300 rounded" /> Medium</span>
          <span className="flex items-center gap-1"><span className="w-3 h-3 bg-gray-50 border rounded" /> None</span>
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Violation History</h2>
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th className="p-3">User</th>
              <th className="p-3">Rule</th>
              <th className="p-3">Date</th>
              <th className="p-3">Status</th>
            </tr>
          </thead>
          <tbody>
            {violations.map(v => (
              <tr key={v.id} className="border-b hover:bg-gray-50">
                <td className="p-3 font-medium">{v.user}</td>
                <td className="p-3 text-gray-600">{v.rule}</td>
                <td className="p-3 text-gray-500">{v.date}</td>
                <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs capitalize ${statusColor(v.status)}`}>{v.status}</span></td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </div>
  );
}
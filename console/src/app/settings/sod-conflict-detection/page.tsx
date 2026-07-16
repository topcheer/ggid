'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";

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
  const t = useTranslations();

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [rules, setRules] = useState<SodRule[]>([]);
  const [violations, setViolations] = useState<Violation[]>([]);

  const [sensitivity, setSensitivity] = useState('moderate');
  const [autoRemediate, setAutoRemediate] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [newRule, setNewRule] = useState({ ruleName: '', roleA: 'admin', roleB: 'auditor', conflictLevel: 'medium' });

  useEffect(() => {
    fetch("/api/v1/policies/sod/rules", {
      headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
    })
      .then(res => { if (!res.ok) return null; return res.json(); })
      .then(data => {
        setRules(data.rules || data.items || []);
        setViolations(data.violations || []);
        setLoading(false);
      })
      .catch(err => { setError(err.message); setLoading(false); });
  }, []);

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

  if (loading) return (
    <div className="p-6"><h1 className="text-2xl font-bold mb-4"> {t("backend3.sodConflictDetection.title")}</h1><p>Loading...</p></div>
  );
  if (error) return (
    <div className="p-6"><h1 className="text-2xl font-bold mb-4"> {t("backend3.sodConflictDetection.title")}</h1><p className="text-red-600">Error: {error}</p></div>
  );
  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{t("backend3.sodConflictDetection.title")}</h1>
        <p className="text-gray-600">Detect and prevent conflicting role assignments across the organization.</p>
      </div>

      <div className="grid grid-cols-3 gap-4">
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold">{rules.length}</div>
          <div className="text-sm text-gray-500">SOD Rules</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold text-red-600">{violations.filter(v => v.status === 'open').length}</div>
          <div className="text-sm text-gray-500">{t("backend3.sodConflictDetection.openViolations")}</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold text-green-600">{violations.filter(v => v.status === 'resolved' || v.status === 'remediated').length}</div>
          <div className="text-sm text-gray-500">{t("backend3.sodConflictDetection.resolved")}</div>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">{t("backend3.sodConflictDetection.sensitivityLevel")}</h2>
          <select value={sensitivity} onChange={e => setSensitivity(e.target.value)} className="w-full border rounded px-3 py-2 text-sm">
            <option value="strict">Strict - All conflict levels trigger violation</option>
            <option value="moderate">Moderate - High and critical trigger violation</option>
            <option value="relaxed">Relaxed - Only critical triggers violation</option>
          </select>
        </section>

        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Auto-Remediation</h2>
          <label className="flex items-center justify-between">
            <span className="text-sm">{t("backend3.sodConflictDetection.autoRemove")}</span>
            <input type="checkbox" checked={autoRemediate} onChange={e => setAutoRemediate(e.target.checked)} className="rounded" />
          </label>
          <p className="text-xs text-gray-400">When enabled, the system automatically removes the less recently assigned role upon conflict detection.</p>
        </section>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">SOD Rules</h2>
          <button onClick={() => setShowForm(!showForm)} className="px-3 py-1 bg-blue-600 text-white rounded text-sm">
            {showForm ? 'Cancel' : t("backend3.sodConflictDetection.addRule")}
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
              <option value="critical">{t("backend3.sodConflictDetection.critical")}</option>
              <option value="high">{t("backend3.sodConflictDetection.high")}</option>
              <option value="medium">{t("backend3.sodConflictDetection.medium")}</option>
            </select>
            <button onClick={addRule} disabled={!newRule.ruleName} className="col-span-4 px-3 py-1.5 bg-blue-600 text-white rounded text-sm disabled:opacity-50">{t("backend3.sodConflictDetection.addRule")}</button>
          </div>
        )}

        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th scope="col" className="p-3">{t("backend3.sodConflictDetection.rule")}</th>
              <th scope="col" className="p-3">{t("backend3.sodConflictDetection.roleA")}</th>
              <th scope="col" className="p-3">{t("backend3.sodConflictDetection.roleB")}</th>
              <th scope="col" className="p-3">{t("backend3.sodConflictDetection.conflictLevel")}</th>
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
        <h2 className="text-lg font-semibold">{t("backend3.sodConflictDetection.conflictMatrixHeatmap")}</h2>
        <table className="text-xs">
          <thead>
            <tr>
              <th scope="col" className="p-2"></th>
              {roles.map(r => <th scope="col" key={r} className="p-2 text-gray-500 capitalize">{r}</th>)}
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
              <th scope="col" className="p-3">User</th>
              <th scope="col" className="p-3">{t("backend3.sodConflictDetection.rule")}</th>
              <th scope="col" className="p-3">{t("backend3.sodConflictDetection.date")}</th>
              <th scope="col" className="p-3">Status</th>
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
'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";

interface SodRule {
  id: string;
  roleA: string;
  roleB: string;
  conflictLevel: string;
  enabled: boolean;
}

interface Violation {
  id: string;
  user: string;
  rule: string;
  date: string;
  status: string;
}

export default function SodRulesConfigPage() {
  const t = useTranslations();

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [rules, setRules] = useState<SodRule[]>([]);
  const [violations, setViolations] = useState<Violation[]>([]);

  const [showForm, setShowForm] = useState(false);
  const [sensitivity, setSensitivity] = useState('moderate');
  const [autoRemediate, setAutoRemediate] = useState(false);
  const [newRule, setNewRule] = useState({ roleA: '', roleB: '', conflictLevel: 'medium' });

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

  const roles = ['admin', 'auditor', 'developer', 'finance', 'security', 'operations'];
  const levels = ['high', 'medium', 'low'];

  const addRule = () => {
    setRules(prev => [...prev, { id: `s${prev.length + 1}`, roleA: newRule.roleA, roleB: newRule.roleB, conflictLevel: newRule.conflictLevel, enabled: true }]);
    setShowForm(false);
    setNewRule({ roleA: '', roleB: '', conflictLevel: 'medium' });
  };

  const toggleRule = (id: string) => {
    setRules(prev => prev.map(r => r.id === id ? { ...r, enabled: !r.enabled } : r));
  };

  const levelColor = (l: string): string =>
    l === 'high' ? 'bg-red-100 text-red-700' : l === 'medium' ? 'bg-amber-100 text-amber-700' : 'bg-green-100 text-green-700';

  const matrix: string[][] = roles.map(r1 => roles.map(r2 => {
    const rule = rules.find(r => (r.roleA === r1 && r.roleB === r2) || (r.roleA === r2 && r.roleB === r1));
    return rule ? rule.conflictLevel : '-';
  }));

  const cellColor = (v: string): string =>
    v === 'high' ? 'bg-red-200' : v === 'medium' ? 'bg-amber-200' : v === 'low' ? 'bg-green-200' : '';

  if (loading) return (
    <div className="p-6"><h1 className="text-2xl font-bold mb-4"> {t("backend3.sodRulesConfig.title")}</h1><p>Loading...</p></div>
  );
  if (error) return (
    <div className="p-6"><h1 className="text-2xl font-bold mb-4"> {t("backend3.sodRulesConfig.title")}</h1><p className="text-red-600">Error: {error}</p></div>
  );
  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold"> {t("backend3.sodRulesConfig.title")}</h1>
        <p className="text-gray-600">Separation of Duties conflict detection rules and remediation.</p>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <section className="bg-white rounded-lg shadow p-4">
          <label className="text-sm font-medium">{t("backend3.sodRulesConfig.sensitivityLevel")}</label>
          <select aria-label="sensitivity" value={sensitivity} onChange={e => setSensitivity(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1">
            <option value="strict">Strict — all conflict levels enforced</option>
            <option value="moderate">Moderate — high + medium enforced</option>
            <option value="relaxed">Relaxed — only high enforced</option>
          </select>
        </section>

        <label className="flex items-center justify-between bg-white rounded-lg shadow p-4">
          <span className="text-sm font-medium">Auto-Remediation</span>
          <input aria-label="Auto remediate" type="checkbox" checked={autoRemediate} onChange={e => setAutoRemediate(e.target.checked)} className="rounded" />
        </label>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">SOD Rules</h2>
          <button onClick={() => setShowForm(!showForm)} className="px-3 py-1 bg-blue-600 text-white rounded text-sm">{showForm ? 'Cancel' : 'Add Rule'}</button>
        </div>

        {showForm && (
          <div className="flex gap-3 border rounded p-3">
            <select aria-label="Select option" value={newRule.roleA} onChange={e => setNewRule(prev => ({ ...prev, roleA: e.target.value }))} className="border rounded px-2 py-1 text-sm">
              <option value="">Role A...</option>
              {roles.map(r => <option key={r} value={r}>{r}</option>)}
            </select>
            <span className="text-gray-400">×</span>
            <select aria-label="Select option" value={newRule.roleB} onChange={e => setNewRule(prev => ({ ...prev, roleB: e.target.value }))} className="border rounded px-2 py-1 text-sm">
              <option value="">Role B...</option>
              {roles.map(r => <option key={r} value={r}>{r}</option>)}
            </select>
            <select aria-label="Select option" value={newRule.conflictLevel} onChange={e => setNewRule(prev => ({ ...prev, conflictLevel: e.target.value }))} className="border rounded px-2 py-1 text-sm">
              {levels.map(l => <option key={l} value={l}>{l}</option>)}
            </select>
            <button aria-label="action" onClick={addRule} disabled={!newRule.roleA || !newRule.roleB} className="px-3 py-1 bg-blue-600 text-white rounded text-sm disabled:opacity-50">{t("backend3.sodRulesConfig.add")}</button>
          </div>
        )}

        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th scope="col" className="p-3">{t("backend3.sodRulesConfig.roleA")}</th>
              <th scope="col" className="p-3">{t("backend3.sodRulesConfig.roleB")}</th>
              <th scope="col" className="p-3">{t("backend3.sodRulesConfig.conflictLevel")}</th>
              <th scope="col" className="p-3">{t("backend3.sodRulesConfig.enabled")}</th>
            </tr>
          </thead>
          <tbody>
            {rules.map(r => (
              <tr key={r.id} className="border-b">
                <td className="p-3 font-medium">{r.roleA}</td>
                <td className="p-3 font-medium">{r.roleB}</td>
                <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs capitalize ${levelColor(r.conflictLevel)}`}>{r.conflictLevel}</span></td>
                <td className="p-3"><input aria-label="Toggle" type="checkbox" checked={r.enabled} onChange={() => toggleRule(r.id)} className="rounded" /></td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{t("backend3.sodRulesConfig.conflictMatrixHeatmap")}</h2>
        <table className="text-xs">
          <thead>
            <tr>
              <th scope="col" className="p-2"></th>
              {roles.map(r => <th scope="col" key={r} className="p-2 text-left">{r}</th>)}
            </tr>
          </thead>
          <tbody>
            {roles.map((r1, i) => (
              <tr key={r1}>
                <td className="p-2 font-medium">{r1}</td>
                {matrix[i].map((v, j) => (
                  <td key={j} className={`p-2 text-center ${cellColor(v)}`}>{v !== '-' ? v : ''}</td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{t("backend3.sodRulesConfig.violationHistory")}</h2>
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th scope="col" className="p-3">{t("backend3.sodRulesConfig.user")}</th>
              <th scope="col" className="p-3">{t("backend3.sodRulesConfig.rule")}</th>
              <th scope="col" className="p-3">{t("backend3.sodRulesConfig.date")}</th>
              <th scope="col" className="p-3">{t("backend3.sodRulesConfig.status")}</th>
            </tr>
          </thead>
          <tbody>
            {violations.map(v => (
              <tr key={v.id} className="border-b">
                <td className="p-3 font-medium">{v.user}</td>
                <td className="p-3 text-gray-600">{v.rule}</td>
                <td className="p-3 text-gray-500">{v.date}</td>
                <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs ${v.status === 'open' ? 'bg-red-100 text-red-700' : 'bg-green-100 text-green-700'}`}>{v.status}</span></td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </div>
  );
}

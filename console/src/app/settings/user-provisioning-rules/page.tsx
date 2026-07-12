'use client';
import { useState } from 'react';

interface FieldMapping {
  sourceField: string;
  targetField: string;
  defaultValue: string;
}

interface ProvisioningRule {
  id: string;
  source: string;
  trigger: string;
  enabled: boolean;
  fieldMappings: FieldMapping[];
}

interface ExecutionLog {
  id: string;
  rule: string;
  source: string;
  status: string;
  timestamp: string;
  details: string;
}

export default function UserProvisioningRulesPage() {
  const [rules, setRules] = useState<ProvisioningRule[]>([
    { id: 'r1', source: 'HR', trigger: 'create', enabled: true, fieldMappings: [
      { sourceField: 'employee_id', targetField: 'username', defaultValue: '' },
      { sourceField: 'email', targetField: 'email', defaultValue: '' },
      { sourceField: 'department', targetField: 'department', defaultValue: 'general' },
    ]},
    { id: 'r2', source: 'SCIM', trigger: 'create', enabled: true, fieldMappings: [
      { sourceField: 'userName', targetField: 'username', defaultValue: '' },
      { sourceField: 'displayName', targetField: 'full_name', defaultValue: '' },
    ]},
    { id: 'r3', source: 'HR', trigger: 'update', enabled: true, fieldMappings: [
      { sourceField: 'department', targetField: 'department', defaultValue: '' },
      { sourceField: 'title', targetField: 'job_title', defaultValue: '' },
    ]},
    { id: 'r4', source: 'IaC', trigger: 'delete', enabled: false, fieldMappings: [
      { sourceField: 'resource_id', targetField: 'user_id', defaultValue: '' },
    ]},
  ]);

  const [logs] = useState<ExecutionLog[]>([
    { id: 'l1', rule: 'HR-create', source: 'HR', status: 'success', timestamp: '2026-07-12 09:30', details: 'Provisioned alice@ggid.io' },
    { id: 'l2', rule: 'SCIM-create', source: 'SCIM', status: 'success', timestamp: '2026-07-12 08:15', details: 'Provisioned bob@ggid.io' },
    { id: 'l3', rule: 'HR-update', source: 'HR', status: 'failed', timestamp: '2026-07-11 17:45', details: 'Field mapping error: missing department' },
  ]);

  const [showForm, setShowForm] = useState(false);
  const [autoProvision, setAutoProvision] = useState(true);
  const [newRule, setNewRule] = useState({ source: 'HR', trigger: 'create' });
  const [testData, setTestData] = useState('{}');
  const [testResult, setTestResult] = useState('');

  const sources = ['HR', 'SCIM', 'IaC'];
  const triggers = ['create', 'update', 'delete'];

  const toggleRule = (id: string) => {
    setRules(prev => prev.map(r => r.id === id ? { ...r, enabled: !r.enabled } : r));
  };

  const addRule = () => {
    setRules(prev => [...prev, {
      id: `r${prev.length + 1}`,
      source: newRule.source,
      trigger: newRule.trigger,
      enabled: true,
      fieldMappings: [{ sourceField: '', targetField: '', defaultValue: '' }],
    }]);
    setShowForm(false);
  };

  const runTest = () => {
    try {
      const data = JSON.parse(testData);
      setTestResult(`Rule would provision user with fields: ${Object.keys(data).join(', ')}`);
    } catch {
      setTestResult('Invalid JSON');
    }
  };

  const syncStatus: Record<string, string> = { HR: 'synced', SCIM: 'synced', IaC: 'error' };

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">User Provisioning Rules</h1>
          <p className="text-gray-600">Automated user provisioning from HR, SCIM, and Infrastructure-as-Code sources.</p>
        </div>
        <div className="flex gap-2">
          <label className="flex items-center gap-2 px-3 py-1.5 border rounded text-sm">
            <input type="checkbox" checked={autoProvision} onChange={e => setAutoProvision(e.target.checked)} className="rounded" />
            Auto-Provision
          </label>
          <button onClick={() => setShowForm(!showForm)} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">
            {showForm ? 'Cancel' : 'Add Rule'}
          </button>
        </div>
      </div>

      <div className="grid grid-cols-3 gap-4">
        {sources.map(s => (
          <div key={s} className="bg-white rounded-lg shadow p-4">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium">{s} Sync</span>
              <span className={`px-2 py-0.5 rounded text-xs ${syncStatus[s] === 'synced' ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'}`}>{syncStatus[s]}</span>
            </div>
            <div className="text-xs text-gray-500 mt-1">
              {s === 'HR' && 'Last sync: 5 min ago'}
              {s === 'SCIM' && 'Last sync: 2 min ago'}
              {s === 'IaC' && 'Error: connection refused'}
            </div>
          </div>
        ))}
      </div>

      {showForm && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Create Provisioning Rule</h2>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="text-sm font-medium">Source</label>
              <select value={newRule.source} onChange={e => setNewRule(prev => ({ ...prev, source: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1">
                {sources.map(s => <option key={s} value={s}>{s}</option>)}
              </select>
            </div>
            <div>
              <label className="text-sm font-medium">Trigger</label>
              <select value={newRule.trigger} onChange={e => setNewRule(prev => ({ ...prev, trigger: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1">
                {triggers.map(t => <option key={t} value={t}>{t}</option>)}
              </select>
            </div>
          </div>
          <button onClick={addRule} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">Create Rule</button>
        </section>
      )}

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Provisioning Rules</h2>
        <div className="space-y-3">
          {rules.map(r => (
            <div key={r.id} className="border rounded p-4">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <label className="flex items-center gap-2">
                    <input type="checkbox" checked={r.enabled} onChange={() => toggleRule(r.id)} className="rounded" />
                    <span className="font-mono text-sm font-medium">{r.source}-{r.trigger}</span>
                  </label>
                  <span className={`px-2 py-0.5 rounded text-xs ${r.enabled ? 'bg-green-100 text-green-700' : 'bg-gray-200 text-gray-600'}`}>{r.enabled ? 'enabled' : 'disabled'}</span>
                </div>
              </div>
              <table className="w-full text-xs mt-3">
                <thead className="text-left text-gray-500">
                  <tr>
                    <th className="py-1">Source Field</th>
                    <th className="py-1">Target Field</th>
                    <th className="py-1">Default</th>
                  </tr>
                </thead>
                <tbody>
                  {r.fieldMappings.map((m, idx) => (
                    <tr key={idx} className="border-t">
                      <td className="py-1 font-mono">{m.sourceField}</td>
                      <td className="py-1 font-mono">{m.targetField}</td>
                      <td className="py-1 text-gray-500">{m.defaultValue || '-'}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ))}
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Test Rule with Sample Data</h2>
        <textarea value={testData} onChange={e => setTestData(e.target.value)} rows={3} placeholder='{"employee_id": "emp123", "email": "new@ggid.io"}' className="w-full border rounded px-3 py-2 text-sm font-mono" />
        <button onClick={runTest} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">Test</button>
        {testResult && <div className="text-sm p-3 bg-blue-50 rounded">{testResult}</div>}
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Rule Execution Log</h2>
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th className="p-3">Rule</th>
              <th className="p-3">Source</th>
              <th className="p-3">Status</th>
              <th className="p-3">Timestamp</th>
              <th className="p-3">Details</th>
            </tr>
          </thead>
          <tbody>
            {logs.map(l => (
              <tr key={l.id} className="border-b">
                <td className="p-3 font-mono text-xs">{l.rule}</td>
                <td className="p-3">{l.source}</td>
                <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs ${l.status === 'success' ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'}`}>{l.status}</span></td>
                <td className="p-3 text-gray-500 text-xs">{l.timestamp}</td>
                <td className="p-3 text-gray-600 text-xs">{l.details}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </div>
  );
}
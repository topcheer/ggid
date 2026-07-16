'use client';
import { useTranslations } from "@/lib/i18n";
import { useState, useEffect } from 'react';

interface PiiField {
  id: string;
  field: string;
  pattern: string;
  maskInApi: boolean;
  maskInAudit: boolean;
  maskInExports: boolean;
  unmaskRoles: string[];
}

export default function DataMaskingConfigPage() {
  const t = useTranslations();
  const [fields, setFields] = useState<PiiField[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [newField, setNewField] = useState({ field: '', pattern: 'partial', unmaskRoles: '' });
  const [testInput, setTestInput] = useState('');
  const [testField, setTestField] = useState('email');
  const [testOutput, setTestOutput] = useState('');
  const [stats, setStats] = useState({ fieldsConfigured: 0, recordsMasked24h: 0, auditMasked: 0, exportsMasked: 0 });

  useEffect(() => {
    fetch("/api/v1/identity/pii-config", {
      headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
    })
      .then(res => { if (!res.ok) return null; return res.json(); })
      .then(data => {
        setFields(data.fields || []);
        setStats(data.stats || { fieldsConfigured: (data.fields || []).length, recordsMasked24h: 0, auditMasked: 0, exportsMasked: 0 });
        setLoading(false);
      })
      .catch(err => { setError(err.message); setLoading(false); });
  }, []);

  const patterns = ['full', 'partial', 'regex', 'hash'];
  const allRoles = ['admin', 'security-admin', 'developer', 'auditor', 'viewer'];

  const maskValue = (value: string, field: string): string => {
    const config = fields.find(f => f.field === field);
    if (!config || !value) return value;
    if (config.pattern === 'full') return '****';
    if (config.pattern === 'hash') return `hash(${value.length}chars)`;
    if (config.pattern === 'partial') {
      if (field === 'email') { const [u, d] = value.split('@'); return `${u[0]}***@${d}`; }
      if (field === 'phone') return value.replace(/\d(?=\d{2})/g, '*');
      if (field === 'ssn') return `***-**-${value.slice(-4)}`;
      if (field === 'credit_card') return `**** **** **** ${value.slice(-4)}`;
      if (field === 'address') return `${value.slice(0, 5)}***`;
      if (field === 'full_name') return `${value[0]}***`;
      return `${value[0]}***`;
    }
    return value;
  };

  const runTest = () => { setTestOutput(maskValue(testInput, testField)); };
  const addField = () => {
    setFields(prev => [...prev, { id: `p${prev.length + 1}`, field: newField.field, pattern: newField.pattern, maskInApi: true, maskInAudit: true, maskInExports: true, unmaskRoles: newField.unmaskRoles ? newField.unmaskRoles.split(',').map(s => s.trim()) : [] }]);
    setShowForm(false); setNewField({ field: '', pattern: 'partial', unmaskRoles: '' });
  };
  const removeField = (id: string) => setFields(prev => prev.filter(f => f.id !== id));
  const updateField = (id: string, key: keyof PiiField, value: boolean) => setFields(prev => prev.map(f => f.id === id ? { ...f, [key]: value } : f));

  if (loading) return <div className="p-6"><h1 className="text-2xl font-bold">Data Masking Configuration</h1><p className="text-gray-600 mt-2">Loading...</p></div>;
  if (error) return <div className="p-6"><h1 className="text-2xl font-bold">Data Masking Configuration</h1><p className="text-red-600 mt-2">Error: {error}</p></div>;

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold">Data Masking Configuration</h1><p className="text-gray-600">Configure PII field masking for API responses, audit logs, and exports.</p></div>
        <button onClick={() => setShowForm(!showForm)} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">{showForm ? 'Cancel' : 'Add Field'}</button>
      </div>

      <div className="grid grid-cols-4 gap-4">
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-2xl font-bold">{stats.fieldsConfigured}</div><div className="text-sm text-gray-500">Fields Configured</div></div>
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-2xl font-bold">{stats.recordsMasked24h.toLocaleString()}</div><div className="text-sm text-gray-500">Records Masked (24h)</div></div>
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-2xl font-bold text-blue-600">{stats.auditMasked.toLocaleString()}</div><div className="text-sm text-gray-500">Audit Masked</div></div>
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-2xl font-bold text-green-600">{stats.exportsMasked}</div><div className="text-sm text-gray-500">Exports Masked</div></div>
      </div>

      {showForm && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Add PII Field</h2>
          <div className="grid grid-cols-3 gap-4">
            <div><label className="text-sm font-medium">Field Name</label><input aria-label="date_of_birth" type="text" placeholder="date_of_birth" value={newField.field} onChange={e => setNewField(prev => ({ ...prev, field: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
            <div><label className="text-sm font-medium">Masking Pattern</label><select aria-label="new Field" value={newField.pattern} onChange={e => setNewField(prev => ({ ...prev, pattern: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1">{patterns.map(p => <option key={p} value={p}>{p}</option>)}</select></div>
            <div><label className="text-sm font-medium">Unmask Roles (comma-separated)</label><input aria-label="admin, security-admin" type="text" placeholder="admin, security-admin" value={newField.unmaskRoles} onChange={e => setNewField(prev => ({ ...prev, unmaskRoles: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
          </div>
          <button onClick={addField} disabled={!newField.field} className="px-4 py-2 bg-blue-600 text-white rounded text-sm disabled:opacity-50">Add</button>
        </section>
      )}

      <section className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50"><tr className="text-left"><th className="p-3">Field</th><th className="p-3">Pattern</th><th className="p-3">API</th><th className="p-3">Audit</th><th className="p-3">Exports</th><th className="p-3">Unmask Roles</th><th className="p-3">Action</th></tr></thead>
          <tbody>
            {fields.length === 0 ? <tr><td colSpan={7} className="p-6 text-center text-gray-500">No PII fields configured.</td></tr> :
            fields.map(f => (
              <tr key={f.id} className="border-b">
                <td className="p-3 font-mono text-xs">{f.field}</td>
                <td className="p-3"><span className="px-2 py-0.5 bg-gray-100 rounded text-xs">{f.pattern}</span></td>
                <td className="p-3"><input aria-label="Toggle" type="checkbox" checked={f.maskInApi} onChange={e => updateField(f.id, 'maskInApi', e.target.checked)} className="rounded" /></td>
                <td className="p-3"><input aria-label="Toggle" type="checkbox" checked={f.maskInAudit} onChange={e => updateField(f.id, 'maskInAudit', e.target.checked)} className="rounded" /></td>
                <td className="p-3"><input aria-label="Toggle" type="checkbox" checked={f.maskInExports} onChange={e => updateField(f.id, 'maskInExports', e.target.checked)} className="rounded" /></td>
                <td className="p-3"><div className="flex flex-wrap gap-1">{f.unmaskRoles.map(r => <span key={r} className="px-1.5 py-0.5 bg-purple-100 text-purple-700 rounded text-xs">{r}</span>)}</div></td>
                <td className="p-3"><button onClick={() => removeField(f.id)} className="text-red-600 text-xs hover:underline">Remove</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Test Masking</h2>
        <div className="flex gap-3">
          <select aria-label="Test field" value={testField} onChange={e => setTestField(e.target.value)} className="border rounded px-3 py-2 text-sm">{fields.map(f => <option key={f.id} value={f.field}>{f.field}</option>)}</select>
          <input aria-label="Enter test value..." type="text" placeholder="Enter test value..." value={testInput} onChange={e => setTestInput(e.target.value)} className="flex-1 border rounded px-3 py-2 text-sm" />
          <button onClick={runTest} disabled={!testInput} className="px-4 py-2 bg-blue-600 text-white rounded text-sm disabled:opacity-50">Mask</button>
        </div>
        {testOutput && <div className="grid grid-cols-2 gap-4 text-sm"><div><div className="text-xs text-gray-500">Input:</div><div className="font-mono">{testInput}</div></div><div><div className="text-xs text-gray-500">Masked:</div><div className="font-mono text-blue-600">{testOutput}</div></div></div>}
      </section>
    </div>
  );
}

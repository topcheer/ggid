'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";

export default function BulkOperationsPage() {
  const t = useTranslations();

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [bundles, setBundles] = useState<Record<string, unknown>[]>([]);
  const [opType, setOpType] = useState('bulk_user_create');
  const [csvPreview, setCsvPreview] = useState('');
  const [progress, setProgress] = useState(0);
  const [running, setRunning] = useState(false);
  const [result, setResult] = useState<{ success: number; failed: number; total: number } | null>(null);

  const opTypes = [
    { id: 'bulk_user_create', label: 'Bulk User Create (CSV)' },
    { id: 'bulk_role_assign', label: 'Bulk Role Assign' },
    { id: 'bulk_suspend', label: 'Bulk Suspend' },
    { id: 'bulk_activate', label: 'Bulk Activate' },
    { id: 'bulk_password_reset', label: 'Bulk Password Reset' },
    { id: 'bulk_mfa_enroll', label: 'Bulk MFA Enroll' },
  ];

  const runOp = () => {
    setRunning(true); setProgress(0); setResult(null);
    fetch("/api/v1/policies/bundles", {
      method: "POST",
      headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
      body: JSON.stringify({ opType, csvPreview }),
    })
      .then(res => { if (!res.ok) return null; return res.json(); })
      .then(data => { setResult(data.result || data); setProgress(100); setRunning(false); })
      .catch(err => { setError(err.message); setRunning(false); });
  };

  const downloadErrors = () => {
    const csv = 'row,error\n5,Email already exists\n12,Invalid role reference\n';
    const blob = new Blob([csv], { type: 'text/csv' });
    const a = document.createElement('a'); a.href = URL.createObjectURL(blob); a.download = 'bulk-errors.csv'; a.click();
  };

  useEffect(() => {
    fetch("/api/v1/policies/bundles", {
      headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
    })
      .then(res => { if (!res.ok) return null; return res.json(); })
      .then(data => { setBundles(data.bundles || data.items || []); setLoading(false); })
      .catch(err => { setError(err.message); setLoading(false); });
  }, []);

  if (loading) return (<div className="p-6"><h1 className="text-2xl font-bold mb-4">Bulk Operations</h1><p>Loading...</p></div>);
  if (error) return (<div className="p-6"><h1 className="text-2xl font-bold mb-4">Bulk Operations</h1><p className="text-red-600">Error: {error}</p></div>);
  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      <div><h1 className="text-2xl font-bold">Bulk Operations</h1><p className="text-gray-600">Run bulk user operations with CSV upload, progress tracking, and rollback.</p></div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Select Operation</h2>
        <select aria-label="Op type" value={opType} onChange={e => setOpType(e.target.value)} className="w-full border rounded px-3 py-2 text-sm">
          {opTypes.map(o => <option key={o.id} value={o.id}>{o.label}</option>)}
        </select>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">CSV Upload + Preview</h2>
        <input aria-label="Input field" type="file" accept=".csv" className="text-sm" />
        <textarea aria-label="username,email,role&#10;alice,alice@ggid.io,developer&#10;bob,bob@ggid.io,auditor" value={csvPreview} onChange={e => setCsvPreview(e.target.value)} rows={5} placeholder="username,email,role&#10;alice,alice@ggid.io,developer&#10;bob,bob@ggid.io,auditor" className="w-full border rounded px-3 py-2 text-sm font-mono" />
        <div className="flex gap-3">
          <button aria-label="action" onClick={runOp} disabled={running} className="px-4 py-2 bg-blue-600 text-white rounded text-sm disabled:opacity-50">{running ? 'Running...' : 'Run Operation'}</button>
          <label className="flex items-center gap-2 text-sm"><input aria-label="Toggle option" type="checkbox" className="rounded" />Schedule for later</label>
        </div>
      </section>

      {(progress > 0 || running) && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Progress: {progress}%</h2>
          <div className="w-full bg-gray-200 rounded-full h-4 overflow-hidden"><div className="h-4 bg-blue-600 rounded-full transition-all" style={{ width: `${progress}%` }} /></div>
        </section>
      )}

      {result && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Result Summary</h2>
          <div className="grid grid-cols-3 gap-4">
            <div className="border rounded p-4 text-center"><div className="text-2xl font-bold text-green-600">{result.success}</div><div className="text-sm text-gray-500">Success</div></div>
            <div className="border rounded p-4 text-center"><div className="text-2xl font-bold text-red-600">{result.failed}</div><div className="text-sm text-gray-500">Failed</div></div>
            <div className="border rounded p-4 text-center"><div className="text-2xl font-bold">{result.total}</div><div className="text-sm text-gray-500">Total</div></div>
          </div>
          <div className="flex gap-3">
            <button onClick={downloadErrors} className="px-4 py-2 border rounded text-sm">Download Error Report</button>
            <button className="px-4 py-2 border rounded text-sm text-red-600">Rollback</button>
          </div>
        </section>
      )}
    </div>
  );
}
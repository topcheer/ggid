'use client';
import { useState } from 'react';

interface ExportJob { id: string; type: string; status: string; size: string; created: string; download: boolean; }

export default function DataExportCenterPage() {
  const [jobs, setJobs] = useState<ExportJob[]>([
    { id: 'j1', type: 'users', status: 'completed', size: '2.3 MB', created: '2026-07-12 10:00', download: true },
    { id: 'j2', type: 'audit_events', status: 'completed', size: '15.7 MB', created: '2026-07-11 14:00', download: true },
    { id: 'j3', type: 'roles', status: 'processing', size: '-', created: '2026-07-12 14:30', download: false },
  ]);
  const [showForm, setShowForm] = useState(false);
  const [newJob, setNewJob] = useState({ type: 'users', format: 'csv', startDate: '', endDate: '', maskPii: true });
  const [scheduled, setScheduled] = useState(false);

  const exportTypes = ['users', 'roles', 'audit_events', 'config', 'organizations'];
  const formats = ['CSV', 'JSON', 'Parquet'];

  const createJob = () => {
    setJobs(prev => [...prev, { id: `j${prev.length + 1}`, type: newJob.type, status: 'processing', size: '-', created: new Date().toISOString().slice(0, 16).replace('T', ' '), download: false }]);
    setShowForm(false);
    setTimeout(() => setJobs(prev => prev.map(j => j.id === `j${prev.length}` ? { ...j, status: 'completed', size: '1.2 MB', download: true } : j)), 2000);
  };

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold">Data Export Center</h1><p className="text-gray-600">Create, schedule, and download data exports with PII masking.</p></div>
        <button onClick={() => setShowForm(!showForm)} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">{showForm ? 'Cancel' : 'Create Export'}</button>
      </div>

      {showForm && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Create Export Job</h2>
          <div className="grid grid-cols-2 gap-4">
            <div><label className="text-sm font-medium">Export Type</label><select value={newJob.type} onChange={e => setNewJob(prev => ({ ...prev, type: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1">{exportTypes.map(t => <option key={t} value={t}>{t}</option>)}</select></div>
            <div><label className="text-sm font-medium">Format</label><select value={newJob.format} onChange={e => setNewJob(prev => ({ ...prev, format: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1">{formats.map(f => <option key={f} value={f}>{f}</option>)}</select></div>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div><label className="text-sm font-medium">Start Date</label><input type="date" value={newJob.startDate} onChange={e => setNewJob(prev => ({ ...prev, startDate: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
            <div><label className="text-sm font-medium">End Date</label><input type="date" value={newJob.endDate} onChange={e => setNewJob(prev => ({ ...prev, endDate: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
          </div>
          <div className="flex gap-4">
            <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={newJob.maskPii} onChange={e => setNewJob(prev => ({ ...prev, maskPii: e.target.checked }))} className="rounded" />Mask PII fields</label>
            <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={scheduled} onChange={e => setScheduled(e.target.checked)} className="rounded" />Schedule weekly</label>
          </div>
          <button onClick={createJob} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">Create Export</button>
        </section>
      )}

      <section className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full text-sm"><thead className="bg-gray-50"><tr className="text-left"><th className="p-3">Job ID</th><th className="p-3">Type</th><th className="p-3">Status</th><th className="p-3">Size</th><th className="p-3">Created</th><th className="p-3">Action</th></tr></thead>
          <tbody>{jobs.map(j => (
            <tr key={j.id} className="border-b">
              <td className="p-3 font-mono text-xs">{j.id}</td><td className="p-3 font-medium">{j.type}</td>
              <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs ${j.status === 'completed' ? 'bg-green-100 text-green-700' : 'bg-blue-100 text-blue-700'}`}>{j.status}</span></td>
              <td className="p-3 text-gray-500">{j.size}</td><td className="p-3 text-gray-500">{j.created}</td>
              <td className="p-3">{j.download ? <button className="text-blue-600 text-xs hover:underline">Download</button> : <span className="text-xs text-gray-400">Processing...</span>}</td>
            </tr>))}</tbody></table>
      </section>

      <p className="text-xs text-gray-400">Download links expire after 24 hours. All export actions are recorded in the audit trail.</p>
    </div>
  );
}
'use client';
import { useState } from 'react';

interface RequestLog {
  id: string;
  requestId: string;
  method: string;
  path: string;
  status: number;
  duration: string;
  timestamp: string;
}

export default function RequestIdTrackingPage() {
  const [format, setFormat] = useState('uuid');
  const [headerName, setHeaderName] = useState('X-Request-ID');
  const [propagation, setPropagation] = useState(true);
  const [searchId, setSearchId] = useState('');
  const [filterStatus, setFilterStatus] = useState('all');

  const [logs] = useState<RequestLog[]>([
    { id: 'l1', requestId: 'a1b2c3d4-e5f6-7890-abcd-ef1234567890', method: 'POST', path: '/api/v1/auth/login', status: 200, duration: '45ms', timestamp: '2026-07-12 14:32:01' },
    { id: 'l2', requestId: 'b2c3d4e5-f6a7-8901-bcde-f12345678901', method: 'GET', path: '/api/v1/users', status: 200, duration: '12ms', timestamp: '2026-07-12 14:31:55' },
    { id: 'l3', requestId: 'c3d4e5f6-a7b8-9012-cdef-123456789012', method: 'POST', path: '/api/v1/auth/login', status: 401, duration: '8ms', timestamp: '2026-07-12 14:31:30' },
    { id: 'l4', requestId: 'd4e5f6a7-b8c9-0123-defa-234567890123', method: 'PUT', path: '/api/v1/users/me', status: 200, duration: '23ms', timestamp: '2026-07-12 14:30:15' },
    { id: 'l5', requestId: 'e5f6a7b8-c9d0-1234-efab-345678901234', method: 'DELETE', path: '/api/v1/orgs/org-123', status: 403, duration: '5ms', timestamp: '2026-07-12 14:29:00' },
  ]);

  const formats = ['uuid', 'ulid', 'nanoid'];

  const filtered = logs.filter(l =>
    (searchId === '' || l.requestId.includes(searchId)) &&
    (filterStatus === 'all' ||
      (filterStatus === 'success' && l.status < 400) ||
      (filterStatus === 'error' && l.status >= 400))
  );

  const statusColor = (s: number): string =>
    s < 300 ? 'text-green-600' : s < 400 ? 'text-blue-600' : s < 500 ? 'text-amber-600' : 'text-red-600';

  const exportTrace = () => {
    const data = JSON.stringify(filtered, null, 2);
    const blob = new Blob([data], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url; a.download = 'request-trace.json'; a.click();
  };

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Request ID Tracking</h1>
        <p className="text-gray-600">Configure request ID format, propagation, and trace inspection.</p>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Configuration</h2>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="text-sm font-medium">ID Format</label>
            <select value={format} onChange={e => setFormat(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1">
              {formats.map(f => <option key={f} value={f}>{f.toUpperCase()}</option>)}
            </select>
          </div>
          <div>
            <label className="text-sm font-medium">Header Name</label>
            <input type="text" value={headerName} onChange={e => setHeaderName(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1 font-mono" />
          </div>
        </div>
        <label className="flex items-center justify-between">
          <span className="text-sm">Propagate to downstream services (inject {headerName} into all outbound calls)</span>
          <input type="checkbox" checked={propagation} onChange={e => setPropagation(e.target.checked)} className="rounded" />
        </label>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">Request Log Viewer</h2>
          <button onClick={exportTrace} className="px-3 py-1 border rounded text-sm">Export Trace</button>
        </div>
        <div className="flex gap-3">
          <input type="text" placeholder="Search by Request ID..." value={searchId} onChange={e => setSearchId(e.target.value)} className="flex-1 border rounded px-3 py-2 text-sm font-mono" />
          <select value={filterStatus} onChange={e => setFilterStatus(e.target.value)} className="border rounded px-3 py-2 text-sm">
            <option value="all">All</option>
            <option value="success">Success (2xx/3xx)</option>
            <option value="error">Errors (4xx/5xx)</option>
          </select>
        </div>
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th className="p-3">Request ID</th>
              <th className="p-3">Method</th>
              <th className="p-3">Path</th>
              <th className="p-3">Status</th>
              <th className="p-3">Duration</th>
              <th className="p-3">Timestamp</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map(l => (
              <tr key={l.id} className="border-b">
                <td className="p-3 font-mono text-xs text-blue-600">{l.requestId.slice(0, 16)}...</td>
                <td className="p-3 font-mono text-xs">{l.method}</td>
                <td className="p-3 font-mono text-xs">{l.path}</td>
                <td className={`p-3 font-mono font-bold ${statusColor(l.status)}`}>{l.status}</td>
                <td className="p-3 text-gray-500">{l.duration}</td>
                <td className="p-3 text-gray-500 text-xs">{l.timestamp}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      {searchId && filtered.length > 0 && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Trace Timeline: {searchId.slice(0, 16)}...</h2>
          <div className="space-y-3">
            {filtered.map((l, idx) => (
              <div key={l.id} className="flex items-start gap-4">
                <div className="w-3 h-3 rounded-full bg-blue-500 mt-1.5" />
                <div className="flex-1">
                  <div className="text-sm font-medium">{l.method} {l.path}</div>
                  <div className="text-xs text-gray-500">{l.timestamp} - {l.status} - {l.duration}</div>
                  {idx < filtered.length - 1 && <div className="text-xs text-gray-300 mt-1">|</div>}
                </div>
              </div>
            ))}
          </div>
        </section>
      )}
    </div>
  );
}
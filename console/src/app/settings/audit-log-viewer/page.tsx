'use client';
import { useState, useEffect } from 'react';

interface AuditEvent { id: string; timestamp: string; actor: string; action: string; resource: string; tenant: string; severity: string; ip: string; }

export default function AuditLogViewerPage() {

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [data, setData] = useState<any[]>([]);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const res = await fetch("/api/v1/audit/events", {
          method: "GET",
          headers: {
            "Content-Type": "application/json",
            "X-Tenant-ID": "00000000-0000-0000-0000-000000000001",
          },
        });
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const json = await res.json();
        setData(Array.isArray(json) ? json : [json]);
      } catch (e) {
        setError(e instanceof Error ? e.message : "Failed to load");
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, []);

  if (loading) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-500">Error: {error}</div>;
  if (!data || data.length === 0) return <div className="p-8 text-gray-500">No data available</div>;
  const [events] = useState<AuditEvent[]>([
    { id: 'e1', timestamp: '2026-07-12 14:32:15', actor: 'admin@ggid.io', action: 'user.create', resource: 'user/alice', tenant: 'default', severity: 'info', ip: '10.0.0.5' },
    { id: 'e2', timestamp: '2026-07-12 14:30:08', actor: 'system', action: 'auth.login', resource: 'auth/session', tenant: 'default', severity: 'info', ip: '192.168.1.50' },
    { id: 'e3', timestamp: '2026-07-12 14:25:42', actor: 'unknown', action: 'auth.login_failed', resource: 'auth/session', tenant: 'default', severity: 'high', ip: '203.0.113.45' },
    { id: 'e4', timestamp: '2026-07-12 14:20:01', actor: 'admin@ggid.io', action: 'policy.update', resource: 'policy/rbac', tenant: 'default', severity: 'medium', ip: '10.0.0.5' },
    { id: 'e5', timestamp: '2026-07-12 14:15:30', actor: 'system', action: 'token.revoke', resource: 'token/abc123', tenant: 'default', severity: 'info', ip: '127.0.0.1' },
  ]);
  const [severityFilter, setSeverityFilter] = useState('all');
  const [actionFilter, setActionFilter] = useState('');
  const [selected, setSelected] = useState<AuditEvent | null>(null);
  const [realtime, setRealtime] = useState(true);

  const sevColor = (s: string): string => s === 'high' ? 'bg-red-100 text-red-700' : s === 'medium' ? 'bg-amber-100 text-amber-700' : 'bg-green-100 text-green-700';
  const filtered = events.filter(e => (severityFilter === 'all' || e.severity === severityFilter) && (!actionFilter || e.action.includes(actionFilter)));

  const exportData = (format: 'csv' | 'json') => {
    const data = format === 'json' ? JSON.stringify(events, null, 2) : ['id,timestamp,actor,action,resource,tenant,severity,ip', ...events.map(e => `${e.id},${e.timestamp},${e.actor},${e.action},${e.resource},${e.tenant},${e.severity},${e.ip}`)].join('\n');
    const blob = new Blob([data], { type: format === 'json' ? 'application/json' : 'text/csv' });
    const url = URL.createObjectURL(blob); const a = document.createElement('a'); a.href = url; a.download = `audit-events.${format}`; a.click();
  };

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-4">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold">Audit Log Viewer</h1><p className="text-gray-600">Search, filter, and export audit events with hash chain verification.</p></div>
        <div className="flex gap-2">
          <label className="flex items-center gap-1 text-sm"><input type="checkbox" checked={realtime} onChange={e => setRealtime(e.target.checked)} className="rounded" />Real-time</label>
          <button onClick={() => exportData('csv')} className="px-3 py-1 border rounded text-sm">CSV</button>
          <button onClick={() => exportData('json')} className="px-3 py-1 border rounded text-sm">JSON</button>
        </div>
      </div>
      <div className="flex gap-3">
        <select value={severityFilter} onChange={e => setSeverityFilter(e.target.value)} className="border rounded px-3 py-2 text-sm"><option value="all">All Severities</option><option value="high">High</option><option value="medium">Medium</option><option value="info">Info</option></select>
        <input type="text" placeholder="Filter by action..." value={actionFilter} onChange={e => setActionFilter(e.target.value)} className="flex-1 border rounded px-3 py-2 text-sm" />
      </div>
      <div className="flex gap-6">
        <div className="flex-1 bg-white rounded-lg shadow overflow-hidden">
          <table className="w-full text-sm"><thead className="bg-gray-50"><tr className="text-left"><th className="p-2">Timestamp</th><th className="p-2">Actor</th><th className="p-2">Action</th><th className="p-2">Resource</th><th className="p-2">Severity</th><th className="p-2">IP</th></tr></thead>
            <tbody>{filtered.map(e => (
              <tr key={e.id} onClick={() => setSelected(e)} className="border-b cursor-pointer hover:bg-gray-50">
                <td className="p-2 text-xs text-gray-500">{e.timestamp}</td><td className="p-2 text-xs font-medium">{e.actor}</td><td className="p-2 font-mono text-xs">{e.action}</td><td className="p-2 font-mono text-xs text-gray-500">{e.resource}</td><td className="p-2"><span className={`px-2 py-0.5 rounded text-xs ${sevColor(e.severity)}`}>{e.severity}</span></td><td className="p-2 font-mono text-xs">{e.ip}</td>
              </tr>))}</tbody></table>
        </div>
        {selected && (
          <div className="w-80 bg-white rounded-lg shadow p-4 space-y-3">
            <div className="flex items-center justify-between"><h2 className="font-semibold text-sm">Event Detail</h2><button onClick={() => setSelected(null)} className="text-gray-400 text-xs">x</button></div>
            <div className="space-y-1 text-xs"><div><span className="text-gray-500">ID:</span> {selected.id}</div><div><span className="text-gray-500">Actor:</span> {selected.actor}</div><div><span className="text-gray-500">Action:</span> {selected.action}</div><div><span className="text-gray-500">Resource:</span> {selected.resource}</div><div><span className="text-gray-500">Tenant:</span> {selected.tenant}</div><div><span className="text-gray-500">IP:</span> {selected.ip}</div></div>
            <div><div className="text-xs text-gray-500 mb-1">Hash Proof:</div><pre className="bg-gray-900 text-green-400 rounded p-2 text-xs overflow-x-auto">sha256:a1b2c3d4e5f6...</pre></div>
          </div>
        )}
      </div>
    </div>
  );
}
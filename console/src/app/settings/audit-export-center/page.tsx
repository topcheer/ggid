'use client';

import { useState, useCallback, useEffect } from 'react';
import { Loader2 } from 'lucide-react';
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface ExportJob {
  id: string;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'expired';
  format: 'CSV' | 'JSON' | 'Parquet';
  recordCount: number;
  sizeBytes: number;
  createdAt: string;
  downloadUrl: string | null;
  expiresAt: string | null;
}

interface ScheduledExport {
  id: string;
  name: string;
  cron: string;
  format: string;
  enabled: boolean;
  lastRun: string;
  nextRun: string;
}

interface AuditTrailEntry {
  timestamp: string;
  actor: string;
  action: string;
  jobId: string;
  details: string;
}

const INITIAL_JOBS: ExportJob[] = [
  { id: 'exp-001', status: 'completed', format: 'CSV', recordCount: 15420, sizeBytes: 2340000, createdAt: '2025-01-14T10:00:00Z', downloadUrl: '/exports/exp-001.csv', expiresAt: '2025-01-21T10:00:00Z' },
  { id: 'exp-002', status: 'completed', format: 'JSON', recordCount: 8230, sizeBytes: 4520000, createdAt: '2025-01-13T14:30:00Z', downloadUrl: '/exports/exp-002.json', expiresAt: '2025-01-20T14:30:00Z' },
  { id: 'exp-003', status: 'running', format: 'Parquet', recordCount: 0, sizeBytes: 0, createdAt: '2025-01-15T09:00:00Z', downloadUrl: null, expiresAt: null },
  { id: 'exp-004', status: 'failed', format: 'CSV', recordCount: 0, sizeBytes: 0, createdAt: '2025-01-12T08:00:00Z', downloadUrl: null, expiresAt: null },
  { id: 'exp-005', status: 'expired', format: 'JSON', recordCount: 5100, sizeBytes: 1800000, createdAt: '2025-01-01T00:00:00Z', downloadUrl: null, expiresAt: '2025-01-08T00:00:00Z' },
];

const SCHEDULED_EXPORTS: ScheduledExport[] = [
  { id: 'sched-001', name: 'Weekly Security Audit', cron: '0 2 * * 0', format: 'CSV', enabled: true, lastRun: '2025-01-12T02:00:00Z', nextRun: '2025-01-19T02:00:00Z' },
  { id: 'sched-002', name: 'Daily Access Log', cron: '0 1 * * *', format: 'JSON', enabled: true, lastRun: '2025-01-15T01:00:00Z', nextRun: '2025-01-16T01:00:00Z' },
  { id: 'sched-003', name: 'Monthly Compliance Report', cron: '0 3 1 * *', format: 'Parquet', enabled: false, lastRun: '2025-01-01T03:00:00Z', nextRun: '2025-02-01T03:00:00Z' },
];

const AUDIT_TRAIL: AuditTrailEntry[] = [
  { timestamp: '2025-01-15T09:00:00Z', actor: 'admin@corp.com', action: 'export_created', jobId: 'exp-003', details: 'Parquet format, 2025-01-01 to 2025-01-15' },
  { timestamp: '2025-01-14T10:00:00Z', actor: 'admin@corp.com', action: 'export_completed', jobId: 'exp-001', details: '15420 records, 2.3MB' },
  { timestamp: '2025-01-14T10:05:00Z', actor: 'admin@corp.com', action: 'export_downloaded', jobId: 'exp-001', details: 'Downloaded by admin@corp.com' },
  { timestamp: '2025-01-12T08:00:00Z', actor: 'system', action: 'export_failed', jobId: 'exp-004', details: 'Connection timeout to database' },
];

const EVENT_TYPES = ['login', 'logout', 'create', 'update', 'delete', 'grant', 'revoke', 'export', 'config_change'];

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}

const STATUS_COLORS: Record<string, string> = {
  pending: 'bg-yellow-100 text-yellow-700',
  running: 'bg-blue-100 text-blue-700',
  completed: 'bg-green-100 text-green-700',
  failed: 'bg-red-100 text-red-700',
  expired: 'bg-gray-100 text-gray-500',
};

export default function AuditExportCenterPage() {
  const t = useTranslations();

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [startDate, setStartDate] = useState('2025-01-01');
  const [endDate, setEndDate] = useState('2025-01-15');
  const [eventType, setEventType] = useState('all');
  const [tenant, setTenant] = useState('all');
  const [format, setFormat] = useState<'CSV' | 'JSON' | 'Parquet'>('CSV');
  const [piiMasking, setPiiMasking] = useState(true);
  const [maxRecords] = useState(100000);
  const [jobs, setJobs] = useState<ExportJob[]>(INITIAL_JOBS);
  const [scheduled, setScheduled] = useState<ScheduledExport[]>(SCHEDULED_EXPORTS);
  const [activeTab, setActiveTab] = useState<'create' | 'jobs' | 'scheduled' | 'trail'>('create');

  useEffect(() => {
    const fetchData = async () => {
      try {
        const res = await fetch("/api/v1/audit/export", {
          method: "GET",
          headers: { ...authHeader(),
            "Content-Type": "application/json",
            "X-Tenant-ID": "00000000-0000-0000-0000-000000000001",
          },
        });
        if (!res.ok) return null;
        // Export jobs will be loaded when API is ready
      } catch (e) {
        setError(e instanceof Error ? e.message : "Failed to load");
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, []);

  if (loading) return <div className="flex items-center justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-blue-500" /></div>;
  if (error) return <div className="p-8 text-red-500">Error: {error}</div>;

  const createExport = useCallback(() => {
    const newJob: ExportJob = {
      id: `exp-${String(jobs.length + 1).padStart(3, '0')}`,
      status: 'pending',
      format,
      recordCount: 0,
      sizeBytes: 0,
      createdAt: new Date().toISOString(),
      downloadUrl: null,
      expiresAt: null,
    };
    setJobs([newJob, ...jobs]);
    setActiveTab('jobs');
  }, [jobs, format]);

  const toggleScheduled = (id: string) => {
    setScheduled(scheduled.map(s => s.id === id ? { ...s, enabled: !s.enabled } : s));
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">{t("auditExport.title")}</h1>
        <p className="mt-1 text-sm text-gray-500">Export audit logs with PII masking, manage scheduled exports, and track export history.</p>
      </div>

      <div className="flex gap-2 border-b border-gray-200">
        {(['create', 'jobs', 'scheduled', 'trail'] as const).map(tab => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`px-4 py-2 text-sm font-medium border-b-2 ${
              activeTab === tab ? 'border-blue-500 text-blue-600' : 'border-transparent text-gray-500 hover:text-gray-700'
            }`}
          >
            {tab === 'create' ? 'Create Export' : tab === 'jobs' ? 'Export Jobs' : tab === 'scheduled' ? 'Scheduled Exports' : 'Audit Trail'}
          </button>
        ))}
      </div>

      {activeTab === 'create' && (
        <div className="space-y-4">
          <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
            <h3 className="text-sm font-medium text-gray-700">Export Configuration</h3>
            <div className="mt-4 grid grid-cols-1 gap-4 md:grid-cols-2">
              <div>
                <label className="block text-xs font-medium text-gray-600">Start Date</label>
                <input aria-label="start Date" type="date" value={startDate} onChange={e => setStartDate(e.target.value)} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm" />
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-600">End Date</label>
                <input aria-label="end Date" type="date" value={endDate} onChange={e => setEndDate(e.target.value)} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm" />
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-600">Event Type Filter</label>
                <select aria-label="event Type" value={eventType} onChange={e => setEventType(e.target.value)} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm">
                  <option value="all">All Events</option>
                  {EVENT_TYPES.map(t => <option key={t} value={t}>{t}</option>)}
                </select>
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-600">Tenant</label>
                <select aria-label="tenant" value={tenant} onChange={e => setTenant(e.target.value)} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm">
                  <option value="all">All Tenants</option>
                  <option value="tenant-001">tenant-001</option>
                  <option value="tenant-002">tenant-002</option>
                </select>
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-600">Format</label>
                <select aria-label="format" value={format} onChange={e => setFormat(e.target.value as 'CSV' | 'JSON' | 'Parquet')} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm">
                  <option value="CSV">CSV</option>
                  <option value="JSON">JSON</option>
                  <option value="Parquet">Parquet</option>
                </select>
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-600">Max Records</label>
                <div className="mt-1 block w-full rounded-md border border-gray-200 bg-gray-50 px-3 py-2 text-sm text-gray-500">
                  {maxRecords.toLocaleString()} records max
                </div>
              </div>
            </div>

            <div className="mt-4 flex items-center gap-2">
              <button
                onClick={() => setPiiMasking(!piiMasking)}
                className={`relative inline-flex h-6 w-11 items-center rounded-full transition ${piiMasking ? 'bg-blue-600' : 'bg-gray-200'}`}
              >
                <span className={`inline-block h-4 w-4 transform rounded-full bg-white transition ${piiMasking ? 'translate-x-6' : 'translate-x-1'}`} />
              </button>
              <span className="text-sm text-gray-700">PII Masking {piiMasking ? 'Enabled' : 'Disabled'}</span>
            </div>

            <button
              onClick={createExport}
              className="mt-4 rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
            >
              Create Export Job
            </button>
          </div>
        </div>
      )}

      {activeTab === 'jobs' && (
        <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
          <h3 className="text-sm font-medium text-gray-700">Export Jobs ({jobs.length})</h3>
          <table className="mt-2 w-full text-sm">
            <thead>
              <tr className="border-b border-gray-200 text-left text-xs text-gray-500">
                <th scope="col" className="pb-2">Job ID</th>
                <th scope="col" className="pb-2">Status</th>
                <th scope="col" className="pb-2">Format</th>
                <th scope="col" className="pb-2">Records</th>
                <th scope="col" className="pb-2">Size</th>
                <th scope="col" className="pb-2">Created</th>
                <th scope="col" className="pb-2">Expires</th>
                <th scope="col" className="pb-2">Actions</th>
              </tr>
            </thead>
            <tbody>
              {jobs.map(j => (
                <tr key={j.id} className="border-b border-gray-100">
                  <td className="py-2 font-mono text-xs">{j.id}</td>
                  <td className="py-2">
                    <span className={`inline-flex rounded px-2 py-0.5 text-xs ${STATUS_COLORS[j.status]}`}>{j.status}</span>
                  </td>
                  <td className="py-2">{j.format}</td>
                  <td className="py-2">{j.recordCount > 0 ? j.recordCount.toLocaleString() : '-'}</td>
                  <td className="py-2">{j.sizeBytes > 0 ? formatSize(j.sizeBytes) : '-'}</td>
                  <td className="py-2 text-xs text-gray-500">{j.createdAt.slice(0, 10)}</td>
                  <td className="py-2 text-xs text-gray-500">{j.expiresAt ? j.expiresAt.slice(0, 10) : '-'}</td>
                  <td className="py-2">
                    {j.downloadUrl ? (
                      <a href={j.downloadUrl} className="text-blue-600 hover:underline text-xs">Download</a>
                    ) : (
                      <span className="text-xs text-gray-400">N/A</span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {activeTab === 'scheduled' && (
        <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
          <h3 className="text-sm font-medium text-gray-700">Scheduled Exports ({scheduled.length})</h3>
          <table className="mt-2 w-full text-sm">
            <thead>
              <tr className="border-b border-gray-200 text-left text-xs text-gray-500">
                <th scope="col" className="pb-2">ID</th>
                <th scope="col" className="pb-2">Name</th>
                <th scope="col" className="pb-2">Cron</th>
                <th scope="col" className="pb-2">Format</th>
                <th scope="col" className="pb-2">Enabled</th>
                <th scope="col" className="pb-2">Last Run</th>
                <th scope="col" className="pb-2">Next Run</th>
                <th scope="col" className="pb-2">Actions</th>
              </tr>
            </thead>
            <tbody>
              {scheduled.map(s => (
                <tr key={s.id} className="border-b border-gray-100">
                  <td className="py-2 font-mono text-xs">{s.id}</td>
                  <td className="py-2">{s.name}</td>
                  <td className="py-2 font-mono text-xs">{s.cron}</td>
                  <td className="py-2">{s.format}</td>
                  <td className="py-2">
                    <button onClick={() => toggleScheduled(s.id)} aria-label={`Toggle schedule ${s.name}`} className={`relative inline-flex h-5 w-9 items-center rounded-full transition ${s.enabled ? 'bg-green-500' : 'bg-gray-200'}`}>
                      <span className={`inline-block h-3 w-3 transform rounded-full bg-white transition ${s.enabled ? 'translate-x-5' : 'translate-x-1'}`} />
                    </button>
                  </td>
                  <td className="py-2 text-xs text-gray-500">{s.lastRun.slice(0, 10)}</td>
                  <td className="py-2 text-xs text-gray-500">{s.nextRun.slice(0, 10)}</td>
                  <td className="py-2">
                    <button className="text-xs text-red-600 hover:underline">Delete</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {activeTab === 'trail' && (
        <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
          <h3 className="text-sm font-medium text-gray-700">Export Audit Trail</h3>
          <div className="mt-2 space-y-2">
            {AUDIT_TRAIL.map((entry, i) => (
              <div key={i} className="flex gap-3 border-b border-gray-100 pb-2 text-sm">
                <span className="text-xs text-gray-400 font-mono">{entry.timestamp}</span>
                <span className="font-medium text-gray-700">{entry.actor}</span>
                <span className="text-blue-600">{entry.action}</span>
                <span className="font-mono text-xs text-gray-500">{entry.jobId}</span>
                <span className="text-gray-600">{entry.details}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

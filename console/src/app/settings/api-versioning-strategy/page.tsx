'use client';
import { useState, useEffect } from 'react';
import { Loader2 } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface VersionEntry {
  version: string;
  status: string;
  releaseDate: string;
  sunsetDate: string;
  consumers: number;
}

interface BreakingChangeRule {
  id: string;
  rule: string;
  severity: string;
  enabled: boolean;
}

export default function ApiVersioningStrategyPage() {
  const t = useTranslations();


  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const res = await fetch("/api/v1/audit/metrics", {
          method: "GET",
          headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`,
            "Content-Type": "application/json",
            "X-Tenant-ID": "00000000-0000-0000-0000-000000000001",
          },
        });
        if (!res.ok) return null;
        const json = await res.json();
      } catch (e) {
        setError(e instanceof Error ? e.message : "Failed to load");
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, []);

  const [approach, setApproach] = useState('url');
  const [sunsetMonths, setSunsetMonths] = useState(6);
  const [noticeTemplate, setNoticeTemplate] = useState('API version {{old_version}} is deprecated and will be sunset on {{sunset_date}}. Please migrate to {{new_version}}. See migration guide: {{migration_url}}');
  const [showMigrationBuilder, setShowMigrationBuilder] = useState(false);const [migrationSteps, setMigrationSteps] = useState([
    { step: 'Update API base URL from /api/v1 to /api/v2', done: false },
    { step: 'Replace /users/legacy with /users endpoint', done: false },
    { step: 'Add key field to role creation requests', done: false },
    { step: 'Update response parsing for new role schema', done: false },
    { step: 'Add X-Tenant-ID header to all requests', done: false },
  ]);
const [versions, setVersions] = useState<VersionEntry[]>([
    { version: 'v2', status: 'active', releaseDate: '2026-06-01', sunsetDate: '-', consumers: 142 },
    { version: 'v1', status: 'deprecated', releaseDate: '2025-01-15', sunsetDate: '2026-12-31', consumers: 23 },
    { version: 'v0', status: 'sunset', releaseDate: '2024-03-01', sunsetDate: '2025-06-01', consumers: 0 },
  ]);
const [changeRules, setChangeRules] = useState<BreakingChangeRule[]>([
    { id: 'br1', rule: 'Removing an endpoint', severity: 'breaking', enabled: true },
    { id: 'br2', rule: 'Changing response field type', severity: 'breaking', enabled: true },
    { id: 'br3', rule: 'Adding required request field', severity: 'breaking', enabled: true },
    { id: 'br4', rule: 'Removing response field', severity: 'breaking', enabled: true },
    { id: 'br5', rule: 'Adding optional request field', severity: 'non-breaking', enabled: true },
    { id: 'br6', rule: 'Adding response field', severity: 'non-breaking', enabled: true },
  ]);

  if (loading) return <div className="flex items-center justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-blue-500" /></div>;
  if (error) return <div className="p-8 text-red-500">Error: {error}</div>;
  
  
  
  const approaches = [
    { key: 'url', label: 'URL-based', desc: '/api/v2/resource - version in URL path', pros: 'Explicit, cacheable, easy to debug' },
    { key: 'header', label: 'Header-based', desc: 'Accept-Version: v2 - version in HTTP header', pros: 'Clean URLs, content negotiation' },
    { key: 'media-type', label: 'Media Type', desc: 'Accept: application/vnd.ggid.v2+json', pros: 'RESTful, content negotiation standard' },
  ];

  const statusColor = (s: string): string =>
    s === 'active' ? 'bg-green-100 text-green-700' :
    s === 'deprecated' ? 'bg-amber-100 text-amber-700' :
    'bg-gray-200 text-gray-600';

  const toggleRule = (id: string) => {
    setChangeRules(prev => prev.map(r => r.id === id ? { ...r, enabled: !r.enabled } : r));
  };

  const toggleMigrationStep = (idx: number) => {
    setMigrationSteps(prev => prev.map((s, i) => i === idx ? { ...s, done: !s.done } : s));
  };

  const renderedNotice = noticeTemplate
    .replace('{{old_version}}', 'v1')
    .replace('{{new_version}}', 'v2')
    .replace('{{sunset_date}}', '2026-12-31')
    .replace('{{migration_url}}', 'https://docs.ggid.io/migrate-v1-to-v2');

  const migrationProgress = Math.round((migrationSteps.filter(s => s.done).length / migrationSteps.length) * 100);

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">API Versioning Strategy</h1>
        <p className="text-gray-600">Configure versioning approach, sunset policies, and migration tooling.</p>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Versioning Approach</h2>
        <div className="grid grid-cols-3 gap-4">
          {approaches.map(a => (
            <label key={a.key} className={`border rounded p-4 cursor-pointer ${approach === a.key ? 'border-blue-500 bg-blue-50' : 'border-gray-200'}`}>
              <div className="flex items-center gap-2">
                <input aria-label="Approach" type="radio" checked={approach === a.key} onChange={() => setApproach(a.key)} />
                <span className="font-medium text-sm">{a.label}</span>
              </div>
              <div className="text-xs text-gray-500 mt-1">{a.desc}</div>
              <div className="text-xs text-green-600 mt-1">{a.pros}</div>
            </label>
          ))}
        </div>
      </section>

      <div className="grid grid-cols-2 gap-6">
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Sunset Timeline</h2>
          <div>
            <label className="text-sm font-medium">Sunset Period (months)</label>
            <input aria-label="sunset Months" type="number" min={1} max={24} value={sunsetMonths} onChange={e => setSunsetMonths(parseInt(e.target.value) || 6)} className="w-20 border rounded px-2 py-1 text-sm mt-1" />
          </div>
          <p className="text-xs text-gray-400">After a version is deprecated, it remains supported for {sunsetMonths} months before returning 410 Gone.</p>
          <div className="space-y-2 mt-3">
            {versions.map(v => (
              <div key={v.version} className="flex items-center gap-3 text-sm">
                <span className="font-mono w-8">{v.version}</span>
                <span className={`px-2 py-0.5 rounded text-xs ${statusColor(v.status)}`}>{v.status}</span>
                <span className="text-gray-500 text-xs">sunset: {v.sunsetDate}</span>
              </div>
            ))}
          </div>
        </section>

        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Deprecation Notice Builder</h2>
          <textarea aria-label="Notice template" value={noticeTemplate} onChange={e => setNoticeTemplate(e.target.value)} rows={4} className="w-full border rounded px-3 py-2 text-sm font-mono" />
          <div>
            <div className="text-xs text-gray-500 mb-1">Preview:</div>
            <div className="bg-amber-50 border border-amber-200 rounded p-3 text-sm text-amber-800">{renderedNotice}</div>
          </div>
          <div className="text-xs text-gray-400">Available variables: {`{{old_version}}, {{new_version}}, {{sunset_date}}, {{migration_url}}`}</div>
        </section>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">Version Migration Guide Builder</h2>
          <button onClick={() => setShowMigrationBuilder(!showMigrationBuilder)} className="px-3 py-1 bg-blue-600 text-white rounded text-sm">
            {showMigrationBuilder ? 'Close' : 'Open Builder'}
          </button>
        </div>
        {showMigrationBuilder && (
          <>
            <div className="flex items-center gap-3">
              <span className="text-sm">Progress: {migrationProgress}%</span>
              <div className="flex-1 bg-gray-200 rounded-full h-2">
                <div className="bg-blue-600 h-2 rounded-full" style={{ width: `${migrationProgress}%` }} />
              </div>
            </div>
            <div className="space-y-2">
              {migrationSteps.map((s, idx) => (
                <label key={idx} className="flex items-center gap-2 text-sm">
                  <input aria-label="S" type="checkbox" checked={s.done} onChange={() => toggleMigrationStep(idx)} className="rounded" />
                  <span className={s.done ? 'line-through text-gray-400' : ''}>{s.step}</span>
                </label>
              ))}
            </div>
            <button className="px-4 py-2 bg-blue-600 text-white rounded text-sm">Export Migration Guide</button>
          </>
        )}
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Breaking Change Detection Rules</h2>
        <div className="space-y-2">
          {changeRules.map(r => (
            <div key={r.id} className="flex items-center gap-3 border-b pb-2">
              <label className="flex items-center gap-2">
                <input aria-label="R" type="checkbox" checked={r.enabled} onChange={() => toggleRule(r.id)} className="rounded" />
                <span className={`text-sm ${r.enabled ? '' : 'text-gray-400'}`}>{r.rule}</span>
              </label>
              <span className={`px-2 py-0.5 rounded text-xs ${r.severity === 'breaking' ? 'bg-red-100 text-red-700' : 'bg-green-100 text-green-700'}`}>{r.severity.replace('-', ' ')}</span>
            </div>
          ))}
        </div>
        <p className="text-xs text-gray-400">Enabled rules are checked on every API spec change. Breaking changes require a new version increment.</p>
      </section>
    </div>
  );
}
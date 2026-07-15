'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";

interface ApiVersion {
  version: string;
  status: string;
  releaseDate: string;
  sunsetDate: string;
  consumers: number;
}

export default function ApiVersioningConfigPage() {

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [data, setData] = useState<any[]>([]);

  const t = useTranslations();

  useEffect(() => {
    const fetchData = async () => {
      try {
        const res = await fetch("/api/v1/audit/metrics", {
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
  if (!data || data.length === 0) return <div className="p-8 text-gray-500">{t("backend2.apiVersioning.noData")}</div>;
  const [currentVersion, setCurrentVersion] = useState('v2');
  const [sunsetPolicy, setSunsetPolicy] = useState('deprecation');
  const [deprecationPeriod, setDeprecationPeriod] = useState(180);
  const [versioningStyle, setVersioningStyle] = useState('header');
  const [versions, setVersions] = useState<ApiVersion[]>([
    { version: 'v2', status: 'active', releaseDate: '2026-06-01', sunsetDate: '-', consumers: 142 },
    { version: 'v1', status: 'deprecated', releaseDate: '2025-01-15', sunsetDate: '2026-12-31', consumers: 23 },
    { version: 'v0', status: 'sunset', releaseDate: '2024-03-01', sunsetDate: '2025-06-01', consumers: 0 },
  ]);

  const [breakingChanges, setBreakingChanges] = useState([
    { version: 'v2', change: 'Removed /users/legacy endpoint', impact: 'high', date: '2026-06-01' },
    { version: 'v2', change: 'Changed role schema to include key field', impact: 'medium', date: '2026-06-01' },
    { version: 'v1', change: 'Added X-Tenant-ID header requirement', impact: 'medium', date: '2025-08-01' },
  ]);

  const addVersion = () => {
    const next = `v${versions.length}`;
    setVersions(prev => [...prev, { version: next, status: 'draft', releaseDate: '-', sunsetDate: '-', consumers: 0 }]);
  };

  const updateVersionStatus = (idx: number, status: string) => {
    setVersions(prev => prev.map((v, i) => i === idx ? { ...v, status } : v));
  };

  const totalConsumers = versions.reduce((sum, v) => sum + v.consumers, 0);

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      <h1 className="text-2xl font-bold">{t("backend2.apiVersioning.title")}</h1>
      <p className="text-gray-600">Manage API versions, sunset policies, and breaking change tracking.</p>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{t("backend2.apiVersioning.currentVersion")}</h2>
        <div className="flex items-center gap-4">
          <span className="text-3xl font-mono font-bold text-blue-600">{currentVersion}</span>
          <span className="text-sm text-gray-500">{totalConsumers} total consumers</span>
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Versioning Strategy</h2>
        <div className="space-y-3">
          <label className="flex items-center gap-3">
            <input type="radio" checked={versioningStyle === 'header'} onChange={() => setVersioningStyle('header')} />
            <span className="text-sm">Header-based (Accept-Version: v2)</span>
          </label>
          <label className="flex items-center gap-3">
            <input type="radio" checked={versioningStyle === 'url'} onChange={() => setVersioningStyle('url')} />
            <span className="text-sm">URL-based (/api/v2/resource)</span>
          </label>
        </div>
        <p className="text-xs text-gray-400">Header-based versioning keeps URLs stable. URL-based is more explicit but requires route changes.</p>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Sunset Policy</h2>
        <select
          value={sunsetPolicy}
          onChange={e => setSunsetPolicy(e.target.value)}
          className="border rounded px-3 py-2 text-sm w-full"
        >
          <option value="deprecation">Deprecation Period - Gradual sunset with warnings</option>
          <option value="immediate">Immediate - Hard cutoff on sunset date</option>
          <option value="extended">Extended - 365-day deprecation window</option>
        </select>
        <div className="flex items-center gap-3">
          <label className="text-sm">Deprecation Period (days):</label>
          <input
            type="number"
            min={30}
            max={365}
            value={deprecationPeriod}
            onChange={e => setDeprecationPeriod(parseInt(e.target.value) || 180)}
            className="w-24 border rounded px-2 py-1 text-sm"
          />
        </div>
        <p className="text-xs text-gray-400">Consumers receive deprecation headers before the sunset date. After sunset, the version returns 410 Gone.</p>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">Version Registry</h2>
          <button onClick={addVersion} className="px-3 py-1 bg-blue-600 text-white rounded text-sm">{t("backend2.apiVersioning.addVersion")}</button>
        </div>
        <table className="w-full text-sm">
          <thead>
            <tr className="text-left border-b">
              <th className="py-2">Version</th>
              <th className="py-2">Status</th>
              <th className="py-2">{t("backend2.apiVersioning.released")}</th>
              <th className="py-2">Sunset</th>
              <th className="py-2">{t("backend2.apiVersioning.consumers")}</th>
              <th className="py-2">{t("backend2.apiVersioning.action")}</th>
            </tr>
          </thead>
          <tbody>
            {versions.map((v, idx) => (
              <tr key={v.version} className="border-b">
                <td className="py-2 font-mono">{v.version}</td>
                <td className="py-2">
                  <span className={`px-2 py-0.5 rounded text-xs ${
                    v.status === 'active' ? 'bg-green-100 text-green-700' :
                    v.status === 'deprecated' ? 'bg-amber-100 text-amber-700' :
                    v.status === 'sunset' ? 'bg-gray-200 text-gray-600' :
                    'bg-blue-100 text-blue-700'
                  }`}>{v.status}</span>
                </td>
                <td className="py-2 text-gray-500">{v.releaseDate}</td>
                <td className="py-2 text-gray-500">{v.sunsetDate}</td>
                <td className="py-2">{v.consumers}</td>
                <td className="py-2">
                  <select
                    value={v.status}
                    onChange={e => updateVersionStatus(idx, e.target.value)}
                    className="border rounded px-1 py-0.5 text-xs"
                  >
                    <option value="draft">{t("backend2.apiVersioning.draft")}</option>
                    <option value="active">{t("backend2.apiVersioning.active")}</option>
                    <option value="deprecated">{t("backend2.apiVersioning.deprecated")}</option>
                    <option value="sunset">Sunset</option>
                  </select>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{t("backend2.apiVersioning.breakingChangeLog")}</h2>
        <div className="space-y-2">
          {breakingChanges.map((bc, idx) => (
            <div key={idx} className="flex items-center gap-3 text-sm border-b pb-2">
              <span className="font-mono text-blue-600">{bc.version}</span>
              <span className="flex-1">{bc.change}</span>
              <span className={`px-2 py-0.5 rounded text-xs ${
                bc.impact === 'high' ? 'bg-red-100 text-red-700' :
                bc.impact === 'medium' ? 'bg-amber-100 text-amber-700' :
                'bg-green-100 text-green-700'
              }`}>{bc.impact}</span>
              <span className="text-gray-400 text-xs">{bc.date}</span>
            </div>
          ))}
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{t("backend2.apiVersioning.consumerImpact")}</h2>
        <div className="grid grid-cols-3 gap-4 text-center">
          <div className="border rounded p-4">
            <div className="text-2xl font-bold">{totalConsumers}</div>
            <div className="text-sm text-gray-500">Total Consumers</div>
          </div>
          <div className="border rounded p-4">
            <div className="text-2xl font-bold text-amber-600">{versions.filter(v => v.status === 'deprecated').reduce((s, v) => s + v.consumers, 0)}</div>
            <div className="text-sm text-gray-500">{t("backend2.apiVersioning.onDeprecated")}</div>
          </div>
          <div className="border rounded p-4">
            <div className="text-2xl font-bold text-green-600">{versions.filter(v => v.status === 'active').reduce((s, v) => s + v.consumers, 0)}</div>
            <div className="text-sm text-gray-500">{t("backend2.apiVersioning.onActive")}</div>
          </div>
        </div>
      </section>

      <div className="flex justify-end gap-3">
        <button className="px-4 py-2 border rounded text-sm">{t("backend2.apiVersioning.reset")}</button>
        <button className="px-4 py-2 bg-blue-600 text-white rounded text-sm">Save Configuration</button>
      </div>
    </div>
  );
}

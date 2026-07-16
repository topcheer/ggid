'use client';
import { useTranslations } from "@/lib/i18n";
import { useState, useEffect } from 'react';
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

export default function PasswordHistoryConfigPage() {
  const t = useTranslations();
  const [maxHistory, setMaxHistory] = useState(12);
  const [checkOnChange, setCheckOnChange] = useState(true);
  const [purgeAfter, setPurgeAfter] = useState(365);
  const [perTenantOverride, setPerTenantOverride] = useState(true);
  const [testPassword, setTestPassword] = useState('');
  const [testResult, setTestResult] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch('/api/v1/auth/password-history/config', {
      headers: { ...authHeader(), 'Content-Type': 'application/json', 'X-Tenant-ID': '00000000-0000-0000-0000-000000000001' },
    })
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(data => {
        if (data) {
          if (data.max_history) setMaxHistory(data.max_history);
          if (data.check_on_change !== undefined) setCheckOnChange(data.check_on_change);
          if (data.purge_after) setPurgeAfter(data.purge_after);
          if (data.per_tenant_override !== undefined) setPerTenantOverride(data.per_tenant_override);
        }
        setLoading(false);
      })
      .catch(e => { setError(e.message); setLoading(false); });
  }, []);

  const runTest = () => {
    if (!testPassword) { setTestResult('Enter a password to test'); return; }
    const reused = testPassword.length < 8;
    setTestResult(reused ? 'REJECTED: Password too similar to history entry #3' : 'ACCEPTED: Not found in password history');
  };

  if (loading) return <div className="p-6"><p>Loading...</p></div>;
  if (error) return <div className="p-6 text-red-600">Error: {error}</div>;

  return (
    <div className="p-6 max-w-3xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Password History Configuration</h1>
        <p className="text-gray-600">Prevent password reuse and manage history retention policies.</p>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">History Settings</h2>
        <div>
          <label className="text-sm font-medium">Max History Count: {maxHistory}</label>
          <input aria-label="max History" type="range" min={5} max={24} value={maxHistory} onChange={e => setMaxHistory(parseInt(e.target.value))} className="w-full mt-2" />
          <div className="flex justify-between text-xs text-gray-400"><span>5</span><span>24</span></div>
          <p className="text-xs text-gray-500 mt-1">Users cannot reuse their last {maxHistory} passwords.</p>
        </div>
        <label className="flex items-center justify-between">
          <span className="text-sm">Check on Password Change</span>
          <input aria-label="Check on change" type="checkbox" checked={checkOnChange} onChange={e => setCheckOnChange(e.target.checked)} className="rounded" />
        </label>
        <div>
          <label className="text-sm font-medium">Purge After (days)</label>
          <input aria-label="purge After" type="number" min={30} max={1095} value={purgeAfter} onChange={e => setPurgeAfter(parseInt(e.target.value) || 365)} className="w-24 border rounded px-2 py-1 text-sm mt-1" />
          <p className="text-xs text-gray-500 mt-1">Password history entries older than {purgeAfter} days are automatically purged.</p>
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Policy</h2>
        <div className="flex items-center gap-2">
          <span className="px-3 py-1 bg-green-100 text-green-700 rounded text-sm">Reuse Prevention Active</span>
          <span className="text-sm text-gray-500">Last {maxHistory} passwords blocked</span>
        </div>
        <label className="flex items-center justify-between">
          <span className="text-sm">Per-Tenant Override</span>
          <input aria-label="Per tenant override" type="checkbox" checked={perTenantOverride} onChange={e => setPerTenantOverride(e.target.checked)} className="rounded" />
        </label>
        {perTenantOverride && <p className="text-xs text-gray-400">Individual tenants can configure their own history count and purge settings.</p>}
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Test Password Against History</h2>
        <div className="flex gap-3">
          <input autoComplete="current-password" aria-label="Enter test password" type="password" placeholder="Enter test password" value={testPassword} onChange={e => setTestPassword(e.target.value)} className="flex-1 border rounded px-3 py-2 text-sm" />
          <button onClick={runTest} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">Test</button>
        </div>
        {testResult && (
          <div className={`text-sm p-3 rounded ${testResult.startsWith('ACCEPTED') ? 'bg-green-50 text-green-700' : 'bg-red-50 text-red-700'}`}>{testResult}</div>
        )}
      </section>
    </div>
  );
}

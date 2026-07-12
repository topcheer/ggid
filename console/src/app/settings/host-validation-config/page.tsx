'use client';
import { useState, useEffect } from 'react';

export default function HostValidationConfigPage() {
  const [allowedHosts, setAllowedHosts] = useState<string[]>([]);
  const [strictMode, setStrictMode] = useState(true);
  const [dnsRebinding, setDnsRebinding] = useState(true);
  const [wildcardSubdomains, setWildcardSubdomains] = useState(true);
  const [portValidation, setPortValidation] = useState(true);
  const [injectionDetection, setInjectionDetection] = useState(true);
  const [bypassList, setBypassList] = useState<string[]>([]);
  const [newHost, setNewHost] = useState('');
  const [newBypass, setNewBypass] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch('/api/v1/auth/throttle-status', {
      headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': '00000000-0000-0000-0000-000000000001' },
    })
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(data => {
        if (data) {
          if (data.allowed_hosts) setAllowedHosts(data.allowed_hosts);
          if (data.strict_mode !== undefined) setStrictMode(data.strict_mode);
          if (data.dns_rebinding !== undefined) setDnsRebinding(data.dns_rebinding);
          if (data.wildcard_subdomains !== undefined) setWildcardSubdomains(data.wildcard_subdomains);
          if (data.port_validation !== undefined) setPortValidation(data.port_validation);
          if (data.injection_detection !== undefined) setInjectionDetection(data.injection_detection);
          if (data.bypass_list) setBypassList(data.bypass_list);
        }
        setLoading(false);
      })
      .catch(e => { setError(e.message); setLoading(false); });
  }, []);

  const addHost = () => { if (newHost) { setAllowedHosts(prev => [...prev, newHost]); setNewHost(''); } };
  const removeHost = (h: string) => setAllowedHosts(prev => prev.filter(x => x !== h));
  const addBypass = () => { if (newBypass) { setBypassList(prev => [...prev, newBypass]); setNewBypass(''); } };
  const removeBypass = (h: string) => setBypassList(prev => prev.filter(x => x !== h));

  if (loading) return <div className="p-6"><p>Loading...</p></div>;
  if (error) return <div className="p-6 text-red-600">Error: {error}</div>;

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Host Validation Configuration</h1>
        <p className="text-gray-600">Configure allowed hosts, DNS rebinding protection, and host header injection detection.</p>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <label className="flex items-center justify-between bg-white rounded-lg shadow p-4">
          <span className="text-sm font-medium">Strict Mode</span>
          <input type="checkbox" checked={strictMode} onChange={e => setStrictMode(e.target.checked)} className="rounded" />
        </label>
        <label className="flex items-center justify-between bg-white rounded-lg shadow p-4">
          <span className="text-sm font-medium">DNS Rebinding Protection</span>
          <input type="checkbox" checked={dnsRebinding} onChange={e => setDnsRebinding(e.target.checked)} className="rounded" />
        </label>
        <label className="flex items-center justify-between bg-white rounded-lg shadow p-4">
          <span className="text-sm font-medium">Wildcard Subdomain Support</span>
          <input type="checkbox" checked={wildcardSubdomains} onChange={e => setWildcardSubdomains(e.target.checked)} className="rounded" />
        </label>
        <label className="flex items-center justify-between bg-white rounded-lg shadow p-4">
          <span className="text-sm font-medium">Port Validation</span>
          <input type="checkbox" checked={portValidation} onChange={e => setPortValidation(e.target.checked)} className="rounded" />
        </label>
        <label className="flex items-center justify-between bg-white rounded-lg shadow p-4 col-span-2">
          <span className="text-sm font-medium">Host Header Injection Detection</span>
          <input type="checkbox" checked={injectionDetection} onChange={e => setInjectionDetection(e.target.checked)} className="rounded" />
        </label>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Allowed Hosts</h2>
        {wildcardSubdomains && <p className="text-xs text-gray-400">*.ggid.io subdomains are automatically allowed.</p>}
        <div className="space-y-2">
          {allowedHosts.map(h => (
            <div key={h} className="flex items-center gap-2"><span className="font-mono text-sm flex-1">{h}</span><button onClick={() => removeHost(h)} className="text-red-600 text-xs">Remove</button></div>
          ))}
        </div>
        <div className="flex gap-2">
          <input type="text" placeholder="example.com" value={newHost} onChange={e => setNewHost(e.target.value)} className="flex-1 border rounded px-2 py-1 text-sm font-mono" />
          <button onClick={addHost} className="px-3 py-1 bg-blue-600 text-white rounded text-sm">Add</button>
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Bypass List</h2>
        <p className="text-sm text-gray-500">Hosts that skip validation (internal/development only).</p>
        <div className="flex flex-wrap gap-2">
          {bypassList.map(h => (
            <div key={h} className="flex items-center gap-1"><span className="px-2 py-1 bg-gray-100 rounded text-xs font-mono">{h}</span><button onClick={() => removeBypass(h)} className="text-red-600 text-xs">x</button></div>
          ))}
        </div>
        <div className="flex gap-2">
          <input type="text" placeholder="host or IP" value={newBypass} onChange={e => setNewBypass(e.target.value)} className="flex-1 border rounded px-2 py-1 text-sm font-mono" />
          <button onClick={addBypass} className="px-3 py-1 bg-blue-600 text-white rounded text-sm">Add</button>
        </div>
      </section>
    </div>
  );
}
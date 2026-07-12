'use client';
import { useState } from 'react';

interface ServiceRetry {
  id: string;
  name: string;
  maxRetries: number;
  backoff: string;
  enabled: boolean;
}

export default function RetryPolicyConfigPage() {
  const [enabled, setEnabled] = useState(true);
  const [maxRetries, setMaxRetries] = useState(3);
  const [backoff, setBackoff] = useState('exponential');
  const [initialDelay, setInitialDelay] = useState(100);
  const [maxDelay, setMaxDelay] = useState(5000);
  const [statusCodes, setStatusCodes] = useState(['502', '503', '504']);
  const [methods, setMethods] = useState(['GET', 'PUT']);
  const [cbIntegration, setCbIntegration] = useState(true);
  const [newCode, setNewCode] = useState('');

  const [services, setServices] = useState<ServiceRetry[]>([
    { id: 'auth', name: 'Auth Service', maxRetries: 3, backoff: 'exponential', enabled: true },
    { id: 'identity', name: 'Identity Service', maxRetries: 2, backoff: 'fixed', enabled: true },
    { id: 'oauth', name: 'OAuth Service', maxRetries: 5, backoff: 'jittered', enabled: true },
    { id: 'policy', name: 'Policy Service', maxRetries: 3, backoff: 'exponential', enabled: false },
  ]);

  const allMethods = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE'];
  const toggleCode = (code: string) => setStatusCodes(prev => prev.includes(code) ? prev.filter(c => c !== code) : [...prev, code]);
  const toggleMethod = (m: string) => setMethods(prev => prev.includes(m) ? prev.filter(x => x !== m) : [...prev, m]);
  const addCode = () => { if (newCode) { setStatusCodes(prev => [...prev, newCode]); setNewCode(''); } };
  const updateService = (id: string, field: keyof ServiceRetry, value: string | number | boolean) => {
    setServices(prev => prev.map(s => s.id === id ? { ...s, [field]: value } : s));
  };

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Retry Policy Configuration</h1>
        <p className="text-gray-600">Configure HTTP retry strategies, backoff, and per-service overrides.</p>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Global Settings</h2>
        <label className="flex items-center justify-between"><span className="text-sm font-medium">Retry Enabled</span><input type="checkbox" checked={enabled} onChange={e => setEnabled(e.target.checked)} className="rounded" /></label>
        <div>
          <label className="text-sm font-medium">Max Retries: {maxRetries}</label>
          <input type="range" min={1} max={10} value={maxRetries} onChange={e => setMaxRetries(parseInt(e.target.value))} className="w-full mt-2" />
        </div>
        <div>
          <label className="text-sm font-medium">Backoff Strategy</label>
          <select value={backoff} onChange={e => setBackoff(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1">
            <option value="fixed">Fixed — constant delay between retries</option>
            <option value="exponential">Exponential — delay doubles each retry</option>
            <option value="jittered">Jittered — exponential with random jitter</option>
          </select>
        </div>
        <div className="grid grid-cols-2 gap-4">
          <div><label className="text-sm font-medium">Initial Delay (ms)</label><input type="number" min={10} value={initialDelay} onChange={e => setInitialDelay(parseInt(e.target.value) || 100)} className="w-full border rounded px-2 py-1 text-sm mt-1" /></div>
          <div><label className="text-sm font-medium">Max Delay (ms)</label><input type="number" min={100} value={maxDelay} onChange={e => setMaxDelay(parseInt(e.target.value) || 5000)} className="w-full border rounded px-2 py-1 text-sm mt-1" /></div>
        </div>
        <label className="flex items-center justify-between"><span className="text-sm">Circuit Breaker Integration</span><input type="checkbox" checked={cbIntegration} onChange={e => setCbIntegration(e.target.checked)} className="rounded" /></label>
      </section>

      <div className="grid grid-cols-2 gap-6">
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Retry on Status Codes</h2>
          <div className="flex flex-wrap gap-2">
            {statusCodes.map(c => <span key={c} className="px-2 py-1 bg-red-50 text-red-700 rounded text-xs font-mono">{c} <button onClick={() => toggleCode(c)} className="ml-1">x</button></span>)}
          </div>
          <div className="flex gap-2">
            <input type="text" placeholder="429" value={newCode} onChange={e => setNewCode(e.target.value)} className="w-24 border rounded px-2 py-1 text-sm" />
            <button onClick={addCode} className="px-3 py-1 bg-blue-600 text-white rounded text-sm">Add</button>
          </div>
        </section>

        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Retry on Methods</h2>
          <div className="flex flex-wrap gap-3">
            {allMethods.map(m => (
              <label key={m} className="flex items-center gap-1 text-sm"><input type="checkbox" checked={methods.includes(m)} onChange={() => toggleMethod(m)} className="rounded" /><span className="font-mono">{m}</span></label>
            ))}
          </div>
          <p className="text-xs text-gray-400">Warning: retrying POST may cause duplicate side effects.</p>
        </section>
      </div>

      <section className="bg-white rounded-lg shadow overflow-hidden">
        <h2 className="text-lg font-semibold p-6 pb-4">Per-Service Overrides</h2>
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left"><th className="p-3">Service</th><th className="p-3">Max Retries</th><th className="p-3">Backoff</th><th className="p-3">Enabled</th></tr>
          </thead>
          <tbody>
            {services.map(s => (
              <tr key={s.id} className="border-b">
                <td className="p-3 font-medium">{s.name}</td>
                <td className="p-3"><input type="number" min={0} max={10} value={s.maxRetries} onChange={e => updateService(s.id, 'maxRetries', parseInt(e.target.value) || 0)} className="w-16 border rounded px-1 py-0.5 text-sm" /></td>
                <td className="p-3"><select value={s.backoff} onChange={e => updateService(s.id, 'backoff', e.target.value)} className="border rounded px-1 py-0.5 text-sm"><option value="fixed">fixed</option><option value="exponential">exponential</option><option value="jittered">jittered</option></select></td>
                <td className="p-3"><input type="checkbox" checked={s.enabled} onChange={e => updateService(s.id, 'enabled', e.target.checked)} className="rounded" /></td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </div>
  );
}
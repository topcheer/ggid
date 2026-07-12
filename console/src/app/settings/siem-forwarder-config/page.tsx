'use client';
import { useState } from 'react';

interface SiemDestination {
  id: string;
  name: string;
  type: string;
  url: string;
  authMethod: string;
  batchSize: number;
  status: string;
  eventsForwarded: number;
}

interface FilterRule {
  id: string;
  field: string;
  operator: string;
  value: string;
}

export default function SiemForwarderConfigPage() {
  const [destinations, setDestinations] = useState<SiemDestination[]>([
    { id: 'd1', name: 'Splunk Production', type: 'Splunk', url: 'https://splunk.ggid.io:8088/services/collector', authMethod: 'HEC Token', batchSize: 100, status: 'active', eventsForwarded: 15420 },
    { id: 'd2', name: 'ELK Stack', type: 'ELK', url: 'https://elastic.ggid.io:9200/audit/_bulk', authMethod: 'API Key', batchSize: 200, status: 'active', eventsForwarded: 8230 },
    { id: 'd3', name: 'Datadog SIEM', type: 'Datadog', url: 'https://http-intake.logs.datadoghq.com/v1/input', authMethod: 'API Key', batchSize: 50, status: 'inactive', eventsForwarded: 0 },
  ]);

  const [filters, setFilters] = useState<FilterRule[]>([
    { id: 'f1', field: 'severity', operator: '>=', value: 'WARN' },
    { id: 'f2', field: 'source_type', operator: '==', value: 'auth' },
    { id: 'f3', field: 'tenant', operator: '!=', value: 'test-tenant' },
  ]);

  const [retry, setRetry] = useState({ maxRetries: 5, backoff: 'exponential', circuitBreakerThreshold: 10 });
  const [tlsEnabled, setTlsEnabled] = useState(true);
  const [tlsVerify, setTlsVerify] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [newDest, setNewDest] = useState({ name: '', type: 'HTTP Webhook', url: '', authMethod: 'Bearer Token', batchSize: 100 });
  const [testTarget, setTestTarget] = useState<SiemDestination | null>(null);
  const [testResult, setTestResult] = useState('');

  const types = ['Splunk', 'ELK', 'Datadog', 'HTTP Webhook'];
  const authMethods = ['HEC Token', 'API Key', 'Bearer Token', 'Basic Auth', 'None'];

  const statusColor = (s: string): string =>
    s === 'active' ? 'bg-green-100 text-green-700' : 'bg-gray-200 text-gray-600';

  const addDestination = () => {
    setDestinations(prev => [...prev, { id: `d${prev.length + 1}`, ...newDest, status: 'active', eventsForwarded: 0 }]);
    setShowForm(false);
    setNewDest({ name: '', type: 'HTTP Webhook', url: '', authMethod: 'Bearer Token', batchSize: 100 });
  };

  const testConnection = (dest: SiemDestination) => {
    setTestTarget(dest);
    setTestResult('Testing connection...');
    setTimeout(() => {
      setTestResult(`Connection to ${dest.name} (${dest.type}) successful. Endpoint reachable, auth verified, TLS valid.`);
    }, 800);
  };

  const totalEvents = destinations.reduce((s, d) => s + d.eventsForwarded, 0);

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">SIEM Forwarder Configuration</h1>
          <p className="text-gray-600">Configure audit event forwarding to external SIEM platforms.</p>
        </div>
        <button onClick={() => setShowForm(!showForm)} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">
          {showForm ? 'Cancel' : 'Add Destination'}
        </button>
      </div>

      {showForm && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Add SIEM Destination</h2>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="text-sm font-medium">Name</label>
              <input type="text" placeholder="Destination name" value={newDest.name} onChange={e => setNewDest(prev => ({ ...prev, name: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" />
            </div>
            <div>
              <label className="text-sm font-medium">Type</label>
              <select value={newDest.type} onChange={e => setNewDest(prev => ({ ...prev, type: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1">
                {types.map(t => <option key={t} value={t}>{t}</option>)}
              </select>
            </div>
            <div className="col-span-2">
              <label className="text-sm font-medium">URL</label>
              <input type="text" placeholder="https://siem.example.com/ingest" value={newDest.url} onChange={e => setNewDest(prev => ({ ...prev, url: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1 font-mono" />
            </div>
            <div>
              <label className="text-sm font-medium">Auth Method</label>
              <select value={newDest.authMethod} onChange={e => setNewDest(prev => ({ ...prev, authMethod: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1">
                {authMethods.map(a => <option key={a} value={a}>{a}</option>)}
              </select>
            </div>
            <div>
              <label className="text-sm font-medium">Batch Size</label>
              <input type="number" min={1} max={1000} value={newDest.batchSize} onChange={e => setNewDest(prev => ({ ...prev, batchSize: parseInt(e.target.value) || 100 }))} className="w-full border rounded px-3 py-2 text-sm mt-1" />
            </div>
          </div>
          <button onClick={addDestination} disabled={!newDest.name || !newDest.url} className="px-4 py-2 bg-blue-600 text-white rounded text-sm disabled:opacity-50">Add Destination</button>
        </section>
      )}

      <div className="grid grid-cols-3 gap-4">
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold">{destinations.length}</div>
          <div className="text-sm text-gray-500">Destinations</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold text-green-600">{destinations.filter(d => d.status === 'active').length}</div>
          <div className="text-sm text-gray-500">Active</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold text-blue-600">{totalEvents.toLocaleString()}</div>
          <div className="text-sm text-gray-500">Events Forwarded</div>
        </div>
      </div>

      <section className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th className="p-3">Name</th>
              <th className="p-3">Type</th>
              <th className="p-3">URL</th>
              <th className="p-3">Auth</th>
              <th className="p-3">Batch</th>
              <th className="p-3">Status</th>
              <th className="p-3">Events</th>
              <th className="p-3">Action</th>
            </tr>
          </thead>
          <tbody>
            {destinations.map(d => (
              <tr key={d.id} className="border-b hover:bg-gray-50">
                <td className="p-3 font-medium">{d.name}</td>
                <td className="p-3"><span className="px-2 py-0.5 bg-blue-100 text-blue-700 rounded text-xs">{d.type}</span></td>
                <td className="p-3 font-mono text-xs text-gray-500 truncate max-w-xs">{d.url}</td>
                <td className="p-3 text-gray-600 text-xs">{d.authMethod}</td>
                <td className="p-3 text-gray-500">{d.batchSize}</td>
                <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs ${statusColor(d.status)}`}>{d.status}</span></td>
                <td className="p-3 text-gray-500">{d.eventsForwarded.toLocaleString()}</td>
                <td className="p-3"><button onClick={() => testConnection(d)} className="text-blue-600 text-xs hover:underline">Test</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <div className="grid grid-cols-2 gap-6">
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Event Filter Rules</h2>
          <div className="space-y-2">
            {filters.map(f => (
              <div key={f.id} className="flex items-center gap-2 text-sm border-b pb-2">
                <span className="font-mono text-xs bg-gray-100 px-2 py-0.5 rounded">{f.field}</span>
                <span className="text-gray-400">{f.operator}</span>
                <span className="font-mono text-xs">{f.value}</span>
              </div>
            ))}
          </div>
          <p className="text-xs text-gray-400">Only events matching all filter rules are forwarded.</p>
        </section>

        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Retry Policy</h2>
          <div className="space-y-3">
            <div>
              <label className="text-sm font-medium">Max Retries</label>
              <input type="number" min={0} max={20} value={retry.maxRetries} onChange={e => setRetry(prev => ({ ...prev, maxRetries: parseInt(e.target.value) || 0 }))} className="w-20 border rounded px-2 py-1 text-sm mt-1" />
            </div>
            <div>
              <label className="text-sm font-medium">Backoff Strategy</label>
              <select value={retry.backoff} onChange={e => setRetry(prev => ({ ...prev, backoff: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1">
                <option value="exponential">Exponential</option>
                <option value="linear">Linear</option>
                <option value="fixed">Fixed interval</option>
              </select>
            </div>
            <div>
              <label className="text-sm font-medium">Circuit Breaker Threshold</label>
              <input type="number" min={1} max={100} value={retry.circuitBreakerThreshold} onChange={e => setRetry(prev => ({ ...prev, circuitBreakerThreshold: parseInt(e.target.value) || 10 }))} className="w-20 border rounded px-2 py-1 text-sm mt-1" />
            </div>
          </div>
        </section>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">TLS Configuration</h2>
        <div className="space-y-3">
          <label className="flex items-center justify-between">
            <span className="text-sm">Enable TLS for all destinations</span>
            <input type="checkbox" checked={tlsEnabled} onChange={e => setTlsEnabled(e.target.checked)} className="rounded" />
          </label>
          <label className="flex items-center justify-between">
            <span className="text-sm">Verify server certificates</span>
            <input type="checkbox" checked={tlsVerify} onChange={e => setTlsVerify(e.target.checked)} className="rounded" />
          </label>
        </div>
      </section>

      {testTarget && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 max-w-md w-full mx-4 space-y-4">
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-semibold">Test: {testTarget.name}</h2>
              <button onClick={() => { setTestTarget(null); setTestResult(''); }} className="text-gray-400 hover:text-gray-600">X</button>
            </div>
            <div className={`p-3 rounded text-sm ${testResult.includes('successful') ? 'bg-green-50 text-green-700' : 'bg-amber-50 text-amber-700'}`}>{testResult}</div>
            {testResult.includes('successful') && (
              <button onClick={() => { setTestTarget(null); setTestResult(''); }} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">Close</button>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
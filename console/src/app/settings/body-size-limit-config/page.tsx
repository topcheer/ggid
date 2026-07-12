'use client';
import { useState } from 'react';

interface RouteOverride {
  route: string;
  maxSize: number;
}

export default function BodySizeLimitConfigPage() {
  const [globalMax, setGlobalMax] = useState(10);
  const [postLimit, setPostLimit] = useState(10);
  const [putLimit, setPutLimit] = useState(10);
  const [patchLimit, setPatchLimit] = useState(5);
  const [multipartLimit, setMultipartLimit] = useState(50);
  const [fileUploadLimit, setFileUploadLimit] = useState(100);
  const [errorMsg, setErrorMsg] = useState('Request body too large. Maximum allowed size is {max} MB.');
  const [bypassList, setBypassList] = useState(['/healthz', '/metrics', '/readyz']);
  const [newBypass, setNewBypass] = useState('');
  const [overrides, setOverrides] = useState<RouteOverride[]>([
    { route: '/api/v1/upload', maxSize: 100 },
    { route: '/api/v1/audit/events', maxSize: 1 },
    { route: '/api/v1/users/bulk', maxSize: 20 },
  ]);
  const [newRoute, setNewRoute] = useState('');
  const [newRouteSize, setNewRouteSize] = useState(10);

  const [stats] = useState({ blocked24h: 15, blocked7d: 89, avgSize: 2.3, maxSize: 45.6 });

  const addBypass = () => { if (newBypass) { setBypassList(prev => [...prev, newBypass]); setNewBypass(''); } };
  const removeBypass = (route: string) => setBypassList(prev => prev.filter(r => r !== route));
  const addOverride = () => { if (newRoute) { setOverrides(prev => [...prev, { route: newRoute, maxSize: newRouteSize }]); setNewRoute(''); setNewRouteSize(10); } };
  const removeOverride = (route: string) => setOverrides(prev => prev.filter(r => r.route !== route));

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Body Size Limit Configuration</h1>
        <p className="text-gray-600">Configure request body size limits, per-route overrides, and monitoring.</p>
      </div>

      <div className="grid grid-cols-4 gap-4">
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-2xl font-bold text-red-600">{stats.blocked24h}</div><div className="text-sm text-gray-500">Blocked (24h)</div></div>
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-2xl font-bold text-amber-600">{stats.blocked7d}</div><div className="text-sm text-gray-500">Blocked (7d)</div></div>
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-2xl font-bold">{stats.avgSize}MB</div><div className="text-sm text-gray-500">Avg Size</div></div>
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-2xl font-bold">{stats.maxSize}MB</div><div className="text-sm text-gray-500">Max Seen</div></div>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Global Limits</h2>
        <div>
          <label className="text-sm font-medium">Global Max Body Size: {globalMax} MB</label>
          <input type="range" min={1} max={500} value={globalMax} onChange={e => setGlobalMax(parseInt(e.target.value))} className="w-full mt-2" />
        </div>
        <div className="grid grid-cols-3 gap-4">
          <div><label className="text-sm font-medium">POST Limit (MB)</label><input type="number" min={1} value={postLimit} onChange={e => setPostLimit(parseInt(e.target.value) || 10)} className="w-full border rounded px-2 py-1 text-sm mt-1" /></div>
          <div><label className="text-sm font-medium">PUT Limit (MB)</label><input type="number" min={1} value={putLimit} onChange={e => setPutLimit(parseInt(e.target.value) || 10)} className="w-full border rounded px-2 py-1 text-sm mt-1" /></div>
          <div><label className="text-sm font-medium">PATCH Limit (MB)</label><input type="number" min={1} value={patchLimit} onChange={e => setPatchLimit(parseInt(e.target.value) || 5)} className="w-full border rounded px-2 py-1 text-sm mt-1" /></div>
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Special Limits</h2>
        <div className="grid grid-cols-2 gap-4">
          <div><label className="text-sm font-medium">Multipart Limit (MB)</label><input type="number" min={1} value={multipartLimit} onChange={e => setMultipartLimit(parseInt(e.target.value) || 50)} className="w-full border rounded px-2 py-1 text-sm mt-1" /></div>
          <div><label className="text-sm font-medium">File Upload Limit (MB)</label><input type="number" min={1} value={fileUploadLimit} onChange={e => setFileUploadLimit(parseInt(e.target.value) || 100)} className="w-full border rounded px-2 py-1 text-sm mt-1" /></div>
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Per-Route Overrides</h2>
        <table className="w-full text-sm">
          <thead className="bg-gray-50"><tr className="text-left"><th className="p-3">Route</th><th className="p-3">Max Size (MB)</th><th className="p-3">Action</th></tr></thead>
          <tbody>
            {overrides.map(o => (
              <tr key={o.route} className="border-b"><td className="p-3 font-mono text-xs">{o.route}</td><td className="p-3">{o.maxSize}</td><td className="p-3"><button onClick={() => removeOverride(o.route)} className="text-red-600 text-xs hover:underline">Remove</button></td></tr>
            ))}
          </tbody>
        </table>
        <div className="flex gap-2">
          <input type="text" placeholder="/api/v1/route" value={newRoute} onChange={e => setNewRoute(e.target.value)} className="flex-1 border rounded px-2 py-1 text-sm font-mono" />
          <input type="number" placeholder="MB" value={newRouteSize} onChange={e => setNewRouteSize(parseInt(e.target.value) || 10)} className="w-20 border rounded px-2 py-1 text-sm" />
          <button onClick={addOverride} className="px-3 py-1 bg-blue-600 text-white rounded text-sm">Add</button>
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Bypass List</h2>
        <p className="text-sm text-gray-500">Routes that skip body size enforcement (health checks, metrics).</p>
        <div className="flex flex-wrap gap-2">
          {bypassList.map(r => (
            <div key={r} className="flex items-center gap-1"><span className="px-2 py-1 bg-gray-100 rounded text-xs font-mono">{r}</span><button onClick={() => removeBypass(r)} className="text-red-600 text-xs">x</button></div>
          ))}
        </div>
        <div className="flex gap-2">
          <input type="text" placeholder="/api/v1/route" value={newBypass} onChange={e => setNewBypass(e.target.value)} className="flex-1 border rounded px-2 py-1 text-sm font-mono" />
          <button onClick={addBypass} className="px-3 py-1 bg-blue-600 text-white rounded text-sm">Add</button>
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Error Response (413)</h2>
        <textarea value={errorMsg} onChange={e => setErrorMsg(e.target.value)} rows={2} className="w-full border rounded px-3 py-2 text-sm" />
        <p className="text-xs text-gray-400">Available variables: {`{max}`}, {`{size}`}, {`{route}`}</p>
      </section>
    </div>
  );
}
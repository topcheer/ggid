'use client';
import { useState } from 'react';

interface ContentType { type: string; enabled: boolean; }

export default function CompressionConfigPage() {
  const [enabled, setEnabled] = useState(true);
  const [algorithms, setAlgorithms] = useState(['gzip', 'brotli']);
  const [minSize, setMinSize] = useState(1024);
  const [level, setLevel] = useState(6);
  const [prefetch, setPrefetch] = useState(true);
  const [contentTypes, setContentTypes] = useState<ContentType[]>([
    { type: 'text/html', enabled: true },
    { type: 'text/css', enabled: true },
    { type: 'application/json', enabled: true },
    { type: 'application/javascript', enabled: true },
    { type: 'application/xml', enabled: true },
    { type: 'text/plain', enabled: true },
    { type: 'image/svg+xml', enabled: true },
  ]);
  const [skipTypes, setSkipTypes] = useState(['image/jpeg', 'image/png', 'video/mp4', 'application/zip']);
  const [newContentType, setNewContentType] = useState('');
  const [newSkipType, setNewSkipType] = useState('');
  const [stats] = useState({ bytesSaved: '4.2 GB', ratio: 68.5, requestsCompressed: 15420 });

  const allAlgorithms = ['gzip', 'brotli', 'zstd'];
  const toggleAlg = (a: string) => setAlgorithms(prev => prev.includes(a) ? prev.filter(x => x !== a) : [...prev, a]);
  const toggleContentType = (idx: number) => setContentTypes(prev => prev.map((c, i) => i === idx ? { ...c, enabled: !c.enabled } : c));
  const addContentType = () => { if (newContentType) { setContentTypes(prev => [...prev, { type: newContentType, enabled: true }]); setNewContentType(''); } };
  const addSkipType = () => { if (newSkipType) { setSkipTypes(prev => [...prev, newSkipType]); setNewSkipType(''); } };
  const removeSkipType = (t: string) => setSkipTypes(prev => prev.filter(x => x !== t));

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Compression Configuration</h1>
        <p className="text-gray-600">Configure response compression algorithms, thresholds, and content type rules.</p>
      </div>

      <div className="grid grid-cols-3 gap-4">
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-2xl font-bold text-green-600">{stats.bytesSaved}</div><div className="text-sm text-gray-500">Bytes Saved (24h)</div></div>
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-2xl font-bold">{stats.ratio}%</div><div className="text-sm text-gray-500">Compression Ratio</div></div>
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-2xl font-bold">{stats.requestsCompressed.toLocaleString()}</div><div className="text-sm text-gray-500">Requests Compressed</div></div>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">General Settings</h2>
        <label className="flex items-center justify-between"><span className="text-sm font-medium">Enable Compression</span><input type="checkbox" checked={enabled} onChange={e => setEnabled(e.target.checked)} className="rounded" /></label>
        <div>
          <label className="text-sm font-medium">Algorithms (in priority order)</label>
          <div className="flex gap-3 mt-2">
            {allAlgorithms.map(a => (
              <label key={a} className="flex items-center gap-1 text-sm"><input type="checkbox" checked={algorithms.includes(a)} onChange={() => toggleAlg(a)} className="rounded" /><span className="font-mono">{a}</span></label>
            ))}
          </div>
        </div>
        <div className="grid grid-cols-2 gap-4">
          <div><label className="text-sm font-medium">Min Size Threshold (bytes)</label><input type="number" min={0} value={minSize} onChange={e => setMinSize(parseInt(e.target.value) || 1024)} className="w-full border rounded px-2 py-1 text-sm mt-1" /></div>
          <div><label className="text-sm font-medium">Compression Level: {level}</label><input type="range" min={1} max={9} value={level} onChange={e => setLevel(parseInt(e.target.value))} className="w-full mt-2" /></div>
        </div>
        <label className="flex items-center justify-between"><span className="text-sm">Prefetch compressed variants</span><input type="checkbox" checked={prefetch} onChange={e => setPrefetch(e.target.checked)} className="rounded" /></label>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Content Types</h2>
        <div className="space-y-2">
          {contentTypes.map((c, idx) => (
            <label key={c.type} className="flex items-center justify-between border-b pb-1"><span className="font-mono text-sm">{c.type}</span><input type="checkbox" checked={c.enabled} onChange={() => toggleContentType(idx)} className="rounded" /></label>
          ))}
        </div>
        <div className="flex gap-2">
          <input type="text" placeholder="text/html" value={newContentType} onChange={e => setNewContentType(e.target.value)} className="flex-1 border rounded px-2 py-1 text-sm font-mono" />
          <button onClick={addContentType} className="px-3 py-1 bg-blue-600 text-white rounded text-sm">Add</button>
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Skip Compression For</h2>
        <p className="text-sm text-gray-500">Content types that should never be compressed (already compressed binary formats).</p>
        <div className="flex flex-wrap gap-2">
          {skipTypes.map(t => (
            <div key={t} className="flex items-center gap-1"><span className="px-2 py-1 bg-gray-100 rounded text-xs font-mono">{t}</span><button onClick={() => removeSkipType(t)} className="text-red-600 text-xs">x</button></div>
          ))}
        </div>
        <div className="flex gap-2">
          <input type="text" placeholder="image/webp" value={newSkipType} onChange={e => setNewSkipType(e.target.value)} className="flex-1 border rounded px-2 py-1 text-sm font-mono" />
          <button onClick={addSkipType} className="px-3 py-1 bg-blue-600 text-white rounded text-sm">Add</button>
        </div>
      </section>
    </div>
  );
}
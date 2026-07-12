'use client';
import { useState, useEffect } from 'react';

interface ClientDpop {
  clientId: string;
  enforce: boolean;
}

export default function DpopConfigPage() {
  const [enabled, setEnabled] = useState(true);
  const [proofExpiry, setProofExpiry] = useState(60);
  const [nonceTtl, setNonceTtl] = useState(600);
  const [algorithms, setAlgorithms] = useState(['ES256', 'RS256', 'EdDSA']);
  const [requireSensitive, setRequireSensitive] = useState(true);
  const [clients, setClients] = useState<ClientDpop[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [debugProof, setDebugProof] = useState('');
  const [debugResult, setDebugResult] = useState('');

  useEffect(() => {
    fetch('/api/v1/auth/token-reuse-check', {
      headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': '00000000-0000-0000-0000-000000000001' },
    })
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(data => {
        if (data) {
          if (data.enabled !== undefined) setEnabled(data.enabled);
          if (data.proof_expiry) setProofExpiry(data.proof_expiry);
          if (data.nonce_ttl) setNonceTtl(data.nonce_ttl);
          if (data.algorithms) setAlgorithms(data.algorithms);
          if (data.require_sensitive !== undefined) setRequireSensitive(data.require_sensitive);
          if (data.clients) setClients(data.clients);
        }
        setLoading(false);
      })
      .catch(e => { setError(e.message); setLoading(false); });
  }, []);

  const allAlgorithms = ['ES256', 'RS256', 'EdDSA', 'ES384', 'ES512'];

  const toggleAlgorithm = (alg: string) => {
    setAlgorithms(prev => prev.includes(alg) ? prev.filter(a => a !== alg) : [...prev, alg]);
  };

  const toggleClient = (idx: number) => {
    setClients(prev => prev.map((c, i) => i === idx ? { ...c, enforce: !c.enforce } : c));
  };

  const validateProof = () => {
    if (!debugProof.trim()) { setDebugResult('Paste a DPoP proof JWT to validate'); return; }
    try {
      const parts = debugProof.split('.');
      if (parts.length !== 3) { setDebugResult('Invalid JWT format: expected 3 parts'); return; }
      const header = JSON.parse(atob(parts[0]));
      const payload = JSON.parse(atob(parts[1]));
      setDebugResult(`Valid JWT\nHeader: ${JSON.stringify(header, null, 2)}\nPayload: ${JSON.stringify(payload, null, 2)}`);
    } catch {
      setDebugResult('Failed to parse JWT');
    }
  };

  if (loading) return <div className="p-6"><p>Loading...</p></div>;
  if (error) return <div className="p-6 text-red-600">Error: {error}</div>;

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">DPoP Configuration</h1>
        <p className="text-gray-600">Demonstration of Proof of Possession settings for sender-constrained tokens.</p>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">General Settings</h2>
        <label className="flex items-center justify-between">
          <span className="text-sm font-medium">Enable DPoP</span>
          <input type="checkbox" checked={enabled} onChange={e => setEnabled(e.target.checked)} className="rounded" />
        </label>
        <div>
          <label className="text-sm font-medium">Proof Token Expiry: {proofExpiry}s</label>
          <input type="range" min={30} max={120} value={proofExpiry} onChange={e => setProofExpiry(parseInt(e.target.value))} className="w-full mt-2" />
          <div className="flex justify-between text-xs text-gray-400"><span>30s</span><span>120s</span></div>
        </div>
        <div>
          <label className="text-sm font-medium">Nonce Cache TTL (replay prevention)</label>
          <input type="number" min={60} max={3600} value={nonceTtl} onChange={e => setNonceTtl(parseInt(e.target.value) || 600)} className="w-24 border rounded px-2 py-1 text-sm mt-1" />
          <p className="text-xs text-gray-500 mt-1">DPoP nonces are cached for {nonceTtl}s to prevent replay attacks.</p>
        </div>
        <label className="flex items-center justify-between">
          <span className="text-sm">Require DPoP for Sensitive Scopes</span>
          <input type="checkbox" checked={requireSensitive} onChange={e => setRequireSensitive(e.target.checked)} className="rounded" />
        </label>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Supported Algorithms</h2>
        <div className="flex flex-wrap gap-3">
          {allAlgorithms.map(alg => (
            <label key={alg} className="flex items-center gap-1 text-sm">
              <input type="checkbox" checked={algorithms.includes(alg)} onChange={() => toggleAlgorithm(alg)} className="rounded" />
              <span className="font-mono">{alg}</span>
            </label>
          ))}
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Per-Client DPoP Enforcement</h2>
        <div className="space-y-2">
          {clients.map((c, idx) => (
            <label key={c.clientId} className="flex items-center justify-between border-b pb-2">
              <span className="font-mono text-sm">{c.clientId}</span>
              <input type="checkbox" checked={c.enforce} onChange={() => toggleClient(idx)} className="rounded" />
            </label>
          ))}
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Proof Validation Debug Viewer</h2>
        <textarea value={debugProof} onChange={e => setDebugProof(e.target.value)} rows={3} placeholder="Paste DPoP proof JWT..." className="w-full border rounded px-3 py-2 text-sm font-mono" />
        <button onClick={validateProof} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">Validate</button>
        {debugResult && <pre className="bg-gray-900 text-green-400 rounded p-3 text-xs overflow-x-auto max-h-48">{debugResult}</pre>}
      </section>
    </div>
  );
}

'use client';
import { useState, useEffect } from 'react';

export default function EncryptionConfigPage() {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [algorithm, setAlgorithm] = useState('AES-256-GCM');
  const [keyRotation, setKeyRotation] = useState(90);
  const [kmsProvider, setKmsProvider] = useState('internal');
  const [envelopeEncryption, setEnvelopeEncryption] = useState(true);
  const [tlsMinVersion, setTlsMinVersion] = useState('1.2');
  const [cipherSuites, setCipherSuites] = useState<string[]>([]);
  const [hsmEnabled, setHsmEnabled] = useState(false);
  const [keyBackup, setKeyBackup] = useState(true);

  useEffect(() => {
    fetch("/api/v1/identity/encryption-config", {
      headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
    })
      .then(res => { if (!res.ok) throw new Error(`HTTP ${res.status}`); return res.json(); })
      .then(data => {
        if (data.algorithm) setAlgorithm(data.algorithm);
        if (data.keyRotation) setKeyRotation(data.keyRotation);
        if (data.kmsProvider) setKmsProvider(data.kmsProvider);
        if (data.envelopeEncryption !== undefined) setEnvelopeEncryption(data.envelopeEncryption);
        if (data.tlsMinVersion) setTlsMinVersion(data.tlsMinVersion);
        if (data.cipherSuites) setCipherSuites(data.cipherSuites);
        if (data.hsmEnabled !== undefined) setHsmEnabled(data.hsmEnabled);
        if (data.keyBackup !== undefined) setKeyBackup(data.keyBackup);
        setLoading(false);
      })
      .catch(err => { setError(err.message); setLoading(false); });
  }, []);

  const allCiphers = ['TLS_AES_256_GCM_SHA384', 'TLS_CHACHA20_POLY1305_SHA256', 'TLS_AES_128_GCM_SHA256', 'ECDHE-RSA-AES256-GCM-SHA384', 'ECDHE-ECDSA-AES256-GCM-SHA384'];
  const toggleCipher = (c: string) => setCipherSuites(prev => prev.includes(c) ? prev.filter(x => x !== c) : [...prev, c]);

  if (loading) return <div className="p-6"><h1 className="text-2xl font-bold">Encryption Configuration</h1><p className="text-gray-600 mt-2">Loading...</p></div>;
  if (error) return <div className="p-6"><h1 className="text-2xl font-bold">Encryption Configuration</h1><p className="text-red-600 mt-2">Error: {error}</p></div>;

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      <div><h1 className="text-2xl font-bold">Encryption Configuration</h1><p className="text-gray-600">Configure data encryption, key management, and TLS settings.</p></div>

      <div className="grid grid-cols-4 gap-4">
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-sm font-bold text-green-600">Active</div><div className="text-xs text-gray-500">Encryption Status</div></div>
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-sm font-bold">{algorithm.split('-')[0]}</div><div className="text-xs text-gray-500">Algorithm</div></div>
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-sm font-bold">{keyRotation}d</div><div className="text-xs text-gray-500">Next Rotation</div></div>
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-sm font-bold">TLS {tlsMinVersion}</div><div className="text-xs text-gray-500">Min TLS</div></div>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Data Encryption</h2>
        <div><label className="text-sm font-medium">Algorithm</label><select value={algorithm} onChange={e => setAlgorithm(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1"><option value="AES-256-GCM">AES-256-GCM</option><option value="ChaCha20-Poly1305">ChaCha20-Poly1305</option></select></div>
        <div><label className="text-sm font-medium">Key Rotation Interval (days)</label><input type="number" min={1} max={365} value={keyRotation} onChange={e => setKeyRotation(parseInt(e.target.value) || 90)} className="w-24 border rounded px-2 py-1 text-sm mt-1" /></div>
        <div><label className="text-sm font-medium">KMS Provider</label><select value={kmsProvider} onChange={e => setKmsProvider(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1"><option value="internal">Internal (built-in)</option><option value="aws-kms">AWS KMS</option><option value="gcp-kms">Google Cloud KMS</option><option value="azure-kv">Azure Key Vault</option><option value="hashicorp-vault">HashiCorp Vault</option></select></div>
        <label className="flex items-center justify-between"><span className="text-sm">Envelope Encryption</span><input type="checkbox" checked={envelopeEncryption} onChange={e => setEnvelopeEncryption(e.target.checked)} className="rounded" /></label>
        <label className="flex items-center justify-between"><span className="text-sm">HSM Integration</span><input type="checkbox" checked={hsmEnabled} onChange={e => setHsmEnabled(e.target.checked)} className="rounded" /></label>
        <label className="flex items-center justify-between"><span className="text-sm">Key Backup</span><input type="checkbox" checked={keyBackup} onChange={e => setKeyBackup(e.target.checked)} className="rounded" /></label>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">TLS Configuration</h2>
        <div><label className="text-sm font-medium">Minimum TLS Version</label><select value={tlsMinVersion} onChange={e => setTlsMinVersion(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1"><option value="1.0">TLS 1.0 (deprecated)</option><option value="1.1">TLS 1.1 (deprecated)</option><option value="1.2">TLS 1.2</option><option value="1.3">TLS 1.3 (recommended)</option></select></div>
        <div><label className="text-sm font-medium">Cipher Suites</label><div className="space-y-2 mt-2">{allCiphers.map(c => <label key={c} className="flex items-center gap-2 text-sm"><input type="checkbox" checked={cipherSuites.includes(c)} onChange={() => toggleCipher(c)} className="rounded" /><span className="font-mono text-xs">{c}</span></label>)}</div></div>
      </section>
    </div>
  );
}

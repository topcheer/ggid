'use client';
import { useTranslations } from "@/lib/i18n";
import { useState, useEffect } from 'react';

interface ClientOverride {
  clientId: string;
  bindingType: string;
  enforcement: string;
}

export default function TokenBindingConfigPage() {
  const t = useTranslations();
  const [dpopEnabled, setDpopEnabled] = useState(true);
  const [proofExpiry, setProofExpiry] = useState(300);
  const [senderConstrained, setSenderConstrained] = useState(true);
  const [mtlsBinding, setMtlsBinding] = useState(false);
  const [replayDetection, setReplayDetection] = useState(true);
  const [enforcementPolicy, setEnforcementPolicy] = useState('strict');
  const [overrides, setOverrides] = useState<ClientOverride[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch('/api/v1/auth/token-reuse-check', {
      headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': '00000000-0000-0000-0000-000000000001' },
    })
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(data => {
        if (data) {
          if (data.dpop_enabled !== undefined) setDpopEnabled(data.dpop_enabled);
          if (data.proof_expiry) setProofExpiry(data.proof_expiry);
          if (data.sender_constrained !== undefined) setSenderConstrained(data.sender_constrained);
          if (data.mtls_binding !== undefined) setMtlsBinding(data.mtls_binding);
          if (data.replay_detection !== undefined) setReplayDetection(data.replay_detection);
          if (data.enforcement_policy) setEnforcementPolicy(data.enforcement_policy);
          if (data.overrides) setOverrides(data.overrides);
        }
        setLoading(false);
      })
      .catch(e => { setError(e.message); setLoading(false); });
  }, []);

  const addOverride = () => {
    setOverrides(prev => [...prev, { clientId: '', bindingType: 'dpop', enforcement: 'optional' }]);
  };

  const updateOverride = (idx: number, field: keyof ClientOverride, value: string) => {
    setOverrides(prev => prev.map((o, i) => i === idx ? { ...o, [field]: value } : o));
  };

  const removeOverride = (idx: number) => {
    setOverrides(prev => prev.filter((_, i) => i !== idx));
  };

  if (loading) return <div className="p-6"><p>Loading...</p></div>;
  if (error) return <div className="p-6 text-red-600">Error: {error}</div>;

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      <h1 className="text-2xl font-bold">{t("backend.tokenBindingConfig.title")}</h1>
      <p className="text-gray-600">Configure sender-constrained tokens via DPoP, mTLS binding, and replay detection.</p>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{t("backend.tokenBindingConfig.bindingSettings")}</h2>
        <label className="flex items-center justify-between">
          <span className="text-sm">{t("backend.tokenBindingConfig.dpop")}</span>
          <input type="checkbox" checked={dpopEnabled} onChange={e => setDpopEnabled(e.target.checked)} className="rounded" />
        </label>
        <label className="flex items-center justify-between">
          <span className="text-sm">Sender-Constrained Tokens</span>
          <input type="checkbox" checked={senderConstrained} onChange={e => setSenderConstrained(e.target.checked)} className="rounded" />
        </label>
        <label className="flex items-center justify-between">
          <span className="text-sm">mTLS Token Binding</span>
          <input type="checkbox" checked={mtlsBinding} onChange={e => setMtlsBinding(e.target.checked)} className="rounded" />
        </label>
        <label className="flex items-center justify-between">
          <span className="text-sm">Token Replay Detection</span>
          <input type="checkbox" checked={replayDetection} onChange={e => setReplayDetection(e.target.checked)} className="rounded" />
        </label>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{t("backend.tokenBindingConfig.proofTokenExpiry")}</h2>
        <div className="flex items-center gap-3">
          <input
            type="number"
            min={30}
            max={3600}
            value={proofExpiry}
            onChange={e => setProofExpiry(parseInt(e.target.value) || 300)}
            className="w-24 border rounded px-2 py-1 text-sm"
          />
          <span className="text-sm text-gray-500">seconds</span>
        </div>
        <p className="text-xs text-gray-400">Maximum lifetime of a DPoP proof JWT. Shorter values increase security but require more frequent proof generation.</p>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{t("backend.tokenBindingConfig.title")}</h2>
        <select
          value={enforcementPolicy}
          onChange={e => setEnforcementPolicy(e.target.value)}
          className="border rounded px-3 py-2 text-sm w-full"
        >
          <option value="strict">Strict - Reject tokens without valid binding proof</option>
          <option value="permissive">Permissive - Log warnings but allow unbound tokens</option>
          <option value="audit">Audit Only - Log and monitor, no enforcement</option>
        </select>
        <p className="text-xs text-gray-400">Strict mode rejects any access token that lacks a valid binding proof. Permissive mode logs but allows.</p>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">Per-Client Overrides</h2>
          <button onClick={addOverride} className="px-3 py-1 bg-blue-600 text-white rounded text-sm">{t("backend.tokenBindingConfig.addOverride")}</button>
        </div>
        <div className="space-y-3">
          {overrides.map((o, idx) => (
            <div key={idx} className="flex items-center gap-3">
              <input
                type="text"
                placeholder="Client ID"
                value={o.clientId}
                onChange={e => updateOverride(idx, 'clientId', e.target.value)}
                className="flex-1 border rounded px-2 py-1 text-sm"
              />
              <select
                value={o.bindingType}
                onChange={e => updateOverride(idx, 'bindingType', e.target.value)}
                className="border rounded px-2 py-1 text-sm"
              >
                <option value="dpop">{t("backend.tokenBindingConfig.dpop")}</option>
                <option value="mtls">mTLS</option>
                <option value="none">{t("backend.tokenBindingConfig.none")}</option>
              </select>
              <select
                value={o.enforcement}
                onChange={e => updateOverride(idx, 'enforcement', e.target.value)}
                className="border rounded px-2 py-1 text-sm"
              >
                <option value="required">{t("backend.tokenBindingConfig.required")}</option>
                <option value="optional">{t("backend.tokenBindingConfig.optional")}</option>
                <option value="disabled">{t("backend.tokenBindingConfig.disabled")}</option>
              </select>
              <button onClick={() => removeOverride(idx)} className="text-red-600 text-sm">{t("backend.tokenBindingConfig.remove")}</button>
            </div>
          ))}
          {overrides.length === 0 && <p className="text-sm text-gray-400">No overrides configured. Global policy applies to all clients.</p>}
        </div>
      </section>

      <div className="flex justify-end gap-3">
        <button className="px-4 py-2 border rounded text-sm">Reset</button>
        <button className="px-4 py-2 bg-blue-600 text-white rounded text-sm">Save Configuration</button>
      </div>
    </div>
  );
}

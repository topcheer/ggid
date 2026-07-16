'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";

interface BindingConfig {
  clientId: string;
  dpop: string;
  mtls: string;
  senderConstrained: string;
}

interface Thumbprint {
  id: string;
  certName: string;
  thumbprint: string;
  added: string;
}

export default function TokenBindingStrategiesPage() {
  const [configs, setConfigs] = useState<BindingConfig[]>([]);

  const [proofLifetime, setProofLifetime] = useState(300);
  const [replayDetection, setReplayDetection] = useState(true);
  const [replayWindow, setReplayWindow] = useState(60);
  const [thumbprints, setThumbprints] = useState<Thumbprint[]>([]);
  const [showAddThumb, setShowAddThumb] = useState(false);
  const [newThumb, setNewThumb] = useState({ certName: '', thumbprint: '' });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const t = useTranslations();

  useEffect(() => {
    fetch('/api/v1/auth/sessions/anomaly-score', {
      headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': '00000000-0000-0000-0000-000000000001' },
    })
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(data => {
        if (data) {
          if (data.configs) setConfigs(data.configs);
          if (data.proof_lifetime) setProofLifetime(data.proof_lifetime);
          if (data.replay_detection !== undefined) setReplayDetection(data.replay_detection);
          if (data.replay_window) setReplayWindow(data.replay_window);
          if (data.thumbprints) setThumbprints(data.thumbprints);
        }
        setLoading(false);
      })
      .catch(e => { setError(e.message); setLoading(false); });
  }, []);

  const methods = [
    { name: 'DPoP', desc: 'Demonstration of Proof of Possession', pros: ['No cert management', 'Works with any transport', 'RFC 9449 standard'], cons: ['Higher overhead per request', 'Requires JS crypto in browser'] },
    { name: 'mTLS', desc: 'Mutual TLS certificate binding', pros: ['Hardware-backed security', 'Enterprise-grade', 'No app changes needed'], cons: ['Cert distribution complexity', 'Mobile platform limitations'] },
    { name: 'Sender-Constrained', desc: 'Token bound to sender via proof', pros: ['Token replay prevention', 'Standards-based'], cons: ['Requires client cooperation', 'Additional round trips'] },
  ];

  const updateConfig = (idx: number, field: keyof BindingConfig, value: string) => {
    setConfigs(prev => prev.map((c, i) => i === idx ? { ...c, [field]: value } : c));
  };

  const addThumbprint = () => {
    setThumbprints(prev => [...prev, { id: `t${prev.length + 1}`, certName: newThumb.certName, thumbprint: newThumb.thumbprint, added: new Date().toISOString().slice(0, 10) }]);
    setShowAddThumb(false);
    setNewThumb({ certName: '', thumbprint: '' });
  };

  const removeThumbprint = (id: string) => {
    setThumbprints(prev => prev.filter(t => t.id !== id));
  };

  const enforcementColor = (v: string): string =>
    v === 'required' ? 'bg-red-50 text-red-700' :
    v === 'optional' ? 'bg-amber-50 text-amber-700' :
    'bg-gray-100 text-gray-500';

  if (loading) return <div className="p-6"><p>{t("tokenBindingStrategies.loading")}</p></div>;
  if (error) return <div className="p-6 text-red-600">{t("common.error")}: {error}</div>;

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{t("tokenBindingStrategies.title")}</h1>
        <p className="text-gray-600">{t("tokenBindingStrategies.subtitle")}</p>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{t("tokenBindingStrategies.bindingMethods")}</h2>
        <div className="grid grid-cols-3 gap-4">
          {methods.map(m => (
            <div key={m.name} className="border rounded p-4">
              <div className="font-medium text-sm">{m.name}</div>
              <div className="text-xs text-gray-500 mt-1">{m.desc}</div>
              <div className="mt-2 text-xs">
                <div className="font-medium text-green-600">{t("tokenBindingStrategies.pros")}</div>
                <ul className="list-disc list-inside text-gray-600">{m.pros.map(p => <li key={p}>{p}</li>)}</ul>
              </div>
              <div className="mt-2 text-xs">
                <div className="font-medium text-red-600">{t("tokenBindingStrategies.cons")}</div>
                <ul className="list-disc list-inside text-gray-600">{m.cons.map(c => <li key={c}>{c}</li>)}</ul>
              </div>
            </div>
          ))}
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{t("tokenBindingStrategies.bindingEnforcement")}</h2>
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th scope="col" className="p-3">{t("tokenBindingStrategies.client")}</th>
              <th scope="col" className="p-3">{t("tokenBindingStrategies.dpop")}</th>
              <th scope="col" className="p-3">{t("tokenBindingStrategies.mtls")}</th>
              <th scope="col" className="p-3">{t("tokenBindingStrategies.senderConstrained")}</th>
            </tr>
          </thead>
          <tbody>
            {configs.map((c, idx) => (
              <tr key={c.clientId} className="border-b">
                <td className="p-3 font-mono text-xs">{c.clientId}</td>
                {(['dpop', 'mtls', 'senderConstrained'] as const).map(field => (
                  <td key={field} className="p-3">
                    <select
                      value={c[field]}
                      onChange={e => updateConfig(idx, field, e.target.value)}
                      className={`border rounded px-2 py-1 text-xs ${enforcementColor(c[field])}`}
                    >
                      <option value="required">{t("tokenBindingStrategies.required")}</option>
                      <option value="optional">{t("tokenBindingStrategies.optional")}</option>
                      <option value="disabled">{t("tokenBindingStrategies.disabled")}</option>
                    </select>
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <div className="grid grid-cols-2 gap-6">
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">{t("tokenBindingStrategies.proofTokenSettings")}</h2>
          <div>
            <label className="text-sm font-medium">{t("tokenBindingStrategies.proofTokenLifetime")}</label>
            <input aria-label="proof Lifetime" type="number" min={30} max={3600} value={proofLifetime} onChange={e => setProofLifetime(parseInt(e.target.value) || 300)} className="w-24 border rounded px-2 py-1 text-sm mt-1" />
          </div>
          <label className="flex items-center justify-between">
            <span className="text-sm">Token Replay Detection</span>
            <input aria-label="Replay detection" type="checkbox" checked={replayDetection} onChange={e => setReplayDetection(e.target.checked)} className="rounded" />
          </label>
          {replayDetection && (
            <div>
              <label className="text-sm font-medium">{t("tokenBindingStrategies.replayWindow")}</label>
              <input aria-label="replay Window" type="number" min={10} max={300} value={replayWindow} onChange={e => setReplayWindow(parseInt(e.target.value) || 60)} className="w-24 border rounded px-2 py-1 text-sm mt-1" />
              <p className="text-xs text-gray-400 mt-1">Tokens presented within this window after first use are rejected as replays.</p>
            </div>
          )}
        </section>

        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-semibold">{t("tokenBindingStrategies.certThumbprint")}</h2>
            <button onClick={() => setShowAddThumb(!showAddThumb)} className="px-3 py-1 bg-blue-600 text-white rounded text-sm">
              {showAddThumb ? t("common.cancel") : t("oidcClaimMapping.add")}
            </button>
          </div>
          {showAddThumb && (
            <div className="space-y-2 border rounded p-3">
              <input aria-label="Certificate name" type="text" placeholder="Certificate name" value={newThumb.certName} onChange={e => setNewThumb(prev => ({ ...prev, certName: e.target.value }))} className="w-full border rounded px-2 py-1 text-sm" />
              <input aria-label="SHA-256 thumbprint (hex:colon)" type="text" placeholder="SHA-256 thumbprint (hex:colon)" value={newThumb.thumbprint} onChange={e => setNewThumb(prev => ({ ...prev, thumbprint: e.target.value }))} className="w-full border rounded px-2 py-1 text-sm font-mono" />
              <button onClick={addThumbprint} disabled={!newThumb.certName || !newThumb.thumbprint} className="px-3 py-1 bg-blue-600 text-white rounded text-sm disabled:opacity-50">{t("tokenBindingStrategies.addToAllowlist")}</button>
            </div>
          )}
          <div className="space-y-2">
            {thumbprints.map(th => (
              <div key={th.id} className="flex items-center gap-2 border-b pb-1">
                <span className="text-sm font-medium flex-1">{th.certName}</span>
                <span className="font-mono text-xs text-gray-500">{th.thumbprint}</span>
                <button onClick={() => removeThumbprint(th.id)} className="text-red-600 text-xs">{t("tokenBindingStrategies.remove")}</button>
              </div>
            ))}
          </div>
        </section>
      </div>
    </div>
  );
}

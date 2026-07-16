'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";

interface RiskWeight {
  factor: string;
  weight: number;
  enabled: boolean;
}

interface RiskAction {
  threshold: number;
  action: string;
  label: string;
}

export default function RiskEngineConfigPage() {
  const t = useTranslations();

  const [weights, setWeights] = useState<RiskWeight[]>([]);
  const [actions, setActions] = useState<RiskAction[]>([]);

  const [ipReputation, setIpReputation] = useState(true);
  const [deviceFingerprint, setDeviceFingerprint] = useState(true);
  const [realTimeEval, setRealTimeEval] = useState(true);
  const [riskThreshold, setRiskThreshold] = useState(50);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch('/api/v1/auth/risk-scoring/config', {
      headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, 'Content-Type': 'application/json', 'X-Tenant-ID': '00000000-0000-0000-0000-000000000001' },
    })
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(data => {
        if (data) {
          if (data.weights) setWeights(data.weights);
          if (data.actions) setActions(data.actions);
          if (data.ip_reputation !== undefined) setIpReputation(data.ip_reputation);
          if (data.device_fingerprint !== undefined) setDeviceFingerprint(data.device_fingerprint);
          if (data.real_time_eval !== undefined) setRealTimeEval(data.real_time_eval);
          if (data.risk_threshold) setRiskThreshold(data.risk_threshold);
        }
        setLoading(false);
      })
      .catch(e => { setError(e.message); setLoading(false); });
  }, []);

  const totalWeight = weights.filter(w => w.enabled).reduce((sum, w) => sum + w.weight, 0);

  const handleWeightChange = (idx: number, value: number) => {
    setWeights(prev => prev.map((w, i) => i === idx ? { ...w, weight: value } : w));
  };

  const handleToggle = (idx: number) => {
    setWeights(prev => prev.map((w, i) => i === idx ? { ...w, enabled: !w.enabled } : w));
  };

  const handleActionChange = (idx: number, field: 'threshold' | 'action' | 'label', value: string | number) => {
    setActions(prev => prev.map((a, i) => i === idx ? { ...a, [field]: value } : a));
  };

  if (loading) return <div className="p-6"><p>Loading...</p></div>;
  if (error) return <div className="p-6 text-red-600">Error: {error}</div>;

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      <h1 className="text-2xl font-bold">Risk Engine Configuration</h1>
      <p className="text-gray-600">Configure risk scoring weights, thresholds, and automated response actions.</p>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Risk Score Weights</h2>
        <div className="text-sm text-gray-500">Total Weight: {totalWeight}/100 {totalWeight !== 100 && <span className="text-amber-600">(should sum to 100)</span>}</div>
        <div className="space-y-3">
          {weights.map((w, idx) => (
            <div key={w.factor} className="flex items-center gap-4">
              <label className="flex items-center gap-2 w-48">
                <input aria-label="W" type="checkbox" checked={w.enabled} onChange={() => handleToggle(idx)} className="rounded" />
                <span className="text-sm">{w.factor}</span>
              </label>
              <input
                type="number"
                min={0}
                max={100}
                value={w.weight}
                onChange={e => handleWeightChange(idx, parseInt(e.target.value) || 0)}
                className="w-20 border rounded px-2 py-1 text-sm"
              />
              <span className="text-sm text-gray-400">%</span>
            </div>
          ))}
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Risk Threshold</h2>
        <div className="flex items-center gap-4">
          <input
            type="range"
            min={0}
            max={100}
            value={riskThreshold}
            onChange={e => setRiskThreshold(parseInt(e.target.value))}
            className="flex-1"
          />
          <span className="text-lg font-mono w-12 text-right">{riskThreshold}</span>
        </div>
        <p className="text-sm text-gray-500">Sessions scoring above this threshold trigger risk actions.</p>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Action Mapping</h2>
        <div className="space-y-3">
          {actions.map((a, idx) => (
            <div key={idx} className="flex items-center gap-4">
              <span className="text-sm w-12">{'>'}=</span>
              <input
                type="number"
                min={0}
                max={100}
                value={a.threshold}
                onChange={e => handleActionChange(idx, 'threshold', parseInt(e.target.value) || 0)}
                className="w-20 border rounded px-2 py-1 text-sm"
              />
              <select
                value={a.action}
                onChange={e => handleActionChange(idx, 'action', e.target.value)}
                className="border rounded px-2 py-1 text-sm"
              >
                <option value="allow">Allow</option>
                <option value="step_up">Step-up Auth</option>
                <option value="challenge_mfa">Challenge MFA</option>
                <option value="block">Block</option>
              </select>
              <span className="text-sm text-gray-500">{a.label}</span>
            </div>
          ))}
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Detection Modules</h2>
        <div className="space-y-3">
          <label className="flex items-center justify-between">
            <span className="text-sm">IP Reputation Lookup</span>
            <input aria-label="Ip reputation" type="checkbox" checked={ipReputation} onChange={e => setIpReputation(e.target.checked)} className="rounded" />
          </label>
          <label className="flex items-center justify-between">
            <span className="text-sm">Device Fingerprint Analysis</span>
            <input aria-label="Device fingerprint" type="checkbox" checked={deviceFingerprint} onChange={e => setDeviceFingerprint(e.target.checked)} className="rounded" />
          </label>
          <label className="flex items-center justify-between">
            <span className="text-sm">Real-time Risk Evaluation</span>
            <input aria-label="Real time eval" type="checkbox" checked={realTimeEval} onChange={e => setRealTimeEval(e.target.checked)} className="rounded" />
          </label>
        </div>
        <p className="text-xs text-gray-400">When real-time evaluation is disabled, risk scores are computed asynchronously after login.</p>
      </section>

      <div className="flex justify-end gap-3">
        <button className="px-4 py-2 border rounded text-sm">Reset</button>
        <button className="px-4 py-2 bg-blue-600 text-white rounded text-sm">Save Configuration</button>
      </div>
    </div>
  );
}

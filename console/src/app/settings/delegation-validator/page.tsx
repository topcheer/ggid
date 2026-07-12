'use client';

import { useState, useCallback, useEffect } from 'react';

interface DelegationRule {
  id: string;
  name: string;
  description: string;
  enabled: boolean;
}

interface ValidationHistoryEntry {
  id: string;
  timestamp: string;
  delegator: string;
  delegatee: string;
  scopes: string[];
  valid: boolean;
  reasons: string[];
}

const INITIAL_RULES: DelegationRule[] = [];

const HISTORY: ValidationHistoryEntry[] = [];

export default function DelegationValidatorPage() {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [rules, setRules] = useState<DelegationRule[]>(INITIAL_RULES);
  const [delegator, setDelegator] = useState('alice@corp.com');
  const [delegatee, setDelegatee] = useState('bob@corp.com');
  const [scopes, setScopes] = useState('audit:read, audit:export');
  const [maxDepth, setMaxDepth] = useState(3);
  const [result, setResult] = useState<{ valid: boolean; reasons: string[] } | null>(null);
  const [validating, setValidating] = useState(false);

  useEffect(() => {
    fetch("/api/v1/policies/delegations", {
      headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
    })
      .then(res => { if (!res.ok) throw new Error(`HTTP ${res.status}`); return res.json(); })
      .then(data => { setRules(data.rules || []); setLoading(false); })
      .catch(err => { setError(err.message); setLoading(false); });
  }, []);

  const toggleRule = useCallback((id: string) => {
    setRules(rules.map(r => r.id === id ? { ...r, enabled: !r.enabled } : r));
  }, [rules]);

  const validate = useCallback(() => {
    setValidating(true);
    fetch("/api/v1/policy/delegation/validate", {
      method: "POST",
      headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
      body: JSON.stringify({ delegator, delegatee, scopes: scopes.split(',').map(s => s.trim()).filter(Boolean), maxDepth }),
    })
      .then(res => { if (!res.ok) throw new Error(`HTTP ${res.status}`); return res.json(); })
      .then(data => { setResult(data); setValidating(false); })
      .catch(err => { setError(err.message); setValidating(false); });
  }, [delegator, delegatee, scopes, maxDepth]);

  if (loading) return (<div className="p-6"><h1 className="text-2xl font-bold mb-4">Delegation Validator</h1><p>Loading...</p></div>);
  if (error) return (<div className="p-6"><h1 className="text-2xl font-bold mb-4">Delegation Validator</h1><p className="text-red-600">Error: {error}</p></div>);
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Delegation Validator</h1>
        <p className="mt-1 text-sm text-gray-500">Validate delegation chains against rules including self-delegation, circular prevention, and scope narrowing.</p>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <div className="space-y-4">
          <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
            <h3 className="text-sm font-medium text-gray-700">Validate Delegation</h3>
            <div className="mt-4 space-y-3">
              <div>
                <label className="block text-xs font-medium text-gray-600">Delegator</label>
                <input value={delegator} onChange={e => setDelegator(e.target.value)} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm font-mono" />
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-600">Delegatee</label>
                <input value={delegatee} onChange={e => setDelegatee(e.target.value)} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm font-mono" />
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-600">Scopes (comma-separated)</label>
                <input value={scopes} onChange={e => setScopes(e.target.value)} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm font-mono" />
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-600">Max Depth: {maxDepth}</label>
                <input type="range" min={1} max={10} value={maxDepth} onChange={e => setMaxDepth(Number(e.target.value))} className="mt-2 w-full" />
              </div>
              <button
                onClick={validate}
                disabled={validating}
                className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
              >
                {validating ? 'Validating...' : 'Validate Delegation'}
              </button>
            </div>
          </div>

          {result && (
            <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
              <h3 className="text-sm font-medium text-gray-700">Validation Result</h3>
              <div className="mt-3">
                <span className={`text-2xl font-bold ${result.valid ? 'text-green-600' : 'text-red-600'}`}>
                  {result.valid ? 'VALID' : 'INVALID'}
                </span>
                {result.reasons.length > 0 && (
                  <div className="mt-3 space-y-1">
                    {result.reasons.map((reason, i) => (
                      <div key={i} className="flex items-start gap-2 text-sm">
                        <span className="text-red-500 mt-0.5">{'!'}</span>
                        <span className="text-gray-700">{reason}</span>
                      </div>
                    ))}
                  </div>
                )}
                {result.valid && (
                  <p className="mt-2 text-sm text-green-600">Delegation chain is valid. All rules passed.</p>
                )}
              </div>
            </div>
          )}
        </div>

        <div className="space-y-4">
          <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
            <h3 className="text-sm font-medium text-gray-700">Delegation Rules</h3>
            <div className="mt-4 space-y-3">
              {rules.map(r => (
                <div key={r.id} className="flex items-center justify-between border-b border-gray-100 pb-3">
                  <div>
                    <span className="text-sm font-medium text-gray-700">{r.name}</span>
                    <p className="text-xs text-gray-400">{r.description}</p>
                  </div>
                  <button onClick={() => toggleRule(r.id)} className={`relative inline-flex h-5 w-9 items-center rounded-full transition ${r.enabled ? 'bg-green-500' : 'bg-gray-200'}`}>
                    <span className={`inline-block h-3 w-3 transform rounded-full bg-white transition ${r.enabled ? 'translate-x-5' : 'translate-x-1'}`} />
                  </button>
                </div>
              ))}
            </div>
          </div>

          <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
            <h3 className="text-sm font-medium text-gray-700">Validation History</h3>
            <div className="mt-3 space-y-2">
              {HISTORY.map(h => (
                <div key={h.id} className="border-b border-gray-100 pb-2 text-sm">
                  <div className="flex items-center gap-2">
                    <span className={`inline-flex rounded px-1.5 py-0.5 text-[10px] font-medium ${h.valid ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'}`}>
                      {h.valid ? 'VALID' : 'INVALID'}
                    </span>
                    <span className="font-mono text-xs text-gray-600">{h.delegator} {'->'} {h.delegatee}</span>
                    <span className="text-xs text-gray-400">{h.timestamp.slice(0, 10)}</span>
                  </div>
                  {h.reasons.length > 0 && (
                    <div className="mt-1 pl-2 text-xs text-red-600">
                      {h.reasons.join('; ')}
                    </div>
                  )}
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
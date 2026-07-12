'use client';

import { useState, useCallback } from 'react';

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

const INITIAL_RULES: DelegationRule[] = [
  { id: 'rule-1', name: 'Self-Delegation Prevention', description: 'Users cannot delegate to themselves', enabled: true },
  { id: 'rule-2', name: 'Circular Delegation Prevention', description: 'Prevents A->B->A delegation cycles', enabled: true },
  { id: 'rule-3', name: 'Max Depth Limit', description: 'Delegation chain cannot exceed configured max depth', enabled: true },
  { id: 'rule-4', name: 'Scope Narrowing', description: 'Delegatee cannot have broader scopes than delegator', enabled: true },
  { id: 'rule-5', name: 'Tenant Boundary Check', description: 'Delegator and delegatee must be in same tenant', enabled: true },
  { id: 'rule-6', name: 'Active Status Check', description: 'Both parties must have active accounts', enabled: false },
];

const HISTORY: ValidationHistoryEntry[] = [
  { id: 'vh-001', timestamp: '2025-01-15T10:30:00Z', delegator: 'alice@corp.com', delegatee: 'bob@corp.com', scopes: ['audit:read', 'audit:export'], valid: true, reasons: [] },
  { id: 'vh-002', timestamp: '2025-01-15T09:15:00Z', delegator: 'alice@corp.com', delegatee: 'alice@corp.com', scopes: ['openid', 'profile'], valid: false, reasons: ['Self-delegation not allowed'] },
  { id: 'vh-003', timestamp: '2025-01-14T14:00:00Z', delegator: 'bob@corp.com', delegatee: 'charlie@corp.com', scopes: ['admin:all'], valid: false, reasons: ['Scope narrowing violated: delegatee scope broader than delegator'] },
  { id: 'vh-004', timestamp: '2025-01-14T08:00:00Z', delegator: 'admin@corp.com', delegatee: 'service-bot@corp.com', scopes: ['users:read', 'users:write'], valid: true, reasons: [] },
  { id: 'vh-005', timestamp: '2025-01-13T16:30:00Z', delegator: 'alice@corp.com', delegatee: 'charlie@corp.com', scopes: ['deploy:write'], valid: false, reasons: ['Circular delegation detected: alice->bob->charlie->alice'] },
];

export default function DelegationValidatorPage() {
  const [rules, setRules] = useState<DelegationRule[]>(INITIAL_RULES);
  const [delegator, setDelegator] = useState('alice@corp.com');
  const [delegatee, setDelegatee] = useState('bob@corp.com');
  const [scopes, setScopes] = useState('audit:read, audit:export');
  const [maxDepth, setMaxDepth] = useState(3);
  const [result, setResult] = useState<{ valid: boolean; reasons: string[] } | null>(null);
  const [validating, setValidating] = useState(false);

  const toggleRule = useCallback((id: string) => {
    setRules(rules.map(r => r.id === id ? { ...r, enabled: !r.enabled } : r));
  }, [rules]);

  const validate = useCallback(() => {
    setValidating(true);
    setTimeout(() => {
      const reasons: string[] = [];
      if (delegator === delegatee) reasons.push('Self-delegation not allowed');
      if (scopes.includes('admin:all')) reasons.push('Scope narrowing violated: admin:all is too broad for delegation');
      if (maxDepth > 5) reasons.push('Max depth limit exceeded (configured max: 5)');
      setResult({ valid: reasons.length === 0, reasons });
      setValidating(false);
    }, 500);
  }, [delegator, delegatee, scopes, maxDepth]);

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
'use client';

import { useState, useCallback } from 'react';

interface PolicyRule {
  id: string;
  name: string;
  effect: 'allow' | 'deny';
  matched: boolean;
  priority: number;
}

interface SimulationResult {
  decision: 'allow' | 'deny' | 'indeterminate';
  matchedRules: PolicyRule[];
  trace: string[];
  evaluatedAt: string;
  durationMs: number;
}

interface BatchResult {
  rowIndex: number;
  subject: string;
  resource: string;
  action: string;
  decision: string;
  matchedRules: number;
}

interface ImpactAnalysis {
  affectedUsers: number;
  allowCount: number;
  denyCount: number;
  indeterminateCount: number;
}

interface DiffEntry {
  field: string;
  before: string;
  after: string;
  changed: boolean;
}

const SAMPLE_POLICIES = [
  { id: 'pol-001', name: 'Admin Full Access', effect: 'allow' as const, priority: 100 },
  { id: 'pol-002', name: 'Read-Only Users', effect: 'allow' as const, priority: 50 },
  { id: 'pol-003', name: 'Deny Deleted Users', effect: 'deny' as const, priority: 200 },
  { id: 'pol-004', name: 'Tenant Isolation', effect: 'deny' as const, priority: 150 },
  { id: 'pol-005', name: 'MFA Required for Admin', effect: 'deny' as const, priority: 180 },
];

export default function PolicySimulationCenterPage() {
  const [selectedPolicy, setSelectedPolicy] = useState(SAMPLE_POLICIES[0].id);
  const [subjectAttrs, setSubjectAttrs] = useState('{\n  "role": "admin",\n  "department": "engineering",\n  "tenant_id": "tenant-001"\n}');
  const [resourceAttrs, setResourceAttrs] = useState('{\n  "type": "document",\n  "owner": "user-123",\n  "classification": "confidential"\n}');
  const [action, setAction] = useState('read');
  const [environment, setEnvironment] = useState('{\n  "time": "2025-01-15T10:00:00Z",\n  "ip": "192.168.1.50",\n  "mfa_verified": true\n}');
  const [result, setResult] = useState<SimulationResult | null>(null);
  const [simulating, setSimulating] = useState(false);
  const [batchResults, setBatchResults] = useState<BatchResult[]>([]);
  const [csvInput, setCsvInput] = useState('subject,resource,action\nuser-1,doc-1,read\nuser-2,doc-2,write\nuser-3,doc-3,delete');
  const [impact, setImpact] = useState<ImpactAnalysis | null>(null);
  const [diffEntries, setDiffEntries] = useState<DiffEntry[]>([]);
  const [activeTab, setActiveTab] = useState<'single' | 'batch' | 'impact'>('single');

  const simulate = useCallback(() => {
    setSimulating(true);
    setTimeout(() => {
      const rules: PolicyRule[] = SAMPLE_POLICIES.map(p => ({
        ...p,
        matched: Math.random() > 0.4,
      }));
      const denyMatch = rules.find(r => r.matched && r.effect === 'deny');
      const allowMatch = rules.find(r => r.matched && r.effect === 'allow');
      const decision = denyMatch ? 'deny' : allowMatch ? 'allow' : 'indeterminate';
      setResult({
        decision,
        matchedRules: rules.filter(r => r.matched),
        trace: [
          'Evaluating subject attributes: role=admin, department=engineering',
          `Checking policy: ${SAMPLE_POLICIES[0].name} (priority ${SAMPLE_POLICIES[0].priority})`,
          `Checking policy: ${SAMPLE_POLICIES[2].name} (priority ${SAMPLE_POLICIES[2].priority})`,
          denyMatch ? `Matched deny rule: ${denyMatch.name}` : 'No deny rules matched',
          allowMatch ? `Matched allow rule: ${allowMatch.name}` : 'No allow rules matched',
          `Final decision: ${decision}`,
        ],
        evaluatedAt: new Date().toISOString(),
        durationMs: Math.floor(Math.random() * 20) + 1,
      });
      setSimulating(false);
    }, 600);
  }, []);

  const runBatch = useCallback(() => {
    const lines = csvInput.trim().split('\n').slice(1);
    const results: BatchResult[] = lines.map((line, i) => {
      const [subject, resource, act] = line.split(',');
      const decisions = ['allow', 'deny', 'indeterminate'];
      const dec = decisions[Math.floor(Math.random() * decisions.length)];
      return {
        rowIndex: i + 1,
        subject: subject || '',
        resource: resource || '',
        action: act || '',
        decision: dec,
        matchedRules: Math.floor(Math.random() * 3) + 1,
      };
    });
    setBatchResults(results);
  }, [csvInput]);

  const runImpact = useCallback(() => {
    const total = Math.floor(Math.random() * 5000) + 100;
    const allow = Math.floor(total * 0.6);
    const deny = Math.floor(total * 0.3);
    setImpact({
      affectedUsers: total,
      allowCount: allow,
      denyCount: deny,
      indeterminateCount: total - allow - deny,
    });
    setDiffEntries([
      { field: 'Decision for admin users', before: 'allow', after: 'deny', changed: true },
      { field: 'Decision for read-only users', before: 'allow', after: 'allow', changed: false },
      { field: 'MFA enforcement', before: 'optional', after: 'required', changed: true },
      { field: 'Tenant isolation', before: 'soft', after: 'hard', changed: true },
      { field: 'Resource classification check', before: 'N/A', after: 'enforced', changed: true },
    ]);
  }, []);

  const decisionColor = (dec: string) =>
    dec === 'allow' ? 'text-green-600' : dec === 'deny' ? 'text-red-600' : 'text-yellow-600';

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Policy Simulation Center</h1>
        <p className="mt-1 text-sm text-gray-500">Test policy decisions before deployment, run batch simulations, and analyze impact.</p>
      </div>

      <div className="flex gap-2 border-b border-gray-200">
        {(['single', 'batch', 'impact'] as const).map(tab => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`px-4 py-2 text-sm font-medium border-b-2 ${
              activeTab === tab ? 'border-blue-500 text-blue-600' : 'border-transparent text-gray-500 hover:text-gray-700'
            }`}
          >
            {tab === 'single' ? 'Single Simulation' : tab === 'batch' ? 'Batch Simulation' : 'Impact Analysis'}
          </button>
        ))}
      </div>

      {activeTab === 'single' && (
        <div className="space-y-4">
          <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
            <label className="block text-sm font-medium text-gray-700">Policy Selector</label>
            <select
              value={selectedPolicy}
              onChange={e => setSelectedPolicy(e.target.value)}
              className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
            >
              {SAMPLE_POLICIES.map(p => (
                <option key={p.id} value={p.id}>{p.name} (priority: {p.priority}, effect: {p.effect})</option>
              ))}
            </select>
          </div>

          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
              <label className="block text-sm font-medium text-gray-700">Subject Attributes (JSON)</label>
              <textarea
                value={subjectAttrs}
                onChange={e => setSubjectAttrs(e.target.value)}
                rows={6}
                className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 font-mono text-xs"
              />
            </div>
            <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
              <label className="block text-sm font-medium text-gray-700">Resource Attributes (JSON)</label>
              <textarea
                value={resourceAttrs}
                onChange={e => setResourceAttrs(e.target.value)}
                rows={6}
                className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 font-mono text-xs"
              />
            </div>
          </div>

          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
              <label className="block text-sm font-medium text-gray-700">Action</label>
              <input
                value={action}
                onChange={e => setAction(e.target.value)}
                className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
              />
            </div>
            <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
              <label className="block text-sm font-medium text-gray-700">Environment (JSON)</label>
              <textarea
                value={environment}
                onChange={e => setEnvironment(e.target.value)}
                rows={3}
                className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 font-mono text-xs"
              />
            </div>
          </div>

          <button
            onClick={simulate}
            disabled={simulating}
            className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
          >
            {simulating ? 'Simulating...' : 'Simulate Decision'}
          </button>

          {result && (
            <div className="space-y-4">
              <div className="rounded-lg border border-gray-200 bg-white p-6 shadow-sm">
                <div className="flex items-center justify-between">
                  <div>
                    <span className="text-sm text-gray-500">Decision:</span>
                    <span className={`ml-2 text-2xl font-bold ${decisionColor(result.decision)}`}>{result.decision.toUpperCase()}</span>
                  </div>
                  <div className="text-right text-xs text-gray-400">
                    <div>Evaluated: {result.evaluatedAt}</div>
                    <div>Duration: {result.durationMs}ms</div>
                  </div>
                </div>
              </div>

              <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
                <h3 className="text-sm font-medium text-gray-700">Matched Rules</h3>
                <table className="mt-2 w-full text-sm">
                  <thead>
                    <tr className="border-b border-gray-200 text-left text-xs text-gray-500">
                      <th className="pb-2">Rule ID</th>
                      <th className="pb-2">Name</th>
                      <th className="pb-2">Effect</th>
                      <th className="pb-2">Priority</th>
                    </tr>
                  </thead>
                  <tbody>
                    {result.matchedRules.map(r => (
                      <tr key={r.id} className="border-b border-gray-100">
                        <td className="py-2 font-mono text-xs">{r.id}</td>
                        <td className="py-2">{r.name}</td>
                        <td className={`py-2 ${r.effect === 'allow' ? 'text-green-600' : 'text-red-600'}`}>{r.effect}</td>
                        <td className="py-2">{r.priority}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
                <h3 className="text-sm font-medium text-gray-700">Evaluation Trace</h3>
                <div className="mt-2 space-y-1 font-mono text-xs text-gray-600">
                  {result.trace.map((line, i) => (
                    <div key={i} className="flex gap-2">
                      <span className="text-gray-400">{i + 1}.</span>
                      <span>{line}</span>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          )}
        </div>
      )}

      {activeTab === 'batch' && (
        <div className="space-y-4">
          <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
            <label className="block text-sm font-medium text-gray-700">Batch Input (CSV)</label>
            <p className="mt-1 text-xs text-gray-400">Format: subject,resource,action</p>
            <textarea
              value={csvInput}
              onChange={e => setCsvInput(e.target.value)}
              rows={8}
              className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 font-mono text-xs"
            />
            <button
              onClick={runBatch}
              className="mt-2 rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
            >
              Run Batch Simulation
            </button>
          </div>

          {batchResults.length > 0 && (
            <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
              <h3 className="text-sm font-medium text-gray-700">Batch Results ({batchResults.length} rows)</h3>
              <table className="mt-2 w-full text-sm">
                <thead>
                  <tr className="border-b border-gray-200 text-left text-xs text-gray-500">
                    <th className="pb-2">Row</th>
                    <th className="pb-2">Subject</th>
                    <th className="pb-2">Resource</th>
                    <th className="pb-2">Action</th>
                    <th className="pb-2">Decision</th>
                    <th className="pb-2">Matched Rules</th>
                  </tr>
                </thead>
                <tbody>
                  {batchResults.map(r => (
                    <tr key={r.rowIndex} className="border-b border-gray-100">
                      <td className="py-2 text-xs">{r.rowIndex}</td>
                      <td className="py-2 font-mono text-xs">{r.subject}</td>
                      <td className="py-2 font-mono text-xs">{r.resource}</td>
                      <td className="py-2">{r.action}</td>
                      <td className={`py-2 font-medium ${decisionColor(r.decision)}`}>{r.decision}</td>
                      <td className="py-2">{r.matchedRules}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}

      {activeTab === 'impact' && (
        <div className="space-y-4">
          <button
            onClick={runImpact}
            className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
          >
            Run Impact Analysis
          </button>

          {impact && (
            <>
              <div className="grid grid-cols-1 gap-4 md:grid-cols-4">
                <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
                  <p className="text-xs text-gray-500">Affected Users</p>
                  <p className="mt-1 text-2xl font-bold text-gray-900">{impact.affectedUsers.toLocaleString()}</p>
                </div>
                <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
                  <p className="text-xs text-gray-500">Allow</p>
                  <p className="mt-1 text-2xl font-bold text-green-600">{impact.allowCount.toLocaleString()}</p>
                </div>
                <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
                  <p className="text-xs text-gray-500">Deny</p>
                  <p className="mt-1 text-2xl font-bold text-red-600">{impact.denyCount.toLocaleString()}</p>
                </div>
                <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
                  <p className="text-xs text-gray-500">Indeterminate</p>
                  <p className="mt-1 text-2xl font-bold text-yellow-600">{impact.indeterminateCount.toLocaleString()}</p>
                </div>
              </div>

              <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
                <h3 className="text-sm font-medium text-gray-700">Before / After Diff</h3>
                <table className="mt-2 w-full text-sm">
                  <thead>
                    <tr className="border-b border-gray-200 text-left text-xs text-gray-500">
                      <th className="pb-2">Field</th>
                      <th className="pb-2">Before</th>
                      <th className="pb-2">After</th>
                      <th className="pb-2">Changed</th>
                    </tr>
                  </thead>
                  <tbody>
                    {diffEntries.map((d, i) => (
                      <tr key={i} className="border-b border-gray-100">
                        <td className="py-2">{d.field}</td>
                        <td className="py-2 text-gray-500">{d.before}</td>
                        <td className="py-2 font-medium">{d.after}</td>
                        <td className="py-2">
                          {d.changed ? (
                            <span className="inline-flex rounded bg-yellow-100 px-2 py-0.5 text-xs text-yellow-700">changed</span>
                          ) : (
                            <span className="text-xs text-gray-400">unchanged</span>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </>
          )}
        </div>
      )}
    </div>
  );
}

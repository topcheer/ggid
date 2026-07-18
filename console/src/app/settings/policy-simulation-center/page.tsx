'use client';
import { useState, useEffect, useCallback } from 'react';
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface PolicyRule { id: string; name: string; effect: 'allow' | 'deny'; matched: boolean; priority: number; }
interface SimulationResult { decision: 'allow' | 'deny' | 'indeterminate'; matchedRules: PolicyRule[]; trace: string[]; evaluatedAt: string; durationMs: number; }
interface BatchResult { rowIndex: number; subject: string; resource: string; action: string; decision: string; matchedRules: number; }
interface ImpactAnalysis { affectedUsers: number; allowCount: number; denyCount: number; indeterminateCount: number; }

export default function PolicySimulationCenterPage() {
  const t = useTranslations();

  const [policies, setPolicies] = useState<{ id: string; name: string; effect: 'allow' | 'deny'; priority: number }[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedPolicy, setSelectedPolicy] = useState('');
  const [subjectAttrs, setSubjectAttrs] = useState('{\n  "role": "admin",\n  "department": "engineering"\n}');
  const [resourceAttrs, setResourceAttrs] = useState('{\n  "type": "document",\n  "owner": "user-123"\n}');
  const [action, setAction] = useState('read');
  const [environment, setEnvironment] = useState('{\n  "time": "2025-01-15T10:00:00Z",\n  "ip": "192.168.1.50"\n}');
  const [result, setResult] = useState<SimulationResult | null>(null);
  const [simulating, setSimulating] = useState(false);
  const [batchResults, setBatchResults] = useState<BatchResult[]>([]);
  const [csvInput, setCsvInput] = useState('subject,resource,action\nuser-1,doc-1,read\nuser-2,doc-2,write');
  const [impact, setImpact] = useState<ImpactAnalysis | null>(null);
  const [activeTab, setActiveTab] = useState<'single' | 'batch' | 'impact'>('single');

  useEffect(() => {
    fetch("/api/v1/policy/policies", {
      headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
    })
      .then(res => { if (!res.ok) return null; return res.json(); })
      .then(data => {
        const items = data.policies || data.items || [];
        setPolicies(items);
        if (items.length > 0) setSelectedPolicy(items[0].id);
        setLoading(false);
      })
      .catch(err => { setError(err.message); setLoading(false); });
  }, []);

  const simulate = useCallback(() => {
    setSimulating(true);
    fetch("/api/v1/policy/simulate", {
      method: "POST",
      headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
      body: JSON.stringify({ policyId: selectedPolicy, subject: subjectAttrs, resource: resourceAttrs, action, environment }),
    })
      .then(res => { if (!res.ok) return null; return res.json(); })
      .then(data => { setResult(data); setSimulating(false); })
      .catch(err => { setError(err.message); setSimulating(false); });
  }, [selectedPolicy, subjectAttrs, resourceAttrs, action, environment]);

  const runBatch = useCallback(() => {
    fetch("/api/v1/policy/simulate/batch", {
      method: "POST",
      headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
      body: JSON.stringify({ csv: csvInput }),
    })
      .then(res => { if (!res.ok) return null; return res.json(); })
      .then(data => { setBatchResults(data.results || []); })
      .catch(err => { setError(err.message); });
  }, [csvInput]);

  const runImpact = useCallback(() => {
    fetch("/api/v1/policy/simulate/impact", {
      method: "POST",
      headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
      body: JSON.stringify({ policyId: selectedPolicy }),
    })
      .then(res => { if (!res.ok) return null; return res.json(); })
      .then(data => { setImpact(data); })
      .catch(err => { setError(err.message); });
  }, [selectedPolicy]);

  const decisionColor = (dec: string) => dec === 'allow' ? 'text-green-600' : dec === 'deny' ? 'text-red-600' : 'text-yellow-600';

  if (loading) return <div className="p-6"><h1 className="text-2xl font-bold">Policy Simulation Center</h1><p className="text-gray-600 mt-2">Loading...</p></div>;
  if (error) return <div className="p-6"><h1 className="text-2xl font-bold">Policy Simulation Center</h1><p className="text-red-600 mt-2">Error: {error}</p></div>;

  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold text-gray-900">Policy Simulation Center</h1><p className="mt-1 text-sm text-gray-500">Test policy decisions before deployment, run batch simulations, and analyze impact.</p></div>

      <div className="flex gap-2 border-b border-gray-200">
        {(['single', 'batch', 'impact'] as const).map(tab => (
          <button key={tab} onClick={() => setActiveTab(tab)} className={`px-4 py-2 text-sm font-medium border-b-2 ${activeTab === tab ? 'border-blue-500 text-blue-600' : 'border-transparent text-gray-500 hover:text-gray-700'}`}>{tab === 'single' ? 'Single Simulation' : tab === 'batch' ? 'Batch Simulation' : 'Impact Analysis'}</button>
        ))}
      </div>

      {activeTab === 'single' && (
        <div className="space-y-4">
          <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
            <label className="block text-sm font-medium text-gray-700">Policy Selector</label>
            <select aria-label="selected Policy" value={selectedPolicy} onChange={e => setSelectedPolicy(e.target.value)} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm">
              {policies.map(p => <option key={p.id} value={p.id}>{p.name} (priority: {p.priority}, effect: {p.effect})</option>)}
            </select>
          </div>
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm"><label className="block text-sm font-medium text-gray-700">Subject Attributes (JSON)</label><textarea aria-label="Text input" value={subjectAttrs} onChange={e => setSubjectAttrs(e.target.value)} rows={6} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 font-mono text-xs" /></div>
            <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm"><label className="block text-sm font-medium text-gray-700">Resource Attributes (JSON)</label><textarea aria-label="Text input" value={resourceAttrs} onChange={e => setResourceAttrs(e.target.value)} rows={6} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 font-mono text-xs" /></div>
          </div>
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm"><label className="block text-sm font-medium text-gray-700">Action</label><input aria-label="action" value={action} onChange={e => setAction(e.target.value)} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm" /></div>
            <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm"><label className="block text-sm font-medium text-gray-700">Environment (JSON)</label><textarea aria-label="Text input" value={environment} onChange={e => setEnvironment(e.target.value)} rows={3} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 font-mono text-xs" /></div>
          </div>
          <button onClick={simulate} disabled={simulating} aria-label="Run policy simulation" className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50">{simulating ? 'Simulating...' : 'Run Simulation'}</button>
          {result && (
            <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
              <h3 className="text-sm font-medium text-gray-700">Result</h3>
              <div className="mt-2"><span className={`text-2xl font-bold ${decisionColor(result.decision)}`}>{result.decision.toUpperCase()}</span><span className="ml-3 text-sm text-gray-500">{result.durationMs}ms</span></div>
              {result.matchedRules && result.matchedRules.length > 0 && (<div className="mt-3 space-y-1">{result.matchedRules.map((r: any, i: number) => <div key={i} className="text-sm"><span className={`px-2 py-0.5 rounded text-xs ${r.effect === 'deny' ? 'bg-red-100 text-red-700' : 'bg-green-100 text-green-700'}`}>{r.effect}</span> {r.name}</div>)}</div>)}
              {result.trace && result.trace.length > 0 && (<div className="mt-3 space-y-1">{result.trace.map((t: any, i: number) => <div key={i} className="text-xs text-gray-600 font-mono">{t}</div>)}</div>)}
            </div>
          )}
        </div>
      )}

      {activeTab === 'batch' && (
        <div className="space-y-4">
          <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm"><label className="block text-sm font-medium text-gray-700">Batch CSV Input</label><textarea aria-label="Text input" value={csvInput} onChange={e => setCsvInput(e.target.value)} rows={8} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 font-mono text-xs" /></div>
          <button onClick={runBatch} aria-label="Run batch simulation" className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700">Run Batch</button>
          {batchResults.length > 0 && (
            <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm overflow-hidden">
              <table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">#</th><th scope="col">Subject</th><th>Resource</th><th>Action</th><th>Decision</th><th>Rules</th></tr></thead>
                <tbody>{batchResults.map(r => <tr key={r.rowIndex} className="border-b"><td className="py-2">{r.rowIndex}</td><td className="font-mono text-xs">{r.subject}</td><td className="font-mono text-xs">{r.resource}</td><td className="text-xs">{r.action}</td><td><span className={`font-bold ${decisionColor(r.decision)}`}>{r.decision}</span></td><td className="text-xs">{r.matchedRules}</td></tr>)}</tbody>
              </table>
            </div>
          )}
        </div>
      )}

      {activeTab === 'impact' && (
        <div className="space-y-4">
          <button onClick={runImpact} aria-label="Run impact analysis" className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700">Run Impact Analysis</button>
          {impact && (
            <div className="grid grid-cols-4 gap-4">
              <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm text-center"><div className="text-2xl font-bold">{impact.affectedUsers}</div><div className="text-xs text-gray-500">Affected Users</div></div>
              <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm text-center"><div className="text-2xl font-bold text-green-600">{impact.allowCount}</div><div className="text-xs text-gray-500">Allow</div></div>
              <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm text-center"><div className="text-2xl font-bold text-red-600">{impact.denyCount}</div><div className="text-xs text-gray-500">Deny</div></div>
              <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm text-center"><div className="text-2xl font-bold text-yellow-600">{impact.indeterminateCount}</div><div className="text-xs text-gray-500">Indeterminate</div></div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

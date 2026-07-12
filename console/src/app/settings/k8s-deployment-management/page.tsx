'use client';

import { useState, useCallback, useEffect } from 'react';

interface Deployment {
  id: string;
  service: string;
  namespace: string;
  replicas: number;
  readyReplicas: number;
  image: string;
  version: string;
  status: 'Running' | 'Updating' | 'Failed';
  uptime: string;
  errorRate: number;
  created: string;
}

interface Pod {
  name: string;
  status: 'Running' | 'Pending' | 'Failed' | 'CrashLoopBackOff';
  restarts: number;
  cpuUsage: number;
  memoryUsage: number;
  age: string;
  node: string;
}

interface Revision {
  revision: number;
  image: string;
  deployedAt: string;
  replicas: number;
}

const INITIAL_DEPLOYMENTS: Deployment[] = [
  { id: 'dep-gateway', service: 'gateway', namespace: 'ggid', replicas: 3, readyReplicas: 3, image: 'ggid/gateway:latest', version: '1.4.2', status: 'Running', uptime: '5d12h', errorRate: 0.02, created: '2025-01-10T08:00:00Z' },
  { id: 'dep-auth', service: 'auth', namespace: 'ggid', replicas: 2, readyReplicas: 2, image: 'ggid/auth:latest', version: '1.4.2', status: 'Running', uptime: '5d12h', errorRate: 0.05, created: '2025-01-10T08:00:00Z' },
  { id: 'dep-identity', service: 'identity', namespace: 'ggid', replicas: 2, readyReplicas: 2, image: 'ggid/identity:latest', version: '1.4.1', status: 'Running', uptime: '3d8h', errorRate: 0.01, created: '2025-01-12T10:00:00Z' },
  { id: 'dep-oauth', service: 'oauth', namespace: 'ggid', replicas: 2, readyReplicas: 1, image: 'ggid/oauth:latest', version: '1.4.3', status: 'Updating', uptime: '0h', errorRate: 0.15, created: '2025-01-15T09:00:00Z' },
  { id: 'dep-policy', service: 'policy', namespace: 'ggid', replicas: 2, readyReplicas: 2, image: 'ggid/policy:latest', version: '1.4.2', status: 'Running', uptime: '5d12h', errorRate: 0.03, created: '2025-01-10T08:00:00Z' },
  { id: 'dep-org', service: 'org', namespace: 'ggid', replicas: 1, readyReplicas: 1, image: 'ggid/org:latest', version: '1.4.0', status: 'Running', uptime: '7d0h', errorRate: 0.0, created: '2025-01-08T08:00:00Z' },
  { id: 'dep-audit', service: 'audit', namespace: 'ggid', replicas: 2, readyReplicas: 0, image: 'ggid/audit:latest', version: '1.4.3', status: 'Failed', uptime: '0h', errorRate: 1.0, created: '2025-01-15T08:30:00Z' },
];

const PODS_BY_SERVICE: Record<string, Pod[]> = {
  gateway: [
    { name: 'gateway-7f9d-abc12', status: 'Running', restarts: 0, cpuUsage: 45, memoryUsage: 128, age: '5d12h', node: 'node-01' },
    { name: 'gateway-7f9d-def34', status: 'Running', restarts: 0, cpuUsage: 52, memoryUsage: 135, age: '5d12h', node: 'node-02' },
    { name: 'gateway-7f9d-ghi56', status: 'Running', restarts: 0, cpuUsage: 38, memoryUsage: 120, age: '5d12h', node: 'node-03' },
  ],
  auth: [
    { name: 'auth-6b8c-abc12', status: 'Running', restarts: 0, cpuUsage: 30, memoryUsage: 95, age: '5d12h', node: 'node-01' },
    { name: 'auth-6b8c-def34', status: 'Running', restarts: 1, cpuUsage: 35, memoryUsage: 102, age: '5d12h', node: 'node-02' },
  ],
  oauth: [
    { name: 'oauth-5c7e-abc12', status: 'Running', restarts: 0, cpuUsage: 40, memoryUsage: 110, age: '10m', node: 'node-01' },
    { name: 'oauth-5c7e-def34', status: 'Pending', restarts: 0, cpuUsage: 0, memoryUsage: 0, age: '2m', node: 'node-03' },
  ],
  audit: [
    { name: 'audit-4d6f-abc12', status: 'CrashLoopBackOff', restarts: 8, cpuUsage: 0, memoryUsage: 0, age: '30m', node: 'node-02' },
    { name: 'audit-4d6f-def34', status: 'Failed', restarts: 5, cpuUsage: 0, memoryUsage: 0, age: '30m', node: 'node-03' },
  ],
};

const REVISIONS: Revision[] = [
  { revision: 15, image: 'ggid/oauth:1.4.3', deployedAt: '2025-01-15T09:00:00Z', replicas: 2 },
  { revision: 14, image: 'ggid/oauth:1.4.2', deployedAt: '2025-01-10T08:00:00Z', replicas: 2 },
  { revision: 13, image: 'ggid/oauth:1.4.1', deployedAt: '2025-01-05T08:00:00Z', replicas: 2 },
  { revision: 12, image: 'ggid/oauth:1.4.0', deployedAt: '2024-12-28T08:00:00Z', replicas: 1 },
];

const STATUS_COLORS: Record<string, string> = {
  Running: 'bg-green-100 text-green-700',
  Updating: 'bg-blue-100 text-blue-700',
  Failed: 'bg-red-100 text-red-700',
  Pending: 'bg-yellow-100 text-yellow-700',
  CrashLoopBackOff: 'bg-red-100 text-red-700',
};

const CONFIG_DIFF = `--- Current
+++ Proposed
@@ spec:
   replicas: 2
-  image: ggid/oauth:1.4.2
+  image: ggid/oauth:1.4.3
   resources:
     requests:
-      cpu: 100m
+      cpu: 200m
       memory: 128Mi
     limits:
       cpu: 500m
-      memory: 256Mi
+      memory: 512Mi
   env:
+    - name: LOG_LEVEL
+      value: debug`;

export default function K8sDeploymentManagementPage() {

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [data, setData] = useState<any[]>([]);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const res = await fetch("/healthz", {
          method: "GET",
          headers: {
            "Content-Type": "application/json",
            "X-Tenant-ID": "00000000-0000-0000-0000-000000000001",
          },
        });
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const json = await res.json();
        setData(Array.isArray(json) ? json : [json]);
      } catch (e) {
        setError(e instanceof Error ? e.message : "Failed to load");
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, []);

  if (loading) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-500">Error: {error}</div>;
  if (!data || data.length === 0) return <div className="p-8 text-gray-500">No data available</div>;
  const [deployments, setDeployments] = useState<Deployment[]>(INITIAL_DEPLOYMENTS);
  const [selectedDep, setSelectedDep] = useState<Deployment | null>(null);
  const [maxSurge, setMaxSurge] = useState(1);
  const [maxUnavailable, setMaxUnavailable] = useState(0);
  const [scaleValue, setScaleValue] = useState(3);
  const [activeTab, setActiveTab] = useState<'deployments' | 'pods' | 'rolling' | 'diff' | 'rollback'>('deployments');
  const [showDiff, setShowDiff] = useState(false);

  const pods = selectedDep ? PODS_BY_SERVICE[selectedDep.service] || [] : [];

  const restartDeployment = useCallback((id: string) => {
    setDeployments(deployments.map(d =>
      d.id === id ? { ...d, status: 'Updating', readyReplicas: 0 } : d
    ));
  }, [deployments]);

  const scaleDeployment = useCallback((id: string, replicas: number) => {
    setDeployments(deployments.map(d =>
      d.id === id ? { ...d, replicas, readyReplicas: replicas, status: 'Running' } : d
    ));
  }, [deployments]);

  const rollback = useCallback((revision: number) => {
    const rev = REVISIONS.find(r => r.revision === revision);
    if (rev && selectedDep) {
      setDeployments(deployments.map(d =>
        d.id === selectedDep.id ? {
          ...d,
          image: rev.image,
          version: rev.image.split(':').pop() || d.version,
          status: 'Updating',
          readyReplicas: 0,
        } : d
      ));
    }
  }, [deployments, selectedDep]);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">K8s Deployment Management</h1>
        <p className="mt-1 text-sm text-gray-500">Manage Kubernetes deployments, rolling updates, pod status, and rollbacks.</p>
      </div>

      <div className="flex gap-2 border-b border-gray-200">
        {(['deployments', 'pods', 'rolling', 'diff', 'rollback'] as const).map(tab => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`px-4 py-2 text-sm font-medium border-b-2 ${
              activeTab === tab ? 'border-blue-500 text-blue-600' : 'border-transparent text-gray-500 hover:text-gray-700'
            }`}
          >
            {tab === 'deployments' ? 'Deployments' : tab === 'pods' ? 'Pods' : tab === 'rolling' ? 'Rolling Update' : tab === 'diff' ? 'Config Diff' : 'Rollback'}
          </button>
        ))}
      </div>

      {activeTab === 'deployments' && (
        <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-200 text-left text-xs text-gray-500">
                <th className="pb-2">Service</th>
                <th className="pb-2">Namespace</th>
                <th className="pb-2">Replicas</th>
                <th className="pb-2">Image</th>
                <th className="pb-2">Version</th>
                <th className="pb-2">Status</th>
                <th className="pb-2">Uptime</th>
                <th className="pb-2">Error Rate</th>
                <th className="pb-2">Actions</th>
              </tr>
            </thead>
            <tbody>
              {deployments.map(d => (
                <tr
                  key={d.id}
                  className={`border-b border-gray-100 cursor-pointer hover:bg-gray-50 ${selectedDep?.id === d.id ? 'bg-blue-50' : ''}`}
                  onClick={() => { setSelectedDep(d); setScaleValue(d.replicas); }}
                >
                  <td className="py-2 font-medium">{d.service}</td>
                  <td className="py-2 text-xs font-mono">{d.namespace}</td>
                  <td className="py-2">{d.readyReplicas}/{d.replicas}</td>
                  <td className="py-2 text-xs font-mono">{d.image}</td>
                  <td className="py-2">{d.version}</td>
                  <td className="py-2">
                    <span className={`inline-flex rounded px-2 py-0.5 text-xs ${STATUS_COLORS[d.status]}`}>{d.status}</span>
                  </td>
                  <td className="py-2 text-xs text-gray-500">{d.uptime}</td>
                  <td className="py-2">
                    <span className={d.errorRate > 0.1 ? 'text-red-600 font-medium' : 'text-gray-600'}>
                      {(d.errorRate * 100).toFixed(1)}%
                    </span>
                  </td>
                  <td className="py-2">
                    <button
                      onClick={(e) => { e.stopPropagation(); restartDeployment(d.id); }}
                      className="text-xs text-blue-600 hover:underline"
                    >Restart</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {selectedDep && (
            <div className="mt-4 rounded-md bg-blue-50 p-3 text-sm text-blue-700">
              Selected: <span className="font-medium">{selectedDep.service}</span> — {selectedDep.readyReplicas}/{selectedDep.replicas} replicas, version {selectedDep.version}
            </div>
          )}
        </div>
      )}

      {activeTab === 'pods' && (
        <div className="space-y-4">
          {!selectedDep ? (
            <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
              <p className="text-sm text-gray-400">Select a deployment to view its pods.</p>
            </div>
          ) : (
            <>
              <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
                <h3 className="text-sm font-medium text-gray-700">Pods for {selectedDep.service} ({pods.length})</h3>
                <table className="mt-2 w-full text-sm">
                  <thead>
                    <tr className="border-b border-gray-200 text-left text-xs text-gray-500">
                      <th className="pb-2">Pod Name</th>
                      <th className="pb-2">Status</th>
                      <th className="pb-2">Restarts</th>
                      <th className="pb-2">CPU</th>
                      <th className="pb-2">Memory</th>
                      <th className="pb-2">Age</th>
                      <th className="pb-2">Node</th>
                    </tr>
                  </thead>
                  <tbody>
                    {pods.map(p => (
                      <tr key={p.name} className="border-b border-gray-100">
                        <td className="py-2 font-mono text-xs">{p.name}</td>
                        <td className="py-2">
                          <span className={`inline-flex rounded px-2 py-0.5 text-xs ${STATUS_COLORS[p.status] || 'bg-gray-100 text-gray-600'}`}>{p.status}</span>
                        </td>
                        <td className="py-2">
                          <span className={p.restarts > 3 ? 'text-red-600 font-medium' : ''}>{p.restarts}</span>
                        </td>
                        <td className="py-2">
                          <div className="flex items-center gap-1">
                            <div className="h-1.5 w-16 rounded-full bg-gray-200">
                              <div className={`h-full rounded-full ${p.cpuUsage > 80 ? 'bg-red-500' : p.cpuUsage > 60 ? 'bg-yellow-500' : 'bg-green-500'}`} style={{ width: `${p.cpuUsage}%` }} />
                            </div>
                            <span className="text-xs text-gray-500">{p.cpuUsage}m</span>
                          </div>
                        </td>
                        <td className="py-2">
                          <div className="flex items-center gap-1">
                            <div className="h-1.5 w-16 rounded-full bg-gray-200">
                              <div className={`h-full rounded-full ${p.memoryUsage > 400 ? 'bg-red-500' : p.memoryUsage > 200 ? 'bg-yellow-500' : 'bg-green-500'}`} style={{ width: `${Math.min(p.memoryUsage / 5, 100)}%` }} />
                            </div>
                            <span className="text-xs text-gray-500">{p.memoryUsage}Mi</span>
                          </div>
                        </td>
                        <td className="py-2 text-xs text-gray-500">{p.age}</td>
                        <td className="py-2 text-xs font-mono">{p.node}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
                <h3 className="text-sm font-medium text-gray-700">Scale Deployment</h3>
                <div className="mt-3 flex items-center gap-4">
                  <input
                    type="range"
                    min={1}
                    max={10}
                    value={scaleValue}
                    onChange={e => setScaleValue(Number(e.target.value))}
                    className="flex-1"
                  />
                  <span className="text-lg font-bold text-gray-900">{scaleValue}</span>
                  <button
                    onClick={() => scaleDeployment(selectedDep.id, scaleValue)}
                    className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
                  >Apply</button>
                </div>
              </div>
            </>
          )}
        </div>
      )}

      {activeTab === 'rolling' && (
        <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
          <h3 className="text-sm font-medium text-gray-700">Rolling Update Configuration</h3>
          <div className="mt-4 grid grid-cols-1 gap-4 md:grid-cols-2">
            <div>
              <label className="block text-xs font-medium text-gray-600">Max Surge</label>
              <input
                type="number"
                min={0}
                max={10}
                value={maxSurge}
                onChange={e => setMaxSurge(Number(e.target.value))}
                className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
              />
              <p className="mt-1 text-xs text-gray-400">Max number of pods created above desired count</p>
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-600">Max Unavailable</label>
              <input
                type="number"
                min={0}
                max={10}
                value={maxUnavailable}
                onChange={e => setMaxUnavailable(Number(e.target.value))}
                className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
              />
              <p className="mt-1 text-xs text-gray-400">Max number of pods unavailable during update</p>
            </div>
          </div>
          <div className="mt-4 flex gap-2">
            <button className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700">Apply Rolling Update</button>
            <button onClick={() => setShowDiff(!showDiff)} className="rounded-md bg-gray-200 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-300">Preview Config Diff</button>
          </div>
          {showDiff && (
            <pre className="mt-4 overflow-x-auto rounded-md bg-gray-900 p-4 text-xs text-gray-100 font-mono whitespace-pre">{CONFIG_DIFF}</pre>
          )}
        </div>
      )}

      {activeTab === 'diff' && (
        <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
          <h3 className="text-sm font-medium text-gray-700">Config Diff Viewer</h3>
          <pre className="mt-3 overflow-x-auto rounded-md bg-gray-900 p-4 text-xs text-gray-100 font-mono whitespace-pre">{CONFIG_DIFF}</pre>
        </div>
      )}

      {activeTab === 'rollback' && (
        <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
          <h3 className="text-sm font-medium text-gray-700">Rollback to Previous Revision</h3>
          <table className="mt-2 w-full text-sm">
            <thead>
              <tr className="border-b border-gray-200 text-left text-xs text-gray-500">
                <th className="pb-2">Revision</th>
                <th className="pb-2">Image</th>
                <th className="pb-2">Deployed At</th>
                <th className="pb-2">Replicas</th>
                <th className="pb-2">Action</th>
              </tr>
            </thead>
            <tbody>
              {REVISIONS.map(r => (
                <tr key={r.revision} className="border-b border-gray-100">
                  <td className="py-2 font-mono">#{r.revision}</td>
                  <td className="py-2 font-mono text-xs">{r.image}</td>
                  <td className="py-2 text-xs text-gray-500">{r.deployedAt.slice(0, 19).replace('T', ' ')}</td>
                  <td className="py-2">{r.replicas}</td>
                  <td className="py-2">
                    <button
                      onClick={() => rollback(r.revision)}
                      className="rounded bg-orange-100 px-3 py-1 text-xs font-medium text-orange-700 hover:bg-orange-200"
                    >Rollback</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

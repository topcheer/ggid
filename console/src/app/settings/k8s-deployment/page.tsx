"use client";
import { useState } from "react";

interface Deployment {
  name: string;
  namespace: string;
  replicas: number;
  available: number;
  image: string;
  age_days: number;
}

interface ResourceLimits {
  cpu_request: string;
  cpu_limit: string;
  memory_request: string;
  memory_limit: string;
}

interface ProbeConfig {
  path: string;
  port: number;
  initial_delay_s: number;
  period_s: number;
  timeout_s: number;
}

const defaultDeployments: Deployment[] = [
  { name: "ggid-gateway", namespace: "production", replicas: 3, available: 3, image: "ggid/gateway:v2.1.0", age_days: 12 },
  { name: "ggid-auth", namespace: "production", replicas: 2, available: 2, image: "ggid/auth:v2.1.0", age_days: 12 },
  { name: "ggid-identity", namespace: "production", replicas: 2, available: 1, image: "ggid/identity:v2.0.9", age_days: 5 },
  { name: "ggid-policy", namespace: "production", replicas: 2, available: 2, image: "ggid/policy:v2.1.0", age_days: 12 },
];

export default function K8sDeploymentPage() {
  const [deployments] = useState<Deployment[]>(defaultDeployments);
  const [newImage, setNewImage] = useState("");
  const [maxSurge, setMaxSurge] = useState(1);
  const [maxUnavailable, setMaxUnavailable] = useState(0);
  const [resourceLimits, setResourceLimits] = useState<ResourceLimits>({ cpu_request: "100m", cpu_limit: "500m", memory_request: "128Mi", memory_limit: "512Mi" });
  const [livenessProbe, setLivenessProbe] = useState<ProbeConfig>({ path: "/healthz", port: 8080, initial_delay_s: 15, period_s: 10, timeout_s: 3 });
  const [readinessProbe, setReadinessProbe] = useState<ProbeConfig>({ path: "/ready", port: 8080, initial_delay_s: 5, period_s: 5, timeout_s: 3 });
  const [selectedDeployment, setSelectedDeployment] = useState<string | null>(null);

  return (
    <div className="p-8 space-y-6 max-w-5xl">
      <h1 className="text-2xl font-bold">Kubernetes Deployment Management</h1>
      <p className="text-gray-600">Manage K8s deployments, rolling updates, probes, and resource limits.</p>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Deployments</h2>
        <table className="w-full text-sm">
          <thead><tr className="border-b text-left"><th className="py-2">Name</th><th>Namespace</th><th>Replicas</th><th>Available</th><th>Image</th><th>Age</th><th>Action</th></tr></thead>
          <tbody>
            {deployments.map((d: Deployment, i: number) => (
              <tr key={i} className={`border-b cursor-pointer hover:bg-gray-50 ${selectedDeployment === d.name ? "bg-blue-50" : ""}`} onClick={() => setSelectedDeployment(d.name)}>
                <td className="py-2 font-medium">{d.name}</td><td className="font-mono text-xs">{d.namespace}</td><td>{d.replicas}</td><td><span className={d.available < d.replicas ? "text-red-600 font-medium" : "text-green-600"}>{d.available}/{d.replicas}</span></td><td className="font-mono text-xs">{d.image}</td><td>{d.age_days}d</td>
                <td><button className="text-xs text-blue-600 hover:underline">Rollback</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {selectedDeployment && (
        <div className="bg-white rounded-lg p-6 shadow space-y-4">
          <h2 className="text-lg font-semibold">Rolling Update: {selectedDeployment}</h2>
          <div className="grid grid-cols-3 gap-4">
            <div><label className="block text-sm font-medium mb-1">New Image</label><input type="text" value={newImage} onChange={(e) => setNewImage(e.target.value)} placeholder="ggid/auth:v2.2.0" className="border rounded px-3 py-2 w-full font-mono text-sm" /></div>
            <div><label className="block text-sm font-medium mb-1">Max Surge</label><input type="number" value={maxSurge} onChange={(e) => setMaxSurge(parseInt(e.target.value) || 0)} className="border rounded px-3 py-2 w-full" /></div>
            <div><label className="block text-sm font-medium mb-1">Max Unavailable</label><input type="number" value={maxUnavailable} onChange={(e) => setMaxUnavailable(parseInt(e.target.value) || 0)} className="border rounded px-3 py-2 w-full" /></div>
          </div>
          <button className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700">Apply Rolling Update</button>
        </div>
      )}

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Resource Limits</h2>
        <div className="grid grid-cols-4 gap-4">
          <div><label className="block text-sm font-medium mb-1">CPU Request</label><input type="text" value={resourceLimits.cpu_request} onChange={(e) => setResourceLimits({ ...resourceLimits, cpu_request: e.target.value })} className="border rounded px-3 py-2 w-full font-mono text-sm" /></div>
          <div><label className="block text-sm font-medium mb-1">CPU Limit</label><input type="text" value={resourceLimits.cpu_limit} onChange={(e) => setResourceLimits({ ...resourceLimits, cpu_limit: e.target.value })} className="border rounded px-3 py-2 w-full font-mono text-sm" /></div>
          <div><label className="block text-sm font-medium mb-1">Memory Request</label><input type="text" value={resourceLimits.memory_request} onChange={(e) => setResourceLimits({ ...resourceLimits, memory_request: e.target.value })} className="border rounded px-3 py-2 w-full font-mono text-sm" /></div>
          <div><label className="block text-sm font-medium mb-1">Memory Limit</label><input type="text" value={resourceLimits.memory_limit} onChange={(e) => setResourceLimits({ ...resourceLimits, memory_limit: e.target.value })} className="border rounded px-3 py-2 w-full font-mono text-sm" /></div>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-6">
        <div className="bg-white rounded-lg p-6 shadow space-y-3">
          <h2 className="text-lg font-semibold">Liveness Probe</h2>
          <div className="grid grid-cols-2 gap-3">
            <div><label className="block text-xs font-medium mb-1">Path</label><input type="text" value={livenessProbe.path} onChange={(e) => setLivenessProbe({ ...livenessProbe, path: e.target.value })} className="border rounded px-2 py-1 w-full text-sm font-mono" /></div>
            <div><label className="block text-xs font-medium mb-1">Port</label><input type="number" value={livenessProbe.port} onChange={(e) => setLivenessProbe({ ...livenessProbe, port: parseInt(e.target.value) || 0 })} className="border rounded px-2 py-1 w-full text-sm" /></div>
            <div><label className="block text-xs font-medium mb-1">Initial Delay (s)</label><input type="number" value={livenessProbe.initial_delay_s} onChange={(e) => setLivenessProbe({ ...livenessProbe, initial_delay_s: parseInt(e.target.value) || 0 })} className="border rounded px-2 py-1 w-full text-sm" /></div>
            <div><label className="block text-xs font-medium mb-1">Period (s)</label><input type="number" value={livenessProbe.period_s} onChange={(e) => setLivenessProbe({ ...livenessProbe, period_s: parseInt(e.target.value) || 0 })} className="border rounded px-2 py-1 w-full text-sm" /></div>
          </div>
        </div>
        <div className="bg-white rounded-lg p-6 shadow space-y-3">
          <h2 className="text-lg font-semibold">Readiness Probe</h2>
          <div className="grid grid-cols-2 gap-3">
            <div><label className="block text-xs font-medium mb-1">Path</label><input type="text" value={readinessProbe.path} onChange={(e) => setReadinessProbe({ ...readinessProbe, path: e.target.value })} className="border rounded px-2 py-1 w-full text-sm font-mono" /></div>
            <div><label className="block text-xs font-medium mb-1">Port</label><input type="number" value={readinessProbe.port} onChange={(e) => setReadinessProbe({ ...readinessProbe, port: parseInt(e.target.value) || 0 })} className="border rounded px-2 py-1 w-full text-sm" /></div>
            <div><label className="block text-xs font-medium mb-1">Initial Delay (s)</label><input type="number" value={readinessProbe.initial_delay_s} onChange={(e) => setReadinessProbe({ ...readinessProbe, initial_delay_s: parseInt(e.target.value) || 0 })} className="border rounded px-2 py-1 w-full text-sm" /></div>
            <div><label className="block text-xs font-medium mb-1">Period (s)</label><input type="number" value={readinessProbe.period_s} onChange={(e) => setReadinessProbe({ ...readinessProbe, period_s: parseInt(e.target.value) || 0 })} className="border rounded px-2 py-1 w-full text-sm" /></div>
          </div>
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">HPA Summary</h2>
        <div className="grid grid-cols-3 gap-4">
          <div className="text-center border rounded p-3"><div className="text-2xl font-bold">2-10</div><div className="text-xs text-gray-500">Replicas Range</div></div>
          <div className="text-center border rounded p-3"><div className="text-2xl font-bold text-blue-600">70%</div><div className="text-xs text-gray-500">CPU Target</div></div>
          <div className="text-center border rounded p-3"><div className="text-2xl font-bold text-green-600">3</div><div className="text-xs text-gray-500">Current Replicas</div></div>
        </div>
      </div>
    </div>
  );
}

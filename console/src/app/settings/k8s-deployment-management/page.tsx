'use client';
import { useState, useEffect } from 'react';
import { Loader2 } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface K8sDeployment {
  name: string;
  namespace: string;
  replicas: number;
  strategy: string;
  status: string;
  desiredReplicas: number;
}

const defaultDeployments: K8sDeployment[] = [
  { name: 'identity-service', namespace: 'ggid', replicas: 3, strategy: 'RollingUpdate', status: 'Healthy', desiredReplicas: 3 },
  { name: 'policy-service', namespace: 'ggid', replicas: 2, strategy: 'Recreate', status: 'Healthy', desiredReplicas: 2 },
  { name: 'audit-service', namespace: 'ggid', replicas: 1, strategy: 'RollingUpdate', status: 'Progressing', desiredReplicas: 2 },
  { name: 'notification-service', namespace: 'ggid', replicas: 2, strategy: 'RollingUpdate', status: 'Degraded', desiredReplicas: 2 },
];

export default function K8sDeploymentManagementPage() {

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const t = useTranslations();

  useEffect(() => {
    const fetchData = async () => {
      try {
        const res = await fetch("/api/v1/audit/metrics", {
          method: "GET",
          headers: {
            "Content-Type": "application/json",
            "X-Tenant-ID": "00000000-0000-0000-0000-000000000001",
          },
        });
        if (!res.ok) return null;
        const json = await res.json();
      } catch (e) {
        setError(e instanceof Error ? e.message : "Failed to load");
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, []);

  const [deployments, setDeployments] = useState<K8sDeployment[]>(defaultDeployments);
  if (loading) return <div className="flex items-center justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-blue-500" /></div>;
  if (error) return <div className="p-8 text-red-500">Error: {error}</div>;

  const scale = (name: string, delta: number) => {
    setDeployments(prev => prev.map(d => d.name === name ? { ...d, replicas: Math.max(0, d.replicas + delta), desiredReplicas: Math.max(0, d.desiredReplicas + delta) } : d));
  };

  const restart = (name: string) => {
    setDeployments(prev => prev.map(d => d.name === name ? { ...d, status: 'Progressing' } : d));
    setTimeout(() => setDeployments(prev => prev.map(d => d.name === name ? { ...d, status: 'Healthy' } : d)), 2000);
  };

  const statusClass = (status: string) => {
    switch (status) {
      case 'Healthy': return 'bg-green-100 text-green-700';
      case 'Progressing': return 'bg-amber-100 text-amber-700';
      case 'Degraded': return 'bg-red-100 text-red-700';
      default: return 'bg-gray-100 text-gray-700';
    }
  };

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <h1 className="text-2xl font-bold">{t("backend2.k8sDeploymentManagement.title")}</h1>
      <p className="text-gray-600">Scale, restart, and inspect Kubernetes deployments.</p>

      <div className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left border-b">
              <th scope="col" className="px-4 py-3">{"Deployment"}</th>
              <th scope="col" className="px-4 py-3">{"Namespace"}</th>
              <th scope="col" className="px-4 py-3">{"Replicas"}</th>
              <th scope="col" className="px-4 py-3">{"Strategy"}</th>
              <th scope="col" className="px-4 py-3">{"Status"}</th>
              <th scope="col" className="px-4 py-3">{t("backend2.k8sDeploymentManagement.actions")}</th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {deployments.map(d => (
              <tr key={d.name} className="hover:bg-gray-50">
                <td className="px-4 py-3 font-mono">{d.name}</td>
                <td className="px-4 py-3">{d.namespace}</td>
                <td className="px-4 py-3">{d.replicas}/{d.desiredReplicas}</td>
                <td className="px-4 py-3">{d.strategy}</td>
                <td className="px-4 py-3">
                  <span className={`px-2 py-1 rounded text-xs font-medium ${statusClass(d.status)}`}>{d.status}</span>
                </td>
                <td className="px-4 py-3 flex gap-2">
                  <button onClick={() => scale(d.name, 1)} className="px-2 py-1 text-xs border rounded">+1</button>
                  <button onClick={() => scale(d.name, -1)} className="px-2 py-1 text-xs border rounded">-1</button>
                  <button onClick={() => restart(d.name)} className="px-2 py-1 text-xs bg-blue-600 text-white rounded">{"Restart"}</button>
                  <button aria-label="action" className="px-2 py-1 text-xs border rounded">{"Edit"}</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

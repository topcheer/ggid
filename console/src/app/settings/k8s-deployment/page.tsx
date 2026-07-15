'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";

interface Deployment { serviceName: string; image: string; replicas: number; cpu: string; memory: string; status: string; }

const defaultDeployments: Deployment[] = [
  { serviceName: 'identity-service', image: 'ggid/identity:v1.2.3', replicas: 3, cpu: '500m', memory: '512Mi', status: 'Running' },
  { serviceName: 'policy-service', image: 'ggid/policy:v1.2.3', replicas: 2, cpu: '250m', memory: '256Mi', status: 'Running' },
  { serviceName: 'audit-service', image: 'ggid/audit:v1.2.3', replicas: 2, cpu: '500m', memory: '1Gi', status: 'Pending' },
];

export default function K8sDeploymentPage() {

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [data, setData] = useState<any[]>([]);

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
  if (!data || data.length === 0) return <div className="p-8 text-gray-500">{t("backend2.k8sDeployment.noData")}</div>;
  const [deployments, setDeployments] = useState<Deployment[]>(defaultDeployments);

  const addDeployment = () => {
    setDeployments(prev => [...prev, { serviceName: 'new-service', image: 'ggid/new:latest', replicas: 1, cpu: '100m', memory: '128Mi', status: 'Pending' }]);
  };

  const deleteDeployment = (idx: number) => {
    setDeployments(prev => prev.filter((_, i) => i !== idx));
  };

  const statusClass = (status: string) => {
    switch (status) {
      case 'Running': return 'bg-green-100 text-green-700';
      case 'Pending': return 'bg-amber-100 text-amber-700';
      case 'Failed': return 'bg-red-100 text-red-700';
      default: return 'bg-gray-100 text-gray-700';
    }
  };

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">{t("backend2.k8sDeployment.title")}</h1>
        <button onClick={addDeployment} className="px-3 py-1 bg-blue-600 text-white rounded text-sm">{t("backend2.k8sDeployment.addDeployment")}</button>
      </div>
      <p className="text-gray-600">Manage Kubernetes deployment manifests and resource settings.</p>

      <div className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left border-b">
              <th className="px-4 py-3">{t("backend2.k8sDeployment.serviceName")}</th>
              <th className="px-4 py-3">{t("backend2.k8sDeployment.image")}</th>
              <th className="px-4 py-3">{t("backend2.k8sDeployment.replicas")}</th>
              <th className="px-4 py-3">{t("backend2.k8sDeployment.cpu")}</th>
              <th className="px-4 py-3">{t("backend2.k8sDeployment.memory")}</th>
              <th className="px-4 py-3">{t("backend2.k8sDeployment.status")}</th>
              <th className="px-4 py-3">{t("backend2.k8sDeployment.actions")}</th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {deployments.map((d, idx) => (
              <tr key={idx} className="hover:bg-gray-50">
                <td className="px-4 py-3 font-mono">{d.serviceName}</td>
                <td className="px-4 py-3 font-mono text-xs">{d.image}</td>
                <td className="px-4 py-3">{d.replicas}</td>
                <td className="px-4 py-3">{d.cpu}</td>
                <td className="px-4 py-3">{d.memory}</td>
                <td className="px-4 py-3">
                  <span className={`px-2 py-1 rounded text-xs font-medium ${statusClass(d.status)}`}>{d.status}</span>
                </td>
                <td className="px-4 py-3 flex gap-2">
                  <button className="text-xs text-blue-600 hover:underline">{t("backend2.k8sDeployment.edit")}</button>
                  <button onClick={() => deleteDeployment(idx)} className="text-xs text-red-600 hover:underline">{t("backend2.k8sDeployment.delete")}</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

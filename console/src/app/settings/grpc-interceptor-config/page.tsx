'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";

interface Interceptor { id: string; type: string; enabled: boolean; order: number; config: string; }

export default function GrpcInterceptorConfigPage() {

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
  if (!data || data.length === 0) return <div className="p-8 text-gray-500">{t("backend2.grpcInterceptor.noData")}</div>;
  const [serviceName, setServiceName] = useState('identity.v1.IdentityService');
  const [interceptors, setInterceptors] = useState<Interceptor[]>([
    { id: 'auth', type: 'AuthInterceptor', enabled: true, order: 1, config: 'validate access token' },
    { id: 'log', type: 'LoggingInterceptor', enabled: true, order: 2, config: 'log all unary calls' },
    { id: 'metric', type: 'MetricsInterceptor', enabled: false, order: 3, config: 'emit grpc_server metrics' },
  ]);

  const toggleInterceptor = (id: string) => {
    setInterceptors(prev => prev.map(i => i.id === id ? { ...i, enabled: !i.enabled } : i));
  };

  const updateOrder = (id: string, order: string) => {
    setInterceptors(prev => prev.map(i => i.id === id ? { ...i, order: parseInt(order) || 0 } : i));
  };

  const addInterceptor = () => {
    const id = `i${interceptors.length + 1}`;
    setInterceptors(prev => [...prev, { id, type: 'CustomInterceptor', enabled: true, order: prev.length + 1, config: '' }]);
  };

  const deleteInterceptor = (id: string) => {
    setInterceptors(prev => prev.filter(i => i.id !== id));
  };

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      <h1 className="text-2xl font-bold">{t("backend2.grpcInterceptor.title")}</h1>
      <p className="text-gray-600">Configure unary and streaming interceptors for gRPC services.</p>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{"Service Name"}</h2>
        <input
          type="text"
          value={serviceName}
          onChange={e => setServiceName(e.target.value)}
          className="w-full border rounded px-3 py-2 text-sm font-mono"
        />
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">{"Interceptors"}</h2>
          <button onClick={addInterceptor} className="px-3 py-1 bg-blue-600 text-white rounded text-sm">{"Add Interceptor"}</button>
        </div>
        <div className="space-y-3">
          {interceptors.map(i => (
            <div key={i.id} className="border rounded p-4 space-y-3">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <input
                    type="checkbox"
                    checked={i.enabled}
                    onChange={() => toggleInterceptor(i.id)}
                    className="w-4 h-4"
                  />
                  <span className="font-mono font-medium">{i.type}</span>
                </div>
                <button onClick={() => deleteInterceptor(i.id)} className="text-xs text-red-600 hover:underline">{"Delete"}</button>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-1">
                  <label className="text-xs text-gray-500">{"Type"}</label>
                  <input
                    type="text"
                    value={i.type}
                    onChange={e => setInterceptors(prev => prev.map(x => x.id === i.id ? { ...x, type: e.target.value } : x))}
                    className="w-full border rounded px-2 py-1 text-sm font-mono"
                  />
                </div>
                <div className="space-y-1">
                  <label className="text-xs text-gray-500">{t("backend2.grpcInterceptor.order")}</label>
                  <input
                    type="number"
                    value={i.order}
                    onChange={e => updateOrder(i.id, e.target.value)}
                    className="w-full border rounded px-2 py-1 text-sm"
                  />
                </div>
              </div>
              <div className="space-y-1">
                <label className="text-xs text-gray-500">{t("backend2.grpcInterceptor.enabled")}</label>
                <div className="text-sm">{i.enabled ? 'Yes' : 'No'}</div>
              </div>
            </div>
          ))}
        </div>
      </section>
    </div>
  );
}

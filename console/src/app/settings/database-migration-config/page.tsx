'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";

interface Migration { id: string; name: string; status: 'pending' | 'applied' | 'failed' | 'rolled_back'; appliedAt: string; duration: string; }

const defaultMigrations: Migration[] = [
  { id: '001', name: 'create_users_table', status: 'applied', appliedAt: '2026-01-10 09:00', duration: '12ms' },
  { id: '002', name: 'create_roles_table', status: 'applied', appliedAt: '2026-01-10 09:01', duration: '8ms' },
  { id: '003', name: 'add_mfa_columns', status: 'applied', appliedAt: '2026-03-15 11:20', duration: '22ms' },
  { id: '004', name: 'add_audit_index', status: 'pending', appliedAt: '-', duration: '-' },
  { id: '005', name: 'seed_default_policies', status: 'failed', appliedAt: '2026-05-01 14:00', duration: '45ms' },
];

export default function DatabaseMigrationConfigPage() {

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
        if (!res.ok) return null;
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
  if (!data || data.length === 0) return <div className="p-8 text-gray-500">{t("backend2.dbMigration.noData")}</div>;
  const [migrations, setMigrations] = useState<Migration[]>(defaultMigrations);

  const runMigration = (id: string) => {
    setMigrations(prev => prev.map(m => m.id === id ? { ...m, status: 'applied', appliedAt: new Date().toLocaleString(), duration: '10ms' } : m));
  };

  const rollbackMigration = (id: string) => {
    setMigrations(prev => prev.map(m => m.id === id ? { ...m, status: 'rolled_back', appliedAt: '-', duration: '-' } : m));
  };

  const statusClass = (status: string) => {
    switch (status) {
      case 'applied': return 'bg-green-100 text-green-700';
      case 'pending': return 'bg-blue-100 text-blue-700';
      case 'failed': return 'bg-red-100 text-red-700';
      case 'rolled_back': return 'bg-gray-100 text-gray-700';
      default: return 'bg-gray-100 text-gray-700';
    }
  };

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <h1 className="text-2xl font-bold">{t("backend2.dbMigration.title")}</h1>
      <p className="text-gray-600">View and execute database migrations for the platform.</p>

      <div className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left border-b">
              <th className="px-4 py-3">ID</th>
              <th className="px-4 py-3">{"Migration"}</th>
              <th className="px-4 py-3">{"Status"}</th>
              <th className="px-4 py-3">{"Applied At"}</th>
              <th className="px-4 py-3">{"Duration"}</th>
              <th className="px-4 py-3">{"Actions"}</th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {migrations.map(m => (
              <tr key={m.id} className="hover:bg-gray-50">
                <td className="px-4 py-3 font-mono">{m.id}</td>
                <td className="px-4 py-3 font-mono">{m.name}</td>
                <td className="px-4 py-3">
                  <span className={`px-2 py-1 rounded text-xs font-medium ${statusClass(m.status)}`}>{m.status}</span>
                </td>
                <td className="px-4 py-3 text-gray-500">{m.appliedAt}</td>
                <td className="px-4 py-3">{m.duration}</td>
                <td className="px-4 py-3 flex gap-2">
                  {m.status === 'pending' || m.status === 'failed' ? (
                    <button onClick={() => runMigration(m.id)} className="px-2 py-1 text-xs bg-blue-600 text-white rounded">{"Run"}</button>
                  ) : null}
                  {m.status === 'applied' ? (
                    <button onClick={() => rollbackMigration(m.id)} className="px-2 py-1 text-xs border rounded">{"Rollback"}</button>
                  ) : null}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

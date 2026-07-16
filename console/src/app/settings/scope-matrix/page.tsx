"use client";
import { useTranslations } from "@/lib/i18n";

import { useState, useEffect, useCallback } from "react";
import { Grid3x3, Check, Save } from "lucide-react";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface MatrixData {
  clients: { client_id: string; client_name: string }[];
  scopes: string[];
  grants: Record<string, Record<string, boolean>>;
  usage: Record<string, Record<string, number>>;
}

export default function ScopeMatrixPage() {
  const t = useTranslations();
  const [data, setData] = useState<MatrixData | null>(null);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [grants, setGrants] = useState<Record<string, Record<string, boolean>>>({});

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/oauth/scope-matrix", { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const json = await res.json();
        setData(json);
        setGrants(json.grants || {});
      }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const toggleGrant = (clientId: string, scope: string) => {
    setGrants((prev) => ({
      ...prev,
      [clientId]: { ...prev[clientId], [scope]: !prev[clientId]?.[scope] },
    }));
  };

  const save = async () => {
    setSaving(true);
    try {
      await fetch("/api/v1/oauth/scope-matrix", {
        method: "PUT",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
        body: JSON.stringify({ grants }),
      });
    } catch { /* noop */ }
    finally { setSaving(false); }
  };

  if (!data || loading) return <p className="text-sm text-gray-500 text-center py-8">Loading...</p>;

  const maxUsage = Math.max(...data.clients.flatMap((c) => data.scopes.map((s) => data.usage[c.client_id]?.[s] || 0)), 1);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2"><Grid3x3 className="w-6 h-6 text-blue-500" />{t("scopeMatrix.title")}</h1>
          <p className="text-sm text-gray-500 mt-1">Manage scope grants per client with usage analytics.</p>
        </div>
        <button aria-label="Save" onClick={save} disabled={saving} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50 flex items-center gap-2"><Save className="w-4 h-4" /> {saving ? "Saving..." : "Save Changes"}</button>
      </div>

      {/* Grid */}
      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 dark:bg-gray-900/50 sticky top-0">
            <tr>
              <th scope="col" className="px-4 py-3 text-left font-medium sticky left-0 bg-gray-50 dark:bg-gray-900/50">Client</th>
              {data.scopes.map((s) => (
                <th scope="col" key={s} className="px-3 py-3 text-center font-medium font-mono text-xs whitespace-nowrap">{s}</th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y dark:divide-gray-800">
            {data.clients.map((c) => (
              <tr key={c.client_id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                <td className="px-4 py-3 font-medium sticky left-0 bg-white dark:bg-gray-900 whitespace-nowrap">{c.client_name}</td>
                {data.scopes.map((s) => (
                  <td key={s} className="px-3 py-3 text-center">
                    <button onClick={() => toggleGrant(c.client_id, s)} className="w-6 h-6 rounded flex items-center justify-center mx-auto">
                      {grants[c.client_id]?.[s] ? (
                        <span className="w-5 h-5 rounded bg-green-500 text-white flex items-center justify-center"><Check className="w-3 h-3" /></span>
                      ) : (
                        <span className="w-5 h-5 rounded border-2 border-gray-200 dark:border-gray-700" />
                      )}
                    </button>
                    {(data.usage[c.client_id]?.[s] || 0) > 0 && (
                      <span className="text-xs text-gray-400 mt-0.5 block">{data.usage[c.client_id][s]}</span>
                    )}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Usage bar chart */}
      <div className="rounded-lg border dark:border-gray-800 p-4">
        <h3 className="font-semibold mb-4">Scope Usage by Client</h3>
        <div className="space-y-3">
          {data.clients.map((c) => {
            const totalUsage = data.scopes.reduce((sum, s) => sum + (data.usage[c.client_id]?.[s] || 0), 0);
            return (
              <div key={c.client_id}>
                <div className="flex items-center justify-between text-xs mb-1">
                  <span className="font-medium">{c.client_name}</span>
                  <span className="text-gray-400">{totalUsage} total calls</span>
                </div>
                <div className="flex items-end gap-0.5 h-8">
                  {data.scopes.map((s) => {
                    const usage = data.usage[c.client_id]?.[s] || 0;
                    return <div key={s} className="flex-1 bg-blue-500 rounded-t" style={{ height: `${(usage / maxUsage) * 100}%`, minHeight: usage > 0 ? "3px" : "0" }} title={`${s}: ${usage}`} />;
                  })}
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}

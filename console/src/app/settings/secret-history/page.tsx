"use client";

import { useState, useCallback } from "react";
import { KeyRound, Clock, GitCompare } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface RotationEntry {
  id: string;
  rotated_at: string;
  rotated_by: string;
  thumbprint: string;
  age_days: number;
}

interface SecretHistory {
  client_id: string;
  client_name: string;
  current: RotationEntry;
  previous: RotationEntry | null;
  rotation_log: RotationEntry[];
}

interface Client { client_id: string; client_name: string; }

export default function SecretHistoryPage() {
  const t = useTranslations();

  const [clients] = useState<Client[]>([{ client_id: "c1", client_name: "Web App" }, { client_id: "c2", client_name: "Mobile App" }, { client_id: "c3", client_name: "API Service" }]);
  const [clientId, setClientId] = useState("");
  const [data, setData] = useState<SecretHistory | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchHistory = useCallback(async () => {
    if (!clientId) return;
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/oauth/secret-history?client_id=${encodeURIComponent(clientId)}`, { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, [clientId]);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><KeyRound className="w-6 h-6 text-yellow-500" /> {t("secretHistory.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Track client secret rotation history with thumbprint verification.</p>
      </div>

      <select value={clientId} onChange={(e) => setClientId(e.target.value)} className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm">
        <option value="">Select Client</option>
        {clients.map((c) => <option key={c.client_id} value={c.client_id}>{c.client_name}</option>)}
      </select>

      {data && (
        <>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="rounded-lg border border-green-300 dark:border-green-800 bg-green-50 dark:bg-green-900/20 p-4">
              <h3 className="text-sm font-semibold text-green-700 dark:text-green-400 mb-2">Current Secret</h3>
              <div className="space-y-1 text-sm"><div className="flex items-center gap-2"><Clock className="w-3.5 h-3.5 text-gray-400" /><span className="text-gray-500">Rotated:</span><span className="font-medium">{data.current.rotated_at}</span></div><div><span className="text-gray-500">By:</span><span className="font-mono text-xs ml-1">{data.current.rotated_by}</span></div><div><span className="text-gray-500">Thumbprint:</span><span className="font-mono text-xs ml-1">{data.current.thumbprint}</span></div><div><span className="text-gray-500">Age:</span><span className="font-bold ml-1">{data.current.age_days} days</span></div></div>
            </div>
            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="text-sm font-semibold text-gray-500 mb-2">Previous Secret</h3>
              {data.previous ? (<div className="space-y-1 text-sm"><div className="flex items-center gap-2"><Clock className="w-3.5 h-3.5 text-gray-400" /><span className="text-gray-500">Rotated:</span><span className="font-medium">{data.previous.rotated_at}</span></div><div><span className="text-gray-500">Thumbprint:</span><span className="font-mono text-xs ml-1">{data.previous.thumbprint}</span></div><div><span className="text-gray-500">Age:</span><span className="font-bold ml-1">{data.previous.age_days} days</span></div></div>) : <p className="text-xs text-gray-400">No previous secret.</p>}
            </div>
          </div>

          <div className="rounded-lg border dark:border-gray-800 p-4">
            <h3 className="text-sm font-semibold flex items-center gap-2 mb-3"><GitCompare className="w-4 h-4 text-gray-400" /> Rotation Timeline</h3>
            <div className="relative pl-6">
              <div className="absolute left-2 top-0 bottom-0 w-px bg-gray-200 dark:bg-gray-800" />
              <div className="space-y-3">{data.rotation_log.map((r) => (
                <div key={r.id} className="relative">
                  <div className="absolute -left-4 w-3 h-3 rounded-full bg-yellow-500 border-2 border-yellow-200" />
                  <div className="ml-2 text-sm"><div className="flex items-center justify-between"><span className="font-medium">{r.rotated_at}</span><span className="text-xs text-gray-400">{r.age_days}d ago</span></div><div className="text-xs text-gray-500 mt-0.5">By {r.rotated_by} - <span className="font-mono">{r.thumbprint}</span></div></div>
                </div>
              ))}{data.rotation_log.length === 0 && <p className="text-xs text-gray-400">No rotations recorded.</p>}</div>
            </div>
          </div>
        </>
      )}
      {!data && !loading && clientId && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
      {!clientId && <p className="text-sm text-gray-500 text-center py-8">Select a client to view secret history.</p>}
    </div>
  );
}

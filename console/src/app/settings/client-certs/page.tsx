"use client";

import React, { useState } from "react";
import { useApi } from "@/lib/api";
import {
  KeyRound, Loader2, AlertCircle, X, RefreshCw, AlertOctagon, ToggleLeft, ToggleRight,
} from "lucide-react";

interface ClientCert {
  id: string;
  client_id: string;
  client_name: string;
  cert_serial: string;
  issuer: string;
  subject: string;
  fingerprint: string;
  issued_at: string;
  expires_at: string;
  status: "active" | "expired" | "revoked" | "pending_rotation";
  auto_rotate: boolean;
}

const statusColors: Record<string, string> = {
  active: "text-green-600 bg-green-100 dark:bg-green-900/30 dark:text-green-400",
  expired: "text-red-600 bg-red-100 dark:bg-red-900/30 dark:text-red-400",
  revoked: "text-gray-600 bg-gray-100 dark:bg-gray-700 dark:text-gray-400",
  pending_rotation: "text-yellow-600 bg-yellow-100 dark:bg-yellow-900/30 dark:text-yellow-400",
};

export default function ClientCertsPage() {
  const { apiFetch } = useApi();
  const [certs, setCerts] = useState<ClientCert[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [rotating, setRotating] = useState<string | null>(null);
  const [toggling, setToggling] = useState<string | null>(null);

  useState(() => {
    (async () => {
      try { setCerts(await apiFetch<ClientCert[]>("/api/v1/oauth/client-certs").catch(() => [])); }
      catch { setError("Failed to load certificates"); }
      finally { setLoading(false); }
    })();
  });

  const handleRotate = async (certId: string) => {
    setRotating(certId);
    try { await apiFetch(`/api/v1/oauth/client-certs/${certId}/rotate`, { method: "POST" }); setCerts(await apiFetch<ClientCert[]>("/api/v1/oauth/client-certs").catch(() => certs)); }
    catch { setError("Rotation failed"); }
    finally { setRotating(null); }
  };

  const handleToggleAuto = async (certId: string) => {
    setToggling(certId);
    try { await apiFetch(`/api/v1/oauth/client-certs/${certId}`, { method: "PATCH", body: JSON.stringify({ auto_rotate: !certs.find((c) => c.id === certId)?.auto_rotate }) }); setCerts((p) => p.map((c) => c.id === certId ? { ...c, auto_rotate: !c.auto_rotate } : c)); }
    catch { setError("Toggle failed"); }
    finally { setToggling(null); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const expired = certs.filter((c) => c.status === "expired" || (c.expires_at && new Date(c.expires_at) < new Date()));
  const expiringSoon = certs.filter((c) => c.status === "active" && c.expires_at && new Date(c.expires_at).getTime() - Date.now() < 30 * 86400000);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><KeyRound className="h-6 w-6 text-indigo-600" /> Client Certificates</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">OAuth client certificate lifecycle: status monitoring, rotation, and auto-rotate.</p>
      </div>

      {/* Expired alert */}
      {expired.length > 0 && (
        <div className="flex items-center gap-3 rounded-xl border border-red-200 bg-red-50 px-4 py-3 dark:border-red-800 dark:bg-red-900/20"><AlertOctagon className="h-5 w-5 text-red-600 shrink-0" /><div><span className="font-medium text-red-700 dark:text-red-400">{expired.length} expired certificate{expired.length > 1 ? "s" : ""}</span><p className="text-sm text-red-600 dark:text-red-400">Rotate expired certificates immediately to maintain mTLS connectivity.</p></div></div>
      )}
      {expiringSoon.length > 0 && (
        <div className="flex items-center gap-3 rounded-xl border border-yellow-200 bg-yellow-50 px-4 py-3 dark:border-yellow-800 dark:bg-yellow-900/20"><AlertCircle className="h-5 w-5 text-yellow-600 shrink-0" /><div><span className="font-medium text-yellow-700 dark:text-yellow-400">{expiringSoon.length} certificate{expiringSoon.length > 1 ? "s" : ""} expiring within 30 days</span></div></div>
      )}

      {error && <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>
      : certs.length === 0 ? (
        <div className={cardCls}><div className="py-12 text-center"><KeyRound className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No client certificates registered.</p></div></div>
      ) : (
        <div className="overflow-x-auto rounded-xl border border-gray-200 dark:border-gray-700">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 dark:bg-gray-800"><tr>
              <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Client</th>
              <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Serial</th>
              <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Issuer</th>
              <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Issued</th>
              <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Expires</th>
              <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Status</th>
              <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Auto-Rotate</th>
              <th className="px-4 py-3 text-right font-semibold text-gray-600 dark:text-gray-300">Actions</th>
            </tr></thead>
            <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
              {certs.map((c) => {
                const isExpired = c.status === "expired" || (c.expires_at && new Date(c.expires_at) < new Date());
                return (
                  <tr key={c.id} className={`bg-white dark:bg-gray-900 ${isExpired ? "opacity-60" : ""}`}>
                    <td className="px-4 py-3"><div className="font-medium text-gray-900 dark:text-white">{c.client_name}</div><div className="text-xs text-gray-400 font-mono">{c.client_id.slice(0, 16)}</div></td>
                    <td className="px-4 py-3 font-mono text-xs text-gray-500">{c.cert_serial.slice(0, 24)}</td>
                    <td className="px-4 py-3 text-gray-500">{c.issuer}</td>
                    <td className="px-4 py-3 text-gray-400">{c.issued_at ? new Date(c.issued_at).toLocaleDateString() : "—"}</td>
                    <td className="px-4 py-3"><span className={isExpired ? "text-red-500" : "text-gray-400"}>{c.expires_at ? new Date(c.expires_at).toLocaleDateString() : "—"}</span></td>
                    <td className="px-4 py-3"><span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${statusColors[c.status] || ""}`}>{c.status.replace(/_/g, " ")}</span></td>
                    <td className="px-4 py-3"><button onClick={() => handleToggleAuto(c.id)} disabled={toggling === c.id}>{toggling === c.id ? <Loader2 className="h-4 w-4 animate-spin" /> : c.auto_rotate ? <ToggleRight className="h-6 w-6 text-green-600" /> : <ToggleLeft className="h-6 w-6 text-gray-300" />}</button></td>
                    <td className="px-4 py-3 text-right"><button onClick={() => handleRotate(c.id)} disabled={rotating === c.id} className="flex items-center gap-1 text-xs text-indigo-600 hover:underline">{rotating === c.id ? <Loader2 className="h-3 w-3 animate-spin" /> : <RefreshCw className="h-3 w-3" />} Rotate</button></td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

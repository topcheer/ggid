"use client";

import { useState, useEffect, useCallback } from "react";
import { HeartPulse, CheckCircle, XCircle, AlertTriangle, Clock, RefreshCw } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface SpHealth {
  sp_entity_id: string;
  metadata_url: string;
  metadata_valid: boolean;
  cert_expiry_days: number;
  cert_expires_at: string;
  response_test: "pass" | "fail" | "untested";
  acs_url: string;
  acs_status: "ok" | "error" | "unknown";
  slo_url: string;
  slo_status: "ok" | "error" | "unknown";
  idp_connected: boolean;
  last_sync: string;
  errors: { timestamp: string; message: string }[];
}

export default function SamlSpHealthPage() {
  const t = useTranslations();
  const [data, setData] = useState<SpHealth | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/oauth/saml-sp-health", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const certColor = data ? (data.cert_expiry_days <= 7 ? "text-red-600" : data.cert_expiry_days <= 30 ? "text-yellow-600" : "text-green-600") : "";

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold flex items-center gap-2"><HeartPulse className="w-6 h-6 text-red-500" /> SAML SP Health</h1><p className="text-sm text-gray-500 mt-1">Monitor SAML Service Provider metadata, certificates, and connectivity.</p></div>
        <button onClick={fetchData} className="px-3 py-2 rounded-lg border dark:border-gray-700 text-sm flex items-center gap-2"><RefreshCw className="w-4 h-4" /> Refresh</button>
      </div>

      {data && (
        <>
          <div className="rounded-lg border dark:border-gray-800 p-4 space-y-3">
            <div className="flex items-center justify-between"><div><span className="font-semibold">{data.sp_entity_id}</span><p className="text-xs text-gray-400 font-mono mt-0.5">{data.metadata_url}</p></div><span className={`px-2 py-1 rounded text-xs font-medium ${data.idp_connected ? "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400" : "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400"}`}>{data.idp_connected ? "IdP Connected" : "IdP Disconnected"}</span></div>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-3 pt-2">
              <div className="flex items-center gap-2 text-sm">{data.metadata_valid ? <CheckCircle className="w-5 h-5 text-green-500" /> : <XCircle className="w-5 h-5 text-red-500" />}<span>Metadata {data.metadata_valid ? "Valid" : "Invalid"}</span></div>
              <div className="flex items-center gap-2 text-sm"><Clock className={`w-5 h-5 ${certColor}`} /><span className={certColor}>Cert: {data.cert_expiry_days}d left</span></div>
              <div className="flex items-center gap-2 text-sm">{data.acs_status === "ok" ? <CheckCircle className="w-5 h-5 text-green-500" /> : <XCircle className="w-5 h-5 text-red-500" />}<span>ACS {data.acs_status}</span></div>
              <div className="flex items-center gap-2 text-sm">{data.slo_status === "ok" ? <CheckCircle className="w-5 h-5 text-green-500" /> : <XCircle className="w-5 h-5 text-red-500" />}<span>SLO {data.slo_status}</span></div>
            </div>
            <div className="text-xs text-gray-400">Last sync: {data.last_sync} | Response test: <span className={data.response_test === "pass" ? "text-green-600" : data.response_test === "fail" ? "text-red-600" : ""}>{data.response_test}</span></div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-2">ACS URL</h3><p className="font-mono text-xs text-gray-500 break-all">{data.acs_url}</p></div>
            <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-2">SLO URL</h3><p className="font-mono text-xs text-gray-500 break-all">{data.slo_url}</p></div>
          </div>

          {data.errors.length > 0 && (
            <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold flex items-center gap-2 mb-3"><AlertTriangle className="w-4 h-4 text-red-500" /> Recent Errors</h3><div className="space-y-1">{data.errors.map((e, i) => (<div key={i} className="flex items-center gap-2 text-sm"><span className="text-xs text-gray-400">{e.timestamp}</span><span className="text-red-600">{e.message}</span></div>))}</div></div>
          )}
        </>
      )}
      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
    </div>
  );
}

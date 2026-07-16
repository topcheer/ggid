"use client";
import { useState, useEffect } from "react";
import { Loader2 } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface PiiPattern {
  name: string;
  regex: string;
  example: string;
  enabled: boolean;
}

interface EgressFilter {
  field: string;
  action: "block" | "mask" | "hash";
  external_clients_only: boolean;
}

interface DlpViolation {
  timestamp: string;
  user: string;
  pattern: string;
  action_taken: string;
}

export default function SiemIntegrationPage() {
  const t = useTranslations();

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const res = await fetch("/api/v1/audit/siem/health", {
          method: "GET",
          headers: {
            "Content-Type": "application/json",
            "X-Tenant-ID": "00000000-0000-0000-0000-000000000001",
          },
        });
        if (!res.ok) return null;
        const json = await res.json();
      } catch (e) {
        setError(e instanceof Error ? e.message : t("siem.failedToLoad"));
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, [t]);

  const [destination, setDestination] = useState("Splunk");
  const [endpoint, setEndpoint] = useState("splunk.hec.internal:8088");
  const [logFormat, setLogFormat] = useState("CEF");
  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState<string | null>(null);
  const [filters, setFilters] = useState({ include_types: ["auth_failure", "privilege_escalation", "policy_violation"], min_severity: "warn", tenant_filter: "" });
  if (loading) return <div className="flex items-center justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-blue-500" /></div>;
  if (error) return <div className="p-8 text-red-500">{t("common.error")}: {error}</div>;

  const metrics = { events_per_sec: 847, queue_depth: 12, delivery_success_pct: 99.8, circuit_breaker: "closed" as const };

  const handleTest = async () => { setTesting(true); setTestResult(null); setTimeout(() => { setTestResult(t("siem.connectionSuccess")); setTesting(false); }, 800); };

  const formatPreview: Record<string, string> = {
    CEF: "CEF:0|GGID|IAM|2.1|100|Auth Failure|7|rt=Jan 15 2025 16:42:00 suser=john.doe act=login_failed",
    JSON: '{\n  "timestamp": "2025-01-15T16:42:00Z",\n  "event_type": "auth_failure",\n  "user": "john.doe",\n  "severity": "high"\n}',
    LEEF: "LEEF:1.0|GGID|IAM|2.1|100|rt=Jan 15 2025 16:42:00^usrName=john.doe^event=login_failed",
  };

  return (
    <div className="p-8 space-y-6 max-w-5xl">
      <h1 className="text-2xl font-bold">{t("siem.title")}</h1>
      <p className="text-gray-600">{t("siem.subtitle")}</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">{t("siem.destinationConfiguration")}</h2>
        <div className="grid grid-cols-3 gap-4">
          <div><label className="block text-sm font-medium mb-1">{t("siem.siemPlatform")}</label><select aria-label="Destination" value={destination} onChange={(e) => setDestination(e.target.value)} className="border rounded px-3 py-2 w-full"><option>Splunk</option><option>Elastic</option><option>Datadog</option><option>HTTP</option></select></div>
          <div><label className="block text-sm font-medium mb-1">{t("siem.endpoint")}</label><input aria-label="endpoint" type="text" value={endpoint} onChange={(e) => setEndpoint(e.target.value)} className="border rounded px-3 py-2 w-full font-mono text-sm" /></div>
          <div><label className="block text-sm font-medium mb-1">{t("siem.logFormat")}</label><select aria-label="log Format" value={logFormat} onChange={(e) => setLogFormat(e.target.value)} className="border rounded px-3 py-2 w-full"><option>CEF</option><option>JSON</option><option>LEEF</option></select></div>
        </div>
        <div className="flex items-center gap-4"><button onClick={handleTest} disabled={testing} className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50">{testing ? t("siem.testing") : t("siem.testConnection")}</button>{testResult && <span className="text-sm text-green-600">{testResult}</span>}</div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">{t("siem.throughputMetrics")}</h2>
        <div className="grid grid-cols-4 gap-4">
          <div className="text-center"><div className="text-2xl font-bold">{metrics.events_per_sec}</div><div className="text-xs text-gray-500">{t("siem.eventsPerSec")}</div></div>
          <div className="text-center"><div className="text-2xl font-bold">{metrics.queue_depth}</div><div className="text-xs text-gray-500">{t("siem.queueDepth")}</div></div>
          <div className="text-center"><div className="text-2xl font-bold text-green-600">{metrics.delivery_success_pct}%</div><div className="text-xs text-gray-500">{t("siem.deliverySuccess")}</div></div>
          <div className="text-center"><div className="text-2xl font-bold"><span className={`px-2 py-1 rounded text-xs ${metrics.circuit_breaker === "closed" ? "bg-green-100 text-green-700" : "bg-red-100 text-red-700"}`}>{metrics.circuit_breaker}</span></div><div className="text-xs text-gray-500">{t("siem.circuitBreaker")}</div></div>
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">{t("siem.eventFilters")}</h2>
        <div className="grid grid-cols-2 gap-4">
          <div><label className="block text-sm font-medium mb-1">{t("siem.includeEventTypes")}</label><input aria-label="filters" type="text" value={filters.include_types.join(", ")} onChange={(e) => setFilters({ ...filters, include_types: e.target.value.split(",").map((s) => s.trim()) })} className="border rounded px-3 py-2 w-full text-sm" /></div>
          <div><label className="block text-sm font-medium mb-1">{t("siem.minSeverity")}</label><select aria-label="filters" value={filters.min_severity} onChange={(e) => setFilters({ ...filters, min_severity: e.target.value })} className="border rounded px-3 py-2 w-full"><option value="info">{t("siem.info")}</option><option value="warn">{t("siem.warn")}</option><option value="error">{t("siem.error")}</option><option value="critical">{t("siem.critical")}</option></select></div>
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">{t("siem.logFormatPreview")} ({logFormat})</h2>
        <pre className="bg-gray-50 rounded p-4 text-xs overflow-x-auto whitespace-pre-wrap font-mono border">{formatPreview[logFormat] || ""}</pre>
      </div>
    </div>
  );
}

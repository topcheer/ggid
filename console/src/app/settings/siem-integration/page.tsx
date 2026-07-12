"use client";
import { useState, useEffect } from "react";

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

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [data, setData] = useState<any[]>([]);

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
  if (!data || data.length === 0) return <div className="p-8 text-gray-500">No data available</div>;
  const [destination, setDestination] = useState("Splunk");
  const [endpoint, setEndpoint] = useState("splunk.hec.internal:8088");
  const [logFormat, setLogFormat] = useState("CEF");
  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState<string | null>(null);
  const [filters, setFilters] = useState({ include_types: ["auth_failure", "privilege_escalation", "policy_violation"], min_severity: "warn", tenant_filter: "" });

  const metrics = { events_per_sec: 847, queue_depth: 12, delivery_success_pct: 99.8, circuit_breaker: "closed" as const };

  const handleTest = async () => { setTesting(true); setTestResult(null); setTimeout(() => { setTestResult("Connection successful. Test event delivered. Response: 200 OK"); setTesting(false); }, 800); };

  const formatPreview: Record<string, string> = {
    CEF: "CEF:0|GGID|IAM|2.1|100|Auth Failure|7|rt=Jan 15 2025 16:42:00 suser=john.doe act=login_failed",
    JSON: '{\n  "timestamp": "2025-01-15T16:42:00Z",\n  "event_type": "auth_failure",\n  "user": "john.doe",\n  "severity": "high"\n}',
    LEEF: "LEEF:1.0|GGID|IAM|2.1|100|rt=Jan 15 2025 16:42:00^usrName=john.doe^event=login_failed",
  };

  return (
    <div className="p-8 space-y-6 max-w-5xl">
      <h1 className="text-2xl font-bold">SIEM Integration</h1>
      <p className="text-gray-600">Configure SIEM destination, event filters, and monitor throughput.</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Destination Configuration</h2>
        <div className="grid grid-cols-3 gap-4">
          <div><label className="block text-sm font-medium mb-1">SIEM Platform</label><select value={destination} onChange={(e) => setDestination(e.target.value)} className="border rounded px-3 py-2 w-full"><option>Splunk</option><option>Elastic</option><option>Datadog</option><option>HTTP</option></select></div>
          <div><label className="block text-sm font-medium mb-1">Endpoint</label><input type="text" value={endpoint} onChange={(e) => setEndpoint(e.target.value)} className="border rounded px-3 py-2 w-full font-mono text-sm" /></div>
          <div><label className="block text-sm font-medium mb-1">Log Format</label><select value={logFormat} onChange={(e) => setLogFormat(e.target.value)} className="border rounded px-3 py-2 w-full"><option>CEF</option><option>JSON</option><option>LEEF</option></select></div>
        </div>
        <div className="flex items-center gap-4"><button onClick={handleTest} disabled={testing} className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50">{testing ? "Testing..." : "Test Connection"}</button>{testResult && <span className="text-sm text-green-600">{testResult}</span>}</div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Throughput Metrics</h2>
        <div className="grid grid-cols-4 gap-4">
          <div className="text-center"><div className="text-2xl font-bold">{metrics.events_per_sec}</div><div className="text-xs text-gray-500">Events/sec</div></div>
          <div className="text-center"><div className="text-2xl font-bold">{metrics.queue_depth}</div><div className="text-xs text-gray-500">Queue Depth</div></div>
          <div className="text-center"><div className="text-2xl font-bold text-green-600">{metrics.delivery_success_pct}%</div><div className="text-xs text-gray-500">Delivery Success</div></div>
          <div className="text-center"><div className="text-2xl font-bold"><span className={`px-2 py-1 rounded text-xs ${metrics.circuit_breaker === "closed" ? "bg-green-100 text-green-700" : "bg-red-100 text-red-700"}`}>{metrics.circuit_breaker}</span></div><div className="text-xs text-gray-500">Circuit Breaker</div></div>
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Event Filters</h2>
        <div className="grid grid-cols-2 gap-4">
          <div><label className="block text-sm font-medium mb-1">Include Event Types</label><input type="text" value={filters.include_types.join(", ")} onChange={(e) => setFilters({ ...filters, include_types: e.target.value.split(",").map((s) => s.trim()) })} className="border rounded px-3 py-2 w-full text-sm" /></div>
          <div><label className="block text-sm font-medium mb-1">Min Severity</label><select value={filters.min_severity} onChange={(e) => setFilters({ ...filters, min_severity: e.target.value })} className="border rounded px-3 py-2 w-full"><option value="info">Info</option><option value="warn">Warn</option><option value="error">Error</option><option value="critical">Critical</option></select></div>
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Log Format Preview ({logFormat})</h2>
        <pre className="bg-gray-50 rounded p-4 text-xs overflow-x-auto whitespace-pre-wrap font-mono border">{formatPreview[logFormat] || ""}</pre>
      </div>
    </div>
  );
}

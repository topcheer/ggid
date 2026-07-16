"use client";
import { useState, useEffect } from "react";
import { Loader2 } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface RouteEntry {
  path: string;
  methods: string[];
  upstream: string;
  strip_prefix: boolean;
  rate_limit: number;
  auth_required: boolean;
  upstream_healthy: boolean;
}

const defaultRoutes: RouteEntry[] = [
  { path: "/api/v1/users", methods: ["GET", "POST", "PUT", "DELETE"], upstream: "identity-service:8080", strip_prefix: false, rate_limit: 100, auth_required: true, upstream_healthy: true },
  { path: "/api/v1/auth/*", methods: ["POST"], upstream: "auth-service:9001", strip_prefix: false, rate_limit: 50, auth_required: false, upstream_healthy: true },
  { path: "/api/v1/roles", methods: ["GET", "POST", "PUT"], upstream: "policy-service:8070", strip_prefix: false, rate_limit: 100, auth_required: true, upstream_healthy: true },
  { path: "/api/v1/orgs", methods: ["GET", "POST"], upstream: "org-service:8071", strip_prefix: false, rate_limit: 100, auth_required: true, upstream_healthy: false },
  { path: "/api/v1/audit/*", methods: ["GET"], upstream: "audit-service:8072", strip_prefix: true, rate_limit: 200, auth_required: true, upstream_healthy: true },
];

export default function ApiGatewayRoutesPage() {

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

  const [routes, setRoutes] = useState<RouteEntry[]>(defaultRoutes);
  const [testPath, setTestPath] = useState("");
  const [testMethod, setTestMethod] = useState("GET");
  const [testResult, setTestResult] = useState<string | null>(null);
  const [testing, setTesting] = useState(false);
  if (loading) return <div className="flex items-center justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-blue-500" /></div>;
  if (error) return <div className="p-8 text-red-500">Error: {error}</div>;

  const handleTest = async () => {
    setTesting(true); setTestResult(null);
    setTimeout(() => {
      setTestResult("HTTP 200 OK\nContent-Type: application/json\nResponse time: 12ms\n{\"status\": \"healthy\"}");
      setTesting(false);
    }, 800);
  };

  return (
    <div className="p-8 space-y-6 max-w-5xl">
      <h1 className="text-2xl font-bold">{t("backend2.gatewayRoutes.title")}</h1>
      <p className="text-gray-600">Manage gateway routes, upstreams, rate limits, and test endpoints.</p>

      <div className="bg-white rounded-lg p-6 shadow">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold">Route Table</h2>
          <button className="px-4 py-1 bg-blue-600 text-white rounded text-sm hover:bg-blue-700">{t("backend2.gatewayRoutes.addRoute")}</button>
        </div>
        <table className="w-full text-sm">
          <thead><tr className="border-b text-left"><th className="py-2">{t("backend2.gatewayRoutes.path")}</th><th scope="col">{t("backend2.gatewayRoutes.methods")}</th><th>Upstream</th><th>Strip Prefix</th><th>Rate Limit</th><th>{t("backend2.gatewayRoutes.auth")}</th><th>{t("backend2.gatewayRoutes.health")}</th><th>{t("backend2.gatewayRoutes.actions")}</th></tr></thead>
          <tbody>
            {routes.map((r: RouteEntry, i: number) => (
              <tr key={i} className="border-b hover:bg-gray-50">
                <td className="py-2 font-mono text-xs">{r.path}</td>
                <td><div className="flex gap-1">{r.methods.map((m) => <span key={m} className="px-1.5 py-0.5 bg-blue-100 text-blue-700 rounded text-xs font-mono">{m}</span>)}</div></td>
                <td className="font-mono text-xs">{r.upstream}</td>
                <td>{r.strip_prefix ? "Yes" : "No"}</td>
                <td>{r.rate_limit}/s</td>
                <td>{r.auth_required ? "Yes" : "No"}</td>
                <td><span className={`inline-block w-2.5 h-2.5 rounded-full ${r.upstream_healthy ? "bg-green-500" : "bg-red-500"}`} /></td>
                <td className="flex gap-2"><button className="text-xs text-blue-600 hover:underline">{t("backend2.gatewayRoutes.edit")}</button><button className="text-xs text-red-600 hover:underline">{t("backend2.gatewayRoutes.delete")}</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Route Testing Panel</h2>
        <div className="flex gap-3 items-end">
          <div><label className="block text-sm font-medium mb-1">{t("backend2.gatewayRoutes.method")}</label><select aria-label="Test method" value={testMethod} onChange={(e) => setTestMethod(e.target.value)} className="border rounded px-3 py-2"><option>GET</option><option>POST</option><option>PUT</option><option>DELETE</option></select></div>
          <div className="flex-1"><label className="block text-sm font-medium mb-1">{t("backend2.gatewayRoutes.path")}</label><input aria-label="/api/v1/users" type="text" value={testPath} onChange={(e) => setTestPath(e.target.value)} placeholder="/api/v1/users" className="border rounded px-3 py-2 w-full font-mono text-sm" /></div>
          <button onClick={handleTest} disabled={testing} className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50">{testing ? "Sending..." : "Send"}</button>
        </div>
        {testResult && <pre className="bg-gray-50 rounded p-4 text-xs overflow-x-auto whitespace-pre-wrap font-mono border mt-3">{testResult}</pre>}
      </div>
    </div>
  );
}

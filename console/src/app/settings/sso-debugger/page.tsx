"use client";
import { useState } from "react";
import { Bug, Play, CheckCircle, XCircle, Clock, AlertTriangle } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
interface FlowStep { name: string; duration_ms: number; status: "ok" | "error" | "pending"; detail?: string; }
interface SsoResult { steps: FlowStep[]; total_ms: number; success: boolean; request_url: string; response_status: number; response_body: string; error?: string; }
export default function SsoDebuggerPage() {
  const t = useTranslations();
  const [protocol, setProtocol] = useState("saml");
  const [result, setResult] = useState<SsoResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const trace = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch("/api/v1/auth/sso-debug?protocol=" + protocol, { method: "POST", headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (!res.ok) return null;
      setResult(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Failed to trace SSO flow"); }
    finally { setLoading(false); }
  };
  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><Bug className="w-6 h-6 text-purple-500" /> SSO Debugger</h1><p className="text-sm text-gray-500 mt-1">Trace SSO login flows step-by-step and inspect requests/responses.</p></div>
      {error && <div className="rounded-lg border border-red-200 dark:border-red-900 bg-red-50 dark:bg-red-900/20 p-3 text-sm text-red-600 flex items-center justify-between"><span className="flex items-center gap-2"><AlertTriangle className="w-4 h-4" /> {error}</span><button onClick={() => setError(null)} className="text-xs underline hover:text-red-700">Dismiss</button></div>}
      <div className="flex items-center gap-3">
        <select value={protocol} onChange={(e) => setProtocol(e.target.value)} aria-label="SSO protocol" className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm">
          <option value="saml">SAML</option><option value="oidc">OIDC</option>
        </select>
        <button onClick={trace} disabled={loading} aria-label="Trace SSO login" className="px-4 py-2 rounded-lg bg-purple-600 text-white text-sm font-medium flex items-center gap-2"><Play className="w-4 h-4" /> {loading ? "Tracing..." : "Trace Login"}</button>
      </div>
      {loading && <div className="rounded-lg border dark:border-gray-800 p-8 text-center"><div className="inline-block w-5 h-5 border-2 border-current border-t-transparent rounded-full animate-spin text-blue-600 mb-2" /><div className="text-sm text-gray-500">Tracing SSO flow...</div></div>}
      {result && !loading && (
        <>
          <div className={"rounded-lg border-2 p-3 flex items-center gap-2 " + (result.success ? "border-green-300 dark:border-green-800 bg-green-50 dark:bg-green-900/20" : "border-red-300 dark:border-red-800 bg-red-50 dark:bg-red-900/20")}>
            {result.success ? <CheckCircle className="w-5 h-5 text-green-500" /> : <XCircle className="w-5 h-5 text-red-500" />}
            <span className={"font-semibold text-sm " + (result.success ? "text-green-700 dark:text-green-400" : "text-red-700 dark:text-red-400")}>{result.success ? "Login successful" : "Login failed"}</span>
            <span className="text-xs text-gray-500 ml-auto">Total: {result.total_ms}ms</span>
          </div>
          <div className="rounded-lg border dark:border-gray-800 p-4">
            <h3 className="text-sm font-semibold mb-3">Flow Steps</h3>
            <div className="relative pl-6">
              <div className="absolute left-2.5 top-0 bottom-0 w-px bg-gray-200 dark:bg-gray-800" />
              <div className="space-y-3">
                {result.steps.map((s, i) => (
                  <div key={i} className="relative">
                    <div className={"absolute -left-4 w-3 h-3 rounded-full border-2 " + (s.status === "ok" ? "bg-green-500 border-green-200" : s.status === "error" ? "bg-red-500 border-red-200" : "bg-gray-300 border-gray-100 dark:bg-gray-700 dark:border-gray-800")} />
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium">{s.name}</span>
                      <span className={"text-xs " + (s.status === "ok" ? "text-green-600" : s.status === "error" ? "text-red-600" : "text-gray-400")}>{s.status}</span>
                      <span className="text-xs text-gray-400 flex items-center gap-1"><Clock className="w-3 h-3" />{s.duration_ms}ms</span>
                    </div>
                    {s.detail && <p className="text-xs text-gray-500 mt-0.5">{s.detail}</p>}
                    {s.status === "error" && <p className="text-xs text-red-500 mt-0.5">{result.error}</p>}
                  </div>
                ))}
              </div>
            </div>
          </div>
          {result.request_url && <div className="rounded-lg border dark:border-gray-800 p-4"><span className="text-xs text-gray-500">Request URL</span><pre className="mt-1 text-xs font-mono bg-gray-50 dark:bg-gray-900 rounded p-2 overflow-x-auto">{result.request_url}</pre></div>}
          {result.response_body && <div className="rounded-lg border dark:border-gray-800 p-4">
            <div className="flex items-center gap-2 mb-1">
              <span className="text-xs text-gray-500">Response</span>
              <span className={"px-1.5 py-0.5 rounded text-xs font-bold " + (result.response_status < 400 ? "bg-green-100 dark:bg-green-900/30 dark:text-green-400" : "bg-red-100 dark:bg-red-900/30 dark:text-red-400")}>{result.response_status}</span>
            </div>
            <pre className="text-xs font-mono bg-gray-50 dark:bg-gray-900 rounded p-2 max-h-48 overflow-auto">{result.response_body}</pre>
          </div>}
        </>
      )}
      {!result && !loading && <p className="text-sm text-gray-500 text-center py-8">Select protocol and trace a login flow.</p>}
    </div>
  );
}

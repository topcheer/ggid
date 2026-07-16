"use client";
import { useState, useEffect, useCallback } from "react";
import { Bug, Play, RotateCcw } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
interface DeliveryLog { id: string; timestamp: string; status: number; latency_ms: number; success: boolean; }
interface TestResult { status: number; status_text: string; headers: Record<string, string>; body: string; latency_ms: number; success?: boolean; }
export default function WebhookDebuggerPage() {
  const t = useTranslations();
  const [endpoints, setEndpoints] = useState<string[]>([]);
  const [selected, setSelected] = useState("");
  const [payload, setPayload] = useState("{\n  \"event\": \"user.created\",\n  \"data\": {\n    \"user_id\": \"usr-123\"\n  }\n}");
  const [result, setResult] = useState<TestResult | null>(null);
  const [logs, setLogs] = useState<DeliveryLog[]>([]);
  const [loading, setLoading] = useState(false);
  const fetchEndpoints = useCallback(async () => { try { const res = await fetch("/api/v1/webhooks/endpoints", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); if (res.ok) { const d = await res.json(); setEndpoints(d.endpoints || d || []); } } catch { /* noop */ } }, []);
  const fetchLogs = useCallback(async () => { if (!selected) return; try { const res = await fetch("/api/v1/webhooks/endpoints/" + selected + "/deliveries", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); if (res.ok) { const d = await res.json(); setLogs(d.deliveries || d || []); } } catch { /* noop */ } }, [selected]);
  useEffect(() => { fetchEndpoints(); }, [fetchEndpoints]);
  useEffect(() => { fetchLogs(); }, [fetchLogs]);
  const send = async () => { if (!selected) return; setLoading(true); try { const res = await fetch("/api/v1/webhooks/endpoints/" + selected + "/test", { method: "POST", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: payload }); if (res.ok) setResult(await res.json()); } catch { /* noop */ } finally { setLoading(false); } };
  const replay = async (id: string) => { try { await fetch("/api/v1/webhooks/deliveries/" + id + "/replay", { method: "POST", headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); fetchLogs(); } catch { /* noop */ } };
  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><Bug className="w-6 h-6 text-purple-500" /> Webhook Debugger</h1><p className="text-sm text-gray-500 mt-1">Test webhook endpoints and inspect delivery results.</p></div>
      <select aria-label="Selected" value={selected} onChange={(e) => setSelected(e.target.value)} className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"><option value="">Select Endpoint</option>{endpoints.map((ep) => <option key={ep} value={ep}>{ep}</option>)}</select>
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-2">Test Payload</h3><textarea aria-label="Payload" value={payload} onChange={(e) => setPayload(e.target.value)} rows={10} className="w-full px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-xs font-mono" /><button onClick={send} disabled={loading || !selected} className="mt-2 px-4 py-2 rounded-lg bg-purple-600 text-white text-sm font-medium flex items-center gap-2"><Play className="w-4 h-4" /> {loading ? "Sending..." : "Send Test"}</button></div>
        <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-2">Response</h3>{result ? (<div className="space-y-2"><div className="flex items-center gap-2"><span className={"px-2 py-0.5 rounded text-xs font-bold " + (result.success ? "bg-green-100 dark:bg-green-900/30 dark:text-green-400" : "bg-red-100 dark:bg-red-900/30 dark:text-red-400")}>{result.status} {result.status_text}</span><span className="text-xs text-gray-500">{result.latency_ms}ms</span></div><div><span className="text-xs text-gray-500">Headers</span><pre className="mt-1 text-xs font-mono bg-gray-50 dark:bg-gray-900 rounded p-2 max-h-24 overflow-y-auto">{JSON.stringify(result.headers, null, 2)}</pre></div><div><span className="text-xs text-gray-500">Body</span><pre className="mt-1 text-xs font-mono bg-gray-50 dark:bg-gray-900 rounded p-2 max-h-40 overflow-y-auto">{result.body}</pre></div></div>) : <p className="text-xs text-gray-400">No response yet.</p>}</div>
      </div>
      <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-2">Delivery Log</h3><div className="overflow-x-auto"><table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-3 py-2 text-left font-medium">Timestamp</th><th className="px-3 py-2 text-left font-medium">Status</th><th className="px-3 py-2 text-left font-medium">Latency</th><th className="px-3 py-2 text-left font-medium">Result</th><th className="px-3 py-2 text-left font-medium">Action</th></tr></thead><tbody className="divide-y dark:divide-gray-800">{logs.map((l) => (<tr key={l.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-3 py-2 text-xs text-gray-500">{l.timestamp}</td><td className="px-3 py-2 text-xs font-bold">{l.status}</td><td className="px-3 py-2 text-xs">{l.latency_ms}ms</td><td className="px-3 py-2"><span className={"text-xs " + (l.success ? "text-green-600" : "text-red-600")}>{l.success ? "OK" : "FAILED"}</span></td><td className="px-3 py-2"><button onClick={() => replay(l.id)} className="text-xs text-blue-600 hover:underline flex items-center gap-1"><RotateCcw className="w-3 h-3" /> Replay</button></td></tr>))}{logs.length === 0 && <tr><td colSpan={5} className="px-3 py-4 text-center text-gray-500 text-sm">No deliveries.</td></tr>}</tbody></table></div></div>
    </div>
  );
}

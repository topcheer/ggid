"use client";
import { useTranslations } from "@/lib/i18n";
import { useState } from "react";
import { ArrowLeftRight, Play, Clock } from "lucide-react";

interface ExchangeResult { access_token: string; token_type: string; expires_in: number; scope: string; issued_token_type: string; }
interface HistoryItem { id: string; subject: string; audience: string; scope: string; timestamp: string; success: boolean; }

export default function OAuthTokenExchangePage() {
  const t = useTranslations();
  const [subjectToken, setSubjectToken] = useState("");
  const [actorToken, setActorToken] = useState("");
  const [audience, setAudience] = useState("");
  const [scope, setScope] = useState("");
  const [resource, setResource] = useState("");
  const [result, setResult] = useState<ExchangeResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [history] = useState<HistoryItem[]>([]);

  const exchange = async () => {
    if (!subjectToken) return;
    setLoading(true);
    try {
      const res = await fetch("/api/v1/oauth/token-exchange", { method: "POST", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify({ grant_type: "urn:ietf:params:oauth:grant-type:token-exchange", subject_token: subjectToken, actor_token: actorToken || undefined, audience, scope, resource }) });
      if (res.ok) setResult(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  };

  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><ArrowLeftRight className="w-6 h-6 text-blue-500" />{t("oauthTokenExchange.title")}</h1><p className="text-sm text-gray-500 mt-1">Exchange tokens for impersonation or delegation flows.</p></div>

      <div className="rounded-lg border dark:border-gray-800 p-4 space-y-3">
        <div><label className="text-sm font-medium">Subject Token</label><textarea aria-label="eyJhbG..." value={subjectToken} onChange={(e) => setSubjectToken(e.target.value)} placeholder="eyJhbG..." className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono h-20" /></div>
        <div><label className="text-sm font-medium">Actor Token (optional)</label><textarea aria-label="eyJhbG..." value={actorToken} onChange={(e) => setActorToken(e.target.value)} placeholder="eyJhbG..." className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono h-20" /></div>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
          <div><label className="text-sm font-medium">Audience</label><input aria-label="https://api.example.com" type="text" value={audience} onChange={(e) => setAudience(e.target.value)} placeholder="https://api.example.com" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div>
          <div><label className="text-sm font-medium">Scope</label><input aria-label="read write" type="text" value={scope} onChange={(e) => setScope(e.target.value)} placeholder="read write" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div>
          <div><label className="text-sm font-medium">Resource</label><input aria-label="urn:api:example" type="text" value={resource} onChange={(e) => setResource(e.target.value)} placeholder="urn:api:example" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div>
        </div>
        <button onClick={exchange} disabled={loading || !subjectToken} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium disabled:opacity-50 flex items-center gap-2"><Play className="w-4 h-4" /> {loading ? "Exchanging..." : "Exchange Token"}</button>
      </div>

      {result && (<div className="rounded-lg border border-green-300 dark:border-green-800 bg-green-50 dark:bg-green-900/20 p-4"><h3 className="text-sm font-semibold text-green-700 dark:text-green-400 mb-2">Exchange Successful</h3><div className="space-y-1 text-sm"><div><span className="text-gray-500">Token Type:</span> <span className="font-mono">{result.issued_token_type}</span></div><div><span className="text-gray-500">Expires In:</span> <span className="font-bold">{result.expires_in}s</span></div><div><span className="text-gray-500">Scope:</span> <span className="font-mono">{result.scope}</span></div><div><span className="text-gray-500">Access Token:</span><pre className="font-mono text-xs bg-white dark:bg-gray-900 rounded p-2 mt-1 overflow-x-auto">{result.access_token.substring(0, 100)}...</pre></div></div></div>)}

      <div className="overflow-x-auto rounded-lg border dark:border-gray-800"><table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Subject</th><th className="px-4 py-3 text-left font-medium">Audience</th><th className="px-4 py-3 text-left font-medium">Scope</th><th className="px-4 py-3 text-left font-medium">Status</th><th className="px-4 py-3 text-left font-medium">Time</th></tr></thead><tbody className="divide-y dark:divide-gray-800">{history.map((h) => (<tr key={h.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3 font-mono text-xs truncate max-w-xs">{h.subject.substring(0, 40)}...</td><td className="px-4 py-3 text-xs text-gray-500">{h.audience}</td><td className="px-4 py-3 text-xs">{h.scope}</td><td className="px-4 py-3">{h.success ? <span className="text-xs text-green-600">Success</span> : <span className="text-xs text-red-600">Failed</span>}</td><td className="px-4 py-3 text-xs text-gray-400">{h.timestamp}</td></tr>))}{history.length === 0 && <tr><td colSpan={5} className="px-4 py-8 text-center text-gray-500">No exchange history.</td></tr>}</tbody></table></div>
    </div>
  );
}

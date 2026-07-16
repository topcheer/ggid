"use client";

import { useRedirectURIValidationConfig } from "@ggid/sdk-react";
import { Link2, ShieldCheck } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function RedirectURIValidationConfigPage() {
  const t = useTranslations();
  const { data, loading, error, refresh, testUri } = useRedirectURIValidationConfig();
  if (loading) return <div className="p-8 text-gray-400">Loading redirect URI validation...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div><h1 className="text-2xl font-bold">Redirect URI Validation</h1><p className="text-sm text-gray-400 mt-1">OAuth redirect URI security policy</p></div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Save</button>
      </div>

      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold mb-3 flex items-center gap-2"><ShieldCheck className="w-4 h-4 text-green-400" /> Global Policy</h2>
        <div className="grid grid-cols-2 gap-4">
          <label className="flex items-center gap-2 text-sm"><input type="checkbox" defaultChecked={data?.https_only} /> HTTPS Only</label>
          <label className="flex items-center gap-2 text-sm"><input type="checkbox" defaultChecked={data?.exact_match_only} /> Exact Match Only (no wildcards)</label>
          <label className="flex items-center gap-2 text-sm"><input type="checkbox" defaultChecked={data?.localhost_allowlist} /> Allow localhost</label>
          <label className="flex items-center gap-2 text-sm"><input type="checkbox" defaultChecked={data?.fragment_allowed} /> Allow fragments</label>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
        <div className="bg-gray-900 rounded-xl p-6"><h2 className="text-sm font-semibold mb-3">Custom Scheme Allowlist</h2><div className="space-y-1">{(data?.custom_schemes ?? []).map((s) => (<div key={s} className="text-xs font-mono bg-gray-800 rounded px-2 py-1">{s}</div>))}</div></div>
        <div className="bg-gray-900 rounded-xl p-6"><h2 className="text-sm font-semibold mb-3">Per-Client Allowed Patterns</h2><div className="space-y-2">{(data?.per_client ?? []).map((c) => (<div key={c.client} className="bg-gray-800 rounded p-3"><p className="text-xs font-medium mb-1">{c.client}</p><div className="space-y-0.5">{c.uris.map((u) => (<div key={u} className="text-xs font-mono text-gray-400">{u}</div>))}</div></div>))}</div></div>
      </div>

      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-sm font-semibold mb-3 flex items-center gap-2"><Link2 className="w-4 h-4 text-blue-400" /> URI Tester</h2>
        <div className="flex gap-2"><input aria-label="https://example.com/callback" type="text" placeholder="https://example.com/callback" className="flex-1 px-3 py-2 bg-gray-800 rounded-lg text-sm" id="uri-test-input" /><button onClick={() => testUri((document.getElementById("uri-test-input") as HTMLInputElement)?.value || "")} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Test</button></div>
      </div>
    </div>
  );
}

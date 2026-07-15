"use client";
import { useTranslations } from "@/lib/i18n";

import { useOAuthIssuerMetadataConfig } from "@ggid/sdk-react";
import { Globe, Code } from "lucide-react";

export default function OAuthIssuerMetadataConfigPage() {
  const t = useTranslations();
  const { data, loading, error, refresh } = useOAuthIssuerMetadataConfig();
  if (loading) return <div className="p-8 text-gray-400">Loading...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div><h1 className="text-2xl font-bold">OAuth Issuer Metadata</h1><p className="text-sm text-gray-400 mt-1">OIDC discovery configuration</p></div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Save</button>
      </div>

      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold mb-3 flex items-center gap-2"><Globe className="w-4 h-4 text-blue-400" /> Issuer</h2>
        <input type="text" defaultValue={data?.issuer_url} className="w-full px-3 py-2 bg-gray-800 rounded-lg text-sm font-mono" />
        <p className="text-xs text-blue-400 mt-1">Discovery: {data?.issuer_url}/.well-known/openid-configuration</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
        <div className="bg-gray-900 rounded-xl p-6"><h2 className="text-sm font-semibold mb-3">Supported Response Types</h2><div className="flex flex-wrap gap-2">{(data?.response_types ?? []).map((r) => <span key={r} className="text-xs font-mono px-2 py-1 bg-gray-800 rounded text-blue-400">{r}</span>)}</div></div>
        <div className="bg-gray-900 rounded-xl p-6"><h2 className="text-sm font-semibold mb-3">Subject Types</h2><div className="flex flex-wrap gap-2">{(data?.subject_types ?? []).map((s) => <span key={s} className="text-xs font-mono px-2 py-1 bg-gray-800 rounded text-green-400">{s}</span>)}</div></div>
        <div className="bg-gray-900 rounded-xl p-6"><h2 className="text-sm font-semibold mb-3">Claim Types</h2><div className="flex flex-wrap gap-2">{(data?.claim_types ?? []).map((c) => <span key={c} className="text-xs font-mono px-2 py-1 bg-gray-800 rounded text-purple-400">{c}</span>)}</div></div>
        <div className="bg-gray-900 rounded-xl p-6 space-y-2"><h2 className="text-sm font-semibold mb-3">Parameters</h2><label className="flex items-center gap-2 text-sm"><input type="checkbox" defaultChecked={data?.request_param_supported} /> request_parameter_supported</label><label className="flex items-center gap-2 text-sm"><input type="checkbox" defaultChecked={data?.request_uri_supported} /> request_uri_parameter_supported</label><label className="flex items-center gap-2 text-sm"><input type="checkbox" defaultChecked={data?.require_request_uri} /> require_request_uri_registration</label></div>
      </div>

      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-sm font-semibold mb-3 flex items-center gap-2"><Code className="w-4 h-4 text-green-400" /> Well-Known Preview</h2>
        <pre className="bg-gray-800 rounded-lg p-4 text-xs font-mono overflow-x-auto text-gray-300">{JSON.stringify(data?.well_known ?? {}, null, 2)}</pre>
      </div>
    </div>
  );
}

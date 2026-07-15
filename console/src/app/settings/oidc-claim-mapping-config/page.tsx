"use client";

import { useOidcClaimMappingConfig } from "@ggid/sdk-react";
import { Key, Plus } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function OidcClaimMappingConfigPage() {
  const { data, loading, error, refresh } = useOidcClaimMappingConfig();
  const t = useTranslations();
  if (loading) return <div className="p-8 text-gray-400">{t("oidcClaimMappingConfig.loading")}</div>;
  if (error) return <div className="p-8 text-red-400">{t("common.error")}: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div><h1 className="text-2xl font-bold">{t("oidcClaimMappingConfig.title")}</h1><p className="text-sm text-gray-400 mt-1">{t("oidcClaimMappingConfig.subtitle")}</p></div>
        <div className="flex gap-2"><button className="flex items-center gap-1 px-3 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg text-sm font-medium transition"><Plus className="w-4 h-4" /> {t("oidcClaimMapping.addClaim")}</button><button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">{t("oidcClaimMappingConfig.save")}</button></div>
      </div>

      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold mb-4 flex items-center gap-2"><Key className="w-4 h-4 text-blue-400" /> Claim Mappings</h2>
        <table className="w-full text-sm"><thead><tr className="border-b border-gray-800 text-gray-400"><th className="text-left py-2">{t("oidcClaimMappingConfig.claimName")}</th><th className="text-left py-2">{t("oidcClaimMappingConfig.source")}</th><th className="text-left py-2">{t("oidcClaimMappingConfig.transform")}</th><th className="text-left py-2">{t("oidcClaimMappingConfig.tokenType")}</th></tr></thead>
          <tbody>{(data?.claims ?? []).map((c) => (
            <tr key={c.name} className="border-b border-gray-800"><td className="py-2 font-mono text-xs text-blue-400">{c.name}</td><td className="py-2 text-xs"><span className="px-2 py-0.5 rounded bg-gray-700">{c.source}</span></td><td className="py-2 text-xs text-gray-400">{c.transform}</td><td className="py-2 text-xs"><span className="px-2 py-0.5 rounded bg-purple-900 text-purple-300">{c.token_type}</span></td></tr>
          ))}</tbody>
        </table>
      </div>

      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-sm font-semibold mb-3">{t("oidcClaimMappingConfig.scopeToClaims")}</h2>
        <div className="overflow-x-auto"><table className="w-full text-xs"><thead><tr className="border-b border-gray-800 text-gray-400"><th className="text-left py-2">{t("oidcClaimMappingConfig.scope")}</th>{(data?.all_claims ?? []).map((c) => <th key={c} className="text-center py-2 px-1">{c}</th>)}</tr></thead>
          <tbody>{(data?.scope_matrix ?? []).map((row) => (
            <tr key={row.scope} className="border-b border-gray-800"><td className="py-2 font-mono text-blue-400">{row.scope}</td>{(data?.all_claims ?? []).map((c) => <td key={c} className="text-center py-2">{row.claims.includes(c) ? <span className="text-green-400">✓</span> : <span className="text-gray-700">-</span>}</td>)}</tr>
          ))}</tbody>
        </table></div>
      </div>
    </div>
  );
}

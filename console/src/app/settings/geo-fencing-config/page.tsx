"use client";

import { useGeoFencingConfig } from "@ggid/sdk-react";
import { Globe, Plus } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function GeoFencingConfigPage() {
  const t = useTranslations();

  const { data, loading, error, refresh } = useGeoFencingConfig();
  if (loading) return <div className="p-8 text-gray-400">{t("big1.geoFencingConfig.loadingGeoFencing")}</div>;
  if (error) return <div className="p-8 text-red-400">{t("big1.geoFencingConfig.error")}{error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div><h1 className="text-2xl font-bold">{t("big1.geoFencingConfig.title")}</h1><p className="text-sm text-gray-400 mt-1">{t("big1.geoFencingConfig.locationBasedAccessControls")}</p></div>
        <div className="flex gap-2"><button className="flex items-center gap-1 px-3 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg text-sm font-medium transition"><Plus className="w-4 h-4" />{t("big1.geoFencingConfig.addRule")}</button><button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">{t("big1.geoFencingConfig.save")}</button></div>
      </div>

      <div className="flex items-center gap-3 mb-6">
        <label className="text-sm">{t("big1.geoFencingConfig.enabled")}</label>
        <input aria-label="Toggle option" type="checkbox" defaultChecked={data?.enabled} />
      </div>

      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold mb-4 flex items-center gap-2"><Globe className="w-4 h-4 text-blue-400" />{t("big1.geoFencingConfig.rules")}</h2>
        <table className="w-full text-sm"><thead><tr className="border-b border-gray-800 text-gray-400"><th className="text-left py-2">{t("big1.geoFencingConfig.country")}</th><th className="text-left py-2">{t("big1.geoFencingConfig.cidr")}</th><th className="text-left py-2">{t("big1.geoFencingConfig.action")}</th><th className="text-left py-2">{t("big1.geoFencingConfig.label")}</th></tr></thead>
          <tbody>{(data?.rules ?? []).map((r) => (
            <tr key={r.id} className="border-b border-gray-800"><td className="py-2">{r.country}</td><td className="py-2 font-mono text-xs">{r.cidr}</td><td className="py-2"><span className={"text-xs px-2 py-0.5 rounded " + (r.action === "allow" ? "bg-green-900 text-green-300" : r.action === "challenge" ? "bg-yellow-900 text-yellow-300" : "bg-red-900 text-red-300")}>{r.action}</span></td><td className="py-2 text-xs text-gray-400">{r.label}</td></tr>
          ))}</tbody>
        </table>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="bg-gray-900 rounded-xl p-6"><h2 className="text-sm font-semibold mb-3">{t("big1.geoFencingConfig.whitelistIps")}</h2><div className="space-y-1">{(data?.whitelist_ips ?? []).map((ip) => (<div key={ip} className="text-xs font-mono bg-gray-800 rounded px-2 py-1">{ip}</div>))}</div></div>
        <div className="bg-gray-900 rounded-xl p-6"><h2 className="text-sm font-semibold mb-3">{t("big1.geoFencingConfig.mapCoverage")}</h2><div className="flex flex-wrap gap-1">{(data?.rules ?? []).map((r) => (<span key={r.id} className={"text-xs px-2 py-0.5 rounded " + (r.action === "allow" ? "bg-green-900 text-green-300" : r.action === "challenge" ? "bg-yellow-900 text-yellow-300" : "bg-red-900 text-red-300")}>{r.country}</span>))}</div></div>
      </div>
    </div>
  );
}

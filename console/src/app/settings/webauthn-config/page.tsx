"use client";

import { useWebauthnConfig } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";
import { Fingerprint, Shield } from "lucide-react";

export default function WebauthnConfigPage() {
  const t = useTranslations();
  const { data, loading, error, refresh } = useWebauthnConfig();
  if (loading) return <div className="p-8 text-gray-400">{t("webauthnConfig.loading")}</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div><h1 className="text-2xl font-bold">{t("webauthnConfig.title")}</h1><p className="text-sm text-gray-400 mt-1">{t("webauthnConfig.subtitle")}</p></div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">{t("common.save")}</button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="bg-gray-900 rounded-xl p-6 space-y-4">
          <h2 className="text-sm font-semibold flex items-center gap-2"><Fingerprint className="w-4 h-4 text-blue-400" /> {t("webauthnConfig.relyingParty")}</h2>
          <div><label className="text-xs text-gray-400">{t("webauthnConfig.rpId")}</label><input type="text" defaultValue={data?.rp_id} className="w-full mt-1 px-3 py-2 bg-gray-800 rounded-lg text-sm" /></div>
          <div><label className="text-xs text-gray-400">{t("webauthnConfig.rpName")}</label><input type="text" defaultValue={data?.rp_name} className="w-full mt-1 px-3 py-2 bg-gray-800 rounded-lg text-sm" /></div>
          <div><label className="text-xs text-gray-400">{t("webauthnConfig.origin")}</label><input type="text" defaultValue={data?.origin} className="w-full mt-1 px-3 py-2 bg-gray-800 rounded-lg text-sm" /></div>
        </div>
        <div className="bg-gray-900 rounded-xl p-6 space-y-4">
          <h2 className="text-sm font-semibold flex items-center gap-2"><Shield className="w-4 h-4 text-green-400" /> {t("webauthnConfig.securityPolicy")}</h2>
          <div><label className="text-xs text-gray-400">{t("webauthnConfig.attestationReq")}</label><select aria-label="Select option" defaultValue={data?.attestation_requirement} className="w-full mt-1 px-3 py-2 bg-gray-800 rounded-lg text-sm"><option>none</option><option>indirect</option><option>direct</option></select></div>
          <div><label className="text-xs text-gray-400">{t("webauthnConfig.userVerification")}</label><select defaultValue={data?.user_verification} className="w-full mt-1 px-3 py-2 bg-gray-800 rounded-lg text-sm"><option>required</option><option>preferred</option><option>discouraged</option></select></div>
          <div><label className="text-xs text-gray-400">{t("webauthnConfig.timeout")}: {data?.timeout_seconds}s</label><input type="range" min="30" max="600" defaultValue={data?.timeout_seconds} className="w-full mt-1" /></div>
        </div>
      </div>

      <div className="bg-gray-900 rounded-xl p-6 mt-6">
        <h2 className="text-sm font-semibold mb-3">{t("webauthnConfig.supportedAlgorithms")}</h2>
        <div className="grid grid-cols-3 gap-3">
          {(data?.supported_algorithms ?? []).map((alg) => (
            <label key={alg.id} className="flex items-center gap-2 bg-gray-800 rounded-lg p-3 cursor-pointer">
              <input type="checkbox" defaultChecked={alg.enabled} />
              <div><p className="text-sm font-medium font-mono">{alg.id}</p><p className="text-xs text-gray-400">COSE: {alg.cose_id}</p></div>
            </label>
          ))}
        </div>
      </div>

      <div className="bg-gray-900 rounded-xl p-6 mt-6">
        <h2 className="text-sm font-semibold mb-3">{t("webauthnConfig.perPlatform")}</h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {(data?.per_platform ?? []).map((p) => (
            <div key={p.platform} className="bg-gray-800 rounded-lg p-4">
              <p className="text-sm font-semibold mb-2">{p.platform}</p>
              <div className="space-y-1 text-xs text-gray-400">
                <div className="flex justify-between"><span>{t("webauthnConfig.authType")}</span><span>{p.authenticator_type}</span></div>
                <div className="flex justify-between"><span>{t("webauthnConfig.attachment")}</span><span>{p.attachment}</span></div>
                <div className="flex justify-between"><span>{t("settings.enabled")}</span><span className={p.enabled ? "text-green-400" : "text-red-400"}>{p.enabled ? t("rateLimits.on") : t("rateLimits.off")}</span></div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

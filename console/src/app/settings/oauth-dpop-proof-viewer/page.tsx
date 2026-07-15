"use client";

import { useOAuthDpopProofViewer } from "@ggid/sdk-react";
import { KeyRound, Fingerprint, Clock, CheckCircle, XCircle, AlertTriangle } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function OAuthDpopProofViewerPage() {

  const { data, loading, error, refresh } = useOAuthDpopProofViewer();
  const t = useTranslations();

  if (loading) return <div className="p-8 text-gray-400">{t("oauthDpopViewer.loading")}</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const proof = data?.proof;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">{t("oauthDpopViewer.title")}</h1>
          <p className="text-sm text-gray-400 mt-1">{t("oauthDpopViewer.subtitle")}</p>
        </div>
        <button
          onClick={refresh}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          Refresh
        </button>
      </div>

      {/* Proof Status Banner */}
      <div className="bg-gray-900 rounded-xl p-4 mb-6 flex items-center gap-4">
        {proof?.valid ? (
          <CheckCircle className="w-8 h-8 text-green-400" />
        ) : (
          <XCircle className="w-8 h-8 text-red-400" />
        )}
        <div>
          <p className="text-lg font-semibold">{proof?.valid ? "Proof Valid" : "Proof Invalid"}</p>
          <p className="text-sm text-gray-400">{proof?.valid ? "Signature verified, key binding confirmed" : proof?.error_message ?? "Verification failed"}</p>
        </div>
      </div>

      {/* JWT Sections */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 mb-6">
        {/* Header */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold text-gray-400 mb-3">{t("oauthDpopViewer.jwtHeader")}</h2>
          <div className="space-y-2">
            <div className="bg-gray-800 rounded-lg p-3">
              <p className="text-xs text-gray-500">typ</p>
              <p className="text-sm font-mono">{proof?.header?.typ ?? "dpop+jwt"}</p>
            </div>
            <div className="bg-gray-800 rounded-lg p-3">
              <p className="text-xs text-gray-500">alg</p>
              <p className="text-sm font-mono">{proof?.header?.alg ?? "ES256"}</p>
            </div>
            <div className="bg-gray-800 rounded-lg p-3">
              <p className="text-xs text-gray-500">jwk (thumbprint)</p>
              <p className="text-xs font-mono break-all text-blue-400">{proof?.header?.jwk_thumbprint ?? "N/A"}</p>
            </div>
          </div>
        </div>

        {/* Payload */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold text-gray-400 mb-3">{t("oauthDpopViewer.jwtPayload")}</h2>
          <div className="space-y-2">
            <PayloadItem icon={<KeyRound className="w-3 h-3" />} label="htm" value={proof?.payload?.htm ?? "POST"} />
            <PayloadItem icon={<Clock className="w-3 h-3" />} label="htu" value={proof?.payload?.htu ?? "https://api.example.com/token"} />
            <PayloadItem icon={<Fingerprint className="w-3 h-3" />} label="jti" value={proof?.payload?.jti ?? "-wYQu9O9oSzZ3M8jKqP"} mono />
            <PayloadItem icon={<KeyRound className="w-3 h-3" />} label="ath" value={proof?.payload?.ath ?? "czZmNGRlNjk4MzU2Nzc4NTQ0Njg="} mono />
            <PayloadItem icon={<Clock className="w-3 h-3" />} label="iat" value={String(proof?.payload?.iat ?? 1700000000)} mono />
          </div>
        </div>

        {/* Signature */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold text-gray-400 mb-3">{t("oauthDpopViewer.signature")}</h2>
          <div className="bg-gray-800 rounded-lg p-3">
            <p className="text-xs text-gray-500 mb-1">{t("oauthDpopViewer.encoded")}</p>
            <p className="text-xs font-mono break-all text-gray-300">{proof?.signature ?? "ZmFrZXNpZ25hdHVyZQ"}</p>
          </div>
          <div className="mt-3">
            <div className="flex items-center gap-2 mb-2">
              <span className="text-xs text-gray-500">{t("oauthDpopViewer.keyBinding")}</span>
              {proof?.key_binding_verified ? (
                <span className="flex items-center gap-1 text-xs text-green-400"><CheckCircle className="w-3 h-3" /> Verified</span>
              ) : (
                <span className="flex items-center gap-1 text-xs text-red-400"><XCircle className="w-3 h-3" /> Failed</span>
              )}
            </div>
            <p className="text-xs text-gray-400">Algorithm: {proof?.key_binding_algorithm ?? "ES256"}</p>
          </div>
        </div>
      </div>

      {/* Proof Validity Timeline + Error Analysis */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">{t("oauthDpopViewer.proofValidity")}</h2>
          <div className="space-y-2">
            {(data?.validity_timeline ?? []).map((step, i) => (
              <div key={i} className="flex items-center gap-3">
                <div className={"w-7 h-7 rounded-full flex items-center justify-center text-xs " + (step.passed ? "bg-green-900 text-green-300" : "bg-red-900 text-red-300")}>
                  {step.passed ? "\u2713" : "\u2717"}
                </div>
                <div className="flex-1">
                  <p className="text-sm font-medium">{step.check}</p>
                  <p className="text-xs text-gray-400">{step.detail}</p>
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
            <AlertTriangle className="w-5 h-5 text-yellow-400" />
            Error Analysis
          </h2>
          <div className="space-y-2">
            {(data?.error_analysis ?? []).map((err, i) => (
              <div key={i} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-center justify-between mb-1">
                  <p className="text-sm font-medium font-mono">{err.code}</p>
                  <span className={"text-xs px-2 py-0.5 rounded " + (err.severity === "error" ? "bg-red-900 text-red-300" : "bg-yellow-900 text-yellow-300")}>
                    {err.severity}
                  </span>
                </div>
                <p className="text-xs text-gray-400">{err.description}</p>
                {err.remediation && (
                  <p className="text-xs text-blue-400 mt-1">{t("oauthDpopViewer.fix")} {err.remediation}</p>
                )}
              </div>
            ))}
            {(data?.error_analysis ?? []).length === 0 && (
              <p className="text-sm text-gray-500 text-center py-4">{t("oauthDpopViewer.noErrors")}</p>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function PayloadItem({ icon, label, value, mono }: { icon: React.ReactNode; label: string; value: string; mono?: boolean }) {
  return (
    <div className="bg-gray-800 rounded-lg p-3">
      <div className="flex items-center gap-1 mb-1 text-gray-500">
        {icon}
        <span className="text-xs">{label}</span>
      </div>
      <p className={"text-sm " + (mono ? "font-mono" : "font-medium")}>{value}</p>
    </div>
  );
}

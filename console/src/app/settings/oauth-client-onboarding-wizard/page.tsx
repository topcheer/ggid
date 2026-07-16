"use client";

import { useState } from "react";
import { useOAuthClientOnboardingWizard } from "@ggid/sdk-react";
import { CheckCircle, ArrowRight, Key, Zap } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function OAuthClientOnboardingWizardPage() {

  const { data, loading, error, refresh } = useOAuthClientOnboardingWizard();
  const [step, setStep] = useState(1);
  const t = useTranslations();

  if (loading) return <div className="p-8 text-gray-400">{t("oauthOnboarding.loading")}</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const steps = ["App Info", "Grant Types", "Redirect URIs", "Scopes", "Review"];

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="mb-8">
        <h1 className="text-2xl font-bold">{t("oauthOnboarding.title")}</h1>
        <p className="text-sm text-gray-400 mt-1">{t("oauthOnboarding.subtitle")}</p>
      </div>

      {/* Stepper */}
      <div className="flex items-center gap-1 mb-8">
        {steps.map((s, i) => (
          <div key={s} className="flex items-center gap-1">
            <button onClick={() => setStep(i + 1)} className={"flex items-center gap-2 px-3 py-2 rounded-lg text-xs font-medium transition " + (step === i + 1 ? "bg-blue-600 text-white" : step > i + 1 ? "bg-green-900 text-green-300" : "bg-gray-800 text-gray-400")}>
              {step > i + 1 ? <CheckCircle className="w-3 h-3" /> : <span>{i + 1}</span>}
              {s}
            </button>
            {i < steps.length - 1 && <ArrowRight className="w-3 h-3 text-gray-600" />}
          </div>
        ))}
      </div>

      {/* Step Content */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        {step === 1 && (
          <div className="space-y-4">
            <h2 className="text-sm font-semibold">{t("oauthOnboarding.appInfo")}</h2>
            <input type="text" placeholder="App name" className="w-full px-3 py-2 bg-gray-800 rounded-lg text-sm" defaultValue={data?.app_info?.name ?? ""} />
            <input type="text" placeholder="Description" className="w-full px-3 py-2 bg-gray-800 rounded-lg text-sm" defaultValue={data?.app_info?.description ?? ""} />
            <select aria-label="Select option" className="w-full px-3 py-2 bg-gray-800 rounded-lg text-sm">
              <option>{t("oauthOnboarding.webApp")}</option><option>{t("oauthOnboarding.mobileApp")}</option><option>{t("oauthOnboarding.serviceM2M")}</option>
            </select>
          </div>
        )}
        {step === 2 && (
          <div className="space-y-3">
            <h2 className="text-sm font-semibold">{t("oauthOnboarding.grantTypes")}</h2>
            {(data?.grant_types ?? []).map((g) => (
              <label key={g.value} className="flex items-center gap-2 bg-gray-800 rounded-lg p-3 cursor-pointer">
                <input type="checkbox" defaultChecked={g.selected} />
                <div><p className="text-sm font-medium">{g.value}</p><p className="text-xs text-gray-400">{g.description}</p></div>
              </label>
            ))}
          </div>
        )}
        {step === 3 && (
          <div className="space-y-3">
            <h2 className="text-sm font-semibold">{t("oauthOnboarding.redirectUris")}</h2>
            {(data?.redirect_uris ?? []).map((uri) => (
              <div key={uri} className="flex items-center gap-2 bg-gray-800 rounded-lg p-2">
                <span className="text-xs font-mono text-blue-400 flex-1">{uri}</span>
                <span className="text-xs text-gray-500">{t("oauthOnboarding.httpsVerified")}</span>
              </div>
            ))}
            <input type="text" placeholder="https://your-app.com/callback" className="w-full px-3 py-2 bg-gray-800 rounded-lg text-sm" />
          </div>
        )}
        {step === 4 && (
          <div className="space-y-3">
            <h2 className="text-sm font-semibold">{t("oauthOnboarding.requestedScopes")}</h2>
            {(data?.scopes ?? []).map((sc) => (
              <label key={sc.name} className="flex items-center gap-2 bg-gray-800 rounded-lg p-3 cursor-pointer">
                <input type="checkbox" defaultChecked={sc.required} disabled={sc.required} />
                <div><p className="text-sm font-medium font-mono">{sc.name}</p><p className="text-xs text-gray-400">{sc.description}</p></div>
                {sc.required && <span className="text-xs text-yellow-400 ml-auto">{t("oauthOnboarding.required")}</span>}
              </label>
            ))}
          </div>
        )}
        {step === 5 && (
          <div className="space-y-4">
            <h2 className="text-sm font-semibold">{t("oauthOnboarding.reviewGenerate")}</h2>
            {data?.credentials && (
              <div className="bg-gray-800 rounded-lg p-4">
                <div className="flex items-center gap-2 mb-3"><Key className="w-4 h-4 text-yellow-400" /><span className="text-sm font-medium">{t("oauthOnboarding.credentials")}</span></div>
                <div className="space-y-2">
                  <div><p className="text-xs text-gray-500">{t("oauthOnboarding.clientId")}</p><p className="text-sm font-mono text-green-400">{data.credentials.client_id}</p></div>
                  <div><p className="text-xs text-gray-500">{t("oauthOnboarding.clientSecret")}</p><p className="text-sm font-mono text-red-400">{data.credentials.client_secret}</p></div>
                </div>
              </div>
            )}
            <button className="flex items-center gap-2 px-4 py-2 bg-green-600 hover:bg-green-700 rounded-lg text-sm font-medium transition">
              <Zap className="w-4 h-4" /> Test Connection
            </button>
          </div>
        )}
      </div>

      {/* Navigation */}
      <div className="flex justify-between">
        <button onClick={() => setStep(Math.max(1, step - 1))} className="px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg text-sm font-medium" disabled={step === 1}>{t("oauthOnboarding.previous")}</button>
        {step < 5 ? (
          <button onClick={() => setStep(Math.min(5, step + 1))} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium">{t("oauthOnboarding.next")}</button>
        ) : (
          <button className="px-4 py-2 bg-green-600 hover:bg-green-700 rounded-lg text-sm font-medium">{t("oauthOnboarding.completeRegistration")}</button>
        )}
      </div>
    </div>
  );
}

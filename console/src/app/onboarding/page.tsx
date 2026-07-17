"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";
import { Building2, Users, KeyRound, Check, ArrowRight, ArrowLeft, Rocket } from "lucide-react";

export default function OnboardingPage() {
  const router = useRouter();
  const { apiFetch } = useApi();
  const t = useTranslations();
  const [step, setStep] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [done, setDone] = useState(false);

  // Form state
  const [orgName, setOrgName] = useState("");
  const [userName, setUserName] = useState("");
  const [userEmail, setUserEmail] = useState("");
  const [userPassword, setUserPassword] = useState("");
  const [apiKeyName, setApiKeyName] = useState("");
  const [generatedKey, setGeneratedKey] = useState("");
  const [generatedKeyId, setGeneratedKeyId] = useState("");

  // Redirect to onboarding if first-time user
  useEffect(() => {
    const completed = localStorage.getItem("ggid_onboarding_completed");
    if (completed === "true") {
      router.push("/");
    }
  }, [router]);

  const handleCreateOrg = async () => {
    setLoading(true);
    setError("");
    try {
      await apiFetch("/api/v1/organizations", {
        method: "POST",
        body: JSON.stringify({ name: orgName }),
      });
      setStep(1);
    } catch {
      // API may not be available — allow skipping
      setStep(1);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateUser = async () => {
    setLoading(true);
    setError("");
    try {
      await apiFetch("/api/v1/auth/register", {
        method: "POST",
        body: JSON.stringify({ username: userName, email: userEmail, password: userPassword }),
      });
      setStep(2);
    } catch {
      setStep(2);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateApiKey = async () => {
    setLoading(true);
    setError("");
    try {
      const data = await apiFetch<{ key?: string; id?: string; secret?: string }>("/api/v1/api-keys", {
        method: "POST",
        body: JSON.stringify({ name: apiKeyName || "Onboarding Key", scopes: ["read", "write"] }),
      });
      setGeneratedKey(data.secret || data.key || "demo-key-not-available");
      setGeneratedKeyId(data.id || "");
    } catch {
      setGeneratedKey("demo-key-not-available");
    } finally {
      setLoading(false);
    }
  };

  const handleFinish = () => {
    localStorage.setItem("ggid_onboarding_completed", "true");
    setDone(true);
    setTimeout(() => router.push("/"), 1500);
  };

  const steps = [
    { icon: Building2, title: t("onboarding.step1Title"), desc: t("onboarding.step1Desc") },
    { icon: Users, title: t("onboarding.step2Title"), desc: t("onboarding.step2Desc") },
    { icon: KeyRound, title: t("onboarding.step3Title"), desc: t("onboarding.step3Desc") },
  ];

  const inputCls = "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 focus:border-brand-500 focus:outline-none";
  const btnCls = "flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50";

  if (done) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-gray-50 dark:bg-gray-950">
        <div className="text-center">
          <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-green-100">
            <Check className="h-8 w-8 text-green-600" />
          </div>
          <h1 className="text-2xl font-bold dark:text-gray-100">{t("onboarding.complete")}</h1>
          <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">{t("onboarding.redirecting")}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50 dark:bg-gray-950 px-4">
      <div className="w-full max-w-lg">
        {/* Progress indicator */}
        <div className="mb-8 flex items-center justify-center gap-2">
          {steps.map((s, i) => (
            <div key={i} className="flex items-center">
              <div className={`flex h-9 w-9 items-center justify-center rounded-full text-sm font-medium ${
                i < step ? "bg-green-500 text-white" : i === step ? "bg-brand-600 text-white" : "bg-gray-200 text-gray-400 dark:bg-gray-700"
              }`}>
                {i < step ? <Check className="h-4 w-4" /> : i + 1}
              </div>
              {i < steps.length - 1 && <div className={`h-0.5 w-12 ${i < step ? "bg-green-500" : "bg-gray-200 dark:bg-gray-700"}`} />}
            </div>
          ))}
        </div>

        <div className="rounded-xl border border-gray-200 bg-white p-8 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          {/* Step header */}
          <div className="mb-6 flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-brand-100 text-brand-600">
              {(() => {
                const Icon = steps[step].icon;
                return <Icon className="h-5 w-5" />;
              })()}
            </div>
            <div>
              <h1 className="text-lg font-semibold dark:text-gray-100">{steps[step].title}</h1>
              <p className="text-sm text-gray-500 dark:text-gray-400">{steps[step].desc}</p>
            </div>
          </div>

          {error && <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400">{error}</div>}

          {/* Step 0: Create Organization */}
          {step === 0 && (
            <div className="space-y-4">
              <div>
                <label className="mb-1 block text-sm font-medium dark:text-gray-300">{t("onboarding.orgName")}</label>
                <input aria-label="Acme Corporation" value={orgName} onChange={(e) => setOrgName(e.target.value)} className={inputCls} placeholder="Acme Corporation" autoFocus />
              </div>
              <div className="flex gap-2">
                <button onClick={() => { setStep(1); }} className="text-sm text-gray-400 hover:text-gray-600">
                  {t("common.skip")}
                </button>
                <button onClick={handleCreateOrg} disabled={loading || !orgName} className={btnCls + " ml-auto"} aria-label="ArrowRight">
                  {loading ? t("common.loading") : "Continue"} <ArrowRight className="h-4 w-4" />
                </button>
              </div>
            </div>
          )}

          {/* Step 1: Add User */}
          {step === 1 && (
            <div className="space-y-4">
              <div>
                <label className="mb-1 block text-sm font-medium dark:text-gray-300">{t("users.usernameLbl")}</label>
                <input aria-label="john.doe" value={userName} onChange={(e) => setUserName(e.target.value)} className={inputCls} placeholder="john.doe" autoFocus />
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium dark:text-gray-300">{t("users.email")}</label>
                <input autoComplete="email" value={userEmail} onChange={(e) => setUserEmail(e.target.value)} type="email" className={inputCls} placeholder="john@acme.com" />
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium dark:text-gray-300">{t("users.passwordLbl")}</label>
                <input autoComplete="current-password" value={userPassword} onChange={(e) => setUserPassword(e.target.value)} type="password" className={inputCls} placeholder="••••••••" />
              </div>
              <div className="flex gap-2">
                <button onClick={() => setStep(0)} className="text-sm text-gray-400 hover:text-gray-600">
                  <ArrowLeft className="mr-1 inline h-3 w-3" />{t("common.back")}
                </button>
                <button onClick={() => { setStep(2); }} className="text-sm text-gray-400 hover:text-gray-600">
                  {t("common.skip")}
                </button>
                <button onClick={handleCreateUser} disabled={loading || !userName} className={btnCls + " ml-auto"} aria-label="ArrowRight">
                  {loading ? t("common.loading") : "Continue"} <ArrowRight className="h-4 w-4" />
                </button>
              </div>
            </div>
          )}

          {/* Step 2: API Key */}
          {step === 2 && (
            <div className="space-y-4">
              {!generatedKey ? (
                <>
                  <div>
                    <label className="mb-1 block text-sm font-medium dark:text-gray-300">{t("onboarding.keyName")}</label>
                    <input aria-label="CI/CD Pipeline" value={apiKeyName} onChange={(e) => setApiKeyName(e.target.value)} className={inputCls} placeholder="CI/CD Pipeline" autoFocus />
                  </div>
                  <div className="flex gap-2">
                    <button onClick={() => setStep(1)} className="text-sm text-gray-400 hover:text-gray-600">
                      <ArrowLeft className="mr-1 inline h-3 w-3" />{t("common.back")}
                    </button>
                    <button onClick={() => { handleFinish(); }} className="text-sm text-gray-400 hover:text-gray-600">
                      {t("common.skip")}
                    </button>
                    <button onClick={handleCreateApiKey} disabled={loading} className={btnCls + " ml-auto"} aria-label="KeyRound">
                      <KeyRound className="h-4 w-4" /> {t("onboarding.generateKey")}
                    </button>
                  </div>
                </>
              ) : (
                <>
                  <div className="rounded-lg border border-green-200 bg-green-50 p-4 dark:border-green-800 dark:bg-green-950">
                    <p className="text-sm font-medium text-green-700 dark:text-green-400">{t("onboarding.keyGenerated")}</p>
                    {generatedKeyId && <p className="mt-1 font-mono text-xs text-gray-500">ID: {generatedKeyId}</p>}
                    <div className="mt-2 flex items-center gap-2">
                      <code className="flex-1 break-all rounded bg-white px-2 py-1 font-mono text-xs dark:bg-gray-800">{generatedKey}</code>
                      <button onClick={() => navigator.clipboard.writeText(generatedKey)} className="rounded-lg border border-gray-300 px-2 py-1 text-xs hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700">{t("common.copy")}</button>
                    </div>
                  </div>
                  <button onClick={handleFinish} className={btnCls + " w-full justify-center"} aria-label="Rocket">
                    <Rocket className="h-4 w-4" /> {t("onboarding.finish")}
                  </button>
                </>
              )}
            </div>
          )}
        </div>

        <p className="mt-4 text-center text-xs text-gray-400 dark:text-gray-500">
          {t("onboarding.footer")}
        </p>
      </div>
    </div>
  );
}

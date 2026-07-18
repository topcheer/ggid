"use client";
import { useState } from "react";
import {
  Rocket, Loader2, AlertCircle, Check, ChevronRight, ChevronLeft,
  User, Building2, Shield, Globe, CheckCircle2, Lock, Zap, Play,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function OnboardingPage() {
  const t = useTranslations();
  const [step, setStep] = useState(0);
  const [done, setDone] = useState(false);
  const [launching, setLaunching] = useState(false);

  // Form state
  const [adminName, setAdminName] = useState("");
  const [adminEmail, setAdminEmail] = useState("");
  const [adminPass, setAdminPass] = useState("");
  const [orgName, setOrgName] = useState("");
  const [mfaRequired, setMfaRequired] = useState(true);
  const [minLength, setMinLength] = useState(12);
  const [sessionTimeout, setSessionTimeout] = useState(60);
  const [ssoEnabled, setSsoEnabled] = useState(false);
  const [ssoProvider, setSsoProvider] = useState("oidc");

  const steps = [t("onboarding.stepAdmin"), t("onboarding.stepOrg"), t("onboarding.stepPolicy"), t("onboarding.stepSso"), t("onboarding.stepReview")];
  const icons = [User, Building2, Shield, Globe, CheckCircle2];
  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const canProceed = () => {
    if (step === 0) return adminName && adminEmail && adminPass.length >= 8;
    if (step === 1) return orgName;
    return true;
  };

  const launch = () => { setLaunching(true); setTimeout(() => { setLaunching(false); setDone(true); }, 1500); };

  if (done) {
    return (
      <div className="flex min-h-screen items-center justify-center px-4">
        <div className={card + " max-w-md text-center"}>
          <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-green-100 dark:bg-green-900/30"><CheckCircle2 className="h-8 w-8 text-green-500" /></div>
          <h1 className="mt-4 text-xl font-bold text-gray-900 dark:text-white">{t("onboarding.complete")}</h1>
          <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">{t("onboarding.completeDesc")}</p>
          <div className="mt-4 rounded-lg bg-gray-50 dark:bg-gray-900/50 p-4 text-left space-y-1 text-xs">
            <p className="flex items-center justify-between"><span className="text-gray-400">{t("onboarding.admin")}</span><span className="font-medium">{adminEmail}</span></p>
            <p className="flex items-center justify-between"><span className="text-gray-400">{t("onboarding.org")}</span><span className="font-medium">{orgName}</span></p>
            <p className="flex items-center justify-between"><span className="text-gray-400">MFA</span><span className="font-medium">{mfaRequired ? t("onboarding.required2") : t("onboarding.optional")}</span></p>
            <p className="flex items-center justify-between"><span className="text-gray-400">{t("onboarding.sso")}</span><span className="font-medium">{ssoEnabled ? ssoProvider.toUpperCase() : t("onboarding.skipped")}</span></p>
          </div>
          <a href="/dashboard" className="mt-4 flex items-center justify-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"><Rocket className="h-4 w-4" /> {t("onboarding.launchConsole")}</a>
        </div>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <div className="text-center"><h1 className="flex items-center justify-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Rocket className="h-7 w-7 text-blue-500" /> {t("onboarding.title")}</h1><p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("onboarding.subtitle")}</p></div>

      {/* Step indicator */}
      <div className="flex items-center justify-center gap-2">{steps.map((label: any, i: any) => { const Icon = icons[i]; return (
        <div key={i} className="flex items-center"><div className="flex flex-col items-center gap-1"><div className={`flex h-9 w-9 items-center justify-center rounded-full ${i < step ? "bg-green-500 text-white" : i === step ? "bg-blue-600 text-white" : "bg-gray-200 dark:bg-gray-700 text-gray-400"}`}>{i < step ? <Check className="h-4 w-4" /> : <Icon className="h-4 w-4" />}</div><span className={`text-xs hidden sm:block ${i === step ? "font-medium text-blue-600" : "text-gray-400"}`}>{label}</span></div>{i < steps.length - 1 && <div className={`h-px w-8 mx-1 ${i < step ? "bg-green-400" : "bg-gray-200 dark:bg-gray-700"}`} />}</div>
      );})}</div>

      {/* Step content */}
      <div className={card}>
        {step === 0 && (<div className="space-y-3"><h2 className="text-lg font-semibold flex items-center gap-2"><User className="h-5 w-5 text-blue-500" /> {t("onboarding.createAdmin")}</h2><div><label className="text-sm font-medium">{t("onboarding.name")}</label><input type="text" value={adminName} onChange={e => setAdminName(e.target.value)} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" autoFocus /></div><div><label className="text-sm font-medium">{t("onboarding.email")}</label><input type="email" value={adminEmail} onChange={e => setAdminEmail(e.target.value)} placeholder="admin@company.com" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div><div><label className="text-sm font-medium">{t("onboarding.password")}</label><input type="password" value={adminPass} onChange={e => setAdminPass(e.target.value)} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" />{adminPass && adminPass.length < 8 && <p className="mt-1 text-xs text-red-500">{t("onboarding.passwordMin")}</p>}</div></div>)}

        {step === 1 && (<div className="space-y-3"><h2 className="text-lg font-semibold flex items-center gap-2"><Building2 className="h-5 w-5 text-blue-500" /> {t("onboarding.setupOrg")}</h2><div><label className="text-sm font-medium">{t("onboarding.orgName")}</label><input type="text" value={orgName} onChange={e => setOrgName(e.target.value)} placeholder="Acme Corporation" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" autoFocus /></div><div><label className="text-sm font-medium">{t("onboarding.tenantId")}</label><input type="text" defaultValue="default" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div></div>)}

        {step === 2 && (<div className="space-y-4"><h2 className="text-lg font-semibold flex items-center gap-2"><Shield className="h-5 w-5 text-blue-500" /> {t("onboarding.authPolicy")}</h2><label className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700"><span className="text-sm font-medium">{t("onboarding.mfaRequired")}</span><button onClick={() => setMfaRequired(!mfaRequired)} aria-pressed={mfaRequired} className={`relative h-6 w-11 rounded-full transition ${mfaRequired ? "bg-green-500" : "bg-gray-300 dark:bg-gray-700"}`}><span className={`absolute top-0.5 h-5 w-5 rounded-full bg-white transition ${mfaRequired ? "left-5" : "left-0.5"}`} /></button></label><div><label className="text-sm font-medium">{t("onboarding.minPasswordLength")}</label><div className="mt-1 flex items-center gap-3"><input type="range" min={8} max={32} value={minLength} onChange={e => setMinLength(parseInt(e.target.value))} className="flex-1 accent-blue-500" /><span className="text-sm font-mono w-8">{minLength}</span></div></div><div><label className="text-sm font-medium">{t("onboarding.sessionTimeout")}</label><div className="mt-1 flex items-center gap-3"><input type="range" min={5} max={480} value={sessionTimeout} onChange={e => setSessionTimeout(parseInt(e.target.value))} className="flex-1 accent-blue-500" /><span className="text-sm font-mono w-12">{sessionTimeout}m</span></div></div></div>)}

        {step === 3 && (<div className="space-y-4"><h2 className="text-lg font-semibold flex items-center gap-2"><Globe className="h-5 w-5 text-blue-500" /> {t("onboarding.identityProviders")}</h2><label className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700"><span className="text-sm font-medium">{t("onboarding.enableSso")}</span><button onClick={() => setSsoEnabled(!ssoEnabled)} aria-pressed={ssoEnabled} className={`relative h-6 w-11 rounded-full transition ${ssoEnabled ? "bg-green-500" : "bg-gray-300 dark:bg-gray-700"}`}><span className={`absolute top-0.5 h-5 w-5 rounded-full bg-white transition ${ssoEnabled ? "left-5" : "left-0.5"}`} /></button></label>{ssoEnabled && (<div><label className="text-sm font-medium">{t("onboarding.provider")}</label><div className="mt-1 flex gap-2">{["oidc", "saml", "google"].map(p => <button key={p} onClick={() => setSsoProvider(p)} aria-pressed={ssoProvider === p} className={`rounded-lg border px-3 py-1.5 text-sm uppercase ${ssoProvider === p ? "border-blue-500 bg-blue-50 dark:bg-blue-950/30 text-blue-600" : "border-gray-300 dark:border-gray-700"}`}>{p}</button>)}</div></div>)}<p className="text-xs text-gray-400">{t("onboarding.ssoOptional")}</p></div>)}

        {step === 4 && (<div className="space-y-3"><h2 className="text-lg font-semibold flex items-center gap-2"><CheckCircle2 className="h-5 w-5 text-blue-500" /> {t("onboarding.review")}</h2><div className="rounded-lg bg-gray-50 dark:bg-gray-900/50 p-4 space-y-2 text-sm"><p className="flex items-center justify-between"><span className="text-gray-400">{t("onboarding.admin")}</span><span className="font-medium">{adminEmail}</span></p><p className="flex items-center justify-between"><span className="text-gray-400">{t("onboarding.org")}</span><span className="font-medium">{orgName}</span></p><p className="flex items-center justify-between"><span className="text-gray-400">MFA</span><span className="font-medium">{mfaRequired ? t("onboarding.required2") : t("onboarding.optional")}</span></p><p className="flex items-center justify-between"><span className="text-gray-400">{t("onboarding.pwdPolicy")}</span><span className="font-medium">≥ {minLength} chars</span></p><p className="flex items-center justify-between"><span className="text-gray-400">{t("onboarding.session")}</span><span className="font-medium">{sessionTimeout}min</span></p><p className="flex items-center justify-between"><span className="text-gray-400">{t("onboarding.sso")}</span><span className="font-medium">{ssoEnabled ? ssoProvider.toUpperCase() : t("onboarding.skipped")}</span></p></div></div>)}
      </div>

      {/* Navigation */}
      <div className="flex justify-between">
        <button onClick={() => setStep(Math.max(0, step - 1))} disabled={step === 0} className="flex items-center gap-1 rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700 disabled:opacity-30"><ChevronLeft className="h-4 w-4" /> {t("onboarding.back")}</button>
        {step < 4 ? (
          <button onClick={() => setStep(step + 1)} disabled={!canProceed()} className="flex items-center gap-1 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50">{t("onboarding.next")} <ChevronRight className="h-4 w-4" /></button>
        ) : (
          <button onClick={launch} disabled={launching} className="flex items-center gap-2 rounded-lg bg-green-600 px-4 py-2 text-sm font-medium text-white hover:bg-green-700 disabled:opacity-50">{launching ? <Loader2 className="h-4 w-4 animate-spin" /> : <Rocket className="h-4 w-4" />} {t("onboarding.finish")}</button>
        )}
      </div>
    </div>
  );
}

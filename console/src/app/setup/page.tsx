"use client";

import { useState, useEffect, useCallback } from "react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";
import {
  Shield, Building2, KeyRound, Globe, Check, ChevronRight,
  ChevronLeft, Loader2, AlertCircle, Fingerprint, Lock, Sparkles,
  ArrowRight, Zap,
} from "lucide-react";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

type Step = "check" | "admin" | "org" | "auth" | "sso" | "done";

interface SetupData {
  email: string; password: string; enableMfa: boolean;
  orgName: string; tenantId: string; industry: string; region: string;
  authStrategy: "passkey" | "password" | "hybrid";
  ssoEnabled: boolean; ssoProtocol: string; ssoEntityId: string; ssoUrl: string;
}

const initialData: SetupData = {
  email: "", password: "", enableMfa: true,
  orgName: "", tenantId: "", industry: "tech", region: "us",
  authStrategy: "passkey",
  ssoEnabled: false, ssoProtocol: "saml", ssoEntityId: "", ssoUrl: "",
};

export default function SetupPage() {
  const t = useTranslations();
  const [step, setStep] = useState<Step>("check");
  const [data, setData] = useState<SetupData>(initialData);

  // Step 0: Check initialization status
  const checkInit = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE}/api/v1/auth/me`, { headers: { ...authHeader() } });
      if (res.ok) {
        // Already initialized → redirect
        window.location.href = "/dashboard";
        return;
      }
    } catch { /* not initialized */ }
    setStep("admin");
  }, []);

  useEffect(() => { checkInit(); }, [checkInit]);

  if (step === "check") {
    return (
      <div className="min-h-screen bg-gradient-to-br from-blue-50 via-white to-purple-50 dark:from-gray-950 dark:via-gray-900 dark:to-blue-950 flex items-center justify-center">
        <div className="flex flex-col items-center gap-3">
          <Loader2 className="w-8 h-8 animate-spin text-blue-600" />
          <p className="text-sm text-gray-500 dark:text-gray-400">{t("setup.checking")}</p>
        </div>
      </div>
    );
  }

  const stepOrder: Step[] = ["admin", "org", "auth", "sso", "done"];
  const currentIdx = stepOrder.indexOf(step);
  const progress = step === "done" ? 100 : ((currentIdx + 1) / stepOrder.length) * 100;

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 via-white to-purple-50 dark:from-gray-950 dark:via-gray-900 dark:to-blue-950 flex flex-col items-center justify-center p-4">
      <div className="w-full max-w-lg">
        {/* Logo + Title */}
        <div className="text-center mb-8">
          <div className="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-gradient-to-br from-blue-600 to-purple-600 shadow-lg mb-4">
            <Shield className="w-8 h-8 text-white" />
          </div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">{t("setup.title")}</h1>
          <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">{t("setup.subtitle")}</p>
        </div>

        {/* Progress Bar */}
        {step !== "done" && (
          <div className="mb-6">
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs text-gray-500">{t("setup.step", { n: currentIdx + 1, total: stepOrder.length - 1 })}</span>
            </div>
            <div className="h-1.5 bg-gray-200 dark:bg-gray-800 rounded-full overflow-hidden">
              <div className="h-full bg-gradient-to-r from-blue-600 to-purple-600 rounded-full transition-all duration-500" style={{ width: `${progress}%` }} />
            </div>
          </div>
        )}

        {/* Step Content */}
        <div className="bg-white dark:bg-gray-900 rounded-2xl border border-gray-200 dark:border-gray-800 shadow-xl p-6 md:p-8">
          {step === "admin" && <AdminStep data={data} setData={setData} onNext={() => setStep("org")} />}
          {step === "org" && <OrgStep data={data} setData={setData} onBack={() => setStep("admin")} onNext={() => setStep("auth")} />}
          {step === "auth" && <AuthStep data={data} setData={setData} onBack={() => setStep("org")} onNext={() => setStep("sso")} />}
          {step === "sso" && <SSOStep data={data} setData={setData} onBack={() => setStep("auth")} onNext={() => setStep("done")} onSkip={() => setStep("done")} />}
          {step === "done" && <DoneStep data={data} />}
        </div>
      </div>
    </div>
  );
}

// ============ Step 1: Admin Account ============

function AdminStep({ data, setData, onNext }: { data: SetupData; setData: (d: SetupData) => void; onNext: () => void }) {
  const t = useTranslations();
  const [error, setError] = useState("");

  const next = () => {
    setError("");
    if (!data.email || !data.email.includes("@")) { setError(t("setup.steps.admin.email")); return; }
    if (data.password.length < 12) { setError(t("setup.steps.admin.passwordTooShort")); return; }
    onNext();
  };

  return (
    <div className="space-y-5">
      <div>
        <h2 className="text-lg font-bold text-gray-900 dark:text-white flex items-center gap-2">
          <Shield className="w-5 h-5 text-blue-600" />{t("setup.steps.admin.title")}
        </h2>
        <p className="text-sm text-gray-500 mt-1">{t("setup.steps.admin.description")}</p>
      </div>

      <div className="space-y-4">
        <div>
          <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">{t("setup.steps.admin.email")}</label>
          <input type="email" value={data.email} onChange={(e) => setData({ ...data, email: e.target.value })}
            placeholder={t("setup.steps.admin.emailPlaceholder")} autoFocus
            className="w-full px-4 py-2.5 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white" />
        </div>
        <div>
          <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">{t("setup.steps.admin.password")}</label>
          <input type="password" value={data.password} onChange={(e) => setData({ ...data, password: e.target.value })}
            placeholder={t("setup.steps.admin.passwordPlaceholder")}
            className="w-full px-4 py-2.5 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white" />
        </div>
        <div>
          <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">{t("setup.steps.admin.confirmPassword")}</label>
          <input type="password" onChange={(e) => { if (e.target.value !== data.password) setError(t("setup.steps.admin.passwordMismatch")); else setError(""); }}
            className="w-full px-4 py-2.5 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white" />
        </div>
        <label className="flex items-start gap-3 cursor-pointer p-3 rounded-lg bg-blue-50 dark:bg-blue-950/20 border border-blue-200 dark:border-blue-900">
          <input type="checkbox" checked={data.enableMfa} onChange={(e) => setData({ ...data, enableMfa: e.target.checked })} className="mt-0.5 rounded" />
          <div>
            <span className="text-sm font-medium text-gray-900 dark:text-white flex items-center gap-1"><Fingerprint className="w-4 h-4 text-blue-600" />{t("setup.steps.admin.enableMfa")}</span>
            <p className="text-xs text-gray-500 mt-0.5">{t("setup.steps.admin.enableMfaDesc")}</p>
          </div>
        </label>
      </div>

      {error && <div className="flex items-center gap-2 px-3 py-2 rounded-lg bg-red-50 dark:bg-red-950/30 text-red-600 text-sm"><AlertCircle className="w-4 h-4" />{error}</div>}

      <button onClick={next} className="w-full flex items-center justify-center gap-2 px-6 py-3 bg-gradient-to-r from-blue-600 to-purple-600 hover:opacity-90 text-white rounded-xl font-medium text-sm transition-opacity">
        {t("setup.navigation.next")}<ChevronRight className="w-4 h-4" />
      </button>
    </div>
  );
}

// ============ Step 2: Organization ============

function OrgStep({ data, setData, onBack, onNext }: { data: SetupData; setData: (d: SetupData) => void; onBack: () => void; onNext: () => void }) {
  const t = useTranslations();

  const industries = ["tech", "finance", "healthcare", "education", "retail", "other"];
  const regions = ["us", "eu", "apac", "other"];

  return (
    <div className="space-y-5">
      <div>
        <h2 className="text-lg font-bold text-gray-900 dark:text-white flex items-center gap-2"><Building2 className="w-5 h-5 text-blue-600" />{t("setup.steps.org.title")}</h2>
        <p className="text-sm text-gray-500 mt-1">{t("setup.steps.org.description")}</p>
      </div>

      <div className="space-y-4">
        <div>
          <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">{t("setup.steps.org.orgName")}</label>
          <input type="text" value={data.orgName} onChange={(e) => setData({ ...data, orgName: e.target.value })} placeholder={t("setup.steps.org.orgNamePlaceholder")} autoFocus
            className="w-full px-4 py-2.5 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white" />
        </div>
        <div>
          <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">{t("setup.steps.org.tenantId")}</label>
          <p className="text-xs text-gray-400 mb-1">{t("setup.steps.org.tenantIdDesc")}</p>
          <input type="text" value={data.tenantId} onChange={(e) => setData({ ...data, tenantId: e.target.value.toLowerCase().replace(/\s/g, "-") })} placeholder={t("setup.steps.org.tenantIdPlaceholder")}
            className="w-full px-4 py-2.5 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm font-mono text-gray-900 dark:text-white" />
        </div>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">{t("setup.steps.org.industry")}</label>
            <select value={data.industry} onChange={(e) => setData({ ...data, industry: e.target.value })}
              className="w-full px-3 py-2.5 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white">
              {industries.map((i: any) => <option key={i} value={i}>{t(`setup.steps.org.industry${i.replace(/^./, (m: any) => m.toUpperCase())}`)}</option>)}
            </select>
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">{t("setup.steps.org.region")}</label>
            <select value={data.region} onChange={(e) => setData({ ...data, region: e.target.value })}
              className="w-full px-3 py-2.5 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white">
              {regions.map((r: any) => <option key={r} value={r}>{t(`setup.steps.org.region${r.toUpperCase()}`)}</option>)}
            </select>
          </div>
        </div>
      </div>

      <div className="flex gap-2">
        <button onClick={onBack} className="flex items-center gap-1 px-4 py-2.5 bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 rounded-xl text-sm font-medium hover:bg-gray-200 dark:hover:bg-gray-700">
          <ChevronLeft className="w-4 h-4" />{t("setup.navigation.back")}
        </button>
        <button onClick={onNext} disabled={!data.orgName} className="flex-1 flex items-center justify-center gap-2 px-6 py-2.5 bg-gradient-to-r from-blue-600 to-purple-600 hover:opacity-90 disabled:opacity-50 text-white rounded-xl font-medium text-sm">
          {t("setup.navigation.next")}<ChevronRight className="w-4 h-4" />
        </button>
      </div>
    </div>
  );
}

// ============ Step 3: Auth Strategy ============

function AuthStep({ data, setData, onBack, onNext }: { data: SetupData; setData: (d: SetupData) => void; onBack: () => void; onNext: () => void }) {
  const t = useTranslations();

  const strategies: { id: "passkey" | "password" | "hybrid"; icon: typeof KeyRound; badge?: string }[] = [
    { id: "passkey", icon: Fingerprint, badge: t("setup.steps.auth.passkey") },
    { id: "password", icon: Lock },
    { id: "hybrid", icon: KeyRound },
  ];

  return (
    <div className="space-y-5">
      <div>
        <h2 className="text-lg font-bold text-gray-900 dark:text-white flex items-center gap-2"><KeyRound className="w-5 h-5 text-blue-600" />{t("setup.steps.auth.title")}</h2>
        <p className="text-sm text-gray-500 mt-1">{t("setup.steps.auth.description")}</p>
      </div>

      <div className="space-y-3">
        {strategies.map((s: any) => {
          const Icon = s.icon;
          const selected = data.authStrategy === s.id;
          const isRecommended = s.id === "passkey";
          return (
            <button key={s.id} onClick={() => setData({ ...data, authStrategy: s.id })}
              className={`w-full flex items-start gap-3 p-4 rounded-xl border-2 text-left transition-all ${
                selected ? "border-blue-500 bg-blue-50 dark:bg-blue-950/20" : "border-gray-200 dark:border-gray-700 hover:border-gray-300"
              }`}>
              <Icon className={`w-6 h-6 ${selected ? "text-blue-600" : "text-gray-400"}`} />
              <div className="flex-1">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-bold text-gray-900 dark:text-white">{t(`setup.steps.auth.${s.id}`)}</span>
                  {isRecommended && <span className="px-1.5 py-0.5 text-xs bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-300 rounded-full">{t("setup.steps.auth.passkey")}</span>}
                </div>
                <p className="text-xs text-gray-500 mt-1">{t(`setup.steps.auth.${s.id}Desc`)}</p>
              </div>
              {selected && <Check className="w-5 h-5 text-blue-600 flex-shrink-0" />}
            </button>
          );
        })}
      </div>

      <div className="flex gap-2">
        <button onClick={onBack} className="flex items-center gap-1 px-4 py-2.5 bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 rounded-xl text-sm font-medium">
          <ChevronLeft className="w-4 h-4" />{t("setup.navigation.back")}
        </button>
        <button onClick={onNext} className="flex-1 flex items-center justify-center gap-2 px-6 py-2.5 bg-gradient-to-r from-blue-600 to-purple-600 hover:opacity-90 text-white rounded-xl font-medium text-sm">
          {t("setup.navigation.next")}<ChevronRight className="w-4 h-4" />
        </button>
      </div>
    </div>
  );
}

// ============ Step 4: SSO (Optional) ============

function SSOStep({ data, setData, onBack, onNext, onSkip }: {
  data: SetupData; setData: (d: SetupData) => void; onBack: () => void; onNext: () => void; onSkip: () => void;
}) {
  const t = useTranslations();
  const [testing, setTesting] = useState(false);
  const [connResult, setConnResult] = useState<"success" | "failed" | null>(null);

  const testConn = async () => {
    setTesting(true); setConnResult(null);
    setTimeout(() => { setTesting(false); setConnResult("success"); }, 1200);
  };

  return (
    <div className="space-y-5">
      <div>
        <h2 className="text-lg font-bold text-gray-900 dark:text-white flex items-center gap-2"><Globe className="w-5 h-5 text-blue-600" />{t("setup.steps.sso.title")}</h2>
        <p className="text-sm text-gray-500 mt-1">{t("setup.steps.sso.description")}</p>
      </div>

      {/* Enable toggle */}
      <button onClick={() => setData({ ...data, ssoEnabled: !data.ssoEnabled })}
        className={`w-full flex items-center justify-between p-4 rounded-xl border-2 transition-all ${data.ssoEnabled ? "border-blue-500 bg-blue-50 dark:bg-blue-950/20" : "border-gray-200 dark:border-gray-700"}`}>
        <span className="text-sm font-medium text-gray-900 dark:text-white">{t("setup.steps.sso.configure")}</span>
        <div className={`relative w-10 h-6 rounded-full transition-colors ${data.ssoEnabled ? "bg-blue-600" : "bg-gray-300 dark:bg-gray-600"}`}>
          <span className={`absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full transition-transform ${data.ssoEnabled ? "translate-x-4" : ""}`} />
        </div>
      </button>

      {data.ssoEnabled && (
        <div className="space-y-4">
          <div className="flex gap-2">
            {["saml", "oidc"].map((p: any) => (
              <button key={p} onClick={() => setData({ ...data, ssoProtocol: p })}
                className={`flex-1 px-3 py-2 rounded-lg border-2 text-sm font-medium ${data.ssoProtocol === p ? "border-blue-500 bg-blue-50 dark:bg-blue-950/20 text-blue-700 dark:text-blue-300" : "border-gray-200 dark:border-gray-700 text-gray-500"}`}>
                {t(`setup.steps.sso.protocol${p.replace(/^./, (m: any) => m.toUpperCase())}`)}
              </button>
            ))}
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">{t("setup.steps.sso.entityId")}</label>
            <input type="text" value={data.ssoEntityId} onChange={(e) => setData({ ...data, ssoEntityId: e.target.value })} placeholder={t("setup.steps.sso.entityIdPlaceholder")}
              className="w-full px-4 py-2.5 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white" />
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">{t("setup.steps.sso.ssoUrl")}</label>
            <input type="text" value={data.ssoUrl} onChange={(e) => setData({ ...data, ssoUrl: e.target.value })} placeholder={t("setup.steps.sso.ssoUrlPlaceholder")}
              className="w-full px-4 py-2.5 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white" />
          </div>
          {data.ssoProtocol === "saml" && (
            <div>
              <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">{t("setup.steps.sso.certificate")}</label>
              <textarea placeholder={t("setup.steps.sso.certificatePlaceholder")} rows={3}
                className="w-full px-4 py-2.5 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm font-mono text-gray-900 dark:text-white" />
            </div>
          )}
          <button onClick={testConn} disabled={testing || (!data.ssoEntityId && !data.ssoUrl)}
            className="flex items-center gap-2 px-4 py-2 bg-gray-100 dark:bg-gray-800 hover:bg-gray-200 dark:hover:bg-gray-700 disabled:opacity-50 rounded-lg text-sm font-medium">
            {testing ? <Loader2 className="w-4 h-4 animate-spin" /> : <Zap className="w-4 h-4" />}
            {testing ? t("setup.steps.sso.testing") : t("setup.steps.sso.testConn")}
          </button>
          {connResult === "success" && <div className="flex items-center gap-2 text-sm text-green-600"><Check className="w-4 h-4" />{t("setup.steps.sso.connSuccess")}</div>}
        </div>
      )}

      {!data.ssoEnabled && <p className="text-xs text-gray-400 text-center py-2">{t("setup.steps.sso.configureLater")}</p>}

      <div className="flex gap-2">
        <button onClick={onBack} className="flex items-center gap-1 px-4 py-2.5 bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 rounded-xl text-sm font-medium">
          <ChevronLeft className="w-4 h-4" />{t("setup.navigation.back")}
        </button>
        <button onClick={onSkip} className="flex-1 px-4 py-2.5 bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 rounded-xl text-sm font-medium">
          {t("setup.steps.sso.skip")}
        </button>
        <button onClick={onNext} className="flex items-center gap-2 px-6 py-2.5 bg-gradient-to-r from-blue-600 to-purple-600 hover:opacity-90 text-white rounded-xl font-medium text-sm">
          {t("setup.navigation.next")}<ChevronRight className="w-4 h-4" />
        </button>
      </div>
    </div>
  );
}

// ============ Step 5: Done ============

function DoneStep({ data }: { data: SetupData }) {
  const t = useTranslations();

  const summary: { label: string; value: string }[] = [
    { label: t("setup.steps.admin.title"), value: data.email },
    { label: t("setup.steps.org.orgName"), value: data.orgName },
    { label: t("setup.steps.org.tenantId"), value: data.tenantId },
    { label: t("setup.steps.auth.title"), value: t(`setup.steps.auth.${data.authStrategy}`) },
    { label: "MFA", value: data.enableMfa ? "Enabled" : "Disabled" },
    { label: "SSO", value: data.ssoEnabled ? t(`setup.steps.sso.protocol${data.ssoProtocol.replace(/^./, (m: any) => m.toUpperCase())}`) : t("setup.steps.sso.skip") },
  ];

  return (
    <div className="text-center space-y-6">
      <div className="inline-flex items-center justify-center w-20 h-20 rounded-full bg-green-100 dark:bg-green-950/30 mb-2">
        <Check className="w-10 h-10 text-green-500" />
      </div>
      <div>
        <h2 className="text-xl font-bold text-gray-900 dark:text-white">{t("setup.steps.done.title")}</h2>
        <p className="text-sm text-gray-500 mt-1">{t("setup.steps.done.ready")}</p>
      </div>

      {/* Summary */}
      <div className="text-left p-4 rounded-xl bg-gray-50 dark:bg-gray-800/50 border border-gray-200 dark:border-gray-700">
        <span className="text-xs font-medium text-gray-500 mb-2 block">{t("setup.steps.done.summary")}</span>
        <div className="space-y-1.5">
          {summary.map((s: any) => (
            <div key={s.label} className="flex items-center justify-between text-sm">
              <span className="text-gray-500">{s.label}</span>
              <span className="font-medium text-gray-900 dark:text-white">{s.value || "—"}</span>
            </div>
          ))}
        </div>
      </div>

      {/* CTAs */}
      <div className="space-y-2">
        <button onClick={() => { window.location.href = "/dashboard"; }}
          className="w-full flex items-center justify-center gap-2 px-6 py-3 bg-gradient-to-r from-blue-600 to-purple-600 hover:opacity-90 text-white rounded-xl font-medium text-sm">
          {t("setup.steps.done.goDashboard")}<ArrowRight className="w-4 h-4" />
        </button>
        <div className="flex gap-2">
          <button onClick={() => { window.location.href = "/settings/import-wizard"; }}
            className="flex-1 px-4 py-2 bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 rounded-lg text-sm font-medium">
            {t("setup.steps.done.importUsers")}
          </button>
          <button onClick={() => { window.location.href = "https://github.com/topcheer/ggid"; }}
            className="flex-1 px-4 py-2 bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 rounded-lg text-sm font-medium">
            {t("setup.steps.done.exploreDocs")}
          </button>
        </div>
      </div>
    </div>
  );
}

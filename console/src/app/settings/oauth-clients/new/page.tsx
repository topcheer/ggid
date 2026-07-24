"use client";

import { useState } from "react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";
import {
  Monitor, Smartphone, Globe, Cpu, ChevronLeft, ChevronRight,
  Check, Loader2, KeyRound, Shield, Lock, Copy, AlertCircle,
  Terminal, ArrowRight, Sparkles,
} from "lucide-react";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "";
type Step = 0 | 1 | 2 | 3 | 4;

type AppType = "web" | "spa" | "mobile" | "m2m";
type AuthMethod = "client_secret" | "pkce" | "mtls";

interface CreatedClient { client_id: string; client_secret: string; }

const appTypes: { id: AppType; icon: typeof Monitor; labelKey: string; descKey: string }[] = [
  { id: "web", icon: Globe, labelKey: "web", descKey: "webDesc" },
  { id: "spa", icon: Monitor, labelKey: "spa", descKey: "spaDesc" },
  { id: "mobile", icon: Smartphone, labelKey: "mobile", descKey: "mobileDesc" },
  { id: "m2m", icon: Cpu, labelKey: "m2m", descKey: "m2mDesc" },
];

const authMethods: { id: AuthMethod; icon: typeof KeyRound; labelKey: string; descKey: string; recommended?: boolean }[] = [
  { id: "pkce", icon: Shield, labelKey: "pkce", descKey: "pkceDesc", recommended: true },
  { id: "client_secret", icon: KeyRound, labelKey: "clientSecret", descKey: "clientSecretDesc" },
  { id: "mtls", icon: Lock, labelKey: "mtls", descKey: "mtlsDesc" },
];

const standardScopes = [
  { id: "openid", labelKey: "openid", descKey: "openidDesc", required: true },
  { id: "profile", labelKey: "profile", descKey: "profileDesc" },
  { id: "email", labelKey: "email", descKey: "emailDesc" },
  { id: "roles", labelKey: "roles", descKey: "rolesDesc" },
  { id: "offline_access", labelKey: "offline", descKey: "offlineDesc" },
];

export default function OAuthClientNewPage() {
  const t = useTranslations();
  const [step, setStep] = useState<Step>(0);
  const [appType, setAppType] = useState<AppType | "">("");
  const [authMethod, setAuthMethod] = useState<AuthMethod | "">("");
  const [selectedScopes, setSelectedScopes] = useState<Set<string>>(new Set(["openid"]));
  const [customScopes, setCustomScopes] = useState("");
  const [clientName, setClientName] = useState("");
  const [redirectUris, setRedirectUris] = useState("");
  const [creating, setCreating] = useState(false);
  const [created, setCreated] = useState<CreatedClient | null>(null);
  const [error, setError] = useState("");
  const [copied, setCopied] = useState(false);

  const steps = [
    { label: t("oauthWizard.steps.appType") },
    { label: t("oauthWizard.steps.authMethod") },
    { label: t("oauthWizard.steps.scopes") },
    { label: t("oauthWizard.steps.confirm") },
  ];

  const toggleScope = (id: string) => {
    if (id === "openid") return; // always required
    const next = new Set(selectedScopes);
    if (next.has(id)) next.delete(id); else next.add(id);
    setSelectedScopes(next);
  };

  const create = async () => {
    setError("");
    if (!clientName) { setError(t("oauthWizard.confirm.clientName")); return; }
    setCreating(true);
    try {
      const scopes = [...selectedScopes];
      if (customScopes.trim()) scopes.push(...customScopes.trim().split(/\s+/));
      const body: Record<string, unknown> = {
        client_name: clientName,
        redirect_uris: redirectUris.split("\n").map((u: any) => u.trim()).filter(Boolean),
        grant_types: appType === "m2m" ? ["client_credentials"] : ["authorization_code"],
        response_types: appType === "m2m" ? ["token"] : ["code"],
        token_endpoint_auth_method: authMethod,
        scopes,
      };
      if (appType === "spa" || appType === "mobile") {
        body.token_endpoint_auth_method = "none"; // PKCE
      }
      const res = await fetch(`${API_BASE}/api/v1/oauth/clients`, {
        method: "POST", headers: { "Content-Type": "application/json", ...authHeader() },
        body: JSON.stringify(body),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error?.detail || data.error || "Failed");
      setCreated({ client_id: data.client_id || data.id, client_secret: data.client_secret || data.secret || "" });
      setStep(4);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to create client");
    } finally { setCreating(false); }
  };

  const canNext = () => {
    if (step === 0) return !!appType;
    if (step === 1) return !!authMethod || appType === "spa" || appType === "m2m";
    if (step === 2) return selectedScopes.size > 0;
    if (step === 3) return !!clientName;
    return false;
  };

  const copyAll = () => {
    if (!created) return;
    navigator.clipboard.writeText(`Client ID: ${created.client_id}\nClient Secret: ${created.client_secret}`);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-800 dark:bg-gray-950 p-4 md:p-8">
      <div className="max-w-2xl mx-auto">
        {/* Header */}
        <div className="mb-6">
          <div className="flex items-center gap-3 mb-1">
            <KeyRound className="w-7 h-7 text-blue-600" />
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white dark:text-white">{t("oauthWizard.title")}</h1>
          </div>
          <p className="text-sm text-gray-500 dark:text-gray-400">{t("oauthWizard.subtitle")}</p>
        </div>

        {/* Stepper */}
        {step < 4 && (
          <div className="flex items-center gap-2 mb-8">
            {steps.map((s: any, i: any) => {
              const isActive = step === i;
              const isPast = step > i;
              return (
                <div key={i} className="flex items-center gap-2 flex-1">
                  {i > 0 && <div className={`h-0.5 flex-1 ${isPast ? "bg-green-500" : "bg-gray-200 dark:bg-gray-700"}`} />}
                  <div className="flex items-center gap-2">
                    <div className={`w-8 h-8 rounded-full flex items-center justify-center text-xs font-medium ${isActive ? "bg-blue-600 text-white" : isPast ? "bg-green-500 text-white" : "bg-gray-200 dark:bg-gray-700 text-gray-400"}`}>
                      {isPast ? <Check className="w-4 h-4" /> : i + 1}
                    </div>
                    <span className={`text-xs hidden sm:inline ${isActive ? "text-blue-600 font-medium" : "text-gray-500"}`}>{s.label}</span>
                  </div>
                </div>
              );
            })}
          </div>
        )}

        {/* Step Content */}
        <div className="bg-white dark:bg-gray-800 dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-700 dark:border-gray-800 p-6">
          {error && <div className="flex items-center gap-2 px-4 py-2 mb-4 rounded-lg bg-red-50 dark:bg-red-950/30 text-red-600 text-sm"><AlertCircle className="w-4 h-4" />{error}</div>}

          {/* Step 0: App Type */}
          {step === 0 && (
            <div className="space-y-3">
              <h3 className="text-sm font-semibold text-gray-900 dark:text-white dark:text-white mb-3">{t("oauthWizard.appType.title")}</h3>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                {appTypes.map((a: any) => {
                  const Icon = a.icon;
                  const selected = appType === a.id;
                  return (
                    <button key={a.id} onClick={() => { setAppType(a.id); if (a.id === "spa") setAuthMethod("pkce"); if (a.id === "m2m") setAuthMethod("client_secret"); }}
                      className={`flex items-start gap-3 p-4 rounded-xl border-2 text-left transition-all ${selected ? "border-blue-500 bg-blue-50 dark:bg-blue-950/20" : "border-gray-200 dark:border-gray-700 hover:border-gray-300"}`}>
                      <Icon className={`w-6 h-6 ${selected ? "text-blue-600" : "text-gray-400"}`} />
                      <div>
                        <span className="text-sm font-bold text-gray-900 dark:text-white dark:text-white">{t(`oauthWizard.appType.${a.labelKey}`)}</span>
                        <p className="text-xs text-gray-400 mt-0.5">{t(`oauthWizard.appType.${a.descKey}`)}</p>
                      </div>
                      {selected && <Check className="w-5 h-5 text-blue-600 ml-auto" />}
                    </button>
                  );
                })}
              </div>
            </div>
          )}

          {/* Step 1: Auth Method */}
          {step === 1 && (
            <div className="space-y-3">
              <h3 className="text-sm font-semibold text-gray-900 dark:text-white dark:text-white mb-3">{t("oauthWizard.authMethod.title")}</h3>
              {authMethods.map((m: any) => {
                const Icon = m.icon;
                const selected = authMethod === m.id;
                const disabled = (appType === "spa" && m.id !== "pkce") || (appType === "m2m" && m.id !== "client_secret");
                return (
                  <button key={m.id} onClick={() => !disabled && setAuthMethod(m.id)} disabled={disabled}
                    className={`w-full flex items-start gap-3 p-4 rounded-xl border-2 text-left transition-all ${selected ? "border-blue-500 bg-blue-50 dark:bg-blue-950/20" : disabled ? "border-gray-100 dark:border-gray-800 opacity-40 cursor-not-allowed" : "border-gray-200 dark:border-gray-700 hover:border-gray-300"}`}>
                    <Icon className={`w-6 h-6 ${selected ? "text-blue-600" : "text-gray-400"}`} />
                    <div className="flex-1">
                      <div className="flex items-center gap-2">
                        <span className="text-sm font-bold text-gray-900 dark:text-white dark:text-white">{t(`oauthWizard.authMethod.${m.labelKey}`)}</span>
                        {m.recommended && <span className="px-1.5 py-0.5 text-xs bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-300 rounded">{t("oauthWizard.authMethod.pkce")}</span>}
                      </div>
                      <p className="text-xs text-gray-400 mt-0.5">{t(`oauthWizard.authMethod.${m.descKey}`)}</p>
                    </div>
                    {selected && <Check className="w-5 h-5 text-blue-600" />}
                  </button>
                );
              })}
            </div>
          )}

          {/* Step 2: Scopes */}
          {step === 2 && (
            <div className="space-y-3">
              <h3 className="text-sm font-semibold text-gray-900 dark:text-white dark:text-white">{t("oauthWizard.scopes.title")}</h3>
              <p className="text-xs text-gray-500 dark:text-gray-400 mb-3">{t("oauthWizard.scopes.description")}</p>
              <div className="space-y-2">
                {standardScopes.map((s: any) => {
                  const selected = selectedScopes.has(s.id);
                  return (
                    <label key={s.id} className={`flex items-center gap-3 p-3 rounded-lg border-2 cursor-pointer transition-all ${selected ? "border-blue-500 bg-blue-50 dark:bg-blue-950/20" : "border-gray-200 dark:border-gray-700 hover:border-gray-300"}`}>
                      <input type="checkbox" checked={selected} onChange={() => toggleScope(s.id)} disabled={s.required} className="rounded" />
                      <div className="flex-1">
                        <span className="text-sm font-medium text-gray-900 dark:text-white dark:text-white">{t(`oauthWizard.scopes.${s.labelKey}`)}{s.required && <span className="text-xs text-gray-400 ml-1">(required)</span>}</span>
                        <p className="text-xs text-gray-400">{t(`oauthWizard.scopes.${s.descKey}`)}</p>
                      </div>
                    </label>
                  );
                })}
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 dark:text-gray-400 mb-1">{t("oauthWizard.scopes.custom")}</label>
                <p className="text-xs text-gray-400 mb-1">{t("oauthWizard.scopes.customDesc")}</p>
                <input type="text" value={customScopes} onChange={(e) => setCustomScopes(e.target.value)} placeholder="read:users write:audit"
                  className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 dark:bg-gray-800 text-sm text-gray-900 dark:text-white dark:text-white" />
              </div>
            </div>
          )}

          {/* Step 3: Confirm + Create */}
          {step === 3 && (
            <div className="space-y-4">
              <h3 className="text-sm font-semibold text-gray-900 dark:text-white dark:text-white mb-3">{t("oauthWizard.confirm.title")}</h3>
              <div className="space-y-3">
                <div>
                  <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 dark:text-gray-400 mb-1">{t("oauthWizard.confirm.clientName")}</label>
                  <input type="text" value={clientName} onChange={(e) => setClientName(e.target.value)} placeholder={t("oauthWizard.confirm.clientNamePlaceholder")} autoFocus
                    className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 dark:bg-gray-800 text-sm text-gray-900 dark:text-white dark:text-white" />
                </div>
                <div>
                  <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 dark:text-gray-400 mb-1">{t("oauthWizard.confirm.redirectUris")}</label>
                  <p className="text-xs text-gray-400 mb-1">{t("oauthWizard.confirm.redirectUrisDesc")}</p>
                  <textarea value={redirectUris} onChange={(e) => setRedirectUris(e.target.value)} placeholder={t("oauthWizard.confirm.redirectUrisPlaceholder")} rows={3}
                    className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 dark:bg-gray-800 text-sm font-mono text-gray-900 dark:text-white dark:text-white" />
                </div>
                <div className="grid grid-cols-2 gap-3 p-3 rounded-lg bg-gray-50 dark:bg-gray-800 dark:bg-gray-800/50">
                  <div><span className="text-xs text-gray-400">{t("oauthWizard.confirm.appTypeLabel")}</span><p className="text-sm font-medium text-gray-900 dark:text-white dark:text-white capitalize">{t(`oauthWizard.appType.${appType}`)}</p></div>
                  <div><span className="text-xs text-gray-400">{t("oauthWizard.confirm.authMethodLabel")}</span><p className="text-sm font-medium text-gray-900 dark:text-white dark:text-white">{authMethod ? t(`oauthWizard.authMethod.${authMethod === "client_secret" ? "clientSecret" : authMethod === "pkce" ? "pkce" : "mtls"}`) : "—"}</p></div>
                </div>
              </div>
              <button onClick={create} disabled={creating || !clientName}
                className="w-full flex items-center justify-center gap-2 px-6 py-3 bg-gradient-to-r from-blue-600 to-purple-600 hover:opacity-90 disabled:opacity-50 text-white rounded-xl font-medium text-sm">
                {creating ? <Loader2 className="w-4 h-4 animate-spin" /> : <Sparkles className="w-4 h-4" />}
                {creating ? t("oauthWizard.confirm.creating") : t("oauthWizard.confirm.create")}
              </button>
            </div>
          )}

          {/* Step 4: Success — show credentials */}
          {step === 4 && created && (
            <div className="text-center space-y-4">
              <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-green-100 dark:bg-green-950/30 mb-2">
                <Check className="w-8 h-8 text-green-500" />
              </div>
              <h3 className="text-lg font-bold text-gray-900 dark:text-white dark:text-white">{t("oauthWizard.confirm.created")}</h3>

              <div className="text-left space-y-3">
                {/* Secret warning */}
                <div className="flex items-center gap-2 p-3 rounded-lg bg-yellow-50 dark:bg-yellow-950/30 border border-yellow-200 dark:border-yellow-800 text-yellow-700 dark:text-yellow-300 text-xs">
                  <AlertCircle className="w-4 h-4 flex-shrink-0" />{t("oauthWizard.confirm.secretWarning")}
                </div>

                <div>
                  <div className="flex items-center justify-between mb-1">
                    <label className="text-xs font-medium text-gray-500">{t("oauthWizard.confirm.clientId")}</label>
                  </div>
                  <code className="block p-3 rounded-lg bg-gray-900 dark:bg-gray-800 text-sm font-mono text-green-400 break-all">{created.client_id}</code>
                </div>
                {created.client_secret && (
                  <div>
                    <label className="text-xs font-medium text-gray-500 mb-1 block">{t("oauthWizard.confirm.clientSecret")}</label>
                    <code className="block p-3 rounded-lg bg-gray-900 dark:bg-gray-800 text-sm font-mono text-orange-400 break-all">{created.client_secret}</code>
                  </div>
                )}

                <button onClick={copyAll} className="flex items-center gap-1.5 px-3 py-1.5 bg-gray-100 dark:bg-gray-700 dark:bg-gray-800 text-gray-600 dark:text-gray-400 dark:text-gray-400 rounded-lg text-sm hover:bg-gray-200">
                  {copied ? <Check className="w-4 h-4 text-green-500" /> : <Copy className="w-4 h-4" />}{t("oauthWizard.confirm.copyAll")}
                </button>

                {/* curl example */}
                <div>
                  <label className="text-xs font-medium text-gray-500 mb-1 block">{t("oauthWizard.confirm.curlExample")}</label>
                  <pre className="p-3 rounded-lg bg-gray-900 dark:bg-gray-800 text-xs font-mono text-gray-300 overflow-x-auto">{`curl -X POST ${API_BASE}/api/v1/oauth/token \\
  -H "Content-Type: application/x-www-form-urlencoded" \\
  -d "grant_type=client_credentials" \\
  -d "client_id=${created.client_id}" \\
  -d "client_secret=${created.client_secret}"`}</pre>
                </div>
              </div>

              <a href="/oauth-clients" className="inline-flex items-center gap-2 px-6 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-medium">
                {t("oauthWizard.confirm.close")}<ArrowRight className="w-4 h-4" />
              </a>
            </div>
          )}

          {/* Navigation (hidden on step 3/4) */}
          {step < 3 && (
            <div className="flex items-center justify-between mt-6 pt-4 border-t border-gray-200 dark:border-gray-700 dark:border-gray-800">
              {step > 0 ? (
                <button onClick={() => setStep((step - 1) as Step)} className="flex items-center gap-1.5 px-4 py-2 bg-gray-100 dark:bg-gray-700 dark:bg-gray-800 text-gray-600 dark:text-gray-400 dark:text-gray-400 rounded-lg text-sm font-medium">
                  <ChevronLeft className="w-4 h-4" />{t("oauthWizard.nav.back")}
                </button>
              ) : <div />}
              <button onClick={() => setStep((step + 1) as Step)} disabled={!canNext()}
                className="flex items-center gap-1.5 px-6 py-2 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg text-sm font-medium">
                {t("oauthWizard.nav.next")}<ChevronRight className="w-4 h-4" />
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

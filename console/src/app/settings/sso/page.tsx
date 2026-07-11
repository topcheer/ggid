"use client";

import { useEffect, useState, useRef } from "react";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";
import {
  Save,
  ShieldCheck,
  Globe,
  Key,
  Lock,
  CheckCircle2,
  XCircle,
  Building2,
  Loader2,
  Upload,
  ChevronRight,
  Trash2,
  Pencil,
  Plus,
  ArrowLeft,
  FileText,
  Settings2,
  Award,
} from "lucide-react";

interface SocialProvider {
  id: string;
  name: string;
  enabled: boolean;
}

interface SAMLProvider {
  id: string;
  name: string;
  type: "SAML";
  entityId: string;
  ssoUrl: string;
  sloUrl: string;
  x509Cert: string;
  metadataUrl: string;
  metadataXml: string;
  attributeMappings: Record<string, string>;
  active: boolean;
}

interface OIDCProvider {
  id: string;
  name: string;
  type: "OIDC";
  discoveryUrl: string;
  clientId: string;
  clientSecret: string;
  scopes: string[];
  customScope: string;
  tokenEndpoint: string;
  active: boolean;
}

type Provider = SAMLProvider | OIDCProvider;

interface TestResult {
  status: "success" | "fail";
  responseTime: number;
  providerInfo: string;
  details: string;
}

interface Toast {
  type: "success" | "error";
  message: string;
}

const GGID_FIELDS = ["email", "username", "firstName", "lastName", "displayName", "phone", "department", "groups"];
const SAML_ATTRS = ["email", "username", "firstName", "lastName", "displayName", "groups", "department", "title", "uid"];

export default function SSOConnectionsPage() {
  const { apiFetch } = useApi();
  const t = useTranslations();
  const [toast, setToast] = useState<Toast | null>(null);
  const [testingId, setTestingId] = useState<string | null>(null);
  const [testResults, setTestResults] = useState<Record<string, TestResult>>({});

  // Wizard state
  const [showWizard, setShowWizard] = useState(false);
  const [wizardType, setWizardType] = useState<"SAML" | "OIDC">("SAML");
  const [wizardStep, setWizardStep] = useState(1);

  // SAML wizard form
  const [samlForm, setSamlForm] = useState({
    name: "",
    entityId: "",
    ssoUrl: "",
    sloUrl: "",
    x509Cert: "",
    metadataUrl: "",
    metadataXml: "",
    attributeMappings: {
      email: "email",
      username: "uid",
      firstName: "firstName",
      lastName: "lastName",
    } as Record<string, string>,
  });

  // OIDC form
  const [oidcForm, setOidcForm] = useState({
    name: "",
    discoveryUrl: "",
    clientId: "",
    clientSecret: "",
    scopes: ["openid", "profile", "email"] as string[],
    customScope: "",
    tokenEndpoint: "",
  });

  const [providers, setProviders] = useState<Provider[]>([]);
  const [editingId, setEditingId] = useState<string | null>(null);

  const samlCertRef = useRef<HTMLInputElement>(null);
  const samlMetadataRef = useRef<HTMLInputElement>(null);

  const [socialProviders, setSocialProviders] = useState<SocialProvider[]>([
    { id: "google", name: "Google", enabled: false },
    { id: "github", name: "GitHub", enabled: false },
    { id: "microsoft", name: "Microsoft", enabled: false },
    { id: "apple", name: "Apple", enabled: false },
    { id: "slack", name: "Slack", enabled: false },
  ]);

  useEffect(() => {
    if (toast) {
      const t = setTimeout(() => setToast(null), 3000);
      return () => clearTimeout(t);
    }
  }, [toast]);

  useEffect(() => {
    const loadConfig = async () => {
      try {
        const data = await apiFetch<Record<string, unknown>>("/api/v1/settings/sso");
        if (data.saml && typeof data.saml === "object") {
          const s = data.saml as Record<string, unknown>;
          setSamlForm((prev) => ({
            ...prev,
            name: (s.idp_name as string) || prev.name,
            entityId: (s.entity_id as string) || prev.entityId,
            ssoUrl: (s.sso_url as string) || prev.ssoUrl,
            x509Cert: (s.x509_cert as string) || prev.x509Cert,
          }));
        }
        if (data.oidc && typeof data.oidc === "object") {
          const o = data.oidc as Record<string, unknown>;
          setOidcForm((prev) => ({
            ...prev,
            discoveryUrl: (o.discovery_url as string) || prev.discoveryUrl,
            clientId: (o.client_id as string) || prev.clientId,
            clientSecret: (o.client_secret as string) || prev.clientSecret,
          }));
        }
        if (data.social && Array.isArray(data.social)) {
          setSocialProviders((prev) =>
            prev.map((p) => {
              const remote = (data.social as SocialProvider[]).find((r) => r.id === p.id);
              return remote ? { ...p, enabled: remote.enabled } : p;
            }),
          );
        }
      } catch {
        // No existing config — keep defaults
      }
    };
    loadConfig();
  }, [apiFetch]);

  const inputCls =
    "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";
  const labelCls = "mb-1 block text-xs font-medium text-gray-500";
  const cardCls =
    "rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const headingCls = "mb-4 text-lg font-semibold text-gray-900 dark:text-gray-100";

  const showToast = (type: "success" | "error", message: string) => setToast({ type, message });

  // --- File handlers ---
  const handleCertUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => {
      setSamlForm((prev) => ({ ...prev, x509Cert: reader.result as string }));
    };
    reader.readAsText(file);
  };

  const handleMetadataUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => {
      setSamlForm((prev) => ({ ...prev, metadataXml: reader.result as string }));
    };
    reader.readAsText(file);
  };

  // --- Wizard save ---
  const saveSamlProvider = async () => {
    if (!samlForm.entityId || !samlForm.ssoUrl) {
      showToast("error", "Entity ID and SSO URL are required");
      return;
    }
    const provider: SAMLProvider = {
      id: editingId || `saml-${Date.now()}`,
      name: samlForm.name || "SAML Provider",
      type: "SAML",
      entityId: samlForm.entityId,
      ssoUrl: samlForm.ssoUrl,
      sloUrl: samlForm.sloUrl,
      x509Cert: samlForm.x509Cert,
      metadataUrl: samlForm.metadataUrl,
      metadataXml: samlForm.metadataXml,
      attributeMappings: samlForm.attributeMappings,
      active: true,
    };
    try {
      await apiFetch("/api/v1/settings/sso/saml", {
        method: "POST",
        body: JSON.stringify(provider),
      });
      setProviders((prev) => {
        const idx = prev.findIndex((p) => p.id === provider.id);
        if (idx >= 0) {
          const copy = [...prev];
          copy[idx] = provider;
          return copy;
        }
        return [...prev, provider];
      });
      showToast("success", "SAML provider saved and activated");
      resetWizard();
    } catch {
      setProviders((prev) => {
        const idx = prev.findIndex((p) => p.id === provider.id);
        if (idx >= 0) {
          const copy = [...prev];
          copy[idx] = provider;
          return copy;
        }
        return [...prev, provider];
      });
      showToast("success", "SAML provider saved locally");
      resetWizard();
    }
  };

  const saveOidcProvider = async () => {
    if (!oidcForm.clientId || !oidcForm.discoveryUrl) {
      showToast("error", "Client ID and Discovery URL are required");
      return;
    }
    const allScopes = oidcForm.customScope
      ? [...oidcForm.scopes, oidcForm.customScope]
      : oidcForm.scopes;
    const provider: OIDCProvider = {
      id: editingId || `oidc-${Date.now()}`,
      name: oidcForm.name || "OIDC Provider",
      type: "OIDC",
      discoveryUrl: oidcForm.discoveryUrl,
      clientId: oidcForm.clientId,
      clientSecret: oidcForm.clientSecret,
      scopes: allScopes,
      customScope: oidcForm.customScope,
      tokenEndpoint: oidcForm.tokenEndpoint,
      active: true,
    };
    try {
      await apiFetch("/api/v1/settings/sso/oidc", {
        method: "POST",
        body: JSON.stringify(provider),
      });
      setProviders((prev) => {
        const idx = prev.findIndex((p) => p.id === provider.id);
        if (idx >= 0) {
          const copy = [...prev];
          copy[idx] = provider;
          return copy;
        }
        return [...prev, provider];
      });
      showToast("success", "OIDC provider saved and activated");
      resetWizard();
    } catch {
      setProviders((prev) => {
        const idx = prev.findIndex((p) => p.id === provider.id);
        if (idx >= 0) {
          const copy = [...prev];
          copy[idx] = provider;
          return copy;
        }
        return [...prev, provider];
      });
      showToast("success", "OIDC provider saved locally");
      resetWizard();
    }
  };

  const resetWizard = () => {
    setShowWizard(false);
    setWizardStep(1);
    setEditingId(null);
    setSamlForm({
      name: "",
      entityId: "",
      ssoUrl: "",
      sloUrl: "",
      x509Cert: "",
      metadataUrl: "",
      metadataXml: "",
      attributeMappings: { email: "email", username: "uid", firstName: "firstName", lastName: "lastName" },
    });
    setOidcForm({
      name: "",
      discoveryUrl: "",
      clientId: "",
      clientSecret: "",
      scopes: ["openid", "profile", "email"],
      customScope: "",
      tokenEndpoint: "",
    });
  };

  // --- Test connection ---
  const testConnection = async (provider: Provider) => {
    setTestingId(provider.id);
    const startTime = Date.now();
    try {
      await apiFetch(`/api/v1/settings/sso/${provider.type.toLowerCase()}/test`, {
        method: "POST",
        body: JSON.stringify(provider),
      });
      const elapsed = Date.now() - startTime;
      const result: TestResult = {
        status: "success",
        responseTime: elapsed,
        providerInfo: provider.type === "SAML" ? `${(provider as SAMLProvider).entityId}` : `${(provider as OIDCProvider).discoveryUrl}`,
        details: `Connection verified. IdP responded in ${elapsed}ms.`,
      };
      setTestResults((prev) => ({ ...prev, [provider.id]: result }));
      showToast("success", `${provider.name}: connection test succeeded`);
    } catch {
      const elapsed = Date.now() - startTime;
      const result: TestResult = {
        status: "fail",
        responseTime: elapsed,
        providerInfo: provider.name,
        details: `Connection failed after ${elapsed}ms. Check credentials and network.`,
      };
      setTestResults((prev) => ({ ...prev, [provider.id]: result }));
      showToast("error", `${provider.name}: connection test failed`);
    } finally {
      setTestingId(null);
    }
  };

  // --- Provider actions ---
  const editProvider = (p: Provider) => {
    setEditingId(p.id);
    if (p.type === "SAML") {
      const s = p as SAMLProvider;
      setSamlForm({
        name: s.name,
        entityId: s.entityId,
        ssoUrl: s.ssoUrl,
        sloUrl: s.sloUrl,
        x509Cert: s.x509Cert,
        metadataUrl: s.metadataUrl,
        metadataXml: s.metadataXml,
        attributeMappings: s.attributeMappings,
      });
      setWizardType("SAML");
    } else {
      const o = p as OIDCProvider;
      setOidcForm({
        name: o.name,
        discoveryUrl: o.discoveryUrl,
        clientId: o.clientId,
        clientSecret: o.clientSecret,
        scopes: o.scopes.filter((s) => ["openid", "profile", "email"].includes(s)),
        customScope: o.scopes.filter((s) => !["openid", "profile", "email"].includes(s)).join(" "),
        tokenEndpoint: o.tokenEndpoint,
      });
      setWizardType("OIDC");
    }
    setShowWizard(true);
    setWizardStep(1);
  };

  const deleteProvider = (id: string) => {
    setProviders((prev) => prev.filter((p) => p.id !== id));
    showToast("success", "Provider deleted");
  };

  const toggleProviderActive = (id: string) => {
    setProviders((prev) =>
      prev.map((p) => (p.id === id ? { ...p, active: !p.active } : p)),
    );
  };

  const toggleSocial = async (id: string) => {
    const updated = socialProviders.map((p) =>
      p.id === id ? { ...p, enabled: !p.enabled } : p,
    );
    setSocialProviders(updated);
    try {
      await apiFetch("/api/v1/settings/sso/social", {
        method: "PUT",
        body: JSON.stringify({ providers: updated }),
      });
    } catch {
      setSocialProviders(socialProviders);
    }
  };

  // --- Wizard next/prev ---
  const canProceedStep1 = wizardType === "SAML"
    ? !!(samlForm.entityId && samlForm.ssoUrl && (samlForm.x509Cert || samlForm.metadataXml || samlForm.metadataUrl))
    : !!(oidcForm.discoveryUrl && oidcForm.clientId);

  const steps = wizardType === "SAML" ? 3 : 1;

  return (
    <div className="max-w-5xl">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">{t("sso.connections")}</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {t("sso.subtitle")}
          </p>
        </div>
        {!showWizard && (
          <button
            onClick={() => { setShowWizard(true); setWizardType("SAML"); setWizardStep(1); }}
            className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
          >
            <Plus className="h-4 w-4" /> {t("sso.addProvider")}
          </button>
        )}
      </div>

      {/* Toast */}
      {toast && (
        <div
          className={`mb-4 flex items-center gap-2 rounded-lg border p-3 text-sm ${
            toast.type === "success"
              ? "border-green-200 bg-green-50 text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400"
              : "border-red-200 bg-red-50 text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400"
          }`}
        >
          {toast.type === "success" ? (
            <CheckCircle2 className="h-4 w-4 shrink-0" />
          ) : (
            <XCircle className="h-4 w-4 shrink-0" />
          )}
          {toast.message}
        </div>
      )}

      {/* ===== Wizard ===== */}
      {showWizard && (
        <div className="mb-6">
          <div className={cardCls}>
            {/* Wizard header */}
            <div className="mb-6 flex items-center justify-between">
              <h2 className={headingCls}>
                {editingId ? t("sso.editProvider") : t("sso.addNewProvider")}
              </h2>
              <button
                onClick={resetWizard}
                className="flex items-center gap-1 text-sm text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
              >
                <XCircle className="h-4 w-4" /> {t("common.cancel")}
              </button>
            </div>

            {/* Type selector (only when adding new) */}
            {!editingId && (
              <div className="mb-6 flex gap-3">
                <button
                  onClick={() => { setWizardType("SAML"); setWizardStep(1); }}
                  className={`flex-1 rounded-lg border-2 p-4 text-left transition-colors ${
                    wizardType === "SAML"
                      ? "border-brand-600 bg-brand-50 dark:bg-brand-950/30"
                      : "border-gray-200 hover:border-gray-300 dark:border-gray-700"
                  }`}
                >
                  <Building2 className="mb-2 h-5 w-5 text-brand-600" />
                  <p className="text-sm font-semibold">{t("sso.samlIdp")}</p>
                  <p className="text-xs text-gray-500">Okta, ADFS, Azure AD, OneLogin</p>
                </button>
                <button
                  onClick={() => { setWizardType("OIDC"); setWizardStep(1); }}
                  className={`flex-1 rounded-lg border-2 p-4 text-left transition-colors ${
                    wizardType === "OIDC"
                      ? "border-brand-600 bg-brand-50 dark:bg-brand-950/30"
                      : "border-gray-200 hover:border-gray-300 dark:border-gray-700"
                  }`}
                >
                  <Key className="mb-2 h-5 w-5 text-brand-600" />
                  <p className="text-sm font-semibold">{t("sso.oidcProvider")}</p>
                  <p className="text-xs text-gray-500">Google, Microsoft, Keycloak</p>
                </button>
              </div>
            )}

            {/* Step indicator */}
            {wizardType === "SAML" && (
              <div className="mb-6 flex items-center gap-2">
                {[1, 2, 3].map((step, i) => (
                  <div key={step} className="flex items-center gap-2">
                    <div
                      className={`flex h-8 w-8 items-center justify-center rounded-full text-xs font-bold ${
                        wizardStep >= step
                          ? "bg-brand-600 text-white"
                          : "bg-gray-200 text-gray-400 dark:bg-gray-700"
                      }`}
                    >
                      {wizardStep > step ? <CheckCircle2 className="h-4 w-4" /> : step}
                    </div>
                    {i < 2 && (
                      <div className={`h-0.5 w-12 ${wizardStep > step ? "bg-brand-600" : "bg-gray-200 dark:bg-gray-700"}`} />
                    )}
                  </div>
                ))}
                <div className="ml-3 flex gap-3 text-xs">
                  <span className={wizardStep === 1 ? "font-semibold text-gray-700 dark:text-gray-300" : "text-gray-400"}>Metadata</span>
                  <span className={wizardStep === 2 ? "font-semibold text-gray-700 dark:text-gray-300" : "text-gray-400"}>Attributes</span>
                  <span className={wizardStep === 3 ? "font-semibold text-gray-700 dark:text-gray-300" : "text-gray-400"}>Certificate</span>
                </div>
              </div>
            )}

            {/* ===== SAML Wizard Steps ===== */}
            {wizardType === "SAML" && wizardStep === 1 && (
              <div className="space-y-4">
                <div className="mb-2 flex items-center gap-2 text-sm font-semibold text-gray-700 dark:text-gray-300">
                  <FileText className="h-4 w-4 text-brand-600" /> Step 1: IdP Metadata
                </div>
                <div className="grid gap-4 sm:grid-cols-2">
                  <div>
                    <label className={labelCls}>Provider Name</label>
                    <input
                      value={samlForm.name}
                      onChange={(e) => setSamlForm({ ...samlForm, name: e.target.value })}
                      placeholder="e.g. Okta Production"
                      className={inputCls}
                    />
                  </div>
                  <div>
                    <label className={labelCls}>Entity ID</label>
                    <input
                      value={samlForm.entityId}
                      onChange={(e) => setSamlForm({ ...samlForm, entityId: e.target.value })}
                      placeholder="https://idp.example.com/entity"
                      className={`${inputCls} font-mono`}
                    />
                  </div>
                  <div className="sm:col-span-2">
                    <label className={labelCls}>SSO URL</label>
                    <input
                      value={samlForm.ssoUrl}
                      onChange={(e) => setSamlForm({ ...samlForm, ssoUrl: e.target.value })}
                      placeholder="https://idp.example.com/sso"
                      className={`${inputCls} font-mono`}
                    />
                  </div>
                </div>
                {/* Metadata upload or URL */}
                <div className="rounded-lg border border-gray-200 p-4 dark:border-gray-700">
                  <p className="mb-3 text-xs font-medium text-gray-500">Import Metadata (optional)</p>
                  <div className="grid gap-3 sm:grid-cols-2">
                    <div>
                      <button
                        onClick={() => samlMetadataRef.current?.click()}
                        className="flex w-full items-center justify-center gap-2 rounded-lg border-2 border-dashed border-gray-300 px-4 py-6 text-sm text-gray-500 hover:border-brand-400 hover:text-brand-600 dark:border-gray-600"
                      >
                        <Upload className="h-5 w-5" />
                        {samlForm.metadataXml ? "XML Loaded" : "Upload metadata XML"}
                      </button>
                      <input ref={samlMetadataRef} type="file" accept=".xml" className="hidden" onChange={handleMetadataUpload} />
                    </div>
                    <div>
                      <label className={labelCls}>Or paste metadata URL</label>
                      <input
                        value={samlForm.metadataUrl}
                        onChange={(e) => setSamlForm({ ...samlForm, metadataUrl: e.target.value })}
                        placeholder="https://idp.example.com/metadata.xml"
                        className={`${inputCls} font-mono`}
                      />
                    </div>
                  </div>
                  {samlForm.metadataXml && (
                    <p className="mt-2 text-xs text-green-600 dark:text-green-400">
                      <CheckCircle2 className="mr-1 inline h-3 w-3" />
                      {samlForm.metadataXml.length} chars of XML loaded
                    </p>
                  )}
                </div>
                <div className="flex justify-end">
                  <button
                    onClick={() => setWizardStep(2)}
                    disabled={!canProceedStep1}
                    className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
                  >
                    Next: Attribute Mapping <ChevronRight className="h-4 w-4" />
                  </button>
                </div>
              </div>
            )}

            {/* Step 2: Attribute Mapping */}
            {wizardType === "SAML" && wizardStep === 2 && (
              <div className="space-y-4">
                <div className="mb-2 flex items-center gap-2 text-sm font-semibold text-gray-700 dark:text-gray-300">
                  <Settings2 className="h-4 w-4 text-brand-600" /> Step 2: Attribute Mapping
                </div>
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  Map SAML attributes from your IdP to GGID user fields.
                </p>
                <div className="space-y-3">
                  {GGID_FIELDS.map((field) => (
                    <div key={field} className="grid grid-cols-[1fr_auto_1fr] items-center gap-3">
                      <div className="rounded-lg bg-gray-50 px-3 py-2 text-sm font-medium text-gray-700 dark:bg-gray-700 dark:text-gray-300">
                        {field}
                      </div>
                      <ChevronRight className="h-4 w-4 text-gray-400" />
                      <select
                        value={samlForm.attributeMappings[field] || ""}
                        onChange={(e) =>
                          setSamlForm({
                            ...samlForm,
                            attributeMappings: { ...samlForm.attributeMappings, [field]: e.target.value },
                          })
                        }
                        className={inputCls}
                      >
                        <option value="">-- Not mapped --</option>
                        {SAML_ATTRS.map((attr) => (
                          <option key={attr} value={attr}>{attr}</option>
                        ))}
                      </select>
                    </div>
                  ))}
                </div>
                <div className="flex justify-between">
                  <button
                    onClick={() => setWizardStep(1)}
                    className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700"
                  >
                    <ArrowLeft className="h-4 w-4" /> Back
                  </button>
                  <button
                    onClick={() => setWizardStep(3)}
                    className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
                  >
                    Next: Certificate <ChevronRight className="h-4 w-4" />
                  </button>
                </div>
              </div>
            )}

            {/* Step 3: Signing Certificate */}
            {wizardType === "SAML" && wizardStep === 3 && (
              <div className="space-y-4">
                <div className="mb-2 flex items-center gap-2 text-sm font-semibold text-gray-700 dark:text-gray-300">
                  <Award className="h-4 w-4 text-brand-600" /> Step 3: Signing Certificate
                </div>
                <div>
                  <label className={labelCls}>Upload Certificate (.pem / .crt)</label>
                  <button
                    onClick={() => samlCertRef.current?.click()}
                    className="flex w-full items-center justify-center gap-2 rounded-lg border-2 border-dashed border-gray-300 px-4 py-8 text-sm text-gray-500 hover:border-brand-400 hover:text-brand-600 dark:border-gray-600"
                  >
                    <Upload className="h-6 w-6" />
                    {samlForm.x509Cert ? "Certificate Loaded" : "Click to upload .pem or .crt file"}
                  </button>
                  <input ref={samlCertRef} type="file" accept=".pem,.crt,.cer" className="hidden" onChange={handleCertUpload} />
                </div>
                <div>
                  <label className={labelCls}>Or paste certificate (PEM)</label>
                  <textarea
                    value={samlForm.x509Cert}
                    onChange={(e) => setSamlForm({ ...samlForm, x509Cert: e.target.value })}
                    placeholder={"-----BEGIN CERTIFICATE-----\nMIID...\n-----END CERTIFICATE-----"}
                    rows={6}
                    className={`${inputCls} font-mono text-xs`}
                  />
                </div>
                <div className="flex justify-between">
                  <button
                    onClick={() => setWizardStep(2)}
                    className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700"
                  >
                    <ArrowLeft className="h-4 w-4" /> {t("sso.back")}
                  </button>
                  <button
                    onClick={saveSamlProvider}
                    className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
                  >
                    <Save className="h-4 w-4" /> {t("sso.saveActivate")}
                  </button>
                </div>
              </div>
            )}

            {/* ===== OIDC Form (single step) ===== */}
            {wizardType === "OIDC" && (
              <div className="space-y-4">
                <div className="mb-2 flex items-center gap-2 text-sm font-semibold text-gray-700 dark:text-gray-300">
                  <Key className="h-4 w-4 text-brand-600" /> OIDC Provider Configuration
                </div>
                <div className="grid gap-4 sm:grid-cols-2">
                  <div>
                    <label className={labelCls}>Provider Name</label>
                    <input
                      value={oidcForm.name}
                      onChange={(e) => setOidcForm({ ...oidcForm, name: e.target.value })}
                      placeholder="e.g. Google Workspace"
                      className={inputCls}
                    />
                  </div>
                  <div>
                    <label className={labelCls}>Discovery URL</label>
                    <input
                      value={oidcForm.discoveryUrl}
                      onChange={(e) => setOidcForm({ ...oidcForm, discoveryUrl: e.target.value })}
                      placeholder="https://accounts.google.com/.well-known/openid-configuration"
                      className={`${inputCls} font-mono`}
                    />
                  </div>
                  <div>
                    <label className={labelCls}>Client ID</label>
                    <input
                      value={oidcForm.clientId}
                      onChange={(e) => setOidcForm({ ...oidcForm, clientId: e.target.value })}
                      placeholder="your-client-id"
                      className={`${inputCls} font-mono`}
                    />
                  </div>
                  <div>
                    <label className={labelCls}>Client Secret</label>
                    <input
                      type="password"
                      value={oidcForm.clientSecret}
                      onChange={(e) => setOidcForm({ ...oidcForm, clientSecret: e.target.value })}
                      placeholder="your-client-secret"
                      className={`${inputCls} font-mono`}
                    />
                  </div>
                </div>
                {/* Scope mapping */}
                <div>
                  <label className={labelCls}>Scopes</label>
                  <div className="flex flex-wrap gap-2">
                    {["openid", "profile", "email"].map((scope) => (
                      <button
                        key={scope}
                        onClick={() =>
                          setOidcForm((prev) => ({
                            ...prev,
                            scopes: prev.scopes.includes(scope)
                              ? prev.scopes.filter((s) => s !== scope)
                              : [...prev.scopes, scope],
                          }))
                        }
                        className={`rounded-lg px-3 py-1.5 text-xs font-medium transition-colors ${
                          oidcForm.scopes.includes(scope)
                            ? "bg-brand-600 text-white"
                            : "border border-gray-300 text-gray-600 dark:border-gray-600 dark:text-gray-300"
                        }`}
                      >
                        {scope}
                      </button>
                    ))}
                  </div>
                  <input
                    value={oidcForm.customScope}
                    onChange={(e) => setOidcForm({ ...oidcForm, customScope: e.target.value })}
                    placeholder="Custom scope (e.g. groups)"
                    className={`${inputCls} mt-2 font-mono`}
                  />
                </div>
                <div>
                  <label className={labelCls}>Token Endpoint Override (optional)</label>
                  <input
                    value={oidcForm.tokenEndpoint}
                    onChange={(e) => setOidcForm({ ...oidcForm, tokenEndpoint: e.target.value })}
                    placeholder="https://provider.example.com/oauth/token"
                    className={`${inputCls} font-mono`}
                  />
                </div>
                <div className="flex justify-end">
                  <button
                    onClick={saveOidcProvider}
                    className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
                  >
                    <Save className="h-4 w-4" /> {t("sso.saveActivate")}
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* ===== Provider List ===== */}
      {providers.length > 0 && (
        <div className="mb-6">
          <h2 className="mb-3 text-sm font-semibold uppercase text-gray-500">{t("sso.configuredProviders")}</h2>
          <div className="grid gap-3 sm:grid-cols-2">
            {providers.map((provider) => {
              const result = testResults[provider.id];
              return (
                <div
                  key={provider.id}
                  className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-800"
                >
                  <div className="mb-3 flex items-start justify-between">
                    <div className="flex items-center gap-3">
                      <div className={`flex h-10 w-10 items-center justify-center rounded-lg ${provider.type === "SAML" ? "bg-indigo-100 text-indigo-600" : "bg-purple-100 text-purple-600"}`}>
                        {provider.type === "SAML" ? <Building2 className="h-5 w-5" /> : <Key className="h-5 w-5" />}
                      </div>
                      <div>
                        <p className="text-sm font-semibold text-gray-900 dark:text-gray-100">{provider.name}</p>
                        <p className="text-xs text-gray-500">{provider.type}</p>
                      </div>
                    </div>
                    <span
                      className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                        provider.active
                          ? "bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-400"
                          : "bg-gray-100 text-gray-500"
                      }`}
                    >
                      {provider.active ? "Active" : "Inactive"}
                    </span>
                  </div>

                  {/* Provider details */}
                  <div className="mb-3 space-y-1 text-xs text-gray-500 dark:text-gray-400">
                    {provider.type === "SAML" ? (
                      <>
                        <p>Entity: <span className="font-mono">{(provider as SAMLProvider).entityId}</span></p>
                        <p>SSO URL: <span className="font-mono">{(provider as SAMLProvider).ssoUrl.slice(0, 40)}...</span></p>
                      </>
                    ) : (
                      <>
                        <p>Discovery: <span className="font-mono">{(provider as OIDCProvider).discoveryUrl.slice(0, 40)}...</span></p>
                        <p>Scopes: <span className="font-mono">{(provider as OIDCProvider).scopes.join(" ")}</span></p>
                      </>
                    )}
                  </div>

                  {/* Test result */}
                  {result && (
                    <div
                      className={`mb-3 rounded-lg border p-2 text-xs ${
                        result.status === "success"
                          ? "border-green-200 bg-green-50 dark:border-green-800 dark:bg-green-950"
                          : "border-red-200 bg-red-50 dark:border-red-800 dark:bg-red-950"
                      }`}
                    >
                      <div className="flex items-center gap-2">
                        {result.status === "success" ? (
                          <CheckCircle2 className="h-4 w-4 text-green-600" />
                        ) : (
                          <XCircle className="h-4 w-4 text-red-600" />
                        )}
                        <span className={`font-semibold ${result.status === "success" ? "text-green-700 dark:text-green-400" : "text-red-700 dark:text-red-400"}`}>
                          {result.status === "success" ? "Connected" : "Failed"}
                        </span>
                        <span className="text-gray-400">{result.responseTime}ms</span>
                      </div>
                      <p className="mt-1 text-gray-500">{result.details}</p>
                    </div>
                  )}

                  {/* Actions */}
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => testConnection(provider)}
                      disabled={testingId === provider.id}
                      className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-1.5 text-xs font-medium hover:bg-gray-50 disabled:opacity-50 dark:border-gray-600 dark:hover:bg-gray-700"
                    >
                      {testingId === provider.id ? (
                        <Loader2 className="h-3.5 w-3.5 animate-spin" />
                      ) : (
                        <ShieldCheck className="h-3.5 w-3.5" />
                      )}
                      Test
                    </button>
                    <button
                      onClick={() => editProvider(provider)}
                      className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-1.5 text-xs font-medium hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700"
                    >
                      <Pencil className="h-3.5 w-3.5" /> Edit
                    </button>
                    <button
                      onClick={() => toggleProviderActive(provider.id)}
                      className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-1.5 text-xs font-medium hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700"
                    >
                      {provider.active ? "Deactivate" : "Activate"}
                    </button>
                    <button
                      onClick={() => deleteProvider(provider.id)}
                      className="ml-auto rounded-lg p-1.5 text-red-500 hover:bg-red-50 dark:hover:bg-red-950"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* ===== Social Login Toggles ===== */}
      {!showWizard && (
        <div className={cardCls}>
          <h2 className={`flex items-center gap-2 ${headingCls}`}>
            <Globe className="h-5 w-5 text-brand-600" /> Social Login Providers
          </h2>
          <p className="mb-4 text-sm text-gray-500 dark:text-gray-400">
            Enable or disable social login options for your users.
          </p>
          <div className="space-y-3">
            {socialProviders.map((provider) => (
              <div
                key={provider.id}
                className="flex items-center justify-between rounded-lg border border-gray-200 p-4 dark:border-gray-700"
              >
                <div className="flex items-center gap-3">
                  <span className={`inline-flex h-2.5 w-2.5 rounded-full ${provider.enabled ? "bg-green-500" : "bg-gray-300"}`} />
                  <div>
                    <p className="text-sm font-medium text-gray-900 dark:text-gray-100">{provider.name}</p>
                    <p className="text-xs text-gray-500 dark:text-gray-400">{provider.enabled ? "Enabled" : "Disabled"}</p>
                  </div>
                </div>
                <button
                  role="switch"
                  aria-checked={provider.enabled}
                  onClick={() => toggleSocial(provider.id)}
                  className={`relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none ${
                    provider.enabled ? "bg-brand-600" : "bg-gray-200 dark:bg-gray-600"
                  }`}
                >
                  <span
                    className={`pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${
                      provider.enabled ? "translate-x-5" : "translate-x-0"
                    }`}
                  />
                </button>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

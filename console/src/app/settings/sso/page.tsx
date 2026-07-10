"use client";

import { useEffect, useState } from "react";
import { useApi } from "@/lib/api";
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
} from "lucide-react";

interface SocialProvider {
  id: string;
  name: string;
  enabled: boolean;
}

interface Toast {
  type: "success" | "error";
  message: string;
}

export default function SSOConnectionsPage() {
  const { apiFetch } = useApi();
  const [toast, setToast] = useState<Toast | null>(null);
  const [testing, setTesting] = useState<"saml" | "oidc" | null>(null);

  const [samlConfig, setSamlConfig] = useState({
    idp_name: "",
    entity_id: "",
    sso_url: "",
    x509_cert: "",
    active: false,
  });

  const [oidcConfig, setOidcConfig] = useState({
    provider: "google",
    client_id: "",
    client_secret: "",
    discovery_url: "",
    scopes: "openid profile email",
    active: false,
  });

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

  // Attempt to load existing config from API; fall back gracefully
  useEffect(() => {
    const loadConfig = async () => {
      try {
        const data = await apiFetch<Record<string, unknown>>("/api/v1/settings/sso");
        if (data.saml && typeof data.saml === "object") {
          const s = data.saml as Record<string, unknown>;
          setSamlConfig({
            idp_name: (s.idp_name as string) || "",
            entity_id: (s.entity_id as string) || "",
            sso_url: (s.sso_url as string) || "",
            x509_cert: (s.x509_cert as string) || "",
            active: !!s.active,
          });
        }
        if (data.oidc && typeof data.oidc === "object") {
          const o = data.oidc as Record<string, unknown>;
          setOidcConfig((prev) => ({
            ...prev,
            provider: (o.provider as string) || prev.provider,
            client_id: (o.client_id as string) || "",
            client_secret: (o.client_secret as string) || "",
            discovery_url: (o.discovery_url as string) || "",
            scopes: (o.scopes as string) || prev.scopes,
            active: !!o.active,
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

  const saveSaml = async () => {
    if (!samlConfig.entity_id || !samlConfig.sso_url) {
      showToast("error", "Entity ID and SSO URL are required");
      return;
    }
    try {
      await apiFetch("/api/v1/settings/sso/saml", {
        method: "POST",
        body: JSON.stringify(samlConfig),
      });
      showToast("success", "SAML configuration saved");
      setSamlConfig({ ...samlConfig, active: true });
    } catch {
      showToast("error", "Failed to save SAML config");
    }
  };

  const saveOidc = async () => {
    if (!oidcConfig.client_id || !oidcConfig.discovery_url) {
      showToast("error", "Client ID and Discovery URL are required");
      return;
    }
    try {
      await apiFetch("/api/v1/settings/sso/oidc", {
        method: "POST",
        body: JSON.stringify(oidcConfig),
      });
      showToast("success", "OIDC configuration saved");
      setOidcConfig({ ...oidcConfig, active: true });
    } catch {
      showToast("error", "Failed to save OIDC config");
    }
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
      // optimistic update — revert on error
      setSocialProviders(socialProviders);
    }
  };

  const testSaml = async () => {
    setTesting("saml");
    try {
      await apiFetch("/api/v1/settings/sso/saml/test", { method: "POST" });
      showToast("success", "SAML connection test succeeded");
    } catch {
      showToast("error", "SAML connection test failed — check IdP settings");
    } finally {
      setTesting(null);
    }
  };

  const testOidc = async () => {
    setTesting("oidc");
    try {
      await apiFetch("/api/v1/settings/sso/oidc/test", { method: "POST" });
      showToast("success", "OIDC flow test succeeded");
    } catch {
      showToast("error", "OIDC flow test failed — check client credentials");
    } finally {
      setTesting(null);
    }
  };

  const StatusDot = ({ active }: { active: boolean }) => (
    <span
      className={`inline-flex h-2.5 w-2.5 rounded-full ${active ? "bg-green-500" : "bg-gray-300"}`}
      title={active ? "Active" : "Inactive"}
    />
  );

  return (
    <div className="max-w-4xl">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">SSO Connections</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Configure SAML, OIDC, and social login providers for your organization.
        </p>
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

      {/* ===== SAML IdP Config ===== */}
      <div className="mb-6">
        <div className={cardCls}>
          <div className="mb-4 flex items-center justify-between">
            <h2 className={`flex items-center gap-2 ${headingCls}`}>
              <Building2 className="h-5 w-5 text-brand-600" /> SAML Identity Provider
              <span className="ml-2 flex items-center gap-1.5 text-xs font-normal text-gray-500">
                <StatusDot active={samlConfig.active} />
                {samlConfig.active ? "Connected" : "Not configured"}
              </span>
            </h2>
            <button
              onClick={saveSaml}
              className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-3 py-1.5 text-sm text-white hover:bg-brand-700"
            >
              <Save className="h-4 w-4" /> Save SAML Config
            </button>
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <div>
              <label className={labelCls}>IdP Name</label>
              <input
                value={samlConfig.idp_name}
                onChange={(e) => setSamlConfig({ ...samlConfig, idp_name: e.target.value })}
                placeholder="e.g. Okta Production"
                className={inputCls}
              />
            </div>
            <div>
              <label className={labelCls}>Entity ID</label>
              <input
                value={samlConfig.entity_id}
                onChange={(e) => setSamlConfig({ ...samlConfig, entity_id: e.target.value })}
                placeholder="https://idp.example.com/entity"
                className={`${inputCls} font-mono`}
              />
            </div>
            <div className="sm:col-span-2">
              <label className={labelCls}>SSO URL</label>
              <input
                value={samlConfig.sso_url}
                onChange={(e) => setSamlConfig({ ...samlConfig, sso_url: e.target.value })}
                placeholder="https://idp.example.com/sso"
                className={`${inputCls} font-mono`}
              />
            </div>
            <div className="sm:col-span-2">
              <label className={labelCls}>x509 Certificate (PEM)</label>
              <textarea
                value={samlConfig.x509_cert}
                onChange={(e) => setSamlConfig({ ...samlConfig, x509_cert: e.target.value })}
                placeholder={"-----BEGIN CERTIFICATE-----\nMIID...\n-----END CERTIFICATE-----"}
                rows={5}
                className={`${inputCls} font-mono text-xs`}
              />
            </div>
          </div>
          <div className="mt-4">
            <button
              onClick={testSaml}
              disabled={testing === "saml"}
              className="flex items-center gap-2 rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
            >
              {testing === "saml" ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <ShieldCheck className="h-4 w-4" />
              )}
              Test SAML Connection
            </button>
          </div>
        </div>
      </div>

      {/* ===== OIDC Provider Setup ===== */}
      <div className="mb-6">
        <div className={cardCls}>
          <div className="mb-4 flex items-center justify-between">
            <h2 className={`flex items-center gap-2 ${headingCls}`}>
              <Key className="h-5 w-5 text-brand-600" /> OIDC Provider
              <span className="ml-2 flex items-center gap-1.5 text-xs font-normal text-gray-500">
                <StatusDot active={oidcConfig.active} />
                {oidcConfig.active ? "Connected" : "Not configured"}
              </span>
            </h2>
            <button
              onClick={saveOidc}
              className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-3 py-1.5 text-sm text-white hover:bg-brand-700"
            >
              <Save className="h-4 w-4" /> Save OIDC Config
            </button>
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <div>
              <label className={labelCls}>Provider</label>
              <select
                value={oidcConfig.provider}
                onChange={(e) => setOidcConfig({ ...oidcConfig, provider: e.target.value })}
                className={inputCls}
              >
                <option value="google">Google</option>
                <option value="microsoft">Microsoft</option>
                <option value="github">GitHub</option>
                <option value="generic">Generic</option>
              </select>
            </div>
            <div>
              <label className={labelCls}>Client ID</label>
              <input
                value={oidcConfig.client_id}
                onChange={(e) => setOidcConfig({ ...oidcConfig, client_id: e.target.value })}
                placeholder="your-client-id"
                className={`${inputCls} font-mono`}
              />
            </div>
            <div>
              <label className={labelCls}>Client Secret</label>
              <input
                type="password"
                value={oidcConfig.client_secret}
                onChange={(e) => setOidcConfig({ ...oidcConfig, client_secret: e.target.value })}
                placeholder="your-client-secret"
                className={`${inputCls} font-mono`}
              />
            </div>
            <div>
              <label className={labelCls}>Scopes</label>
              <input
                value={oidcConfig.scopes}
                onChange={(e) => setOidcConfig({ ...oidcConfig, scopes: e.target.value })}
                className={`${inputCls} font-mono`}
              />
            </div>
            <div className="sm:col-span-2">
              <label className={labelCls}>Discovery URL</label>
              <input
                value={oidcConfig.discovery_url}
                onChange={(e) => setOidcConfig({ ...oidcConfig, discovery_url: e.target.value })}
                placeholder="https://accounts.google.com/.well-known/openid-configuration"
                className={`${inputCls} font-mono`}
              />
            </div>
          </div>
          <div className="mt-4">
            <button
              onClick={testOidc}
              disabled={testing === "oidc"}
              className="flex items-center gap-2 rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
            >
              {testing === "oidc" ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <ShieldCheck className="h-4 w-4" />
              )}
              Test OIDC Flow
            </button>
          </div>
        </div>
      </div>

      {/* ===== Social Login Toggles ===== */}
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
                <StatusDot active={provider.enabled} />
                <div>
                  <p className="text-sm font-medium text-gray-900 dark:text-gray-100">
                    {provider.name}
                  </p>
                  <p className="text-xs text-gray-500 dark:text-gray-400">
                    {provider.enabled ? "Enabled" : "Disabled"}
                  </p>
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
    </div>
  );
}

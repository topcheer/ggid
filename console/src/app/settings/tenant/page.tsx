"use client";

import { useEffect, useState, useRef } from "react";
import { useApi } from "@/lib/api";
import {
  Save, RotateCcw, Shield, Clock, Building2, Palette, Upload,
  Lock, Fingerprint, Smartphone, Mail, Server, Globe, Key, Check,
} from "lucide-react";

interface AuthMethod {
  key: string;
  label: string;
  icon: React.ElementType;
  description: string;
}

const AUTH_METHODS: AuthMethod[] = [
  { key: "password", label: "Password", icon: Lock, description: "Traditional username + password" },
  { key: "totp", label: "TOTP", icon: Smartphone, description: "Time-based one-time passwords" },
  { key: "webauthn", label: "WebAuthn", icon: Fingerprint, description: "Security keys & passkeys" },
  { key: "sms", label: "SMS", icon: Smartphone, description: "SMS verification codes" },
  { key: "email_link", label: "Email Link", icon: Mail, description: "Magic link via email" },
  { key: "ldap", label: "LDAP", icon: Server, description: "LDAP / Active Directory" },
  { key: "saml", label: "SAML", icon: Globe, description: "SAML 2.0 SSO" },
  { key: "oauth_oidc", label: "OAuth/OIDC", icon: Key, description: "OAuth 2.0 / OpenID Connect" },
];

export default function TenantSettingsPage() {
  const { apiFetch, TENANT_ID } = useApi();
  const [saving, setSaving] = useState(false);
  const [msg, setMsg] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const [config, setConfig] = useState({
    name: "Default Tenant",
    logoDataUrl: "",
    allowedAuthMethods: ["password", "totp"] as string[],
    sessionTimeout: 60,
    requireMfa: false,
    primaryColor: "#6366f1",
    accentColor: "#8b5cf6",
  });

  // Snapshot for reset
  const [snapshot, setSnapshot] = useState(config);

  // Auto-dismiss messages
  useEffect(() => {
    if (msg) {
      const t = setTimeout(() => setMsg(null), 3000);
      return () => clearTimeout(t);
    }
  }, [msg]);

  // Fetch current tenant config
  useEffect(() => {
    const fetchTenant = async () => {
      try {
        const data = await apiFetch<Record<string, unknown>>(`/api/v1/tenants/${TENANT_ID}`);
        const methods = data.allowed_auth_methods;
        const next = {
          name: (data.name as string) || "Default Tenant",
          logoDataUrl: (data.logo_url as string) || "",
          allowedAuthMethods: Array.isArray(methods) ? methods as string[] : ["password", "totp"],
          sessionTimeout: Number(data.session_timeout) || 60,
          requireMfa: Boolean(data.require_mfa),
          primaryColor: (data.primary_color as string) || "#6366f1",
          accentColor: (data.accent_color as string) || "#8b5cf6",
        };
        setConfig(next);
        setSnapshot(next);
      } catch (err) {
        setError("Failed to load tenant configuration. Using defaults.");
        // Use defaults if API unavailable
      } finally {
        setLoading(false);
      }
    };
    fetchTenant();
  }, [apiFetch, TENANT_ID]);

  const handleLogoUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    if (!file.type.startsWith("image/")) {
      setMsg("Please select an image file");
      return;
    }
    const reader = new FileReader();
    reader.onload = () => {
      setConfig((prev) => ({ ...prev, logoDataUrl: reader.result as string }));
    };
    reader.readAsDataURL(file);
  };

  const toggleAuthMethod = (key: string) => {
    setConfig((prev) => ({
      ...prev,
      allowedAuthMethods: prev.allowedAuthMethods.includes(key)
        ? prev.allowedAuthMethods.filter((m) => m !== key)
        : [...prev.allowedAuthMethods, key],
    }));
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      await apiFetch(`/api/v1/tenants/${TENANT_ID}`, {
        method: "PUT",
        body: JSON.stringify({
          name: config.name,
          logo_url: config.logoDataUrl,
          allowed_auth_methods: config.allowedAuthMethods,
          session_timeout: config.sessionTimeout,
          require_mfa: config.requireMfa,
          primary_color: config.primaryColor,
          accent_color: config.accentColor,
        }),
      });
      setMsg("Tenant settings saved successfully");
      setSnapshot(config);
    } catch {
      setMsg("Failed to save tenant settings");
    } finally {
      setSaving(false);
    }
  };

  const handleReset = () => {
    setConfig(snapshot);
    setMsg("Changes reset to last saved state");
  };

  const inputCls = "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";
  const labelCls = "mb-1 block text-xs font-medium text-gray-500";
  const cardCls = "rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const headingCls = "mb-4 text-lg font-semibold text-gray-900 dark:text-gray-100";

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="h-6 w-6 animate-spin rounded-full border-2 border-brand-500 border-t-transparent" />
        <span className="ml-2 text-sm text-gray-500">Loading tenant settings...</span>
      </div>
    );
  }

  return (
    <div>
      {error && (
        <div className="mb-4 rounded-lg border border-amber-200 bg-amber-50 dark:border-amber-900 dark:bg-amber-950/30 p-3">
          <p className="text-sm text-amber-600 dark:text-amber-400">{error}</p>
        </div>
      )}
      <div className="mb-6 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-gray-100">
          <Building2 className="h-7 w-7 text-brand-600" />
          Tenant Settings
        </h1>
        <div className="flex gap-2">
          <button
            onClick={handleReset}
            className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 dark:hover:bg-gray-600"
          >
            <RotateCcw className="h-4 w-4" /> Reset
          </button>
          <button
            onClick={handleSave}
            disabled={saving}
            className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
          >
            <Save className="h-4 w-4" /> {saving ? "Saving..." : "Save Changes"}
          </button>
        </div>
      </div>

      {msg && (
        <div className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400">
          {msg}
        </div>
      )}

      <div className="space-y-6">
        {/* General Info */}
        <div className={cardCls}>
          <h2 className={headingCls}>
            <Building2 className="mr-2 inline h-5 w-5 text-brand-600" /> General Information
          </h2>
          <div className="grid gap-6 sm:grid-cols-2">
            <div>
              <label className={labelCls}>Tenant Name</label>
              <input
                value={config.name}
                onChange={(e) => setConfig({ ...config, name: e.target.value })}
                className={inputCls}
                placeholder="My Organization"
              />
            </div>
            <div>
              <label className={labelCls}>Tenant ID</label>
              <input
                value={TENANT_ID}
                disabled
                className={`${inputCls} cursor-not-allowed font-mono text-xs opacity-60`}
              />
            </div>
          </div>
        </div>

        {/* Logo Upload */}
        <div className={cardCls}>
          <h2 className={headingCls}>
            <Upload className="mr-2 inline h-5 w-5 text-brand-600" /> Logo
          </h2>
          <div className="flex items-center gap-6">
            <div className="flex h-24 w-24 shrink-0 items-center justify-center overflow-hidden rounded-full border-2 border-gray-200 bg-gray-100 dark:border-gray-700 dark:bg-gray-700">
              {config.logoDataUrl ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img src={config.logoDataUrl} alt="Logo" className="h-full w-full object-cover" />
              ) : (
                <Building2 className="h-10 w-10 text-gray-400" />
              )}
            </div>
            <div className="flex-1">
              <input
                ref={fileInputRef}
                type="file"
                accept="image/*"
                onChange={handleLogoUpload}
                className="hidden"
              />
              <button
                onClick={() => fileInputRef.current?.click()}
                className="flex items-center gap-2 rounded-lg border border-brand-600 px-4 py-2 text-sm font-medium text-brand-600 hover:bg-brand-50 dark:hover:bg-brand-900/30"
              >
                <Upload className="h-4 w-4" /> Upload Logo
              </button>
              {config.logoDataUrl && (
                <button
                  onClick={() => setConfig({ ...config, logoDataUrl: "" })}
                  className="ml-2 rounded-lg border border-red-300 px-3 py-2 text-sm text-red-600 hover:bg-red-50 dark:border-red-800 dark:hover:bg-red-950"
                >
                  Remove
                </button>
              )}
              <p className="mt-2 text-xs text-gray-400">PNG, JPG, or SVG. Square recommended.</p>
            </div>
          </div>
        </div>

        {/* Auth Policy */}
        <div className={cardCls}>
          <h2 className={headingCls}>
            <Shield className="mr-2 inline h-5 w-5 text-brand-600" /> Authentication Policy
          </h2>
          <p className="mb-4 text-sm text-gray-500 dark:text-gray-400">
            Select which authentication methods are available to users in this tenant.
          </p>
          <div className="grid gap-3 sm:grid-cols-2">
            {AUTH_METHODS.map((method) => {
              const checked = config.allowedAuthMethods.includes(method.key);
              return (
                <label
                  key={method.key}
                  className={`flex cursor-pointer items-start gap-3 rounded-lg border p-4 transition-colors ${
                    checked
                      ? "border-brand-400 bg-brand-50 dark:border-brand-700 dark:bg-brand-900/20"
                      : "border-gray-200 hover:border-gray-300 dark:border-gray-700 dark:hover:border-gray-600"
                  }`}
                >
                  <input
                    type="checkbox"
                    checked={checked}
                    onChange={() => toggleAuthMethod(method.key)}
                    className="mt-0.5 h-4 w-4 rounded"
                  />
                  <method.icon className="mt-0.5 h-5 w-5 text-gray-500 dark:text-gray-400" />
                  <div>
                    <p className="text-sm font-medium text-gray-900 dark:text-gray-100">{method.label}</p>
                    <p className="text-xs text-gray-500 dark:text-gray-400">{method.description}</p>
                  </div>
                  {checked && <Check className="ml-auto mt-0.5 h-4 w-4 text-brand-600" />}
                </label>
              );
            })}
          </div>
        </div>

        {/* Session & MFA */}
        <div className={cardCls}>
          <h2 className={headingCls}>
            <Clock className="mr-2 inline h-5 w-5 text-brand-600" /> Session & Security
          </h2>
          <div className="space-y-6">
            {/* Session Timeout */}
            <div>
              <div className="mb-2 flex items-center justify-between">
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
                  Session Timeout
                </label>
                <span className="rounded-lg bg-gray-100 px-3 py-1 text-sm font-semibold text-gray-900 dark:bg-gray-700 dark:text-gray-100">
                  {config.sessionTimeout} min
                  {config.sessionTimeout >= 60 && (
                    <span className="ml-1 text-xs text-gray-500">
                      ({(config.sessionTimeout / 60).toFixed(1)}h)
                    </span>
                  )}
                </span>
              </div>
              <input
                type="range"
                min={5}
                max={1440}
                step={5}
                value={config.sessionTimeout}
                onChange={(e) => setConfig({ ...config, sessionTimeout: Number(e.target.value) })}
                className="w-full accent-brand-600"
              />
              <div className="mt-1 flex justify-between text-xs text-gray-400">
                <span>5 min</span>
                <span>24h (1440 min)</span>
              </div>
              <input
                type="number"
                min={5}
                max={1440}
                value={config.sessionTimeout}
                onChange={(e) => {
                  const val = Math.min(1440, Math.max(5, Number(e.target.value) || 5));
                  setConfig({ ...config, sessionTimeout: val });
                }}
                className={`${inputCls} mt-2 max-w-[120px]`}
              />
            </div>

            {/* MFA Enforcement */}
            <div className="flex items-center justify-between rounded-lg border border-gray-200 p-4 dark:border-gray-700">
              <div className="flex items-center gap-3">
                <Shield className="h-5 w-5 text-gray-500 dark:text-gray-400" />
                <div>
                  <p className="text-sm font-medium text-gray-900 dark:text-gray-100">
                    Require MFA for all users
                  </p>
                  <p className="text-xs text-gray-500 dark:text-gray-400">
                    Enforce multi-factor authentication across the entire tenant
                  </p>
                </div>
              </div>
              <button
                onClick={() => setConfig({ ...config, requireMfa: !config.requireMfa })}
                className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                  config.requireMfa ? "bg-brand-600" : "bg-gray-300 dark:bg-gray-600"
                }`}
              >
                <span
                  className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                    config.requireMfa ? "translate-x-6" : "translate-x-1"
                  }`}
                />
              </button>
            </div>
          </div>
        </div>

        {/* Branding Colors */}
        <div className={cardCls}>
          <h2 className={headingCls}>
            <Palette className="mr-2 inline h-5 w-5 text-brand-600" /> Branding Colors
          </h2>
          <div className="grid gap-6 sm:grid-cols-2">
            <div>
              <label className={labelCls}>Primary Color</label>
              <div className="flex items-center gap-3">
                <input
                  type="color"
                  value={config.primaryColor}
                  onChange={(e) => setConfig({ ...config, primaryColor: e.target.value })}
                  className="h-10 w-14 rounded border border-gray-300 dark:border-gray-600"
                />
                <input
                  value={config.primaryColor}
                  onChange={(e) => setConfig({ ...config, primaryColor: e.target.value })}
                  className={`${inputCls} font-mono`}
                />
              </div>
            </div>
            <div>
              <label className={labelCls}>Accent Color</label>
              <div className="flex items-center gap-3">
                <input
                  type="color"
                  value={config.accentColor}
                  onChange={(e) => setConfig({ ...config, accentColor: e.target.value })}
                  className="h-10 w-14 rounded border border-gray-300 dark:border-gray-600"
                />
                <input
                  value={config.accentColor}
                  onChange={(e) => setConfig({ ...config, accentColor: e.target.value })}
                  className={`${inputCls} font-mono`}
                />
              </div>
            </div>
          </div>

          {/* Live preview */}
          <div className="mt-4 rounded-lg border border-gray-200 p-6 dark:border-gray-700">
            <p className="mb-3 text-xs font-medium text-gray-500">Preview</p>
            <div className="flex items-center gap-4">
              <div
                className="flex h-12 w-12 items-center justify-center rounded-lg text-lg font-bold text-white"
                style={{ backgroundColor: config.primaryColor }}
              >
                {config.name.charAt(0) || "G"}
              </div>
              <div>
                <p className="font-semibold text-gray-900 dark:text-gray-100">{config.name}</p>
                <button
                  className="mt-1 rounded-lg px-3 py-1 text-xs font-medium text-white"
                  style={{ backgroundColor: config.accentColor }}
                >
                  Sample Button
                </button>
              </div>
              <div
                className="ml-auto rounded-lg px-4 py-2 text-sm font-medium"
                style={{ backgroundColor: config.primaryColor, color: "white" }}
              >
                Primary Button
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

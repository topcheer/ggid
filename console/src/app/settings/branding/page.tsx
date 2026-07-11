"use client";

import { useState, useEffect, useRef } from "react";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";
import {
  Palette,
  Save,
  Loader2,
  Eye,
  Upload,
  X,
  Mail,
} from "lucide-react";

interface BrandingConfig {
  logo_url: string;
  primary_color: string;
  secondary_color: string;
  css_override: string;
  custom_domain: string;
}

const STORAGE_KEY = "ggid_branding_config";
const MAX_LOGO_SIZE = 1024 * 1024; // 1MB

const defaultConfig: BrandingConfig = {
  logo_url: "",
  primary_color: "#6366f1",
  secondary_color: "#8b5cf6",
  css_override: "",
  custom_domain: "",
};

type EmailTemplate = "welcome" | "password-reset" | "magic-link" | "mfa-enroll";

const EMAIL_TEMPLATES: { value: EmailTemplate; label: string }[] = [
  { value: "welcome", label: "Welcome Email" },
  { value: "password-reset", label: "Password Reset" },
  { value: "magic-link", label: "Magic Link Login" },
  { value: "mfa-enroll", label: "MFA Enrollment" },
];

const EMAIL_SUBJECTS: Record<EmailTemplate, string> = {
  welcome: "Welcome to GGID!",
  "password-reset": "Reset your password",
  "magic-link": "Your magic sign-in link",
  "mfa-enroll": "Enroll in Multi-Factor Authentication",
};

const EMAIL_BODIES: Record<EmailTemplate, string> = {
  welcome:
    "Your account has been created. Click the button below to get started and set up your profile.",
  "password-reset":
    "We received a request to reset your password. Click the button below to choose a new password.",
  "magic-link":
    "Use the button below to securely sign in to your account. This link expires in 15 minutes.",
  "mfa-enroll":
    "Enhance your account security by enrolling in multi-factor authentication. Click the button below to set up your authenticator app.",
};

const EMAIL_BUTTONS: Record<EmailTemplate, string> = {
  welcome: "Get Started",
  "password-reset": "Reset Password",
  "magic-link": "Sign In",
  "mfa-enroll": "Set Up MFA",
};

export default function BrandingSettingsPage() {
  const { apiFetch } = useApi();
  const t = useTranslations();
  const [config, setConfig] = useState<BrandingConfig>(defaultConfig);
  const [msg, setMsg] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  // Logo file upload preview (data URL for display)
  const [logoPreview, setLogoPreview] = useState<string>("");
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Email template preview selection
  const [emailTemplate, setEmailTemplate] = useState<EmailTemplate>("welcome");

  // Load from localStorage or API
  useEffect(() => {
    const stored = typeof window !== "undefined" ? localStorage.getItem(STORAGE_KEY) : null;
    if (stored) {
      try {
        const parsed = JSON.parse(stored);
        setConfig({ ...defaultConfig, ...parsed });
        if (parsed.logo_url) setLogoPreview(parsed.logo_url);
      } catch {
        // ignore
      }
    }
  }, []);

  useEffect(() => {
    if (msg) {
      const t = setTimeout(() => setMsg(null), 3000);
      return () => clearTimeout(t);
    }
  }, [msg]);

  const handleSave = async () => {
    setSaving(true);
    try {
      await apiFetch("/api/v1/settings/branding", {
        method: "POST",
        body: JSON.stringify(config),
      });
      setMsg("Branding settings saved to server");
    } catch {
      // Fallback: save to localStorage
      localStorage.setItem(STORAGE_KEY, JSON.stringify(config));
      setMsg("Endpoint unavailable — saved to localStorage");
    } finally {
      setSaving(false);
    }
  };

  const handleLogoUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    // Accept only SVG and PNG
    if (!file.type.match(/image\/(svg\+xml|png)/) && !file.name.match(/\.(svg|png)$/i)) {
      setMsg("Please upload an SVG or PNG file");
      return;
    }
    // Enforce 1MB max size
    if (file.size > MAX_LOGO_SIZE) {
      setMsg("Logo file must be 1MB or less");
      return;
    }
    const reader = new FileReader();
    reader.onload = () => {
      const dataUrl = reader.result as string;
      setLogoPreview(dataUrl);
      setConfig({ ...config, logo_url: dataUrl });
    };
    reader.readAsDataURL(file);
  };

  const handleRemoveLogo = () => {
    setLogoPreview("");
    setConfig({ ...config, logo_url: "" });
    if (fileInputRef.current) fileInputRef.current.value = "";
  };

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-2xl font-bold dark:text-gray-100">
          <Palette className="h-6 w-6 text-brand-600" /> {t("branding.loginCustomization")}
        </h1>
      </div>

      {msg && (
        <div className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400">
          {msg}
        </div>
      )}

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        {/* Config Form */}
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-lg font-semibold dark:text-gray-100">{t("branding.title")}</h2>
            <button
              onClick={handleSave}
              disabled={saving}
              className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
            >
              {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />} {t("common.save")}
            </button>
          </div>

          <div className="space-y-4">
            {/* Logo Upload */}
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">{t("branding.logo")}</label>
              {logoPreview ? (
                <div className="flex items-center gap-3 rounded-lg border border-gray-200 p-3 dark:border-gray-600">
                  {/* eslint-disable-next-line @next/next/no-img-element */}
                  <img
                    src={logoPreview}
                    alt="Logo preview"
                    className="h-12 max-w-32 object-contain"
                  />
                  <button
                    onClick={handleRemoveLogo}
                    className="flex items-center gap-1 text-xs text-red-500 hover:text-red-700"
                   aria-label="Close">
                    <X className="h-3.5 w-3.5" /> {t("common.remove")}
                  </button>
                </div>
              ) : (
                <div
                  onClick={() => fileInputRef.current?.click()}
                  className="flex cursor-pointer flex-col items-center justify-center rounded-lg border-2 border-dashed border-gray-300 py-8 transition hover:border-brand-400 dark:border-gray-600"
                >
                  <Upload className="mb-2 h-6 w-6 text-gray-400" />
                  <span className="text-xs text-gray-500">{t("branding.clickToUpload")}</span>
                  <span className="mt-0.5 text-xs text-gray-400">{t("branding.svgPng")}</span>
                </div>
              )}
              <input
                ref={fileInputRef}
                type="file"
                accept="image/svg+xml,image/png,.svg,.png"
                onChange={handleLogoUpload}
                className="hidden"
              />
            </div>

            {/* Primary Color */}
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">{t("branding.primaryColor")}</label>
              <div className="flex items-center gap-2">
                <input
                  type="color"
                  value={config.primary_color}
                  onChange={(e) => setConfig({ ...config, primary_color: e.target.value })}
                  className="h-9 w-12 rounded border border-gray-300 dark:border-gray-600"
                />
                <input
                  value={config.primary_color}
                  onChange={(e) => setConfig({ ...config, primary_color: e.target.value })}
                  className="flex-1 rounded-lg border border-gray-300 px-3 py-2 text-sm font-mono dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
                />
              </div>
            </div>

            {/* Secondary Color */}
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">{t("branding.secondaryColor")}</label>
              <div className="flex items-center gap-2">
                <input
                  type="color"
                  value={config.secondary_color}
                  onChange={(e) => setConfig({ ...config, secondary_color: e.target.value })}
                  className="h-9 w-12 rounded border border-gray-300 dark:border-gray-600"
                />
                <input
                  value={config.secondary_color}
                  onChange={(e) => setConfig({ ...config, secondary_color: e.target.value })}
                  className="flex-1 rounded-lg border border-gray-300 px-3 py-2 text-sm font-mono dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
                />
              </div>
            </div>

            {/* Custom Domain */}
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">{t("branding.customDomain")}</label>
              <input
                value={config.custom_domain}
                onChange={(e) => setConfig({ ...config, custom_domain: e.target.value })}
                placeholder="login.example.com"
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm font-mono dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
              />
              <p className="mt-1 text-xs text-gray-400">{t("branding.cnameHint")}</p>
            </div>

            {/* CSS Override */}
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">{t("branding.cssOverride")}</label>
              <textarea
                value={config.css_override}
                onChange={(e) => setConfig({ ...config, css_override: e.target.value })}
                rows={8}
                placeholder="/* Custom CSS applied to the login page */\n.login-card {\n  border-radius: 16px;\n}"
                className="w-full rounded-lg border border-gray-300 px-3 py-2 font-mono text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
              />
            </div>
          </div>
        </div>

        {/* Live Previews */}
        <div className="space-y-6">
          {/* Login Page Preview */}
          <div className="sticky top-6 rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <h2 className="mb-4 flex items-center gap-2 text-lg font-semibold dark:text-gray-100">
              <Eye className="h-5 w-5 text-brand-600" /> {t("branding.loginPreview")}
            </h2>

            {/* Simulated login page background */}
            <div
              className="overflow-hidden rounded-lg border border-gray-200 dark:border-gray-700"
              style={{ backgroundColor: "#f9fafb" }}
            >
              {/* Top bar */}
              <div
                className="flex h-10 items-center px-4"
                style={{ backgroundColor: config.primary_color }}
              >
                {config.logo_url ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img
                    src={config.logo_url}
                    alt="Logo"
                    className="h-6 object-contain"
                    onError={(e) => {
                      (e.target as HTMLImageElement).style.display = "none";
                    }}
                  />
                ) : (
                  <div
                    className="flex h-6 w-6 items-center justify-center rounded text-xs font-bold text-white"
                    style={{ backgroundColor: "rgba(255,255,255,0.3)" }}
                  >
                    G
                  </div>
                )}
                <span className="ml-2 text-sm font-medium text-white">GGID Login</span>
                {config.custom_domain && (
                  <span className="ml-auto text-xs text-white/80">{config.custom_domain}</span>
                )}
              </div>

              {/* Login card */}
              <div className="flex items-center justify-center p-8">
                <div className="w-full max-w-xs rounded-xl border border-gray-200 bg-white p-6 shadow-lg">
                  {/* Logo in card */}
                  <div className="mb-4 flex justify-center">
                    {config.logo_url ? (
                      // eslint-disable-next-line @next/next/no-img-element
                      <img
                        src={config.logo_url}
                        alt="Logo"
                        className="h-12 object-contain"
                        onError={(e) => {
                          (e.target as HTMLImageElement).style.display = "none";
                        }}
                      />
                    ) : (
                      <div
                        className="flex h-12 w-12 items-center justify-center rounded-xl text-xl font-bold text-white"
                        style={{ backgroundColor: config.primary_color }}
                      >
                        G
                      </div>
                    )}
                  </div>

                  <h3 className="mb-1 text-center text-lg font-semibold text-gray-900">Sign In</h3>
                  <p className="mb-4 text-center text-xs text-gray-500">
                    {config.custom_domain || "identity.ggid.dev"}
                  </p>

                  <div className="space-y-3">
                    <input
                      type="text"
                      disabled
                      placeholder="Username"
                      className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-400"
                    />
                    <input
                      type="password"
                      disabled
                      placeholder="Password"
                      className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-400"
                    />
                    <button
                      type="button"
                      disabled
                      className="w-full rounded-lg py-2 text-sm font-medium text-white"
                      style={{ backgroundColor: config.primary_color }}
                    >
                      Sign In
                    </button>
                  </div>

                  <p className="mt-4 text-center text-xs text-gray-400">
                    Forgot password?
                  </p>
                </div>
              </div>
            </div>
          </div>

          {/* Email Template Preview */}
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <h2 className="mb-4 flex items-center gap-2 text-lg font-semibold dark:text-gray-100">
              <Mail className="h-5 w-5 text-brand-600" /> Email Template Preview
            </h2>

            {/* Template selector */}
            <div className="mb-4">
              <label className="mb-1 block text-xs font-medium text-gray-500">Template</label>
              <select
                value={emailTemplate}
                onChange={(e) => setEmailTemplate(e.target.value as EmailTemplate)}
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
              >
                {EMAIL_TEMPLATES.map((t) => (
                  <option key={t.value} value={t.value}>
                    {t.label}
                  </option>
                ))}
              </select>
            </div>

            {/* Rendered email preview */}
            <div className="overflow-hidden rounded-lg border border-gray-200 dark:border-gray-700">
              {/* Email header bar */}
              <div
                className="flex items-center gap-2 px-4 py-3"
                style={{ backgroundColor: config.primary_color }}
              >
                {config.logo_url ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img
                    src={config.logo_url}
                    alt="Logo"
                    className="h-5 object-contain"
                    onError={(e) => {
                      (e.target as HTMLImageElement).style.display = "none";
                    }}
                  />
                ) : (
                  <div
                    className="flex h-5 w-5 items-center justify-center rounded text-xs font-bold text-white"
                    style={{ backgroundColor: "rgba(255,255,255,0.3)" }}
                  >
                    G
                  </div>
                )}
                <span className="text-sm font-semibold text-white">GGID</span>
              </div>

              {/* Email body */}
              <div className="bg-white p-6 dark:bg-gray-900">
                <h3 className="mb-3 text-base font-semibold text-gray-900 dark:text-gray-100">
                  {EMAIL_SUBJECTS[emailTemplate]}
                </h3>
                <p className="mb-4 text-sm text-gray-600 dark:text-gray-400">
                  Hi there,
                </p>
                <p className="mb-4 text-sm text-gray-600 dark:text-gray-400">
                  {EMAIL_BODIES[emailTemplate]}
                </p>

                {/* Branded CTA button */}
                <div className="mb-4">
                  <span
                    className="inline-block cursor-default rounded-lg px-6 py-2.5 text-sm font-medium text-white"
                    style={{ backgroundColor: config.primary_color }}
                  >
                    {EMAIL_BUTTONS[emailTemplate]}
                  </span>
                </div>

                <p className="text-xs text-gray-400">
                  If you didn&apos;t request this, you can safely ignore this email.
                </p>
                <hr className="my-4 border-gray-200 dark:border-gray-700" />
                <p className="text-xs text-gray-400">
                  {config.custom_domain || "identity.ggid.dev"} — Powered by GGID
                </p>
              </div>
            </div>
          </div>

          {/* CSS Override Preview */}
          {config.css_override && (
            <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
              <p className="mb-1 text-xs font-medium text-gray-500">CSS Override (raw):</p>
              <pre className="max-h-32 overflow-auto rounded-lg bg-gray-900 p-3 text-xs text-green-400">
                {config.css_override}
              </pre>
            </div>
          )}

          {/* Summary */}
          <div className="space-y-2 rounded-lg border border-gray-100 p-3 dark:border-gray-700">
            <div className="flex items-center justify-between text-xs">
              <span className="text-gray-500">Primary Color</span>
              <span className="font-mono text-gray-700 dark:text-gray-300">{config.primary_color}</span>
            </div>
            <div className="flex items-center justify-between text-xs">
              <span className="text-gray-500">Secondary Color</span>
              <span className="font-mono text-gray-700 dark:text-gray-300">{config.secondary_color}</span>
            </div>
            <div className="flex items-center justify-between text-xs">
              <span className="text-gray-500">Logo</span>
              <span className="text-gray-700 dark:text-gray-300">
                {config.logo_url ? "Custom" : "Default (G)"}
              </span>
            </div>
            <div className="flex items-center justify-between text-xs">
              <span className="text-gray-500">Domain</span>
              <span className="text-gray-700 dark:text-gray-300">
                {config.custom_domain || "Not set"}
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

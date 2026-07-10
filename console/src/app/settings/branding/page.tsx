"use client";

import { useState, useEffect } from "react";
import { useApi } from "@/lib/api";
import { Palette, Save, Loader2, Eye } from "lucide-react";

interface BrandingConfig {
  logo_url: string;
  primary_color: string;
  css_override: string;
  custom_domain: string;
}

const STORAGE_KEY = "ggid_branding_config";

const defaultConfig: BrandingConfig = {
  logo_url: "",
  primary_color: "#6366f1",
  css_override: "",
  custom_domain: "",
};

export default function BrandingSettingsPage() {
  const { apiFetch } = useApi();
  const [config, setConfig] = useState<BrandingConfig>(defaultConfig);
  const [msg, setMsg] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  // Load from localStorage or API
  useEffect(() => {
    const stored = typeof window !== "undefined" ? localStorage.getItem(STORAGE_KEY) : null;
    if (stored) {
      try {
        setConfig({ ...defaultConfig, ...JSON.parse(stored) });
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

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-2xl font-bold dark:text-gray-100">
          <Palette className="h-6 w-6 text-brand-600" /> Login Customization
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
            <h2 className="text-lg font-semibold dark:text-gray-100">Branding Settings</h2>
            <button
              onClick={handleSave}
              disabled={saving}
              className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
            >
              {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />} Save
            </button>
          </div>

          <div className="space-y-4">
            {/* Logo URL */}
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">Logo URL</label>
              <input
                value={config.logo_url}
                onChange={(e) => setConfig({ ...config, logo_url: e.target.value })}
                placeholder="https://example.com/logo.png"
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
              />
            </div>

            {/* Primary Color */}
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">Primary Color</label>
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

            {/* Custom Domain */}
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">Custom Domain</label>
              <input
                value={config.custom_domain}
                onChange={(e) => setConfig({ ...config, custom_domain: e.target.value })}
                placeholder="login.example.com"
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm font-mono dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
              />
              <p className="mt-1 text-xs text-gray-400">CNAME this domain to your GGID instance</p>
            </div>

            {/* CSS Override */}
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">CSS Override</label>
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

        {/* Live Preview */}
        <div>
          <div className="sticky top-6 rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <h2 className="mb-4 flex items-center gap-2 text-lg font-semibold dark:text-gray-100">
              <Eye className="h-5 w-5 text-brand-600" /> Live Preview
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

            {/* CSS Override Preview */}
            {config.css_override && (
              <div className="mt-4">
                <p className="mb-1 text-xs font-medium text-gray-500">CSS Override (raw):</p>
                <pre className="max-h-32 overflow-auto rounded-lg bg-gray-900 p-3 text-xs text-green-400">
                  {config.css_override}
                </pre>
              </div>
            )}

            {/* Summary */}
            <div className="mt-4 space-y-2 rounded-lg border border-gray-100 p-3 dark:border-gray-700">
              <div className="flex items-center justify-between text-xs">
                <span className="text-gray-500">Primary Color</span>
                <span className="font-mono text-gray-700 dark:text-gray-300">{config.primary_color}</span>
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
    </div>
  );
}

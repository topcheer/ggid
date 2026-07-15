"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  Palette,
  Save,
  Loader2,
  Eye,
  Upload,
  X,
  Mail,
  RotateCcw,
  Code2,
  Sun,
  Moon,
  FileWarning,
} from "lucide-react";

interface BrandingCustomConfig {
  logo_url: string;
  primary_color: string;
  secondary_color: string;
  background_color: string;
  css_override: string;
}

const STORAGE_KEY = "ggid_branding_custom";
const MAX_LOGO_SIZE = 1024 * 1024; // 1MB
const MAX_CSS_SIZE = 10240; // 10KB

const defaultConfig: BrandingCustomConfig = {
  logo_url: "",
  primary_color: "#6366f1",
  secondary_color: "#8b5cf6",
  background_color: "#f9fafb",
  css_override: "",
};

const COLOR_PRESETS = [
  "#6366f1", // Indigo
  "#3b82f6", // Blue
  "#10b981", // Emerald
  "#f59e0b", // Amber
  "#ef4444", // Red
  "#8b5cf6", // Violet
];

export default function BrandingCustomPage() {
  const { apiFetch, TENANT_ID } = useApi();
  const [config, setConfig] = useState<BrandingCustomConfig>(defaultConfig);
  const [savedConfig, setSavedConfig] = useState<BrandingCustomConfig>(defaultConfig);
  const [msg, setMsg] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [logoPreview, setLogoPreview] = useState<string>("");
  const [logoPreviewBg, setLogoPreviewBg] = useState<"light" | "dark">("light");
  const [cssOverLimit, setCssOverLimit] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Load saved config
  useEffect(() => {
    const stored = typeof window !== "undefined" ? localStorage.getItem(STORAGE_KEY) : null;
    if (stored) {
      try {
        const parsed = JSON.parse(stored);
        const cfg = { ...defaultConfig, ...parsed };
        setConfig(cfg);
        setSavedConfig(cfg);
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

  const updateField = (field: keyof BrandingCustomConfig, value: string) => {
    setConfig((prev) => ({ ...prev, [field]: value }));
  };

  const handleCssChange = (value: string) => {
    setCssOverLimit(value.length > MAX_CSS_SIZE);
    updateField("css_override", value);
  };

  const handleLogoUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    if (!file.type.match(/image\/(svg\+xml|png)/) && !file.name.match(/\.(svg|png)$/i)) {
      setMsg("Please upload an SVG or PNG file");
      return;
    }
    if (file.size > MAX_LOGO_SIZE) {
      setMsg("Logo file must be 1MB or less");
      return;
    }
    const reader = new FileReader();
    reader.onload = () => {
      const dataUrl = reader.result as string;
      setLogoPreview(dataUrl);
      updateField("logo_url", dataUrl);
    };
    reader.readAsDataURL(file);
  };

  const handleRemoveLogo = () => {
    setLogoPreview("");
    updateField("logo_url", "");
    if (fileInputRef.current) fileInputRef.current.value = "";
  };

  const handleSave = async () => {
    if (cssOverLimit) {
      setMsg("CSS exceeds 10KB limit — please reduce before saving");
      return;
    }
    setSaving(true);
    try {
      await apiFetch(`/api/v1/tenants/${TENANT_ID}/branding`, {
        method: "PUT",
        body: JSON.stringify(config),
      });
      setSavedConfig(config);
      localStorage.setItem(STORAGE_KEY, JSON.stringify(config));
      setMsg("Branding saved to server");
    } catch {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(config));
      setSavedConfig(config);
      setMsg("Endpoint unavailable — saved to localStorage");
    } finally {
      setSaving(false);
    }
  };

  const handleReset = () => {
    setConfig(savedConfig);
    setLogoPreview(savedConfig.logo_url);
    setCssOverLimit(false);
    setMsg("Reverted to saved state");
  };

  const handleResetCss = () => {
    updateField("css_override", "");
    setCssOverLimit(false);
  };

  const hasChanges = JSON.stringify(config) !== JSON.stringify(savedConfig);

  const inputCls = "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";
  const labelCls = "mb-1 block text-xs font-medium text-gray-500";
  const cardCls = "rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold dark:text-gray-100">
            <Palette className="h-6 w-6 text-brand-600" /> Branding Customization
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Customize logo, colors, and CSS for your login page and email templates.
          </p>
        </div>
        <div className="flex gap-2">
          <button
            onClick={handleReset}
            disabled={!hasChanges}
            aria-label="Revert branding changes"
            className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-700 hover:bg-gray-50 disabled:opacity-40 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
          >
            <RotateCcw className="h-4 w-4" /> Reset
          </button>
          <button
            onClick={handleSave}
            disabled={saving || !hasChanges || cssOverLimit}
            aria-label="Save branding customizations"
            className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
          >
            {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />} Save
          </button>
        </div>
      </div>

      {msg && (
        <div className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400">
          {msg}
        </div>
      )}

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        {/* ===== LEFT: Configuration ===== */}
        <div className="space-y-6">
          {/* Logo Upload */}
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-lg font-semibold dark:text-gray-100">
              <Upload className="h-5 w-5 text-brand-600" /> Logo Upload
            </h2>

            {logoPreview ? (
              <div className="space-y-3">
                <div className="flex items-center gap-3 rounded-lg border border-gray-200 p-3 dark:border-gray-600">
                  <div className="flex h-12 w-12 items-center justify-center overflow-hidden rounded">
                    {/* eslint-disable-next-line @next/next/no-img-element */}
                    <img src={logoPreview} alt="Uploaded logo preview" className="max-h-12 max-w-12 object-contain" />
                  </div>
                  <span className="flex-1 truncate text-xs text-gray-500">{config.logo_url.substring(0, 60)}...</span>
                  <button
                    onClick={handleRemoveLogo}
                    className="flex items-center gap-1 text-xs text-red-500 hover:text-red-700"
                    aria-label="Remove logo">
                    <X className="h-3.5 w-3.5" /> Remove
                  </button>
                </div>

                {/* Preview on light/dark backgrounds */}
                <div>
                  <div className="mb-2 flex items-center gap-2">
                    <button
                      onClick={() => setLogoPreviewBg("light")}
                      aria-label="Preview logo on light background"
                      className={`flex items-center gap-1 rounded-md px-2 py-1 text-xs ${logoPreviewBg === "light" ? "bg-brand-100 text-brand-700 dark:bg-brand-900/40 dark:text-brand-400" : "text-gray-400"}`}
                    >
                      <Sun className="h-3 w-3" /> Light
                    </button>
                    <button
                      onClick={() => setLogoPreviewBg("dark")}
                      aria-label="Preview logo on dark background"
                      className={`flex items-center gap-1 rounded-md px-2 py-1 text-xs ${logoPreviewBg === "dark" ? "bg-brand-100 text-brand-700 dark:bg-brand-900/40 dark:text-brand-400" : "text-gray-400"}`}
                    >
                      <Moon className="h-3 w-3" /> Dark
                    </button>
                  </div>
                  <div
                    className="flex items-center justify-center rounded-lg border p-6"
                    style={{
                      backgroundColor: logoPreviewBg === "light" ? "#ffffff" : "#1f2937",
                      borderColor: logoPreviewBg === "light" ? "#e5e7eb" : "#374151",
                    }}
                  >
                    {/* eslint-disable-next-line @next/next/no-img-element */}
                    <img src={logoPreview} alt="Logo preview on selected background" className="max-h-16 max-w-full object-contain" />
                  </div>
                </div>
              </div>
            ) : (
              <div
                onClick={() => fileInputRef.current?.click()}
                role="button"
                aria-label="Upload logo file"
                tabIndex={0}
                onKeyDown={(e) => { if (e.key === "Enter" || e.key === " ") fileInputRef.current?.click(); }}
                className="flex cursor-pointer flex-col items-center justify-center rounded-lg border-2 border-dashed border-gray-300 py-8 transition hover:border-brand-400 dark:border-gray-600"
              >
                <Upload className="mb-2 h-6 w-6 text-gray-400" />
                <span className="text-xs text-gray-500">Drag &amp; drop or click to upload</span>
                <span className="mt-0.5 text-xs text-gray-400">PNG or SVG, max 1MB</span>
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

          {/* Color Scheme */}
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-lg font-semibold dark:text-gray-100">
              <Palette className="h-5 w-5 text-brand-600" /> Color Scheme
            </h2>

            {/* Primary */}
            <ColorPicker
              label="Primary Color"
              value={config.primary_color}
              presets={COLOR_PRESETS}
              onChange={(v) => updateField("primary_color", v)}
            />

            {/* Secondary */}
            <ColorPicker
              label="Secondary / Accent"
              value={config.secondary_color}
              presets={COLOR_PRESETS}
              onChange={(v) => updateField("secondary_color", v)}
            />

            {/* Background */}
            <ColorPicker
              label="Background Color"
              value={config.background_color}
              presets={["#f9fafb", "#ffffff", "#111827", "#0f172a", "#1e1b4b", "#fef3c7"]}
              onChange={(v) => updateField("background_color", v)}
            />

            {/* Live swatch bar */}
            <div className="mt-4">
              <label className={labelCls}>Color Combination Preview</label>
              <div className="flex h-12 overflow-hidden rounded-lg border border-gray-200 dark:border-gray-600">
                <div className="flex-1 flex items-center justify-center text-xs font-medium text-white" style={{ backgroundColor: config.primary_color }}>
                  Primary
                </div>
                <div className="flex-1 flex items-center justify-center text-xs font-medium text-white" style={{ backgroundColor: config.secondary_color }}>
                  Accent
                </div>
                <div className="flex-1 flex items-center justify-center text-xs font-medium text-gray-700 dark:text-gray-300" style={{ backgroundColor: config.background_color }}>
                  Background
                </div>
              </div>
            </div>
          </div>

          {/* Custom CSS */}
          <div className={cardCls}>
            <div className="mb-4 flex items-center justify-between">
              <h2 className="flex items-center gap-2 text-lg font-semibold dark:text-gray-100">
                <Code2 className="h-5 w-5 text-brand-600" /> Custom CSS Injection
              </h2>
              <button
                onClick={handleResetCss}
                aria-label="Reset custom CSS to default"
                className="flex items-center gap-1 text-xs text-gray-500 hover:text-red-500"
              >
                <RotateCcw className="h-3 w-3" /> Reset to default
              </button>
            </div>
            <textarea
              value={config.css_override}
              onChange={(e) => handleCssChange(e.target.value)}
              rows={8}
              placeholder={`/* Custom CSS applied to login page */\n.login-card {\n  border-radius: 16px;\n  box-shadow: 0 4px 24px rgba(0,0,0,0.1);\n}`}
              className={`w-full rounded-lg border px-3 py-2 font-mono text-sm dark:bg-gray-700 dark:text-gray-200 ${cssOverLimit ? "border-red-400" : "border-gray-300 dark:border-gray-600"}`}
            />
            <div className="mt-2 flex items-center justify-between text-xs">
              <span className={cssOverLimit ? "flex items-center gap-1 text-red-500" : "text-gray-400"}>
                {cssOverLimit && <FileWarning className="h-3 w-3" />}
                {cssOverLimit
                  ? `Exceeds 10KB limit (${config.css_override.length} bytes)`
                  : `${config.css_override.length.toLocaleString()} / 10,000 bytes`}
              </span>
              <span className="text-gray-400">Applied to all public-facing pages</span>
            </div>
          </div>
        </div>

        {/* ===== RIGHT: Live Previews ===== */}
        <div className="space-y-6">
          {/* Login Page Preview */}
          <div className="sticky top-6 rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <h2 className="mb-4 flex items-center gap-2 text-lg font-semibold dark:text-gray-100">
              <Eye className="h-5 w-5 text-brand-600" /> Login Page Preview
            </h2>

            <div
              className="overflow-hidden rounded-lg border border-gray-200 dark:border-gray-700"
              style={{ backgroundColor: config.background_color }}
            >
              {/* Top banner */}
              <div className="flex h-10 items-center px-4" style={{ backgroundColor: config.primary_color }}>
                {config.logo_url ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img
                    src={config.logo_url}
                    alt="Header logo preview"
                    className="h-6 object-contain"
                    onError={(e) => { (e.target as HTMLImageElement).style.display = "none"; }}
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
              </div>

              {/* Login card */}
              <div className="flex items-center justify-center p-8">
                <div className="w-full max-w-xs rounded-xl border border-gray-200 bg-white p-6 shadow-lg">
                  {/* Logo */}
                  <div className="mb-4 flex justify-center">
                    {config.logo_url ? (
                      // eslint-disable-next-line @next/next/no-img-element
                      <img
                        src={config.logo_url}
                        alt="Login card logo preview"
                        className="h-12 object-contain"
                        onError={(e) => { (e.target as HTMLImageElement).style.display = "none"; }}
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
                  <p className="mb-4 text-center text-xs text-gray-500">identity.ggid.dev</p>

                  <div className="space-y-3">
                    <input
                      type="text"
                      disabled
                      placeholder="Email"
                      className="w-full rounded-lg border px-3 py-2 text-sm text-gray-400"
                      style={{ borderColor: config.secondary_color + "44" }}
                    />
                    <input
                      type="password"
                      disabled
                      placeholder="Password"
                      className="w-full rounded-lg border px-3 py-2 text-sm text-gray-400"
                      style={{ borderColor: config.secondary_color + "44" }}
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

                  <p className="mt-4 text-center text-xs" style={{ color: config.secondary_color }}>
                    Forgot password?
                  </p>
                </div>
              </div>
            </div>
          </div>

          {/* Email Template Preview */}
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-lg font-semibold dark:text-gray-100">
              <Mail className="h-5 w-5 text-brand-600" /> Email Template Branding
            </h2>
            <div className="overflow-hidden rounded-lg border border-gray-200 dark:border-gray-700">
              {/* Header banner */}
              <div
                className="flex items-center gap-2 px-4 py-3"
                style={{ backgroundColor: config.primary_color }}
              >
                {config.logo_url ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img
                    src={config.logo_url}
                    alt="Email header logo preview"
                    className="h-5 object-contain"
                    onError={(e) => { (e.target as HTMLImageElement).style.display = "none"; }}
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

              {/* Body */}
              <div className="bg-white p-6 dark:bg-gray-900">
                <h3 className="mb-3 text-base font-semibold text-gray-900 dark:text-gray-100">
                  Welcome to GGID!
                </h3>
                <p className="mb-4 text-sm text-gray-600 dark:text-gray-400">
                  Your account has been created. Click the button below to get started.
                </p>
                <div className="mb-4">
                  <span
                    className="inline-block cursor-default rounded-lg px-6 py-2.5 text-sm font-medium text-white"
                    style={{ backgroundColor: config.primary_color }}
                  >
                    Get Started
                  </span>
                </div>
                <p className="text-xs text-gray-400">
                  identity.ggid.dev — Powered by GGID
                </p>
              </div>
            </div>
          </div>

          {/* CSS raw preview */}
          {config.css_override && (
            <div className={cardCls}>
              <p className="mb-1 text-xs font-medium text-gray-500">CSS Override (raw preview):</p>
              <pre className="max-h-40 overflow-auto rounded-lg bg-gray-900 p-3 text-xs text-green-400">
                {config.css_override}
              </pre>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

// --- Color Picker sub-component ---

function ColorPicker({
  label,
  value,
  presets,
  onChange,
}: {
  label: string;
  value: string;
  presets: string[];
  onChange: (value: string) => void;
}) {
  const labelCls = "mb-1 block text-xs font-medium text-gray-500";
  return (
    <div className="mb-4">
      <label className={labelCls}>{label}</label>
      <div className="flex items-center gap-2">
        <input
          type="color"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className="h-9 w-12 shrink-0 rounded border border-gray-300 dark:border-gray-600"
        />
        <input
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className="flex-1 rounded-lg border border-gray-300 px-3 py-2 text-sm font-mono dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
        />
      </div>
      {/* Presets */}
      <div className="mt-2 flex items-center gap-1.5">
        {presets.map((c) => (
          <button
            key={c}
            onClick={() => onChange(c)}
            className={`h-6 w-6 rounded-full border-2 transition ${value.toLowerCase() === c.toLowerCase() ? "border-gray-800 dark:border-white" : "border-transparent hover:border-gray-400"}`}
            style={{ backgroundColor: c }}
            title={c}
          />
        ))}
      </div>
    </div>
  );
}

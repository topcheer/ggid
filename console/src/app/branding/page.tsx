"use client";

import { useState, useRef } from "react";
import { useApi } from "@/lib/api";
import { Upload, Save, RotateCcw, Palette, Code, Mail, Eye, Check } from "lucide-react";

const PRESET_COLORS = [
  { name: "Ocean", primary: "#0066CC", secondary: "#003D7A", accent: "#00C2FF" },
  { name: "Forest", primary: "#15803D", secondary: "#14532D", accent: "#4ADE80" },
  { name: "Sunset", primary: "#EA580C", secondary: "#7C2D12", accent: "#FB923C" },
  { name: "Royal", primary: "#7C3AED", secondary: "#4C1D95", accent: "#A78BFA" },
  { name: "Rose", primary: "#E11D48", secondary: "#881337", accent: "#FB7185" },
  { name: "Slate", primary: "#334155", secondary: "#1E293B", accent: "#64748B" },
];

export default function BrandingPage() {
  const { API_BASE, TENANT_ID } = useApi();
  const [logo, setLogo] = useState<string>("");
  const [logoName, setLogoName] = useState<string>("");
  const [primaryColor, setPrimaryColor] = useState("#0066CC");
  const [secondaryColor, setSecondaryColor] = useState("#003D7A");
  const [accentColor, setAccentColor] = useState("#00C2FF");
  const [customCss, setCustomCss] = useState("");
  const [activeTab, setActiveTab] = useState<"login" | "email">("login");
  const [saved, setSaved] = useState(false);
  const fileRef = useRef<HTMLInputElement>(null);

  const handleLogoUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      const reader = new FileReader();
      reader.onload = () => {
        setLogo(reader.result as string);
        setLogoName(file.name);
      };
      reader.readAsDataURL(file);
    }
  };

  const handleSave = async () => {
    try {
      await fetch(`${API_BASE}/api/v1/tenants/${TENANT_ID}/branding`, {
        method: "PUT",
        headers: { "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify({ logo, primaryColor, secondaryColor, accentColor, customCss }),
      });
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
    } catch (e) { /* noop */ }
  };

  const handleReset = () => {
    setLogo(""); setLogoName(""); setPrimaryColor("#0066CC");
    setSecondaryColor("#003D7A"); setAccentColor("#00C2FF"); setCustomCss("");
  };

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Branding Settings</h1>
          <p className="text-sm text-gray-500 mt-1">Customize logos, colors, and styling</p>
        </div>
        <div className="flex gap-2">
          <button onClick={handleReset} className="flex items-center gap-1.5 px-4 py-2 text-sm text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg transition">
            <RotateCcw className="w-4 h-4" /> Reset
          </button>
          <button onClick={handleSave} className="flex items-center gap-1.5 px-4 py-2 text-sm text-white rounded-lg transition" style={{ backgroundColor: primaryColor }}>
            {saved ? <><Check className="w-4 h-4" /> Saved!</> : <><Save className="w-4 h-4" /> Save</>}
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Left: Configuration */}
        <div className="space-y-6">
          {/* Logo Upload */}
          <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-5">
            <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
              <Upload className="w-4 h-4" /> Logo Upload
            </h3>
            <div className="flex items-center gap-4">
              <div className="w-20 h-20 rounded-xl border-2 border-dashed border-gray-300 dark:border-gray-700 flex items-center justify-center overflow-hidden bg-gray-50 dark:bg-gray-800">
                {logo ? (
                  <img src={logo} alt="Logo" className="w-full h-full object-contain" />
                ) : (
                  <span className="text-xs text-gray-400">No logo</span>
                )}
              </div>
              <div className="flex-1">
                <input ref={fileRef} type="file" accept="image/png,image/svg+xml,image/jpeg" onChange={handleLogoUpload} className="hidden" />
                <button onClick={() => fileRef.current?.click()} className="px-3 py-1.5 text-xs font-medium text-gray-700 dark:text-gray-300 border border-gray-300 dark:border-gray-700 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-800 transition">
                  Choose File
                </button>
                {logoName && <p className="text-xs text-gray-500 mt-1">{logoName}</p>}
                <p className="text-xs text-gray-400 mt-1">PNG, SVG. Max 1MB.</p>
              </div>
            </div>
          </div>

          {/* Color Scheme */}
          <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-5">
            <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
              <Palette className="w-4 h-4" /> Color Scheme
            </h3>
            <div className="space-y-4">
              {[
                { label: "Primary", value: primaryColor, setter: setPrimaryColor },
                { label: "Secondary", value: secondaryColor, setter: setSecondaryColor },
                { label: "Accent", value: accentColor, setter: setAccentColor },
              ].map(({ label, value, setter }) => (
                <div key={label} className="flex items-center gap-3">
                  <label className="text-sm text-gray-600 dark:text-gray-400 w-20">{label}</label>
                  <input type="color" value={value} onChange={(e) => setter(e.target.value)} className="w-10 h-10 rounded cursor-pointer border border-gray-300 dark:border-gray-700" />
                  <input type="text" value={value} onChange={(e) => setter(e.target.value)} className="flex-1 px-3 py-1.5 text-sm font-mono border border-gray-300 dark:border-gray-700 rounded-lg bg-transparent text-gray-900 dark:text-white" />
                </div>
              ))}
            </div>
            {/* Presets */}
            <div className="mt-4">
              <p className="text-xs text-gray-400 mb-2">Presets</p>
              <div className="flex flex-wrap gap-2">
                {PRESET_COLORS.map((preset) => (
                  <button key={preset.name} onClick={() => { setPrimaryColor(preset.primary); setSecondaryColor(preset.secondary); setAccentColor(preset.accent); }}
                    className="flex items-center gap-1.5 px-2 py-1 rounded-lg border border-gray-200 dark:border-gray-800 hover:border-gray-400 transition">
                    <div className="flex -space-x-1">
                      <div className="w-4 h-4 rounded-full border border-white" style={{ backgroundColor: preset.primary }} />
                      <div className="w-4 h-4 rounded-full border border-white" style={{ backgroundColor: preset.secondary }} />
                      <div className="w-4 h-4 rounded-full border border-white" style={{ backgroundColor: preset.accent }} />
                    </div>
                    <span className="text-xs text-gray-600 dark:text-gray-400">{preset.name}</span>
                  </button>
                ))}
              </div>
            </div>
            {/* Swatch preview */}
            <div className="mt-4 flex rounded-lg overflow-hidden h-8">
              <div className="flex-1" style={{ backgroundColor: primaryColor }} />
              <div className="flex-1" style={{ backgroundColor: secondaryColor }} />
              <div className="flex-1" style={{ backgroundColor: accentColor }} />
            </div>
          </div>

          {/* Custom CSS */}
          <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-5">
            <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
              <Code className="w-4 h-4" /> Custom CSS Injection
            </h3>
            <textarea value={customCss} onChange={(e) => setCustomCss(e.target.value)}
              placeholder="/* Custom CSS */&#10;.login-form { border-radius: 12px; }"
              className="w-full h-40 px-3 py-2 text-sm font-mono border border-gray-300 dark:border-gray-700 rounded-lg bg-gray-50 dark:bg-gray-800 text-gray-900 dark:text-white resize-none" />
            <div className="flex items-center justify-between mt-2">
              <span className="text-xs text-gray-400">{customCss.length} / 10240 bytes</span>
              <button onClick={() => setCustomCss("")} className="text-xs text-gray-500 hover:text-gray-700 dark:hover:text-gray-300">Reset CSS</button>
            </div>
          </div>
        </div>

        {/* Right: Live Previews */}
        <div className="space-y-6">
          <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-5">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-semibold text-gray-900 dark:text-white flex items-center gap-2">
                <Eye className="w-4 h-4" /> Live Preview
              </h3>
              <div className="flex gap-1">
                <button onClick={() => setActiveTab("login")} className={`flex items-center gap-1 px-2.5 py-1 text-xs rounded-md transition ${activeTab === "login" ? "text-white" : "text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-800"}`} style={activeTab === "login" ? { backgroundColor: primaryColor } : {}}>
                  Login Page
                </button>
                <button onClick={() => setActiveTab("email")} className={`flex items-center gap-1 px-2.5 py-1 text-xs rounded-md transition ${activeTab === "email" ? "text-white" : "text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-800"}`} style={activeTab === "email" ? { backgroundColor: primaryColor } : {}}>
                  <Mail className="w-3 h-3" /> Email
                </button>
              </div>
            </div>

            {activeTab === "login" ? (
              <div className="rounded-xl border border-gray-200 dark:border-gray-800 p-6 bg-gray-50 dark:bg-gray-950">
                <div className="max-w-xs mx-auto space-y-4">
                  <div className="flex justify-center">
                    {logo ? (
                      <img src={logo} alt="Logo" className="h-12 object-contain" />
                    ) : (
                      <div className="h-12 w-12 rounded-xl flex items-center justify-center text-white font-bold text-xl" style={{ backgroundColor: primaryColor }}>
                        GG
                      </div>
                    )}
                  </div>
                  <h2 className="text-center text-lg font-semibold text-gray-900 dark:text-white">Sign In</h2>
                  <input type="email" placeholder="Email address" disabled className="w-full px-3 py-2 text-sm border rounded-lg bg-white dark:bg-gray-900 text-gray-500" style={{ borderColor: `${primaryColor}40` }} />
                  <input type="password" placeholder="Password" disabled className="w-full px-3 py-2 text-sm border rounded-lg bg-white dark:bg-gray-900 text-gray-500" style={{ borderColor: `${primaryColor}40` }} />
                  <button disabled className="w-full py-2 text-sm font-medium text-white rounded-lg" style={{ backgroundColor: primaryColor }}>
                    Sign In
                  </button>
                  <div className="text-center">
                    <a href="#" className="text-xs" style={{ color: accentColor }}>Forgot password?</a>
                  </div>
                </div>
              </div>
            ) : (
              <div className="rounded-xl border border-gray-200 dark:border-gray-800 overflow-hidden">
                <div className="h-16 flex items-center px-4" style={{ backgroundColor: primaryColor }}>
                  {logo ? <img src={logo} alt="Logo" className="h-8" /> : <span className="text-white font-bold">Your Brand</span>}
                </div>
                <div className="p-4 bg-white dark:bg-gray-900">
                  <h4 className="text-sm font-semibold text-gray-900 dark:text-white">Welcome to GGID</h4>
                  <p className="text-xs text-gray-500 mt-1">Your verification code is:</p>
                  <div className="my-2 p-2 rounded text-center text-2xl font-bold tracking-widest text-white" style={{ backgroundColor: accentColor }}>
                    123456
                  </div>
                  <p className="text-xs text-gray-400">This code expires in 10 minutes.</p>
                </div>
                <div className="px-4 py-2 text-xs text-gray-400" style={{ backgroundColor: secondaryColor, color: "#fff8" }}>
                  &copy; 2024 Your Brand. All rights reserved.
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

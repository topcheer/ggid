"use client";

import { useBrandingConfig } from "@ggid/sdk-react";
import { Palette, Upload } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function BrandingConfigPage() {
  const t = useTranslations();

  const { data, loading, error, refresh } = useBrandingConfig();
  if (loading) return (
    <div className="p-8 flex flex-col items-center justify-center">
      <div className="inline-block w-6 h-6 border-2 border-current border-t-transparent rounded-full animate-spin text-blue-600 mb-2" />
      <div className="text-gray-400">Loading...</div>
    </div>
  );
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div><h1 className="text-2xl font-bold">Tenant Branding</h1><p className="text-sm text-gray-400 mt-1">Customize look and feel</p></div>
        <button onClick={refresh} aria-label="Save branding configuration" className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Save</button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="space-y-6">
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-sm font-semibold mb-3 flex items-center gap-2"><Upload className="w-4 h-4 text-blue-400" /> Logo</h2>
            <div className="border-2 border-dashed border-gray-700 rounded-lg p-6 text-center">
              <img src={data?.logo_url} alt="Tenant logo preview" className="h-12 mx-auto mb-2" />
              <p className="text-xs text-gray-400">Click or drag to upload</p>
            </div>
          </div>
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-sm font-semibold mb-3 flex items-center gap-2"><Palette className="w-4 h-4 text-purple-400" /> Colors</h2>
            <div className="space-y-3">
              <div><label className="text-xs text-gray-400">Primary</label><div className="flex items-center gap-2"><input type="color" defaultValue={data?.primary_color} aria-label="Primary color" className="w-10 h-8 rounded" /><input type="text" defaultValue={data?.primary_color} aria-label="Primary color hex" className="flex-1 px-2 py-1 bg-gray-800 rounded text-xs font-mono" /></div></div>
              <div><label className="text-xs text-gray-400">Secondary</label><div className="flex items-center gap-2"><input type="color" defaultValue={data?.secondary_color} aria-label="Secondary color" className="w-10 h-8 rounded" /><input type="text" defaultValue={data?.secondary_color} aria-label="Secondary color hex" className="flex-1 px-2 py-1 bg-gray-800 rounded text-xs font-mono" /></div></div>
            </div>
          </div>
          <div className="bg-gray-900 rounded-xl p-6"><h2 className="text-sm font-semibold mb-3">Custom CSS</h2><textarea defaultValue={data?.custom_css} aria-label="Custom CSS" rows={6} className="w-full px-3 py-2 bg-gray-800 rounded-lg text-xs font-mono" /></div>
          <div className="bg-gray-900 rounded-xl p-6 space-y-2">
            <h2 className="text-sm font-semibold mb-3">Settings</h2>
            <div><label className="text-xs text-gray-400">Theme</label><select defaultValue={data?.theme} aria-label="Theme" className="w-full mt-1 px-3 py-2 bg-gray-800 rounded-lg text-sm"><option>dark</option><option>light</option><option>auto</option></select></div>
            <div><label className="text-xs text-gray-400">Custom Domain</label><input type="text" defaultValue={data?.custom_domain} aria-label="Custom domain" className="w-full mt-1 px-3 py-2 bg-gray-800 rounded-lg text-sm" /></div>
          </div>
        </div>
        <div className="space-y-6">
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-sm font-semibold mb-3">Login Page Preview</h2>
            <div className="rounded-lg p-8 text-center" style={{ background: data?.primary_color || "#1e40af" }}>
              <img src={data?.logo_url} alt="Branded login logo" className="h-8 mx-auto mb-3" />
              <div className="bg-white rounded-lg p-4 max-w-xs mx-auto">
                <p className="text-sm text-gray-800 font-medium">Sign in to your account</p>
                <div className="mt-2 space-y-2">
                  <input type="text" placeholder="Email" aria-label="Preview email" className="w-full px-2 py-1.5 border border-gray-300 rounded text-xs" />
                  <input autoComplete="current-password" type="password" placeholder="Password" aria-label="Preview password" className="w-full px-2 py-1.5 border border-gray-300 rounded text-xs" />
                  <button aria-label="Preview sign in" className="w-full py-1.5 text-white text-xs rounded" style={{ background: data?.primary_color || "#1e40af" }}>Sign In</button>
                </div>
              </div>
            </div>
          </div>
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-sm font-semibold mb-3">Email Preview</h2>
            <div className="bg-white rounded-lg p-4 text-black">
              <div className="text-center mb-2"><img src={data?.logo_url} alt="Branded email logo" className="h-6 mx-auto" /></div>
              <p className="text-xs text-gray-600">Hi John, welcome to GGID. Click the button below to verify your email.</p>
              <button aria-label="Preview verify email" className="mt-2 px-3 py-1.5 text-white text-xs rounded" style={{ background: data?.primary_color || "#1e40af" }}>Verify Email</button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

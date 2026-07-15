"use client";

import { useState, useEffect, useCallback } from "react";
import { Globe, Save, Search, Shield, Zap } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface GeofenceRule {
  allowed_countries: string[];
  denied_regions: string[];
  action: "allow" | "deny" | "mfa";
  enabled: boolean;
}

const countries = ["US", "CN", "GB", "DE", "FR", "JP", "KR", "AU", "CA", "IN", "BR", "SG", "NL", "SE", "CH"];
const regions = ["EU", "NA", "APAC", "LATAM", "MEA", "EEU"];

export default function GeofencingPage() {
  const t = useTranslations();

  const [rule, setRule] = useState<GeofenceRule>({ allowed_countries: [], denied_regions: [], action: "mfa", enabled: true });
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [testIp, setTestIp] = useState("");
  const [testResult, setTestResult] = useState<{ ip: string; country: string; action: string } | null>(null);

  const fetchRule = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/auth/geofencing", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const data = await res.json(); if (data) setRule(data); }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchRule(); }, [fetchRule]);

  const save = async () => {
    setSaving(true);
    try { await fetch("/api/v1/auth/geofencing", { method: "POST", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify(rule) }); } catch { /* noop */ } finally { setSaving(false); }
  };

  const testIpCheck = async () => {
    if (!testIp) return;
    try { const res = await fetch(`/api/v1/auth/geofencing/test?ip=${encodeURIComponent(testIp)}`, { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); if (res.ok) setTestResult(await res.json()); } catch { /* noop */ }
  };

  const toggleCountry = (code: string) => {
    setRule((prev) => ({ ...prev, allowed_countries: prev.allowed_countries.includes(code) ? prev.allowed_countries.filter((c) => c !== code) : [...prev.allowed_countries, code] }));
  };
  const toggleRegion = (r: string) => {
    setRule((prev) => ({ ...prev, denied_regions: prev.denied_regions.includes(r) ? prev.denied_regions.filter((x) => x !== r) : [...prev.denied_regions, r] }));
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold flex items-center gap-2"><Globe className="w-6 h-6 text-blue-500" /> {t("geofencing.title")}</h1><p className="text-sm text-gray-500 mt-1">Configure geo-based access rules with country/region controls.</p></div>
        <button onClick={save} disabled={saving} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50 flex items-center gap-2"><Save className="w-4 h-4" /> {saving ? "Saving..." : "Save Rules"}</button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Allowed countries */}
        <div className="rounded-lg border dark:border-gray-800 p-4">
          <h3 className="font-semibold mb-3">Allowed Countries ({rule.allowed_countries.length})</h3>
          <div className="grid grid-cols-4 gap-2">
            {countries.map((c) => (
              <label key={c} className="flex items-center gap-1 text-xs cursor-pointer">
                <input type="checkbox" checked={rule.allowed_countries.includes(c)} onChange={() => toggleCountry(c)} className="rounded" /><span className="font-mono">{c}</span>
              </label>
            ))}
          </div>
        </div>

        {/* Denied regions */}
        <div className="rounded-lg border dark:border-gray-800 p-4">
          <h3 className="font-semibold mb-3">Denied Regions ({rule.denied_regions.length})</h3>
          <div className="grid grid-cols-3 gap-2">
            {regions.map((r) => (
              <label key={r} className="flex items-center gap-1 text-xs cursor-pointer">
                <input type="checkbox" checked={rule.denied_regions.includes(r)} onChange={() => toggleRegion(r)} className="rounded" /><span className="font-mono">{r}</span>
              </label>
            ))}
          </div>
        </div>
      </div>

      {/* Action + enabled */}
      <div className="rounded-lg border dark:border-gray-800 p-4 flex items-center gap-4">
        <div className="flex items-center gap-2"><Shield className="w-4 h-4 text-gray-400" /><label className="text-sm font-medium">Action for violations:</label><select value={rule.action} onChange={(e) => setRule({ ...rule, action: e.target.value as GeofenceRule["action"] })} className="px-3 py-1.5 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm"><option value="allow">Allow</option><option value="deny">Deny</option><option value="mfa">Require MFA</option></select></div>
        <label className="flex items-center gap-2 text-sm cursor-pointer ml-auto"><input type="checkbox" checked={rule.enabled} onChange={(e) => setRule({ ...rule, enabled: e.target.checked })} className="rounded" /><span>{rule.enabled ? "Enabled" : "Disabled"}</span></label>
      </div>

      {/* Test IP */}
      <div className="rounded-lg border dark:border-gray-800 p-4 space-y-3">
        <h3 className="font-semibold flex items-center gap-2"><Search className="w-4 h-4" /> Test IP Address</h3>
        <div className="flex items-center gap-2"><input type="text" value={testIp} onChange={(e) => setTestIp(e.target.value)} placeholder="8.8.8.8" className="flex-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm font-mono" /><button onClick={testIpCheck} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700">Test</button></div>
        {testResult && <div className="flex items-center gap-3 text-sm"><Zap className="w-4 h-4 text-blue-500" /><span>IP: <span className="font-mono">{testResult.ip}</span> · Country: {testResult.country} · Action: <span className={`px-2 py-0.5 rounded text-xs font-medium ${testResult.action === "deny" ? "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400" : testResult.action === "mfa" ? "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400" : "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"}`}>{testResult.action}</span></span></div>}
      </div>
    </div>
  );
}

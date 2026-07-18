"use client";

import { useState, useEffect, useCallback } from "react";
import { Search, ShieldAlert, Key, Smartphone, Monitor, Link2, Lightbulb, Gauge } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface ExposureData {
  user_id: string;
  username: string;
  exposure_score: number;
  active_tokens: number;
  active_sessions: number;
  linked_providers: { provider: string; connected_at: string }[];
  api_keys: { id: string; name: string; last_used: string; scopes: string[] }[];
  recommendations: string[];
}

const providerIcons: Record<string, typeof Key> = {
  google: Smartphone,
  github: Link2,
  microsoft: Monitor,
  saml: Key,
  ldap: Key,
};

export default function CredentialExposurePage() {
  const t = useTranslations();

  const [search, setSearch] = useState("");
  const [data, setData] = useState<ExposureData | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async (user: string) => {
    if (!user) return;
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/auth/credential-exposure?user=${encodeURIComponent(user)}`, { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => {
    if (!search) return;
    fetchData(search);
  }, [search, fetchData]);

  const scoreColor = data ? (data.exposure_score >= 70 ? "#ef4444" : data.exposure_score >= 40 ? "#f59e0b" : "#10b981") : "#3b82f6";
  const scoreLabel = data ? (data.exposure_score >= 70 ? "High Risk" : data.exposure_score >= 40 ? "Moderate" : "Low Risk") : "";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><ShieldAlert className="w-6 h-6 text-orange-500" /> {t("credentialExposure.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Assess user credential exposure across tokens, sessions, and linked providers.</p>
      </div>

      <div className="relative max-w-md">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
        <input aria-label="Search by username or user ID..." type="text" placeholder="Search by username or user ID..." value={search} onChange={(e) => setSearch(e.target.value)} className="w-full pl-9 pr-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" />
      </div>

      {data && (
        <div className="space-y-4">
          {/* Exposure score + stats */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <div className="rounded-lg border dark:border-gray-800 p-4 flex flex-col items-center justify-center">
              <span className="text-sm text-gray-500 mb-2 flex items-center gap-1"><Gauge className="w-4 h-4" /> Exposure Score</span>
              <div className="relative w-24 h-24">
                <svg viewBox="0 0 64 64" className="w-full h-full">
                  <circle cx={32} cy={32} r={28} fill="none" stroke="currentColor" strokeWidth={6} className="text-gray-200 dark:text-gray-800" />
                  <circle cx={32} cy={32} r={28} fill="none" stroke={scoreColor} strokeWidth={6} strokeDasharray={`${data.exposure_score * 1.76} 176`} strokeLinecap="round" transform="rotate(-90 32 32)" />
                </svg>
                <div className="absolute inset-0 flex flex-col items-center justify-center">
                  <span className="text-2xl font-bold" style={{ color: scoreColor }}>{data.exposure_score}</span>
                  <span className="text-[10px] text-gray-400">/100</span>
                </div>
              </div>
              <span className="text-xs font-medium mt-1" style={{ color: scoreColor }}>{scoreLabel}</span>
            </div>
            <div className="rounded-lg border p-4 dark:border-gray-800">
              <div className="flex items-center justify-between"><span className="text-sm text-gray-500">Active Tokens</span><Key className="w-5 h-5 text-gray-400" /></div>
              <p className="text-2xl font-bold mt-1">{data.active_tokens}</p>
            </div>
            <div className="rounded-lg border p-4 dark:border-gray-800">
              <div className="flex items-center justify-between"><span className="text-sm text-gray-500">Active Sessions</span><Smartphone className="w-5 h-5 text-gray-400" /></div>
              <p className="text-2xl font-bold mt-1">{data.active_sessions}</p>
            </div>
            <div className="rounded-lg border p-4 dark:border-gray-800">
              <div className="flex items-center justify-between"><span className="text-sm text-gray-500">API Keys</span><Key className="w-5 h-5 text-gray-400" /></div>
              <p className="text-2xl font-bold mt-1">{data.api_keys.length}</p>
            </div>
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            {/* Linked providers */}
            <div className="rounded-lg border dark:border-gray-800">
              <div className="px-4 py-3 border-b dark:border-gray-800"><h3 className="font-semibold flex items-center gap-2"><Link2 className="w-4 h-4" /> Linked Providers ({data.linked_providers.length})</h3></div>
              <div className="divide-y dark:divide-gray-800">
                {data.linked_providers.map((p: any, i: number) => {
                  const Icon = providerIcons[p.provider] || Key;
                  return (
                    <div key={i} className="px-4 py-2 flex items-center justify-between text-sm">
                      <div className="flex items-center gap-2"><Icon className="w-4 h-4 text-gray-400" /><span className="font-medium">{p.provider}</span></div>
                      <span className="text-xs text-gray-400">{p.connected_at}</span>
                    </div>
                  );
                })}
                {data.linked_providers.length === 0 && <p className="px-4 py-4 text-sm text-gray-500">No linked providers.</p>}
              </div>
            </div>

            {/* API keys */}
            <div className="rounded-lg border dark:border-gray-800">
              <div className="px-4 py-3 border-b dark:border-gray-800"><h3 className="font-semibold flex items-center gap-2"><Key className="w-4 h-4" /> API Keys ({data.api_keys.length})</h3></div>
              <div className="divide-y dark:divide-gray-800 max-h-48 overflow-y-auto">
                {data.api_keys.map((k: any) => (
                  <div key={k.id} className="px-4 py-2 text-sm">
                    <div className="flex items-center justify-between">
                      <span className="font-medium">{k.name}</span>
                      <span className="text-xs text-gray-400">Last used: {k.last_used}</span>
                    </div>
                    <div className="flex flex-wrap gap-1 mt-1">{k.scopes.map((s: any, i: number) => <span key={i} className="px-1.5 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800 font-mono">{s}</span>)}</div>
                  </div>
                ))}
                {data.api_keys.length === 0 && <p className="px-4 py-4 text-sm text-gray-500">No API keys.</p>}
              </div>
            </div>
          </div>

          {/* Recommendations */}
          {data.recommendations.length > 0 && (
            <div className="rounded-lg border dark:border-gray-800 p-4 bg-blue-50 dark:bg-blue-900/20">
              <h3 className="font-semibold mb-2 flex items-center gap-2"><Lightbulb className="w-4 h-4 text-blue-500" /> Recommendations</h3>
              <ul className="space-y-1">
                {data.recommendations.map((rec: any, i: number) => <li key={i} className="text-sm text-gray-600 dark:text-gray-400 flex items-start gap-2"><span className="text-blue-400 mt-0.5">•</span> {rec}</li>)}
              </ul>
            </div>
          )}
        </div>
      )}

      {!data && !loading && search && <p className="text-sm text-gray-500">No exposure data found.</p>}
      {!data && !search && <p className="text-sm text-gray-500 text-center py-8">Search for a user to view their credential exposure.</p>}
    </div>
  );
}

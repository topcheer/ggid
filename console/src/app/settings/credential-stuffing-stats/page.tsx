"use client";

import { useState, useEffect, useCallback } from "react";
import { ShieldX, Ban, Bot } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface Stats {
  total_attempts: number;
  blocked_by_rate_limit: number;
  blocked_by_captcha: number;
  unique_targeted_accounts: number;
  top_source_ips: { ip: string; attempts: number }[];
  top_user_agents: { ua: string; attempts: number }[];
  attack_pattern: "distributed" | "burst" | "credential_list";
}

const patternColors: Record<string, string> = {
  distributed: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
  burst: "bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400",
  credential_list: "bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-400",
};

export default function CredentialStuffingStatsPage() {
  const t = useTranslations();

  const [data, setData] = useState<Stats | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/auth/credential-stuffing-stats", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><ShieldX className="w-6 h-6 text-red-500" /> {t("credentialStuffingStats.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Monitor credential stuffing attack patterns and mitigation effectiveness.</p>
      </div>

      {data && (
        <>
          {data.attack_pattern && <div className="rounded-lg border border-red-300 dark:border-red-800 bg-red-50 dark:bg-red-900/20 p-3 flex items-center gap-2"><Bot className="w-5 h-5 text-red-500" /><span className="font-semibold text-red-700 dark:text-red-400">Attack Pattern Detected: </span><span className={"px-2 py-0.5 rounded text-xs font-medium " + patternColors[data.attack_pattern]}>{data.attack_pattern.replace("_", " ")}</span></div>}

          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Total Attempts</span><p className="text-xl font-bold text-red-600 mt-1">{data.total_attempts.toLocaleString()}</p></div>
            <div className="rounded-lg border p-4 dark:border-gray-800 flex items-center gap-3"><Ban className="w-8 h-8 text-orange-500" /><div><span className="text-sm text-gray-500">Rate Limited</span><p className="text-xl font-bold mt-1">{data.blocked_by_rate_limit.toLocaleString()}</p></div></div>
            <div className="rounded-lg border p-4 dark:border-gray-800 flex items-center gap-3"><Bot className="w-8 h-8 text-purple-500" /><div><span className="text-sm text-gray-500">CAPTCHA Blocked</span><p className="text-xl font-bold mt-1">{data.blocked_by_captcha.toLocaleString()}</p></div></div>
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Targeted Accounts</span><p className="text-xl font-bold mt-1">{data.unique_targeted_accounts.toLocaleString()}</p></div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">Top Source IPs</h3><div className="space-y-1">{data.top_source_ips.map((ip, i) => (<div key={i} className="flex items-center gap-2"><span className="text-xs font-mono text-gray-500 w-32">{ip.ip}</span><div className="flex-1 bg-gray-100 dark:bg-gray-800 rounded-full h-4 overflow-hidden"><div className="h-full bg-red-500 rounded-full" style={{ width: Math.min((ip.attempts / data.top_source_ips[0].attempts) * 100, 100) + "%" }} /></div><span className="text-xs font-bold w-12 text-right">{ip.attempts}</span></div>))}</div></div>
            <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">Top User Agents</h3><div className="space-y-1">{data.top_user_agents.map((ua, i) => (<div key={i} className="flex items-center gap-2"><span className="text-xs text-gray-500 flex-1 truncate" title={ua.ua}>{ua.ua.length > 50 ? ua.ua.substring(0, 50) + "..." : ua.ua}</span><span className="text-xs font-bold text-red-600">{ua.attempts}</span></div>))}</div></div>
          </div>
        </>
      )}
      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
    </div>
  );
}

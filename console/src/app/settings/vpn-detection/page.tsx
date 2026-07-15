"use client";

import { useState } from "react";
import { ShieldAlert, Search, Globe, Zap, AlertTriangle, CheckCircle2 } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface VPNResult {
  ip: string;
  is_vpn: boolean;
  is_proxy: boolean;
  is_tor: boolean;
  provider: string;
  type: string;
  country: string;
  country_code: string;
  risk_level: "low" | "medium" | "high" | "critical";
  recommendation: string;
}

const riskColors: Record<string, string> = {
  low: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-400",
  medium: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  high: "bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400",
  critical: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
};

export default function VPNDetectionPage() {
  const t = useTranslations();
  const [ipInput, setIpInput] = useState("");
  const [results, setResults] = useState<VPNResult[]>([]);
  const [loading, setLoading] = useState(false);

  const checkIP = async () => {
    const ips = ipInput.split(/[\n,]/).map((s) => s.trim()).filter(Boolean);
    if (ips.length === 0) return;
    setLoading(true);
    try {
      const res = await fetch("/api/v1/auth/vpn-check", { method: "POST", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify({ ips }) });
      if (res.ok) { const data = await res.json(); setResults(data.results || data || []); }
    } catch { /* noop */ }
    finally { setLoading(false); }
  };

  const flagged = results.filter((r) => r.is_vpn || r.is_proxy || r.is_tor);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><ShieldAlert className="w-6 h-6 text-blue-500" /> VPN Detection</h1>
        <p className="text-sm text-gray-500 mt-1">Check IP addresses for VPN, proxy, and Tor exit node usage.</p>
      </div>

      {/* Batch input */}
      <div className="space-y-2">
        <textarea value={ipInput} onChange={(e) => setIpInput(e.target.value)} placeholder="Enter IPs (one per line or comma-separated)..." rows={3} className="w-full px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono" />
        <button onClick={checkIP} disabled={loading} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50 flex items-center gap-2"><Search className="w-4 h-4" /> {loading ? "Checking..." : "Check IPs"}</button>
      </div>

      {/* Stats */}
      {results.length > 0 && (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Checked</span><p className="text-2xl font-bold mt-1">{results.length}</p></div>
          <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">VPN/Proxy</span><p className="text-2xl font-bold mt-1 text-orange-600">{flagged.length}</p></div>
          <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Clean</span><p className="text-2xl font-bold mt-1 text-green-600">{results.length - flagged.length}</p></div>
          <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Countries</span><p className="text-2xl font-bold mt-1">{new Set(results.map((r) => r.country_code)).size}</p></div>
        </div>
      )}

      {/* Results table */}
      {results.length > 0 && (
        <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 dark:bg-gray-900/50">
              <tr>
                <th className="px-4 py-3 text-left font-medium">IP Address</th>
                <th className="px-4 py-3 text-left font-medium">Type</th>
                <th className="px-4 py-3 text-left font-medium">Provider</th>
                <th className="px-4 py-3 text-left font-medium">Country</th>
                <th className="px-4 py-3 text-left font-medium">Risk</th>
                <th className="px-4 py-3 text-left font-medium">Recommendation</th>
              </tr>
            </thead>
            <tbody className="divide-y dark:divide-gray-800">
              {results.map((r, i) => (
                <tr key={i} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                  <td className="px-4 py-3 font-mono text-xs">{r.ip}</td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-1">
                      {r.is_vpn && <span className="px-1.5 py-0.5 rounded text-xs bg-orange-100 dark:bg-orange-900/30 dark:text-orange-400">VPN</span>}
                      {r.is_proxy && <span className="px-1.5 py-0.5 rounded text-xs bg-yellow-100 dark:bg-yellow-900/30 dark:text-yellow-400">Proxy</span>}
                      {r.is_tor && <span className="px-1.5 py-0.5 rounded text-xs bg-red-100 dark:bg-red-900/30 dark:text-red-400">Tor</span>}
                      {!r.is_vpn && !r.is_proxy && !r.is_tor && <span className="text-xs text-green-600 flex items-center gap-1"><CheckCircle2 className="w-3 h-3" /> Clean</span>}
                    </div>
                  </td>
                  <td className="px-4 py-3 text-xs">{r.provider || "-"}</td>
                  <td className="px-4 py-3 text-xs">{r.country}</td>
                  <td className="px-4 py-3"><span className={`px-2 py-0.5 rounded text-xs ${riskColors[r.risk_level]}`}>{r.risk_level}</span></td>
                  <td className="px-4 py-3 text-xs text-gray-500">{r.recommendation}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {results.length === 0 && !loading && <p className="text-sm text-gray-500 text-center py-8">Enter IP addresses to check for VPN/proxy/Tor usage.</p>}
    </div>
  );
}

"use client";

import { useState, useEffect, useCallback } from "react";
import { Shield, Gauge, AlertTriangle, Users } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface PasswordStrengthData {
  total_users: number;
  distribution: { weak: number; fair: number; good: number; strong: number };
  policy_compliance_pct: number;
  avg_entropy_bits: number;
  min_entropy: number;
  max_entropy: number;
  weak_passwords: { user_id: string; username: string; entropy_bits: number; last_changed: string }[];
}

const strengthColors: Record<string, { bg: string; text: string; hex: string }> = {
  weak: { bg: "bg-red-100 dark:bg-red-900/30", text: "text-red-800 dark:text-red-400", hex: "#ef4444" },
  fair: { bg: "bg-yellow-100 dark:bg-yellow-900/30", text: "text-yellow-800 dark:text-yellow-400", hex: "#f59e0b" },
  good: { bg: "bg-blue-100 dark:bg-blue-900/30", text: "text-blue-800 dark:text-blue-400", hex: "#3b82f6" },
  strong: { bg: "bg-green-100 dark:bg-green-900/30", text: "text-green-800 dark:text-green-400", hex: "#10b981" },
};

export default function PasswordStrengthPage() {
  const t = useTranslations();

  const [data, setData] = useState<PasswordStrengthData | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/auth/password-strength", { headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  // Donut chart
  const segments = data ? Object.entries(data.distribution).map(([key, count]) => ({ key, count, hex: strengthColors[key].hex })) : [];
  const total = segments.reduce((s, seg) => s + seg.count, 0) || 1;
  let cumulativeDeg = 0;
  const donutSegments = segments.map((seg) => {
    const startDeg = cumulativeDeg;
    const sweepDeg = (seg.count / total) * 360;
    cumulativeDeg += sweepDeg;
    return { ...seg, startDeg, sweepDeg };
  });

  // Entropy gauge
  const entropyPct = data ? Math.min(100, (data.avg_entropy_bits / 128) * 100) : 0;
  const entropyColor = data ? (data.avg_entropy_bits >= 80 ? "#10b981" : data.avg_entropy_bits >= 50 ? "#f59e0b" : "#ef4444") : "#3b82f6";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Shield className="w-6 h-6 text-blue-500" /> {t("passwordStrength.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Password security distribution and policy compliance overview.</p>
      </div>

      {data && (
        <>
          {/* Top row: donut + gauges */}
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
            {/* Distribution donut */}
            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="font-semibold mb-3">Strength Distribution</h3>
              <div className="flex items-center gap-4">
                <div className="relative w-32 h-32">
                  <svg viewBox="0 0 100 100" className="w-full h-full -rotate-90">
                    {donutSegments.map((seg, i) => {
                      if (seg.count === 0) return null;
                      const r = 40, cx = 50, cy = 50;
                      const startRad = (seg.startDeg - 90) * Math.PI / 180;
                      const endDeg = seg.startDeg + seg.sweepDeg;
                      const endRad = (endDeg - 90) * Math.PI / 180;
                      const x1 = cx + r * Math.cos(startRad), y1 = cy + r * Math.sin(startRad);
                      const x2 = cx + r * Math.cos(endRad), y2 = cy + r * Math.sin(endRad);
                      const largeArc = seg.sweepDeg > 180 ? 1 : 0;
                      return <path key={i} d={`M${cx},${cy} L${x1},${y1} A${r},${r} 0 ${largeArc} 1 ${x2},${y2} Z`} fill={seg.hex} stroke="white" strokeWidth={0.5} />;
                    })}
                    <circle cx={50} cy={50} r={26} fill="white" className="dark:fill-gray-900" />
                  </svg>
                  <div className="absolute inset-0 flex flex-col items-center justify-center">
                    <span className="text-2xl font-bold">{data.total_users}</span>
                    <span className="text-xs text-gray-400">users</span>
                  </div>
                </div>
                <div className="flex-1 space-y-1">
                  {segments.map((seg) => (
                    <div key={seg.key} className="flex items-center gap-2 text-sm">
                      <span className="w-3 h-3 rounded" style={{ backgroundColor: seg.hex }} />
                      <span className="flex-1 capitalize">{seg.key}</span>
                      <span className="text-gray-400">{seg.count} ({((seg.count / total) * 100).toFixed(0)}%)</span>
                    </div>
                  ))}
                </div>
              </div>
            </div>

            {/* Policy compliance gauge */}
            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="font-semibold mb-3 flex items-center gap-2"><Shield className="w-4 h-4" /> Policy Compliance</h3>
              <div className="flex items-center justify-center">
                <div className="relative w-32 h-32">
                  <svg viewBox="0 0 64 64" className="w-full h-full">
                    <circle cx={32} cy={32} r={28} fill="none" stroke="currentColor" strokeWidth={6} className="text-gray-200 dark:text-gray-800" />
                    <circle cx={32} cy={32} r={28} fill="none" stroke={data.policy_compliance_pct >= 80 ? "#10b981" : data.policy_compliance_pct >= 60 ? "#f59e0b" : "#ef4444"} strokeWidth={6} strokeDasharray={`${(data.policy_compliance_pct / 100) * 176} 176`} strokeLinecap="round" transform="rotate(-90 32 32)" />
                  </svg>
                  <div className="absolute inset-0 flex flex-col items-center justify-center">
                    <span className={`text-2xl font-bold ${data.policy_compliance_pct >= 80 ? "text-green-600" : data.policy_compliance_pct >= 60 ? "text-yellow-600" : "text-red-600"}`}>{data.policy_compliance_pct.toFixed(0)}%</span>
                    <span className="text-xs text-gray-400">compliant</span>
                  </div>
                </div>
              </div>
            </div>

            {/* Avg entropy gauge */}
            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="font-semibold mb-3 flex items-center gap-2"><Gauge className="w-4 h-4" /> Avg Entropy</h3>
              <div className="flex items-center justify-center">
                <div className="relative w-32 h-32">
                  <svg viewBox="0 0 64 64" className="w-full h-full">
                    <circle cx={32} cy={32} r={28} fill="none" stroke="currentColor" strokeWidth={6} className="text-gray-200 dark:text-gray-800" />
                    <circle cx={32} cy={32} r={28} fill="none" stroke={entropyColor} strokeWidth={6} strokeDasharray={`${entropyPct * 1.76} 176`} strokeLinecap="round" transform="rotate(-90 32 32)" />
                  </svg>
                  <div className="absolute inset-0 flex flex-col items-center justify-center">
                    <span className="text-2xl font-bold" style={{ color: entropyColor }}>{data.avg_entropy_bits.toFixed(0)}</span>
                    <span className="text-xs text-gray-400">bits</span>
                  </div>
                </div>
              </div>
              <div className="flex justify-between mt-2 text-xs text-gray-400">
                <span>Min: {data.min_entropy}</span>
                <span>Max: {data.max_entropy}</span>
              </div>
            </div>
          </div>

          {/* Weak passwords list */}
          <div className="rounded-lg border dark:border-gray-800">
            <div className="px-4 py-3 border-b dark:border-gray-800">
              <h3 className="font-semibold flex items-center gap-2"><AlertTriangle className="w-4 h-4 text-red-500" /> Weak Passwords ({data.weak_passwords.length})</h3>
            </div>
            <div className="divide-y dark:divide-gray-800 max-h-64 overflow-y-auto">
              {data.weak_passwords.map((u, i) => (
                <div key={i} className="px-4 py-2 flex items-center justify-between text-sm">
                  <div className="flex items-center gap-2">
                    <span className="w-2 h-2 rounded-full bg-red-500" />
                    <Users className="w-3 h-3 text-gray-400" />
                    <span className="font-medium">{u.username}</span>
                  </div>
                  <div className="flex items-center gap-3 text-xs text-gray-400">
                    <span className="px-2 py-0.5 rounded bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400">{u.entropy_bits} bits</span>
                    <span>Changed: {u.last_changed}</span>
                  </div>
                </div>
              ))}
              {data.weak_passwords.length === 0 && <p className="px-4 py-4 text-sm text-gray-500">No weak passwords detected.</p>}
            </div>
          </div>
        </>
      )}

      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
    </div>
  );
}

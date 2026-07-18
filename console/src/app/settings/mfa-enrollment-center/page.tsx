"use client";
import { useState, useEffect } from "react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface MfaFactor {
  type: "TOTP" | "WebAuthn" | "Push" | "SMS";
  name: string;
  enrolled_at: string;
  last_used: string;
  status: "active" | "disabled";
}

interface ComplianceStats {
  total_users: number;
  enrolled: number;
  pending: number;
  exemptions: number;
}

export default function MfaEnrollmentCenterPage() {
  const t = useTranslations();

  const [factors, setFactors] = useState<MfaFactor[]>([]);
  const [compliance, setCompliance] = useState<ComplianceStats>({ total_users: 0, enrolled: 0, pending: 0, exemptions: 0 });
  const [factorPriority, setFactorPriority] = useState<string[]>(["WebAuthn", "TOTP", "Push", "SMS"]);
  const [forceEnroll, setForceEnroll] = useState(true);
  const [showRecovery, setShowRecovery] = useState(false);
  const [recoveryCodes, setRecoveryCodes] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch("/api/v1/auth/mfa/factors", {
      headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
    })
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(data => {
        if (data) {
          if (data.factors) setFactors(data.factors);
          if (data.compliance) setCompliance(data.compliance);
          if (data.factor_priority) setFactorPriority(data.factor_priority);
          if (data.force_enroll !== undefined) setForceEnroll(data.force_enroll);
          if (data.recovery_codes) setRecoveryCodes(data.recovery_codes);
        }
        setLoading(false);
      })
      .catch(e => { setError(e.message); setLoading(false); });
  }, []);

  const enrolledPct = compliance.total_users > 0 ? Math.round((compliance.enrolled / compliance.total_users) * 100) : 0;
  const factorColors: Record<string, string> = { TOTP: "bg-blue-100 text-blue-700", WebAuthn: "bg-green-100 text-green-700", Push: "bg-purple-100 text-purple-700", SMS: "bg-yellow-100 text-yellow-700" };

  if (loading) return <div className="p-8"><p>Loading...</p></div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;

  return (
    <div className="p-8 space-y-6 max-w-5xl">
      <h1 className="text-2xl font-bold">MFA Enrollment Center</h1>
      <p className="text-gray-600">Manage multi-factor authentication enrollment, recovery, and compliance.</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">MFA Compliance Dashboard</h2>
        <div className="flex items-center gap-4"><div className="flex-1 bg-gray-200 rounded-full h-6"><div className="bg-green-600 h-6 rounded-full flex items-center justify-center text-xs text-white font-medium" style={{ width: `${enrolledPct}%` }}>{enrolledPct}%</div></div></div>
        <div className="grid grid-cols-4 gap-4"><div className="text-center"><div className="text-2xl font-bold">{compliance.total_users}</div><div className="text-xs text-gray-500">Total Users</div></div><div className="text-center"><div className="text-2xl font-bold text-green-600">{compliance.enrolled}</div><div className="text-xs text-gray-500">Enrolled</div></div><div className="text-center"><div className="text-2xl font-bold text-yellow-600">{compliance.pending}</div><div className="text-xs text-gray-500">Pending</div></div><div className="text-center"><div className="text-2xl font-bold text-gray-400">{compliance.exemptions}</div><div className="text-xs text-gray-500">Exemptions</div></div></div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Factor Inventory</h2>
        <table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Factor</th><th scope="col">Name</th><th>Enrolled</th><th>Last Used</th><th>Status</th></tr></thead><tbody>
          {factors.map((f: MfaFactor, i: number) => (<tr key={i} className="border-b"><td className="py-2"><span className={`px-2 py-1 rounded text-xs ${factorColors[f.type] || ""}`}>{f.type}</span></td><td className="font-medium">{f.name}</td><td className="text-xs text-gray-500">{f.enrolled_at}</td><td className="text-xs text-gray-500">{f.last_used}</td><td><span className={`px-2 py-1 rounded text-xs ${f.status === "active" ? "bg-green-100 text-green-700" : "bg-gray-100 text-gray-500"}`}>{f.status}</span></td></tr>))}
        </tbody></table>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Factor Priority</h2>
        <div className="space-y-2">{factorPriority.map((f: string, i: number) => (<div key={f} className="flex items-center gap-3 border rounded p-2"><span className="text-gray-400 font-mono text-sm">#{i + 1}</span><span className={`px-2 py-1 rounded text-xs ${factorColors[f] || ""}`}>{f}</span><button onClick={() => { if (i > 0) { const next = [...factorPriority]; [next[i - 1], next[i]] = [next[i], next[i - 1]]; setFactorPriority(next); } }} className="ml-auto text-xs text-blue-600 hover:underline" disabled={i === 0}>Up</button><button onClick={() => { if (i < factorPriority.length - 1) { const next = [...factorPriority]; [next[i + 1], next[i]] = [next[i], next[i + 1]]; setFactorPriority(next); } }} className="text-xs text-blue-600 hover:underline" disabled={i === factorPriority.length - 1}>Down</button></div>))}</div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Recovery Codes</h2>
        <button onClick={() => setShowRecovery(!showRecovery)} className="px-4 py-1 bg-blue-600 text-white rounded text-sm hover:bg-blue-700">{showRecovery ? "Hide" : "Show"} Recovery Codes</button>
        {showRecovery && (<div className="grid grid-cols-4 gap-2 mt-3">{recoveryCodes.map((c: any, i: number) => (<div key={i} className="bg-gray-50 border rounded p-2 text-center font-mono text-sm">{c}</div>))}</div>)}
        <button className="px-4 py-1 border rounded text-sm hover:bg-gray-50">Regenerate Codes</button>
      </div>

      <div className="bg-white rounded-lg p-4 shadow flex items-center gap-3"><input aria-label="Force enroll" type="checkbox" checked={forceEnroll} onChange={(e) => setForceEnroll(e.target.checked)} className="w-4 h-4" /><label className="text-sm font-medium">Force MFA enrollment for all users without exemption</label></div>
    </div>
  );
}

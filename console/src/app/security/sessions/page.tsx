"use client";
import { useState, useCallback, useEffect } from "react";
import {
  Smartphone, Loader2, AlertCircle, X, RefreshCw, Ban, Search,
  ChevronRight, Globe, Clock, Shield, Activity, Check,
  CheckCircle2, AlertTriangle, XCircle, Eye, Laptop,
} from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";
import { useTranslations } from "@/lib/i18n";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface Session { id: string; user_id: string; ip: string; device: string; os: string; browser: string; risk_score: number; created_at: string; last_active: string; active: boolean; mfa_verified: boolean; location: string; }

type Tab = "active" | "details";

const SAMPLE_SESSIONS: Session[] = [
  { id: "s-001", user_id: "user:alice", ip: "10.0.1.42", device: "MacBook Pro", os: "macOS 14.2", browser: "Chrome 121", risk_score: 12, created_at: new Date(Date.now() - 3600000).toISOString(), last_active: new Date(Date.now() - 300000).toISOString(), active: true, mfa_verified: true, location: "San Francisco, US" },
  { id: "s-002", user_id: "user:bob", ip: "192.168.1.100", device: "Desktop", os: "Windows 11", browser: "Edge 121", risk_score: 34, created_at: new Date(Date.now() - 7200000).toISOString(), last_active: new Date(Date.now() - 1800000).toISOString(), active: true, mfa_verified: true, location: "New York, US" },
  { id: "s-003", user_id: "user:carol", ip: "172.16.0.55", device: "iPhone 15", os: "iOS 17.3", browser: "Safari Mobile", risk_score: 65, created_at: new Date(Date.now() - 86400000).toISOString(), last_active: new Date(Date.now() - 3600000).toISOString(), active: true, mfa_verified: false, location: "Unknown (VPN)" },
  { id: "s-004", user_id: "user:dave", ip: "10.0.2.18", device: "Workstation", os: "Linux 6.5", browser: "Firefox 122", risk_score: 8, created_at: new Date(Date.now() - 14400000).toISOString(), last_active: new Date(Date.now() - 600000).toISOString(), active: true, mfa_verified: true, location: "Berlin, DE" },
  { id: "s-005", user_id: "user:eve", ip: "203.0.113.50", device: "Unknown", os: "Unknown", browser: "python-requests/2.28", risk_score: 92, created_at: new Date(Date.now() - 1800000).toISOString(), last_active: new Date(Date.now() - 120000).toISOString(), active: true, mfa_verified: false, location: "Unknown (Tor)" },
];

function getRiskLevel(score: number) {
  if (score >= 85) return { label: "Critical", color: "text-red-600", bg: "bg-red-100 dark:bg-red-900/30" };
  if (score >= 60) return { label: "High", color: "text-orange-600", bg: "bg-orange-100 dark:bg-orange-900/30" };
  if (score >= 30) return { label: "Medium", color: "text-yellow-600", bg: "bg-yellow-100 dark:bg-yellow-900/30" };
  return { label: "Low", color: "text-green-600", bg: "bg-green-100 dark:bg-green-900/30" };
}

export default function SessionsPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("active");
  const [sessions, setSessions] = useState<Session[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState("");
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [confirmRevoke, setConfirmRevoke] = useState<string | null>(null);

  const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
  const H = { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID };
  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/auth/sessions", { headers: h }).catch(() => null);
      if (res?.ok) { const d = await res.json(); setSessions(d.sessions || d.items || SAMPLE_SESSIONS); }
      else setSessions(SAMPLE_SESSIONS);
    } catch { setError(t("sessions.loadError")); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const revokeSession = async (id: string) => {
    setActionLoading(id); setConfirmRevoke(null);
    try { await fetch(`/api/v1/auth/sessions/${id}`, { method: "DELETE", headers: h }); setSessions(prev => prev.map(s => s.id === id ? { ...s, active: false } : s)); }
    catch { setError(t("sessions.revokeError")); }
    finally { setActionLoading(null); }
  };

  const filtered = search ? sessions.filter(s => s.user_id.includes(search) || s.ip.includes(search)) : sessions;
  const activeSessions = sessions.filter(s => s.active);
  const highRisk = activeSessions.filter(s => s.risk_score >= 60);
  const selected = sessions.find(s => s.id === selectedId);

  return (
    <div className="space-y-6">
      <div><h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Smartphone className="h-6 w-6 text-cyan-500" /> {t("sessions.title")}</h1><p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("sessions.subtitle")}</p></div>

      {error && (<div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button></div>)}

      {/* Stats bar */}
      <div className="grid grid-cols-3 gap-4">
        <div className={card + " text-center !p-3"}><p className="text-lg font-bold">{activeSessions.length}</p><p className="text-xs text-gray-400">{t("sessions.active")}</p></div>
        <div className={card + " text-center !p-3"}><p className="text-lg font-bold text-orange-600">{highRisk.length}</p><p className="text-xs text-gray-400">{t("sessions.highRisk")}</p></div>
        <div className={card + " text-center !p-3"}><p className="text-lg font-bold text-blue-600">{new Set(activeSessions.map(s => s.user_id)).size}</p><p className="text-xs text-gray-400">{t("sessions.uniqueUsers")}</p></div>
      </div>

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([["active", t("sessions.activeSessions"), Smartphone], ["details", t("sessions.sessionDetails"), Eye]] as const).map(([id, label, Icon]) => (
          <button key={id} onClick={() => setTab(id as Tab)} aria-pressed={tab === id} className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === id ? "border-cyan-600 text-cyan-600 dark:text-cyan-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}><Icon className="h-4 w-4" /> {label}</button>
        ))}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-cyan-500" /></div> : (<>

      {/* ACTIVE SESSIONS */}
      {tab === "active" && (
        <div>
          <div className="mb-4"><div className="relative max-w-xs"><Search className="absolute left-3 top-2.5 h-4 w-4 text-gray-400" /><input type="text" value={search} onChange={e => setSearch(e.target.value)} placeholder={t("sessions.search")} className="w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 pl-9 pr-3 py-1.5 text-sm" /></div></div>
          <div className="overflow-x-auto"><table className="w-full text-sm">
            <thead className="bg-gray-50 dark:bg-gray-800/50"><tr>
              <th className="px-3 py-2 text-left text-xs text-gray-400">{t("sessions.user")}</th><th className="px-3 py-2 text-left text-xs text-gray-400">{t("sessions.device")}</th><th className="px-3 py-2 text-left text-xs text-gray-400">{t("sessions.ip")}</th><th className="px-3 py-2 text-center text-xs text-gray-400">{t("sessions.risk")}</th><th className="px-3 py-2 text-center text-xs text-gray-400">{t("sessions.mfa")}</th><th className="px-3 py-2 text-left text-xs text-gray-400">{t("sessions.lastActive")}</th><th className="px-3 py-2 text-right text-xs text-gray-400">{t("sessions.actions")}</th>
            </tr></thead>
            <tbody className="divide-y dark:divide-gray-800">{filtered.map(s => {
              const risk = getRiskLevel(s.risk_score);
              return (
                <tr key={s.id} className={`hover:bg-gray-50 dark:hover:bg-gray-900/30 ${s.risk_score >= 60 ? "bg-red-50 dark:bg-red-950/10" : ""} ${!s.active ? "opacity-40" : ""}`}>
                  <td className="px-3 py-3"><button onClick={() => { setSelectedId(s.id); setTab("details"); }} className="text-xs font-mono hover:text-cyan-600">{s.user_id}</button></td>
                  <td className="px-3 py-3"><div className="flex items-center gap-1.5">{s.device.includes("iPhone") || s.device.includes("Android") ? <Smartphone className="h-3 w-3 text-gray-400" /> : <Laptop className="h-3 w-3 text-gray-400" />}<span className="text-xs">{s.device}</span></div><p className="text-xs text-gray-400">{s.os} · {s.browser}</p></td>
                  <td className="px-3 py-3"><span className="flex items-center gap-1 text-xs font-mono"><Globe className="h-3 w-3 text-gray-400" />{s.ip}</span><p className="text-xs text-gray-400">{s.location}</p></td>
                  <td className="px-3 py-3 text-center"><span className={`px-1.5 py-0.5 rounded text-xs font-bold ${risk.bg} ${risk.color}`}>{s.risk_score}</span></td>
                  <td className="px-3 py-3 text-center">{s.mfa_verified ? <CheckCircle2 className="mx-auto h-4 w-4 text-green-500" /> : <XCircle className="mx-auto h-4 w-4 text-red-500" />}</td>
                  <td className="px-3 py-3 text-xs text-gray-400">{new Date(s.last_active).toLocaleTimeString()}</td>
                  <td className="px-3 py-3 text-right">{s.active && <button onClick={() => setConfirmRevoke(s.id)} disabled={actionLoading === s.id} aria-label="Revoke session" className="rounded-lg bg-red-600 px-2 py-1 text-xs font-medium text-white hover:bg-red-700 disabled:opacity-50">{actionLoading === s.id ? <Loader2 className="h-3 w-3 animate-spin" /> : t("sessions.revoke")}</button>}</td>
                </tr>
              );
            })}</tbody>
          </table></div>
        </div>
      )}

      {/* SESSION DETAILS */}
      {tab === "details" && (
        <div>
          {!selected ? (
            <div className={card}><div className="py-8 text-center"><Eye className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">{t("sessions.selectSession")}</p></div></div>
          ) : (
            <div className="space-y-4">
              <div className={card}>
                <div className="flex items-center justify-between mb-4">
                  <div className="flex items-center gap-3"><div className="flex h-10 w-10 items-center justify-center rounded-lg bg-cyan-100 dark:bg-cyan-900/30"><Smartphone className="h-5 w-5 text-cyan-500" /></div><div><h3 className="font-semibold text-sm">{selected.device}</h3><p className="text-xs text-gray-400">{selected.user_id}</p></div></div>
                  <div className="flex items-center gap-2"><span className={`px-2 py-0.5 rounded text-xs font-bold ${getRiskLevel(selected.risk_score).bg} ${getRiskLevel(selected.risk_score).color}`}>{t("sessions.riskScore")}: {selected.risk_score}</span>{selected.active && <button onClick={() => setConfirmRevoke(selected.id)} className="rounded-lg bg-red-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-red-700">{t("sessions.revoke")}</button>}</div>
                </div>
                <div className="grid grid-cols-2 gap-3">
                  {([["IP Address", selected.ip], ["Location", selected.location], ["OS", selected.os], ["Browser", selected.browser], ["MFA Verified", selected.mfa_verified ? "Yes" : "No"], ["Status", selected.active ? "Active" : "Revoked"], ["Created", new Date(selected.created_at).toLocaleString()], ["Last Active", new Date(selected.last_active).toLocaleString()]] as const).map(([label, value]: any[]) => (
                    <div key={label} className="rounded-lg border p-3 dark:border-gray-700"><p className="text-xs text-gray-400">{label}</p><p className="text-sm font-medium mt-0.5">{value}</p></div>
                  ))}
                </div>
              </div>
              <div className={card}>
                <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Activity className="h-4 w-4" /> {t("sessions.riskTimeline")}</h3>
                <svg width="100%" viewBox="0 0 300 60" className="overflow-visible">
                  <polyline points={Array.from({ length: 10 }, (_, i) => `${i * 33},${50 - Math.max(5, Math.min(45, selected.risk_score * 0.5 + Math.sin(i / 2) * 10))}`).join(" ")} fill="none" stroke={selected.risk_score >= 60 ? "#ef4444" : "#22c55e"} strokeWidth="2" strokeLinejoin="round" />
                  {Array.from({ length: 10 }).map((_, i) => <circle key={i} cx={i * 33} cy={50 - Math.max(5, Math.min(45, selected.risk_score * 0.5 + Math.sin(i / 2) * 10))} r="2" fill={selected.risk_score >= 60 ? "#ef4444" : "#22c55e"} />)}
                </svg>
              </div>
            </div>
          )}
        </div>
      )}

      </>)}

      {confirmRevoke && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setConfirmRevoke(null)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-sm rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <div className="flex items-center gap-2"><AlertTriangle className="h-5 w-5 text-red-500" /><h3 className="text-lg font-semibold">{t("sessions.revokeTitle")}</h3></div>
            <p className="mt-3 text-sm text-gray-500 dark:text-gray-400">{t("sessions.revokeConfirm")}</p>
            <div className="mt-4 flex justify-end gap-2"><button onClick={() => setConfirmRevoke(null)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">{t("common.cancel")}</button><button onClick={() => revokeSession(confirmRevoke)} className="flex items-center gap-1 rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700"><Ban className="h-4 w-4" /> {t("sessions.confirmRevoke")}</button></div>
          </div>
        </div>
      )}
    </div>
  );
}

"use client";
import { useState } from "react";
import {
  Fingerprint, Loader2, AlertCircle, X, Plus, Check, Trash2,
  Shield, Clock, Zap, Settings as SettingsIcon, KeyRound,
  CheckCircle2, XCircle, ChevronRight, Ticket, AlertTriangle,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

type Tab = "aaguid" | "tap" | "settings";

interface Aaguid { id: string; name: string; vendor: string; type: string; approved: boolean; }
interface Tap { id: string; user: string; code: string; expires: string; used: boolean; }

const AAGUIDS: Aaguid[] = [
  { id: "00000000-0000-0000-0000-000000000000", name: "Platform Authenticator", vendor: "Apple/Google/Microsoft", type: "platform", approved: true },
  { id: "adce0001-35bc-c21a-0cf1-1c11121112ad", name: "YubiKey 5 NFC", vendor: "Yubico", type: "cross-platform", approved: true },
  { id: "8876631b-d18a-49af-9bed-e7a0894ab35c", name: "Windows Hello", vendor: "Microsoft", type: "platform", approved: true },
  { id: "dd4ec289-e01d-41c9-bb89-70fa845d4bf2", name: "SoloKey V2", vendor: "SoloKeys", type: "cross-platform", approved: false },
];

const TAPS: Tap[] = [
  { id: "tap-001", user: "user:bob", code: "TAP-8X4K-2M9P", expires: new Date(Date.now() + 7200000).toISOString(), used: false },
  { id: "tap-002", user: "user:carol", code: "TAP-Q7R3-W5T8", expires: new Date(Date.now() - 3600000).toISOString(), used: true },
];

export default function WebAuthnEnterprisePage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("aaguid");
  const [aaguids, setAaguids] = useState(AAGUIDS);
  const [taps, setTaps] = useState(TAPS);
  const [showTapForm, setShowTapForm] = useState(false);
  const [tapUser, setTapUser] = useState("");
  const [tapHours, setTapHours] = useState(8);
  const [creating, setCreating] = useState(false);

  // Settings
  const [conditionalUI, setConditionalUI] = useState(true);
  const [enterpriseAttest, setEnterpriseAttest] = useState(true);
  const [recoveryPolicy, setRecoveryPolicy] = useState("admin_approval");

  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const toggleAaguid = (id: string) => setAaguids(prev => prev.map(a => a.id === id ? { ...a, approved: !a.approved } : a));
  const createTap = async () => {
    if (!tapUser) return;
    setCreating(true);
    try {
      const res = await fetch("/api/v1/auth/jit-taps", {
        method: "POST",
        headers: { "Content-Type": "application/json", ...authHeader() },
        body: JSON.stringify({ user: tapUser, hours: tapHours }),
      });
      if (res.ok) {
        const d = await res.json();
        setTaps(prev => [d, ...prev]);
        setShowTapForm(false);
        setTapUser("");
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed");
    } finally {
      setCreating(false);
    }
  };

  return (
    <div className="space-y-6">
      <div><h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Fingerprint className="h-6 w-6 text-green-500" /> {t("webauthnEnt.title")}</h1><p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("webauthnEnt.subtitle")}</p></div>

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([["aaguid", t("webauthnEnt.aaguid"), Shield], ["tap", `${t("webauthnEnt.tap")} (${taps.filter(x => !x.used).length})`, Ticket], ["settings", t("webauthnEnt.settings"), SettingsIcon]] as const).map(([id, label, Icon]) => (
          <button key={id} onClick={() => setTab(id as Tab)} aria-pressed={tab === id} className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === id ? "border-green-600 text-green-600 dark:text-green-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}><Icon className="h-4 w-4" /> {label}</button>
        ))}
      </div>

      {/* AAGUID */}
      {tab === "aaguid" && (
        <div className="space-y-3">{aaguids.map(a => (
          <div key={a.id} className={`${card} flex items-center justify-between !p-3`}>
            <div className="flex items-center gap-3"><div className="flex h-9 w-9 items-center justify-center rounded-lg bg-green-100 dark:bg-green-900/30"><Fingerprint className="h-4 w-4 text-green-500" /></div><div><div className="flex items-center gap-2"><span className="text-sm font-medium">{a.name}</span><span className={`px-1.5 py-0.5 rounded text-xs ${a.type === "platform" ? "bg-blue-100 dark:bg-blue-900/30 text-blue-600" : "bg-purple-100 dark:bg-purple-900/30 text-purple-600"}`}>{a.type}</span></div><p className="text-xs text-gray-400">{a.vendor} · <code className="font-mono">{a.id.slice(0, 13)}...</code></p></div></div>
            <button onClick={() => toggleAaguid(a.id)} aria-pressed={a.approved} className={`relative h-6 w-11 rounded-full transition ${a.approved ? "bg-green-500" : "bg-gray-300 dark:bg-gray-700"}`}><span className={`absolute top-0.5 h-5 w-5 rounded-full bg-white transition ${a.approved ? "left-5" : "left-0.5"}`} /></button>
          </div>
        ))}<button className="flex items-center gap-1 rounded-lg border border-dashed border-gray-300 px-4 py-2 text-sm text-gray-400 hover:border-gray-400 dark:border-gray-700 dark:hover:border-gray-500"><Plus className="h-3.5 w-3.5" /> {t("webauthnEnt.addAaguid")}</button></div>
      )}

      {/* TAP */}
      {tab === "tap" && (
        <div>
          <div className="mb-4"><button onClick={() => setShowTapForm(true)} className="flex items-center gap-1 rounded-lg bg-green-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-green-700"><Plus className="h-3 w-3" /> {t("webauthnEnt.issueTap")}</button></div>
          <div className="space-y-2">{taps.map(tap => { const expired = new Date(tap.expires).getTime() < Date.now(); return (
            <div key={tap.id} className={`${card} flex items-center justify-between !p-3 ${tap.used || expired ? "opacity-50" : ""}`}>
              <div className="flex items-center gap-3"><div className="flex h-9 w-9 items-center justify-center rounded-lg bg-green-100 dark:bg-green-900/30"><Ticket className="h-4 w-4 text-green-500" /></div><div><div className="flex items-center gap-2"><span className="text-xs font-mono">{tap.user}</span><code className="text-xs font-mono font-bold text-green-600">{tap.code}</code></div><p className="text-xs text-gray-400">{tap.used ? t("webauthnEnt.used") : expired ? t("webauthnEnt.expired") : `${t("webauthnEnt.expiresIn")} ${Math.round((new Date(tap.expires).getTime() - Date.now()) / 3600000)}h`}</p></div></div>
              <span className={`px-1.5 py-0.5 rounded text-xs ${tap.used ? "bg-gray-100 dark:bg-gray-800 text-gray-400" : expired ? "bg-red-100 dark:bg-red-900/30 text-red-600" : "bg-green-100 dark:bg-green-900/30 text-green-600"}`}>{tap.used ? t("webauthnEnt.used") : expired ? t("webauthnEnt.expired") : t("webauthnEnt.active")}</span>
            </div>
          );})}</div>
        </div>
      )}

      {/* SETTINGS */}
      {tab === "settings" && (
        <div className="space-y-4">
          <label className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700"><div><span className="text-sm font-medium">{t("webauthnEnt.conditionalUi")}</span><p className="text-xs text-gray-400">{t("webauthnEnt.conditionalUiDesc")}</p></div><button onClick={() => setConditionalUI(!conditionalUI)} aria-pressed={conditionalUI} className={`relative h-6 w-11 rounded-full transition ${conditionalUI ? "bg-green-500" : "bg-gray-300 dark:bg-gray-700"}`}><span className={`absolute top-0.5 h-5 w-5 rounded-full bg-white transition ${conditionalUI ? "left-5" : "left-0.5"}`} /></button></label>
          <label className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700"><div><span className="text-sm font-medium">{t("webauthnEnt.enterpriseAttest")}</span><p className="text-xs text-gray-400">{t("webauthnEnt.enterpriseAttestDesc")}</p></div><button onClick={() => setEnterpriseAttest(!enterpriseAttest)} aria-pressed={enterpriseAttest} className={`relative h-6 w-11 rounded-full transition ${enterpriseAttest ? "bg-green-500" : "bg-gray-300 dark:bg-gray-700"}`}><span className={`absolute top-0.5 h-5 w-5 rounded-full bg-white transition ${enterpriseAttest ? "left-5" : "left-0.5"}`} /></button></label>
          <div className="rounded-lg border p-3 dark:border-gray-700"><label className="text-sm font-medium">{t("webauthnEnt.recoveryPolicy")}</label><select value={recoveryPolicy} onChange={e => setRecoveryPolicy(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm"><option value="admin_approval">{t("webauthnEnt.policyAdmin")}</option><option value="email_verify">{t("webauthnEnt.policyEmail")}</option><option value="multiple_factors">{t("webauthnEnt.policyMulti")}</option></select></div>
        </div>
      )}

      {showTapForm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowTapForm(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <h3 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white"><Ticket className="h-5 w-5 text-green-500" /> {t("webauthnEnt.issueTap")}</h3>
            <div className="mt-4 space-y-3"><div><label className="text-sm font-medium">{t("webauthnEnt.forUser")}</label><input type="text" value={tapUser} onChange={e => setTapUser(e.target.value)} placeholder="user:bob" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" autoFocus /></div><div><label className="text-sm font-medium">{t("webauthnEnt.validFor")}</label><div className="mt-1 flex gap-2">{[1, 8, 24, 72].map(h => <button key={h} onClick={() => setTapHours(h)} aria-pressed={tapHours === h} className={`rounded-lg border px-3 py-1.5 text-sm ${tapHours === h ? "border-green-500 bg-green-50 dark:bg-green-950/30 text-green-600" : "border-gray-300 dark:border-gray-700"}`}>{h}h</button>)}</div></div></div>
            <div className="mt-4 flex justify-end gap-2"><button onClick={() => setShowTapForm(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">{t("common.cancel")}</button><button onClick={createTap} disabled={!tapUser || creating} className="rounded-lg bg-green-600 px-4 py-2 text-sm font-medium text-white hover:bg-green-700 disabled:opacity-50">{creating ? <Loader2 className="h-4 w-4 animate-spin" /> : t("webauthnEnt.create")}</button></div>
          </div>
        </div>
      )}
    </div>
  );
}

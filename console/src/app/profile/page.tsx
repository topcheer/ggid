"use client";
import { useState, useEffect, useCallback } from "react";
import { authHeader } from "@/lib/auth-helpers";
import {
  User, Shield, Smartphone, Loader2, AlertCircle, X, Check,
  Key, Lock, Mail, Phone, CheckCircle2, XCircle, Plus, Ban,
  RefreshCw, ChevronRight, Fingerprint, Globe,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

type Tab = "profile" | "security" | "devices";

interface Device { id: string; name: string; os: string; lastSeen: string; trusted: boolean; }

export default function EnhancedProfilePage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("profile");
  const [saving, setSaving] = useState(false);

  // Profile
  const [name, setName] = useState("Alice Chen");
  const [email, setEmail] = useState("alice@company.com");
  const [phone, setPhone] = useState("+1-555-0100");
  const [phoneVerified, setPhoneVerified] = useState(true);

  // Security
  const [mfaMethods, setMfaMethods] = useState<{ type: string; name: string; enabled: boolean }[]>([]);
  const [linkedAccounts, setLinkedAccounts] = useState<{ provider: string; email: string; connected: boolean }[]>([]);
  const [devices, setDevices] = useState<Device[]>([]);
  const [loadingProfile, setLoadingProfile] = useState(true);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "";

  // Fetch real profile data
  useEffect(() => {
    const loadProfile = async () => {
      setLoadingProfile(true);
      try {
        // Fetch MFA status
        const mfaRes = await fetch(`${API_BASE}/api/v1/auth/mfa/status`, { headers: { ...authHeader() } });
        if (mfaRes.ok) {
          const mfaData = await mfaRes.json();
          setMfaMethods(mfaData.methods || mfaData.factors || []);
        }
      } catch { /* empty state */ }

      try {
        // Fetch linked accounts
        const linkRes = await fetch(`${API_BASE}/api/v1/auth/account-linking`, { headers: { ...authHeader() } });
        if (linkRes.ok) {
          const linkData = await linkRes.json();
          setLinkedAccounts(linkData.accounts || linkData || []);
        }
      } catch { /* empty state */ }

      try {
        // Fetch sessions as device proxy
        const sessRes = await fetch(`${API_BASE}/api/v1/auth/sessions`, { headers: { ...authHeader() } });
        if (sessRes.ok) {
          const sessData = await sessRes.json();
          const sessions = sessData.sessions || sessData || [];
          setDevices(sessions.map((s: Record<string, string>) => ({
            id: s.session_id || s.id,
            name: s.device || s.user_agent?.split(' ').pop() || 'Unknown Device',
            os: s.user_agent || 'Unknown',
            lastSeen: s.last_active || s.created_at || new Date().toISOString(),
            trusted: s.trusted === true,
          })));
        }
      } catch { /* empty state */ }

      setLoadingProfile(false);
    };
    loadProfile();
  }, []);

  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const saveProfile = () => { setSaving(true); setTimeout(() => setSaving(false), 800); };
  const revokeDevice = (id: string) => setDevices(prev => prev.filter(d => d.id !== id));

  return (
    <div className="space-y-6">
      <div><h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><User className="h-6 w-6 text-blue-500" /> {t("profile.title")}</h1><p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("profile.subtitle")}</p></div>

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([["profile", t("profile.profile"), User], ["security", t("profile.security"), Shield], ["devices", `${t("profile.devices")} (${devices.length})`, Smartphone]] as const).map(([id, label, Icon]) => (
          <button key={id} onClick={() => setTab(id as Tab)} aria-pressed={tab === id} className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === id ? "border-blue-600 text-blue-600 dark:text-blue-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}><Icon className="h-4 w-4" /> {label}</button>
        ))}
      </div>

      {/* PROFILE */}
      {tab === "profile" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={card}>
            <h3 className="mb-4 text-sm font-semibold uppercase text-gray-400">{t("profile.personalInfo")}</h3>
            <div className="space-y-3">
              <div><label className="text-sm font-medium">{t("profile.fullName")}</label><input type="text" value={name} onChange={e => setName(e.target.value)} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>
              <div><label className="text-sm font-medium">{t("profile.email")}</label><div className="mt-1 flex gap-2"><div className="relative flex-1"><Mail className="absolute left-3 top-2.5 h-4 w-4 text-gray-400" /><input type="email" value={email} onChange={e => setEmail(e.target.value)} className="w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 pl-9 pr-3 py-2 text-sm" /></div>{email && <span className="flex items-center gap-1 px-2 py-1 rounded text-xs bg-green-100 dark:bg-green-900/30 text-green-600"><CheckCircle2 className="h-3 w-3" /> {t("profile.verified")}</span>}</div></div>
              <div><label className="text-sm font-medium">{t("profile.phone")}</label><div className="mt-1 flex gap-2"><div className="relative flex-1"><Phone className="absolute left-3 top-2.5 h-4 w-4 text-gray-400" /><input type="tel" value={phone} onChange={e => setPhone(e.target.value)} className="w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 pl-9 pr-3 py-2 text-sm" /></div>{phoneVerified ? <span className="flex items-center gap-1 px-2 py-1 rounded text-xs bg-green-100 dark:bg-green-900/30 text-green-600"><CheckCircle2 className="h-3 w-3" /> {t("profile.verified")}</span> : <button className="px-2 py-1 rounded text-xs bg-blue-600 text-white">{t("profile.verify")}</button>}</div></div>
              <button onClick={saveProfile} disabled={saving} className="flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50">{saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />} {t("profile.save")}</button>
            </div>
          </div>
          <div className={card}>
            <h3 className="mb-4 text-sm font-semibold uppercase text-gray-400">{t("profile.avatar")}</h3>
            <div className="flex items-center gap-4"><div className="flex h-20 w-20 items-center justify-center rounded-full bg-blue-100 dark:bg-blue-900/30"><span className="text-2xl font-bold text-blue-600">{name.charAt(0)}</span></div><button className="rounded-lg border border-gray-300 px-3 py-1.5 text-sm dark:border-gray-700">{t("profile.uploadPhoto")}</button></div>
          </div>
        </div>
      )}

      {/* SECURITY */}
      {tab === "security" && (
        <div className="space-y-6">
          <div className={card}>
            <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Lock className="h-4 w-4" /> {t("profile.password")}</h3>
            <button className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">{t("profile.changePassword")}</button>
          </div>
          <div className={card}>
            <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Key className="h-4 w-4" /> {t("profile.mfaMethods")}</h3>
            <div className="space-y-2">{mfaMethods.map((m: any, i: any) => (
              <div key={i} className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700">
                <div className="flex items-center gap-3">{m.type === "webauthn" ? <Fingerprint className="h-5 w-5 text-green-500" /> : m.type === "totp" ? <Smartphone className="h-5 w-5 text-blue-500" /> : <Phone className="h-5 w-5 text-gray-400" />}<div><span className="text-sm font-medium">{m.name}</span><p className="text-xs text-gray-400">{m.type}</p></div></div>
                <button onClick={() => setMfaMethods(prev => prev.map((x: any, j: any) => j === i ? { ...x, enabled: !x.enabled } : x))} aria-pressed={m.enabled} className={`relative h-6 w-11 rounded-full transition ${m.enabled ? "bg-green-500" : "bg-gray-300 dark:bg-gray-700"}`}><span className={`absolute top-0.5 h-5 w-5 rounded-full bg-white transition ${m.enabled ? "left-5" : "left-0.5"}`} /></button>
              </div>
            ))}<button className="mt-2 flex items-center gap-1 rounded-lg bg-blue-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-blue-700"><Plus className="h-3 w-3" /> {t("profile.addMfa")}</button></div>
          </div>
          <div className={card}>
            <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Globe className="h-4 w-4" /> {t("profile.linkedAccounts")}</h3>
            <div className="space-y-2">{linkedAccounts.map((acc: any, i: any) => (
              <div key={i} className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700"><div className="flex items-center gap-3"><Globe className="h-5 w-5 text-gray-400" /><div><span className="text-sm font-medium">{acc.provider}</span>{acc.email && <p className="text-xs text-gray-400">{acc.email}</p>}</div></div>{acc.connected ? <span className="flex items-center gap-1 px-2 py-0.5 rounded text-xs bg-green-100 dark:bg-green-900/30 text-green-600"><CheckCircle2 className="h-3 w-3" /> {t("profile.connected")}</span> : <button className="rounded-lg border border-gray-300 px-3 py-1 text-xs dark:border-gray-700">{t("profile.connect")}</button>}</div>
            ))}</div>
          </div>
        </div>
      )}

      {/* DEVICES */}
      {tab === "devices" && (
        <div className="space-y-2">
          {devices.map(d => (
            <div key={d.id} className={`${card} flex items-center justify-between !p-3`}>
              <div className="flex items-center gap-3"><div className="flex h-9 w-9 items-center justify-center rounded-lg bg-gray-100 dark:bg-gray-700"><Smartphone className="h-4 w-4 text-gray-500" /></div><div><div className="flex items-center gap-2"><span className="text-sm font-medium">{d.name}</span>{d.trusted && <span className="px-1.5 py-0.5 rounded text-xs bg-green-100 dark:bg-green-900/30 text-green-600">{t("profile.trusted")}</span>}</div><p className="text-xs text-gray-400">{d.os} · {t("profile.lastSeen")} {new Date(d.lastSeen).toLocaleDateString()}</p></div></div>
              <button onClick={() => revokeDevice(d.id)} aria-label={"Revoke " + d.name} className="rounded p-1.5 text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20"><Ban className="h-3.5 w-3.5" /></button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

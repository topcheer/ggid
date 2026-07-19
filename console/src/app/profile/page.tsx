"use client";
import { useState, useEffect, useCallback } from "react";
import { authHeader } from "@/lib/auth-helpers";
import { DEFAULT_TENANT_ID } from "@/lib/api-config";
import {
  User, Shield, Smartphone, Loader2, AlertCircle, X, Check,
  Key, Lock, Mail, Phone, CheckCircle2, XCircle, Plus, Ban,
  RefreshCw, ChevronRight, Fingerprint, Globe, Eye, EyeOff,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

type Tab = "profile" | "security" | "devices";

interface Device { id: string; name: string; os: string; lastSeen: string; trusted: boolean; }

export default function EnhancedProfilePage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("profile");
  const [saving, setSaving] = useState(false);

  // Profile
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [phone, setPhone] = useState("");
  const [phoneVerified, setPhoneVerified] = useState(false);
  const [profileLoaded, setProfileLoaded] = useState(false);
  const [profileSaved, setProfileSaved] = useState(false);

  // Change password
  const [showChangePw, setShowChangePw] = useState(false);
  const [curPw, setCurPw] = useState("");
  const [newPw, setNewPw] = useState("");
  const [confirmPw, setConfirmPw] = useState("");
  const [showPw, setShowPw] = useState(false);
  const [pwScore, setPwScore] = useState(0);
  const [pwError, setPwError] = useState("");
  const [pwSuccess, setPwSuccess] = useState("");
  const [changingPw, setChangingPw] = useState(false);

  // Security
  const [mfaMethods, setMfaMethods] = useState<{ type: string; name: string; enabled: boolean }[]>([]);
  const [linkedAccounts, setLinkedAccounts] = useState<{ provider: string; email: string; connected: boolean }[]>([]);
  const [devices, setDevices] = useState<Device[]>([]);
  const [loadingProfile, setLoadingProfile] = useState(true);

  // MFA wizard state
  const [mfaSetup, setMfaSetup] = useState<"idle" | "qr" | "backup">("idle");
  const [deviceId, setDeviceId] = useState("");
  const [qrCodeUrl, setQrCodeUrl] = useState("");
  const [mfaSecret, setMfaSecret] = useState("");
  const [totpCode, setTotpCode] = useState("");
  const [verifying, setVerifying] = useState(false);
  const [backupCodes, setBackupCodes] = useState<string[]>([]);

  const setupTotp = async () => {
    setMfaSetup("qr"); setTotpCode("");
    try {
      const res = await fetch(`${API_BASE}/api/v1/auth/mfa/setup`, {
        method: "POST",
        headers: { "Content-Type": "application/json", ...authHeader() },
        body: JSON.stringify({ method: "totp" }),
      });
      if (res.ok) {
        const d = await res.json();
        const rawUri = d.qr_code_uri || d.qr_code_url || d.qr_url || "";
        // If it's an otpauth:// URI, generate QR via API and extract secret
        let extractedSecret = d.secret || "";
        if (!extractedSecret && rawUri) {
          const m = rawUri.match(/secret=([^&]+)/);
          if (m) extractedSecret = decodeURIComponent(m[1]);
        }
        const qrImg = rawUri.startsWith("data:") ? rawUri
          : rawUri.startsWith("otpauth:") ? `https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(rawUri)}`
          : rawUri;
        setQrCodeUrl(qrImg);
        setMfaSecret(extractedSecret || d.otpauth_secret || "");
        setDeviceId(d.device_id || "");
      }
    } catch { /* show error */ }
  };

  const verifyTotp = async () => {
    setVerifying(true);
    try {
      const res = await fetch(`${API_BASE}/api/v1/auth/mfa/verify`, {
        method: "POST",
        headers: { "Content-Type": "application/json", ...authHeader() },
        body: JSON.stringify({ device_id: deviceId, code: totpCode }),
      });
      if (res.ok) { const d = await res.json(); setBackupCodes(d.backup_codes || []); setMfaSetup("backup"); setMfaMethods(prev => [...prev, { type: "totp", name: "Authenticator App", enabled: true }]); }
    } catch { /* error */ }
    setVerifying(false);
  };

  const disableMfa = async (type: string) => {
    if (!confirm(`Remove ${type} MFA method?`)) return;
    try {
      await fetch(`${API_BASE}/api/v1/auth/mfa/disable`, { method: "POST", headers: { "Content-Type": "application/json", ...authHeader() }, body: JSON.stringify({ method: type }) });
      setMfaMethods(prev => prev.filter((m: any) => m.type !== type));
    } catch { /* error */ }
  };

  const unlinkAccount = async (provider: string) => {
    if (!confirm(`Unlink ${provider}?`)) return;
    try {
      await fetch(`${API_BASE}/api/v1/auth/account-linking/${provider.toLowerCase()}`, { method: "DELETE", headers: { ...authHeader() } });
      setLinkedAccounts(prev => prev.filter(a => a.provider !== provider));
    } catch { /* error */ }
  }

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
          const mf = mfaData.methods || mfaData.factors || [];
          setMfaMethods(Array.isArray(mf) ? mf : []);
        }
      } catch { /* empty state */ }

      try {
        // Fetch linked accounts
        const linkRes = await fetch(`${API_BASE}/api/v1/auth/account-linking`, { headers: { ...authHeader() } });
        if (linkRes.ok) {
          const linkData = await linkRes.json();
          const la = linkData.accounts || linkData || [];
          setLinkedAccounts(Array.isArray(la) ? la : []);
        }
      } catch { /* empty state */ }

      try {
        // Fetch sessions as device proxy
        const sessRes = await fetch(`${API_BASE}/api/v1/auth/sessions`, { headers: { ...authHeader() } });
        if (sessRes.ok) {
          const sessData = await sessRes.json();
          const sessions = sessData.sessions || sessData || [];
          if (!Array.isArray(sessions)) { setDevices([]); return; }
          setDevices(sessions.map((s: Record<string, string>) => ({
            id: s.session_id || s.id,
            name: s.device || s.user_agent?.split(' ').pop() || 'Unknown Device',
            os: s.user_agent || 'Unknown',
            lastSeen: s.last_active || s.created_at || new Date().toISOString(),
            trusted: String(s.trusted) === "true",
          })));
        }
      } catch { /* empty state */ }

      // Load profile from /users/me or JWT
      try {
        const meRes = await fetch(`${API_BASE}/api/v1/users/me`, {
          headers: {
            ...authHeader(),
            "X-Tenant-ID": localStorage.getItem("ggid_tenant_id") || DEFAULT_TENANT_ID,
          },
        });
        if (meRes.ok) {
          const me = await meRes.json();
          setName(me.display_name || me.name || me.username || "");
          setEmail(me.email || "");
          setPhone(me.phone || "");
          setPhoneVerified(me.phone_verified || false);
        }
      } catch {
        // Fallback: read from JWT payload
        try {
          const token = localStorage.getItem("ggid_access_token") || "";
          if (token) {
            const payload = JSON.parse(atob(token.split(".")[1]));
            setName(payload.name || payload.username || "");
            setEmail(payload.email || "");
          }
        } catch {}
      }
      setProfileLoaded(true);

      setLoadingProfile(false);
    };
    loadProfile();
  }, []);

  const saveProfile = async () => {
    setSaving(true);
    try {
      const res = await fetch(`${API_BASE}/api/v1/users/me`, {
        method: "PUT",
        headers: { "Content-Type": "application/json", ...authHeader() },
        body: JSON.stringify({ display_name: name, email, phone }),
      });
      if (res.ok) {
        setProfileSaved(true);
        setTimeout(() => setProfileSaved(false), 3000);
      }
    } catch {}
    setSaving(false);
  };

  const checkPwStrength = (pw: string) => {
    if (pw.length === 0) { setPwScore(0); return; }
    let score = pw.length < 8 ? 1 : 2;
    if (/[A-Z]/.test(pw)) score++;
    if (/[0-9]/.test(pw)) score++;
    if (/[^A-Za-z0-9]/.test(pw)) score++;
    setPwScore(Math.min(score, 5));
  };

  const handleChangePassword = async (e: React.FormEvent) => {
    e.preventDefault();
    setPwError(""); setPwSuccess("");
    if (newPw.length < 12) { setPwError("New password must be at least 12 characters"); return; }
    if (newPw !== confirmPw) { setPwError("Passwords do not match"); return; }
    if (newPw === curPw) { setPwError("New password must be different"); return; }
    setChangingPw(true);
    try {
      const res = await fetch(`${API_BASE}/api/v1/auth/change-password`, {
        method: "POST",
        headers: { "Content-Type": "application/json", ...authHeader() },
        body: JSON.stringify({ current_password: curPw, new_password: newPw }),
      });
      if (res.ok) {
        setPwSuccess("Password changed successfully.");
        setCurPw(""); setNewPw(""); setConfirmPw("");
        setShowChangePw(false);
        setTimeout(() => setPwSuccess(""), 5000);
      } else {
        const d = await res.json().catch(() => ({}));
        setPwError(d.error?.message || d.error || "Failed to change password");
      }
    } catch {
      setPwError("Network error — please try again");
    }
    setChangingPw(false);
  };

  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";


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
              <div><label className="text-sm font-medium">{t("profile.fullName")}</label><input type="text" value={name} onChange={e => setName(e.target.value)} placeholder={!profileLoaded ? "Loading..." : "Enter your name"} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm disabled:opacity-50" disabled={!profileLoaded} /></div>
              <div><label className="text-sm font-medium">{t("profile.email")}</label><div className="mt-1 flex gap-2"><div className="relative flex-1"><Mail className="absolute left-3 top-2.5 h-4 w-4 text-gray-400" /><input type="email" value={email} onChange={e => setEmail(e.target.value)} placeholder={!profileLoaded ? "Loading..." : "you@example.com"} className="w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 pl-9 pr-3 py-2 text-sm disabled:opacity-50" disabled={!profileLoaded} /></div>{email && <span className="flex items-center gap-1 px-2 py-1 rounded text-xs bg-green-100 dark:bg-green-900/30 text-green-600"><CheckCircle2 className="h-3 w-3" /> {t("profile.verified")}</span>}</div></div>
              <div><label className="text-sm font-medium">{t("profile.phone")}</label><div className="mt-1 flex gap-2"><div className="relative flex-1"><Phone className="absolute left-3 top-2.5 h-4 w-4 text-gray-400" /><input type="tel" value={phone} onChange={e => setPhone(e.target.value)} className="w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 pl-9 pr-3 py-2 text-sm" /></div>{phoneVerified ? <span className="flex items-center gap-1 px-2 py-1 rounded text-xs bg-green-100 dark:bg-green-900/30 text-green-600"><CheckCircle2 className="h-3 w-3" /> {t("profile.verified")}</span> : <button className="px-2 py-1 rounded text-xs bg-blue-600 text-white">{t("profile.verify")}</button>}</div></div>
              {profileSaved && <span className="flex items-center gap-1 text-xs text-green-600"><Check className="h-3 w-3" /> Saved</span>}
              <button onClick={saveProfile} disabled={saving || !profileLoaded} className="flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50">{saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />} {t("profile.save")}</button>
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
            {pwSuccess && <div className="mb-3 flex items-center gap-2 rounded-lg border border-green-200 bg-green-50 px-3 py-2 text-sm text-green-700 dark:border-green-800 dark:bg-green-950"><Check className="h-4 w-4" /> {pwSuccess}</div>}
            {pwError && <div className="mb-3 flex items-center gap-2 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-800 dark:bg-red-950"><AlertCircle className="h-4 w-4" /> {pwError}</div>}
            {showChangePw ? (
              <form onSubmit={handleChangePassword} className="space-y-3">
                <div>
                  <label className="text-xs font-medium text-gray-500">Current Password</label>
                  <input type={showPw ? "text" : "password"} value={curPw} onChange={(e) => setCurPw(e.target.value)} required className="mt-1 w-full rounded-lg border border-gray-300 dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" placeholder="••••••••" />
                </div>
                <div>
                  <label className="text-xs font-medium text-gray-500">New Password</label>
                  <div className="relative">
                    <input type={showPw ? "text" : "password"} value={newPw} onChange={(e) => { setNewPw(e.target.value); checkPwStrength(e.target.value); }} required className="mt-1 w-full rounded-lg border border-gray-300 dark:border-gray-700 dark:bg-gray-900 px-3 py-2 pr-9 text-sm" placeholder="At least 12 characters" />
                    <button type="button" onClick={() => setShowPw(!showPw)} className="absolute right-3 top-3 text-gray-400">{showPw ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}</button>
                  </div>
                  {pwScore > 0 && (
                    <div className="mt-1 flex gap-1">
                      {[1,2,3,4,5].map(n => <div key={n} className={`h-1 flex-1 rounded-full ${n <= pwScore ? (pwScore <= 2 ? "bg-red-500" : pwScore <= 3 ? "bg-amber-500" : "bg-green-500") : "bg-gray-200 dark:bg-gray-700"}`} />)}
                    </div>
                  )}
                </div>
                <div>
                  <label className="text-xs font-medium text-gray-500">Confirm New Password</label>
                  <input type={showPw ? "text" : "password"} value={confirmPw} onChange={(e) => setConfirmPw(e.target.value)} required className="mt-1 w-full rounded-lg border border-gray-300 dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" placeholder="••••••••" />
                </div>
                <div className="flex gap-2">
                  <button type="submit" disabled={changingPw} className="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50">{changingPw ? <Loader2 className="h-4 w-4 animate-spin" /> : "Change Password"}</button>
                  <button type="button" onClick={() => { setShowChangePw(false); setCurPw(""); setNewPw(""); setConfirmPw(""); setPwError(""); }} className="rounded-lg border border-gray-300 dark:border-gray-700 px-4 py-2 text-sm">Cancel</button>
                </div>
              </form>
            ) : (
              <button onClick={() => setShowChangePw(true)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">{t("profile.changePassword")}</button>
            )}
          </div>
          <div className={card}>
            <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Key className="h-4 w-4" /> {t("profile.mfaMethods")}</h3>

            {/* MFA Setup Wizard */}
            {mfaSetup === "qr" && (
              <div className="mb-4 rounded-lg border border-blue-200 dark:border-blue-900 bg-blue-50 dark:bg-blue-950/20 p-4">
                <h4 className="text-sm font-medium mb-2">Scan QR Code</h4>
                {qrCodeUrl ? (
                  <img src={qrCodeUrl} alt="MFA QR Code" className="w-48 h-48 mx-auto rounded-lg" />
                ) : (
                  <div className="w-48 h-48 mx-auto flex items-center justify-center text-gray-400"><Loader2 className="w-8 h-8 animate-spin" /></div>
                )}
                {mfaSecret && (
                  <div className="mt-2 rounded-lg bg-gray-100 dark:bg-gray-800 p-3 text-center">
                    <p className="text-xs text-gray-500 mb-1">Or enter this key manually:</p>
                    <code className="font-mono text-sm text-gray-900 dark:text-gray-100 select-all break-all">{mfaSecret}</code>
                    <button
                      onClick={() => { navigator.clipboard.writeText(mfaSecret); }}
                      className="ml-2 text-xs text-blue-600 hover:underline"
                    >Copy</button>
                  </div>
                )}
                <div className="mt-3">
                  <input type="text" maxLength={6} value={totpCode} onChange={(e) => setTotpCode(e.target.value)} placeholder="Enter 6-digit code" className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 text-center text-lg font-mono tracking-widest" />
                </div>
                <div className="mt-2 flex gap-2">
                  <button onClick={verifyTotp} disabled={totpCode.length !== 6 || verifying} className="flex-1 rounded-lg bg-blue-600 px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50">
                    {verifying ? <Loader2 className="w-4 h-4 animate-spin" /> : "Verify"}
                  </button>
                  <button onClick={() => { setMfaSetup("idle"); setQrCodeUrl(""); setTotpCode(""); }} className="rounded-lg border border-gray-300 px-3 py-1.5 text-sm">Cancel</button>
                </div>
              </div>
            )}

            {mfaSetup === "backup" && (
              <div className="mb-4 rounded-lg border border-green-200 dark:border-green-900 bg-green-50 dark:bg-green-950/20 p-4">
                <h4 className="text-sm font-medium mb-2">Save Your Backup Codes</h4>
                <div className="grid grid-cols-2 gap-1 font-mono text-sm">
                  {backupCodes.map((code: string, i: any) => <div key={i} className="px-2 py-1 rounded bg-white dark:bg-gray-800">{code}</div>)}
                </div>
                <p className="text-xs text-gray-500 mt-2">Store these safely — each can be used once.</p>
                <button onClick={() => setMfaSetup("idle")} className="mt-2 rounded-lg bg-green-600 px-3 py-1.5 text-sm font-medium text-white">I've Saved Them</button>
              </div>
            )}

            {/* Existing MFA methods */}
            <div className="space-y-2">
              {mfaMethods.length === 0 && mfaSetup === "idle" && (
                <p className="text-sm text-gray-400 py-2">No MFA methods enrolled. Enable TOTP to secure your account.</p>
              )}
              {mfaMethods.map((m: any, i: any) => (
                <div key={i} className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700">
                  <div className="flex items-center gap-3">{m.type === "webauthn" ? <Fingerprint className="h-5 w-5 text-green-500" /> : m.type === "totp" ? <Smartphone className="h-5 w-5 text-blue-500" /> : <Phone className="h-5 w-5 text-gray-400" />}<div><span className="text-sm font-medium">{m.name}</span><p className="text-xs text-gray-400">{m.type}</p></div></div>
                  <button onClick={() => disableMfa(m.type)} className="rounded-lg border border-red-300 px-2 py-1 text-xs text-red-600 hover:bg-red-50 dark:hover:bg-red-950">Disable</button>
                </div>
              ))}
              {mfaSetup === "idle" && (
                <button onClick={setupTotp} className="mt-2 flex items-center gap-1 rounded-lg bg-blue-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-blue-700">
                  <Plus className="h-3 w-3" /> Enable TOTP
                </button>
              )}
            </div>
          </div>

          {/* Social Accounts */}
          <div className={card}>
            <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Globe className="h-4 w-4" /> {t("profile.linkedAccounts")}</h3>
            <div className="space-y-2">
              {linkedAccounts.length === 0 && <p className="text-sm text-gray-400 py-2">No linked social accounts.</p>}
              {linkedAccounts.map((acc: any, i: any) => (
                <div key={i} className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700">
                  <div className="flex items-center gap-3"><Globe className="h-5 w-5 text-gray-400" /><div><span className="text-sm font-medium">{acc.provider}</span>{acc.email && <p className="text-xs text-gray-400">{acc.email}</p>}</div></div>
                  {acc.connected ? (
                    <button onClick={() => unlinkAccount(acc.provider)} className="rounded-lg border border-red-300 px-2 py-1 text-xs text-red-600 hover:bg-red-50 dark:hover:bg-red-950">Unlink</button>
                  ) : (
                    <a href={`${API_BASE}/api/v1/auth/social/${acc.provider.toLowerCase()}/connect`} className="rounded-lg border border-gray-300 px-3 py-1 text-xs dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-800">Connect</a>
                  )}
                </div>
              ))}
              {/* Always-available providers */}
              {["Google", "GitHub"].filter(p => !linkedAccounts.some(a => a.provider === p)).map(provider => (
                <div key={provider} className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700">
                  <div className="flex items-center gap-3"><Globe className="h-5 w-5 text-gray-400" /><span className="text-sm font-medium">{provider}</span></div>
                  <a href={`${API_BASE}/api/v1/auth/social/${provider.toLowerCase()}/connect`} className="rounded-lg border border-gray-300 px-3 py-1 text-xs dark:border-gray-700 hover:bg-gray-50">Connect</a>
                </div>
              ))}
            </div>
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

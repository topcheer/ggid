"use client";

import { useEffect, useState, useRef } from "react";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";
import {
  Save, Building2, Upload, Flag, Gauge, Lock, Clock, Shield, Check, Palette,
} from "lucide-react";

const FEATURE_FLAGS = [
  { key: "scim", label: "tenant.enableScim" },
  { key: "webauthn", label: "tenant.enableWebauthn" },
  { key: "social", label: "tenant.enableSocial" },
  { key: "saml", label: "tenant.enableSaml" },
  { key: "branding", label: "tenant.enableBranding" },
  { key: "audit_export", label: "tenant.enableAuditExport" },
  { key: "api_keys", label: "tenant.enableApiKeys" },
  { key: "webhooks", label: "tenant.enableWebhooks" },
] as const;

export default function TenantConfigPage() {
  const { apiFetch, TENANT_ID } = useApi();
  const t = useTranslations();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [msg, setMsg] = useState<string | null>(null);

  // Profile
  const [profile, setProfile] = useState({ name: "", logoUrl: "", domain: "" });
  const [savingProfile, setSavingProfile] = useState(false);

  // Feature flags
  const [flags, setFlags] = useState<Record<string, boolean>>({
    scim: false, webauthn: false, social: false, saml: false,
    branding: false, audit_export: false, api_keys: false, webhooks: false,
  });
  const [savingFlags, setSavingFlags] = useState(false);

  // Rate limit
  const [rateLimit, setRateLimit] = useState({ requestsPerMinute: 100, burstLimit: 200 });
  const [savingRate, setSavingRate] = useState(false);

  // Password policy
  const [pwPolicy, setPwPolicy] = useState({
    minLength: 12, requireUpper: true, requireLower: true, requireDigit: true,
    requireSpecial: true, historyCount: 5, expiryDays: 90,
  });
  const [savingPw, setSavingPw] = useState(false);

  // Session policy
  const [sessPolicy, setSessPolicy] = useState({
    timeout: 60, idleTimeout: 30, concurrentLimit: 5,
  });
  const [savingSess, setSavingSess] = useState(false);

  // MFA enforcement
  const [mfa, setMfa] = useState({ requireMfa: false, method: "TOTP", gracePeriod: 7 });
  const [savingMfa, setSavingMfa] = useState(false);
  const [loading, setLoading] = useState(true);
  const [fetchError, setFetchError] = useState<string | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      setLoading(true);
      setFetchError(null);
      try {
        const data = await apiFetch<Record<string, unknown>>(`/api/v1/tenants/${TENANT_ID}`);
        if (data.name) setProfile({ name: data.name as string, logoUrl: (data.logo_url as string) || "", domain: (data.domain as string) || "" });
        if (data.feature_flags) setFlags(prev => ({ ...prev, ...data.feature_flags as Record<string, boolean> }));
        if (data.rate_limits) {
          const rl = data.rate_limits as Record<string, number>;
          setRateLimit({ requestsPerMinute: rl.requests_per_minute || 100, burstLimit: rl.burst_limit || 200 });
        }
        if (data.password_policy) {
          const pp = data.password_policy as Record<string, unknown>;
          setPwPolicy({
            minLength: Number(pp.min_length) || 12, requireUpper: Boolean(pp.require_uppercase),
            requireLower: Boolean(pp.require_lowercase), requireDigit: Boolean(pp.require_digit),
            requireSpecial: Boolean(pp.require_special), historyCount: Number(pp.history_count) || 5,
            expiryDays: Number(pp.expiry_days) || 90,
          });
        }
        if (data.session_policy) {
          const sp = data.session_policy as Record<string, number>;
          setSessPolicy({ timeout: sp.timeout || 60, idleTimeout: sp.idle_timeout || 30, concurrentLimit: sp.concurrent_limit || 5 });
        }
        if (data.mfa_config) {
          const mc = data.mfa_config as Record<string, unknown>;
          setMfa({ requireMfa: Boolean(mc.require_mfa), method: (mc.method as string) || "TOTP", gracePeriod: Number(mc.grace_period) || 7 });
        }
      } catch (err) {
        setFetchError(err instanceof Error ? err.message : "Failed to load tenant config");
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, [apiFetch, TENANT_ID]);

  useEffect(() => {
    if (msg) { const t = setTimeout(() => setMsg(null), 3000); return () => clearTimeout(t); }
  }, [msg]);

  const handleLogoUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file || !file.type.startsWith("image/")) { setMsg(t("tenant.selectImage")); return; }
    const reader = new FileReader();
    reader.onload = () => setProfile(prev => ({ ...prev, logoUrl: reader.result as string }));
    reader.readAsDataURL(file);
  };

  const inputCls = "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";
  const labelCls = "mb-1 block text-xs font-medium text-gray-500";
  const cardCls = "rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const headingCls = "mb-4 text-lg font-semibold text-gray-900 dark:text-gray-100";
  const saveBtn = "flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50";

  const Toggle = ({ on, onClick, label }: { on: boolean; onClick: () => void; label?: string }) => (
    <button onClick={onClick} aria-label={label || (on ? "Toggle off" : "Toggle on")} aria-pressed={on} className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${on ? "bg-brand-600" : "bg-gray-300 dark:bg-gray-600"}`}>
      <span className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${on ? "translate-x-6" : "translate-x-1"}`} />
    </button>
  );

  if (loading) return <div className="p-8 text-gray-400">Loading tenant config...</div>;
  if (fetchError) return <div className="p-8 text-red-400">Error: {fetchError}</div>;

  return (
    <div>
      <div className="mb-6">
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-gray-100">
          <Building2 className="h-7 w-7 text-brand-600" /> {t("tenant.title")}
        </h1>
      </div>

      {msg && <div role="status" className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400">{msg}</div>}

      <div className="space-y-6">
        {/* Tenant Profile */}
        <div className={cardCls}>
          <h2 className={headingCls}><Building2 className="mr-2 inline h-5 w-5 text-brand-600" /> {t("tenant.profile")}</h2>
          <div className="grid gap-6 sm:grid-cols-2">
            <div>
              <label className={labelCls}>{t("common.name")}</label>
              <input aria-label="My Organization" value={profile.name} onChange={e => setProfile({ ...profile, name: e.target.value })} className={inputCls} placeholder="My Organization" />
            </div>
            <div>
              <label className={labelCls}>{t("common.domain")}</label>
              <input aria-label="company.com" value={profile.domain} onChange={e => setProfile({ ...profile, domain: e.target.value })} className={inputCls} placeholder="company.com" />
            </div>
          </div>
          <div className="mt-4 flex items-center gap-6">
            <div className="flex h-20 w-20 shrink-0 items-center justify-center overflow-hidden rounded-full border-2 border-gray-200 bg-gray-100 dark:border-gray-700 dark:bg-gray-700">
              {profile.logoUrl ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img src={profile.logoUrl} alt={t("tenant.logoPreview")} className="h-full w-full object-cover" />
              ) : <Building2 className="h-8 w-8 text-gray-400" />}
            </div>
            <div>
              <input aria-label="Input field" ref={fileInputRef} type="file" accept="image/*" onChange={handleLogoUpload} className="hidden" />
              <button onClick={() => fileInputRef.current?.click()} aria-label={t("branding.uploadLogo")} className="flex items-center gap-2 rounded-lg border border-brand-600 px-4 py-2 text-sm font-medium text-brand-600 hover:bg-brand-50 dark:hover:bg-brand-900/30">
                <Upload className="h-4 w-4" /> {t("branding.uploadLogo")}
              </button>
              {profile.logoUrl && <button onClick={() => setProfile({ ...profile, logoUrl: "" })} aria-label={t("tenant.removeLogo")} className="ml-2 rounded-lg border border-red-300 px-3 py-2 text-sm text-red-600 hover:bg-red-50 dark:border-red-800 dark:hover:bg-red-950">{t("tenant.removeLogo")}</button>}
            </div>
          </div>
          <div className="mt-4 flex justify-end">
            <button onClick={async () => { setSavingProfile(true); try { await apiFetch(`/api/v1/tenants/${TENANT_ID}`, { method: "PUT", body: JSON.stringify({ name: profile.name, logo_url: profile.logoUrl, domain: profile.domain }) }); setMsg(t("tenant.profileSaved")); } catch { setMsg(t("tenant.profileSavedOffline")); } finally { setSavingProfile(false); } }} disabled={savingProfile} aria-label={t("tenant.saveProfile")} className={saveBtn}>
              {savingProfile ? <Save className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />} {t("tenant.saveProfile")}
            </button>
          </div>
        </div>

        {/* Feature Flags */}
        <div className={cardCls}>
          <h2 className={headingCls}><Flag className="mr-2 inline h-5 w-5 text-brand-600" /> {t("tenant.featureFlags")}</h2>
          <div className="grid gap-3 sm:grid-cols-2">
            {FEATURE_FLAGS.map(f => (
              <div key={f.key} className="flex items-center justify-between rounded-lg border border-gray-200 p-3 dark:border-gray-700">
                <span className="text-sm font-medium text-gray-700 dark:text-gray-300">{t(f.label)}</span>
                <Toggle on={flags[f.key]} onClick={() => setFlags({ ...flags, [f.key]: !flags[f.key] })} />
              </div>
            ))}
          </div>
          <div className="mt-4 flex justify-end">
            <button onClick={async () => { setSavingFlags(true); try { await apiFetch(`/api/v1/tenants/${TENANT_ID}`, { method: "PUT", body: JSON.stringify({ feature_flags: flags }) }); setMsg(t("tenant.flagsSaved")); } catch { setMsg(t("tenant.flagsSavedOffline")); } finally { setSavingFlags(false); } }} disabled={savingFlags} aria-label={t("tenant.saveFlags")} className={saveBtn}>
              <Save className="h-4 w-4" /> {t("tenant.saveFlags")}
            </button>
          </div>
        </div>

        {/* Rate Limit Config */}
        <div className={cardCls}>
          <h2 className={headingCls}><Gauge className="mr-2 inline h-5 w-5 text-brand-600" /> {t("tenant.rateLimitConfig")}</h2>
          <div className="grid gap-6 sm:grid-cols-2">
            <div>
              <label className={labelCls}>{t("tenant.requestsPerMinute")}</label>
              <input aria-label="rate Limit" type="number" min={1} value={rateLimit.requestsPerMinute} onChange={e => setRateLimit({ ...rateLimit, requestsPerMinute: Number(e.target.value) || 100 })} className={inputCls} />
            </div>
            <div>
              <label className={labelCls}>{t("tenant.burstLimit")}</label>
              <input aria-label="rate Limit" type="number" min={1} value={rateLimit.burstLimit} onChange={e => setRateLimit({ ...rateLimit, burstLimit: Number(e.target.value) || 200 })} className={inputCls} />
            </div>
          </div>
          <div className="mt-4 flex justify-end">
            <button onClick={async () => { setSavingRate(true); try { await apiFetch(`/api/v1/tenants/${TENANT_ID}`, { method: "PUT", body: JSON.stringify({ rate_limits: { requests_per_minute: rateLimit.requestsPerMinute, burst_limit: rateLimit.burstLimit } }) }); setMsg(t("tenant.rateSaved")); } catch { setMsg(t("tenant.rateSavedOffline")); } finally { setSavingRate(false); } }} disabled={savingRate} aria-label={t("tenant.saveRate")} className={saveBtn}>
              <Save className="h-4 w-4" /> {t("tenant.saveRate")}
            </button>
          </div>
        </div>

        {/* Password Policy */}
        <div className={cardCls}>
          <h2 className={headingCls}><Lock className="mr-2 inline h-5 w-5 text-brand-600" /> {t("tenant.passwordPolicy")}</h2>
          <div className="space-y-4">
            <div>
              <div className="mb-1 flex items-center justify-between">
                <label className={labelCls}>{t("tenant.minPasswordLength")}</label>
                <span className="text-sm font-semibold text-gray-900 dark:text-gray-100">{pwPolicy.minLength}</span>
              </div>
              <input aria-label="pw Policy" type="range" min={8} max={128} value={pwPolicy.minLength} onChange={e => setPwPolicy({ ...pwPolicy, minLength: Number(e.target.value) })} className="w-full accent-brand-600" />
            </div>
            <div className="grid gap-3 sm:grid-cols-2">
              {(["requireUpper", "requireLower", "requireDigit", "requireSpecial"] as const).map((key: any) => (
                <div key={key} className="flex items-center justify-between rounded-lg border border-gray-200 p-3 dark:border-gray-700">
                  <span className="text-sm font-medium text-gray-700 dark:text-gray-300">{t(`tenant.${key}`)}</span>
                  <Toggle on={pwPolicy[key]} onClick={() => setPwPolicy({ ...pwPolicy, [key]: !pwPolicy[key] })} />
                </div>
              ))}
            </div>
            <div className="grid gap-6 sm:grid-cols-2">
              <div><label className={labelCls}>{t("tenant.passwordHistory")}</label><input aria-label="pw Policy" type="number" min={0} max={24} value={pwPolicy.historyCount} onChange={e => setPwPolicy({ ...pwPolicy, historyCount: Number(e.target.value) || 0 })} className={inputCls} /></div>
              <div><label className={labelCls}>{t("tenant.expiryDays")}</label><input aria-label="pw Policy" type="number" min={0} max={365} value={pwPolicy.expiryDays} onChange={e => setPwPolicy({ ...pwPolicy, expiryDays: Number(e.target.value) || 0 })} className={inputCls} /></div>
            </div>
          </div>
          <div className="mt-4 flex justify-end">
            <button onClick={async () => { setSavingPw(true); try { await apiFetch(`/api/v1/tenants/${TENANT_ID}`, { method: "PUT", body: JSON.stringify({ password_policy: { min_length: pwPolicy.minLength, require_uppercase: pwPolicy.requireUpper, require_lowercase: pwPolicy.requireLower, require_digit: pwPolicy.requireDigit, require_special: pwPolicy.requireSpecial, history_count: pwPolicy.historyCount, expiry_days: pwPolicy.expiryDays } }) }); setMsg(t("tenant.pwSaved")); } catch { setMsg(t("tenant.pwSavedOffline")); } finally { setSavingPw(false); } }} disabled={savingPw} aria-label={t("tenant.savePwPolicy")} className={saveBtn}>
              <Save className="h-4 w-4" /> {t("tenant.savePwPolicy")}
            </button>
          </div>
        </div>

        {/* Session Policy */}
        <div className={cardCls}>
          <h2 className={headingCls}><Clock className="mr-2 inline h-5 w-5 text-brand-600" /> {t("tenant.sessionPolicy")}</h2>
          <div className="grid gap-6 sm:grid-cols-3">
            <div><label className={labelCls}>{t("tenant.sessionTimeout")}</label><input aria-label="sess Policy" type="number" min={5} max={1440} value={sessPolicy.timeout} onChange={e => setSessPolicy({ ...sessPolicy, timeout: Number(e.target.value) || 60 })} className={inputCls} /></div>
            <div><label className={labelCls}>{t("tenant.idleTimeout")}</label><input aria-label="sess Policy" type="number" min={1} max={1440} value={sessPolicy.idleTimeout} onChange={e => setSessPolicy({ ...sessPolicy, idleTimeout: Number(e.target.value) || 30 })} className={inputCls} /></div>
            <div><label className={labelCls}>{t("tenant.concurrentSessions")}</label><input aria-label="sess Policy" type="number" min={1} max={100} value={sessPolicy.concurrentLimit} onChange={e => setSessPolicy({ ...sessPolicy, concurrentLimit: Number(e.target.value) || 5 })} className={inputCls} /></div>
          </div>
          <div className="mt-4 flex justify-end">
            <button onClick={async () => { setSavingSess(true); try { await apiFetch(`/api/v1/tenants/${TENANT_ID}`, { method: "PUT", body: JSON.stringify({ session_policy: { timeout: sessPolicy.timeout, idle_timeout: sessPolicy.idleTimeout, concurrent_limit: sessPolicy.concurrentLimit } }) }); setMsg(t("tenant.sessSaved")); } catch { setMsg(t("tenant.sessSavedOffline")); } finally { setSavingSess(false); } }} disabled={savingSess} aria-label={t("tenant.saveSessPolicy")} className={saveBtn}>
              <Save className="h-4 w-4" /> {t("tenant.saveSessPolicy")}
            </button>
          </div>
        </div>

        {/* MFA Enforcement */}
        <div className={cardCls}>
          <h2 className={headingCls}><Shield className="mr-2 inline h-5 w-5 text-brand-600" /> {t("tenant.mfaEnforcement")}</h2>
          <div className="space-y-4">
            <div className="flex items-center justify-between rounded-lg border border-gray-200 p-4 dark:border-gray-700">
              <div className="flex items-center gap-3">
                <Shield className="h-5 w-5 text-gray-500" />
                <div>
                  <p className="text-sm font-medium text-gray-900 dark:text-gray-100">{t("tenant.requireMfa")}</p>
                  <p className="text-xs text-gray-500">{t("tenant.requireMfaDesc")}</p>
                </div>
              </div>
              <Toggle on={mfa.requireMfa} onClick={() => setMfa({ ...mfa, requireMfa: !mfa.requireMfa })} />
            </div>
            <div className="grid gap-6 sm:grid-cols-2">
              <div>
                <label className={labelCls}>{t("tenant.requiredMethod")}</label>
                <select aria-label="mfa" value={mfa.method} onChange={e => setMfa({ ...mfa, method: e.target.value })} className={inputCls}>
                  <option value="TOTP">TOTP</option>
                  <option value="WebAuthn">WebAuthn</option>
                  <option value="Any">Any</option>
                </select>
              </div>
              <div>
                <label className={labelCls}>{t("tenant.gracePeriod")}</label>
                <input aria-label="mfa" type="number" min={0} max={90} value={mfa.gracePeriod} onChange={e => setMfa({ ...mfa, gracePeriod: Number(e.target.value) || 0 })} className={inputCls} />
              </div>
            </div>
          </div>
          <div className="mt-4 flex justify-end">
            <button onClick={async () => { setSavingMfa(true); try { await apiFetch(`/api/v1/tenants/${TENANT_ID}`, { method: "PUT", body: JSON.stringify({ mfa_config: { require_mfa: mfa.requireMfa, method: mfa.method, grace_period: mfa.gracePeriod } }) }); setMsg(t("tenant.mfaSaved")); } catch { setMsg(t("tenant.mfaSavedOffline")); } finally { setSavingMfa(false); } }} disabled={savingMfa} aria-label={t("tenant.saveMfa")} className={saveBtn}>
              <Save className="h-4 w-4" /> {t("tenant.saveMfa")}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

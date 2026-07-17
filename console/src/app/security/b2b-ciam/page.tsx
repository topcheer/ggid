"use client";

import { useState, useCallback, useEffect } from "react";
import {
  Building2, User, Palette, Globe, Check, Loader2, AlertCircle, X,
  RefreshCw, ChevronRight, ChevronLeft, Upload, Eye, TrendingUp,
  Users, Shield, Activity, AlertTriangle, Sparkles, Gauge, Clock,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

// ==================== Types ====================
interface JourneyMetrics {
  signups_30d: number;
  conversion_rate_pct: number;
  active_users_7d: number;
  login_success_rate_pct: number;
  mfa_coverage_pct: number;
  avg_signup_time_sec: number;
  drop_off_step: string;
  profile_completeness_avg: number;
}

interface Tenant {
  id: string;
  org_name: string;
  plan: string;
  created_at: string;
  user_count: number;
  status: "trial" | "active" | "suspended";
  branding: { primary_color: string; logo_url: string; custom_domain: string | null };
}

type Tab = "signup" | "onboarding" | "branding" | "mfa" | "analytics";

const presetColors = [
  { name: "Indigo", value: "#6366f1" }, { name: "Emerald", value: "#10b981" },
  { name: "Blue", value: "#3b82f6" }, { name: "Rose", value: "#f43f5e" },
  { name: "Amber", value: "#f59e0b" }, { name: "Purple", value: "#a855f7" },
];

const wizardSteps = ["Organization", "Admin Account", "Branding", "Domain", "Complete"];

export default function B2BCIAMPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("signup");
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [metrics, setMetrics] = useState<JourneyMetrics | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  // Signup wizard
  const [showWizard, setShowWizard] = useState(false);
  const [wizardStep, setWizardStep] = useState(0);
  const [submitting, setSubmitting] = useState(false);
  // Wizard form state
  const [orgName, setOrgName] = useState("");
  const [orgSize, setOrgSize] = useState("1-50");
  const [industry, setIndustry] = useState("Technology");
  const [adminEmail, setAdminEmail] = useState("");
  const [adminPassword, setAdminPassword] = useState("");
  const [adminName, setAdminName] = useState("");
  const [primaryColor, setPrimaryColor] = useState("#6366f1");
  const [logoUrl, setLogoUrl] = useState("");
  const [customDomain, setCustomDomain] = useState("");
  // Branding
  const [brandColor, setBrandColor] = useState("#6366f1");
  const [brandLogo, setBrandLogo] = useState("");
  const [brandDomain, setBrandDomain] = useState("");
  const [savingBrand, setSavingBrand] = useState(false);
  // MFA config
  const [riskThreshold, setRiskThreshold] = useState(50);
  const [mfaConfig, setMfaConfig] = useState<{ low: string; medium: string; high: string }>({
    low: "none", medium: "optional", high: "required",
  });

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
      const [tenantsRes, metricsRes] = await Promise.all([
        fetch("/api/v1/identity/tenants", { headers: h }).catch(() => null),
        fetch("/api/v1/identity/ciam/metrics", { headers: h }).catch(() => null),
      ]);
      if (tenantsRes?.ok) { const d = await tenantsRes.json(); setTenants(d.tenants || d.items || []); }
      if (metricsRes?.ok) setMetrics(await metricsRes.json());
    } catch { setError("Failed to load CIAM data"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const submitSignup = async () => {
    setSubmitting(true);
    try {
      const res = await fetch("/api/v1/identity/tenants/self-register", {
        method: "POST",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify({ org_name: orgName, org_size: orgSize, industry, admin: { email: adminEmail, password: adminPassword, name: adminName }, branding: { primary_color: primaryColor, logo_url: logoUrl }, custom_domain: customDomain }),
      });
      if (res.ok) { setShowWizard(false); setWizardStep(0); loadData(); }
      else { setError("Registration failed"); }
    } catch { setError("Network error"); }
    finally { setSubmitting(false); }
  };

  const saveBranding = async () => {
    setSavingBrand(true);
    try {
      await fetch("/api/v1/identity/tenants/branding", {
        method: "PUT",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify({ primary_color: brandColor, logo_url: brandLogo, custom_domain: brandDomain }),
      });
    } catch { /* noop */ }
    finally { setSavingBrand(false); }
  };

  const saveMfaConfig = async () => {
    try {
      await fetch("/api/v1/auth/risk-engine/config", {
        method: "PUT",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify({ risk_threshold: riskThreshold, mfa_policy: mfaConfig }),
      });
    } catch { /* noop */ }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <Building2 className="h-6 w-6 text-indigo-500" />
            B2B CIAM — Self-Service & Customer Journey
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Customer identity management — self-registration, branding, risk-based MFA, and journey analytics.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button onClick={() => setShowWizard(true)} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-700">
            <Building2 className="h-4 w-4" /> New Tenant
          </button>
          <button onClick={loadData} disabled={loading} aria-label="Refresh" className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800">
            <RefreshCw className={"h-4 w-4 " + (loading ? "animate-spin" : "")} /> Refresh
          </button>
        </div>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {/* Tabs */}
      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "signup" as Tab, label: "Tenants", icon: Building2 },
          { id: "onboarding" as Tab, label: "Progressive Onboarding", icon: Sparkles },
          { id: "branding" as Tab, label: "Branding", icon: Palette },
          { id: "mfa" as Tab, label: "Risk MFA", icon: Shield },
          { id: "analytics" as Tab, label: "Journey Analytics", icon: TrendingUp },
        ]).map(tb => { const Icon = tb.icon; return (
          <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id}
            className={"flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap " +
              (tab === tb.id ? "border-indigo-600 text-indigo-600 dark:text-indigo-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300")}>
            <Icon className="h-4 w-4" /> {tb.label}
          </button>
        ); })}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-500" /></div> : (<>

      {/* TENANTS TAB */}
      {tab === "signup" && (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {tenants.length === 0 ? (
            <div className={cardCls + " sm:col-span-2 lg:col-span-3"}><div className="py-12 text-center"><Building2 className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No tenants registered.</p><button onClick={() => setShowWizard(true)} className="mt-3 text-sm text-indigo-600 hover:underline">Register first tenant</button></div></div>
          ) : tenants.map(tn => (
            <div key={tn.id} className={cardCls + " hover:shadow-md transition"}>
              <div className="flex items-start justify-between">
                <div className="flex items-center gap-3">
                  <div className="h-10 w-10 rounded-lg flex items-center justify-center" style={{ backgroundColor: tn.branding?.primary_color || "#6366f1" }}>
                    <Building2 className="h-5 w-5 text-white" />
                  </div>
                  <div>
                    <h3 className="font-semibold text-gray-900 dark:text-white">{tn.org_name}</h3>
                    <p className="text-xs text-gray-400">{tn.plan} · {tn.status}</p>
                  </div>
                </div>
              </div>
              <div className="mt-3 grid grid-cols-3 gap-2 text-center">
                <div><p className="text-xs text-gray-400">Users</p><p className="text-sm font-bold">{tn.user_count}</p></div>
                <div><p className="text-xs text-gray-400">Domain</p><p className="text-xs font-mono truncate">{tn.branding?.custom_domain || "default"}</p></div>
                <div><p className="text-xs text-gray-400">Created</p><p className="text-xs">{new Date(tn.created_at).toLocaleDateString()}</p></div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* PROGRESSIVE ONBOARDING TAB */}
      {tab === "onboarding" && (
        <div className="space-y-4">
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Sparkles className="h-4 w-4" /> Profile Completeness</h2>
            {metrics ? (
              <div>
                <div className="flex items-center justify-between mb-2"><span className="text-sm text-gray-500">Average across all users</span><span className="text-lg font-bold text-indigo-600">{metrics.profile_completeness_avg}%</span></div>
                <div className="h-4 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
                  <div className="h-full rounded-full bg-gradient-to-r from-indigo-500 to-purple-500 transition-all" style={{ width: `${metrics.profile_completeness_avg}%` }} />
                </div>
                {metrics.drop_off_step && <p className="mt-2 flex items-center gap-1 text-xs text-yellow-600"><AlertTriangle className="h-3 w-3" /> Most users drop off at: <span className="font-medium">{metrics.drop_off_step}</span></p>}
              </div>
            ) : <p className="text-sm text-gray-400">No data available.</p>}
          </div>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
            {[
              { label: "Upload Avatar", desc: "Users with profile photo", icon: Upload },
              { label: "Set MFA", desc: "Users with 2FA enabled", icon: Shield },
              { label: "Complete Profile", desc: "All fields filled", icon: Check },
            ].map(item => { const Icon = item.icon; return (
              <div key={item.label} className={cardCls}>
                <Icon className="h-6 w-6 text-indigo-400" />
                <h3 className="mt-2 font-medium text-sm">{item.label}</h3>
                <p className="text-xs text-gray-400">{item.desc}</p>
                <div className="mt-2 flex items-center gap-2">
                  <div className="h-2 flex-1 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700"><div className="h-full rounded-full bg-green-500" style={{ width: "65%" }} /></div>
                  <span className="text-xs text-gray-500">65%</span>
                </div>
                <button className="mt-2 text-xs text-indigo-600 hover:underline">Send reminder</button>
              </div>
            ); })}
          </div>
        </div>
      )}

      {/* BRANDING TAB */}
      {tab === "branding" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Palette className="h-4 w-4" /> Brand Customization</h2>
            <div className="space-y-4">
              <div><label className="text-sm font-medium">Primary Color</label>
                <div className="mt-2 flex flex-wrap gap-2">
                  {presetColors.map(c => (
                    <button key={c.value} onClick={() => setBrandColor(c.value)} aria-label={c.name} aria-pressed={brandColor === c.value} className={"h-8 w-8 rounded-lg border-2 transition " + (brandColor === c.value ? "border-gray-900 dark:border-white scale-110" : "border-transparent")} style={{ backgroundColor: c.value }} title={c.name} />
                  ))}
                  <input aria-label="Custom color" type="color" value={brandColor} onChange={e => setBrandColor(e.target.value)} className="h-8 w-8 rounded-lg border-0 cursor-pointer" />
                </div>
              </div>
              <div><label className="text-sm font-medium">Logo URL</label><input aria-label="Logo URL" type="text" value={brandLogo} onChange={e => setBrandLogo(e.target.value)} placeholder="https://cdn.example.com/logo.png" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>
              <div><label className="text-sm font-medium">Custom Domain (CNAME)</label><input aria-label="Custom domain" type="text" value={brandDomain} onChange={e => setBrandDomain(e.target.value)} placeholder="auth.yourcompany.com" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" />
                {brandDomain && <p className="mt-1 text-xs text-gray-400">CNAME target: <span className="font-mono">cname.ggid.dev</span></p>}
              </div>
              <button onClick={saveBranding} disabled={savingBrand} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{savingBrand ? <Loader2 className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />} Save Branding</button>
            </div>
          </div>
          {/* Live preview */}
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Eye className="h-4 w-4" /> Live Preview</h2>
            <div className="overflow-hidden rounded-xl border dark:border-gray-700">
              <div className="p-6 text-center" style={{ backgroundColor: brandColor + "15" }}>
                {brandLogo ? <img src={brandLogo} alt="Logo preview" className="h-12 mx-auto mb-3 rounded" /> : <div className="h-12 w-12 mx-auto mb-3 rounded-lg flex items-center justify-center" style={{ backgroundColor: brandColor }}><Building2 className="h-6 w-6 text-white" /></div>}
                <h3 className="text-lg font-bold" style={{ color: brandColor }}>Sign in to {orgName || "Your Company"}</h3>
                <p className="mt-1 text-xs text-gray-500">Enter your credentials</p>
                <div className="mt-4 space-y-2">
                  <input type="text" disabled placeholder="Email" className="w-full rounded-lg border px-3 py-2 text-sm opacity-60" />
                  <input type="password" disabled placeholder="Password" className="w-full rounded-lg border px-3 py-2 text-sm opacity-60" />
                  <button disabled className="w-full rounded-lg px-3 py-2 text-sm font-medium text-white" style={{ backgroundColor: brandColor }}>Sign In</button>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* RISK MFA TAB */}
      {tab === "mfa" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Shield className="h-4 w-4" /> Risk-Driven MFA Configuration</h2>
            <div>
              <div className="flex items-center justify-between mb-2"><span className="text-sm font-medium">Risk Threshold</span><span className="text-lg font-bold text-indigo-600">{riskThreshold}</span></div>
              <input aria-label="Risk threshold slider" type="range" min={0} max={100} value={riskThreshold} onChange={e => setRiskThreshold(parseInt(e.target.value))} className="w-full accent-indigo-600" />
              <div className="mt-1 flex justify-between text-xs text-gray-400"><span>Always MFA (0)</span><span>Adaptive (50)</span><span>Minimal (100)</span></div>
            </div>
            <div className="mt-6 space-y-3">
              <p className="text-xs font-semibold uppercase text-gray-400">MFA Policy</p>
              {([
                { level: "low", label: "Low Risk (trusted device, known IP)", color: "text-green-600" },
                { level: "medium", label: "Medium Risk (new device, off-hours)", color: "text-yellow-600" },
                { level: "high", label: "High Risk (impossible travel, TOR)", color: "text-red-600" },
              ] as const).map(p => (
                <div key={p.level} className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700">
                  <div><span className={"text-sm font-medium " + p.color}>{p.label}</span><p className="text-xs text-gray-400">Action when risk score triggers</p></div>
                  <select aria-label={`${p.level} risk action`} value={mfaConfig[p.level]} onChange={e => setMfaConfig({ ...mfaConfig, [p.level]: e.target.value })} className="rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-1.5 text-sm">
                    <option value="none">No MFA</option>
                    <option value="optional">Optional</option>
                    <option value="required">Require MFA</option>
                    <option value="block">Block Access</option>
                  </select>
                </div>
              ))}
            </div>
            <button onClick={saveMfaConfig} className="mt-4 flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700"><Check className="h-4 w-4" /> Save MFA Policy</button>
          </div>
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Gauge className="h-4 w-4" /> Risk Assessment Preview</h2>
            <div className="space-y-3">
              {[
                { scenario: "User logs in from office IP at 2pm on trusted laptop", risk: 15, action: "No MFA — seamless login" },
                { scenario: "User logs in from new phone at 3am", risk: 55, action: "Require MFA (TOTP)" },
                { scenario: "User logs in from TOR exit node in different country", risk: 92, action: "Block + alert SOC" },
              ].map((s, i) => (
                <div key={i} className="rounded-lg border p-3 dark:border-gray-700">
                  <p className="text-sm text-gray-700 dark:text-gray-300">{s.scenario}</p>
                  <div className="mt-2 flex items-center gap-3">
                    <div className="flex-1 h-2 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
                      <div className={"h-full rounded-full " + (s.risk < 30 ? "bg-green-500" : s.risk < 70 ? "bg-yellow-500" : "bg-red-500")} style={{ width: `${s.risk}%` }} />
                    </div>
                    <span className={"text-xs font-bold " + (s.risk < 30 ? "text-green-600" : s.risk < 70 ? "text-yellow-600" : "text-red-600")}>{s.risk}</span>
                  </div>
                  <p className="mt-1 text-xs text-gray-400">→ {s.action}</p>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* ANALYTICS TAB */}
      {tab === "analytics" && metrics && (
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-6">
            {([
              { label: "Signups (30d)", value: metrics.signups_30d, icon: Users, color: "text-blue-600" },
              { label: "Conversion", value: metrics.conversion_rate_pct + "%", icon: TrendingUp, color: "text-green-600" },
              { label: "Active (7d)", value: metrics.active_users_7d, icon: Activity, color: "text-purple-600" },
              { label: "Login Success", value: metrics.login_success_rate_pct + "%", icon: Check, color: "text-emerald-600" },
              { label: "MFA Coverage", value: metrics.mfa_coverage_pct + "%", icon: Shield, color: "text-indigo-600" },
              { label: "Avg Signup", value: metrics.avg_signup_time_sec + "s", icon: Clock, color: "text-orange-600" },
            ]).map(m => { const Icon = m.icon; return (
              <div key={m.label} className={cardCls + " text-center"}>
                <Icon className={"h-5 w-5 mx-auto " + m.color} />
                <p className="mt-2 text-2xl font-bold">{m.value}</p>
                <p className="text-xs text-gray-400">{m.label}</p>
              </div>
            ); })}
          </div>
        </div>
      )}
      {tab === "analytics" && !metrics && (
        <div className={cardCls}><div className="py-12 text-center"><TrendingUp className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No analytics data. Backend endpoint /api/v1/identity/ciam/metrics pending.</p></div></div>
      )}

      </>)}

      {/* SIGNUP WIZARD */}
      {showWizard && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowWizard(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 max-h-[90vh] w-full max-w-xl overflow-y-auto rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <div className="mb-6 flex items-center justify-between">
              <h3 className="text-lg font-semibold text-gray-900 dark:text-white">Register New Organization</h3>
              <button onClick={() => setShowWizard(false)} aria-label="Close"><X className="h-5 w-5 text-gray-400" /></button>
            </div>
            <div className="mb-6 flex items-center gap-1">
              {wizardSteps.map((s, i) => (
                <div key={i} className="flex items-center gap-1 flex-1">
                  <div className={"flex h-7 w-7 items-center justify-center rounded-full text-xs font-bold " + (i <= wizardStep ? "bg-indigo-600 text-white" : "bg-gray-200 dark:bg-gray-700 text-gray-400")}>{i < wizardStep ? <Check className="h-3.5 w-3.5" /> : i + 1}</div>
                  {i < wizardSteps.length - 1 && <div className={"h-0.5 flex-1 " + (i < wizardStep ? "bg-indigo-600" : "bg-gray-200 dark:bg-gray-700")} />}
                </div>
              ))}
            </div>
            <div className="min-h-[200px]">
              {wizardStep === 0 && <div className="space-y-3">
                <div><label className="text-sm font-medium">Organization Name *</label><input aria-label="Org name" type="text" value={orgName} onChange={e => setOrgName(e.target.value)} placeholder="Acme Corporation" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" autoFocus /></div>
                <div className="grid grid-cols-2 gap-3">
                  <div><label className="text-sm font-medium">Size</label><select aria-label="Org size" value={orgSize} onChange={e => setOrgSize(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm"><option>1-50</option><option>51-200</option><option>201-1000</option><option>1000+</option></select></div>
                  <div><label className="text-sm font-medium">Industry</label><select aria-label="Industry" value={industry} onChange={e => setIndustry(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm"><option>Technology</option><option>Finance</option><option>Healthcare</option><option>Education</option><option>Manufacturing</option><option>Other</option></select></div>
                </div>
              </div>}
              {wizardStep === 1 && <div className="space-y-3">
                <div><label className="text-sm font-medium">Admin Name *</label><input aria-label="Admin name" type="text" value={adminName} onChange={e => setAdminName(e.target.value)} placeholder="Jane Doe" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" autoFocus /></div>
                <div><label className="text-sm font-medium">Admin Email *</label><input aria-label="Admin email" type="email" value={adminEmail} onChange={e => setAdminEmail(e.target.value)} placeholder="admin@acme.com" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>
                <div><label className="text-sm font-medium">Password *</label><input aria-label="Admin password" type="password" autoComplete="new-password" value={adminPassword} onChange={e => setAdminPassword(e.target.value)} placeholder="••••••••" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>
              </div>}
              {wizardStep === 2 && <div className="space-y-3">
                <label className="text-sm font-medium">Theme Color</label>
                <div className="flex flex-wrap gap-2">
                  {presetColors.map(c => <button key={c.value} onClick={() => setPrimaryColor(c.value)} aria-label={c.name} aria-pressed={primaryColor === c.value} className={"h-10 w-10 rounded-lg border-2 " + (primaryColor === c.value ? "border-gray-900 dark:border-white scale-110" : "border-transparent")} style={{ backgroundColor: c.value }} />)}
                </div>
                <div><label className="text-sm font-medium">Logo URL (optional)</label><input aria-label="Logo URL" type="text" value={logoUrl} onChange={e => setLogoUrl(e.target.value)} placeholder="https://cdn.acme.com/logo.png" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>
              </div>}
              {wizardStep === 3 && <div className="space-y-3">
                <div><label className="text-sm font-medium">Custom Domain (optional)</label><input aria-label="Custom domain" type="text" value={customDomain} onChange={e => setCustomDomain(e.target.value)} placeholder="auth.acme.com" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div>
                <p className="text-xs text-gray-400">Add a CNAME record: <span className="font-mono">{customDomain || "auth.acme.com"} → cname.ggid.dev</span></p>
              </div>}
              {wizardStep === 4 && <div className="space-y-2">
                <p className="text-sm text-gray-500">Review your registration:</p>
                <div className="rounded-lg border dark:border-gray-700 p-4 text-sm space-y-1">
                  <div><span className="text-gray-400">Org:</span> {orgName || "—"}</div>
                  <div><span className="text-gray-400">Size:</span> {orgSize} · {industry}</div>
                  <div><span className="text-gray-400">Admin:</span> {adminName || "—"} ({adminEmail || "—"})</div>
                  <div className="flex items-center gap-2"><span className="text-gray-400">Theme:</span> <div className="h-4 w-4 rounded" style={{ backgroundColor: primaryColor }} /></div>
                  <div><span className="text-gray-400">Domain:</span> {customDomain || "default"}</div>
                </div>
              </div>}
            </div>
            <div className="mt-6 flex justify-between">
              <button onClick={() => setWizardStep(Math.max(0, wizardStep - 1))} disabled={wizardStep === 0} className="flex items-center gap-1 rounded-lg border border-gray-300 px-4 py-2 text-sm disabled:opacity-30 dark:border-gray-700"><ChevronLeft className="h-4 w-4" /> Back</button>
              {wizardStep < wizardSteps.length - 1 ? (
                <button onClick={() => setWizardStep(wizardStep + 1)} className="flex items-center gap-1 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700">Next <ChevronRight className="h-4 w-4" /></button>
              ) : (
                <button onClick={submitSignup} disabled={submitting} className="flex items-center gap-1 rounded-lg bg-green-600 px-4 py-2 text-sm font-medium text-white hover:bg-green-700 disabled:opacity-50">{submitting ? <Loader2 className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />} Create Tenant</button>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

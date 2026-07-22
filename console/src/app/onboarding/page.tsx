"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import {
  Building2, User, Mail, Lock, Eye, EyeOff, Loader2,
  Check, ArrowRight, ArrowLeft, Palette, Globe, Sparkles,
  AlertCircle, PartyPopper, ArrowDown,
} from "lucide-react";
import { API_BASE_URL } from "@/lib/api-config";
import { useTranslations } from "@/lib/i18n";

type Step = 1 | 2 | 3 | 4 | 5;

interface FormData {
  orgName: string;
  orgSize: string;
  industry: string;
  adminName: string;
  adminEmail: string;
  adminPassword: string;
  confirmPassword: string;
  primaryColor: string;
  logoUrl: string;
  customDomain: string;
}

const ORG_SIZES = ["1-10", "11-50", "51-200", "201-1000", "1000+"];
const INDUSTRIES = [
  "Technology", "Finance", "Healthcare", "Education", "Retail",
  "Manufacturing", "Government", "Other",
];

const DEFAULTS: FormData = {
  orgName: "", orgSize: "1-10", industry: "Technology",
  adminName: "", adminEmail: "", adminPassword: "", confirmPassword: "",
  primaryColor: "#4f46e5", logoUrl: "", customDomain: "",
};

export default function OnboardingPage() {
  const router = useRouter();
  const t = useTranslations();
  const [step, setStep] = useState<Step>(1);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [pwStrength, setPwStrength] = useState(0);
  const [result, setResult] = useState<{ tenant_id: string; org_name: string; admin_user_id: string; login_url: string } | null>(null);

  const [form, setForm] = useState<FormData>(DEFAULTS);

  // Password strength check
  useEffect(() => {
    if (!form.adminPassword) { setPwStrength(0); return; }
    let score = 0;
    if (form.adminPassword.length >= 12) score++;
    if (/[A-Z]/.test(form.adminPassword)) score++;
    if (/[0-9]/.test(form.adminPassword)) score++;
    if (/[^A-Za-z0-9]/.test(form.adminPassword)) score++;
    setPwStrength(score);
  }, [form.adminPassword]);

  const update = (k: keyof FormData, v: string) => setForm({ ...form, [k]: v });

  const validateStep = (): boolean => {
    setError("");
    if (step === 1) {
      if (!form.orgName.trim()) { setError("Organization name is required"); return false; }
    }
    if (step === 2) {
      if (!form.adminName.trim()) { setError("Your name is required"); return false; }
      if (!form.adminEmail.trim()) { setError("Email is required"); return false; }
      if (!/^[^@]+@[^@]+\.[^@]+$/.test(form.adminEmail)) { setError("Invalid email format"); return false; }
    }
    if (step === 3) {
      if (pwStrength < 2) { setError("Password is too weak — use 12+ chars with uppercase and numbers"); return false; }
      if (form.adminPassword !== form.confirmPassword) { setError("Passwords do not match"); return false; }
    }
    return true;
  };

  const next = () => {
    if (!validateStep()) return;
    if (step < 4) setStep((step + 1) as Step);
  };
  const back = () => { if (step > 1) setStep((step - 1) as Step); };

  const handleSubmit = async () => {
    setLoading(true);
    setError("");
    try {
      const resp = await fetch(`${API_BASE_URL}/api/v1/identity/tenants/self-register`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          org_name: form.orgName,
          org_size: form.orgSize,
          industry: form.industry,
          admin: {
            email: form.adminEmail,
            password: form.adminPassword,
            name: form.adminName,
          },
          branding: form.primaryColor !== DEFAULTS.primaryColor || form.logoUrl ? {
            primary_color: form.primaryColor,
            logo_url: form.logoUrl,
          } : undefined,
          custom_domain: form.customDomain || undefined,
        }),
      });
      const data = await resp.json();
      if (!resp.ok) {
        throw new Error(data.error?.message || data.error || data.detail || "Registration failed");
      }
      setResult(data);
      setStep(5);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Registration failed");
    } finally {
      setLoading(false);
    }
  };

  const inputCls = "w-full rounded-lg border border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 px-3 py-2.5 text-sm focus:border-brand-500 focus:outline-none focus:ring-2 focus:ring-brand-500/20";

  const steps = [
    { num: 1, label: "Organization" },
    { num: 2, label: "Admin Account" },
    { num: 3, label: "Security" },
    { num: 4, label: "Customize" },
    { num: 5, label: "Done" },
  ];

  return (
    <div className="flex min-h-screen items-center justify-center bg-gradient-to-br from-gray-50 via-white to-brand-50 dark:from-gray-950 dark:via-gray-900 dark:to-gray-950 p-4">
      <div className="w-full max-w-lg">
        {/* Header */}
        <div className="mb-6 text-center">
          <div className="mx-auto mb-3 flex h-12 w-12 items-center justify-center rounded-xl bg-brand-600 text-white text-xl font-bold shadow-lg shadow-brand-600/30">G</div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Create your GGID account</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Set up your organization in under 5 minutes</p>
        </div>

        {/* Progress bar */}
        {step < 5 && (
          <div className="mb-6 flex items-center justify-center gap-2">
            {steps.slice(0, 4).map((s) => (
              <div key={s.num} className="flex items-center">
                <div className={`flex h-7 w-7 items-center justify-center rounded-full text-xs font-bold transition ${
                  step >= s.num ? "bg-brand-600 text-white" : "bg-gray-200 dark:bg-gray-700 text-gray-400"
                }`}>
                  {step > s.num ? <Check className="h-3.5 w-3.5" /> : s.num}
                </div>
                {s.num < 4 && <div className={`h-0.5 w-8 ${step > s.num ? "bg-brand-600" : "bg-gray-200 dark:bg-gray-700"}`} />}
              </div>
            ))}
          </div>
        )}

        {/* Card */}
        <div className="rounded-2xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          {error && (
            <div className="mb-4 flex items-start gap-2 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950/30 dark:text-red-400">
              <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
              <span>{error}</span>
            </div>
          )}

          {/* Step 1: Organization */}
          {step === 1 && (
            <div className="space-y-4">
              <div className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white">
                <Building2 className="h-5 w-5 text-brand-600" /> Organization
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">Company name</label>
                <input value={form.orgName} onChange={(e) => update("orgName", e.target.value)} autoFocus className={inputCls} placeholder="Acme Corp" />
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">Organization size</label>
                <select value={form.orgSize} onChange={(e) => update("orgSize", e.target.value)} className={inputCls}>
                  {ORG_SIZES.map(s => <option key={s} value={s}>{s} employees</option>)}
                </select>
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">Industry</label>
                <select value={form.industry} onChange={(e) => update("industry", e.target.value)} className={inputCls}>
                  {INDUSTRIES.map(i => <option key={i} value={i}>{i}</option>)}
                </select>
              </div>
            </div>
          )}

          {/* Step 2: Admin Account */}
          {step === 2 && (
            <div className="space-y-4">
              <div className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white">
                <User className="h-5 w-5 text-brand-600" /> Admin Account
              </div>
              <p className="text-sm text-gray-500 dark:text-gray-400">You'll be the first administrator for your organization.</p>
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">Your name</label>
                <input value={form.adminName} onChange={(e) => update("adminName", e.target.value)} autoFocus className={inputCls} placeholder="Jane Doe" />
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">Work email</label>
                <div className="relative">
                  <Mail className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
                  <input value={form.adminEmail} onChange={(e) => update("adminEmail", e.target.value)} type="email" className={inputCls + " pl-10"} placeholder="jane@acme.com" />
                </div>
              </div>
            </div>
          )}

          {/* Step 3: Security */}
          {step === 3 && (
            <div className="space-y-4">
              <div className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white">
                <Lock className="h-5 w-5 text-brand-600" /> Security
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">Password</label>
                <input
                  type="password"
                  value={form.adminPassword}
                  onChange={(e) => update("adminPassword", e.target.value)}
                  autoFocus
                  className={inputCls}
                  placeholder="••••••••••••"
                />
                {/* Password strength bar */}
                <div className="mt-2 flex gap-1">
                  {[1, 2, 3, 4].map(n => (
                    <div key={n} className={`h-1.5 flex-1 rounded-full ${pwStrength >= n ? (n <= 2 ? "bg-red-500" : n === 3 ? "bg-amber-500" : "bg-green-500") : "bg-gray-200 dark:bg-gray-700"}`} />
                  ))}
                </div>
                <p className="mt-1 text-xs text-gray-400">12+ characters, with uppercase, numbers, and symbols</p>
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">Confirm password</label>
                <input
                  type="password"
                  value={form.confirmPassword}
                  onChange={(e) => update("confirmPassword", e.target.value)}
                  className={inputCls}
                  placeholder="••••••••••••"
                />
                {form.confirmPassword && form.adminPassword !== form.confirmPassword && (
                  <p className="mt-1 text-xs text-red-500">Passwords do not match</p>
                )}
              </div>
            </div>
          )}

          {/* Step 4: Customize */}
          {step === 4 && (
            <div className="space-y-4">
              <div className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white">
                <Palette className="h-5 w-5 text-brand-600" /> Customize (optional)
              </div>
              <p className="text-sm text-gray-500 dark:text-gray-400">Personalize your Console appearance. You can change this later.</p>
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">Primary color</label>
                <div className="flex items-center gap-3">
                  <input type="color" value={form.primaryColor} onChange={(e) => update("primaryColor", e.target.value)} className="h-10 w-16 rounded cursor-pointer" />
                  <code className="text-sm font-mono text-gray-500">{form.primaryColor}</code>
                </div>
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">Logo URL (optional)</label>
                <input value={form.logoUrl} onChange={(e) => update("logoUrl", e.target.value)} className={inputCls} placeholder="https://acme.com/logo.png" />
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">Custom domain (optional)</label>
                <div className="relative">
                  <Globe className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
                  <input value={form.customDomain} onChange={(e) => update("customDomain", e.target.value)} className={inputCls + " pl-10"} placeholder="auth.acme.com" />
                </div>
              </div>
            </div>
          )}

          {/* Step 5: Success */}
          {step === 5 && result && (
            <div className="space-y-4 text-center">
              <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-green-100 dark:bg-green-950/40">
                <PartyPopper className="h-8 w-8 text-green-600 dark:text-green-400" />
              </div>
              <div>
                <h2 className="text-xl font-bold text-gray-900 dark:text-white">Welcome aboard, {form.adminName}!</h2>
                <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Your organization <strong>{result.org_name}</strong> is ready.</p>
              </div>
              <div className="rounded-lg border border-gray-200 bg-gray-50 p-4 text-left dark:border-gray-700 dark:bg-gray-900">
                <div className="space-y-2 text-sm">
                  <div className="flex justify-between"><span className="text-gray-500 dark:text-gray-400">Organization</span><span className="font-medium text-gray-900 dark:text-white">{result.org_name}</span></div>
                  <div className="flex justify-between"><span className="text-gray-500 dark:text-gray-400">Tenant ID</span><code className="font-mono text-xs text-gray-700 dark:text-gray-300">{result.tenant_id}</code></div>
                  <div className="flex justify-between"><span className="text-gray-500 dark:text-gray-400">Admin email</span><span className="font-medium text-gray-900 dark:text-white">{form.adminEmail}</span></div>
                </div>
              </div>
              <div className="rounded-lg border border-amber-200 bg-amber-50 p-3 text-left text-xs text-amber-700 dark:border-amber-800 dark:bg-amber-950/30 dark:text-amber-400">
                <strong>Next steps:</strong> Log in with your email and password, then configure MFA and invite team members.
              </div>
              <button
                onClick={() => router.push(`/login?tenant=${result.tenant_id}`)}
                className="flex w-full items-center justify-center gap-2 rounded-lg bg-brand-600 px-4 py-2.5 text-sm font-medium text-white hover:bg-brand-700"
              >
                Go to Login <ArrowRight className="h-4 w-4" />
              </button>
            </div>
          )}

          {/* Navigation buttons */}
          {step < 5 && (
            <div className="mt-6 flex items-center justify-between">
              {step > 1 ? (
                <button onClick={back} className="flex items-center gap-1 text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200">
                  <ArrowLeft className="h-4 w-4" /> Back
                </button>
              ) : <div />}
              {step < 4 ? (
                <button onClick={next} className="flex items-center gap-2 rounded-lg bg-brand-600 px-5 py-2.5 text-sm font-medium text-white hover:bg-brand-700">
                  Continue <ArrowRight className="h-4 w-4" />
                </button>
              ) : (
                <button onClick={handleSubmit} disabled={loading} className="flex items-center gap-2 rounded-lg bg-brand-600 px-5 py-2.5 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50">
                  {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Sparkles className="h-4 w-4" />}
                  {loading ? "Creating..." : "Create Account"}
                </button>
              )}
            </div>
          )}
        </div>

        {/* Footer */}
        {step < 5 && (
          <p className="mt-4 text-center text-sm text-gray-500 dark:text-gray-400">
            Already have an account?{" "}
            <a href="/login" className="font-medium text-brand-600 hover:underline">Sign in</a>
          </p>
        )}
      </div>
    </div>
  );
}
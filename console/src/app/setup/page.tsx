"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useI18n } from "@/lib/i18n";

export default function SetupPage() {
  const router = useRouter();
  const { t } = useI18n();
  const [step, setStep] = useState(1);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [strength, setStrength] = useState<{score: number; warnings: string[]}>({score: 0, warnings: []});

  // Real-time password strength check via backend zxcvbn API
  useEffect(() => {
    if (!form.adminPassword) { setStrength({score: 0, warnings: []}); return; }
    const timer = setTimeout(async () => {
      try {
        const resp = await fetch("/api/v1/auth/password/strength", {
          method: "POST",
          headers: {"Content-Type": "application/json"},
          body: JSON.stringify({password: form.adminPassword}),
        });
        if (resp.ok) {
          const data = await resp.json();
          setStrength({score: data.score || 0, warnings: data.suggestions || []});
        }
      } catch {}
    }, 300);
    return () => clearTimeout(timer);
  }, [form.adminPassword]);

  const [form, setForm] = useState({
    adminUsername: "",
    adminEmail: "",
    adminPassword: "",
    confirmPassword: "",
    orgName: "",
  });

  const update = (k: string, v: string) => setForm({ ...form, [k]: v });

  const handleBootstrap = async () => {
    setLoading(true);
    setError("");
    try {
      const resp = await fetch("/api/v1/system/bootstrap", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          admin_username: form.adminUsername,
          admin_email: form.adminEmail,
          admin_password: form.adminPassword,
          tenant_name: form.orgName,
        }),
      });
      const data = await resp.json();
      if (!resp.ok && resp.status !== 200) {
        throw new Error(data.error?.message || data.error || data.detail || "Setup failed");
      }

      // Store tokens
      if (data.access_token) {
        localStorage.setItem("ggid_access_token", data.access_token);
        if (data.refresh_token) localStorage.setItem("ggid_refresh_token", data.refresh_token);

        // Parse scopes from JWT
        try {
          const payload = JSON.parse(atob(data.access_token.split(".")[1]));
          if (payload.tenant_id) localStorage.setItem("ggid_tenant_id", payload.tenant_id);
          if (payload.sub) localStorage.setItem("ggid_user_id", payload.sub);
          const scopes = payload.scopes || ["user"];
          localStorage.setItem("ggid_user_scopes", JSON.stringify(scopes));
        } catch {}
      }

      setStep(4); // Success step
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Setup failed");
    } finally {
      setLoading(false);
    }
  };

  const next = () => {
    setError("");
    if (step === 1) {
      if (!form.orgName.trim()) return setError("Organization name is required");
      setStep(2);
    } else if (step === 2) {
      if (!form.adminUsername.trim()) return setError("Username is required");
      if (!form.adminEmail.trim()) return setError("Email is required");
      if (!/^[^@]+@[^@]+\.[^@]+$/.test(form.adminEmail)) return setError("Invalid email format");
      setStep(3);
    } else if (step === 3) {
      if (strength.score < 2) return setError("Password is too weak. Please use a stronger password.");
      if (form.adminPassword !== form.confirmPassword) return setError("Passwords do not match");
      handleBootstrap();
    }
  };

  const finish = () => router.push("/dashboard");

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 to-slate-100 dark:from-slate-900 dark:to-slate-800 flex items-center justify-center p-4">
      <div className="w-full max-w-lg">
        {/* Logo */}
        <div className="text-center mb-8">
          <div className="inline-flex h-12 w-12 items-center justify-center rounded-xl bg-indigo-600 text-white font-bold text-xl mb-3">G</div>
          <h1 className="text-2xl font-bold text-slate-900 dark:text-white">GGID Setup</h1>
          <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">Initialize your identity platform</p>
        </div>

        {/* Progress */}
        <div className="flex items-center justify-center gap-2 mb-8">
          {[1, 2, 3, 4].map((s) => (
            <div key={s} className={`h-2 w-12 rounded-full transition-colors ${
              s <= step ? "bg-indigo-600" : "bg-slate-200 dark:bg-slate-700"
            }`} />
          ))}
        </div>

        <div className="bg-white dark:bg-slate-800 rounded-2xl shadow-xl p-8 border border-slate-200 dark:border-slate-700">
          {error && (
            <div className="mb-4 rounded-lg bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-800 px-4 py-3 text-sm text-red-700 dark:text-red-300">
              {error}
            </div>
          )}

          {step === 1 && (
            <div className="space-y-4">
              <h2 className="text-lg font-semibold text-slate-900 dark:text-white">Organization</h2>
              <p className="text-sm text-slate-500 dark:text-slate-400">Name your organization. This is your top-level tenant.</p>
              <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1.5">Organization Name</label>
                <input
                  type="text" value={form.orgName} onChange={(e) => update("orgName", e.target.value)}
                  placeholder="Acme Corporation"
                  className="w-full rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-900 px-4 py-2.5 text-sm text-slate-900 dark:text-white focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
                  autoFocus
                />
              </div>
            </div>
          )}

          {step === 2 && (
            <div className="space-y-4">
              <h2 className="text-lg font-semibold text-slate-900 dark:text-white">Administrator Account</h2>
              <p className="text-sm text-slate-500 dark:text-slate-400">This account will have full platform admin access.</p>
              <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1.5">Username</label>
                <input
                  type="text" value={form.adminUsername} onChange={(e) => update("adminUsername", e.target.value)}
                  placeholder="superadmin"
                  className="w-full rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-900 px-4 py-2.5 text-sm text-slate-900 dark:text-white focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
                  autoFocus
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1.5">Email</label>
                <input
                  type="email" value={form.adminEmail} onChange={(e) => update("adminEmail", e.target.value)}
                  placeholder="admin@acme.com"
                  className="w-full rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-900 px-4 py-2.5 text-sm text-slate-900 dark:text-white focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
                />
              </div>
            </div>
          )}

          {step === 3 && (
            <div className="space-y-4">
              <h2 className="text-lg font-semibold text-slate-900 dark:text-white">Set Password</h2>
              <p className="text-sm text-slate-500 dark:text-slate-400">Choose a strong password for the admin account.</p>
              <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1.5">Password</label>
                <input
                  type="password" value={form.adminPassword} onChange={(e) => update("adminPassword", e.target.value)}
                  className="w-full rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-900 px-4 py-2.5 text-sm text-slate-900 dark:text-white focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
                  autoFocus
                />
              </div>
              {/* Password strength meter — real zxcvbn score from backend */}
              {form.adminPassword && (
                <div className="space-y-2">
                  <div className="flex gap-1">
                    {[0,1,2,3].map((i) => (
                      <div key={i} className={`h-1.5 flex-1 rounded-full transition-colors ${
                        i < strength.score
                          ? strength.score <= 1 ? "bg-red-500" : strength.score === 2 ? "bg-yellow-500" : strength.score === 3 ? "bg-blue-500" : "bg-green-500"
                          : "bg-slate-200 dark:bg-slate-700"
                      }`} />
                    ))}
                  </div>
                  <p className={`text-xs font-medium ${
                    strength.score >= 2 ? "text-green-600 dark:text-green-400" : "text-red-500"
                  }`}>
                    {strength.score >= 2 ? "✓ Password strength: Good" : "⚠ Password too weak — score " + strength.score + "/4"}
                  </p>
                  {strength.warnings.length > 0 && (
                    <ul className="space-y-1 text-xs text-slate-500 dark:text-slate-400">
                      {strength.warnings.slice(0, 3).map((w, i) => (
                        <li key={i}>• {w}</li>
                      ))}
                    </ul>
                  )}
                </div>
              )}
              <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1.5">Confirm Password</label>
                <input
                  type="password" value={form.confirmPassword} onChange={(e) => update("confirmPassword", e.target.value)}
                  className={`w-full rounded-lg border px-4 py-2.5 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-transparent ${
                    form.confirmPassword && form.confirmPassword !== form.adminPassword
                      ? "border-red-400 dark:border-red-800 bg-red-50 dark:bg-red-900/20 text-slate-900 dark:text-white"
                      : "border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-900 text-slate-900 dark:text-white"
                  }`}
                />
                {form.confirmPassword && form.confirmPassword !== form.adminPassword && (
                  <p className="mt-1 text-xs text-red-500">Passwords do not match</p>
                )}
              </div>
              {loading && <p className="text-sm text-indigo-600 dark:text-indigo-400">Initializing system...</p>}
            </div>
          )}

          {step === 4 && (
            <div className="text-center space-y-4 py-4">
              <div className="inline-flex h-16 w-16 items-center justify-center rounded-full bg-green-100 dark:bg-green-900/30">
                <svg className="h-8 w-8 text-green-600 dark:text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="2">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                </svg>
              </div>
              <h2 className="text-lg font-semibold text-slate-900 dark:text-white">Setup Complete!</h2>
              <p className="text-sm text-slate-500 dark:text-slate-400">
                Your GGID instance is ready. You are logged in as <strong>{form.adminUsername}</strong>.
              </p>
            </div>
          )}

          {/* Buttons */}
          {step < 4 && (
            <div className="flex gap-3 mt-6">
              {step > 1 && (
                <button
                  onClick={() => setStep(step - 1)}
                  className="px-4 py-2.5 text-sm font-medium text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700 rounded-lg transition-colors"
                >
                  Back
                </button>
              )}
              <button
                onClick={next}
                disabled={loading}
                className="flex-1 px-4 py-2.5 text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {loading ? "Initializing..." : step === 3 ? "Complete Setup" : "Continue"}
              </button>
            </div>
          )}

          {step === 4 && (
            <button
              onClick={finish}
              className="w-full px-4 py-2.5 text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 rounded-lg transition-colors"
            >
              Go to Dashboard
            </button>
          )}
        </div>
      </div>
    </div>
  );
}

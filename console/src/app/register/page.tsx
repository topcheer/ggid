"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { ArrowLeft, UserPlus, Eye, EyeOff } from "lucide-react";
import { API_BASE_URL, DEFAULT_TENANT_ID } from "@/lib/api-config";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

export default function RegisterPage() {
  const router = useRouter();
  const t = useTranslations();
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [pwFeedback, setPwFeedback] = useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError("");
    try {
      const resp = await fetch(`${API_BASE_URL}/api/v1/auth/register`, {
        method: "POST",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": DEFAULT_TENANT_ID },
        body: JSON.stringify({ username, email, password }),
      });
      const data = await resp.json();
      if (resp.ok) {
        router.push("/login?registered=1");
      } else {
        const errMsg = typeof data.error === 'string'
          ? data.error
          : data.error?.message || data.error?.code || data.message || "Registration failed";
        setError(errMsg);
      }
    } catch {
      setError("Network error — is the API running?");
    } finally {
      setLoading(false);
    }
  };

  const inputCls = "w-full rounded-lg border border-gray-300 px-3 py-2 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 focus:border-brand-500 focus:outline-none";

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50 dark:bg-gray-950">
      <div className="w-full max-w-md">
        <div className="mb-8 text-center">
          <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-xl bg-brand-600 text-white text-xl font-bold">G</div>
          <h1 className="text-2xl font-bold dark:text-gray-100">{t("register.title")}</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("register.getStarted")}</p>
        </div>

        <form onSubmit={handleSubmit} className="rounded-xl border border-gray-200 bg-white p-8 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          {error && <div className="mb-4 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400">{error}</div>}

          <div className="mb-4">
            <label className="mb-1 block text-sm font-medium dark:text-gray-300">{t("register.username")}</label>
            <input aria-label="johndoe" value={username} onChange={(e) => setUsername(e.target.value)} required autoFocus className={inputCls} placeholder="johndoe" />
          </div>
          <div className="mb-4">
            <label className="mb-1 block text-sm font-medium dark:text-gray-300">{t("register.email")}</label>
            <input autoComplete="email" value={email} onChange={(e) => setEmail(e.target.value)} type="email" required className={inputCls} placeholder="you@example.com" />
          </div>
          <div className="mb-6">
            <label className="mb-1 block text-sm font-medium dark:text-gray-300">{t("register.password")}</label>
            <div className="relative">
              <input autoComplete="new-password" value={password} onChange={(e) => {
                const pw = e.target.value; setPassword(pw);
                if (pw.length === 0) setPwFeedback("");
                else if (pw.length < 12) setPwFeedback("Minimum 12 characters required");
                else if (!/[A-Z]/.test(pw) || !/[0-9]/.test(pw)) setPwFeedback("Add uppercase letters and numbers for stronger security");
                else setPwFeedback("Strong password");
              }} type={showPassword ? "text" : "password"} required className={inputCls + " pr-10"} placeholder="••••••••" />
              <button type="button" onClick={() => setShowPassword(!showPassword)} aria-label={showPassword ? "Hide password" : "Show password"} className="absolute right-2 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300">
                {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </button>
            </div>
            {pwFeedback && (
              <p className={`mt-1 text-xs ${pwFeedback === "Strong password" ? "text-green-600 dark:text-green-400" : "text-amber-600 dark:text-amber-400"}`}>{pwFeedback}</p>
            )}
          </div>

          <button type="submit" disabled={loading} className="flex w-full items-center justify-center gap-2 rounded-lg bg-brand-600 px-4 py-2.5 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50" aria-label="UserPlus">
            {loading ? t("register.creating") : <><UserPlus className="h-4 w-4" /> {t("register.createAccount")}</>}
          </button>

          <p className="mt-4 text-center text-sm text-gray-500 dark:text-gray-400">
            {t("register.alreadyHaveAccount")}{" "}
            <a href="/login" className="font-medium text-brand-600 hover:underline">{t("register.signIn")}</a>
          </p>
        </form>
      </div>
    </div>
  );
}

"use client";

import { useState } from "react";
import { ArrowLeft, Mail } from "lucide-react";
import { API_BASE_URL, DEFAULT_TENANT_ID } from "@/lib/api-config";
import { useTranslations } from "@/lib/i18n";

export default function ForgotPasswordPage() {
  const t = useTranslations();
  const [email, setEmail] = useState("");
  const [sent, setSent] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError("");
    try {
      const resp = await fetch(`${API_BASE_URL}/api/v1/auth/password/reset`, {
        method: "POST",
        headers: { "Content-Type": "application/json", "X-Tenant-ID": DEFAULT_TENANT_ID },
        body: JSON.stringify({ email }),
      });
      if (resp.ok) {
        setSent(true);
      } else {
        const data = await resp.json().catch(() => ({}));
        setError(data.error || data.message || "Reset failed");
      }
    } catch {
      setError("Network error — is the API running?");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50 dark:bg-gray-950">
      <div className="w-full max-w-md">
        <div className="mb-8 text-center">
          <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-xl bg-brand-600 text-white text-xl font-bold">G</div>
          <h1 className="text-2xl font-bold dark:text-gray-100">Reset Password</h1>
        </div>

        {sent ? (
          <div className="rounded-xl border border-gray-200 bg-white p-8 shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <div className="text-center">
              <Mail className="mx-auto mb-3 h-10 w-10 text-brand-600" />
              <p className="text-sm text-gray-600 dark:text-gray-300">If an account exists for <strong>{email}</strong>, a password reset link has been sent.</p>
            </div>
            <a href="/login" className="mt-6 flex items-center justify-center gap-1 text-sm text-brand-600 hover:underline">
              <ArrowLeft className="h-4 w-4" /> Back to Login
            </a>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="rounded-xl border border-gray-200 bg-white p-8 shadow-sm dark:border-gray-700 dark:bg-gray-800">
            {error && <div className="mb-4 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400">{error}</div>}
            <p className="mb-4 text-sm text-gray-500 dark:text-gray-400">Enter your email address and we&apos;ll send you a link to reset your password.</p>
            <div className="mb-6">
              <label className="mb-1 block text-sm font-medium dark:text-gray-300">Email</label>
              <input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
                autoFocus
                className="w-full rounded-lg border border-gray-300 px-3 py-2 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 focus:border-brand-500 focus:outline-none"
                placeholder="you@example.com"
              />
            </div>
            <button type="submit" disabled={loading} className="w-full rounded-lg bg-brand-600 px-4 py-2.5 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50">
              {loading ? "Sending..." : "Send Reset Link"}
            </button>
            <a href="/login" className="mt-4 flex items-center justify-center gap-1 text-sm text-gray-500 hover:text-gray-700">
              <ArrowLeft className="h-4 w-4" /> Back to Login
            </a>
          </form>
        )}
      </div>
    </div>
  );
}

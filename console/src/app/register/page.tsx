"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { ArrowLeft, UserPlus } from "lucide-react";
import { API_BASE_URL, DEFAULT_TENANT_ID } from "@/lib/api-config";
import { useTranslations } from "@/lib/i18n";

export default function RegisterPage() {
  const router = useRouter();
  const t = useTranslations();
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError("");
    try {
      const resp = await fetch(`${API_BASE_URL}/api/v1/auth/register`, {
        method: "POST",
        headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "Content-Type": "application/json", "X-Tenant-ID": DEFAULT_TENANT_ID },
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
          <h1 className="text-2xl font-bold dark:text-gray-100">Create Account</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Get started with GGID</p>
        </div>

        <form onSubmit={handleSubmit} className="rounded-xl border border-gray-200 bg-white p-8 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          {error && <div className="mb-4 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400">{error}</div>}

          <div className="mb-4">
            <label className="mb-1 block text-sm font-medium dark:text-gray-300">Username</label>
            <input aria-label="johndoe" value={username} onChange={(e) => setUsername(e.target.value)} required autoFocus className={inputCls} placeholder="johndoe" />
          </div>
          <div className="mb-4">
            <label className="mb-1 block text-sm font-medium dark:text-gray-300">Email</label>
            <input autoComplete="email" value={email} onChange={(e) => setEmail(e.target.value)} type="email" required className={inputCls} placeholder="you@example.com" />
          </div>
          <div className="mb-6">
            <label className="mb-1 block text-sm font-medium dark:text-gray-300">Password</label>
            <input autoComplete="new-password" value={password} onChange={(e) => setPassword(e.target.value)} type="password" required className={inputCls} placeholder="••••••••" />
          </div>

          <button type="submit" disabled={loading} className="flex w-full items-center justify-center gap-2 rounded-lg bg-brand-600 px-4 py-2.5 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50" aria-label="UserPlus">
            {loading ? "Creating..." : <><UserPlus className="h-4 w-4" /> Create Account</>}
          </button>

          <p className="mt-4 text-center text-sm text-gray-500 dark:text-gray-400">
            Already have an account?{" "}
            <a href="/login" className="font-medium text-brand-600 hover:underline">Sign In</a>
          </p>
        </form>
      </div>
    </div>
  );
}

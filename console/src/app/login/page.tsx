"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

const API_BASE = process.env.NEXT_PUBLIC_GGID_API || "http://localhost:8080";
const TENANT_ID =
  process.env.NEXT_PUBLIC_TENANT_ID || "00000000-0000-0000-0000-000000000001";

export default function LoginPage() {
  const router = useRouter();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError("");

    try {
      const resp = await fetch(`${API_BASE}/api/v1/auth/login`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-Tenant-ID": TENANT_ID,
        },
        body: JSON.stringify({ username, password }),
      });

      const data = await resp.json();

      if (!resp.ok) {
        setError(data.error || "Login failed");
        return;
      }

      if (typeof window !== "undefined") {
        localStorage.setItem("ggid_access_token", data.access_token);
        localStorage.setItem("ggid_refresh_token", data.refresh_token);
        localStorage.setItem("ggid_session_id", data.session_id);
      }

      router.push("/");
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Network error — is the API running?",
      );
    } finally {
      setLoading(false);
    }
  };

  const handleSocialLogin = async (provider: string) => {
    setError("");
    try {
      const resp = await fetch(
        `${API_BASE}/api/v1/auth/social/${provider}?redirect_uri=/`,
        {
          headers: { "X-Tenant-ID": TENANT_ID },
        },
      );
      const data = await resp.json();
      if (data.auth_url) {
        window.location.href = data.auth_url;
      }
    } catch {
      setError(`${provider} login not configured`);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50">
      <div className="w-full max-w-md">
        <div className="mb-8 text-center">
          <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-xl bg-brand-600 text-white text-xl font-bold">
            G
          </div>
          <h1 className="text-2xl font-bold">GGID Console</h1>
          <p className="mt-1 text-sm text-gray-500">
            Identity &amp; Access Management
          </p>
        </div>

        <form
          onSubmit={handleSubmit}
          className="rounded-xl border border-gray-200 bg-white p-8 shadow-sm"
        >
          {error && (
            <div className="mb-4 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
              {error}
            </div>
          )}

          <div className="mb-4">
            <label className="mb-1 block text-sm font-medium">Username</label>
            <input
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              required
              className="w-full rounded-lg border border-gray-300 px-3 py-2 focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500"
              placeholder="admin"
            />
          </div>

          <div className="mb-6">
            <label className="mb-1 block text-sm font-medium">Password</label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              className="w-full rounded-lg border border-gray-300 px-3 py-2 focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500"
              placeholder="••••••••••••"
            />
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full rounded-lg bg-brand-600 px-4 py-2.5 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
          >
            {loading ? "Signing in..." : "Sign In"}
          </button>

          {/* Social Login */}
          <div className="my-5 flex items-center gap-3">
            <div className="h-px flex-1 bg-gray-200" />
            <span className="text-xs text-gray-400">or continue with</span>
            <div className="h-px flex-1 bg-gray-200" />
          </div>

          <div className="grid grid-cols-3 gap-2">
            <button
              type="button"
              onClick={() => handleSocialLogin("google")}
              className="flex items-center justify-center rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
            >
              Google
            </button>
            <button
              type="button"
              onClick={() => handleSocialLogin("github")}
              className="flex items-center justify-center rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
            >
              GitHub
            </button>
            <button
              type="button"
              onClick={() => handleSocialLogin("oidc")}
              className="flex items-center justify-center rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
            >
              SSO
            </button>
          </div>

          {/* Default credentials hint */}
          <div className="mt-5 rounded-lg bg-blue-50 px-3 py-2 text-center text-xs text-blue-600">
            Default: admin / Admin@123456
          </div>
        </form>

        <p className="mt-4 text-center text-xs text-gray-400">
          GGID IAM Suite · Apache 2.0
        </p>
      </div>
    </div>
  );
}

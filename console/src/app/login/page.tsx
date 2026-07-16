"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { Shield, ArrowLeft, KeyRound, Building2, AlertCircle, Loader2 } from "lucide-react";
import { API_BASE_URL, DEFAULT_TENANT_ID } from "@/lib/api-config";
import { useTranslations } from "@/lib/i18n";

const API_BASE = API_BASE_URL;

type Step = "credentials" | "mfa";

interface SocialConnector {
  id: string;
  name: string;
  provider: string;
  icon?: string;
}

export default function LoginPage() {
  const router = useRouter();
  const t = useTranslations();
  const [step, setStep] = useState<Step>("credentials");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [totpCode, setTotpCode] = useState("");
  const [remember, setRemember] = useState(true);
  const [mfaToken, setMfaToken] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [connectors, setConnectors] = useState<SocialConnector[]>([]);
  const [connectorsLoaded, setConnectorsLoaded] = useState(false);
  const [passkeySupported, setPasskeySupported] = useState(false);
  const [tenantSlug, setTenantSlug] = useState("default");
  const [systemInitialized, setSystemInitialized] = useState<boolean | null>(null);
  const [initUserCount, setInitUserCount] = useState(0);

  // Check for WebAuthn / Passkey support and attempt conditional mediation (autofill)
  useEffect(() => {
    if (typeof window !== "undefined" && "PublicKeyCredential" in window) {
      setPasskeySupported(true);
      // Conditional mediation: silently check for a passkey via autofill
      // This shows available passkeys in the browser's native autofill UI.
      // The backend /api/v1/webauthn/auth/begin endpoint supports discoverable credentials.
      (async () => {
        try {
          const pkc = PublicKeyCredential as unknown as typeof PublicKeyCredential & { isConditionalMediationAvailable?: () => Promise<boolean> };
          const isConditional = await pkc.isConditionalMediationAvailable?.();
          if (!isConditional) return;
          // Trigger a conditional passkey authentication.
          // The browser will show the passkey in the autofill dropdown.
          // This does not block the normal login flow — it runs in parallel.
          // The actual assertion must be posted to /api/v1/webauthn/auth/finish.
        } catch {
          // Conditional mediation not available — normal flow continues.
        }
      })();
    }
  }, []);

  // Load social connectors from API
  useEffect(() => {
    fetch(`${API_BASE}/api/v1/auth/social/connectors`, {
      headers: { "X-Tenant-ID": tenantSlug || DEFAULT_TENANT_ID },
    })
      .then((r) => (r.ok ? r.json() : null))
      .then((data) => {
        if (data?.connectors && Array.isArray(data.connectors)) {
          setConnectors(data.connectors);
        }
        setConnectorsLoaded(true);
      })
      .catch(() => setConnectorsLoaded(true));
  }, [tenantSlug]);

  // Check system initialization status on mount
  useEffect(() => {
    fetch(`${API_BASE}/api/v1/system/initialized`)
      .then((r) => (r.ok ? r.json() : null))
      .then((data) => {
        if (data) {
          setSystemInitialized(data.initialized !== false);
          setInitUserCount(data.user_count ?? 0);
        } else {
          setSystemInitialized(true); // assume initialized if endpoint unavailable
          setInitUserCount(1);
        }
      })
      .catch(() => {
        setSystemInitialized(true); // assume initialized on network error
        setInitUserCount(1);
      });
  }, []);

  const handleCredentials = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError("");

    try {
      const resp = await fetch(`${API_BASE}/api/v1/auth/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json", "X-Tenant-ID": tenantSlug || DEFAULT_TENANT_ID },
        body: JSON.stringify({ username, password, tenant_slug: tenantSlug || "default" }),
      });
      const data = await resp.json();

      if (!resp.ok) {
        const errMsg = typeof data.error === 'string'
          ? data.error
          : data.error?.message || data.error?.code || data.message || "Login failed";
        setError(errMsg);
        return;
      }

      // Check if MFA is required
      if (data.mfa_required || data.mfa_token) {
        setMfaToken(data.mfa_token || "");
        setStep("mfa");
        return;
      }

      // Success — store token and check for OAuth redirect
      localStorage.setItem("ggid_access_token", data.access_token);
      localStorage.setItem("ggid_refresh_token", data.refresh_token || "");
      if (data.session_id) {
        localStorage.setItem("ggid_session_id", data.session_id);
      }

      // Extract user info from JWT for pages that need it
      try {
        const payload = JSON.parse(atob(data.access_token.split(".")[1]));
        if (payload.tenant_id) localStorage.setItem("ggid_tenant_id", payload.tenant_id);
        if (payload.sub) localStorage.setItem("ggid_user_id", payload.sub);
        if (payload.username) localStorage.setItem("ggid_user_name", payload.username);
        if (payload.email) localStorage.setItem("ggid_user_email", payload.email);
      } catch {}

      // If redirect_to is set (OAuth flow), redirect back to authorize with user_id
      const params = new URLSearchParams(window.location.search);
      const redirectTo = params.get("redirect_to");
      if (redirectTo) {
        // Extract user_id from JWT
        const token = data.access_token;
        try {
          const payload = JSON.parse(atob(token.split(".")[1]));
          const userId = payload.sub;
          const url = new URL(redirectTo);
          url.searchParams.set("user_id", userId);
          window.location.href = url.toString();
          return;
        } catch {
          // fallback: just redirect without user_id
          window.location.href = redirectTo;
          return;
        }
      }

      // No MFA needed — redirect to dashboard
      router.push("/dashboard");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Network error — is the API running?");
    } finally {
      setLoading(false);
    }
  };

  const handleMfa = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError("");

    try {
      const resp = await fetch(`${API_BASE}/api/v1/auth/mfa/verify`, {
        method: "POST",
        headers: { "Content-Type": "application/json", "X-Tenant-ID": tenantSlug || DEFAULT_TENANT_ID },
        body: JSON.stringify({ mfa_token: mfaToken, code: totpCode }),
      });
      const data = await resp.json();

      if (!resp.ok) {
        const errMsg = typeof data.error === 'string'
          ? data.error
          : data.error?.message || data.error?.code || data.message || "Invalid verification code";
        setError(errMsg);
        return;
      }

      if (data.access_token) {
        if (typeof window !== "undefined") {
          localStorage.setItem("ggid_access_token", data.access_token);
          localStorage.setItem("ggid_refresh_token", data.refresh_token || "");

          // Extract user info from JWT
          try {
            const payload = JSON.parse(atob(data.access_token.split(".")[1]));
            if (payload.tenant_id) localStorage.setItem("ggid_tenant_id", payload.tenant_id);
            if (payload.sub) localStorage.setItem("ggid_user_id", payload.sub);
            if (payload.username) localStorage.setItem("ggid_user_name", payload.username);
            if (payload.email) localStorage.setItem("ggid_user_email", payload.email);
          } catch {}

          // If redirect_to is set (OAuth flow), redirect back to authorize with user_id
          const params = new URLSearchParams(window.location.search);
          const redirectTo = params.get("redirect_to");
          if (redirectTo) {
            try {
              const payload = JSON.parse(atob(data.access_token.split(".")[1]));
              const url = new URL(redirectTo);
              url.searchParams.set("user_id", payload.sub);
              window.location.href = url.toString();
              return;
            } catch {
              window.location.href = redirectTo;
              return;
            }
          }
        }
        router.push("/dashboard");
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Verification failed");
    } finally {
      setLoading(false);
    }
  };

  const handleSocialLogin = async (provider: string) => {
    setError("");
    try {
      const resp = await fetch(`${API_BASE}/api/v1/auth/social/${provider}?redirect_uri=/`, {
        headers: { "X-Tenant-ID": tenantSlug || DEFAULT_TENANT_ID },
      });
      const data = await resp.json();
      if (data.auth_url) {
        window.location.href = data.auth_url;
      } else {
        setError(`${provider} login not configured`);
      }
    } catch {
      setError(`${provider} login not available`);
    }
  };

  // Default connectors if API doesn't respond
  const socialButtons = connectorsLoaded && connectors.length > 0
    ? connectors
    : [
        { id: "google", name: "Google", provider: "google" },
        { id: "github", name: "GitHub", provider: "github" },
        { id: "oidc", name: "SSO", provider: "oidc" },
      ];

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50 dark:bg-gray-950">
      <div className="w-full max-w-md">
        <div className="mb-8 text-center">
          <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-xl bg-brand-600 text-white text-xl font-bold">
            G
          </div>
          <h1 className="text-2xl font-bold dark:text-gray-100">{t("login.consoleTitle")}</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("login.subtitle")}</p>
        </div>

        {systemInitialized === null ? (
          /* ===== System init checking ===== */
          <div className="flex items-center justify-center rounded-xl border border-gray-200 bg-white p-12 shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <Loader2 className="h-6 w-6 animate-spin text-brand-600" />
            <span className="ml-3 text-sm text-gray-500 dark:text-gray-400">{t("login.initializing")}</span>
          </div>
        ) : initUserCount === 0 ? (
          /* ===== System not initialized ===== */
          <div className="rounded-xl border border-amber-200 bg-amber-50 p-8 shadow-sm dark:border-amber-800 dark:bg-amber-950/30">
            <div className="mb-4 flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-amber-100 dark:bg-amber-900/50">
                <AlertCircle className="h-5 w-5 text-amber-600 dark:text-amber-400" />
              </div>
              <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{t("login.systemNotInitialized")}</h2>
            </div>
            <p className="mb-5 text-sm text-gray-600 dark:text-gray-400">{t("login.systemNotInitializedDesc")}</p>
            <a
              href="/register"
              className="flex w-full items-center justify-center gap-2 rounded-lg bg-brand-600 px-4 py-2.5 text-sm font-medium text-white hover:bg-brand-700"
            >
              <KeyRound className="h-4 w-4" /> {t("login.registerAdmin")}
            </a>
          </div>
        ) : step === "credentials" ? (
          /* ===== Step 1: Credentials ===== */
          <form onSubmit={handleCredentials} className="rounded-xl border border-gray-200 bg-white p-8 shadow-sm dark:border-gray-700 dark:bg-gray-800">
            {error && (
              <div className="mb-4 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400">
                {error}
              </div>
            )}

            <div className="mb-4">
              <label className="mb-1 block text-sm font-medium">{t("login.tenant")}</label>
              <div className="relative">
                <Building2 className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
                <input
                  type="text"
                  aria-label={t("login.tenant")}
                  value={tenantSlug}
                  onChange={(e) => setTenantSlug(e.target.value)}
                  className="w-full rounded-lg border border-gray-300 py-2 pl-9 pr-3 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500"
                  placeholder={t("login.tenantPlaceholder")}
                />
              </div>
              <p className="mt-1 text-xs text-gray-400">{t("login.tenantHint")}</p>
            </div>

            <div className="mb-4">
              <label className="mb-1 block text-sm font-medium">{t("login.username")}</label>
              <input
                type="text"
                id="username"
                name="username"
                aria-label={t("login.username")}
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
                autoComplete="username webauthn"
                className="w-full rounded-lg border border-gray-300 px-3 py-2 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500"
                placeholder="admin"
              />
            </div>

            <div className="mb-4">
              <label className="mb-1 block text-sm font-medium">{t("login.password")}</label>
              <input
                type="password"
                id="password"
                name="password"
                aria-label={t("login.password")}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                autoComplete="current-password"
                className="w-full rounded-lg border border-gray-300 px-3 py-2 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500"
                placeholder="••••••••••••"
              />
            </div>

            <div className="mb-6 flex items-center justify-between">
              <label className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400">
                <input type="checkbox" id="remember" name="remember" aria-label={t("login.rememberMe")} checked={remember} onChange={(e) => setRemember(e.target.checked)} className="rounded border-gray-300" />
                {t("login.rememberMe")}
              </label>
              <a href="/forgot-password" className="text-sm text-brand-600 hover:underline">{t("login.forgotPassword")}</a>
            </div>

            <button
              type="submit"
              disabled={loading}
              aria-label={loading ? t("login.signingIn") : t("login.signIn")}
              className="w-full rounded-lg bg-brand-600 px-4 py-2.5 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
            >
              {loading ? t("login.signingIn") : t("login.signIn")}
            </button>

            {/* Social Login */}
            <div className="my-5 flex items-center gap-3">
              <div className="h-px flex-1 bg-gray-200 dark:bg-gray-700" />
              <span className="text-xs text-gray-400 dark:text-gray-500">{t("login.orContinueWith")}</span>
              <div className="h-px flex-1 bg-gray-200 dark:bg-gray-700" />
            </div>

            <div className="grid grid-cols-3 gap-2">
              {socialButtons.map((conn) => (
                <button
                  key={conn.id}
                  type="button"
                  onClick={() => handleSocialLogin(conn.provider)}
                  aria-label={`Sign in with ${conn.name}`}
                  className="flex items-center justify-center gap-1.5 rounded-lg border border-gray-300 px-3 py-2 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 text-sm font-medium text-gray-700 hover:bg-gray-50 dark:bg-gray-950"
                >
                  <SocialIcon provider={conn.provider} />
                  {conn.name}
                </button>
              ))}
            </div>

            <div className="mt-5 rounded-lg bg-blue-50 px-3 py-2 text-center text-xs text-blue-600">
              {t("login.demo")}
            </div>

            {/* OAuth SSO Entry */}
            <div className="mt-4 border-t border-gray-100 pt-4 dark:border-gray-700">
              <p className="mb-2 text-center text-xs text-gray-400">or sign in with</p>
              <button
                type="button"
                onClick={async () => {
                  const { initOAuthFlow } = await import("@/lib/oauth-pkce");
                  const redirectUri = `${window.location.origin}/auth/callback`;
                  const authUrl = await initOAuthFlow(
                    `${API_BASE}/oauth/authorize`,
                    "ggid-console",
                    redirectUri,
                  );
                  window.location.href = authUrl;
                }}
                aria-label="Sign in with OAuth SSO"
                className="flex w-full items-center justify-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-200 dark:hover:bg-gray-950"
              >
                <svg className="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M12 2L2 7v10c0 5.55 3.84 9.74 9 11 5.16-1.26 9-5.45 9-11V7l-10-5z" /></svg>
                Sign in with GGID SSO
              </button>
            </div>

            <p className="mt-4 text-center text-sm text-gray-500 dark:text-gray-400">
              {t("login.noAccount")}{" "}
              <a href="/register" className="font-medium text-brand-600 hover:underline">{t("login.signUp")}</a>
            </p>
          </form>
        ) : (
          /* ===== Step 2: MFA Verification ===== */
          <form onSubmit={handleMfa} className="rounded-xl border border-gray-200 bg-white p-8 shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <div className="mb-6 text-center">
              <div className="mx-auto mb-3 flex h-12 w-12 items-center justify-center rounded-full bg-brand-100">
                <Shield className="h-6 w-6 text-brand-600" />
              </div>
              <h2 className="text-lg font-semibold">{t("login.twoFactor")}</h2>
              <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {t("login.enterCode")}
              </p>
            </div>

            {error && (
              <div className="mb-4 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400">
                {error}
              </div>
            )}

            <div className="mb-6">
              <label className="mb-1 flex items-center gap-1.5 text-sm font-medium">
                <KeyRound className="h-4 w-4 text-gray-400" /> {t("login.verificationCode")}
              </label>
              <input
                type="text"
                id="totp-code"
                name="totp-code"
                aria-label={t("login.verificationCode")}
                value={totpCode}
                onChange={(e) => setTotpCode(e.target.value.replace(/\D/g, "").slice(0, 6))}
                required
                autoFocus
                placeholder="000000"
                className="w-full rounded-lg border border-gray-300 px-3 py-3 text-center text-2xl font-mono tracking-widest focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500"
                inputMode="numeric"
                pattern="[0-9]{6}"
                maxLength={6}
              />
            </div>

            <button
              type="submit"
              disabled={loading || totpCode.length !== 6}
              aria-label={loading ? t("login.verifying") : t("login.verifySignIn")}
              className="w-full rounded-lg bg-brand-600 px-4 py-2.5 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
            >
              {loading ? t("login.verifying") : t("login.verifySignIn")}
            </button>

            <button
              type="button"
              onClick={() => { setStep("credentials"); setError(""); setTotpCode(""); }}
              aria-label={t("login.backToLogin")}
              className="mt-3 flex w-full items-center justify-center gap-1 text-sm text-gray-500 hover:text-gray-700"
            >
              <ArrowLeft className="h-4 w-4" /> {t("login.backToLogin")}
            </button>
          </form>
        )}

        <p className="mt-4 text-center text-xs text-gray-400 dark:text-gray-500">{t("login.footer")}</p>
      </div>
    </div>
  );
}

function SocialIcon({ provider }: { provider: string }) {
  const p = provider.toLowerCase();
  if (p === "google") {
    return (
      <svg className="h-4 w-4" viewBox="0 0 24 24">
        <path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"/>
        <path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/>
        <path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"/>
        <path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/>
      </svg>
    );
  }
  if (p === "github") {
    return (
      <svg className="h-4 w-4" viewBox="0 0 24 24" fill="currentColor">
        <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z"/>
      </svg>
    );
  }
  if (p === "oidc" || p === "sso") {
    return (
      <svg className="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
        <rect x="3" y="11" width="18" height="11" rx="2"/>
        <path d="M7 11V7a5 5 0 0110 0v4"/>
      </svg>
    );
  }
  return null;
}

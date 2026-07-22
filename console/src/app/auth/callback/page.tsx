"use client";

import { Suspense, useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { API_BASE_URL, DEFAULT_TENANT_ID } from "@/lib/api-config";

function CallbackContent() {
  const router = useRouter();
  const params = useSearchParams();
  const [error, setError] = useState("");

  useEffect(() => {
    // Check for social login callback (token in URL fragment: #access_token=xxx)
    if (typeof window !== "undefined") {
      const hash = window.location.hash.substring(1); // Remove leading #
      const hashParams = new URLSearchParams(hash);
      const accessToken = hashParams.get("access_token");

      if (accessToken) {
        // Social login callback — token already issued by backend
        localStorage.setItem("ggid_access_token", accessToken);

        // Try to get refresh_token if present
        const refreshToken = hashParams.get("refresh_token");
        if (refreshToken) {
          localStorage.setItem("ggid_refresh_token", refreshToken);
        }

        // Extract tenant_id and scopes from JWT
        try {
          const payload = JSON.parse(atob(accessToken.split(".")[1]));
          if (payload.tenant_id) localStorage.setItem("ggid_tenant_id", payload.tenant_id);
          if (payload.sub) localStorage.setItem("ggid_user_id", payload.sub);
          const scopes = payload.scopes || payload.roles || ["user"];
          localStorage.setItem("ggid_user_scopes", JSON.stringify(Array.isArray(scopes) ? scopes : [scopes]));
        } catch {}

        // Redirect to dashboard
        const redirectTo = sessionStorage.getItem("ggid_redirect_after_login") || "/dashboard";
        sessionStorage.removeItem("ggid_redirect_after_login");
        router.replace(redirectTo);
        return;
      }
    }

    // Standard OAuth authorization_code + PKCE flow
    const code = params.get("code");
    const state = params.get("state");

    if (!code) {
      setError("No authorization code received");
      return;
    }

    // Retrieve PKCE flow state from sessionStorage.
    const flowStr = sessionStorage.getItem("ggid_oauth_flow");
    if (!flowStr) {
      setError("Missing PKCE flow state. Please try logging in again.");
      return;
    }

    const flow = JSON.parse(flowStr);

    // Verify state parameter matches (CSRF protection).
    if (state && flow.state && state !== flow.state) {
      setError("State mismatch — possible CSRF attack. Please try again.");
      sessionStorage.removeItem("ggid_oauth_flow");
      return;
    }

    // Exchange authorization code for tokens via OAuth token endpoint.
    const tokenBody = new URLSearchParams({
      grant_type: "authorization_code",
      code: code,
      redirect_uri: flow.redirect_uri,
      client_id: flow.client_id,
      code_verifier: flow.code_verifier,
    });

    fetch(`${API_BASE_URL}/oauth/token`, {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body: tokenBody.toString(),
    })
      .then(async (resp) => {
        const data = await resp.json();
        if (!resp.ok) {
          throw new Error(data.error_description || data.error || "Token exchange failed");
        }

        // Store tokens.
        localStorage.setItem("ggid_access_token", data.access_token);
        if (data.refresh_token) {
          localStorage.setItem("ggid_refresh_token", data.refresh_token);
        }

        // Clean up PKCE flow state.
        sessionStorage.removeItem("ggid_oauth_flow");

        // Redirect to the originally requested page or dashboard.
        const redirectTo = sessionStorage.getItem("ggid_redirect_after_login") || "/";
        sessionStorage.removeItem("ggid_redirect_after_login");
        router.replace(redirectTo);
      })
      .catch((err) => {
        setError(err instanceof Error ? err.message : "Token exchange failed");
        sessionStorage.removeItem("ggid_oauth_flow");
      });
  }, [params, router]);

  if (error) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-gray-50 dark:bg-gray-900">
        <div className="w-full max-w-md rounded-xl bg-white p-8 shadow-lg dark:bg-gray-800">
          <h1 className="mb-4 text-center text-2xl font-bold text-red-600">Authentication Error</h1>
          <p className="mb-6 text-center text-gray-600 dark:text-gray-400">{error}</p>
          <button
            onClick={() => router.push("/login")}
            className="w-full rounded-lg bg-brand-600 px-4 py-2 text-white hover:bg-brand-700"
          >
            Back to Login
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50 dark:bg-gray-900">
      <div className="text-center">
        <div className="mx-auto mb-4 h-8 w-8 animate-spin rounded-full border-4 border-brand-200 border-t-brand-600" />
        <p className="text-gray-600 dark:text-gray-400">Completing sign in...</p>
      </div>
    </div>
  );
}

export default function AuthCallbackPage() {
  return (
    <Suspense
      fallback={
        <div className="flex min-h-screen items-center justify-center bg-gray-50 dark:bg-gray-900">
          <div className="mx-auto h-8 w-8 animate-spin rounded-full border-4 border-brand-200 border-t-brand-600" />
        </div>
      }
    >
      <CallbackContent />
    </Suspense>
  );
}

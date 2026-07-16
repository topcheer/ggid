"use client";

import { useEffect, useState, Suspense } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { Loader2, CheckCircle, XCircle } from "lucide-react";
import { validateCallback, exchangeCodeForTokens } from "@/lib/oauth-pkce";
import { API_BASE_URL, DEFAULT_TENANT_ID } from "@/lib/api-config";

function OAuthCallbackInner() {
  const router = useRouter();
  const params = useSearchParams();
  const [status, setStatus] = useState<"processing" | "success" | "error">("processing");
  const [errorMsg, setErrorMsg] = useState("");

  useEffect(() => {
    const code = params.get("code");
    const state = params.get("state");
    const error = params.get("error");

    if (error) {
      setStatus("error");
      setErrorMsg(error);
      return;
    }

    if (!code || !state) {
      setStatus("error");
      setErrorMsg("Missing authorization code or state parameter");
      return;
    }

    (async () => {
      try {
        const flow = validateCallback(state, code);

        const tokens = await exchangeCodeForTokens(
          `${API_BASE_URL}/oauth/token`,
          code,
          flow,
        );

        // Store tokens
        localStorage.setItem("ggid_access_token", tokens.access_token);
        if (tokens.refresh_token) {
          localStorage.setItem("ggid_refresh_token", tokens.refresh_token);
        }

        setStatus("success");

        // Redirect to dashboard after brief delay
        setTimeout(() => router.push("/dashboard"), 1000);
      } catch (e) {
        setStatus("error");
        setErrorMsg(e instanceof Error ? e.message : "OAuth callback failed");
      }
    })();
  }, [params, router]);

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50 dark:bg-gray-950">
      <div className="w-full max-w-md rounded-xl border border-gray-200 bg-white p-8 shadow-sm dark:border-gray-700 dark:bg-gray-900">
        {status === "processing" && (
          <div className="text-center">
            <Loader2 className="mx-auto h-10 w-10 animate-spin text-brand-600" />
            <p className="mt-4 text-sm text-gray-500">Completing authentication...</p>
          </div>
        )}
        {status === "success" && (
          <div className="text-center">
            <CheckCircle className="mx-auto h-10 w-10 text-green-500" />
            <p className="mt-4 text-sm font-medium text-gray-900 dark:text-gray-100">Authentication successful</p>
            <p className="mt-1 text-xs text-gray-500">Redirecting to dashboard...</p>
          </div>
        )}
        {status === "error" && (
          <div className="text-center">
            <XCircle className="mx-auto h-10 w-10 text-red-500" />
            <p className="mt-4 text-sm font-medium text-gray-900 dark:text-gray-100">Authentication failed</p>
            <p className="mt-1 text-xs text-red-500">{errorMsg}</p>
            <button
              onClick={() => router.push("/login")}
              className="mt-4 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
            >
              Back to Login
            </button>
          </div>
        )}
      </div>
    </div>
  );
}

export default function OAuthCallbackPage() {
  return (
    <Suspense fallback={<div className="flex min-h-screen items-center justify-center"><Loader2 className="h-8 w-8 animate-spin text-brand-600" /></div>}>
      <OAuthCallbackInner />
    </Suspense>
  );
}

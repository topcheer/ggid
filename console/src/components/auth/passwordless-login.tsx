"use client";

import { useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import { Fingerprint, Mail, Smartphone, Loader2 } from "lucide-react";

interface PasswordlessLoginProps {
  apiBase: string;
  tenantId: string;
  authMethod: string; // passkey | sms_otp | email_otp
  onSuccess: (tokenData: { access_token: string; refresh_token?: string }) => void;
  onError: (msg: string) => void;
}

/**
 * PasswordlessLogin — renders the appropriate passwordless login UI
 * based on the client's configured auth method.
 *
 * Supports: passkey, sms_otp, email_otp
 * All flows end with auth_ticket → authorize code → JWT exchange.
 */
export function PasswordlessLogin({ apiBase, tenantId, authMethod, onSuccess, onError }: PasswordlessLoginProps) {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [identifier, setIdentifier] = useState("");
  const [otpCode, setOtpCode] = useState("");
  const [otpSent, setOtpSent] = useState(false);
  const [countdown, setCountdown] = useState(0);

  const exchangeTicketForToken = useCallback(async (ticket: string) => {
    try {
      // Generate random state + PKCE code_verifier
      const state = crypto.getRandomValues(new Uint32Array(8)).join("");
      const codeVerifier = crypto.getRandomValues(new Uint32Array(32)).reduce((s, b) => s + b.toString(16).padStart(2, "0"), "");
      const encoder = new TextEncoder();
      const hashBuf = await crypto.subtle.digest("SHA-256", encoder.encode(codeVerifier));
      const codeChallenge = btoa(String.fromCharCode(...new Uint8Array(hashBuf))).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");

      const authorizeUrl = `${apiBase}/oauth/authorize?auth_ticket=${ticket}&client_id=gcid-console&redirect_uri=${encodeURIComponent(window.location.origin + "/callback")}&response_type=code&scope=openid+profile+email+offline_access&code_challenge=${codeChallenge}&code_challenge_method=S256&state=${state}`;

      // Try to get auth code via fetch with redirect:manual
      let authCode = "";
      try {
        const codeResp = await fetch(authorizeUrl, {
          headers: { "X-Tenant-ID": tenantId },
          redirect: "manual" as RequestRedirect,
        });
        const loc = codeResp.headers.get("location") || "";
        const m = loc.match(/code=([^&]+)/);
        if (m) authCode = m[1];
      } catch {}

      // If manual redirect didn't work, redirect browser to authorize URL.
      // The /callback page will receive the code and exchange it.
      if (!authCode) {
        // Redirect browser to authorize URL. The /callback page will
        // receive the code and exchange it using the stored flow.
        sessionStorage.setItem("ggid_oauth_flow", JSON.stringify({
          code_verifier: codeVerifier,
          redirect_uri: window.location.origin + "/callback",
          client_id: "gcid-console",
          state: state,
        }));
        window.location.href = authorizeUrl;
        return;
      }

      // Exchange code for token
      const tokenResp = await fetch(`${apiBase}/oauth/token`, {
        method: "POST",
        headers: { "Content-Type": "application/x-www-form-urlencoded", "X-Tenant-ID": tenantId },
        body: new URLSearchParams({
          grant_type: "authorization_code",
          code: authCode,
          client_id: "gcid-console",
          redirect_uri: window.location.origin + "/callback",
          code_verifier: codeVerifier,
        }),
      });
      const tokenData = await tokenResp.json();
      if (!tokenData.access_token) throw new Error("Failed to obtain access token");
      onSuccess(tokenData);
    } catch (err) {
      onError(err instanceof Error ? err.message : "Token exchange failed");
    }
  }, [apiBase, tenantId, onSuccess, onError]);

  // --- Passkey login ---
  const handlePasskey = async () => {
    setLoading(true);
    onError("");
    try {
      const beginResp = await fetch(`${apiBase}/api/v1/auth/webauthn/login/begin`, {
        method: "POST",
        headers: { "Content-Type": "application/json", "X-Tenant-ID": tenantId },
      });
      if (!beginResp.ok) {
        const errData = await beginResp.json().catch(() => ({}));
        const errMsg = typeof errData.error === 'string'
          ? errData.error
          : errData.error?.message || errData.error || "Passkey login not available";
        if (errMsg.includes("no credentials") || errMsg.includes("Found no credentials")) {
          throw new Error("No passkey registered. Please log in with password first, then register a passkey in Settings.");
        }
        throw new Error(errMsg);
      }
      const options = await beginResp.json();
      const pk = options.publicKey || options.response || options;

      // base64url decode helper
      const b64urlToBuf = (s: string): Uint8Array => {
        const raw = atob(s.replace(/-/g, "+").replace(/_/g, "/"));
        const arr = new Uint8Array(raw.length);
        for (let i = 0; i < raw.length; i++) arr[i] = raw.charCodeAt(i);
        return arr;
      };
      if (pk.challenge && typeof pk.challenge === "string") pk.challenge = b64urlToBuf(pk.challenge);
      if (pk.allowCredentials) {
        pk.allowCredentials = pk.allowCredentials.map((c: any) => ({ ...c, id: b64urlToBuf(c.id) }));
      }

      const credential = await navigator.credentials.get({ publicKey: pk });
      if (!credential) throw new Error("No credential returned");

      // safe base64url encoder (chunked to avoid stack overflow)
      const bufToB64url = (buf: ArrayBuffer): string => {
        const bytes = new Uint8Array(buf);
        let binary = "";
        const chunk = 8192;
        for (let i = 0; i < bytes.length; i += chunk) {
          binary += String.fromCharCode(...bytes.subarray(i, i + chunk));
        }
        return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
      };

      const pkc = credential as PublicKeyCredential;
      const assertResp = pkc.response as AuthenticatorAssertionResponse;
      const assertBody = {
        id: pkc.id,
        rawId: bufToB64url(pkc.rawId),
        type: pkc.type,
        response: {
          authenticatorData: bufToB64url(assertResp.authenticatorData),
          clientDataJSON: bufToB64url(assertResp.clientDataJSON),
          signature: bufToB64url(assertResp.signature),
          userHandle: assertResp.userHandle ? bufToB64url(assertResp.userHandle) : null,
        },
      };

      const finishResp = await fetch(`${apiBase}/api/v1/auth/webauthn/login/finish`, {
        method: "POST",
        headers: { "Content-Type": "application/json", "X-Tenant-ID": tenantId },
        body: JSON.stringify(assertBody),
      });
      const result = await finishResp.json();
      if (!finishResp.ok) throw new Error(result.error || "Passkey verification failed");
      if (!result.auth_ticket) throw new Error("No auth ticket");

      await exchangeTicketForToken(result.auth_ticket);
    } catch (err) {
      if (err.name === "NotAllowedError") onError("Passkey authentication cancelled");
      else onError(err instanceof Error ? err.message : "Passkey login failed");
    }
    setLoading(false);
  };

  // --- OTP send ---
  const handleSendOTP = async () => {
    if (!identifier) { onError("Please enter your " + (authMethod === "sms_otp" ? "phone number" : "email")); return; }
    setLoading(true);
    onError("");
    try {
      const resp = await fetch(`${apiBase}/api/v1/auth/otp/send`, {
        method: "POST",
        headers: { "Content-Type": "application/json", "X-Tenant-ID": tenantId },
        body: JSON.stringify({ identifier, channel: authMethod === "sms_otp" ? "sms" : "email" }),
      });
      const data = await resp.json();
      if (!resp.ok) throw new Error(data.error || "Failed to send code");
      setOtpSent(true);
      setCountdown(60);
      const tick = () => {
        setCountdown((c) => {
          if (c > 1) { setTimeout(tick, 1000); return c - 1; }
          return 0;
        });
      };
      setTimeout(tick, 1000);
    } catch (err) {
      onError(err instanceof Error ? err.message : "Failed to send code");
    }
    setLoading(false);
  };

  // --- OTP verify ---
  const handleVerifyOTP = async () => {
    if (!otpCode) { onError("Please enter the code"); return; }
    setLoading(true);
    onError("");
    try {
      const resp = await fetch(`${apiBase}/api/v1/auth/otp/verify`, {
        method: "POST",
        headers: { "Content-Type": "application/json", "X-Tenant-ID": tenantId },
        body: JSON.stringify({ identifier, code: otpCode, channel: authMethod === "sms_otp" ? "sms" : "email" }),
      });
      const result = await resp.json();
      if (!resp.ok) throw new Error(result.error || "Verification failed");
      if (!result.auth_ticket) throw new Error("No auth ticket");
      await exchangeTicketForToken(result.auth_ticket);
    } catch (err) {
      onError(err instanceof Error ? err.message : "Verification failed");
    }
    setLoading(false);
  };

  // --- Render ---
  if (authMethod === "passkey") {
    return (
      <button
        type="button"
        onClick={handlePasskey}
        disabled={loading}
        className="w-full flex items-center justify-center gap-2 rounded-xl border border-gray-300 px-4 py-2.5 text-sm font-medium text-gray-700 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 focus:ring-2 focus:ring-brand-500 focus:ring-offset-2"
      >
        {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Fingerprint className="h-4 w-4" />}
        Login with Passkey
      </button>
    );
  }

  // SMS or Email OTP
  const isSMS = authMethod === "sms_otp";
  const label = isSMS ? "Phone Number" : "Email";
  const placeholder = isSMS ? "+86 138 0000 0000" : "user@example.com";
  const Icon = isSMS ? Smartphone : Mail;

  return (
    <div className="space-y-3">
      <div>
        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{label}</label>
        <input
          type={isSMS ? "tel" : "email"}
          value={identifier}
          onChange={(e) => setIdentifier(e.target.value)}
          placeholder={placeholder}
          disabled={otpSent}
          className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-800 dark:text-gray-200 focus:ring-2 focus:ring-brand-500"
        />
      </div>
      {!otpSent ? (
        <button
          type="button"
          onClick={handleSendOTP}
          disabled={loading || !identifier}
          className="w-full flex items-center justify-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
        >
          {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Icon className="h-4 w-4" />}
          Send Code
        </button>
      ) : (
        <>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Verification Code</label>
            <input
              type="text"
              inputMode="numeric"
              maxLength={6}
              value={otpCode}
              onChange={(e) => setOtpCode(e.target.value.replace(/\D/g, ""))}
              placeholder="6-digit code"
              className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm tracking-widest dark:border-gray-600 dark:bg-gray-800 dark:text-gray-200 focus:ring-2 focus:ring-brand-500"
            />
          </div>
          <button
            type="button"
            onClick={handleVerifyOTP}
            disabled={loading || !otpCode}
            className="w-full rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
          >
            {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
            Verify & Login
          </button>
          <button
            type="button"
            onClick={countdown === 0 ? handleSendOTP : undefined}
            disabled={countdown > 0}
            className="w-full text-center text-xs text-gray-500 hover:text-gray-700 disabled:opacity-50"
          >
            {countdown > 0 ? `Resend in ${countdown}s` : "Resend code"}
          </button>
        </>
      )}
    </div>
  );
}

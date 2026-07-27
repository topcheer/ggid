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
      // Exchange auth_ticket for code via /oauth/authorize
      const codeResp = await fetch(
        `${apiBase}/oauth/authorize?auth_ticket=${ticket}&client_id=gcid-console&redirect_uri=${encodeURIComponent(window.location.origin + "/callback")}&response_type=code&scope=openid+profile+email+offline_access&code_challenge=console-pkce&code_challenge_method=S256`,
        { headers: { "X-Tenant-ID": tenantId }, redirect: "manual" as RequestRedirect },
      );
      const loc = codeResp.headers.get("location") || "";
      const codeMatch = loc.match(/code=([^&]+)/);
      if (!codeMatch) throw new Error("Failed to get authorization code");

      // Exchange code for token
      const tokenResp = await fetch(`${apiBase}/oauth/token`, {
        method: "POST",
        headers: { "Content-Type": "application/x-www-form-urlencoded", "X-Tenant-ID": tenantId },
        body: new URLSearchParams({
          grant_type: "authorization_code",
          code: codeMatch[1],
          client_id: "gcid-console",
          redirect_uri: window.location.origin + "/callback",
          code_verifier: "console-pkce",
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
      if (!beginResp.ok) throw new Error("Passkey not available");
      const options = await beginResp.json();

      const credential = await navigator.credentials.get({ publicKey: options.response });
      const assertBody = {
        id: credential.id,
        rawId: btoa(String.fromCharCode(...new Uint8Array(credential.rawId))),
        type: credential.type,
        response: {
          authenticatorData: btoa(String.fromCharCode(...new Uint8Array(credential.response.authenticatorData))),
          clientDataJSON: btoa(String.fromCharCode(...new Uint8Array(credential.response.clientDataJSON))),
          signature: btoa(String.fromCharCode(...new Uint8Array(credential.response.signature))),
          userHandle: credential.response.userHandle ? btoa(String.fromCharCode(...new Uint8Array(credential.response.userHandle))) : null,
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

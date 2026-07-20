/**
 * React hook and components for WebAuthn / Passkey integration
 *
 * Usage:
 * ```tsx
 * import { usePasskey, PasskeyButton } from "@ggid/react/passkey";
 *
 * function App() {
 *   const { register, authenticate, supported, loading } = usePasskey({
 *     apiBaseUrl: "https://ggid.example.com",
 *     authToken: userToken,
 *   });
 *   return <PasskeyButton onClick={() => register(userId)} />;
 * }
 * ```
 */

import { useState, useCallback } from "react";
import {
  bufferToBase64url,
  base64urlToBuffer,
  isWebAuthnSupported,
} from "./passkey";

export interface UsePasskeyOptions {
  apiBaseUrl: string;
  authToken?: string;
  tenantId?: string;
}

export interface PasskeyHook {
  supported: boolean;
  loading: boolean;
  error: string | null;
  success: boolean;
  register: (userId: string) => Promise<boolean>;
  authenticate: () => Promise<Record<string, unknown> | null>;
  reset: () => void;
}

/**
 * React hook for passkey registration and authentication.
 */
export function usePasskey(opts: UsePasskeyOptions): PasskeyHook {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  const register = useCallback(async (userId: string): Promise<boolean> => {
    if (!isWebAuthnSupported()) {
      setError("WebAuthn is not supported in this browser");
      return false;
    }
    setLoading(true);
    setError(null);
    setSuccess(false);
    try {
      const headers: Record<string, string> = {
        "Content-Type": "application/json",
      };
      if (opts.authToken) headers["Authorization"] = `Bearer ${opts.authToken}`;
      if (opts.tenantId) headers["X-Tenant-ID"] = opts.tenantId;

      // Begin
      const beginRes = await fetch(`${opts.apiBaseUrl}/api/v1/auth/webauthn/register/begin`, {
        method: "POST",
        headers,
        body: JSON.stringify({ user_id: userId }),
      });
      if (!beginRes.ok) throw new Error("Failed to start registration");

      const beginData = await beginRes.json();
      const publicKeyOptions = beginData.publicKey || beginData;

      // Create credential
      const decodedOptions: PublicKeyCredentialCreationOptions = {
        challenge: base64urlToBuffer(publicKeyOptions.challenge),
        rp: publicKeyOptions.rp,
        user: { ...publicKeyOptions.user, id: base64urlToBuffer(publicKeyOptions.user.id) },
        pubKeyCredParams: publicKeyOptions.pubKeyCredParams,
        timeout: publicKeyOptions.timeout,
        authenticatorSelection: publicKeyOptions.authenticatorSelection,
        attestation: publicKeyOptions.attestation,
      };

      const credential = await navigator.credentials.create({
        publicKey: decodedOptions,
      }) as PublicKeyCredential | null;

      if (!credential) {
        setError("Registration cancelled");
        return false;
      }

      // Finish
      const response = credential.response as AuthenticatorAttestationResponse;
      const attestation = {
        id: credential.id,
        rawId: bufferToBase64url(credential.rawId),
        type: credential.type,
        response: {
          attestationObject: bufferToBase64url(response.attestationObject),
          clientDataJSON: bufferToBase64url(response.clientDataJSON),
        },
      };

      const finishRes = await fetch(`${opts.apiBaseUrl}/api/v1/auth/webauthn/register/finish`, {
        method: "POST",
        headers,
        body: JSON.stringify(attestation),
      });

      if (finishRes.ok) {
        setSuccess(true);
        return true;
      }
      throw new Error("Failed to verify passkey");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Registration failed");
      return false;
    } finally {
      setLoading(false);
    }
  }, [opts.apiBaseUrl, opts.authToken, opts.tenantId]);

  const authenticate = useCallback(async (): Promise<Record<string, unknown> | null> => {
    if (!isWebAuthnSupported()) {
      setError("WebAuthn is not supported");
      return null;
    }
    setLoading(true);
    setError(null);
    try {
      const headers: Record<string, string> = { "Content-Type": "application/json" };
      if (opts.tenantId) headers["X-Tenant-ID"] = opts.tenantId;

      const beginRes = await fetch(`${opts.apiBaseUrl}/api/v1/auth/webauthn/auth/begin`, {
        method: "POST",
        headers,
      });
      if (!beginRes.ok) throw new Error("Failed to start authentication");

      const beginData = await beginRes.json();
      const publicKeyOptions = beginData.publicKey || beginData;

      const decodedOptions: PublicKeyCredentialRequestOptions = {
        challenge: base64urlToBuffer(publicKeyOptions.challenge),
        rpId: publicKeyOptions.rpId,
        timeout: publicKeyOptions.timeout,
        userVerification: publicKeyOptions.userVerification,
      };

      const assertion = await navigator.credentials.get({
        publicKey: decodedOptions,
      }) as PublicKeyCredential | null;

      if (!assertion) return null;

      const response = assertion.response as AuthenticatorAssertionResponse;
      return {
        id: assertion.id,
        rawId: bufferToBase64url(assertion.rawId),
        type: assertion.type,
        response: {
          authenticatorData: bufferToBase64url(response.authenticatorData),
          clientDataJSON: bufferToBase64url(response.clientDataJSON),
          signature: bufferToBase64url(response.signature),
          userHandle: response.userHandle ? bufferToBase64url(response.userHandle) : null,
        },
      };
    } catch (e) {
      setError(e instanceof Error ? e.message : "Authentication failed");
      return null;
    } finally {
      setLoading(false);
    }
  }, [opts.apiBaseUrl, opts.tenantId]);

  const reset = useCallback(() => {
    setError(null);
    setSuccess(false);
  }, []);

  return {
    supported: isWebAuthnSupported(),
    loading,
    error,
    success,
    register,
    authenticate,
    reset,
  };
}

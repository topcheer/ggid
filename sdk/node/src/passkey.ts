/**
 * WebAuthn / Passkey utilities for GGID SDK (Node.js / browser)
 *
 * Encapsulates the browser-side WebAuthn API for passkey registration and
 * authentication, plus server-side helpers for encoding/decoding.
 *
 * Browser usage:
 * ```ts
 * import { registerPasskey, authenticateWithPasskey } from "@ggid/node/passkey";
 * ```
 *
 * Server-side helpers:
 * ```ts
 * import { bufferToBase64url, base64urlToBuffer } from "@ggid/node/passkey";
 * ```
 */

/** Convert ArrayBuffer to base64url string */
export function bufferToBase64url(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let binary = "";
  for (const b of bytes) binary += String.fromCharCode(b);
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=/g, "");
}

/** Convert base64url string to ArrayBuffer */
export function base64urlToBuffer(b64url: string): ArrayBuffer {
  const padding = "=".repeat((4 - (b64url.length % 4)) % 4);
  const base64 = (b64url + padding).replace(/-/g, "+").replace(/_/g, "/");
  const binary = atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
  return bytes.buffer;
}

/** Check if the current browser supports WebAuthn */
export function isWebAuthnSupported(): boolean {
  return typeof window !== "undefined" && "PublicKeyCredential" in window;
}

/**
 * Register a new passkey (browser-side).
 *
 * @param apiBaseUrl - GGID API base URL
 * @param authToken - JWT access token
 * @param userId - User ID
 * @returns true if registration succeeded
 */
export async function registerPasskey(opts: {
  apiBaseUrl: string;
  authToken: string;
  userId: string;
  tenantId?: string;
}): Promise<boolean> {
  if (!isWebAuthnSupported()) {
    throw new Error("WebAuthn is not supported in this browser");
  }

  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    Authorization: `Bearer ${opts.authToken}`,
  };
  if (opts.tenantId) headers["X-Tenant-ID"] = opts.tenantId;

  // 1. Fetch registration challenge
  const beginRes = await fetch(`${opts.apiBaseUrl}/api/v1/auth/webauthn/register/begin`, {
    method: "POST",
    headers,
    body: JSON.stringify({ user_id: opts.userId }),
  });
  if (!beginRes.ok) throw new Error(`Registration begin failed: ${beginRes.status}`);

  const beginData = await beginRes.json();
  const publicKeyOptions = beginData.publicKey || beginData;

  // 2. Decode and create credential
  const decodedOptions = decodeCreationOptions(publicKeyOptions);
  const credential = await navigator.credentials.create({
    publicKey: decodedOptions,
  }) as PublicKeyCredential | null;

  if (!credential) return false;

  // 3. Send attestation to backend
  const attestation = encodeAttestation(credential);
  const finishRes = await fetch(`${opts.apiBaseUrl}/api/v1/auth/webauthn/register/finish`, {
    method: "POST",
    headers,
    body: JSON.stringify(attestation),
  });

  return finishRes.ok;
}

/**
 * Authenticate with a passkey (browser-side).
 *
 * @param apiBaseUrl - GGID API base URL
 * @returns Credential assertion for server verification, or null if cancelled
 */
export async function authenticateWithPasskey(opts: {
  apiBaseUrl: string;
  tenantId?: string;
}): Promise<Record<string, unknown> | null> {
  if (!isWebAuthnSupported()) {
    throw new Error("WebAuthn is not supported in this browser");
  }

  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };
  if (opts.tenantId) headers["X-Tenant-ID"] = opts.tenantId;

  // 1. Fetch auth challenge
  const beginRes = await fetch(`${opts.apiBaseUrl}/api/v1/auth/webauthn/auth/begin`, {
    method: "POST",
    headers,
  });
  if (!beginRes.ok) throw new Error(`Auth begin failed: ${beginRes.status}`);

  const beginData = await beginRes.json();
  const publicKeyOptions = beginData.publicKey || beginData;

  // 2. Get assertion
  const decodedOptions = decodeRequestOptions(publicKeyOptions);
  const assertion = await navigator.credentials.get({
    publicKey: decodedOptions,
  }) as PublicKeyCredential | null;

  if (!assertion) return null;

  // 3. Encode assertion for server
  return encodeAssertion(assertion);
}

// === Internal helpers ===

function decodeCreationOptions(options: Record<string, unknown>): PublicKeyCredentialCreationOptions {
  const challenge = options.challenge as string;
  const userId = (options.user as Record<string, unknown>)?.id as string;

  return {
    challenge: base64urlToBuffer(challenge),
    rp: options.rp as PublicKeyCredentialRpEntity,
    user: {
      ...(options.user as PublicKeyCredentialUserEntity),
      id: base64urlToBuffer(userId),
    },
    pubKeyCredParams: options.pubKeyCredParams as PublicKeyCredentialParameters[],
    timeout: options.timeout as number | undefined,
    excludeCredentials: (options.excludeCredentials as Array<Record<string, unknown>>)?.map(c => ({
      type: "public-key" as const,
      ...c,
      id: base64urlToBuffer(c.id as string),
    })),
    authenticatorSelection: options.authenticatorSelection as AuthenticatorSelectionCriteria | undefined,
    attestation: options.attestation as AttestationConveyancePreference | undefined,
  };
}

function decodeRequestOptions(options: Record<string, unknown>): PublicKeyCredentialRequestOptions {
  return {
    challenge: base64urlToBuffer(options.challenge as string),
    rpId: options.rpId as string | undefined,
    timeout: options.timeout as number | undefined,
    allowCredentials: (options.allowCredentials as Array<Record<string, unknown>>)?.map(c => ({
      type: "public-key" as const,
      ...c,
      id: base64urlToBuffer(c.id as string),
    })),
    userVerification: options.userVerification as UserVerificationRequirement | undefined,
  };
}

function encodeAttestation(credential: PublicKeyCredential): Record<string, unknown> {
  const response = credential.response as AuthenticatorAttestationResponse;
  return {
    id: credential.id,
    rawId: bufferToBase64url(credential.rawId),
    type: credential.type,
    response: {
      attestationObject: bufferToBase64url(response.attestationObject),
      clientDataJSON: bufferToBase64url(response.clientDataJSON),
    },
    authenticatorAttachment: credential.authenticatorAttachment,
  };
}

function encodeAssertion(credential: PublicKeyCredential): Record<string, unknown> {
  const response = credential.response as AuthenticatorAssertionResponse;
  return {
    id: credential.id,
    rawId: bufferToBase64url(credential.rawId),
    type: credential.type,
    response: {
      authenticatorData: bufferToBase64url(response.authenticatorData),
      clientDataJSON: bufferToBase64url(response.clientDataJSON),
      signature: bufferToBase64url(response.signature),
      userHandle: response.userHandle ? bufferToBase64url(response.userHandle) : null,
    },
  };
}

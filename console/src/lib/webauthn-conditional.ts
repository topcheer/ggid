/**
 * Conditional Create — passwordless passkey upgrade after password login.
 *
 * After a successful password login, the browser can automatically prompt
 * the user to create a passkey (mediation: "conditional"). This is the
 * FIDO 2025 "silent migration" path for converting password users to passkeys.
 *
 * Reference: WebAuthn Level 3 — PublicKeyCredential.isConditionalMediationAvailable()
 */

import { authHeader } from "./auth-helpers";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";
const API_BASE = process.env.NEXT_PUBLIC_API_URL || "";

/** Check if conditional mediation is supported in this browser. */
export async function isConditionalCreateSupported(): Promise<boolean> {
  try {
    // Feature-detect both APIs
    if (typeof PublicKeyCredential === "undefined") return false;
    if (!("isConditionalMediationAvailable" in PublicKeyCredential)) return false;
    return await PublicKeyCredential.isConditionalMediationAvailable();
  } catch {
    return false;
  }
}

/**
 * Check if the user already has a registered passkey.
 * Calls the WebAuthn credentials list endpoint.
 */
export async function userHasPasskey(userId: string): Promise<boolean> {
  try {
    const res = await fetch(`${API_BASE}/api/v1/auth/webauthn/credentials?user_id=${userId}`, {
      headers: { ...authHeader(), "X-Tenant-ID": TENANT_ID },
    });
    if (!res.ok) return false;
    const data = await res.json();
    return Array.isArray(data.credentials) && data.credentials.length > 0;
  } catch {
    return false;
  }
}

/**
 * Attempt to create a passkey after password login.
 *
 * Flow:
 * 1. Check browser support (conditional mediation)
 * 2. Check user doesn't already have a passkey
 * 3. Fetch registration challenge from backend
 * 4. Call navigator.credentials.create({ mediation: "conditional" })
 * 5. Send attestation back to backend for verification
 *
 * All failures are silent — this is a progressive enhancement.
 * Login redirect must NOT be blocked by this function.
 *
 * @returns true if a passkey was successfully created, false otherwise.
 */
export async function offerPasskeyUpgrade(params: {
  accessToken: string;
  userId: string;
}): Promise<boolean> {
  const { accessToken, userId } = params;

  // 1. Feature detection
  const supported = await isConditionalCreateSupported();
  if (!supported) {
    console.debug("[ConditionalCreate] Browser does not support conditional mediation");
    return false;
  }

  // 2. Skip if user already has a passkey
  const hasPasskey = await userHasPasskey(userId);
  if (hasPasskey) {
    console.debug("[ConditionalCreate] User already has a passkey, skipping");
    return false;
  }

  try {
    // 3. Fetch registration challenge from backend
    const beginRes = await fetch(`${API_BASE}/api/v1/auth/webauthn/register/begin`, {
      method: "POST",
      headers: {
        ...authHeader(),
        "Content-Type": "application/json",
        "X-Tenant-ID": TENANT_ID,
      },
      body: JSON.stringify({ user_id: userId }),
    });

    if (!beginRes.ok) {
      console.debug("[ConditionalCreate] Failed to fetch registration challenge");
      return false;
    }

    const beginData = await beginRes.json();
    const publicKeyOptions = beginData.publicKey || beginData;

    // Decode base64url arrays for ArrayBuffer fields
    const decodedOptions = decodeCreationOptions(publicKeyOptions);

    // 4. Create the credential with conditional mediation
    const credential = await navigator.credentials.create({
      publicKey: decodedOptions,
      // @ts-expect-error — mediation is part of CredentialCreationOptions in L3
      mediation: "conditional",
    }) as PublicKeyCredential | null;

    if (!credential) {
      console.debug("[ConditionalCreate] User dismissed the passkey prompt");
      return false;
    }

    // 5. Send attestation back to backend
    const attestation = encodeAttestation(credential);
    const finishRes = await fetch(`${API_BASE}/api/v1/auth/webauthn/register/finish`, {
      method: "POST",
      headers: {
        ...authHeader(),
        "Content-Type": "application/json",
        "X-Tenant-ID": TENANT_ID,
      },
      body: JSON.stringify(attestation),
    });

    if (finishRes.ok) {
      console.debug("[ConditionalCreate] Passkey created successfully");
      return true;
    }

    console.debug("[ConditionalCreate] Backend rejected attestation");
    return false;
  } catch (err) {
    // AbortError is normal — user clicked "Not now" or cancelled
    if (err instanceof DOMException && err.name === "AbortError") {
      console.debug("[ConditionalCreate] User cancelled passkey creation");
    } else {
      console.debug("[ConditionalCreate] Error:", err);
    }
    return false;
  }
}

// ==================== Helpers ====================

/** Decode base64url → ArrayBuffer for WebAuthn API fields. */
function b64urlToBuffer(b64url: string): ArrayBuffer {
  const padding = "=".repeat((4 - (b64url.length % 4)) % 4);
  const base64 = (b64url + padding).replace(/-/g, "+").replace(/_/g, "/");
  const binary = atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes.buffer;
}

/** ArrayBuffer → base64url string. */
function bufferToB64url(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let binary = "";
  for (const b of bytes) {
    binary += String.fromCharCode(b);
  }
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=/g, "");
}

/** Decode PublicKeyCredentialCreationOptions from server JSON. */
function decodeCreationOptions(options: Record<string, unknown>): PublicKeyCredentialCreationOptions {
  const challenge = options.challenge as string;
  const userId = (options.user as Record<string, unknown>)?.id as string;

  return {
    challenge: b64urlToBuffer(challenge),
    rp: options.rp as PublicKeyCredentialRpEntity,
    user: {
      ...(options.user as PublicKeyCredentialUserEntity),
      id: b64urlToBuffer(userId),
    },
    pubKeyCredParams: options.pubKeyCredParams as PublicKeyCredentialParameters[],
    timeout: options.timeout as number | undefined,
    excludeCredentials: (options.excludeCredentials as Array<Record<string, unknown>>)?.map(c => ({
      type: "public-key" as const,
      ...c,
      id: b64urlToBuffer(c.id as string),
    })),
    authenticatorSelection: options.authenticatorSelection as AuthenticatorSelectionCriteria | undefined,
    attestation: options.attestation as AttestationConveyancePreference | undefined,
  };
}

/** Encode the PublicKeyCredential for sending to backend finish endpoint. */
function encodeAttestation(credential: PublicKeyCredential): Record<string, unknown> {
  const response = credential.response as AuthenticatorAttestationResponse;
  return {
    id: credential.id,
    rawId: bufferToB64url(credential.rawId),
    type: credential.type,
    response: {
      attestationObject: bufferToB64url(response.attestationObject),
      clientDataJSON: bufferToB64url(response.clientDataJSON),
    },
    authenticatorAttachment: credential.authenticatorAttachment,
  };
}

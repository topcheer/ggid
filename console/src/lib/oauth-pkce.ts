/**
 * OAuth 2.1 PKCE utilities for Console self-registration flow.
 * Implements code_verifier/code_challenge generation per RFC 7636.
 */

function base64UrlEncode(bytes: Uint8Array): string {
  const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_";
  let result = "";
  for (let i = 0; i < bytes.length; i += 3) {
    const b1 = bytes[i] & 0xff;
    const b2 = i + 1 < bytes.length ? bytes[i + 1] & 0xff : 0;
    const b3 = i + 2 < bytes.length ? bytes[i + 2] & 0xff : 0;
    result += chars[b1 >> 2];
    result += chars[((b1 & 0x03) << 4) | (b2 >> 4)];
    result += i + 1 < bytes.length ? chars[((b2 & 0x0f) << 2) | (b3 >> 6)] : "=";
    result += i + 2 < bytes.length ? chars[b3 & 0x3f] : "=";
  }
  return result.replace(/=+$/g, "");
}

/** Generate a cryptographically random code_verifier (43-128 chars). */
export function generateCodeVerifier(): string {
  const array = new Uint8Array(32);
  crypto.getRandomValues(array);
  return base64UrlEncode(array);
}

/** Derive code_challenge (S256) from code_verifier. */
export async function generateCodeChallenge(verifier: string): Promise<string> {
  const encoder = new TextEncoder();
  const data = encoder.encode(verifier);
  const digest = await crypto.subtle.digest("SHA-256", data);
  return base64UrlEncode(new Uint8Array(digest));
}

/** Generate a random state parameter for CSRF protection. */
export function generateState(): string {
  const array = new Uint8Array(16);
  crypto.getRandomValues(array);
  return base64UrlEncode(array);
}

export interface OAuthFlowState {
  code_verifier: string;
  code_challenge: string;
  state: string;
  redirect_uri: string;
  client_id: string;
  scope: string;
}

/**
 * Initialize the OAuth PKCE flow.
 * Stores code_verifier + state in sessionStorage for later validation.
 */
export async function initOAuthFlow(
  authorizeUrl: string,
  clientId: string,
  redirectUri: string,
  tenantId: string,
  scope: string = "openid profile email offline_access",
): Promise<string> {
  const code_verifier = generateCodeVerifier();
  const code_challenge = await generateCodeChallenge(code_verifier);
  const state = generateState();

  const flowState: OAuthFlowState = {
    code_verifier,
    code_challenge,
    state,
    redirect_uri: redirectUri,
    client_id: clientId,
    scope,
  };
  sessionStorage.setItem("ggid_oauth_flow", JSON.stringify(flowState));

  const params = new URLSearchParams({
    response_type: "code",
    client_id: clientId,
    redirect_uri: redirectUri,
    scope,
    state,
    code_challenge,
    code_challenge_method: "S256",
    tenant_id: tenantId,
  });

  return `${authorizeUrl}?${params.toString()}`;
}

/**
 * Validate the OAuth callback.
 * Returns the flow state if valid, throws if state mismatch.
 */
export function validateCallback(state: string, code: string): OAuthFlowState {
  const stored = sessionStorage.getItem("ggid_oauth_flow");
  if (!stored) throw new Error("No OAuth flow in progress");
  const flow = JSON.parse(stored) as OAuthFlowState;
  if (flow.state !== state) throw new Error("State mismatch — possible CSRF attack");
  if (!code) throw new Error("No authorization code received");
  return flow;
}

/**
 * Exchange authorization code for tokens using PKCE.
 */
export async function exchangeCodeForTokens(
  tokenUrl: string,
  code: string,
  flow: OAuthFlowState,
): Promise<{ access_token: string; refresh_token?: string; id_token?: string; expires_in?: number }> {
  const body = new URLSearchParams({
    grant_type: "authorization_code",
    code,
    redirect_uri: flow.redirect_uri,
    client_id: flow.client_id,
    code_verifier: flow.code_verifier,
  });

  const resp = await fetch(tokenUrl, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: body.toString(),
  });

  if (!resp.ok) {
    const err = await resp.text();
    throw new Error(`Token exchange failed: ${err}`);
  }

  // Clean up flow state
  sessionStorage.removeItem("ggid_oauth_flow");

  return resp.json();
}

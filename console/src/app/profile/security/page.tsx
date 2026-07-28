"use client";

import { useState, useEffect, useCallback } from "react";
import { Fingerprint, Plus, Trash2, CheckCircle2, AlertCircle, Loader2, KeyRound } from "lucide-react";

const API_BASE = process.env.NEXT_PUBLIC_API_BASE || "";

function getAuthToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("ggid_access_token");
}

function authHeader(): Record<string, string> {
  const token = getAuthToken();
  return token ? { Authorization: `Bearer ${token}`, "X-Tenant-ID": localStorage.getItem("ggid_tenant_id") || "" } : {};
}

interface WebAuthnCredential {
  id: string;
  credential_id?: string;
  name?: string;
  device_name?: string;
  platform?: string;
  transports?: string[];
  created_at?: string;
  last_used?: string;
}

export default function ProfileSecurityPage() {
  const [credentials, setCredentials] = useState<WebAuthnCredential[]>([]);
  const [loading, setLoading] = useState(true);
  const [registering, setRegistering] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  const loadCredentials = useCallback(async () => {
    setLoading(true);
    try {
      const resp = await fetch(`${API_BASE}/api/v1/auth/webauthn/credentials`, {
        headers: authHeader(),
      });
      if (resp.ok) {
        const data = await resp.json();
        const creds = data.credentials || data.data || (Array.isArray(data) ? data : []);
        setCredentials(creds);
      }
    } catch {
      // noop
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadCredentials();
  }, [loadCredentials]);

  const registerPasskey = async () => {
    setRegistering(true);
    setError("");
    setSuccess("");

    try {
      // Step 1: Begin registration
      const beginResp = await fetch(`${API_BASE}/api/v1/auth/webauthn/register/begin`, {
        method: "POST",
        headers: { "Content-Type": "application/json", ...authHeader() },
        body: JSON.stringify({}),
      });

      if (!beginResp.ok) {
        const err = await beginResp.json().catch(() => ({}));
        throw new Error(err.error?.message || err.error || "Failed to start passkey registration");
      }

      const options = await beginResp.json();

      // Step 2: Create credential via WebAuthn API
      const publicKeyOptions = options.response || options.publicKey || options;
      publicKeyOptions.challenge = base64ToArrayBuffer(publicKeyOptions.challenge);
      publicKeyOptions.user.id = base64ToArrayBuffer(publicKeyOptions.user.id);
      if (publicKeyOptions.excludeCredentials) {
        publicKeyOptions.excludeCredentials = publicKeyOptions.excludeCredentials.map((c: any) => ({
          ...c,
          id: base64ToArrayBuffer(c.id),
        }));
      }

      const credential = await navigator.credentials.create({ publicKey: publicKeyOptions });
      if (!credential) throw new Error("No credential returned");

      const pkCredential = credential as PublicKeyCredential;
      const response = pkCredential.response as AuthenticatorAttestationResponse;

      // Step 3: Finish registration
      const finishBody = {
        id: pkCredential.id,
        rawId: arrayBufferToBase64(pkCredential.rawId),
        type: pkCredential.type,
        response: {
          attestationObject: arrayBufferToBase64(response.attestationObject),
          clientDataJSON: arrayBufferToBase64(response.clientDataJSON),
        },
      };

      const finishResp = await fetch(`${API_BASE}/api/v1/auth/webauthn/register/finish`, {
        method: "POST",
        headers: { "Content-Type": "application/json", ...authHeader() },
        body: JSON.stringify(finishBody),
      });

      if (!finishResp.ok) {
        const err = await finishResp.json().catch(() => ({}));
        throw new Error(err.error?.message || err.error || "Failed to complete registration");
      }

      setSuccess("Passkey registered successfully! You can now use it to log in.");
      loadCredentials();
    } catch (e) {
      const msg = e instanceof Error ? e.message : "Registration failed";
      if (msg.includes("not supported") || msg.includes("PublicKeyCredential")) {
        setError("Your browser does not support passkeys. Please use a modern browser like Chrome, Firefox, or Safari.");
      } else if (msg.includes("cancelled") || msg.includes("aborted")) {
        setError("Passkey registration was cancelled.");
      } else {
        setError(msg);
      }
    } finally {
      setRegistering(false);
    }
  };

  const deleteCredential = async (id: string) => {
    if (!confirm("Remove this passkey? You will need to use password login if it is your only credential.")) return;

    try {
      const resp = await fetch(`${API_BASE}/api/v1/auth/webauthn/credentials/${id}`, {
        method: "DELETE",
        headers: authHeader(),
      });
      if (resp.ok) {
        setSuccess("Passkey removed.");
        loadCredentials();
      } else {
        setError("Failed to remove passkey.");
      }
    } catch {
      setError("Network error.");
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center p-12">
        <Loader2 className="h-6 w-6 animate-spin text-indigo-600" />
      </div>
    );
  }

  return (
    <div className="max-w-2xl mx-auto p-6">
      <div className="mb-6">
        <h1 className="text-xl font-bold flex items-center gap-2">
          <Fingerprint className="h-5 w-5 text-indigo-600" />
          Security & Passkeys
        </h1>
        <p className="text-sm text-gray-500 mt-1">
          Register passkeys for passwordless login. Passkeys are more secure than passwords and work across your devices.
        </p>
      </div>

      {error && (
        <div className="mb-4 flex items-start gap-2 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">
          <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
          <span>{error}</span>
        </div>
      )}

      {success && (
        <div className="mb-4 flex items-start gap-2 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700">
          <CheckCircle2 className="mt-0.5 h-4 w-4 shrink-0" />
          <span>{success}</span>
        </div>
      )}

      {/* Register new passkey */}
      <div className="mb-6 rounded-xl border border-gray-200 bg-white p-4 shadow-sm">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-sm font-semibold text-gray-900">Register New Passkey</h2>
            <p className="text-xs text-gray-500 mt-0.5">Add a passkey from this device or a hardware key.</p>
          </div>
          <button
            onClick={registerPasskey}
            disabled={registering}
            className="flex items-center gap-2 rounded-lg bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
          >
            {registering ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
            {registering ? "Registering..." : "Add Passkey"}
          </button>
        </div>
      </div>

      {/* Registered passkeys */}
      <div className="rounded-xl border border-gray-200 bg-white shadow-sm">
        <div className="border-b border-gray-100 px-4 py-3">
          <h2 className="text-sm font-semibold text-gray-900">Registered Passkeys ({credentials.length})</h2>
        </div>

        {credentials.length === 0 ? (
          <div className="px-4 py-8 text-center">
            <KeyRound className="mx-auto h-8 w-8 text-gray-300" />
            <p className="mt-2 text-sm text-gray-500">No passkeys registered yet.</p>
            <p className="text-xs text-gray-400">Click "Add Passkey" to register your first passkey.</p>
          </div>
        ) : (
          <div className="divide-y divide-gray-100">
            {credentials.map((cred) => (
              <div key={cred.id} className="flex items-center gap-3 px-4 py-3">
                <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-gray-100">
                  <Fingerprint className="h-4 w-4 text-gray-500" />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-gray-900 truncate">
                    {cred.name || cred.device_name || `Passkey ${cred.id.slice(0, 8)}`}
                  </p>
                  <p className="text-xs text-gray-500">
                    {cred.platform || "Unknown platform"}
                    {cred.created_at && ` · Added ${new Date(cred.created_at).toLocaleDateString()}`}
                  </p>
                </div>
                <button
                  onClick={() => deleteCredential(cred.id)}
                  className="rounded-lg p-1.5 text-gray-400 hover:bg-red-50 hover:text-red-500"
                  title="Remove passkey"
                >
                  <Trash2 className="h-4 w-4" />
                </button>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

// Helper functions for WebAuthn encoding
function base64ToArrayBuffer(base64: string): ArrayBuffer {
  const binary = atob(base64.replace(/-/g, "+").replace(/_/g, "/"));
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
  return bytes.buffer;
}

function arrayBufferToBase64(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let binary = "";
  for (let i = 0; i < bytes.length; i++) binary += String.fromCharCode(bytes[i]);
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

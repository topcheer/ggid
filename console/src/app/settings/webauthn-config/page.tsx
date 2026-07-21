"use client";
import { useState, useEffect } from "react";
import {
  Fingerprint, Save, Loader2, CheckCircle2, AlertCircle, Server, Globe,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { usePageTitle } from "@/lib/usePageTitle";
import { authHeader } from "@/lib/auth-helpers";
import { API_BASE_URL } from "@/lib/api-config";

const API_BASE = API_BASE_URL;

export default function WebAuthnConfigPage() {
  usePageTitle("WebAuthn Configuration");
  const t = useTranslations();
  const [rpId, setRpId] = useState("");
  const [origins, setOrigins] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState(false);

  useEffect(() => {
    (async () => {
      try {
        const res = await fetch(`${API_BASE}/api/v1/system/config`, {
          headers: { ...authHeader() },
        });
        if (res.ok) {
          const d = await res.json();
          const config = d.webauthn_config || d.value || {};
          setRpId(config.rp_id || config.rpId || "");
          setOrigins(Array.isArray(config.rp_origins) ? config.rp_origins.join("\n") : (config.rp_origins || ""));
        }
      } catch { /* config not yet set */ }
      setLoading(false);
    })();
  }, []);

  const handleSave = async () => {
    setSaving(true);
    setError("");
    setSuccess(false);
    try {
      const originsList = origins.split("\n").map(o => o.trim()).filter(Boolean);
      const res = await fetch(`${API_BASE}/api/v1/system/config`, {
        method: "PUT",
        headers: { "Content-Type": "application/json", ...authHeader() },
        body: JSON.stringify({
          key: "webauthn_config",
          value: {
            rp_id: rpId,
            rp_origins: originsList,
          },
        }),
      });
      if (res.ok) {
        setSuccess(true);
        setTimeout(() => setSuccess(false), 3000);
      } else {
        const d = await res.json().catch(() => ({}));
        setError(d.error?.message || "Failed to save configuration");
      }
    } catch {
      setError("Network error");
    }
    setSaving(false);
  };

  const currentHost = typeof window !== "undefined" ? window.location.hostname : "";

  if (loading) {
    return <div className="flex justify-center py-20"><Loader2 className="w-8 h-8 animate-spin text-blue-600" /></div>;
  }

  return (
    <div className="mx-auto max-w-2xl p-6">
      <h1 className="mb-1 text-2xl font-bold text-gray-900 dark:text-white dark:text-white">WebAuthn Configuration</h1>
      <p className="mb-6 text-sm text-gray-500">
        Configure the Relying Party (RP) domain for passkey registration and authentication.
      </p>

      {error && (
        <div className="mb-4 flex items-center gap-2 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950">
          <AlertCircle className="h-4 w-4 shrink-0" /> {error}
        </div>
      )}
      {success && (
        <div className="mb-4 flex items-center gap-2 rounded-lg border border-green-200 bg-green-50 px-4 py-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950">
          <CheckCircle2 className="h-4 w-4 shrink-0" /> Configuration saved successfully.
        </div>
      )}

      {/* Current host hint */}
      <div className="mb-6 flex items-center gap-2 rounded-lg border border-blue-200 bg-blue-50 px-4 py-3 dark:border-blue-900 dark:bg-blue-950">
        <Globe className="h-4 w-4 text-blue-500" />
        <span className="text-sm text-blue-700 dark:text-blue-300">
          You are accessing from <strong>{currentHost}</strong>. The RP ID should match this domain.
        </span>
      </div>

      <div className="space-y-5 rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-6 dark:border-gray-800 dark:bg-gray-900">
        {/* RP ID */}
        <div>
          <label className="flex items-center gap-1.5 text-sm font-medium text-gray-700 dark:text-gray-300 dark:text-gray-300">
            <Fingerprint className="h-4 w-4" /> RP ID (Domain)
          </label>
          <input
            type="text"
            value={rpId}
            onChange={e => setRpId(e.target.value)}
            placeholder={currentHost}
            className="mt-1 w-full rounded-lg border border-gray-300 dark:border-gray-600 px-3 py-2 text-sm dark:border-gray-700 dark:bg-gray-800"
          />
          <p className="mt-1 text-xs text-gray-400">
            The domain users use to access Console (e.g., {currentHost}). Passkeys are scoped to this domain.
          </p>
        </div>

        {/* Origins */}
        <div>
          <label className="flex items-center gap-1.5 text-sm font-medium text-gray-700 dark:text-gray-300 dark:text-gray-300">
            <Server className="h-4 w-4" /> Allowed Origins
          </label>
          <textarea
            value={origins}
            onChange={e => setOrigins(e.target.value)}
            rows={4}
            placeholder={`https://${currentHost}`}
            className="mt-1 w-full rounded-lg border border-gray-300 dark:border-gray-600 px-3 py-2 text-sm font-mono dark:border-gray-700 dark:bg-gray-800"
          />
          <p className="mt-1 text-xs text-gray-400">
            One origin per line. Must include protocol (https://). Usually <code>https://{currentHost}</code>.
          </p>
        </div>

        {/* Save button */}
        <div className="flex justify-end">
          <button
            onClick={handleSave}
            disabled={saving || !rpId.trim()}
            className="flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
          >
            {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
            Save Configuration
          </button>
        </div>
      </div>

      {/* Auto-fill suggestion */}
      {!rpId && (
        <button
          onClick={() => { setRpId(currentHost); setOrigins(`https://${currentHost}`); }}
          className="mt-3 text-sm text-blue-600 hover:underline"
        >
          Auto-fill with current domain ({currentHost})
        </button>
      )}
    </div>
  );
}

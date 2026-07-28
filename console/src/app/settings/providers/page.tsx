"use client";

import { useState, useEffect, useCallback } from "react";
import { Save, CheckCircle2, AlertCircle, Loader2, Server, Mail, Phone } from "lucide-react";

const API_BASE = process.env.NEXT_PUBLIC_API_BASE || "";

function getAuthToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("ggid_access_token");
}

function authHeader(): Record<string, string> {
  const token = getAuthToken();
  return token ? { Authorization: `Bearer ${token}` } : {};
}

interface ProviderConfig {
  configured: boolean;
  source: string;
  provider_type: string;
  config: Record<string, unknown>;
  enabled: boolean;
}

interface ProviderStatus {
  password: boolean;
  passkey: boolean;
  sms_otp: boolean;
  email_otp: boolean;
}

export default function ProviderSettingsPage() {
  const [scope, setScope] = useState<"instance" | "tenant">("instance");
  const [status, setStatus] = useState<ProviderStatus | null>(null);
  const [smsConfig, setSmsConfig] = useState<ProviderConfig | null>(null);
  const [emailConfig, setEmailConfig] = useState<ProviderConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState<string | null>(null);
  const [message, setMessage] = useState<{ type: "success" | "error"; text: string } | null>(null);

  // SMS form state
  const [smsProvider, setSmsProvider] = useState("twilio");
  const [smsSid, setSmsSid] = useState("");
  const [smsToken, setSmsToken] = useState("");
  const [smsFrom, setSmsFrom] = useState("");

  // Email form state
  const [emailHost, setEmailHost] = useState("");
  const [emailPort, setEmailPort] = useState("587");
  const [emailUser, setEmailUser] = useState("");
  const [emailPass, setEmailPass] = useState("");
  const [emailFrom, setEmailFrom] = useState("");

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const scopeParam = scope;
      const [statusResp, smsResp, emailResp] = await Promise.all([
        fetch(`${API_BASE}/api/v1/providers/status`).then((r) => r.json()).catch(() => null),
        fetch(`${API_BASE}/api/v1/providers/config?key=sms_provider&scope=${scopeParam}`, { headers: authHeader() })
          .then((r) => r.ok ? r.json() : null).catch(() => null) as Promise<ProviderConfig | null>,
        fetch(`${API_BASE}/api/v1/providers/config?key=email_provider&scope=${scopeParam}`, { headers: authHeader() })
          .then((r) => r.ok ? r.json() : null).catch(() => null) as Promise<ProviderConfig | null>,
      ]);
      setStatus(statusResp);
      setSmsConfig(smsResp);
      setEmailConfig(emailResp);
      if (smsResp?.config) {
        setSmsProvider((smsResp.config.provider as string) || "twilio");
        setSmsSid((smsResp.config.account_sid as string) || "");
        setSmsFrom((smsResp.config.from_number as string) || "");
      }
      if (emailResp?.config) {
        setEmailHost((emailResp.config.host as string) || "");
        setEmailPort(String(emailResp.config.port || "587"));
        setEmailUser((emailResp.config.username as string) || "");
        setEmailFrom((emailResp.config.from_email as string) || "");
      }
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { loadData(); }, [loadData, scope]);

  const saveSms = async () => {
    setSaving("sms");
    setMessage(null);
    try {
      const resp = await fetch(`${API_BASE}/api/v1/providers/config?key=sms_provider&scope=${scope}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json", ...authHeader() },
        body: JSON.stringify({
          provider_type: smsProvider,
          config: {
            provider: smsProvider,
            account_sid: smsSid,
            auth_token: smsToken || undefined,
            from_number: smsFrom,
          },
          enabled: true,
        }),
      });
      if (resp.ok) {
        setMessage({ type: "success", text: "SMS provider configuration saved successfully." });
        loadData();
      } else {
        const err = await resp.json().catch(() => ({}));
        setMessage({ type: "error", text: err.error?.message || "Failed to save SMS config" });
      }
    } catch {
      setMessage({ type: "error", text: "Network error" });
    } finally {
      setSaving(null);
    }
  };

  const saveEmail = async () => {
    setSaving("email");
    setMessage(null);
    try {
      const resp = await fetch(`${API_BASE}/api/v1/providers/config?key=email_provider&scope=${scope}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json", ...authHeader() },
        body: JSON.stringify({
          provider_type: "smtp",
          config: {
            provider: "smtp",
            host: emailHost,
            port: parseInt(emailPort) || 587,
            username: emailUser,
            password: emailPass || undefined,
            from_email: emailFrom,
            use_tls: true,
          },
          enabled: true,
        }),
      });
      if (resp.ok) {
        setMessage({ type: "success", text: "Email provider configuration saved successfully." });
        loadData();
      } else {
        const err = await resp.json().catch(() => ({}));
        setMessage({ type: "error", text: err.error?.message || "Failed to save Email config" });
      }
    } catch {
      setMessage({ type: "error", text: "Network error" });
    } finally {
      setSaving(null);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center p-12">
        <Loader2 className="h-8 w-8 animate-spin text-indigo-600" />
      </div>
    );
  }

  return (
    <div className="max-w-3xl mx-auto p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold flex items-center gap-2">
          <Server className="h-6 w-6 text-indigo-600" />
          Provider Configuration
        </h1>
        <p className="text-sm text-gray-500 mt-1">
          Configure SMS and Email providers for authentication. Settings cascade: App overrides Tenant overrides Instance (global).
        </p>
        {/* Scope selector */}
        <div className="mt-3 flex items-center gap-2">
          <span className="text-xs font-medium text-gray-500">Configuration Scope:</span>
          <button
            onClick={() => setScope("instance")}
            aria-pressed={scope === "instance"}
            aria-label="Instance scope"
            className={`rounded-lg px-3 py-1 text-xs font-medium focus:outline-none focus:ring-2 focus:ring-blue-500 ${scope === "instance" ? "bg-indigo-600 text-white" : "bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400"}`}
          >
            Instance (Global)
          </button>
          <button
            onClick={() => setScope("tenant")}
            aria-pressed={scope === "tenant"}
            aria-label="Tenant scope"
            className={`rounded-lg px-3 py-1 text-xs font-medium focus:outline-none focus:ring-2 focus:ring-blue-500 ${scope === "tenant" ? "bg-indigo-600 text-white" : "bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400"}`}
          >
            Tenant Override
          </button>
          {scope === "tenant" && (
            <span className="text-xs text-amber-500">Editing tenant-level override (leave fields empty to inherit from instance)</span>
          )}
        </div>
      </div>

      {/* Provider Status Banner */}
      {status && (
        <div className="mb-6 grid grid-cols-2 gap-3">
          {[
            { key: "password", label: "Password", icon: null },
            { key: "sms_otp", label: "SMS OTP", icon: Phone },
            { key: "email_otp", label: "Email OTP", icon: Mail },
            { key: "passkey", label: "Passkey", icon: null },
          ].map(({ key, label, icon: Icon }) => {
            const enabled = status[key as keyof ProviderStatus];
            return (
              <div
                key={key}
                className={`flex items-center gap-2 rounded-lg border p-3 ${
                  enabled ? "border-green-200 bg-green-50" : "border-gray-200 bg-gray-50"
                }`}
              >
                {Icon ? <Icon className={`h-4 w-4 ${enabled ? "text-green-600" : "text-gray-400"}`} /> : null}
                <span className={`text-sm font-medium ${enabled ? "text-green-800" : "text-gray-500"}`}>
                  {label}
                </span>
                {enabled ? (
                  <CheckCircle2 className="ml-auto h-4 w-4 text-green-600" />
                ) : (
                  <AlertCircle className="ml-auto h-4 w-4 text-gray-400" />
                )}
              </div>
            );
          })}
        </div>
      )}

      {/* Message */}
      {message && (
        <div
          className={`mb-4 rounded-lg p-3 text-sm ${
            message.type === "success"
              ? "bg-green-50 text-green-800 border border-green-200"
              : "bg-red-50 text-red-800 border border-red-200"
          }`}
        >
          {message.text}
        </div>
      )}

      {/* SMS Provider */}
      <div className="mb-6 rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-lg font-semibold flex items-center gap-2">
            <Phone className="h-5 w-5 text-indigo-600" />
            SMS Provider
          </h2>
          {smsConfig?.configured && (
            <span className="rounded-full bg-indigo-100 px-2 py-1 text-xs text-indigo-700">
              Source: {smsConfig.source}
            </span>
          )}
        </div>

        <div className="space-y-3">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Provider Type</label>
            <select
              value={smsProvider}
              onChange={(e) => setSmsProvider(e.target.value)}
              className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500"
            >
              <option value="twilio">Twilio</option>
              <option value="sns">AWS SNS</option>
              <option value="log">Log (Development Only)</option>
            </select>
          </div>

          {smsProvider === "twilio" && (
            <>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Account SID</label>
                <input
                  type="text"
                  value={smsSid}
                  onChange={(e) => setSmsSid(e.target.value)}
                  placeholder="AC..."
                  className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Auth Token {smsConfig?.config?.auth_token === "****" && <span className="text-gray-400">(configured, leave blank to keep)</span>}
                </label>
                <input
                  type="password"
                  value={smsToken}
                  onChange={(e) => setSmsToken(e.target.value)}
                  placeholder={smsConfig?.config?.auth_token === "****" ? "••••••••" : "Enter token"}
                  className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">From Number</label>
                <input
                  type="text"
                  value={smsFrom}
                  onChange={(e) => setSmsFrom(e.target.value)}
                  placeholder="+1234567890"
                  className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500"
                />
              </div>
            </>
          )}

          <button
            onClick={saveSms}
            disabled={saving === "sms"}
            aria-label="Save SMS provider config"
            className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
          >
            {saving === "sms" ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
            Save SMS Configuration
          </button>
        </div>
      </div>

      {/* Email Provider */}
      <div className="mb-6 rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-lg font-semibold flex items-center gap-2">
            <Mail className="h-5 w-5 text-indigo-600" />
            Email / SMTP Provider
          </h2>
          {emailConfig?.configured && (
            <span className="rounded-full bg-indigo-100 px-2 py-1 text-xs text-indigo-700">
              Source: {emailConfig.source}
            </span>
          )}
        </div>

        <div className="space-y-3">
          <div className="grid grid-cols-3 gap-3">
            <div className="col-span-2">
              <label className="block text-sm font-medium text-gray-700 mb-1">SMTP Host</label>
              <input
                type="text"
                value={emailHost}
                onChange={(e) => setEmailHost(e.target.value)}
                placeholder="smtp.gmail.com"
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Port</label>
              <input
                type="text"
                value={emailPort}
                onChange={(e) => setEmailPort(e.target.value)}
                placeholder="587"
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500"
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Username</label>
            <input
              type="text"
              value={emailUser}
              onChange={(e) => setEmailUser(e.target.value)}
              placeholder="user@example.com"
              className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Password {emailConfig?.config?.password === "****" && <span className="text-gray-400">(configured)</span>}
            </label>
            <input
              type="password"
              value={emailPass}
              onChange={(e) => setEmailPass(e.target.value)}
              placeholder={emailConfig?.config?.password === "****" ? "••••••••" : "Enter password"}
              className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">From Email</label>
            <input
              type="email"
              value={emailFrom}
              onChange={(e) => setEmailFrom(e.target.value)}
              placeholder="noreply@example.com"
              className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500"
            />
          </div>

          <button
            onClick={saveEmail}
            disabled={saving === "email"}
            aria-label="Save email provider config"
            className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
          >
            {saving === "email" ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
            Save Email Configuration
          </button>
        </div>
      </div>
    </div>
  );
}

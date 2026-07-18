"use client";

import { useState } from "react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";
import {
  KeyRound, Terminal, Webhook, Loader2, Send, Check,
  AlertCircle, Copy, Clock, Zap,
} from "lucide-react";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
type TabId = "token" | "api" | "webhook";

export default function IntegrationPlaygroundPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<TabId>("token");

  const tabs: { id: TabId; label: string; icon: typeof KeyRound }[] = [
    { id: "token", label: t("integrationPlayground.tabs.token"), icon: KeyRound },
    { id: "api", label: t("integrationPlayground.tabs.api"), icon: Terminal },
    { id: "webhook", label: t("integrationPlayground.tabs.webhook"), icon: Webhook },
  ];

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 p-4 md:p-8">
      <div className="max-w-4xl mx-auto">
        <div className="mb-6">
          <div className="flex items-center gap-3 mb-1">
            <Zap className="w-7 h-7 text-blue-600" />
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">{t("integrationPlayground.title")}</h1>
          </div>
          <p className="text-gray-600 dark:text-gray-400 text-sm">{t("integrationPlayground.description")}</p>
        </div>

        <div className="flex gap-1 mb-6 bg-gray-200 dark:bg-gray-800 rounded-lg p-1">
          {tabs.map(({ id, label, icon: Icon }) => (
            <button key={id} onClick={() => setTab(id)}
              className={`flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                tab === id ? "bg-white dark:bg-gray-700 text-blue-600 dark:text-blue-400 shadow-sm" : "text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white"
              }`}>
              <Icon className="w-4 h-4" />{label}
            </button>
          ))}
        </div>

        {tab === "token" && <TokenTester />}
        {tab === "api" && <APITester />}
        {tab === "webhook" && <WebhookTester />}
      </div>
    </div>
  );
}

// ============ Token Tester ============

function TokenTester() {
  const t = useTranslations();
  const [email, setEmail] = useState("admin@ggid.dev");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<{ token: string; header: Record<string, unknown>; payload: Record<string, unknown> } | null>(null);
  const [error, setError] = useState("");

  const login = async () => {
    setLoading(true); setError(""); setResult(null);
    try {
      const res = await fetch(`${API_BASE}/api/v1/auth/login`, {
        method: "POST", headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password }),
      });
      if (!res.ok) { setError(`HTTP ${res.status}`); return; }
      const data = await res.json();
      const token = data.access_token || data.token;
      if (!token) { setError("No token in response"); return; }
      const parts = token.split(".");
      if (parts.length !== 3) { setResult({ token, header: {}, payload: { note: "Not a JWT" } }); return; }
      const decode = (s: string) => JSON.parse(atob(s.replace(/-/g, "+").replace(/_/g, "/")));
      setResult({ token, header: decode(parts[0]), payload: decode(parts[1]) });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Request failed");
    } finally { setLoading(false); }
  };

  const copyToken = () => { if (result) { navigator.clipboard.writeText(result.token); } };

  return (
    <div className="space-y-4">
      <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6 space-y-4">
        <h3 className="text-sm font-semibold text-gray-900 dark:text-white">{t("integrationPlayground.token.title")}</h3>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          <div>
            <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">{t("integrationPlayground.token.email")}</label>
            <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} placeholder={t("integrationPlayground.token.emailPlaceholder")}
              className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white" />
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">{t("integrationPlayground.token.password")}</label>
            <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} placeholder={t("integrationPlayground.token.passwordPlaceholder")}
              className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white" />
          </div>
        </div>
        <button onClick={login} disabled={loading || !email || !password}
          className="flex items-center gap-2 px-6 py-2 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg text-sm font-medium">
          {loading ? <Loader2 className="w-4 h-4 animate-spin" /> : <KeyRound className="w-4 h-4" />}
          {loading ? t("integrationPlayground.token.logging") : t("integrationPlayground.token.login")}
        </button>
      </div>

      {error && <div className="flex items-center gap-2 px-4 py-3 rounded-lg bg-red-50 dark:bg-red-950/30 text-red-600 text-sm"><AlertCircle className="w-4 h-4" />{error}</div>}

      {result && (
        <div className="space-y-4">
          {/* Raw token */}
          <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-4">
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs font-medium text-gray-500">{t("integrationPlayground.token.rawToken")}</span>
              <button onClick={copyToken} className="flex items-center gap-1 text-xs text-blue-600 hover:underline"><Copy className="w-3 h-3" />{t("integrationPlayground.token.copyToken")}</button>
            </div>
            <code className="block text-xs font-mono text-gray-900 dark:text-gray-300 break-all bg-gray-50 dark:bg-gray-800 p-3 rounded">{result.token}</code>
          </div>

          {/* JWT Header */}
          <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-4">
            <span className="text-xs font-medium text-gray-500 mb-2 block">{t("integrationPlayground.token.tokenHeader")}</span>
            <pre className="text-xs p-3 bg-gray-50 dark:bg-gray-800 rounded overflow-x-auto text-gray-700 dark:text-gray-300">{JSON.stringify(result.header, null, 2)}</pre>
          </div>

          {/* Decoded Payload */}
          <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-4">
            <span className="text-xs font-medium text-gray-500 mb-2 block">{t("integrationPlayground.token.decodedPayload")}</span>
            <pre className="text-xs p-3 bg-gray-50 dark:bg-gray-800 rounded overflow-x-auto text-gray-700 dark:text-gray-300">{JSON.stringify(result.payload, null, 2)}</pre>
          </div>
        </div>
      )}
    </div>
  );
}

// ============ API Tester ============

function APITester() {
  const t = useTranslations();
  const [method, setMethod] = useState("GET");
  const [endpoint, setEndpoint] = useState("/api/v1/users/me");
  const [params, setParams] = useState("");
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<{ status: number; time: number; body: string } | null>(null);
  const [error, setError] = useState("");

  const presets: { label: string; method: string; path: string }[] = [
    { label: t("integrationPlayground.api.presetLogin"), method: "POST", path: "/api/v1/auth/login" },
    { label: t("integrationPlayground.api.presetMe"), method: "GET", path: "/api/v1/users/me" },
    { label: t("integrationPlayground.api.presetUsers"), method: "GET", path: "/api/v1/users" },
    { label: t("integrationPlayground.api.presetRoles"), method: "GET", path: "/api/v1/roles" },
  ];

  const send = async () => {
    setLoading(true); setError(""); setResult(null);
    const start = Date.now();
    try {
      const opts: RequestInit = { method, headers: { ...authHeader() } };
      if ((method === "POST" || method === "PUT") && params) {
        opts.headers = { "Content-Type": "application/json", ...authHeader() };
        opts.body = params;
      }
      const res = await fetch(`${API_BASE}${endpoint}`, opts);
      const body = await res.text();
      setResult({ status: res.status, time: Date.now() - start, body });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Request failed");
    } finally { setLoading(false); }
  };

  const applyPreset = (p: { method: string; path: string }) => {
    setMethod(p.method); setEndpoint(p.path);
    if (p.method === "POST") setParams(JSON.stringify({ email: "admin@ggid.dev", password: "Admin@123456" }, null, 2));
    else setParams("");
  };

  return (
    <div className="space-y-4">
      {/* Presets */}
      <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-4">
        <div className="flex flex-wrap gap-2">
          {presets.map((p) => (
            <button key={p.label} onClick={() => applyPreset(p)} className="px-3 py-1.5 rounded-lg text-xs font-medium bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700">
              {p.label}
            </button>
          ))}
        </div>
      </div>

      {/* Request builder */}
      <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6 space-y-4">
        <h3 className="text-sm font-semibold text-gray-900 dark:text-white">{t("integrationPlayground.api.title")}</h3>
        <div className="flex gap-2">
          <select value={method} onChange={(e) => setMethod(e.target.value)}
            className="px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm font-mono text-gray-900 dark:text-white">
            <option>GET</option><option>POST</option><option>PUT</option><option>DELETE</option>
          </select>
          <input type="text" value={endpoint} onChange={(e) => setEndpoint(e.target.value)}
            className="flex-1 px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm font-mono text-gray-900 dark:text-white" />
        </div>
        {(method === "POST" || method === "PUT") && (
          <div>
            <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">{t("integrationPlayground.api.params")}</label>
            <textarea value={params} onChange={(e) => setParams(e.target.value)} placeholder={t("integrationPlayground.api.paramsPlaceholder")} rows={4}
              className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm font-mono text-gray-900 dark:text-white" />
          </div>
        )}
        <button onClick={send} disabled={loading || !endpoint}
          className="flex items-center gap-2 px-6 py-2 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg text-sm font-medium">
          {loading ? <Loader2 className="w-4 h-4 animate-spin" /> : <Send className="w-4 h-4" />}
          {loading ? t("integrationPlayground.api.sending") : t("integrationPlayground.api.send")}
        </button>
      </div>

      {error && <div className="flex items-center gap-2 px-4 py-3 rounded-lg bg-red-50 dark:bg-red-950/30 text-red-600 text-sm"><AlertCircle className="w-4 h-4" />{error}</div>}

      {result && (
        <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
          <div className="flex items-center gap-4 mb-3">
            <span className="text-sm font-semibold text-gray-900 dark:text-white">{t("integrationPlayground.api.response")}</span>
            <span className={`px-2 py-0.5 text-xs rounded-full font-medium ${result.status < 300 ? "bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-300" : "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300"}`}>
              {result.status}
            </span>
            <span className="flex items-center gap-1 text-xs text-gray-400"><Clock className="w-3 h-3" />{result.time}ms</span>
          </div>
          <pre className="text-xs p-4 bg-gray-900 dark:bg-gray-800 rounded-lg overflow-x-auto text-gray-300 max-h-96">{result.body}</pre>
        </div>
      )}
    </div>
  );
}

// ============ Webhook Tester ============

function WebhookTester() {
  const t = useTranslations();
  const [url, setUrl] = useState("");
  const [eventType, setEventType] = useState("user.created");
  const [payload, setPayload] = useState(JSON.stringify({ event: "user.created", data: { id: "u-123", email: "newuser@company.com", created_at: new Date().toISOString() } }, null, 2));
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<{ status: number; body: string } | null>(null);
  const [error, setError] = useState("");

  const eventTypes = ["user.created", "user.deleted", "auth.login", "role.changed"];

  const send = async () => {
    setLoading(true); setError(""); setResult(null);
    try {
      const res = await fetch(url, {
        method: "POST", headers: { "Content-Type": "application/json" },
        body: payload,
      });
      const body = await res.text();
      setResult({ status: res.status, body });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Request failed");
    } finally { setLoading(false); }
  };

  const updateEventType = (type: string) => {
    setEventType(type);
    setPayload(JSON.stringify({ event: type, data: { id: `u-${Date.now()}`, timestamp: new Date().toISOString() } }, null, 2));
  };

  return (
    <div className="space-y-4">
      <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6 space-y-4">
        <div>
          <h3 className="text-sm font-semibold text-gray-900 dark:text-white">{t("integrationPlayground.webhook.title")}</h3>
          <p className="text-xs text-gray-500 mt-1">{t("integrationPlayground.webhook.description")}</p>
        </div>
        <div>
          <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">{t("integrationPlayground.webhook.webhookUrl")}</label>
          <input type="text" value={url} onChange={(e) => setUrl(e.target.value)} placeholder={t("integrationPlayground.webhook.webhookUrlPlaceholder")}
            className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white" />
        </div>
        <div>
          <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-2">{t("integrationPlayground.webhook.eventType")}</label>
          <div className="flex flex-wrap gap-2">
            {eventTypes.map((e) => (
              <button key={e} onClick={() => updateEventType(e)} className={`px-3 py-1.5 rounded-lg text-xs font-medium border-2 transition-all ${eventType === e ? "border-blue-500 bg-blue-50 dark:bg-blue-950/20 text-blue-700 dark:text-blue-300" : "border-gray-200 dark:border-gray-700 text-gray-500"}`}>{e}</button>
            ))}
          </div>
        </div>
        <div>
          <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">{t("integrationPlayground.webhook.payload")}</label>
          <textarea value={payload} onChange={(e) => setPayload(e.target.value)} rows={6}
            className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm font-mono text-gray-900 dark:text-white" />
        </div>
        <button onClick={send} disabled={loading || !url}
          className="flex items-center gap-2 px-6 py-2 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg text-sm font-medium">
          {loading ? <Loader2 className="w-4 h-4 animate-spin" /> : <Send className="w-4 h-4" />}
          {loading ? t("integrationPlayground.webhook.sending") : t("integrationPlayground.webhook.send")}
        </button>
      </div>

      {error && <div className="flex items-center gap-2 px-4 py-3 rounded-lg bg-red-50 dark:bg-red-950/30 text-red-600 text-sm"><AlertCircle className="w-4 h-4" />{error}</div>}

      {result && (
        <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
          <div className="flex items-center gap-3 mb-3">
            <span className="text-sm font-semibold text-gray-900 dark:text-white">{t("integrationPlayground.webhook.response")}</span>
            <span className={`px-2 py-0.5 text-xs rounded-full font-medium ${result.status < 300 ? "bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-300" : "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300"}`}>{result.status}</span>
          </div>
          <pre className="text-xs p-4 bg-gray-900 dark:bg-gray-800 rounded-lg overflow-x-auto text-gray-300 max-h-64">{result.body || "(empty response body)"}</pre>
        </div>
      )}
    </div>
  );
}

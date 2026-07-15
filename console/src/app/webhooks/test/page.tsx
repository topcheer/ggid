"use client";

import { useState } from "react";
import { useApi } from "@/lib/api";
import { Send, RefreshCw, Shield, Clock, AlertCircle, CheckCircle2, Zap } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface DeliveryAttempt {
  id: string;
  attempt: number;
  timestamp: string;
  status: number;
  duration: number;
  response: string;
  success: boolean;
}

const WEBHOOK_EVENTS = [
  { id: "user.created", name: "User Created", desc: "Triggered when a new user registers" },
  { id: "user.updated", name: "User Updated", desc: "Triggered when user profile changes" },
  { id: "user.deleted", name: "User Deleted", desc: "Triggered when a user is removed" },
  { id: "auth.login", name: "User Login", desc: "Triggered on successful authentication" },
  { id: "auth.logout", name: "User Logout", desc: "Triggered on session termination" },
  { id: "auth.mfa_challenge", name: "MFA Challenge", desc: "Triggered when MFA is required" },
  { id: "role.assigned", name: "Role Assigned", desc: "Triggered when a role is granted" },
  { id: "org.member_added", name: "Org Member Added", desc: "Triggered on org membership change" },
];

export default function WebhookTesterPage() {
  const t = useTranslations();

  const { API_BASE, TENANT_ID } = useApi();
  const [selectedEvent, setSelectedEvent] = useState("user.created");
  const [webhookUrl, setWebhookUrl] = useState("https://example.com/webhook");
  const [payload, setPayload] = useState(JSON.stringify({
    event: "user.created",
    data: { id: "usr_abc123", email: "john@example.com", name: "John Doe", tenant_id: TENANT_ID },
    timestamp: new Date().toISOString(),
  }, null, 2));
  const [hmacSecret, setHmacSecret] = useState("whsec_XXXXYYYYZZZZ");
  const [hmacSignature, setHmacSignature] = useState("");
  const [sending, setSending] = useState(false);
  const [attempts, setAttempts] = useState<DeliveryAttempt[]>([]);
  const [retrying, setRetrying] = useState(false);

  const generateHmac = async () => {
    // Simulate HMAC generation
    const sig = "sha256=" + Array.from(crypto.getRandomValues(new Uint8Array(32)))
      .map(b => b.toString(16).padStart(2, "0")).join("");
    setHmacSignature(sig);
    return sig;
  };

  const handleSend = async () => {
    setSending(true);
    const sig = await generateHmac();
    const attempt: DeliveryAttempt = {
      id: Math.random().toString(36).substring(7),
      attempt: attempts.length + 1,
      timestamp: new Date().toLocaleTimeString(),
      status: 200,
      duration: Math.floor(Math.random() * 200 + 50),
      response: '{"status": "ok", "received": true}',
      success: true,
    };
    setTimeout(() => {
      // Simulate occasional failure for retry demo
      if (Math.random() > 0.7) {
        attempt.status = 500;
        attempt.success = false;
        attempt.response = '{"error": "Internal Server Error"}';
      }
      setAttempts(prev => [attempt, ...prev]);
      setSending(false);
    }, Math.floor(Math.random() * 500 + 200));
  };

  const handleRetry = async (id: string) => {
    setRetrying(true);
    setTimeout(() => {
      setAttempts(prev => prev.map(a => a.id === id ? { ...a, status: 200, success: true, response: '{"status": "ok"}', attempt: a.attempt + 1 } : a));
      setRetrying(false);
    }, 800);
  };

  const updateEventPayload = (eventId: string) => {
    setSelectedEvent(eventId);
    const sampleData: Record<string, unknown> = {
      "user.created": { id: "usr_abc123", email: "john@example.com", name: "John Doe" },
      "user.updated": { id: "usr_abc123", changes: { name: "Jane Doe" } },
      "user.deleted": { id: "usr_abc123", deleted_at: new Date().toISOString() },
      "auth.login": { user_id: "usr_abc123", ip: "192.168.1.1", device: "Chrome/macOS" },
      "auth.logout": { user_id: "usr_abc123", session_id: "ses_xyz" },
      "auth.mfa_challenge": { user_id: "usr_abc123", method: "totp", challenge_id: "mfa_123" },
      "role.assigned": { user_id: "usr_abc123", role: "admin", assigned_by: "usr_admin" },
      "org.member_added": { org_id: "org_123", user_id: "usr_abc123", role: "member" },
    };
    setPayload(JSON.stringify({ event: eventId, data: sampleData[eventId] || {}, timestamp: new Date().toISOString() }, null, 2));
  };

  const backoffMs = (attempt: number) => Math.min(1000 * Math.pow(2, attempt - 1), 30000);

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Webhook Tester</h1>
        <p className="text-sm text-gray-500 mt-1">Test webhook deliveries with custom payloads</p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Left: Configuration */}
        <div className="space-y-4">
          {/* Event Selector */}
          <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-4">
            <label className="text-sm font-semibold text-gray-900 dark:text-white mb-2 block">Event Type</label>
            <select value={selectedEvent} onChange={(e) => updateEventPayload(e.target.value)}
              className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-700 rounded-lg bg-transparent text-gray-900 dark:text-white">
              {WEBHOOK_EVENTS.map(ev => <option key={ev.id} value={ev.id}>{ev.name} — {ev.desc}</option>)}
            </select>
          </div>

          {/* Webhook URL */}
          <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-4">
            <label className="text-sm font-semibold text-gray-900 dark:text-white mb-2 block">Webhook URL</label>
            <input type="text" value={webhookUrl} onChange={(e) => setWebhookUrl(e.target.value)}
              className="w-full px-3 py-2 text-sm font-mono border border-gray-300 dark:border-gray-700 rounded-lg bg-transparent text-gray-900 dark:text-white" />
          </div>

          {/* Payload Editor */}
          <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-4">
            <label className="text-sm font-semibold text-gray-900 dark:text-white mb-2 block">Payload (JSON)</label>
            <textarea value={payload} onChange={(e) => setPayload(e.target.value)}
              className="w-full h-48 px-3 py-2 text-sm font-mono border border-gray-300 dark:border-gray-700 rounded-lg bg-gray-50 dark:bg-gray-800 text-gray-900 dark:text-white resize-none" />
          </div>

          {/* HMAC Section */}
          <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-4">
            <div className="flex items-center gap-2 mb-3">
              <Shield className="w-4 h-4 text-indigo-500" />
              <label className="text-sm font-semibold text-gray-900 dark:text-white">HMAC Signature</label>
            </div>
            <div className="space-y-2">
              <div>
                <span className="text-xs text-gray-400">Secret</span>
                <input type="text" value={hmacSecret} onChange={(e) => setHmacSecret(e.target.value)} readOnly
                  className="w-full mt-1 px-3 py-1.5 text-xs font-mono border border-gray-300 dark:border-gray-700 rounded-lg bg-gray-50 dark:bg-gray-800 text-gray-500" />
              </div>
              {hmacSignature && (
                <div>
                  <span className="text-xs text-gray-400">X-GGID-Signature</span>
                  <div className="mt-1 px-3 py-1.5 text-xs font-mono bg-green-50 dark:bg-green-950/30 border border-green-200 dark:border-green-800 rounded-lg text-green-700 dark:text-green-400 break-all">
                    {hmacSignature}
                  </div>
                </div>
              )}
            </div>
          </div>

          <button onClick={handleSend} disabled={sending}
            className="w-full flex items-center justify-center gap-2 px-4 py-2.5 text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 rounded-lg transition disabled:opacity-50">
            {sending ? <><RefreshCw className="w-4 h-4 animate-spin" /> Sending...</> : <><Send className="w-4 h-4" /> Send Test Delivery</>}
          </button>
        </div>

        {/* Right: Delivery Results */}
        <div className="space-y-4">
          {/* Retry with Backoff Visualization */}
          {attempts.some(a => !a.success) && (
            <div className="bg-amber-50 dark:bg-amber-950/30 border border-amber-200 dark:border-amber-800 rounded-xl p-4">
              <div className="flex items-center gap-2 mb-3">
                <Zap className="w-4 h-4 text-amber-500" />
                <h3 className="text-sm font-semibold text-amber-900 dark:text-amber-200">Retry with Exponential Backoff</h3>
              </div>
              <div className="space-y-1">
                {[1, 2, 3, 4, 5].map(n => (
                  <div key={n} className="flex items-center gap-2">
                    <span className="text-xs text-amber-700 dark:text-amber-400 w-16">Attempt {n}</span>
                    <div className="flex-1 bg-amber-100 dark:bg-amber-900/40 rounded-full h-2 overflow-hidden">
                      <div className="bg-amber-500 h-full rounded-full transition-all" style={{ width: `${Math.min(100, n * 20)}%` }} />
                    </div>
                    <span className="text-xs text-amber-600 dark:text-amber-500 w-20 text-right">{backoffMs(n)}ms</span>
                  </div>
                ))}
              </div>
              <p className="text-xs text-amber-600 dark:text-amber-500 mt-2">Max retries: 5. Backoff: 2^n seconds (capped at 30s).</p>
            </div>
          )}

          {/* Delivery Attempts */}
          <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-4">
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-sm font-semibold text-gray-900 dark:text-white">Delivery History</h3>
              {attempts.length > 0 && <span className="text-xs text-gray-400">{attempts.filter(a => a.success).length}/{attempts.length} delivered</span>}
            </div>

            {attempts.length === 0 ? (
              <div className="text-center py-12 text-gray-400 text-sm">No deliveries yet. Send a test above.</div>
            ) : (
              <div className="space-y-3">
                {attempts.map(attempt => (
                  <div key={attempt.id} className="border border-gray-200 dark:border-gray-800 rounded-lg p-3">
                    <div className="flex items-center justify-between mb-2">
                      <div className="flex items-center gap-2">
                        {attempt.success ? (
                          <CheckCircle2 className="w-4 h-4 text-green-500" />
                        ) : (
                          <AlertCircle className="w-4 h-4 text-red-500" />
                        )}
                        <span className={`text-xs font-mono px-1.5 py-0.5 rounded ${attempt.success ? "bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-400" : "bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-400"}`}>
                          {attempt.status}
                        </span>
                        <span className="text-xs text-gray-400 flex items-center gap-1">
                          <Clock className="w-3 h-3" /> {attempt.duration}ms
                        </span>
                        <span className="text-xs text-gray-400">Attempt {attempt.attempt}</span>
                      </div>
                      <div className="flex items-center gap-2">
                        <span className="text-xs text-gray-400">{attempt.timestamp}</span>
                        {!attempt.success && (
                          <button onClick={() => handleRetry(attempt.id)} disabled={retrying}
                            className="flex items-center gap-1 px-2 py-1 text-xs text-indigo-600 dark:text-indigo-400 hover:bg-indigo-50 dark:hover:bg-indigo-950/30 rounded transition">
                            <RefreshCw className={`w-3 h-3 ${retrying ? "animate-spin" : ""}`} /> Retry
                          </button>
                        )}
                      </div>
                    </div>
                    <div className="flex gap-2 text-xs">
                      <div className="flex-1">
                        <span className="text-gray-400">Request:</span>
                        <pre className="mt-1 p-2 bg-gray-50 dark:bg-gray-800 rounded text-xs font-mono text-gray-600 dark:text-gray-400 overflow-auto max-h-24">POST {webhookUrl}\nX-GGID-Signature: {hmacSignature.slice(0, 30)}...\n\n{payload.slice(0, 100)}...</pre>
                      </div>
                      <div className="flex-1">
                        <span className="text-gray-400">Response:</span>
                        <pre className="mt-1 p-2 bg-gray-50 dark:bg-gray-800 rounded text-xs font-mono text-gray-600 dark:text-gray-400 overflow-auto max-h-24">{attempt.response}</pre>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

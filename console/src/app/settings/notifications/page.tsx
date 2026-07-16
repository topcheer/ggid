"use client";

import { useState, useEffect, useRef } from "react";
import { useApi } from "@/lib/api";
import { Mail, Save, Loader2, Send, Smartphone, Eye, Code2 } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface EmailTemplate {
  subject: string;
  body: string;
}

interface SmsTemplate {
  body: string;
}

type TemplateType = "welcome" | "password-reset" | "mfa-enrollment" | "login-alert" | "session-revoked";

const STORAGE_KEY = "ggid_notification_templates";

const TEMPLATE_LABELS: Record<TemplateType, string> = {
  welcome: "Welcome Email",
  "password-reset": "Password Reset",
  "mfa-enrollment": "MFA Enrollment",
  "login-alert": "Login Alert",
  "session-revoked": "Session Revoked",
};

const TEMPLATE_TYPES: TemplateType[] = ["welcome", "password-reset", "mfa-enrollment", "login-alert", "session-revoked"];

const VARIABLES = [
  { key: "{{user.name}}", label: "User Name", sample: "John Doe" },
  { key: "{{user.email}}", label: "User Email", sample: "john@example.com" },
  { key: "{{code}}", label: "Verification Code", sample: "829374" },
  { key: "{{ip}}", label: "IP Address", sample: "192.168.1.42" },
  { key: "{{device}}", label: "Device", sample: "Chrome on macOS" },
  { key: "{{reset_link}}", label: "Reset Link", sample: "https://app.example.com/reset?token=abc123" },
  { key: "{{login_time}}", label: "Login Time", sample: "2024-01-15 14:32 UTC" },
];

const SAMPLE_DATA: Record<string, string> = Object.fromEntries(VARIABLES.map((v) => [v.key, v.sample]));

const defaultEmailTemplates: Record<TemplateType, EmailTemplate> = {
  welcome: {
    subject: "Welcome to {{user.name}}!",
    body: `<h1>Welcome, {{user.name}}!</h1>\n<p>Your account has been created successfully with email <strong>{{user.email}}</strong>.</p>\n<p>If you have any questions, feel free to reach out to our support team.</p>`,
  },
  "password-reset": {
    subject: "Password Reset Request",
    body: `<h1>Password Reset</h1>\n<p>Click the link below to reset your password:</p>\n<p><a href="{{reset_link}}">{{reset_link}}</a></p>\n<p>This link expires in 30 minutes.</p>`,
  },
  "mfa-enrollment": {
    subject: "MFA Enrollment - Verification Code",
    body: `<h1>Verify Your Email</h1>\n<p>Your MFA verification code is:</p>\n<h2 style="font-family: monospace; letter-spacing: 4px;">{{code}}</h2>\n<p>Enter this code in the application to complete enrollment.</p>`,
  },
  "login-alert": {
    subject: "New Login from {{device}}",
    body: `<h1>New Login Detected</h1>\n<p>We detected a new login to your account:</p>\n<ul>\n<li><strong>Device:</strong> {{device}}</li>\n<li><strong>IP:</strong> {{ip}}</li>\n<li><strong>Time:</strong> {{login_time}}</li>\n</ul>\n<p>If this wasn't you, please secure your account immediately.</p>`,
  },
  "session-revoked": {
    subject: "Session Revoked",
    body: `<h1>Your Session Was Revoked</h1>\n<p>Your session on {{device}} (IP: {{ip}}) at {{login_time}} has been revoked by an administrator.</p>\n<p>If you believe this is an error, contact support.</p>`,
  },
};

const defaultSmsTemplates: Record<TemplateType, SmsTemplate> = {
  welcome: { body: "Welcome to GGID, {{user.name}}! Your account is ready." },
  "password-reset": { body: "GGID: Your password reset link: {{reset_link}}" },
  "mfa-enrollment": { body: "GGID verification code: {{code}}. Expires in 10 minutes." },
  "login-alert": { body: "GGID: New login from {{device}} at {{login_time}}. IP: {{ip}}." },
  "session-revoked": { body: "GGID: Your session on {{device}} has been revoked." },
};

export default function NotificationsPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [emailTemplates, setEmailTemplates] = useState<Record<TemplateType, EmailTemplate>>(defaultEmailTemplates);
  const [smsTemplates, setSmsTemplates] = useState<Record<TemplateType, SmsTemplate>>(defaultSmsTemplates);
  const [selectedType, setSelectedType] = useState<TemplateType>("welcome");
  const [msg, setMsg] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [sendingTest, setSendingTest] = useState(false);
  const [activeTab, setActiveTab] = useState<"email" | "sms">("email");

  const bodyRef = useRef<HTMLTextAreaElement>(null);

  // Load from localStorage
  useEffect(() => {
    const stored = typeof window !== "undefined" ? localStorage.getItem(STORAGE_KEY) : null;
    if (stored) {
      try {
        const parsed = JSON.parse(stored);
        if (parsed.email) setEmailTemplates({ ...defaultEmailTemplates, ...parsed.email });
        if (parsed.sms) setSmsTemplates({ ...defaultSmsTemplates, ...parsed.sms });
      } catch {
        // ignore
      }
    }
  }, []);

  useEffect(() => {
    if (msg) {
      const t = setTimeout(() => setMsg(null), 3000);
      return () => clearTimeout(t);
    }
  }, [msg]);

  const currentEmail = emailTemplates[selectedType];
  const currentSms = smsTemplates[selectedType];

  const renderWithSample = (text: string): string => {
    let result = text;
    VARIABLES.forEach((v) => {
      result = result.replaceAll(v.key, v.sample);
    });
    return result;
  };

  const insertVariable = (varKey: string) => {
    const textarea = bodyRef.current;
    if (!textarea) return;
    const start = textarea.selectionStart;
    const end = textarea.selectionEnd;
    const body = activeTab === "email" ? currentEmail.body : currentSms.body;
    const newBody = body.slice(0, start) + varKey + body.slice(end);
    if (activeTab === "email") {
      setEmailTemplates({
        ...emailTemplates,
        [selectedType]: { ...currentEmail, body: newBody },
      });
    } else {
      setSmsTemplates({
        ...smsTemplates,
        [selectedType]: { ...currentSms, body: newBody },
      });
    }
    // Restore cursor after variable
    requestAnimationFrame(() => {
      textarea.focus();
      const pos = start + varKey.length;
      textarea.setSelectionRange(pos, pos);
    });
  };

  const handleSave = async () => {
    setSaving(true);
    const payload = activeTab === "email" ? currentEmail : currentSms;
    try {
      await apiFetch(`/api/v1/settings/notifications/templates/${selectedType}`, {
        method: "PUT",
        body: JSON.stringify({ type: selectedType, channel: activeTab, ...payload }),
      });
      setMsg(`${activeTab === "email" ? "Email" : "SMS"} template saved to server`);
    } catch {
      localStorage.setItem(STORAGE_KEY, JSON.stringify({ email: emailTemplates, sms: smsTemplates }));
      setMsg("Endpoint unavailable — saved to localStorage");
    } finally {
      setSaving(false);
    }
  };

  const handleSendTest = async () => {
    setSendingTest(true);
    try {
      await apiFetch("/api/v1/notifications/test", {
        method: "POST",
        body: JSON.stringify({
          type: selectedType,
          channel: activeTab,
          subject: activeTab === "email" ? renderWithSample(currentEmail.subject) : undefined,
          body: renderWithSample(activeTab === "email" ? currentEmail.body : currentSms.body),
        }),
      });
      setMsg("Test notification sent to your account");
    } catch (err) {
      setMsg(err instanceof Error ? `Test send failed: ${err.message}` : "Test send failed");
    } finally {
      setSendingTest(false);
    }
  };

  const smsCharCount = currentSms.body.length;

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-2xl font-bold dark:text-gray-100">
          <Mail className="h-6 w-6 text-brand-600" /> {t("notifications.title")}
        </h1>
        <div className="flex gap-2">
          <button
            onClick={handleSendTest}
            disabled={sendingTest}
            className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700 disabled:opacity-50"
          >
            {sendingTest ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />} Send Test
          </button>
          <button
            onClick={handleSave}
            disabled={saving}
            className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
          >
            {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />} Save Template
          </button>
        </div>
      </div>

      {msg && (
        <div role="status" className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400">
          {msg}
        </div>
      )}

      {/* Template type selector */}
      <div className="mb-4">
        <label className="mb-1.5 block text-xs font-medium text-gray-500">Template Type</label>
        <select
          value={selectedType}
          onChange={(e) => setSelectedType(e.target.value as TemplateType)}
          className="w-full max-w-xs rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
        >
          {TEMPLATE_TYPES.map((t) => (
            <option key={t} value={t}>{TEMPLATE_LABELS[t]}</option>
          ))}
        </select>
      </div>

      {/* Channel tabs */}
      <div className="mb-4 flex gap-1 border-b border-gray-200 dark:border-gray-700">
        <button
          onClick={() => setActiveTab("email")}
          className={`flex items-center gap-1.5 px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
            activeTab === "email"
              ? "border-brand-600 text-brand-600"
              : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
          }`}
        >
          <Mail className="h-4 w-4" /> Email
        </button>
        <button
          onClick={() => setActiveTab("sms")}
          className={`flex items-center gap-1.5 px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
            activeTab === "sms"
              ? "border-brand-600 text-brand-600"
              : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
          }`}
        >
          <Smartphone className="h-4 w-4" /> SMS
        </button>
      </div>

      <div className="grid gap-4 lg:grid-cols-[1fr_220px]">
        {/* Editor area */}
        <div className="space-y-4">
          {activeTab === "email" ? (
            <>
              {/* Subject */}
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500">Subject Line</label>
                <input
                  value={currentEmail.subject}
                  onChange={(e) =>
                    setEmailTemplates({
                      ...emailTemplates,
                      [selectedType]: { ...currentEmail, subject: e.target.value },
                    })
                  }
                  placeholder="Email subject..."
                  className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
                />
              </div>

              {/* HTML body */}
              <div>
                <label className="mb-1 flex items-center gap-1.5 text-xs font-medium text-gray-500">
                  <Code2 className="h-3.5 w-3.5" /> HTML Body
                </label>
                <textarea
                  ref={bodyRef}
                  value={currentEmail.body}
                  onChange={(e) =>
                    setEmailTemplates({
                      ...emailTemplates,
                      [selectedType]: { ...currentEmail, body: e.target.value },
                    })
                  }
                  rows={12}
                  placeholder="Enter HTML email content..."
                  className="w-full rounded-lg border border-gray-300 px-3 py-2 font-mono text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
                />
              </div>

              {/* Live preview */}
              <div>
                <label className="mb-1 flex items-center gap-1.5 text-xs font-medium text-gray-500">
                  <Eye className="h-3.5 w-3.5" /> Live Preview
                </label>
                <div className="overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-800">
                  <div className="border-b border-gray-100 bg-gray-50 px-4 py-2 dark:border-gray-700 dark:bg-gray-900">
                    <p className="text-xs font-semibold text-gray-600 dark:text-gray-400">
                      {renderWithSample(currentEmail.subject)}
                    </p>
                  </div>
                  <div
                    className="p-4 text-sm text-gray-700 dark:text-gray-300 [&_a]:text-brand-600 [&_a]:underline"
                    dangerouslySetInnerHTML={{ __html: renderWithSample(currentEmail.body) }}
                  />
                </div>
              </div>
            </>
          ) : (
            <>
              {/* SMS body */}
              <div>
                <label className="mb-1 flex items-center gap-1.5 text-xs font-medium text-gray-500">
                  <Smartphone className="h-3.5 w-3.5" /> SMS Body (Plain Text)
                </label>
                <textarea
                  ref={bodyRef}
                  value={currentSms.body}
                  onChange={(e) =>
                    setSmsTemplates({
                      ...smsTemplates,
                      [selectedType]: { ...currentSms, body: e.target.value },
                    })
                  }
                  rows={6}
                  placeholder="Enter SMS message..."
                  className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
                />
                <div className="mt-1 flex items-center justify-between">
                  <p className="text-xs text-gray-400">{smsCharCount} characters</p>
                  <span
                    className={`text-xs font-medium ${
                      smsCharCount > 160 ? "text-red-600 dark:text-red-400" : "text-green-600 dark:text-green-400"
                    }`}
                  >
                    {smsCharCount > 160 ? `Exceeds 160 chars (${Math.ceil(smsCharCount / 160)} SMS)` : `${160 - smsCharCount} chars remaining`}
                  </span>
                </div>
              </div>

              {/* SMS preview */}
              <div>
                <label className="mb-1 flex items-center gap-1.5 text-xs font-medium text-gray-500">
                  <Eye className="h-3.5 w-3.5" /> Preview
                </label>
                <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-800">
                  <p className="text-sm text-gray-700 dark:text-gray-300">{renderWithSample(currentSms.body)}</p>
                </div>
              </div>
            </>
          )}
        </div>

        {/* Variable picker sidebar */}
        <div className="lg:sticky lg:top-4 lg:self-start">
          <div className="rounded-lg border border-gray-200 bg-white p-3 shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <h3 className="mb-1 text-xs font-semibold uppercase text-gray-500">Variables</h3>
            <p className="mb-3 text-xs text-gray-400">Click to insert at cursor</p>
            <div className="space-y-1">
              {VARIABLES.map((v) => (
                <button
                  key={v.key}
                  onClick={() => insertVariable(v.key)}
                  className="block w-full rounded border border-gray-200 px-2.5 py-1.5 text-left transition-colors hover:border-brand-300 hover:bg-brand-50 dark:border-gray-600 dark:hover:border-brand-700 dark:hover:bg-brand-950/30"
                >
                  <p className="text-xs font-mono text-brand-600 dark:text-brand-400">{v.key}</p>
                  <p className="text-xs text-gray-400">{v.label}</p>
                </button>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

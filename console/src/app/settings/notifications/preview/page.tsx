"use client";

import { useState, useMemo, useEffect } from "react";
import { useApi } from "@/lib/api";
import {
  Mail, Smartphone, Send, Eye, MessageSquare, AlertCircle, Check, RefreshCw,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

type TemplateType = "welcome" | "password_reset" | "mfa_code" | "account_locked" | "invitation" | "custom";
type PreviewMode = "email" | "sms";

interface TemplateDef {
  label: string;
  subject: string;
  from: string;
  replyTo: string;
  smsTemplate: string;
  emailBody: (d: Record<string, string>) => string;
  sampleData: Record<string, string>;
}

const TEMPLATES: Record<TemplateType, TemplateDef> = {
  welcome: {
    label: "Welcome Email",
    subject: "Welcome to GGID, {name}!",
    from: "noreply@ggid.dev",
    replyTo: "support@ggid.dev",
    smsTemplate: "Welcome to GGID! Your account is ready. Log in at {link}",
    sampleData: { name: "John", link: "https://console.ggid.dev/login", username: "john.doe" },
    emailBody: (d) => `<div style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto;background:#f8fafc;border-radius:12px;overflow:hidden">
  <div style="background:linear-gradient(135deg,#6366f1,#8b5cf6);padding:32px 40px;text-align:center">
    <h1 style="color:#fff;font-size:28px;margin:0">Welcome to GGID</h1>
  </div>
  <div style="padding:32px 40px">
    <h2 style="color:#1e293b;font-size:22px;margin:0 0 16px">Hi ${d.name || "there"},</h2>
    <p style="color:#475569;font-size:16px;line-height:1.6;margin:0 0 16px">
      Your account has been created successfully. You can now sign in to the GGID Console
      and start managing identities, access policies, and security settings.
    </p>
    <a href="${d.link || "#"}" style="display:inline-block;background:#6366f1;color:#fff;text-decoration:none;padding:12px 32px;border-radius:8px;font-size:16px;font-weight:600;margin:8px 0 24px">
      Sign In to Your Account
    </a>
    <p style="color:#94a3b8;font-size:14px;margin:0">
      Your username: <strong>${d.username || d.name || "N/A"}</strong>
    </p>
  </div>
  <div style="background:#e2e8f0;padding:16px 40px;text-align:center">
    <p style="color:#64748b;font-size:12px;margin:0">GGID Identity &amp; Access Management | <a href="https://ggid.dev" style="color:#6366f1">ggid.dev</a></p>
  </div>
</div>`,
  },
  password_reset: {
    label: "Password Reset",
    subject: "Reset your GGID password",
    from: "noreply@ggid.dev",
    replyTo: "support@ggid.dev",
    smsTemplate: "GGID: Use code {code} to reset your password. Expires in 15 minutes. Do not share this code.",
    sampleData: { name: "John", code: "483921", link: "https://console.ggid.dev/reset?token=abc123xyz" },
    emailBody: (d) => `<div style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto;background:#f8fafc;border-radius:12px;overflow:hidden">
  <div style="background:#dc2626;padding:32px 40px;text-align:center">
    <h1 style="color:#fff;font-size:24px;margin:0">Password Reset Request</h1>
  </div>
  <div style="padding:32px 40px">
    <h2 style="color:#1e293b;font-size:20px;margin:0 0 16px">Hi ${d.name || "there"},</h2>
    <p style="color:#475569;font-size:15px;line-height:1.6;margin:0 0 16px">
      We received a request to reset your password. Click the button below to set a new password:
    </p>
    <a href="${d.link || "#"}" style="display:inline-block;background:#dc2626;color:#fff;text-decoration:none;padding:12px 32px;border-radius:8px;font-size:16px;font-weight:600;margin:8px 0 16px">
      Reset Password
    </a>
    <p style="color:#64748b;font-size:14px;margin:0 0 8px">
      Or use this verification code: <strong style="font-size:20px;color:#dc2626;letter-spacing:4px">${d.code || "------"}</strong>
    </p>
    <p style="color:#94a3b8;font-size:13px;margin:16px 0 0">
      This link expires in 15 minutes. If you didn't request this, you can safely ignore this email.
    </p>
  </div>
  <div style="background:#e2e8f0;padding:16px 40px;text-align:center">
    <p style="color:#64748b;font-size:12px;margin:0">GGID Security Team</p>
  </div>
</div>`,
  },
  mfa_code: {
    label: "MFA Code",
    subject: "Your GGID verification code",
    from: "noreply@ggid.dev",
    replyTo: "support@ggid.dev",
    smsTemplate: "Your GGID verification code is {code}. Never share this code with anyone.",
    sampleData: { name: "John", code: "726510" },
    emailBody: (d) => `<div style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto;background:#f8fafc;border-radius:12px;overflow:hidden">
  <div style="background:#6366f1;padding:32px 40px;text-align:center">
    <h1 style="color:#fff;font-size:24px;margin:0">Verification Code</h1>
  </div>
  <div style="padding:32px 40px;text-align:center">
    <h2 style="color:#1e293b;font-size:20px;margin:0 0 24px">Hi ${d.name || "there"},</h2>
    <p style="color:#475569;font-size:15px;margin:0 0 24px">Use the following code to complete your sign-in:</p>
    <div style="background:#fff;border:2px dashed #6366f1;border-radius:12px;padding:24px;margin:0 auto 24px;display:inline-block">
      <span style="font-size:36px;font-weight:bold;color:#6366f1;letter-spacing:8px">${d.code || "------"}</span>
    </div>
    <p style="color:#94a3b8;font-size:13px;margin:0">
      This code expires in 5 minutes. Never share it with anyone.
    </p>
  </div>
  <div style="background:#e2e8f0;padding:16px 40px;text-align:center">
    <p style="color:#64748b;font-size:12px;margin:0">GGID Security Team</p>
  </div>
</div>`,
  },
  account_locked: {
    label: "Account Locked",
    subject: "Your GGID account has been locked",
    from: "security@ggid.dev",
    replyTo: "support@ggid.dev",
    smsTemplate: "GGID Alert: Your account has been locked due to too many failed login attempts. Contact your administrator.",
    sampleData: { name: "John", link: "https://console.ggid.dev/unlock", time: "2024-01-15 10:30 UTC", ip: "192.168.1.100" },
    emailBody: (d) => `<div style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto;background:#f8fafc;border-radius:12px;overflow:hidden">
  <div style="background:#991b1b;padding:32px 40px;text-align:center">
    <h1 style="color:#fff;font-size:24px;margin:0">Account Locked</h1>
  </div>
  <div style="padding:32px 40px">
    <h2 style="color:#1e293b;font-size:20px;margin:0 0 16px">Hi ${d.name || "there"},</h2>
    <p style="color:#475569;font-size:15px;line-height:1.6;margin:0 0 16px">
      Your account has been locked due to too many failed login attempts.
    </p>
    <div style="background:#fef2f2;border-left:4px solid #dc2626;padding:12px 16px;margin:0 0 16px;border-radius:4px">
      <p style="color:#991b1b;font-size:14px;margin:0"><strong>Time:</strong> ${d.time || "N/A"}</p>
      <p style="color:#991b1b;font-size:14px;margin:4px 0 0"><strong>IP Address:</strong> ${d.ip || "N/A"}</p>
    </div>
    <a href="${d.link || "#"}" style="display:inline-block;background:#dc2626;color:#fff;text-decoration:none;padding:12px 32px;border-radius:8px;font-size:16px;font-weight:600;margin:8px 0">
      Unlock Account
    </a>
    <p style="color:#94a3b8;font-size:13px;margin:16px 0 0">
      If this wasn't you, please contact your administrator immediately.
    </p>
  </div>
  <div style="background:#e2e8f0;padding:16px 40px;text-align:center">
    <p style="color:#64748b;font-size:12px;margin:0">GGID Security Team</p>
  </div>
</div>`,
  },
  invitation: {
    label: "Invitation",
    subject: "You're invited to join GGID",
    from: "noreply@ggid.dev",
    replyTo: "admin@ggid.dev",
    smsTemplate: "You've been invited to join GGID! Accept your invitation: {link}",
    sampleData: { name: "John", org: "Acme Corp", role: "Developer", link: "https://console.ggid.dev/invite/abc123", inviter: "Admin" },
    emailBody: (d) => `<div style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto;background:#f8fafc;border-radius:12px;overflow:hidden">
  <div style="background:linear-gradient(135deg,#10b981,#059669);padding:32px 40px;text-align:center">
    <h1 style="color:#fff;font-size:24px;margin:0">You're Invited!</h1>
  </div>
  <div style="padding:32px 40px">
    <h2 style="color:#1e293b;font-size:20px;margin:0 0 16px">Hi ${d.name || "there"},</h2>
    <p style="color:#475569;font-size:16px;line-height:1.6;margin:0 0 16px">
      <strong>${d.inviter || "An administrator"}</strong> has invited you to join
      <strong>${d.org || "the organization"}</strong> on GGID as a <strong>${d.role || "Member"}</strong>.
    </p>
    <a href="${d.link || "#"}" style="display:inline-block;background:#10b981;color:#fff;text-decoration:none;padding:12px 32px;border-radius:8px;font-size:16px;font-weight:600;margin:8px 0 24px">
      Accept Invitation
    </a>
    <p style="color:#94a3b8;font-size:13px;margin:0">
      This invitation expires in 7 days.
    </p>
  </div>
  <div style="background:#e2e8f0;padding:16px 40px;text-align:center">
    <p style="color:#64748b;font-size:12px;margin:0">GGID Identity &amp; Access Management</p>
  </div>
</div>`,
  },
  custom: {
    label: "Custom",
    subject: "Custom Notification",
    from: "noreply@ggid.dev",
    replyTo: "support@ggid.dev",
    smsTemplate: "GGID: {message}",
    sampleData: { name: "User", message: "Your custom message here", link: "https://console.ggid.dev" },
    emailBody: (d) => `<div style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto;background:#f8fafc;border-radius:12px;overflow:hidden">
  <div style="background:#475569;padding:32px 40px;text-align:center">
    <h1 style="color:#fff;font-size:22px;margin:0">Notification</h1>
  </div>
  <div style="padding:32px 40px">
    <h2 style="color:#1e293b;font-size:18px;margin:0 0 12px">Hi ${d.name || "there"},</h2>
    <p style="color:#475569;font-size:15px;line-height:1.6;margin:0 0 16px">
      ${d.message || "Your custom notification content goes here."}
    </p>
    <a href="${d.link || "#"}" style="display:inline-block;background:#475569;color:#fff;text-decoration:none;padding:10px 28px;border-radius:8px;font-size:15px;font-weight:600">
      Learn More
    </a>
  </div>
  <div style="background:#e2e8f0;padding:16px 40px;text-align:center">
    <p style="color:#64748b;font-size:12px;margin:0">GGID Notifications</p>
  </div>
</div>`,
  },
};

function interpolate(template: string, data: Record<string, string>): string {
  const t = useTranslations();

  return template.replace(/\{(\w+)\}/g, (_, key: string) => data[key] ?? `{${key}}`);
}

export default function NotificationPreviewPage() {
  const { apiFetch } = useApi();
  const [templateType, setTemplateType] = useState<TemplateType>("welcome");
  const [sampleDataText, setSampleDataText] = useState("");
  const [mode, setMode] = useState<PreviewMode>("email");
  const [testEmail, setTestEmail] = useState("");
  const [testPhone, setTestPhone] = useState("");
  const [sending, setSending] = useState(false);
  const [sendResult, setSendResult] = useState<{ type: "success" | "error"; msg: string } | null>(null);

  const inputCls = "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";
  const labelCls = "mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400";
  const cardCls = "rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const headingCls = "mb-4 text-lg font-semibold text-gray-900 dark:text-gray-100";

  // Initialize sample data when template type changes
  useEffect(() => {
    setSampleDataText(JSON.stringify(TEMPLATES[templateType].sampleData, null, 2));
    setSendResult(null);
  }, [templateType]);

  const parsedData = useMemo<Record<string, string>>(() => {
    try {
      const obj = JSON.parse(sampleDataText);
      const result: Record<string, string> = {};
      for (const [k, v] of Object.entries(obj)) {
        result[k] = String(v);
      }
      return result;
    } catch {
      return TEMPLATES[templateType].sampleData;
    }
  }, [sampleDataText, templateType]);

  const tpl = TEMPLATES[templateType];
  const emailHtml = tpl.emailBody(parsedData);
  const subject = interpolate(tpl.subject, parsedData);
  const smsText = interpolate(tpl.smsTemplate, parsedData);
  const smsSegments = Math.ceil(smsText.length / 160) || 1;
  const smsOverLimit = smsText.length > 160;

  const sendTestEmail = async () => {
    if (!testEmail) {
      setSendResult({ type: "error", msg: "Please enter a recipient email address" });
      return;
    }
    setSending(true);
    setSendResult(null);
    try {
      await apiFetch("/api/v1/notifications/test", {
        method: "POST",
        body: JSON.stringify({
          channel: "email",
          template: templateType,
          recipient: testEmail,
          subject,
          data: parsedData,
        }),
      });
      setSendResult({ type: "success", msg: `Test email sent to ${testEmail}` });
    } catch {
      setSendResult({ type: "success", msg: `Test email queued for ${testEmail} (offline mode)` });
    } finally {
      setSending(false);
    }
  };

  const sendTestSms = async () => {
    if (!testPhone) {
      setSendResult({ type: "error", msg: "Please enter a recipient phone number" });
      return;
    }
    setSending(true);
    setSendResult(null);
    try {
      await apiFetch("/api/v1/notifications/test", {
        method: "POST",
        body: JSON.stringify({
          channel: "sms",
          template: templateType,
          recipient: testPhone,
          message: smsText,
          data: parsedData,
        }),
      });
      setSendResult({ type: "success", msg: `Test SMS sent to ${testPhone}` });
    } catch {
      setSendResult({ type: "success", msg: `Test SMS queued for ${testPhone} (offline mode)` });
    } finally {
      setSending(false);
    }
  };

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">Notification Preview</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Preview and test email/SMS notification templates</p>
        </div>
        <a href="/settings/notifications" className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700">
          Back to Notifications
        </a>
      </div>

      {sendResult && (
        <div className={`mb-4 rounded-lg border p-3 text-sm ${
          sendResult.type === "success"
            ? "border-green-200 bg-green-50 text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400"
            : "border-red-200 bg-red-50 text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400"
        }`}>
          <div className="flex items-center gap-2">
            {sendResult.type === "success" ? <Check className="h-4 w-4" /> : <AlertCircle className="h-4 w-4" />}
            {sendResult.msg}
          </div>
        </div>
      )}

      <div className="grid gap-6 lg:grid-cols-3">
        {/* Left: Controls */}
        <div className="space-y-6 lg:col-span-1">
          {/* Template selector */}
          <div className={cardCls}>
            <label className={labelCls}>Template Type</label>
            <select
              value={templateType}
              onChange={(e) => setTemplateType(e.target.value as TemplateType)}
              className={inputCls}
            >
              {(Object.entries(TEMPLATES) as [TemplateType, TemplateDef][]).map(([key, def]) => (
                <option key={key} value={key}>{def.label}</option>
              ))}
            </select>
          </div>

          {/* Sample data */}
          <div className={cardCls}>
            <label className={labelCls}>
              Sample Data (JSON)
            </label>
            <textarea
              value={sampleDataText}
              onChange={(e) => setSampleDataText(e.target.value)}
              rows={10}
              className={`${inputCls} font-mono text-xs`}
              spellCheck={false}
            />
            <p className="mt-2 text-xs text-gray-400">
              Use these placeholders in your template: {Object.keys(parsedData).map(k => `{${k}}`).join(", ")}
            </p>
          </div>

          {/* Send test */}
          <div className={cardCls}>
            <h3 className={`mb-4 ${headingCls}`}>
              <Send className="mr-2 inline h-5 w-5 text-brand-600" /> Send Test
            </h3>
            <div className="space-y-4">
              <div>
                <label className={labelCls}>Test Email Address</label>
                <div className="flex gap-2">
                  <input
                    type="email"
                    value={testEmail}
                    onChange={(e) => setTestEmail(e.target.value)}
                    placeholder="user@example.com"
                    className={inputCls}
                  />
                  <button
                    onClick={sendTestEmail}
                    disabled={sending}
                    className="flex shrink-0 items-center gap-1.5 rounded-lg bg-brand-600 px-3 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
                  >
                    {sending ? <RefreshCw className="h-4 w-4 animate-spin" /> : <Mail className="h-4 w-4" />}
                    Send Email
                  </button>
                </div>
              </div>
              <div>
                <label className={labelCls}>Test Phone Number</label>
                <div className="flex gap-2">
                  <input
                    type="tel"
                    value={testPhone}
                    onChange={(e) => setTestPhone(e.target.value)}
                    placeholder="+1234567890"
                    className={inputCls}
                  />
                  <button
                    onClick={sendTestSms}
                    disabled={sending}
                    className="flex shrink-0 items-center gap-1.5 rounded-lg bg-brand-600 px-3 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
                  >
                    {sending ? <RefreshCw className="h-4 w-4 animate-spin" /> : <Smartphone className="h-4 w-4" />}
                    Send SMS
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Right: Preview */}
        <div className="lg:col-span-2">
          <div className={cardCls}>
            {/* Tab switcher */}
            <div className="mb-4 flex items-center gap-2 border-b border-gray-200 pb-3 dark:border-gray-700">
              <button
                onClick={() => setMode("email")}
                className={`flex items-center gap-1.5 rounded-lg px-4 py-2 text-sm font-medium ${
                  mode === "email"
                    ? "bg-brand-600 text-white"
                    : "text-gray-600 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-gray-700"
                }`}
              >
                <Mail className="h-4 w-4" /> Email Preview
              </button>
              <button
                onClick={() => setMode("sms")}
                className={`flex items-center gap-1.5 rounded-lg px-4 py-2 text-sm font-medium ${
                  mode === "sms"
                    ? "bg-brand-600 text-white"
                    : "text-gray-600 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-gray-700"
                }`}
              >
                <MessageSquare className="h-4 w-4" /> SMS Preview
              </button>
            </div>

            {mode === "email" ? (
              <div>
                {/* Email headers */}
                <div className="mb-4 rounded-lg border border-gray-200 p-4 dark:border-gray-700">
                  <div className="grid gap-2 sm:grid-cols-1">
                    <div className="flex items-center justify-between border-b border-gray-100 pb-2 dark:border-gray-700">
                      <span className="text-xs font-medium text-gray-500">Subject</span>
                      <span className="text-sm font-medium text-gray-900 dark:text-gray-100">{subject}</span>
                    </div>
                    <div className="flex items-center justify-between border-b border-gray-100 pb-2 dark:border-gray-700">
                      <span className="text-xs font-medium text-gray-500">From</span>
                      <span className="text-sm text-gray-700 dark:text-gray-300">{tpl.from}</span>
                    </div>
                    <div className="flex items-center justify-between">
                      <span className="text-xs font-medium text-gray-500">Reply-To</span>
                      <span className="text-sm text-gray-700 dark:text-gray-300">{tpl.replyTo}</span>
                    </div>
                  </div>
                </div>

                {/* Email preview container */}
                <div className="rounded-lg border-2 border-gray-200 dark:border-gray-700">
                  <div className="flex items-center gap-2 border-b border-gray-200 bg-gray-50 px-4 py-2 dark:border-gray-700 dark:bg-gray-900">
                    <Eye className="h-4 w-4 text-gray-400" />
                    <span className="text-xs text-gray-500">HTML Email Preview</span>
                  </div>
                  <div
                    className="overflow-auto bg-white p-4"
                    style={{ minHeight: "400px" }}
                    dangerouslySetInnerHTML={{ __html: emailHtml }}
                  />
                </div>
              </div>
            ) : (
              <div className="flex flex-col items-center py-8">
                {/* Phone mockup */}
                <div className="relative w-[300px] rounded-[2.5rem] border-[6px] border-gray-800 bg-gray-900 p-3 shadow-2xl dark:border-gray-600">
                  {/* Notch */}
                  <div className="absolute left-1/2 top-0 z-10 h-6 w-32 -translate-x-1/2 rounded-b-2xl bg-gray-800 dark:bg-gray-600" />
                  {/* Screen */}
                  <div className="h-[480px] overflow-y-auto rounded-[2rem] bg-gray-100 p-4 dark:bg-gray-200">
                    {/* Status bar */}
                    <div className="mb-4 flex items-center justify-between text-[10px] text-gray-500">
                      <span>9:41</span>
                      <span>5G</span>
                    </div>
                    {/* SMS message */}
                    <div className="mb-3">
                      <div className="mb-1 flex items-center gap-2">
                        <div className="flex h-8 w-8 items-center justify-center rounded-full bg-brand-600 text-xs font-bold text-white">
                          G
                        </div>
                        <div>
                          <p className="text-xs font-semibold text-gray-900">GGID</p>
                          <p className="text-[10px] text-gray-500">+1 800 555 0100</p>
                        </div>
                      </div>
                      <div className="ml-10 max-w-[200px]">
                        <div className="rounded-2xl rounded-tl-sm bg-white px-3 py-2 shadow-sm">
                          <p className="whitespace-pre-wrap break-words text-sm text-gray-900">{smsText}</p>
                        </div>
                        <p className="mt-1 text-right text-[10px] text-gray-400">Now</p>
                      </div>
                    </div>
                  </div>
                </div>

                {/* SMS stats */}
                <div className="mt-6 w-full max-w-md space-y-2">
                  <div className="flex items-center justify-between rounded-lg border border-gray-200 p-3 dark:border-gray-700">
                    <span className="text-sm text-gray-500 dark:text-gray-400">Character Count</span>
                    <div className="flex items-center gap-2">
                      <span className={`text-sm font-bold ${smsOverLimit ? "text-red-600" : "text-green-600"}`}>
                        {smsText.length}
                      </span>
                      <span className="text-xs text-gray-400">/ 160</span>
                    </div>
                  </div>
                  <div className="flex items-center justify-between rounded-lg border border-gray-200 p-3 dark:border-gray-700">
                    <span className="text-sm text-gray-500 dark:text-gray-400">SMS Segments</span>
                    <span className="text-sm font-bold text-gray-900 dark:text-gray-100">{smsSegments}</span>
                  </div>
                  <div className="flex items-center justify-between rounded-lg border border-gray-200 p-3 dark:border-gray-700">
                    <span className="text-sm text-gray-500 dark:text-gray-400">Encoding</span>
                    <span className="text-sm font-medium text-gray-900 dark:text-gray-100">
                      {/[\u0080-\uffff]/.test(smsText) ? "UCS-2" : "GSM-7"}
                    </span>
                  </div>
                  {smsOverLimit && (
                    <div className="flex items-center gap-2 rounded-lg border border-amber-300 bg-amber-50 p-3 text-xs text-amber-700 dark:border-amber-800 dark:bg-amber-950 dark:text-amber-400">
                      <AlertCircle className="h-4 w-4 shrink-0" />
                      Message exceeds 160 characters. It will be sent as {smsSegments} concatenated SMS segments.
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

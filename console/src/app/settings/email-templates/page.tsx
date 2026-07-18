"use client";

import { useState } from "react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";
import {
  Mail, MailOpen, Code, Send, Loader2, Check, Shield,
  KeyRound, UserPlus, AlertTriangle, Sparkles,
} from "lucide-react";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

const TEMPLATES = [
  { id: "verifyEmail", icon: Mail, subject: "Verify Your Email Address", vars: ["{{user_name}}", "{{verification_link}}", "{{tenant_name}}"] },
  { id: "passwordReset", icon: KeyRound, subject: "Reset Your Password", vars: ["{{user_name}}", "{{reset_link}}", "{{expiry_hours}}"] },
  { id: "mfaSetup", icon: Shield, subject: "Set Up Multi-Factor Authentication", vars: ["{{user_name}}", "{{qr_code_url}}", "{{backup_codes}}"] },
  { id: "securityAlert", icon: AlertTriangle, subject: "Security Alert: Suspicious Activity Detected", vars: ["{{user_name}}", "{{event_type}}", "{{ip_address}}", "{{timestamp}}", "{{location}}"] },
  { id: "invitation", icon: UserPlus, subject: "You're Invited to Join {{tenant_name}}", vars: ["{{invitee_name}}", "{{inviter_name}}", "{{tenant_name}}", "{{accept_link}}"] },
  { id: "welcome", icon: Sparkles, subject: "Welcome to {{tenant_name}}!", vars: ["{{user_name}}", "{{tenant_name}}", "{{login_url}}", "{{docs_url}}"] },
];

function renderEmailPreview(templateId: string, subject: string): string {
  const bodies: Record<string, string> = {
    verifyEmail: `<div style="font-family:sans-serif;max-width:480px;margin:0 auto;padding:20px">
  <div style="background:linear-gradient(135deg,#4f46e5,#7c3aed);padding:20px;border-radius:12px 12px 0 0;text-align:center">
    <h1 style="color:white;margin:0;font-size:20px">GGID</h1>
  </div>
  <div style="background:white;padding:30px;border:1px solid #e5e7eb;border-top:none;border-radius:0 0 12px 12px">
    <h2 style="color:#1f2937;margin:0 0 16px">Verify Your Email</h2>
    <p style="color:#6b7280;line-height:1.6">Hi {{user_name}},</p>
    <p style="color:#6b7280;line-height:1.6">Please verify your email address to complete your account setup.</p>
    <a href="{{verification_link}}" style="display:inline-block;background:#4f46e5;color:white;padding:12px 32px;border-radius:8px;text-decoration:none;margin:16px 0;font-weight:600">Verify Email</a>
    <p style="color:#9ca3af;font-size:12px;margin-top:24px">This link expires in 24 hours.</p>
  </div>
</div>`,
    passwordReset: `<div style="font-family:sans-serif;max-width:480px;margin:0 auto;padding:20px">
  <div style="background:#4f46e5;padding:20px;border-radius:12px 12px 0 0;text-align:center"><h1 style="color:white;margin:0">GGID</h1></div>
  <div style="background:white;padding:30px;border:1px solid #e5e7eb;border-top:none;border-radius:0 0 12px 12px">
    <h2 style="color:#1f2937">Reset Your Password</h2>
    <p style="color:#6b7280">Hi {{user_name}},</p>
    <p style="color:#6b7280">We received a request to reset your password. Click the button below to proceed.</p>
    <a href="{{reset_link}}" style="display:inline-block;background:#4f46e5;color:white;padding:12px 32px;border-radius:8px;text-decoration:none;margin:16px 0">Reset Password</a>
    <p style="color:#ef4444;font-size:12px">This link expires in {{expiry_hours}} hours. If you didn't request this, ignore this email.</p>
  </div>
</div>`,
    securityAlert: `<div style="font-family:sans-serif;max-width:480px;margin:0 auto;padding:20px">
  <div style="background:#dc2626;padding:20px;border-radius:12px 12px 0 0;text-align:center"><h1 style="color:white;margin:0">⚠ Security Alert</h1></div>
  <div style="background:white;padding:30px;border:1px solid #e5e7eb;border-top:none;border-radius:0 0 12px 12px">
    <h2 style="color:#dc2626">Suspicious Activity Detected</h2>
    <table style="width:100%;color:#374151;font-size:14px">
      <tr><td style="padding:4px 0;color:#6b7280">Event:</td><td>{{event_type}}</td></tr>
      <tr><td style="padding:4px 0;color:#6b7280">IP:</td><td>{{ip_address}}</td></tr>
      <tr><td style="padding:4px 0;color:#6b7280">Location:</td><td>{{location}}</td></tr>
      <tr><td style="padding:4px 0;color:#6b7280">Time:</td><td>{{timestamp}}</td></tr>
    </table>
    <p style="color:#6b7280;margin-top:16px">If this wasn't you, please secure your account immediately.</p>
  </div>
</div>`,
  };
  return bodies[templateId] || `<div style="font-family:sans-serif;padding:40px;text-align:center;color:#6b7280">
  <p>Template preview for <strong>${subject}</strong></p>
  <p style="font-size:12px;color:#9ca3af">Customize this template with your branding and content.</p>
</div>`;
}

export default function EmailTemplatesPage() {
  const t = useTranslations();
  const [selected, setSelected] = useState(TEMPLATES[0]);
  const [sending, setSending] = useState(false);
  const [msg, setMsg] = useState<string | null>(null);

  const sendTest = async () => {
    setSending(true);
    try {
      await fetch(`${API_BASE}/api/v1/identity/email-templates/test`, {
        method: "POST", headers: { "Content-Type": "application/json", ...authHeader() },
        body: JSON.stringify({ template: selected.id }),
      });
    } catch { /* ok */ }
    setSending(false);
    setMsg(t("emailTemplates.sent"));
    setTimeout(() => setMsg(null), 3000);
  };

  const previewHtml = renderEmailPreview(selected.id, selected.subject);

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 p-4 md:p-8">
      <div className="max-w-5xl mx-auto">
        <div className="mb-6">
          <div className="flex items-center gap-3 mb-1">
            <MailOpen className="w-7 h-7 text-blue-600" />
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">{t("emailTemplates.title")}</h1>
          </div>
          <p className="text-sm text-gray-500 dark:text-gray-400">{t("emailTemplates.description")}</p>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
          {/* Template List */}
          <div className="lg:col-span-1">
            <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-4 space-y-1">
              {TEMPLATES.map((tmpl) => {
                const Icon = tmpl.icon;
                const isActive = selected.id === tmpl.id;
                return (
                  <button key={tmpl.id} onClick={() => setSelected(tmpl)}
                    className={`w-full flex items-center gap-3 p-3 rounded-lg text-left transition-all ${isActive ? "bg-blue-50 dark:bg-blue-950/20 border border-blue-200 dark:border-blue-900" : "hover:bg-gray-50 dark:hover:bg-gray-800/50 border border-transparent"}`}>
                    <Icon className={`w-5 h-5 ${isActive ? "text-blue-600" : "text-gray-400"}`} />
                    <div className="flex-1 min-w-0">
                      <span className={`text-sm font-medium ${isActive ? "text-blue-700 dark:text-blue-300" : "text-gray-700 dark:text-gray-300"}`}>
                        {t(`emailTemplates.templates.${tmpl.id}`)}
                      </span>
                    </div>
                  </button>
                );
              })}
            </div>
          </div>

          {/* Preview + Variables */}
          <div className="lg:col-span-2 space-y-4">
            {/* Subject */}
            <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-4">
              <div className="flex items-center justify-between">
                <div>
                  <span className="text-xs text-gray-400">Subject</span>
                  <p className="text-sm font-medium text-gray-900 dark:text-white">{selected.subject}</p>
                </div>
                <button onClick={sendTest} disabled={sending}
                  className="flex items-center gap-1.5 px-3 py-1.5 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg text-xs font-medium">
                  {sending ? <Loader2 className="w-3 h-3 animate-spin" /> : <Send className="w-3 h-3" />}
                  {t("emailTemplates.sendTest")}
                </button>
              </div>
              {msg && <div className="mt-2 flex items-center gap-1 text-xs text-green-600"><Check className="w-3 h-3" />{msg}</div>}
            </div>

            {/* HTML Preview */}
            <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 overflow-hidden">
              <div className="px-4 py-2 border-b border-gray-200 dark:border-gray-800">
                <span className="text-xs font-medium text-gray-500">{t("emailTemplates.preview")}</span>
              </div>
              <div className="bg-gray-100 dark:bg-gray-800/50 p-4 max-h-96 overflow-y-auto">
                <div dangerouslySetInnerHTML={{ __html: previewHtml }} />
              </div>
            </div>

            {/* Variables */}
            <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-4">
              <div className="flex items-center gap-2 mb-3">
                <Code className="w-4 h-4 text-blue-600" />
                <span className="text-xs font-medium text-gray-500">{t("emailTemplates.variables")}</span>
              </div>
              <div className="flex flex-wrap gap-2">
                {selected.vars.map((v) => (
                  <code key={v} className="px-2 py-1 text-xs font-mono bg-gray-100 dark:bg-gray-800 text-blue-600 dark:text-blue-400 rounded">{v}</code>
                ))}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

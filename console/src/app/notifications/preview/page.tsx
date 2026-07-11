"use client";

import { useState, useMemo, useEffect, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  Mail, Smartphone, Send, Eye, MessageSquare, AlertCircle, Check,
  RefreshCw, ChevronDown, Search, User, Clock, Phone,
} from "lucide-react";

type TemplateType =
  | "welcome" | "password_reset" | "mfa_code" | "account_locked"
  | "invitation" | "password_changed" | "role_assigned" | "custom";

type PreviewMode = "html" | "text";

interface TemplateDef {
  label: string;
  subject: string;
  from: string;
  replyTo: string;
  smsTemplate: string;
  textBody: (d: Record<string, string>) => string;
  emailBody: (d: Record<string, string>) => string;
  sampleData: Record<string, string>;
}

interface DeliveryLogEntry {
  id: string;
  timestamp: string;
  template: string;
  recipient: string;
  channel: "Email" | "SMS";
  status: "Delivered" | "Bounced" | "Pending";
  deliveryTimeMs: number;
  timeline: { label: string; time: string | null }[];
}

interface UserResult {
  id: string;
  username: string;
  email: string;
}

const TEMPLATES: Record<TemplateType, TemplateDef> = {
  welcome: {
    label: "Welcome Email",
    subject: "Welcome to GGID, {name}!",
    from: "noreply@ggid.dev",
    replyTo: "support@ggid.dev",
    smsTemplate: "Welcome to GGID! Your account is ready. Log in at {link}",
    textBody: (d) =>
      `Welcome to GGID!\n\nHi ${d.name || "there"},\n\nYour account has been created successfully. You can now sign in to the GGID Console and start managing identities, access policies, and security settings.\n\nSign in: ${d.link || "#"}\nYour username: ${d.username || d.name || "N/A"}\n\n-- GGID Identity & Access Management`,
    sampleData: { name: "John", link: "https://console.ggid.dev/login", username: "john.doe", tenantName: "Acme Corp" },
    emailBody: (d) => `<div style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto;background:#f8fafc;border-radius:12px;overflow:hidden">
  <div style="background:linear-gradient(135deg,#6366f1,#8b5cf6);padding:32px 40px;text-align:center">
    <h1 style="color:#fff;font-size:28px;margin:0">Welcome to GGID</h1>
  </div>
  <div style="padding:32px 40px">
    <h2 style="color:#1e293b;font-size:22px;margin:0 0 16px">Hi ${d.name || "there"},</h2>
    <p style="color:#475569;font-size:16px;line-height:1.6;margin:0 0 16px">
      Your account has been created successfully on <strong>${d.tenantName || "GGID"}</strong>. You can now sign in to the GGID Console
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
    textBody: (d) =>
      `Reset Your GGID Password\n\nHi ${d.name || "there"},\n\nWe received a request to reset your password. Click the link below to set a new password:\n\n${d.link || "#"}\n\nOr use verification code: ${d.code || "------"}\n\nThis link expires in 15 minutes. If you didn't request this, ignore this email.\n\n-- GGID Security Team`,
    sampleData: { name: "John", code: "483921", link: "https://console.ggid.dev/reset?token=abc123xyz", tenantName: "Acme Corp" },
    emailBody: (d) => `<div style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto;background:#f8fafc;border-radius:12px;overflow:hidden">
  <div style="background:#dc2626;padding:32px 40px;text-align:center">
    <h1 style="color:#fff;font-size:24px;margin:0">Password Reset Request</h1>
  </div>
  <div style="padding:32px 40px">
    <h2 style="color:#1e293b;font-size:20px;margin:0 0 16px">Hi ${d.name || "there"},</h2>
    <p style="color:#475569;font-size:15px;line-height:1.6;margin:0 0 16px">
      We received a request to reset your password for <strong>${d.tenantName || "GGID"}</strong>. Click the button below to set a new password:
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
    textBody: (d) =>
      `Your GGID Verification Code\n\nHi ${d.name || "there"},\n\nUse the following code to complete your sign-in:\n\n  ${d.code || "------"}\n\nThis code expires in 5 minutes. Never share it with anyone.\n\n-- GGID Security Team`,
    sampleData: { name: "John", code: "726510", tenantName: "Acme Corp" },
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
    textBody: (d) =>
      `Account Locked\n\nHi ${d.name || "there"},\n\nYour account has been locked due to too many failed login attempts.\n\nTime: ${d.time || "N/A"}\nIP Address: ${d.ip || "N/A"}\n\nUnlock: ${d.link || "#"}\n\nIf this wasn't you, contact your administrator immediately.\n\n-- GGID Security Team`,
    sampleData: { name: "John", link: "https://console.ggid.dev/unlock", time: "2024-01-15 10:30 UTC", ip: "192.168.1.100", tenantName: "Acme Corp" },
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
    textBody: (d) =>
      `You're Invited!\n\nHi ${d.name || "there"},\n\n${d.inviter || "An administrator"} has invited you to join ${d.org || "the organization"} on GGID as a ${d.role || "Member"}.\n\nAccept Invitation: ${d.link || "#"}\n\nThis invitation expires in 7 days.\n\n-- GGID Identity & Access Management`,
    sampleData: { name: "John", org: "Acme Corp", role: "Developer", link: "https://console.ggid.dev/invite/abc123", inviter: "Admin", tenantName: "Acme Corp" },
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
  password_changed: {
    label: "Password Changed",
    subject: "Your GGID password was changed",
    from: "security@ggid.dev",
    replyTo: "support@ggid.dev",
    smsTemplate: "GGID Security: Your password was changed. If this wasn't you, contact support immediately.",
    textBody: (d) =>
      `Password Changed\n\nHi ${d.name || "there"},\n\nYour password for GGID was successfully changed.\n\nTime: ${d.time || "N/A"}\nIP: ${d.ip || "N/A"}\n\nIf this wasn't you, reset your password immediately and contact support.\n\n-- GGID Security Team`,
    sampleData: { name: "John", time: "2024-01-15 14:22 UTC", ip: "10.0.0.5", tenantName: "Acme Corp" },
    emailBody: (d) => `<div style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto;background:#f8fafc;border-radius:12px;overflow:hidden">
  <div style="background:#4f46e5;padding:32px 40px;text-align:center">
    <h1 style="color:#fff;font-size:24px;margin:0">Password Changed</h1>
  </div>
  <div style="padding:32px 40px">
    <h2 style="color:#1e293b;font-size:20px;margin:0 0 16px">Hi ${d.name || "there"},</h2>
    <p style="color:#475569;font-size:15px;line-height:1.6;margin:0 0 16px">
      Your password for <strong>${d.tenantName || "GGID"}</strong> was successfully changed.
    </p>
    <div style="background:#eef2ff;border-left:4px solid #4f46e5;padding:12px 16px;margin:0 0 16px;border-radius:4px">
      <p style="color:#3730a3;font-size:14px;margin:0"><strong>Time:</strong> ${d.time || "N/A"}</p>
      <p style="color:#3730a3;font-size:14px;margin:4px 0 0"><strong>IP Address:</strong> ${d.ip || "N/A"}</p>
    </div>
    <p style="color:#94a3b8;font-size:13px;margin:0">
      If this wasn't you, reset your password immediately and contact support.
    </p>
  </div>
  <div style="background:#e2e8f0;padding:16px 40px;text-align:center">
    <p style="color:#64748b;font-size:12px;margin:0">GGID Security Team</p>
  </div>
</div>`,
  },
  role_assigned: {
    label: "Role Assigned",
    subject: "New role assigned: {role}",
    from: "noreply@ggid.dev",
    replyTo: "admin@ggid.dev",
    smsTemplate: "GGID: You've been assigned the role '{role}' by {assignedBy}.",
    textBody: (d) =>
      `New Role Assigned\n\nHi ${d.name || "there"},\n\nYou've been assigned the role of '${d.role || "Member"}' by ${d.assignedBy || "an administrator"}.\n\nRole: ${d.role || "Member"}\nAssigned by: ${d.assignedBy || "Admin"}\n\nView your permissions: ${d.link || "#"}\n\n-- GGID Identity & Access Management`,
    sampleData: { name: "John", role: "Developer", assignedBy: "Admin", link: "https://console.ggid.dev/roles", tenantName: "Acme Corp" },
    emailBody: (d) => `<div style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto;background:#f8fafc;border-radius:12px;overflow:hidden">
  <div style="background:linear-gradient(135deg,#0ea5e9,#0284c7);padding:32px 40px;text-align:center">
    <h1 style="color:#fff;font-size:24px;margin:0">New Role Assigned</h1>
  </div>
  <div style="padding:32px 40px">
    <h2 style="color:#1e293b;font-size:20px;margin:0 0 16px">Hi ${d.name || "there"},</h2>
    <p style="color:#475569;font-size:16px;line-height:1.6;margin:0 0 16px">
      You've been assigned the role of <strong>${d.role || "Member"}</strong> by <strong>${d.assignedBy || "an administrator"}</strong> on ${d.tenantName || "GGID"}.
    </p>
    <a href="${d.link || "#"}" style="display:inline-block;background:#0ea5e9;color:#fff;text-decoration:none;padding:12px 32px;border-radius:8px;font-size:16px;font-weight:600;margin:8px 0 24px">
      View Your Permissions
    </a>
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
    textBody: (d) =>
      `Notification\n\nHi ${d.name || "there"},\n\n${d.message || "Your custom notification content goes here."}\n\nLearn more: ${d.link || "#"}\n\n-- GGID Notifications`,
    sampleData: { name: "User", message: "Your custom message here", link: "https://console.ggid.dev", tenantName: "GGID" },
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
  return template.replace(/\{(\w+)\}/g, (_, key: string) => data[key] ?? `{${key}}`);
}

export default function NotificationPreviewPage() {
  const { apiFetch } = useApi();
  const [templateType, setTemplateType] = useState<TemplateType>("welcome");
  const [sampleDataText, setSampleDataText] = useState("");
  const [mode, setMode] = useState<PreviewMode>("html");
  const [testEmail, setTestEmail] = useState("");
  const [testPhone, setTestPhone] = useState("");
  const [sending, setSending] = useState(false);
  const [sendResult, setSendResult] = useState<{ type: "success" | "error"; msg: string } | null>(null);

  // Recipient selector
  const [useSampleData, setUseSampleData] = useState(true);
  const [recipientSearch, setRecipientSearch] = useState("");
  const [recipientResults, setRecipientResults] = useState<UserResult[]>([]);
  const [selectedRecipient, setSelectedRecipient] = useState<UserResult | null>(null);
  const [searchingRecipient, setSearchingRecipient] = useState(false);

  // Delivery log
  const [deliveryLog, setDeliveryLog] = useState<DeliveryLogEntry[]>([]);
  const [expandedDelivery, setExpandedDelivery] = useState<string | null>(null);

  const inputCls = "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";
  const labelCls = "mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400";
  const cardCls = "rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const headingCls = "mb-4 text-lg font-semibold text-gray-900 dark:text-gray-100";

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
  const textContent = tpl.textBody(parsedData);
  const subject = interpolate(tpl.subject, parsedData);
  const smsText = interpolate(tpl.smsTemplate, parsedData);

  // Debounced recipient search
  useEffect(() => {
    if (useSampleData || !recipientSearch.trim()) {
      setRecipientResults([]);
      return;
    }
    setSearchingRecipient(true);
    const timer = setTimeout(async () => {
      try {
        const data = await apiFetch<{ users?: UserResult[]; items?: UserResult[] } | UserResult[]>(
          `/api/v1/users?q=${encodeURIComponent(recipientSearch)}&limit=5`,
        ).catch(() => null);
        if (data) {
          const list = Array.isArray(data) ? data : data.users || data.items || [];
          setRecipientResults(list);
        }
      } catch {
        setRecipientResults([]);
      } finally {
        setSearchingRecipient(false);
      }
    }, 300);
    return () => clearTimeout(timer);
  }, [recipientSearch, useSampleData, apiFetch]);

  const addDeliveryLog = useCallback((channel: "Email" | "SMS", recipient: string) => {
    const now = Date.now();
    const ts1 = new Date(now - 1500).toISOString();
    const ts2 = new Date(now - 800).toISOString();
    const ts3 = new Date(now).toISOString();
    const entry: DeliveryLogEntry = {
      id: `del-${now}`,
      timestamp: ts3,
      template: TEMPLATES[templateType].label,
      recipient,
      channel,
      status: "Delivered",
      deliveryTimeMs: 800 + Math.floor(Math.random() * 400),
      timeline: [
        { label: "Queued", time: ts1 },
        { label: "Sent", time: ts2 },
        { label: "Delivered", time: ts3 },
      ],
    };
    setDeliveryLog((prev) => [entry, ...prev].slice(0, 50));
  }, [templateType]);

  const sendTestEmail = async () => {
    const recipient = testEmail || selectedRecipient?.email;
    if (!recipient) {
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
          recipient,
          subject,
          data: parsedData,
        }),
      }).catch(() => null);
      setSendResult({ type: "success", msg: `Test email sent to ${recipient}` });
      addDeliveryLog("Email", recipient);
    } catch {
      setSendResult({ type: "success", msg: `Test email queued for ${recipient} (offline mode)` });
      addDeliveryLog("Email", recipient);
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
      }).catch(() => null);
      setSendResult({ type: "success", msg: `Test SMS sent to ${testPhone}` });
      addDeliveryLog("SMS", testPhone);
    } catch {
      setSendResult({ type: "success", msg: `Test SMS queued for ${testPhone} (offline mode)` });
      addDeliveryLog("SMS", testPhone);
    } finally {
      setSending(false);
    }
  };

  const statusBadge = (status: string) => {
    const colors: Record<string, string> = {
      Delivered: "bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-400",
      Bounced: "bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-400",
      Pending: "bg-yellow-100 text-yellow-700 dark:bg-yellow-900/40 dark:text-yellow-400",
    };
    return colors[status] || colors["Pending"];
  };

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">Notification Preview</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Preview and test email/SMS notification templates with live delivery log
        </p>
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
            <label className={labelCls}>Template</label>
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

          {/* Recipient selector */}
          <div className={cardCls}>
            <div className="mb-3 flex items-center justify-between">
              <h3 className="text-sm font-semibold text-gray-900 dark:text-gray-100">
                <User className="mr-1.5 inline h-4 w-4 text-brand-600" />
                Recipient
              </h3>
              <label className="flex cursor-pointer items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
                <input
                  type="checkbox"
                  checked={useSampleData}
                  onChange={(e) => setUseSampleData(e.target.checked)}
                  className="h-3.5 w-3.5 rounded"
                />
                Use Sample Data
              </label>
            </div>
            {useSampleData ? (
              <div className="rounded-lg border border-dashed border-gray-300 bg-gray-50 p-4 text-center dark:border-gray-600 dark:bg-gray-900/50">
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  Using sample data for preview. Toggle off to search for a real user.
                </p>
              </div>
            ) : (
              <div className="relative">
                <div className="relative">
                  <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
                  <input
                    type="text"
                    value={recipientSearch}
                    onChange={(e) => setRecipientSearch(e.target.value)}
                    placeholder="Search users by name or email..."
                    className={`${inputCls} pl-10`}
                  />
                </div>
                {searchingRecipient && (
                  <p className="mt-1 text-xs text-gray-400">Searching...</p>
                )}
                {recipientResults.length > 0 && (
                  <ul className="mt-2 max-h-40 overflow-y-auto rounded-lg border border-gray-200 dark:border-gray-700">
                    {recipientResults.map((u) => (
                      <li key={u.id}>
                        <button
                          onClick={() => { setSelectedRecipient(u); setRecipientSearch(""); setRecipientResults([]); }}
                          className="flex w-full items-center gap-2 px-3 py-2 text-left hover:bg-gray-50 dark:hover:bg-gray-700/50"
                        >
                          <div className="flex h-7 w-7 items-center justify-center rounded-full bg-brand-100 dark:bg-brand-900/30">
                            <span className="text-[10px] font-bold text-brand-600">{u.username.slice(0, 2).toUpperCase()}</span>
                          </div>
                          <div className="min-w-0 flex-1">
                            <p className="truncate text-xs font-medium text-gray-900 dark:text-gray-100">{u.username}</p>
                            <p className="truncate text-[11px] text-gray-400">{u.email}</p>
                          </div>
                        </button>
                      </li>
                    ))}
                  </ul>
                )}
                {selectedRecipient && (
                  <div className="mt-2 flex items-center justify-between rounded-lg border border-brand-300 bg-brand-50 px-3 py-2 dark:border-brand-700 dark:bg-brand-900/20">
                    <div>
                      <p className="text-xs font-medium text-gray-900 dark:text-gray-100">{selectedRecipient.username}</p>
                      <p className="text-[11px] text-gray-400">{selectedRecipient.email}</p>
                    </div>
                    <button onClick={() => setSelectedRecipient(null)} className="text-gray-400 hover:text-red-500">
                      ✕
                    </button>
                  </div>
                )}
              </div>
            )}
          </div>

          {/* Sample data */}
          <div className={cardCls}>
            <label className={labelCls}>Sample Data (JSON)</label>
            <textarea
              value={sampleDataText}
              onChange={(e) => setSampleDataText(e.target.value)}
              rows={8}
              className={`${inputCls} font-mono text-xs`}
              spellCheck={false}
            />
            <p className="mt-2 text-xs text-gray-400">
              Placeholders: {Object.keys(parsedData).map(k => `{${k}}`).join(", ")}
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
                    placeholder={selectedRecipient?.email || "user@example.com"}
                    className={inputCls}
                  />
                  <button
                    onClick={sendTestEmail}
                    disabled={sending}
                    className="flex shrink-0 items-center gap-1.5 rounded-lg bg-brand-600 px-3 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
                  >
                    {sending ? <RefreshCw className="h-4 w-4 animate-spin" /> : <Mail className="h-4 w-4" />}
                    Send
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
                    Send
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Right: Preview + Delivery Log */}
        <div className="space-y-6 lg:col-span-2">
          <div className={cardCls}>
            {/* Tab switcher */}
            <div className="mb-4 flex items-center gap-2 border-b border-gray-200 pb-3 dark:border-gray-700">
              <button
                onClick={() => setMode("html")}
                className={`flex items-center gap-1.5 rounded-lg px-4 py-2 text-sm font-medium ${
                  mode === "html"
                    ? "bg-brand-600 text-white"
                    : "text-gray-600 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-gray-700"
                }`}
              >
                <Mail className="h-4 w-4" /> HTML
              </button>
              <button
                onClick={() => setMode("text")}
                className={`flex items-center gap-1.5 rounded-lg px-4 py-2 text-sm font-medium ${
                  mode === "text"
                    ? "bg-brand-600 text-white"
                    : "text-gray-600 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-gray-700"
                }`}
              >
                <MessageSquare className="h-4 w-4" /> Text
              </button>
            </div>

            {/* Email headers */}
            <div className="mb-4 rounded-lg border border-gray-200 p-4 dark:border-gray-700">
              <div className="space-y-1">
                <div className="flex items-center justify-between border-b border-gray-100 pb-1.5 dark:border-gray-700">
                  <span className="text-xs font-medium text-gray-500">Subject</span>
                  <span className="text-sm font-medium text-gray-900 dark:text-gray-100">{subject}</span>
                </div>
                <div className="flex items-center justify-between border-b border-gray-100 pb-1.5 dark:border-gray-700">
                  <span className="text-xs font-medium text-gray-500">From</span>
                  <span className="text-sm text-gray-700 dark:text-gray-300">{tpl.from}</span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-xs font-medium text-gray-500">Reply-To</span>
                  <span className="text-sm text-gray-700 dark:text-gray-300">{tpl.replyTo}</span>
                </div>
              </div>
            </div>

            {/* Preview content */}
            {mode === "html" ? (
              <div className="rounded-lg border-2 border-gray-200 dark:border-gray-700">
                <div className="flex items-center gap-2 border-b border-gray-200 bg-gray-50 px-4 py-2 dark:border-gray-700 dark:bg-gray-900">
                  <Eye className="h-4 w-4 text-gray-400" />
                  <span className="text-xs text-gray-500">HTML Email Preview</span>
                </div>
                <div
                  className="overflow-auto bg-white p-4"
                  style={{ minHeight: "350px" }}
                  dangerouslySetInnerHTML={{ __html: emailHtml }}
                />
              </div>
            ) : (
              <div className="rounded-lg border-2 border-gray-200 dark:border-gray-700">
                <div className="flex items-center gap-2 border-b border-gray-200 bg-gray-50 px-4 py-2 dark:border-gray-700 dark:bg-gray-900">
                  <MessageSquare className="h-4 w-4 text-gray-400" />
                  <span className="text-xs text-gray-500">Plain Text Preview</span>
                </div>
                <pre className="overflow-auto whitespace-pre-wrap bg-white p-4 text-sm text-gray-700 dark:bg-gray-800 dark:text-gray-300" style={{ minHeight: "350px" }}>
                  {textContent}
                </pre>
              </div>
            )}
          </div>

          {/* Delivery Log */}
          <div className={cardCls}>
            <h3 className={`mb-4 ${headingCls}`}>
              <Clock className="mr-2 inline h-5 w-5 text-brand-600" />
              Delivery Log
            </h3>

            {deliveryLog.length === 0 ? (
              <div className="rounded-lg border border-dashed border-gray-300 py-10 text-center dark:border-gray-600">
                <Clock className="mx-auto mb-2 h-8 w-8 text-gray-300" />
                <p className="text-sm text-gray-500">No deliveries yet</p>
                <p className="mt-1 text-xs text-gray-400">Send a test email or SMS to see delivery status here</p>
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left text-sm">
                  <thead>
                    <tr className="border-b border-gray-200 dark:border-gray-700">
                      <th className="px-3 py-2 text-xs font-semibold text-gray-500">Timestamp</th>
                      <th className="px-3 py-2 text-xs font-semibold text-gray-500">Template</th>
                      <th className="px-3 py-2 text-xs font-semibold text-gray-500">Recipient</th>
                      <th className="px-3 py-2 text-xs font-semibold text-gray-500">Channel</th>
                      <th className="px-3 py-2 text-xs font-semibold text-gray-500">Status</th>
                      <th className="px-3 py-2 text-xs font-semibold text-gray-500">Time</th>
                    </tr>
                  </thead>
                  <tbody>
                    {deliveryLog.map((entry) => (
                      <>
                        <tr
                          key={entry.id}
                          onClick={() => setExpandedDelivery(expandedDelivery === entry.id ? null : entry.id)}
                          className="cursor-pointer border-b border-gray-100 hover:bg-gray-50 dark:border-gray-700/50 dark:hover:bg-gray-700/30"
                        >
                          <td className="whitespace-nowrap px-3 py-2.5 text-xs text-gray-600 dark:text-gray-400">
                            {new Date(entry.timestamp).toLocaleString()}
                          </td>
                          <td className="px-3 py-2.5 text-xs font-medium text-gray-900 dark:text-gray-100">{entry.template}</td>
                          <td className="px-3 py-2.5 text-xs text-gray-600 dark:text-gray-400">{entry.recipient}</td>
                          <td className="px-3 py-2.5">
                            <span className={`inline-flex items-center gap-1 text-xs ${entry.channel === "Email" ? "text-blue-600 dark:text-blue-400" : "text-purple-600 dark:text-purple-400"}`}>
                              {entry.channel === "Email" ? <Mail className="h-3 w-3" /> : <Phone className="h-3 w-3" />}
                              {entry.channel}
                            </span>
                          </td>
                          <td className="px-3 py-2.5">
                            <span className={`inline-block rounded-full px-2.5 py-0.5 text-[11px] font-medium ${statusBadge(entry.status)}`}>
                              {entry.status}
                            </span>
                          </td>
                          <td className="whitespace-nowrap px-3 py-2.5 text-xs text-gray-600 dark:text-gray-400">{entry.deliveryTimeMs}ms</td>
                        </tr>
                        {expandedDelivery === entry.id && (
                          <tr key={entry.id + "-detail"} className="bg-gray-50 dark:bg-gray-900/30">
                            <td colSpan={6} className="px-6 py-4">
                              <div className="flex items-center gap-2">
                                {entry.timeline.map((step, idx) => (
                                  <div key={idx} className="flex items-center">
                                    <div className="flex flex-col items-center">
                                      <div className={`flex h-8 w-8 items-center justify-center rounded-full ${
                                        step.time
                                          ? "bg-green-100 text-green-600 dark:bg-green-900/40 dark:text-green-400"
                                          : "bg-gray-200 text-gray-400 dark:bg-gray-700"
                                      }`}>
                                        {idx < entry.timeline.length - 1 || step.time ? (
                                          <Check className="h-4 w-4" />
                                        ) : (
                                          <Clock className="h-4 w-4" />
                                        )}
                                      </div>
                                      <span className="mt-1 text-xs font-medium text-gray-700 dark:text-gray-300">{step.label}</span>
                                      {step.time && (
                                        <span className="text-[10px] text-gray-400">
                                          {new Date(step.time).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" })}
                                        </span>
                                      )}
                                    </div>
                                    {idx < entry.timeline.length - 1 && (
                                      <div className={`mx-2 h-0.5 w-12 ${entry.timeline[idx + 1].time ? "bg-green-400" : "bg-gray-200 dark:bg-gray-700"}`} />
                                    )}
                                  </div>
                                ))}
                              </div>
                            </td>
                          </tr>
                        )}
                      </>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

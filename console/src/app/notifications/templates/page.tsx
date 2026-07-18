"use client";

import { useState, useEffect, useRef } from "react";
import { useApi } from "@/lib/api";
import {
  Mail,
  Save,
  Loader2,
  Send,
  Eye,
  Code2,
  Languages,
  Plus,
  FileText,
  Trash2,
  History,
  CheckCircle2,
  Clock,
  ChevronDown,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface TemplateContent {
  subject: string;
  htmlBody: string;
  textBody: string;
  status: "draft" | "active";
  version: number;
}

interface TemplateRecord {
  id: string;
  type: TemplateType;
  language: string;
  content: TemplateContent;
  versions: { version: number; savedAt: string; htmlBody: string; subject: string }[];
}

type TemplateType =
  | "welcome"
  | "password-reset"
  | "mfa-code"
  | "account-locked"
  | "invitation"
  | "password-changed"
  | "role-assigned"
  | "custom";

const TEMPLATE_LABELS: Record<TemplateType, string> = {
  welcome: "Welcome",
  "password-reset": "Password Reset",
  "mfa-code": "MFA Code",
  "account-locked": "Account Locked",
  invitation: "Invitation",
  "password-changed": "Password Changed",
  "role-assigned": "Role Assigned",
  custom: "Custom",
};

const TEMPLATE_TYPES: TemplateType[] = [
  "welcome",
  "password-reset",
  "mfa-code",
  "account-locked",
  "invitation",
  "password-changed",
  "role-assigned",
  "custom",
];

const LANGUAGES = [
  { code: "en", label: "English", flag: "EN" },
  { code: "zh", label: "中文", flag: "ZH" },
  { code: "es", label: "Español", flag: "ES" },
  { code: "fr", label: "Français", flag: "FR" },
  { code: "de", label: "Deutsch", flag: "DE" },
  { code: "ja", label: "日本語", flag: "JA" },
];

const VARIABLES = [
  { key: "{{.UserName}}", label: "User Name", sample: "John Doe" },
  { key: "{{.Code}}", label: "Verification Code", sample: "829374" },
  { key: "{{.Link}}", label: "Action Link", sample: "https://app.example.com/verify?token=abc123" },
  { key: "{{.Email}}", label: "User Email", sample: "john@example.com" },
  { key: "{{.TenantName}}", label: "Tenant Name", sample: "Acme Corp" },
  { key: "{{.ExpiryDate}}", label: "Expiry Date", sample: "2024-01-20 18:00 UTC" },
  { key: "{{.ActionURL}}", label: "Action URL", sample: "https://app.example.com/action?id=xyz" },
];

const SAMPLE_DATA: Record<string, string> = Object.fromEntries(VARIABLES.map((v: any) => [v.key, v.sample]));

function defaultTemplate(type: TemplateType): { subject: string; htmlBody: string; textBody: string } {
  switch (type) {
    case "welcome":
      return {
        subject: "Welcome to {{.TenantName}}, {{.UserName}}!",
        htmlBody: `<h1>Welcome, {{.UserName}}!</h1>
<p>Your account has been created on <strong>{{.TenantName}}</strong>.</p>
<p>Email: {{.Email}}</p>
<p>Get started: <a href="{{.ActionURL}}">{{.ActionURL}}</a></p>`,
        textBody: `Welcome, {{.UserName}}!\n\nYour account has been created on {{.TenantName}}.\nEmail: {{.Email}}\nGet started: {{.ActionURL}}`,
      };
    case "password-reset":
      return {
        subject: "Password Reset — {{.TenantName}}",
        htmlBody: `<h1>Password Reset</h1>
<p>Click the link below to reset your password:</p>
<p><a href="{{.Link}}">{{.Link}}</a></p>
<p>Expires: {{.ExpiryDate}}</p>`,
        textBody: `Password Reset\n\nReset link: {{.Link}}\nExpires: {{.ExpiryDate}}`,
      };
    case "mfa-code":
      return {
        subject: "Your verification code",
        htmlBody: `<h1>Verification Code</h1>
<h2 style="font-family: monospace; letter-spacing: 4px;">{{.Code}}</h2>
<p>Expires: {{.ExpiryDate}}</p>`,
        textBody: `Verification Code: {{.Code}}\nExpires: {{.ExpiryDate}}`,
      };
    case "account-locked":
      return {
        subject: "Account Locked — {{.TenantName}}",
        htmlBody: `<h1>Account Locked</h1>
<p>Your account ({{.Email}}) has been locked due to too many failed attempts.</p>
<p>Contact support or wait until {{.ExpiryDate}}.</p>`,
        textBody: `Account Locked\n\nYour account ({{.Email}}) has been locked.\nContact support or wait until {{.ExpiryDate}}.`,
      };
    case "invitation":
      return {
        subject: "Invitation to {{.TenantName}}",
        htmlBody: `<h1>You're Invited!</h1>
<p>{{.UserName}} has invited you to join {{.TenantName}}.</p>
<p>Accept: <a href="{{.ActionURL}}">{{.ActionURL}}</a></p>
<p>Expires: {{.ExpiryDate}}</p>`,
        textBody: `You're Invited!\n\nJoin {{.TenantName}}.\nAccept: {{.ActionURL}}\nExpires: {{.ExpiryDate}}`,
      };
    case "password-changed":
      return {
        subject: "Password Changed",
        htmlBody: `<h1>Password Changed</h1>
<p>Your password on {{.TenantName}} was changed.</p>
<p>If this wasn't you, reset immediately: <a href="{{.Link}}">{{.Link}}</a></p>`,
        textBody: `Password Changed\n\nYour password on {{.TenantName}} was changed.\nIf this wasn't you, reset: {{.Link}}`,
      };
    case "role-assigned":
      return {
        subject: "New Role Assigned — {{.TenantName}}",
        htmlBody: `<h1>Role Assigned</h1>
<p>Hello {{.UserName}},</p>
<p>You've been assigned a new role on {{.TenantName}}.</p>
<p>View details: <a href="{{.ActionURL}}">{{.ActionURL}}</a></p>`,
        textBody: `Role Assigned\n\nHello {{.UserName}},\nYou've been assigned a new role on {{.TenantName}}.\nView: {{.ActionURL}}`,
      };
    case "custom":
      return {
        subject: "Custom Template — {{.TenantName}}",
        htmlBody: `<h1>{{.UserName}}</h1>\n<p>{{.Email}}</p>`,
        textBody: `{{.UserName}} - {{.Email}}`,
      };
  }
}

const STORAGE_KEY = "ggid_notification_templates_v2";

function genId() {
  return `tpl-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

export default function NotificationTemplatesPage() {
  const t = useTranslations();
  const { apiFetch } = useApi();
  const [templates, setTemplates] = useState<TemplateRecord[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [selectedLang, setSelectedLang] = useState("en");
  const [msg, setMsg] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [sendingTest, setSendingTest] = useState(false);
  const [showPreview, setShowPreview] = useState(false);
  const [showVersionHistory, setShowVersionHistory] = useState(false);

  const htmlRef = useRef<HTMLTextAreaElement>(null);
  const textRef = useRef<HTMLTextAreaElement>(null);

  // Load from localStorage
  useEffect(() => {
    const stored = typeof window !== "undefined" ? localStorage.getItem(STORAGE_KEY) : null;
    if (stored) {
      try {
        setTemplates(JSON.parse(stored));
        return;
      } catch {
        // fall through
      }
    }
    // Initialize with default welcome template
    const init: TemplateRecord = {
      id: genId(),
      type: "welcome",
      language: "en",
      content: { ...defaultTemplate("welcome"), status: "draft", version: 1 },
      versions: [],
    };
    setTemplates([init]);
    setSelectedId(init.id);
  }, []);

  // Persist to localStorage
  useEffect(() => {
    if (templates.length > 0) {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(templates));
    }
  }, [templates]);

  useEffect(() => {
    if (msg) {
      const t = setTimeout(() => setMsg(null), 3000);
      return () => clearTimeout(t);
    }
  }, [msg]);

  const selectedTemplate = templates.find((t: any) => t.id === selectedId) || null;

  // Get content for current language (fall back to English)
  const langTemplate: TemplateRecord | null =
    selectedTemplate && selectedTemplate.language === selectedLang
      ? selectedTemplate
      : templates.find((t: any) => t.type === selectedTemplate?.type && t.language === selectedLang) || null;

  const current = langTemplate?.content || null;

  const updateCurrent = (updater: (c: TemplateContent) => TemplateContent) => {
    if (!langTemplate) return;
    setTemplates((prev) =>
      prev.map((t: any) =>
        t.id === langTemplate.id
          ? { ...t, content: updater({ ...t.content }) }
          : t,
      ),
    );
  };

  const renderWithSample = (text: string): string => {
    let result = text;
    Object.entries(SAMPLE_DATA).forEach(([key, val]) => {
      result = result.replaceAll(key, val);
    });
    return result;
  };

  const insertVariable = (varKey: string, target: "html" | "text") => {
    const textarea = target === "html" ? htmlRef.current : textRef.current;
    if (!textarea) return;
    const start = textarea.selectionStart;
    const end = textarea.selectionEnd;
    if (target === "html" && current) {
      const newBody = current.htmlBody.slice(0, start) + varKey + current.htmlBody.slice(end);
      updateCurrent((c) => ({ ...c, htmlBody: newBody }));
    } else if (current) {
      const newBody = current.textBody.slice(0, start) + varKey + current.textBody.slice(end);
      updateCurrent((c) => ({ ...c, textBody: newBody }));
    }
    requestAnimationFrame(() => {
      textarea.focus();
      const pos = start + varKey.length;
      textarea.setSelectionRange(pos, pos);
    });
  };

  const createTemplate = () => {
    const id = genId();
    const newTpl: TemplateRecord = {
      id,
      type: "custom",
      language: selectedLang,
      content: { ...defaultTemplate("custom"), status: "draft", version: 1 },
      versions: [],
    };
    setTemplates((prev) => [...prev, newTpl]);
    setSelectedId(id);
    setMsg("New template created — choose a type below");
  };

  const deleteTemplate = (id: string) => {
    setTemplates((prev) => {
      const filtered = prev.filter((t: any) => t.id !== id);
      if (selectedId === id && filtered.length > 0) setSelectedId(filtered[0].id);
      return filtered;
    });
  };

  const changeTemplateType = (type: TemplateType) => {
    if (!langTemplate) return;
    if (langTemplate.type === type) return;
    setTemplates((prev) =>
      prev.map((t: any) =>
        t.id === langTemplate.id
          ? { ...t, type, content: { ...defaultTemplate(type), status: "draft", version: 1 }, versions: [] }
          : t,
      ),
    );
  };

  const handleSaveDraft = async () => {
    if (!langTemplate || !current) return;
    setSaving(true);
    try {
      await apiFetch(`/api/v1/settings/notifications/templates/${langTemplate.type}`, {
        method: "PUT",
        body: JSON.stringify({
          type: langTemplate.type,
          language: selectedLang,
          status: "draft",
          subject: current.subject,
          htmlBody: current.htmlBody,
          textBody: current.textBody,
        }),
      });
      setMsg("Saved as draft to server");
    } catch {
      setMsg("Endpoint unavailable — saved to localStorage");
    } finally {
      setSaving(false);
    }
  };

  const handlePublish = async () => {
    if (!langTemplate || !current) return;
    setSaving(true);
    // Save version history
    const newVersion = current.version + 1;
    const versionEntry = {
      version: current.version,
      savedAt: new Date().toISOString(),
      htmlBody: current.htmlBody,
      subject: current.subject,
    };
    updateCurrent((c) => ({ ...c, status: "active", version: newVersion }));
    setTemplates((prev) =>
      prev.map((t: any) =>
        t.id === langTemplate.id
          ? { ...t, content: { ...t.content, status: "active", version: newVersion }, versions: [versionEntry, ...t.versions] }
          : t,
      ),
    );
    try {
      await apiFetch(`/api/v1/settings/notifications/templates/${langTemplate.type}/publish`, {
        method: "POST",
        body: JSON.stringify({
          type: langTemplate.type,
          language: selectedLang,
          status: "active",
          subject: current.subject,
          htmlBody: current.htmlBody,
          textBody: current.textBody,
        }),
      });
      setMsg("Template published and activated");
    } catch {
      setMsg("Published locally — server endpoint unavailable");
    } finally {
      setSaving(false);
    }
  };

  const handleSendTest = async () => {
    if (!current) return;
    setSendingTest(true);
    try {
      await apiFetch("/api/v1/notifications/test", {
        method: "POST",
        body: JSON.stringify({
          subject: renderWithSample(current.subject),
          htmlBody: renderWithSample(current.htmlBody),
        }),
      });
      setMsg("Test email sent to your account");
    } catch (err) {
      setMsg(err instanceof Error ? `Test send failed: ${err.message}` : "Test send failed");
    } finally {
      setSendingTest(false);
    }
  };

  const restoreVersion = (version: number) => {
    if (!langTemplate) return;
    const v = langTemplate.versions.find((x: any) => x.version === version);
    if (!v) return;
    setTemplates((prev) =>
      prev.map((t: any) =>
        t.id === langTemplate.id
          ? { ...t, content: { ...t.content, htmlBody: v.htmlBody, subject: v.subject } }
          : t,
      ),
    );
    setShowVersionHistory(false);
    setMsg(`Restored version ${version}`);
  };

  const inputCls =
    "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";

  if (!current) {
    return (
      <div className="p-8 text-center">
        <Mail className="mx-auto mb-4 h-12 w-12 text-gray-300" />
        <p className="text-gray-500">No template selected. Create one to get started.</p>
        <button
          onClick={createTemplate}
          aria-label="Create new template"
          className="mt-4 inline-flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
        >
          <Plus className="h-4 w-4" /> New Template
        </button>
      </div>
    );
  }

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-2xl font-bold dark:text-gray-100">
          <Mail className="h-6 w-6 text-brand-600" /> Notification Templates
        </h1>
        <div className="flex gap-2">
          <button
            onClick={handleSendTest}
            disabled={sendingTest}
            aria-label="Send test email"
            className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700 disabled:opacity-50"
          >
            {sendingTest ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />} Send Test
          </button>
          <button
            onClick={handleSaveDraft}
            disabled={saving}
            aria-label="Save template as draft"
            className="flex items-center gap-1.5 rounded-lg border border-gray-300 bg-gray-50 px-4 py-2 text-sm font-medium hover:bg-gray-100 dark:border-gray-600 dark:bg-gray-700 dark:hover:bg-gray-600 disabled:opacity-50"
          >
            {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />} Save Draft
          </button>
          <button
            onClick={handlePublish}
            disabled={saving}
            aria-label="Publish template"
            className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
          >
            <CheckCircle2 className="h-4 w-4" /> Publish
          </button>
        </div>
      </div>

      {msg && (
        <div role="status" className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400">
          {msg}
        </div>
      )}

      <div className="grid gap-4 lg:grid-cols-[200px_1fr_220px]">
        {/* ===== Template List Sidebar ===== */}
        <div className="lg:sticky lg:top-4 lg:self-start">
          <div className="rounded-lg border border-gray-200 bg-white p-3 shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <div className="mb-3 flex items-center justify-between">
              <h3 className="text-xs font-semibold uppercase text-gray-500">Templates</h3>
              <button onClick={createTemplate} aria-label="Create new template" className="rounded p-1 text-brand-600 hover:bg-brand-50 dark:hover:bg-brand-950">
                <Plus className="h-4 w-4" />
              </button>
            </div>
            <div className="space-y-1">
              {templates.map((tpl: any) => (
                <div
                  key={tpl.id}
                  onClick={() => { setSelectedId(tpl.id); setSelectedLang(tpl.language); }}
                  className={`group cursor-pointer rounded-lg border p-2 transition-colors ${
                    selectedId === tpl.id
                      ? "border-brand-400 bg-brand-50 dark:border-brand-700 dark:bg-brand-950/30"
                      : "border-gray-200 hover:border-gray-300 dark:border-gray-700"
                  }`}
                >
                  <div className="flex items-center justify-between">
                    <p className="truncate text-xs font-medium">
                      {TEMPLATE_LABELS[tpl.type as keyof typeof TEMPLATE_LABELS]}
                    </p>
                    <button
                      onClick={(e) => { e.stopPropagation(); deleteTemplate(tpl.id); }}
                      className="ml-1 text-gray-300 opacity-0 group-hover:opacity-100 hover:text-red-500"
                      aria-label="Delete template"
                    >
                      <Trash2 className="h-3 w-3" />
                    </button>
                  </div>
                  <div className="mt-1 flex items-center gap-2">
                    <span className="rounded bg-gray-100 px-1 py-0.5 text-[10px] uppercase text-gray-500 dark:bg-gray-700">
                      {tpl.language}
                    </span>
                    <span
                      className={`rounded px-1 py-0.5 text-[10px] font-medium ${
                        tpl.content.status === "active"
                          ? "bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-400"
                          : "bg-yellow-100 text-yellow-700 dark:bg-yellow-950 dark:text-yellow-400"
                      }`}
                    >
                      {tpl.content.status === "active" ? "Active" : "Draft"}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* ===== Editor Area ===== */}
        <div className="space-y-4">
          {/* Type & Language selectors */}
          <div className="grid gap-3 sm:grid-cols-2">
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">Template Type</label>
              <select
                value={langTemplate?.type || "custom"}
                onChange={(e) => changeTemplateType(e.target.value as TemplateType)}
                className={inputCls}
              >
                {TEMPLATE_TYPES.map((t: any) => (
                  <option key={t} value={t}>{TEMPLATE_LABELS[t as keyof typeof TEMPLATE_LABELS]}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="mb-1 flex items-center gap-1.5 text-xs font-medium text-gray-500">
                <Languages className="h-3.5 w-3.5" /> Language
              </label>
              <select
                value={selectedLang}
                onChange={(e) => setSelectedLang(e.target.value)}
                className={inputCls}
              >
                {LANGUAGES.map((l: any) => (
                  <option key={l.code} value={l.code}>{l.label}</option>
                ))}
              </select>
            </div>
          </div>

          {/* Status badge & version history */}
          <div className="flex items-center gap-3">
            <span
              className={`flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-medium ${
                current.status === "active"
                  ? "bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-400"
                  : "bg-yellow-100 text-yellow-700 dark:bg-yellow-950 dark:text-yellow-400"
              }`}
            >
              {current.status === "active" ? <CheckCircle2 className="h-3 w-3" /> : <Clock className="h-3 w-3" />}
              {current.status === "active" ? "Active" : "Draft"} · v{current.version}
            </span>
            {langTemplate && langTemplate.versions.length > 0 && (
              <div className="relative">
                <button
                  onClick={() => setShowVersionHistory(!showVersionHistory)}
                  aria-label="Toggle version history"
                  aria-expanded={showVersionHistory}
                  className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-1 text-xs font-medium hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700"
                >
                  <History className="h-3.5 w-3.5" /> Version History ({langTemplate.versions.length})
                  <ChevronDown className="h-3 w-3" />
                </button>
                {showVersionHistory && (
                  <div className="absolute z-10 mt-1 w-56 rounded-lg border border-gray-200 bg-white p-2 shadow-lg dark:border-gray-700 dark:bg-gray-800">
                    {langTemplate.versions.map((v: any) => (
                      <div
                        key={v.version}
                        onClick={() => restoreVersion(v.version)}
                        className="flex cursor-pointer items-center justify-between rounded p-2 text-xs hover:bg-gray-50 dark:hover:bg-gray-700"
                      >
                        <span>v{v.version}</span>
                        <span className="text-gray-400">{new Date(v.savedAt).toLocaleString()}</span>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}
          </div>

          {/* Subject */}
          <div>
            <label className="mb-1 block text-xs font-medium text-gray-500">Subject Line</label>
            <input
              value={current.subject}
              onChange={(e) => updateCurrent((c) => ({ ...c, subject: e.target.value }))}
              placeholder="Email subject..."
              className={inputCls}
            />
          </div>

          {/* Split view: HTML editor + Plain text editor */}
          <div className="grid gap-4 md:grid-cols-2">
            {/* HTML editor */}
            <div>
              <label className="mb-1 flex items-center gap-1.5 text-xs font-medium text-gray-500">
                <Code2 className="h-3.5 w-3.5" /> HTML Body
              </label>
              <div className="relative">
                <textarea
                  ref={htmlRef}
                  value={current.htmlBody}
                  onChange={(e) => updateCurrent((c) => ({ ...c, htmlBody: e.target.value }))}
                  rows={14}
                  placeholder="Enter HTML email content..."
                  className="w-full rounded-lg border border-gray-300 px-3 py-2 pl-12 font-mono text-xs leading-relaxed dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
                />
                {/* Line numbers gutter */}
                <div className="pointer-events-none absolute left-0 top-0 w-10 select-none overflow-hidden py-2 pr-2 text-right font-mono text-xs leading-relaxed text-gray-400">
                  {current.htmlBody.split("\n").map((_, i) => (
                    <div key={i}>{i + 1}</div>
                  ))}
                </div>
              </div>
            </div>

            {/* Plain text editor */}
            <div>
              <label className="mb-1 flex items-center gap-1.5 text-xs font-medium text-gray-500">
                <FileText className="h-3.5 w-3.5" /> Plain Text Body
              </label>
              <textarea
                ref={textRef}
                value={current.textBody}
                onChange={(e) => updateCurrent((c) => ({ ...c, textBody: e.target.value }))}
                rows={14}
                placeholder="Enter plain text email content..."
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-xs leading-relaxed dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
              />
            </div>
          </div>

          {/* Preview */}
          <div>
            <button
              onClick={() => setShowPreview(!showPreview)}
              aria-label={showPreview ? "Hide preview" : "Show preview"}
              aria-expanded={showPreview}
              className="mb-2 flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-1.5 text-xs font-medium hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700"
            >
              <Eye className="h-3.5 w-3.5" /> {showPreview ? "Hide" : "Show"} Preview
            </button>
            {showPreview && (
              <div className="overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-800">
                {/* Email container header */}
                <div className="border-b border-gray-100 bg-gray-50 px-4 py-2 dark:border-gray-700 dark:bg-gray-900">
                  <p className="text-xs font-semibold text-gray-600 dark:text-gray-400">
                    {renderWithSample(current.subject)}
                  </p>
                </div>
                <div
                  className="p-4 text-sm text-gray-700 dark:text-gray-300 [&_a]:text-brand-600 [&_a]:underline"
                  dangerouslySetInnerHTML={{ __html: renderWithSample(current.htmlBody) }}
                />
              </div>
            )}
          </div>
        </div>

        {/* ===== Variable Selector Sidebar ===== */}
        <div className="lg:sticky lg:top-4 lg:self-start">
          <div className="rounded-lg border border-gray-200 bg-white p-3 shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <h3 className="mb-1 text-xs font-semibold uppercase text-gray-500">Variables</h3>
            <p className="mb-3 text-xs text-gray-400">Click to insert at cursor</p>
            <div className="space-y-1">
              {VARIABLES.map((v: any) => (
                <div key={v.key} className="space-y-0.5">
                  <p className="text-xs text-gray-400">{v.label}</p>
                  <div className="flex gap-1">
                    <button
                      onClick={() => insertVariable(v.key, "html")}
                      aria-label={`Insert ${v.label} variable into HTML body`}
                      className="flex-1 rounded border border-gray-200 px-2 py-1 text-left transition-colors hover:border-brand-300 hover:bg-brand-50 dark:border-gray-600 dark:hover:border-brand-700 dark:hover:bg-brand-950/30"
                    >
                      <p className="font-mono text-xs text-brand-600 dark:text-brand-400">{v.key}</p>
                    </button>
                    <button
                      onClick={() => insertVariable(v.key, "text")}
                      aria-label={`Insert ${v.label} variable into text body`}
                      className="rounded border border-gray-200 px-1.5 py-1 text-xs text-gray-400 hover:border-brand-300 hover:text-brand-600 dark:border-gray-600"
                      title="Insert into text body"
                    >
                      <FileText className="h-3 w-3" />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

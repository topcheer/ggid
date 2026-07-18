"use client";
import { useState, useEffect, useCallback } from "react";
import { useTranslations } from "@/lib/i18n";
import { Mail, Send, Eye } from "lucide-react";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";
const templates = ["welcome", "password_reset", "mfa_setup", "account_locked", "access_granted"];
const variables = ["{{.UserName}}", "{{.UserEmail}}", "{{.ResetLink}}", "{{.OrgName}}", "{{.LoginURL}}", "{{.OTPCode}}", "{{.RoleName}}", "{{.ExpiryTime}}"];
const langs = ["en", "zh", "ja", "es", "de"];
const defaultHtml = { welcome: "<h1>Welcome {{.UserName}}!</h1><p>Your account on {{.OrgName}} is ready. <a href=\"{{.LoginURL}}\">Login now</a></p>", password_reset: "<h1>Reset Your Password</h1><p>Click <a href=\"{{.ResetLink}}\">here</a> to reset. Expires: {{.ExpiryTime}}</p>", mfa_setup: "<h1>MFA Setup</h1><p>Your code: <strong>{{.OTPCode}}</strong></p>", account_locked: "<h1>Account Locked</h1><p>Too many failed attempts. Contact admin.</p>", access_granted: "<h1>Access Granted</h1><p>You now have role: {{.RoleName}}</p>" };
export default function EmailTemplateEditorPage() {
  const t = useTranslations();
  const [template, setTemplate] = useState("welcome");
  const [lang, setLang] = useState("en");
  const [content, setContent] = useState(defaultHtml.welcome);
  const [preview, setPreview] = useState(true);
  const [sending, setSending] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [contentByLang, setContentByLang] = useState<Record<string, string>>({ en: defaultHtml.welcome });
  const loadData = useCallback(async () => {
    setLoading(true); setError(null);
    try { const res = await fetch("/api/v1/notification/email-templates?template=" + template, { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); if (res.ok) { const d = await res.json(); if (d.content) { setContent(d.content); setContentByLang({ ...contentByLang, [template + "_" + lang]: d.content }); } } }
    catch (err) { setError(err instanceof Error ? err.message : t("emailEditor.error")); } finally { setLoading(false); }
  }, [template, lang, t]);
  useEffect(() => { loadData(); }, [loadData]);
  const selectTemplate = (tmpl: string) => { setTemplate(tmpl); const key = tmpl + "_" + lang; setContent(contentByLang[key] || defaultHtml[tmpl as keyof typeof defaultHtml] || ""); };
  const selectLang = (l: string) => { setLang(l); const key = template + "_" + l; setContent(contentByLang[key] || defaultHtml[template as keyof typeof defaultHtml] || ""); };
  const updateContent = (c: string) => { setContent(c); setContentByLang({ ...contentByLang, [template + "_" + lang]: c }); };
  const insertVar = (v: string) => updateContent(content + v);
  const sendTest = async () => { setSending(true); try { await fetch("/api/v1/notification/email-test", { method: "POST", headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify({ template, lang, content }) }); } catch { /* noop */ } finally { setSending(false); } };
  if (loading) return (<div className="p-8 flex items-center justify-center"><div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" /></div>);
  if (error) return (<div className="p-8"><div className="rounded-lg border border-red-300 bg-red-50 dark:bg-red-950 dark:border-red-800 p-4"><p className="text-red-700 dark:text-red-400 text-sm font-medium">{t("common.error")}: {error}</p><button aria-label="action" onClick={loadData} className="mt-2 px-4 py-1.5 rounded-lg bg-red-600 text-white text-sm hover:bg-red-700">{t("common.retry")}</button></div></div>);
  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><Mail className="w-6 h-6 text-blue-500" /> {t("emailEditor.title")}</h1><p className="text-sm text-gray-500 mt-1">{t("emailEditor.subtitle")}</p></div>
      <div className="flex items-center gap-3"><select aria-label="Template" value={template} onChange={(e) => selectTemplate(e.target.value)} className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm">{templates.map((tmpl: any) => <option key={tmpl} value={tmpl}>{tmpl}</option>)}</select><div className="flex gap-1">{langs.map((l: any) => <button key={l} onClick={() => selectLang(l)} className={"px-2 py-1 rounded text-xs font-medium " + (lang === l ? "bg-blue-600 text-white" : "border dark:border-gray-700")}>{l.toUpperCase()}</button>)}</div></div>
      <div className="flex flex-wrap gap-1">{variables.map((v: any) => <button key={v} onClick={() => insertVar(v)} className="px-2 py-0.5 rounded text-xs font-mono bg-gray-100 dark:bg-gray-800 hover:bg-blue-100 dark:hover:bg-blue-900/30">{v}</button>)}</div>
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4"><div><div className="flex items-center justify-between mb-2"><h3 className="text-sm font-semibold">{t("emailEditor.htmlEditor")}</h3><div className="flex gap-2"><button onClick={() => setPreview(!preview)} className="text-xs flex items-center gap-1 text-gray-500"><Eye className="w-3.5 h-3.5" /> {preview ? t("emailEditor.edit") : t("emailEditor.preview")}</button><button onClick={sendTest} disabled={sending} className="text-xs px-3 py-1 rounded bg-blue-600 text-white flex items-center gap-1" aria-label="Send"><Send className="w-3 h-3" /> {sending ? t("emailEditor.sending") : t("emailEditor.sendTest")}</button></div></div>{preview ? <div className="border dark:border-gray-800 rounded-lg p-4 min-h-64" dangerouslySetInnerHTML={{ __html: content }} /> : <textarea aria-label="Content" value={content} onChange={(e) => updateContent(e.target.value)} rows={12} className="w-full px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-xs font-mono" />}</div><div><h3 className="text-sm font-semibold mb-2">{t("emailEditor.preview")}</h3><div className="border dark:border-gray-800 rounded-lg p-4 min-h-64 bg-gray-50 dark:bg-gray-900" dangerouslySetInnerHTML={{ __html: content }} /></div></div>
    </div>
  );
}

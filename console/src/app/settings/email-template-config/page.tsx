"use client";

import { useEmailTemplateConfig } from "@ggid/sdk-react";
import { Mail, Eye } from "lucide-react";
import { useState } from "react";
import { useTranslations } from "@/lib/i18n";

export default function EmailTemplateConfigPage() {
  const t = useTranslations();

  const { data, loading, error, refresh } = useEmailTemplateConfig();
  const [selected, setSelected] = useState("welcome");
  const [lang, setLang] = useState("en");
  if (loading) return <div className="p-8 text-gray-400">{t("big1.emailTemplateConfig.loading")}</div>;
  if (error) return <div className="p-8 text-red-400">{t("big1.emailTemplateConfig.error")}{error}</div>;

  const tmpl = (data?.templates ?? []).find((t) => t.id === selected);

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-6">
        <div><h1 className="text-2xl font-bold">{t("big1.emailTemplateConfig.title")}</h1><p className="text-sm text-gray-400 mt-1">{t("big1.emailTemplateConfig.customizeSystemEmails")}</p></div>
        <button aria-label="action" onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">{t("big1.emailTemplateConfig.save")}</button>
      </div>

      <div className="flex gap-2 mb-4">
        {(data?.templates ?? []).map((t) => (
          <button key={t.id} onClick={() => setSelected(t.id)} className={"px-3 py-1.5 rounded-lg text-sm font-medium transition " + (selected === t.id ? "bg-blue-600" : "bg-gray-800 hover:bg-gray-700")}>{t.name}</button>
        ))}
      </div>

      <div className="flex gap-2 mb-4">
        {["en", "zh", "ja"].map((l) => <button key={l} onClick={() => setLang(l)} className={"text-xs px-2 py-1 rounded " + (lang === l ? "bg-blue-600" : "bg-gray-800")}>{l.toUpperCase()}</button>)}
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-2"><Mail className="w-4 h-4 text-blue-400" /><h2 className="text-sm font-semibold">{t("big1.emailTemplateConfig.htmlEditor")}</h2></div>
          <div className="mb-2 flex flex-wrap gap-1">
            {tmpl?.variables.map((v) => <span key={v} className="text-xs font-mono px-1.5 py-0.5 bg-gray-800 rounded text-blue-400 cursor-pointer">{v}</span>)}
          </div>
          <textarea aria-label="Text input" defaultValue={tmpl?.body_html} rows={12} className="w-full px-3 py-2 bg-gray-800 rounded-lg text-xs font-mono" />
          <label className="flex items-center gap-2 mt-2 text-xs"><input aria-label="Toggle option" type="checkbox" defaultChecked={tmpl?.enabled} />{t("big1.emailTemplateConfig.enabled")}</label>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-2"><Eye className="w-4 h-4 text-green-400" /><h2 className="text-sm font-semibold">{t("big1.emailTemplateConfig.preview")}</h2></div>
          <div className="bg-white rounded-lg p-4 text-black text-sm" dangerouslySetInnerHTML={{ __html: tmpl?.body_html?.replace(/\{\{user_name\}\}/g, "John Doe").replace(/\{\{reset_link\}\}/g, "https://ggid.dev/reset?token=xxx") ?? "" }} />
        </div>
      </div>
    </div>
  );
}

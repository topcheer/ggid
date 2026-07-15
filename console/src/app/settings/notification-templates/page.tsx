"use client";
import { useState, useEffect } from "react";
import { useTranslations } from "@/lib/i18n";

interface Template {
  id: string;
  event_type: string;
  language: string;
  channel: "email" | "sms" | "push" | "webhook";
  subject: string;
  body: string;
}

export default function NotificationTemplatesPage() {
  const t = useTranslations();

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [templates, setTemplates] = useState<Template[]>([]);
  const [selected, setSelected] = useState<Template | null>(null);
  const [previewData] = useState({ user_name: "John Doe", action_url: "https://app.ggid.dev/start", tenant_name: "Acme Corp", code: "123456", device: "Chrome / macOS", location: "San Francisco, US" });
  const [testSending, setTestSending] = useState(false);

  useEffect(() => {
    fetch("/api/v1/auth/email-template/config", {
      headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
    })
      .then(res => { if (!res.ok) throw new Error(`HTTP ${res.status}`); return res.json(); })
      .then(data => { setTemplates(Array.isArray(data) ? data : (data.templates || data.items || [])); setLoading(false); })
      .catch(err => { setError(err.message); setLoading(false); });
  }, []);

  const renderPreview = (body: string) => body.replace(/\{\{(\w+)\}\}/g, (_, key: string) => (previewData as Record<string, string>)[key] || `{{${key}}}`);

  const channelColors: Record<string, string> = { email: "bg-blue-100 text-blue-700", sms: "bg-green-100 text-green-700", push: "bg-purple-100 text-purple-700", webhook: "bg-orange-100 text-orange-700" };

  if (loading) return (
    <div className="p-8"><h1 className="text-2xl font-bold mb-4">Notification Templates</h1><p>Loading...</p></div>
  );
  if (error) return (
    <div className="p-8"><h1 className="text-2xl font-bold mb-4">Notification Templates</h1><p className="text-red-600">Error: {error}</p></div>
  );
  return (
    <div className="p-8 space-y-6 max-w-5xl">
      <h1 className="text-2xl font-bold">Notification Templates</h1>
      <p className="text-gray-600">Manage multi-language notification templates with variables and preview.</p>

      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Templates</h2><table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Event Type</th><th>Language</th><th>Channel</th><th>Subject</th><th>Action</th></tr></thead><tbody>{templates.map((t: Template, i: number) => (<tr key={i} className="border-b hover:bg-gray-50"><td className="py-2 font-mono text-xs">{t.event_type}</td><td><span className="px-1.5 py-0.5 bg-gray-100 rounded text-xs uppercase">{t.language}</span></td><td><span className={`px-2 py-0.5 rounded text-xs ${channelColors[t.channel] || ""}`}>{t.channel}</span></td><td className="text-xs">{t.subject || "-"}</td><td><button onClick={() => setSelected(t)} className="text-xs text-blue-600 hover:underline">Edit</button></td></tr>))}</tbody></table></div>

      {selected && (<div className="bg-white rounded-lg p-6 shadow space-y-4"><div className="flex items-center justify-between"><h2 className="text-lg font-semibold">Edit Template: {selected.event_type} ({selected.language})</h2><button onClick={() => setSelected(null)} className="text-gray-400 hover:text-gray-600">Close</button></div><div><label className="block text-sm font-medium mb-1">Subject</label><input type="text" defaultValue={selected.subject} className="border rounded px-3 py-2 w-full" /></div><div><label className="block text-sm font-medium mb-1">Body <span className="text-xs text-gray-400">{'(use {{user_name}}, {{action_url}}, {{tenant_name}}, {{code}}, {{device}}, {{location}})'}</span></label><textarea defaultValue={selected.body} className="border rounded px-3 py-2 w-full font-mono text-sm" rows={6} /></div><div className="border-l-4 border-blue-400 bg-blue-50 p-3"><div className="text-xs font-medium text-gray-500 mb-1">Preview (with sample data)</div><pre className="text-sm whitespace-pre-wrap font-sans">{renderPreview(selected.body)}</pre></div><div className="flex gap-3"><button className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 text-sm">Save</button><button onClick={() => { setTestSending(true); setTimeout(() => setTestSending(false), 800); }} disabled={testSending} className="px-4 py-2 border rounded hover:bg-gray-50 text-sm disabled:opacity-50">{testSending ? "Sending..." : "Test Send"}</button></div></div>)}

      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-2">Available Variables</h2><div className="flex flex-wrap gap-2">{Object.keys(previewData).map((v) => <span key={v} className="px-2 py-1 bg-gray-100 rounded text-xs font-mono">{'{{'}{v}{'}}'}</span>)}</div></div>
    </div>
  );
}

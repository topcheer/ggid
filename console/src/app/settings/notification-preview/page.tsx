'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";

export default function NotificationPreviewPage() {
  const t = useTranslations();

  const [template, setTemplate] = useState('welcome');
  const [variables, setVariables] = useState({ name: 'Alice', org: 'GGID Corp', code: '123456' });
  const [darkMode, setDarkMode] = useState(false);
  const [mobilePreview, setMobilePreview] = useState(false);
  const [sendResult, setSendResult] = useState('');
  const [sending, setSending] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch('/api/v1/notifications/send', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': '00000000-0000-0000-0000-000000000001' },
      body: JSON.stringify({ action: 'list_templates' }),
    })
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(() => setLoading(false))
      .catch(e => { setError(e.message); setLoading(false); });
  }, []);

  const templates: Record<string, { subject: string; html: string; text: string }> = {
    welcome: { subject: 'Welcome to {{org}}', html: '<h2>Welcome, {{name}}!</h2><p>Your account at {{org}} is ready.</p>', text: 'Welcome, {{name}}! Your account at {{org}} is ready.' },
    reset: { subject: 'Password Reset Request', html: '<h2>Reset Your Password</h2><p>Click the link to reset. Code: {{code}}</p>', text: 'Reset Your Password. Code: {{code}}' },
    mfa: { subject: 'Your MFA Code', html: '<h2>Verification Code</h2><p style="font-size:24px;font-weight:bold">{{code}}</p>', text: 'Your verification code is {{code}}' },
    locked: { subject: 'Account Locked', html: '<h2>Account Locked</h2><p>{{name}}, your account at {{org}} has been locked.</p>', text: '{{name}}, your account at {{org}} has been locked.' },
    breach: { subject: 'Security Alert: Data Breach', html: '<h2>Security Alert</h2><p>{{name}}, your password was found in a breach. Please reset immediately.</p>', text: '{{name}}, your password was found in a breach. Please reset.' },
  };

  const render = (s: string) => s.replace(/\{\{(\w+)\}\}/g, (_, k) => (variables as Record<string, string>)[k] || `{{${k}}}`);
  const current = templates[template];
  const [versions] = useState(['v1.0 (initial)', 'v1.1 (added dark mode)', 'v1.2 (current)']);

  const sendTest = () => {
    setSending(true);
    fetch("/api/v1/notifications/send", {
      method: "POST",
      headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
      body: JSON.stringify({ template, variables, to: "preview@example.com" }),
    })
      .then(res => { if (!res.ok) return null; return res.json(); })
      .then(() => { setSendResult('Test email sent to preview@example.com'); setTimeout(() => setSendResult(''), 3000); })
      .catch(err => setSendResult(`Error: ${err.message}`))
      .finally(() => setSending(false));
  };

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      <div><h1 className="text-2xl font-bold">Notification Preview</h1><p className="text-gray-600">Preview email templates with live variable substitution.</p></div>

      <div className="flex gap-4">
        <select value={template} onChange={e => setTemplate(e.target.value)} className="border rounded px-3 py-2 text-sm">
          {Object.keys(templates).map(t => <option key={t} value={t}>{t}</option>)}
        </select>
        <label className="flex items-center gap-1 text-sm"><input type="checkbox" checked={darkMode} onChange={e => setDarkMode(e.target.checked)} className="rounded" />Dark mode</label>
        <label className="flex items-center gap-1 text-sm"><input type="checkbox" checked={mobilePreview} onChange={e => setMobilePreview(e.target.checked)} className="rounded" />Mobile</label>
        <button onClick={sendTest} disabled={sending} className="px-4 py-2 bg-blue-600 text-white rounded text-sm disabled:opacity-50">{sending ? 'Sending...' : 'Send Test'}</button>
      </div>
      {sendResult && <div className="text-sm p-3 rounded bg-green-50 text-green-700">{sendResult}</div>}

      <div className="grid grid-cols-2 gap-6">
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Variables</h2>
          <div className="space-y-3">
            <div><label className="text-sm font-medium">name</label><input type="text" value={variables.name} onChange={e => setVariables(prev => ({ ...prev, name: e.target.value }))} className="w-full border rounded px-3 py-1 text-sm mt-1" /></div>
            <div><label className="text-sm font-medium">org</label><input type="text" value={variables.org} onChange={e => setVariables(prev => ({ ...prev, org: e.target.value }))} className="w-full border rounded px-3 py-1 text-sm mt-1" /></div>
            <div><label className="text-sm font-medium">code</label><input type="text" value={variables.code} onChange={e => setVariables(prev => ({ ...prev, code: e.target.value }))} className="w-full border rounded px-3 py-1 text-sm mt-1" /></div>
          </div>
        </section>

        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Version History</h2>
          <div className="space-y-1">{versions.map(v => <div key={v} className="text-xs text-gray-600 flex items-center gap-2"><span className="w-2 h-2 bg-gray-400 rounded-full" />{v}</div>)}</div>
        </section>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Live Preview</h2>
        <div className={`rounded p-4 ${darkMode ? 'bg-gray-900 text-white' : 'bg-white border'} ${mobilePreview ? 'max-w-sm mx-auto' : ''}`}>
          <div className={`text-sm ${darkMode ? 'text-gray-400' : 'text-gray-500'}`}>Subject: {render(current.subject)}</div>
          <hr className="my-2 border-gray-200" />
          <div className={darkMode ? 'text-gray-100' : 'text-gray-800'} dangerouslySetInnerHTML={{ __html: render(current.html) }} />
        </div>
        <div><div className="text-xs text-gray-500 mb-1">Plain text version:</div><pre className={`rounded p-3 text-xs whitespace-pre-wrap ${darkMode ? 'bg-gray-900 text-gray-300' : 'bg-gray-50 text-gray-700'}`}>{render(current.text)}</pre></div>
      </section>
    </div>
  );
}
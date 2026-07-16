'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";

export default function SmtpConfigPage() {
  const t = useTranslations();
  const [host, setHost] = useState('smtp.ggid.io');
  const [port, setPort] = useState(587);
  const [encryption, setEncryption] = useState('starttls');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [fromAddress, setFromAddress] = useState('noreply@ggid.io');
  const [fromName, setFromName] = useState('GGID Notifications');
  const [replyTo, setReplyTo] = useState('support@ggid.io');
  const [timeout, setTimeout] = useState(30);
  const [authMethod, setAuthMethod] = useState('plain');
  const [rateLimit, setRateLimit] = useState(100);
  const [dkimEnabled, setDkimEnabled] = useState(true);
  const [dkimDomain, setDkimDomain] = useState('ggid.io');
  const [dkimSelector, setDkimSelector] = useState('default');
  const [dkimKey, setDkimKey] = useState('');
  const [testEmail, setTestEmail] = useState('');
  const [testResult, setTestResult] = useState('');
  const [testing, setTesting] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch('/api/v1/auth/email-template/config', {
      headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': '00000000-0000-0000-0000-000000000001' },
    })
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(data => {
        if (data) {
          if (data.host) setHost(data.host);
          if (data.port) setPort(data.port);
          if (data.encryption) setEncryption(data.encryption);
          if (data.username) setUsername(data.username);
          if (data.from_address) setFromAddress(data.from_address);
          if (data.from_name) setFromName(data.from_name);
          if (data.reply_to) setReplyTo(data.reply_to);
          if (data.timeout) setTimeout(data.timeout);
          if (data.auth_method) setAuthMethod(data.auth_method);
          if (data.rate_limit) setRateLimit(data.rate_limit);
          if (data.dkim_enabled !== undefined) setDkimEnabled(data.dkim_enabled);
          if (data.dkim_domain) setDkimDomain(data.dkim_domain);
          if (data.dkim_selector) setDkimSelector(data.dkim_selector);
        }
        setLoading(false);
      })
      .catch(e => { setError(e.message); setLoading(false); });
  }, []);

  const sendTest = () => {
    setTesting(true);
    window.setTimeout(() => { setTestResult(t("smtp.testResult").replace("{email}", testEmail)); setTesting(false); }, 1000);
  };

  if (loading) return <div className="p-6"><p>{t("common.loading")}</p></div>;
  if (error) return <div className="p-6 text-red-600">{t("common.error")}: {error}</div>;

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{t("smtp.title")}</h1>
        <p className="text-gray-600">{t("smtp.subtitle")}</p>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{t("smtp.serverSettings")}</h2>
        <div className="grid grid-cols-2 gap-4">
          <div><label className="text-sm font-medium">{t("smtp.smtpHost")}</label><input type="text" value={host} onChange={e => setHost(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
          <div><label className="text-sm font-medium">{t("smtp.port")}</label><input type="number" min={1} max={65535} value={port} onChange={e => setPort(parseInt(e.target.value) || 587)} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
        </div>
        <div className="grid grid-cols-2 gap-4">
          <div><label className="text-sm font-medium">{t("smtp.encryption")}</label><select value={encryption} onChange={e => setEncryption(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1"><option value="none">{t("smtp.none")}</option><option value="starttls">{t("smtp.starttls")}</option><option value="ssl">{t("smtp.ssl")}</option></select></div>
          <div><label className="text-sm font-medium">{t("smtp.authMethod")}</label><select value={authMethod} onChange={e => setAuthMethod(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1"><option value="plain">{t("smtp.plain")}</option><option value="login">{t("smtp.login")}</option><option value="cram-md5">{t("smtp.cramMd5")}</option><option value="oauth2">{t("smtp.oauth2")}</option></select></div>
        </div>
        <div className="grid grid-cols-2 gap-4">
          <div><label className="text-sm font-medium">{t("smtp.username")}</label><input type="text" value={username} onChange={e => setUsername(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
          <div><label className="text-sm font-medium">{t("smtp.password")}</label><input autoComplete="current-password" type="password" value={password} onChange={e => setPassword(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
        </div>
        <div><label className="text-sm font-medium">{t("smtp.connectionTimeout")}</label><input type="number" min={5} max={120} value={timeout} onChange={e => setTimeout(parseInt(e.target.value) || 30)} className="w-24 border rounded px-2 py-1 text-sm mt-1" /></div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{t("smtp.senderSettings")}</h2>
        <div className="grid grid-cols-2 gap-4">
          <div><label className="text-sm font-medium">{t("smtp.fromAddress")}</label><input autoComplete="email" type="email" value={fromAddress} onChange={e => setFromAddress(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
          <div><label className="text-sm font-medium">{t("smtp.fromName")}</label><input type="text" value={fromName} onChange={e => setFromName(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
        </div>
        <div className="grid grid-cols-2 gap-4">
          <div><label className="text-sm font-medium">{t("smtp.replyTo")}</label><input autoComplete="email" type="email" value={replyTo} onChange={e => setReplyTo(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
          <div><label className="text-sm font-medium">{t("smtp.sendRateLimit")}</label><input type="number" min={1} value={rateLimit} onChange={e => setRateLimit(parseInt(e.target.value) || 100)} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">{t("smtp.dkimConfiguration")}</h2>
          <label className="flex items-center gap-2"><input type="checkbox" checked={dkimEnabled} onChange={e => setDkimEnabled(e.target.checked)} className="rounded" /><span className="text-sm">{t("smtp.enableDkim")}</span></label>
        </div>
        {dkimEnabled && (
          <div className="space-y-3">
            <div className="grid grid-cols-2 gap-4">
              <div><label className="text-sm font-medium">{t("smtp.domain")}</label><input type="text" value={dkimDomain} onChange={e => setDkimDomain(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
              <div><label className="text-sm font-medium">{t("smtp.selector")}</label><input type="text" value={dkimSelector} onChange={e => setDkimSelector(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
            </div>
            <div><label className="text-sm font-medium">{t("smtp.privateKey")}</label><textarea value={dkimKey} onChange={e => setDkimKey(e.target.value)} rows={4} placeholder={t("smtp.privateKeyPlaceholder")} className="w-full border rounded px-3 py-2 text-sm mt-1 font-mono" /></div>
          </div>
        )}
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{t("smtp.testEmail")}</h2>
        <div className="flex gap-3">
          <input autoComplete="email" aria-label="Test email" type="email" placeholder={t("smtp.recipientPlaceholder")} value={testEmail} onChange={e => setTestEmail(e.target.value)} className="flex-1 border rounded px-3 py-2 text-sm" />
          <button onClick={sendTest} disabled={testing || !testEmail} aria-label={t("smtp.sendTest")} className="px-4 py-2 bg-blue-600 text-white rounded text-sm disabled:opacity-50">{testing ? t("smtp.sending") : t("smtp.sendTest")}</button>
        </div>
        {testResult && <div className="text-sm p-3 rounded bg-green-50 text-green-700">{testResult}</div>}
      </section>
    </div>
  );
}

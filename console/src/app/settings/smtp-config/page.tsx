'use client';
import { useState } from 'react';

export default function SmtpConfigPage() {
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

  const sendTest = () => {
    setTesting(true);
    setTimeout(() => { setTestResult(`Test email sent to ${testEmail}`); setTesting(false); }, 1000);
  };

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">SMTP Configuration</h1>
        <p className="text-gray-600">Configure email delivery, authentication, DKIM signing, and rate limiting.</p>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Server Settings</h2>
        <div className="grid grid-cols-2 gap-4">
          <div><label className="text-sm font-medium">SMTP Host</label><input type="text" value={host} onChange={e => setHost(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
          <div><label className="text-sm font-medium">Port</label><input type="number" min={1} max={65535} value={port} onChange={e => setPort(parseInt(e.target.value) || 587)} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
        </div>
        <div className="grid grid-cols-2 gap-4">
          <div><label className="text-sm font-medium">Encryption</label><select value={encryption} onChange={e => setEncryption(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1"><option value="none">None</option><option value="starttls">START_TLS</option><option value="ssl">SSL/TLS</option></select></div>
          <div><label className="text-sm font-medium">Auth Method</label><select value={authMethod} onChange={e => setAuthMethod(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1"><option value="plain">PLAIN</option><option value="login">LOGIN</option><option value="cram-md5">CRAM-MD5</option><option value="oauth2">OAuth2</option></select></div>
        </div>
        <div className="grid grid-cols-2 gap-4">
          <div><label className="text-sm font-medium">Username</label><input type="text" value={username} onChange={e => setUsername(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
          <div><label className="text-sm font-medium">Password</label><input type="password" value={password} onChange={e => setPassword(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
        </div>
        <div><label className="text-sm font-medium">Connection Timeout (s)</label><input type="number" min={5} max={120} value={timeout} onChange={e => setTimeout_val(parseInt(e.target.value) || 30)} className="w-24 border rounded px-2 py-1 text-sm mt-1" /></div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Sender Settings</h2>
        <div className="grid grid-cols-2 gap-4">
          <div><label className="text-sm font-medium">From Address</label><input type="email" value={fromAddress} onChange={e => setFromAddress(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
          <div><label className="text-sm font-medium">From Name</label><input type="text" value={fromName} onChange={e => setFromName(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
        </div>
        <div className="grid grid-cols-2 gap-4">
          <div><label className="text-sm font-medium">Reply-To</label><input type="email" value={replyTo} onChange={e => setReplyTo(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
          <div><label className="text-sm font-medium">Send Rate Limit (emails/min)</label><input type="number" min={1} value={rateLimit} onChange={e => setRateLimit(parseInt(e.target.value) || 100)} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">DKIM Configuration</h2>
          <label className="flex items-center gap-2"><input type="checkbox" checked={dkimEnabled} onChange={e => setDkimEnabled(e.target.checked)} className="rounded" /><span className="text-sm">Enable DKIM</span></label>
        </div>
        {dkimEnabled && (
          <div className="space-y-3">
            <div className="grid grid-cols-2 gap-4">
              <div><label className="text-sm font-medium">Domain</label><input type="text" value={dkimDomain} onChange={e => setDkimDomain(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
              <div><label className="text-sm font-medium">Selector</label><input type="text" value={dkimSelector} onChange={e => setDkimSelector(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
            </div>
            <div><label className="text-sm font-medium">Private Key (PEM)</label><textarea value={dkimKey} onChange={e => setDkimKey(e.target.value)} rows={4} placeholder="-----BEGIN RSA PRIVATE KEY-----" className="w-full border rounded px-3 py-2 text-sm mt-1 font-mono" /></div>
          </div>
        )}
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Test Email</h2>
        <div className="flex gap-3">
          <input type="email" placeholder="recipient@example.com" value={testEmail} onChange={e => setTestEmail(e.target.value)} className="flex-1 border rounded px-3 py-2 text-sm" />
          <button onClick={sendTest} disabled={testing || !testEmail} className="px-4 py-2 bg-blue-600 text-white rounded text-sm disabled:opacity-50">{testing ? 'Sending...' : 'Send Test'}</button>
        </div>
        {testResult && <div className="text-sm p-3 rounded bg-green-50 text-green-700">{testResult}</div>}
      </section>
    </div>
  );
}
'use client';
import { useState, useEffect } from 'react';

export default function OidcIdpConfigPage() {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [discoveryUrl, setDiscoveryUrl] = useState('');
  const [clientId, setClientId] = useState('');
  const [clientSecret, setClientSecret] = useState('');
  const [scopes, setScopes] = useState<string[]>([]);
  const [redirectUri, setRedirectUri] = useState('');
  const [issuerUrl, setIssuerUrl] = useState('');
  const [jwksUrl, setJwksUrl] = useState('');
  const [userinfoUrl, setUserinfoUrl] = useState('');
  const [logoutUrl, setLogoutUrl] = useState('');
  const [prompt, setPrompt] = useState('login');
  const [acrValues, setAcrValues] = useState('');
  const [testResult, setTestResult] = useState('');
  const [testing, setTesting] = useState(false);

  useEffect(() => {
    fetch("/api/v1/identity/oidc-idp-config", {
      headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
    })
      .then(res => { if (!res.ok) throw new Error(`HTTP ${res.status}`); return res.json(); })
      .then(data => {
        if (data.discoveryUrl) setDiscoveryUrl(data.discoveryUrl);
        if (data.clientId) setClientId(data.clientId);
        if (data.scopes) setScopes(data.scopes);
        if (data.redirectUri) setRedirectUri(data.redirectUri);
        if (data.issuerUrl) setIssuerUrl(data.issuerUrl);
        if (data.jwksUrl) setJwksUrl(data.jwksUrl);
        if (data.userinfoUrl) setUserinfoUrl(data.userinfoUrl);
        if (data.logoutUrl) setLogoutUrl(data.logoutUrl);
        if (data.prompt) setPrompt(data.prompt);
        if (data.acrValues) setAcrValues(data.acrValues);
        setLoading(false);
      })
      .catch(err => { setError(err.message); setLoading(false); });
  }, []);

  const allScopes = ['openid', 'profile', 'email', 'offline_access', 'groups', 'phone', 'address'];
  const toggleScope = (s: string) => setScopes(prev => prev.includes(s) ? prev.filter(x => x !== s) : [...prev, s]);

  const testDiscovery = () => {
    setTesting(true);
    fetch("/api/v1/identity/oidc-idp-config/test", {
      method: "POST",
      headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
      body: JSON.stringify({ discoveryUrl }),
    })
      .then(res => { if (!res.ok) throw new Error(`HTTP ${res.status}`); return res.json(); })
      .then(data => {
        setTestResult(data.message || 'Discovery successful');
        if (data.issuerUrl) setIssuerUrl(data.issuerUrl);
        if (data.jwksUrl) setJwksUrl(data.jwksUrl);
        if (data.userinfoUrl) setUserinfoUrl(data.userinfoUrl);
        if (data.logoutUrl) setLogoutUrl(data.logoutUrl);
        setTesting(false);
      })
      .catch(err => { setTestResult(`Error: ${err.message}`); setTesting(false); });
  };

  if (loading) return <div className="p-6"><h1 className="text-2xl font-bold">OIDC Identity Provider Configuration</h1><p className="text-gray-600 mt-2">Loading...</p></div>;
  if (error) return <div className="p-6"><h1 className="text-2xl font-bold">OIDC Identity Provider Configuration</h1><p className="text-red-600 mt-2">Error: {error}</p></div>;

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      <div><h1 className="text-2xl font-bold">OIDC Identity Provider Configuration</h1><p className="text-gray-600">Configure external OpenID Connect Identity Provider for federated authentication.</p></div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Provider Settings</h2>
        <div><label className="text-sm font-medium">Discovery URL</label><input type="url" value={discoveryUrl} onChange={e => setDiscoveryUrl(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1 font-mono" /></div>
        <div className="grid grid-cols-2 gap-4">
          <div><label className="text-sm font-medium">Client ID</label><input type="text" value={clientId} onChange={e => setClientId(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
          <div><label className="text-sm font-medium">Client Secret</label><input type="password" value={clientSecret} onChange={e => setClientSecret(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
        </div>
        <div><label className="text-sm font-medium">Redirect URI (auto-generated)</label><input type="text" readOnly value={redirectUri} className="w-full border rounded px-3 py-2 text-sm mt-1 font-mono bg-gray-50" /></div>
        <button onClick={testDiscovery} disabled={testing} className="px-4 py-2 bg-blue-600 text-white rounded text-sm disabled:opacity-50">{testing ? 'Testing...' : 'Test Discovery'}</button>
        {testResult && <div className={`text-sm p-3 rounded ${testResult.includes('Error') ? 'bg-red-50 text-red-700' : 'bg-green-50 text-green-700'}`}>{testResult}</div>}
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Scopes</h2>
        <div className="flex flex-wrap gap-3">{allScopes.map(s => (<label key={s} className="flex items-center gap-1 text-sm"><input type="checkbox" checked={scopes.includes(s)} onChange={() => toggleScope(s)} className="rounded" /><span className="font-mono">{s}</span></label>))}</div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Endpoints (auto-discovered)</h2>
        <div className="space-y-3 text-sm">
          <div><div className="text-xs text-gray-500">Issuer URL</div><input type="url" value={issuerUrl} onChange={e => setIssuerUrl(e.target.value)} className="w-full border rounded px-2 py-1 text-sm font-mono mt-1" /></div>
          <div><div className="text-xs text-gray-500">JWKS URL</div><input type="url" value={jwksUrl} onChange={e => setJwksUrl(e.target.value)} className="w-full border rounded px-2 py-1 text-sm font-mono mt-1" /></div>
          <div><div className="text-xs text-gray-500">UserInfo Endpoint</div><input type="url" value={userinfoUrl} onChange={e => setUserinfoUrl(e.target.value)} className="w-full border rounded px-2 py-1 text-sm font-mono mt-1" /></div>
          <div><div className="text-xs text-gray-500">Logout URL</div><input type="url" value={logoutUrl} onChange={e => setLogoutUrl(e.target.value)} className="w-full border rounded px-2 py-1 text-sm font-mono mt-1" /></div>
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Authentication Parameters</h2>
        <div className="grid grid-cols-2 gap-4">
          <div><label className="text-sm font-medium">Prompt</label><select value={prompt} onChange={e => setPrompt(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1"><option value="none">none (no UI)</option><option value="login">login (force re-auth)</option><option value="consent">consent (show consent)</option><option value="select_account">select_account (account picker)</option></select></div>
          <div><label className="text-sm font-medium">ACR Values</label><input type="text" value={acrValues} onChange={e => setAcrValues(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1 font-mono" /></div>
        </div>
      </section>
    </div>
  );
}

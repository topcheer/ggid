'use client';
import { useTranslations } from "@/lib/i18n";
import { useState, useEffect } from 'react';

interface FederationTrust {
  id: string;
  idpName: string;
  protocol: string;
  status: string;
  entityId: string;
  metadataUrl: string;
  lastSync: string;
}

export default function IdentityFederationPage() {
  const t = useTranslations();
  const [trusts, setTrusts] = useState<FederationTrust[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [newTrust, setNewTrust] = useState({ idpName: '', protocol: 'SAML', metadataUrl: '' });
  const [testTarget, setTestTarget] = useState<FederationTrust | null>(null);
  const [testResult, setTestResult] = useState<string>('');

  useEffect(() => {
    fetch("/api/v1/identity/federation/trusts", {
      headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
    })
      .then(res => { if (!res.ok) return null; return res.json(); })
      .then(data => { setTrusts(data.trusts || data.items || []); setLoading(false); })
      .catch(err => { setError(err.message); setLoading(false); });
  }, []);

  const statusColor = (s: string): string => s === 'active' ? 'bg-green-100 text-green-700' : 'bg-gray-200 text-gray-600';
  const protocolColor = (p: string): string => p === 'OIDC' ? 'bg-blue-100 text-blue-700' : 'bg-purple-100 text-purple-700';

  const addTrust = () => {
    const entityId = newTrust.protocol === 'SAML' ? newTrust.metadataUrl.replace('/metadata', '') : newTrust.metadataUrl.replace('/.well-known/openid-configuration', '');
    setTrusts(prev => [...prev, { id: `ft${prev.length + 1}`, idpName: newTrust.idpName, protocol: newTrust.protocol, status: 'active', entityId, metadataUrl: newTrust.metadataUrl, lastSync: new Date().toISOString().slice(0, 16).replace('T', ' ') }]);
    setShowForm(false);
    setNewTrust({ idpName: '', protocol: 'SAML', metadataUrl: '' });
  };

  const testFederation = (trust: FederationTrust) => {
    setTestTarget(trust);
    setTestResult('Testing...');
    fetch("/api/v1/identity/federation/test", {
      method: "POST",
      headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
      body: JSON.stringify({ metadataUrl: trust.metadataUrl, protocol: trust.protocol }),
    })
      .then(res => { if (!res.ok) return null; return res.json(); })
      .then(data => { setTestResult(data.message || `Connection to ${trust.idpName} successful.`); })
      .catch(err => { setTestResult(`Error: ${err.message}`); });
  };

  if (loading) return <div className="p-6"><h1 className="text-2xl font-bold">Identity Federation</h1><p className="text-gray-600 mt-2">Loading...</p></div>;
  if (error) return <div className="p-6"><h1 className="text-2xl font-bold">Identity Federation</h1><p className="text-red-600 mt-2">Error: {error}</p></div>;

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold">{t("backend.identityFederation.title")}</h1><p className="text-gray-600">Configure federation trust relationships with external Identity Providers.</p></div>
        <button onClick={() => setShowForm(!showForm)} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">{showForm ? 'Cancel' : 'Add Trust'}</button>
      </div>

      {showForm && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">{t("backend.identityFederation.addTrustRelationship")}</h2>
          <div><label className="text-sm font-medium">{t("backend.identityFederation.idpName")}</label><input aria-label="e.g. Azure AD, Okta, Auth0" type="text" placeholder="e.g. Azure AD, Okta, Auth0" value={newTrust.idpName} onChange={e => setNewTrust(prev => ({ ...prev, idpName: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
          <div><label className="text-sm font-medium">{t("backend.identityFederation.protocol")}</label><div className="flex gap-4 mt-2"><label className="flex items-center gap-2 text-sm"><input aria-label="New trust" type="radio" checked={newTrust.protocol === 'SAML'} onChange={() => setNewTrust(prev => ({ ...prev, protocol: 'SAML' }))} />SAML 2.0</label><label className="flex items-center gap-2 text-sm"><input type="radio" checked={newTrust.protocol === 'OIDC'} onChange={() => setNewTrust(prev => ({ ...prev, protocol: 'OIDC' }))} />{t("backend.identityFederation.openIdConnect")}</label></div></div>
          <div><label className="text-sm font-medium">{t("backend.identityFederation.metadataUrl")}</label><input aria-label="new Trust" type="text" placeholder={newTrust.protocol === 'SAML' ? 'https://idp.example.com/metadata' : 'https://idp.example.com/.well-known/openid-configuration'} value={newTrust.metadataUrl} onChange={e => setNewTrust(prev => ({ ...prev, metadataUrl: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1 font-mono" /></div>
          <button onClick={addTrust} disabled={!newTrust.idpName || !newTrust.metadataUrl} className="px-4 py-2 bg-blue-600 text-white rounded text-sm disabled:opacity-50">Import & Add Trust</button>
        </section>
      )}

      <div className="grid grid-cols-3 gap-4">
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-2xl font-bold">{trusts.length}</div><div className="text-sm text-gray-500">Trust Relationships</div></div>
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-2xl font-bold text-green-600">{trusts.filter(t => t.status === 'active').length}</div><div className="text-sm text-gray-500">{t("backend.identityFederation.active")}</div></div>
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-2xl font-bold">{new Set(trusts.map(t => t.protocol)).size}</div><div className="text-sm text-gray-500">Protocols</div></div>
      </div>

      <section className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50"><tr className="text-left"><th className="p-3">{t("backend.identityFederation.idpName")}</th><th className="p-3">{t("backend.identityFederation.protocol")}</th><th className="p-3">{t("backend.identityFederation.entityId")}</th><th className="p-3">Status</th><th className="p-3">{t("backend.identityFederation.lastSync")}</th><th className="p-3">{t("backend.identityFederation.action")}</th></tr></thead>
          <tbody>
            {trusts.length === 0 ? <tr><td colSpan={6} className="p-6 text-center text-gray-500">No federation trusts configured.</td></tr> :
            trusts.map(t => (
              <tr key={t.id} className="border-b hover:bg-gray-50">
                <td className="p-3 font-medium">{t.idpName}</td>
                <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs ${protocolColor(t.protocol)}`}>{t.protocol}</span></td>
                <td className="p-3 font-mono text-xs text-gray-500 truncate max-w-xs">{t.entityId}</td>
                <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs ${statusColor(t.status)}`}>{t.status}</span></td>
                <td className="p-3 text-gray-500 text-xs">{t.lastSync}</td>
                <td className="p-3"><button onClick={() => testFederation(t)} className="text-blue-600 text-xs hover:underline">Test</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      {testTarget && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 max-w-md w-full mx-4 space-y-4">
            <div className="flex items-center justify-between"><h2 className="text-lg font-semibold">Federation Test: {testTarget.idpName}</h2><button onClick={() => { setTestTarget(null); setTestResult(''); }} aria-label="Close" className="text-gray-400 hover:text-gray-600">X</button></div>
            <div className="space-y-2 text-sm"><div><span className="text-gray-500">Protocol:</span> {testTarget.protocol}</div><div><span className="text-gray-500">Metadata URL:</span> <span className="font-mono text-xs">{testTarget.metadataUrl}</span></div><div><span className="text-gray-500">Entity ID:</span> <span className="font-mono text-xs">{testTarget.entityId}</span></div></div>
            <div className={`p-3 rounded text-sm ${testResult.includes('successful') || testResult.includes('Error') ? testResult.includes('Error') ? 'bg-red-50 text-red-700' : 'bg-green-50 text-green-700' : 'bg-amber-50 text-amber-700'}`}>{testResult}</div>
          </div>
        </div>
      )}
    </div>
  );
}

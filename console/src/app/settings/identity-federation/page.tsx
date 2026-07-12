'use client';
import { useState } from 'react';

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
  const [trusts, setTrusts] = useState<FederationTrust[]>([
    { id: 'ft1', idpName: 'Azure AD', protocol: 'OIDC', status: 'active', entityId: 'https://login.microsoftonline.com/tenant/v2.0', metadataUrl: 'https://login.microsoftonline.com/tenant/v2.0/.well-known/openid-configuration', lastSync: '2026-07-12 10:30' },
    { id: 'ft2', idpName: 'Okta', protocol: 'SAML', status: 'active', entityId: 'http://www.okta.com/exk123abc', metadataUrl: 'https://company.okta.com/app/exk123abc/sso/saml/metadata', lastSync: '2026-07-11 14:00' },
    { id: 'ft3', idpName: 'Google Workspace', protocol: 'OIDC', status: 'active', entityId: 'https://accounts.google.com', metadataUrl: 'https://accounts.google.com/.well-known/openid-configuration', lastSync: '2026-07-10 09:15' },
    { id: 'ft4', idpName: 'Legacy ADFS', protocol: 'SAML', status: 'inactive', entityId: 'https://adfs.company.com/federationmetadata/2007-06/federationmetadata.xml', metadataUrl: 'https://adfs.company.com/federationmetadata/2007-06/federationmetadata.xml', lastSync: '2026-01-15 08:00' },
  ]);

  const [showForm, setShowForm] = useState(false);
  const [newTrust, setNewTrust] = useState({ idpName: '', protocol: 'SAML', metadataUrl: '' });
  const [testTarget, setTestTarget] = useState<FederationTrust | null>(null);
  const [testResult, setTestResult] = useState<string>('');

  const statusColor = (s: string): string =>
    s === 'active' ? 'bg-green-100 text-green-700' : 'bg-gray-200 text-gray-600';

  const protocolColor = (p: string): string =>
    p === 'OIDC' ? 'bg-blue-100 text-blue-700' : 'bg-purple-100 text-purple-700';

  const addTrust = () => {
    const entityId = newTrust.protocol === 'SAML'
      ? newTrust.metadataUrl.replace('/metadata', '')
      : newTrust.metadataUrl.replace('/.well-known/openid-configuration', '');
    setTrusts(prev => [...prev, {
      id: `ft${prev.length + 1}`,
      idpName: newTrust.idpName,
      protocol: newTrust.protocol,
      status: 'active',
      entityId,
      metadataUrl: newTrust.metadataUrl,
      lastSync: new Date().toISOString().slice(0, 16).replace('T', ' '),
    }]);
    setShowForm(false);
    setNewTrust({ idpName: '', protocol: 'SAML', metadataUrl: '' });
  };

  const testFederation = (trust: FederationTrust) => {
    setTestTarget(trust);
    setTestResult('Testing...');
    setTimeout(() => {
      setTestResult(`Connection to ${trust.idpName} successful. Metadata fetched. Entity ID verified. Trust chain valid.`);
    }, 1000);
  };

  const trustChain = [
    { level: 1, entity: 'GGID SP', type: 'Service Provider' },
    { level: 2, entity: 'Azure AD', type: 'Identity Provider' },
    { level: 3, entity: 'Microsoft Online', type: 'Root CA' },
  ];

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Identity Federation</h1>
          <p className="text-gray-600">Configure federation trust relationships with external Identity Providers.</p>
        </div>
        <button onClick={() => setShowForm(!showForm)} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">
          {showForm ? 'Cancel' : 'Add Trust'}
        </button>
      </div>

      {showForm && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Add Trust Relationship</h2>
          <div>
            <label className="text-sm font-medium">IdP Name</label>
            <input type="text" placeholder="e.g. Azure AD, Okta, Auth0" value={newTrust.idpName} onChange={e => setNewTrust(prev => ({ ...prev, idpName: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" />
          </div>
          <div>
            <label className="text-sm font-medium">Protocol</label>
            <div className="flex gap-4 mt-2">
              <label className="flex items-center gap-2 text-sm">
                <input type="radio" checked={newTrust.protocol === 'SAML'} onChange={() => setNewTrust(prev => ({ ...prev, protocol: 'SAML' }))} />
                SAML 2.0
              </label>
              <label className="flex items-center gap-2 text-sm">
                <input type="radio" checked={newTrust.protocol === 'OIDC'} onChange={() => setNewTrust(prev => ({ ...prev, protocol: 'OIDC' }))} />
                OpenID Connect
              </label>
            </div>
          </div>
          <div>
            <label className="text-sm font-medium">Metadata URL</label>
            <input type="text" placeholder={newTrust.protocol === 'SAML' ? 'https://idp.example.com/metadata' : 'https://idp.example.com/.well-known/openid-configuration'} value={newTrust.metadataUrl} onChange={e => setNewTrust(prev => ({ ...prev, metadataUrl: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1 font-mono" />
          </div>
          <button onClick={addTrust} disabled={!newTrust.idpName || !newTrust.metadataUrl} className="px-4 py-2 bg-blue-600 text-white rounded text-sm disabled:opacity-50">Import & Add Trust</button>
        </section>
      )}

      <div className="grid grid-cols-3 gap-4">
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold">{trusts.length}</div>
          <div className="text-sm text-gray-500">Trust Relationships</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold text-green-600">{trusts.filter(t => t.status === 'active').length}</div>
          <div className="text-sm text-gray-500">Active</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold">{new Set(trusts.map(t => t.protocol)).size}</div>
          <div className="text-sm text-gray-500">Protocols</div>
        </div>
      </div>

      <section className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th className="p-3">IdP Name</th>
              <th className="p-3">Protocol</th>
              <th className="p-3">Entity ID</th>
              <th className="p-3">Status</th>
              <th className="p-3">Last Sync</th>
              <th className="p-3">Action</th>
            </tr>
          </thead>
          <tbody>
            {trusts.map(t => (
              <tr key={t.id} className="border-b hover:bg-gray-50">
                <td className="p-3 font-medium">{t.idpName}</td>
                <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs ${protocolColor(t.protocol)}`}>{t.protocol}</span></td>
                <td className="p-3 font-mono text-xs text-gray-500 truncate max-w-xs">{t.entityId}</td>
                <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs ${statusColor(t.status)}`}>{t.status}</span></td>
                <td className="p-3 text-gray-500 text-xs">{t.lastSync}</td>
                <td className="p-3">
                  <button onClick={() => testFederation(t)} className="text-blue-600 text-xs hover:underline">Test</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Trust Chain Visualization</h2>
        <div className="space-y-3">
          {trustChain.map((node, idx) => (
            <div key={idx} className="flex items-center gap-4">
              <div className="w-8 h-8 rounded-full bg-blue-600 text-white flex items-center justify-center text-xs font-bold">{node.level}</div>
              <div className="flex-1">
                <div className="text-sm font-medium">{node.entity}</div>
                <div className="text-xs text-gray-500">{node.type}</div>
              </div>
              {idx < trustChain.length - 1 && <div className="text-gray-300 text-2xl">|</div>}
            </div>
          ))}
        </div>
      </section>

      {testTarget && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 max-w-md w-full mx-4 space-y-4">
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-semibold">Federation Test: {testTarget.idpName}</h2>
              <button onClick={() => { setTestTarget(null); setTestResult(''); }} className="text-gray-400 hover:text-gray-600">X</button>
            </div>
            <div className="space-y-2 text-sm">
              <div><span className="text-gray-500">Protocol:</span> {testTarget.protocol}</div>
              <div><span className="text-gray-500">Metadata URL:</span> <span className="font-mono text-xs">{testTarget.metadataUrl}</span></div>
              <div><span className="text-gray-500">Entity ID:</span> <span className="font-mono text-xs">{testTarget.entityId}</span></div>
            </div>
            <div className={`p-3 rounded text-sm ${testResult.includes('successful') ? 'bg-green-50 text-green-700' : 'bg-amber-50 text-amber-700'}`}>
              {testResult}
            </div>
            {testResult.includes('successful') && (
              <button onClick={() => { setTestTarget(null); setTestResult(''); }} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">Close</button>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
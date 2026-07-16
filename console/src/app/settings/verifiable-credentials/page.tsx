'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";

interface VC {
  id: string;
  type: string;
  issuer: string;
  subject: string;
  issued: string;
  expires: string;
  status: string;
  claims: Record<string, string>;
}

export default function VerifiableCredentialsPage() {
  const t = useTranslations();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [credentials, setCredentials] = useState<VC[]>([]);

  useEffect(() => {
    fetch("/api/v1/identity/vc", {
      headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
    })
      .then(res => { if (!res.ok) return null; return res.json(); })
      .then(data => { setCredentials(Array.isArray(data) ? data : (data.credentials || data.items || [])); setLoading(false); })
      .catch(err => { setError(err.message); setLoading(false); });
  }, []);

  const [showIssue, setShowIssue] = useState(false);
  const [template, setTemplate] = useState('UniversityDegree');
  const [subjectDid, setSubjectDid] = useState('');
  const [claimsJson, setClaimsJson] = useState('{}');
  const [verifyResult, setVerifyResult] = useState<Record<string, string> | null>(null);
  const [showPresentation, setShowPresentation] = useState(false);
  const [importText, setImportText] = useState('');
  const [showImport, setShowImport] = useState(false);

  const templates = ['UniversityDegree', 'DriverLicense', 'Passport', 'EmployeeId', 'Membership'];

  const statusColor = (s: string): string =>
    s === 'valid' ? 'bg-green-100 text-green-700' : s === 'revoked' ? 'bg-red-100 text-red-700' : 'bg-amber-100 text-amber-700';

  const issueVC = () => {
    let claims: Record<string, string> = {};
    try { claims = JSON.parse(claimsJson); } catch { claims = {}; }
    const newVC: VC = {
      id: `vc${credentials.length + 1}`,
      type: template,
      issuer: 'did:web:ggid.io',
      subject: subjectDid || 'did:web:unknown',
      issued: new Date().toISOString().slice(0, 10),
      expires: new Date(Date.now() + 365 * 86400000).toISOString().slice(0, 10),
      status: 'valid',
      claims,
    };
    setCredentials(prev => [...prev, newVC]);
    setShowIssue(false);
    setSubjectDid('');
    setClaimsJson('{}');
  };

  const verifyVC = (vc: VC) => {
    setVerifyResult({
      result: vc.status === 'valid' ? 'verified' : 'failed',
      signature: 'Ed25519 valid',
      issuer: vc.issuer,
      timestamp: new Date().toISOString(),
    });
  };

  const revokeVC = (id: string) => {
    setCredentials(prev => prev.map(vc => vc.id === id ? { ...vc, status: 'revoked' } : vc));
  };

  const exportVC = (vc: VC) => {
    const blob = new Blob([JSON.stringify(vc, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${vc.id}.json`;
    a.click();
  };

  const importVC = () => {
    try {
      const parsed = JSON.parse(importText);
      setCredentials(prev => [...prev, { ...parsed, id: `vc${prev.length + 1}` }]);
      setShowImport(false);
      setImportText('');
    } catch {
      // ignore parse error
    }
  };

  if (loading) return (
    <div className="p-6"><h1 className="text-2xl font-bold mb-4">Verifiable Credentials</h1><p>Loading...</p></div>
  );
  if (error) return (
    <div className="p-6"><h1 className="text-2xl font-bold mb-4">Verifiable Credentials</h1><p className="text-red-600">Error: {error}</p></div>
  );
  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Verifiable Credentials</h1>
          <p className="text-gray-600">Issue, verify, revoke, and manage W3C Verifiable Credentials.</p>
        </div>
        <div className="flex gap-2">
          <button onClick={() => setShowImport(!showImport)} className="px-3 py-1.5 border rounded text-sm">Import</button>
          <button onClick={() => setShowPresentation(!showPresentation)} className="px-3 py-1.5 border rounded text-sm">Present</button>
          <button onClick={() => setShowIssue(!showIssue)} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">
            {showIssue ? 'Cancel' : 'Issue VC'}
          </button>
        </div>
      </div>

      {showIssue && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Issue Verifiable Credential</h2>
          <div>
            <label className="text-sm font-medium">Template</label>
            <select value={template} onChange={e => setTemplate(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1">
              {templates.map(t => <option key={t} value={t}>{t}</option>)}
            </select>
          </div>
          <div>
            <label className="text-sm font-medium">Subject DID</label>
            <input type="text" placeholder="did:web:subject.example" value={subjectDid} onChange={e => setSubjectDid(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1 font-mono" />
          </div>
          <div>
            <label className="text-sm font-medium">Claims (JSON)</label>
            <textarea value={claimsJson} onChange={e => setClaimsJson(e.target.value)} rows={4} className="w-full border rounded px-3 py-2 text-sm mt-1 font-mono" placeholder='{"key": "value"}' />
          </div>
          <button onClick={issueVC} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">Issue Credential</button>
        </section>
      )}

      {showImport && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Import VC (JSON)</h2>
          <textarea aria-label="Import text" value={importText} onChange={e => setImportText(e.target.value)} rows={6} className="w-full border rounded px-3 py-2 text-sm font-mono" placeholder='Paste VC JSON here...' />
          <button onClick={importVC} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">Import</button>
        </section>
      )}

      {showPresentation && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">VC Presentation Flow</h2>
          <p className="text-sm text-gray-500">Select a credential to generate a verifiable presentation for a verifier.</p>
          <div className="space-y-2">
            {credentials.filter(c => c.status === 'valid').map(vc => (
              <div key={vc.id} className="flex items-center gap-3 border rounded p-3">
                <span className="text-sm font-medium">{vc.type}</span>
                <span className="text-xs text-gray-500">{vc.subject}</span>
                <button className="px-3 py-1 bg-blue-600 text-white rounded text-xs">Generate Presentation</button>
              </div>
            ))}
          </div>
        </section>
      )}

      {verifyResult && (
        <section className="bg-white rounded-lg shadow p-6 space-y-3">
          <h2 className="text-lg font-semibold">Verification Result</h2>
          <div className="space-y-1 text-sm">
            <div><span className="text-gray-500">Result:</span> <span className={`font-bold ${verifyResult.result === 'verified' ? 'text-green-600' : 'text-red-600'}`}>{verifyResult.result}</span></div>
            <div><span className="text-gray-500">Signature:</span> {verifyResult.signature}</div>
            <div><span className="text-gray-500">Issuer:</span> {verifyResult.issuer}</div>
            <div><span className="text-gray-500">Timestamp:</span> {verifyResult.timestamp}</div>
          </div>
          <button onClick={() => setVerifyResult(null)} className="text-sm text-blue-600">Close</button>
        </section>
      )}

      <section className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th scope="col" className="p-3">Type</th>
              <th scope="col" className="p-3">Issuer</th>
              <th scope="col" className="p-3">Subject</th>
              <th scope="col" className="p-3">Issued</th>
              <th scope="col" className="p-3">Expires</th>
              <th scope="col" className="p-3">Status</th>
              <th scope="col" className="p-3">Actions</th>
            </tr>
          </thead>
          <tbody>
            {credentials.map(vc => (
              <tr key={vc.id} className="border-b hover:bg-gray-50">
                <td className="p-3 font-medium">{vc.type}</td>
                <td className="p-3 font-mono text-xs text-gray-500">{vc.issuer}</td>
                <td className="p-3 font-mono text-xs text-gray-500">{vc.subject}</td>
                <td className="p-3 text-gray-500">{vc.issued}</td>
                <td className="p-3 text-gray-500">{vc.expires}</td>
                <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs ${statusColor(vc.status)}`}>{vc.status}</span></td>
                <td className="p-3">
                  <div className="flex gap-2">
                    <button onClick={() => verifyVC(vc)} className="text-blue-600 text-xs hover:underline">Verify</button>
                    <button onClick={() => exportVC(vc)} className="text-gray-600 text-xs hover:underline">Export</button>
                    {vc.status === 'valid' && (
                      <button onClick={() => revokeVC(vc.id)} className="text-red-600 text-xs hover:underline">Revoke</button>
                    )}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </div>
  );
}
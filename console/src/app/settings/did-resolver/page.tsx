'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";

interface ServiceEndpoint {
  id: string;
  type: string;
  serviceEndpoint: string;
}

interface DidDocument {
  id: string;
  method: string;
  verificationStatus: string;
  serviceEndpoints: ServiceEndpoint[];
  linkedVCs: number;
  raw: string;
}

export default function DidResolverPage() {
  const t = useTranslations();

  const [didInput, setDidInput] = useState('');
  const [method, setMethod] = useState('did:web');
  const [result, setResult] = useState<DidDocument | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const methods = ['did:web', 'did:ion', 'did:key', 'did:ebsi'];

  const resolve = () => {
    if (!didInput.trim()) {
      setError('Please enter a DID');
      return;
    }
    setLoading(true);
    setError('');
    fetch(`/api/v1/identity/did?id=${encodeURIComponent(didInput)}`, {
      headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
    })
      .then(res => { if (!res.ok) return null; return res.json(); })
      .then(data => { setResult(data); setLoading(false); })
      .catch(err => { setError(err.message); setLoading(false); });
  };

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{t("big1.didResolver.title")}</h1>
        <p className="text-gray-600">{t("big1.didResolver.resolveDecentralizedIdentifiersAndViewDIDDocuments")}</p>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <div className="flex gap-3">
          <select aria-label="Method" value={method} onChange={e => setMethod(e.target.value)} className="border rounded px-3 py-2 text-sm">
            {methods.map(m => <option key={m} value={m}>{m}</option>)}
          </select>
          <input
            type="text"
            placeholder={`${method}:example.com:user`}
            value={didInput}
            onChange={e => setDidInput(e.target.value)}
            className="flex-1 border rounded px-3 py-2 text-sm font-mono"
          />
          <button onClick={resolve} disabled={loading} aria-label="Resolve DID" className="px-4 py-2 bg-blue-600 text-white rounded text-sm disabled:opacity-50">
            {loading ? t("big1.didResolver.resolving") : t("big1.didResolver.resolve")}
          </button>
        </div>
        {error && <p className="text-sm text-red-600">{error}</p>}
      </section>

      {result && (
        <>
          <section className="bg-white rounded-lg shadow p-6 space-y-4">
            <h2 className="text-lg font-semibold">{t("big1.didResolver.resolutionResult")}</h2>
            <div className="flex items-center gap-4">
              <div>
                <div className="text-xs text-gray-500">{t("big1.didResolver.did")}</div>
                <div className="font-mono text-sm">{result.id}</div>
              </div>
              <div>
                <div className="text-xs text-gray-500">{t("big1.didResolver.method")}</div>
                <div className="text-sm font-medium">{result.method}</div>
              </div>
              <div>
                <div className="text-xs text-gray-500">{t("big1.didResolver.verification")}</div>
                <span className={`px-2 py-0.5 rounded text-xs ${result.verificationStatus === 'verified' ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'}`}>{result.verificationStatus}</span>
              </div>
              <div>
                <div className="text-xs text-gray-500">{t("big1.didResolver.linkedVcs")}</div>
                <div className="text-sm font-bold">{result.linkedVCs}</div>
              </div>
            </div>
          </section>

          <section className="bg-white rounded-lg shadow p-6 space-y-4">
            <h2 className="text-lg font-semibold">{t("big1.didResolver.serviceEndpoints")}</h2>
            <div className="space-y-2">
              {result.serviceEndpoints.map(ep => (
                <div key={ep.id} className="flex items-center gap-3 border-b pb-2">
                  <span className="px-2 py-0.5 bg-blue-100 text-blue-700 rounded text-xs">{ep.type}</span>
                  <span className="font-mono text-xs text-gray-500">{ep.id}</span>
                  <span className="text-sm text-blue-600">{ep.serviceEndpoint}</span>
                </div>
              ))}
            </div>
          </section>

          <section className="bg-white rounded-lg shadow p-6 space-y-4">
            <h2 className="text-lg font-semibold">{t("big1.didResolver.didDocument")}</h2>
            <pre className="bg-gray-900 text-green-400 rounded p-4 text-xs overflow-x-auto max-h-96">{result.raw}</pre>
          </section>
        </>
      )}
    </div>
  );
}
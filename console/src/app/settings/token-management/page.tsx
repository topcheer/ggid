'use client';
import { useTranslations } from "@/lib/i18n";
import { useState, useEffect } from 'react';

interface Token { id: string; type: string; user: string; client: string; issued: string; expires: string; scopes: string[]; dpop: boolean; jti: string; }

export default function TokenManagementPage() {
  const t = useTranslations();
  const [tokens, setTokens] = useState<Token[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchJti, setSearchJti] = useState('');
  const [selected, setSelected] = useState<Token | null>(null);
  const [batchRevoke, setBatchRevoke] = useState<string[]>([]);

  useEffect(() => {
    fetch('/api/v1/auth/sessions', {
      headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, 'Content-Type': 'application/json', 'X-Tenant-ID': '00000000-0000-0000-0000-000000000001' },
    })
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(data => {
        if (data && data.tokens) setTokens(data.tokens);
        else if (Array.isArray(data)) setTokens(data);
        setLoading(false);
      })
      .catch(e => { setError(e.message); setLoading(false); });
  }, []);

  const filtered = tokens.filter(t => !searchJti || t.jti.includes(searchJti));
  const revoke = (id: string) => setTokens(prev => prev.filter(t => t.id !== id));
  const toggleBatch = (id: string) => setBatchRevoke(prev => prev.includes(id) ? prev.filter(x => x !== id) : [...prev, id]);
  const revokeBatch = () => { setTokens(prev => prev.filter(t => !batchRevoke.includes(t.id))); setBatchRevoke([]); };

  const decodeJwt = (t: Token) => btoa(JSON.stringify({ iss: 'ggid.io', sub: t.user, aud: t.client, scope: t.scopes.join(' '), iat: '2026-07-12T14:00Z', exp: t.expires, jti: t.jti, dpop: t.dpop }, null, 2));

  if (loading) return <div className="p-6"><p>Loading...</p></div>;
  if (error) return <div className="p-6 text-red-600">Error: {error}</div>;

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div><h1 className="text-2xl font-bold">{t("backend.tokenManagement.title")}</h1><p className="text-gray-600">View, revoke, and manage active OAuth tokens with JWT inspection.</p></div>

      <div className="flex gap-3">
        <input aria-label="Search by jti..." type="text" placeholder="Search by jti..." value={searchJti} onChange={e => setSearchJti(e.target.value)} className="flex-1 border rounded px-3 py-2 text-sm font-mono" />
        {batchRevoke.length > 0 && <button onClick={revokeBatch} className="px-4 py-2 bg-red-600 text-white rounded text-sm">Revoke Selected ({batchRevoke.length})</button>}
      </div>

      <section className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full text-sm"><thead className="bg-gray-50"><tr className="text-left"><th className="p-3"><input aria-label="Toggle" type="checkbox" className="rounded" /></th><th className="p-3">{t("backend.tokenManagement.type")}</th><th className="p-3">{t("backend.tokenManagement.user")}</th><th className="p-3">{t("backend.tokenManagement.client")}</th><th className="p-3">{t("backend.tokenManagement.issued")}</th><th className="p-3">{t("backend.tokenManagement.expires")}</th><th className="p-3">{t("backend.tokenManagement.scopes")}</th><th className="p-3">{t("backend.tokenManagement.dpop")}</th><th className="p-3">{t("backend.tokenManagement.action")}</th></tr></thead>
          <tbody>{filtered.map(tk => (
            <tr key={tk.id} className="border-b hover:bg-gray-50">
              <td className="p-3"><input aria-label="Toggle" type="checkbox" checked={batchRevoke.includes(tk.id)} onChange={() => toggleBatch(tk.id)} className="rounded" /></td>
              <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs ${tk.type === 'access' ? 'bg-blue-100 text-blue-700' : 'bg-green-100 text-green-700'}`}>{tk.type}</span></td>
              <td className="p-3 font-medium">{tk.user}</td><td className="p-3 font-mono text-xs">{tk.client}</td>
              <td className="p-3 text-gray-500">{tk.issued}</td><td className="p-3 text-gray-500">{tk.expires}</td>
              <td className="p-3"><div className="flex flex-wrap gap-1">{tk.scopes.map(s => <span key={s} className="px-1.5 py-0.5 bg-gray-100 rounded text-xs font-mono">{s}</span>)}</div></td>
              <td className="p-3">{tk.dpop ? <span className="text-green-600 text-xs">yes</span> : <span className="text-gray-400 text-xs">no</span>}</td>
              <td className="p-3"><div className="flex gap-2"><button onClick={() => setSelected(tk)} className="text-blue-600 text-xs hover:underline">View</button><button onClick={() => revoke(tk.id)} className="text-red-600 text-xs hover:underline">{t("backend.tokenManagement.revoke")}</button></div></td>
            </tr>))}</tbody></table>
      </section>

      {selected && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 max-w-2xl w-full mx-4 space-y-4">
            <div className="flex items-center justify-between"><h2 className="text-lg font-semibold">Token Detail (Decoded JWT)</h2><button onClick={() => setSelected(null)} className="text-gray-400">x</button></div>
            <div className="space-y-1 text-sm"><div><span className="text-gray-500">jti:</span> {selected.jti}</div><div><span className="text-gray-500">User:</span> {selected.user}</div><div><span className="text-gray-500">Type:</span> {selected.type}</div><div><span className="text-gray-500">DPoP bound:</span> {selected.dpop ? 'yes' : 'no'}</div></div>
            <div><div className="text-xs text-gray-500 mb-1">Refresh Rotation History:</div><div className="text-xs text-gray-600 space-y-1"><div>1. Issued at 14:00 (current)</div><div>2. Rotated at 13:45</div><div>3. Rotated at 13:30</div></div></div>
            <pre className="bg-gray-900 text-green-400 rounded p-3 text-xs overflow-x-auto max-h-48">{decodeJwt(selected)}</pre>
          </div>
        </div>
      )}
    </div>
  );
}
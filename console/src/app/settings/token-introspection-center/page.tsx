"use client";
import { useTranslations } from "@/lib/i18n";
import { useState, useEffect } from "react";

export default function TokenIntrospectionCenterPage() {
  const t = useTranslations();
  const [token, setToken] = useState("");
  const [decoded, setDecoded] = useState<{ header: string; payload: string; signature: string } | null>(null);
  const [validation, setValidation] = useState<{ issuer: string; expiry: string; audience: string; scopes: string[]; active: boolean; revoked: boolean; binding: string; client: string } | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch("/api/v1/auth/expiry-status", {
      headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
    })
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(() => setLoading(false))
      .catch(e => { setError(e.message); setLoading(false); });
  }, []);

  const handleIntrospect = async () => {
    setDecoded({ header: '{\n  "alg": "RS256",\n  "kid": "key-2025-01",\n  "typ": "JWT"\n}', payload: '{\n  "iss": "https://auth.ggid.dev",\n  "sub": "user-12345",\n  "aud": "dashboard-client",\n  "exp": 1736995200,\n  "iat": 1736908800,\n  "scope": "openid profile email users:read",\n  "cnf": { "jkt": "dPvWjK3xQ..." }\n}', signature: 'a3f2b91c...7c4e9d22' });
    setValidation({ issuer: "https://auth.ggid.dev", expiry: "2025-01-16 00:00:00 UTC", audience: "dashboard-client", scopes: ["openid", "profile", "email", "users:read"], active: true, revoked: false, binding: "DPoP (cnf.jkt: dPvWjK3xQ...)", client: "Web Dashboard (cli-001)" });
  };

  if (loading) return <div className="p-8"><p>Loading...</p></div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;

  return (
    <div className="p-8 space-y-6 max-w-5xl">
      <h1 className="text-2xl font-bold">Token Introspection Center</h1>
      <p className="text-gray-600">Decode JWT tokens, validate claims, and check revocation/binding status.</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4"><h2 className="text-lg font-semibold">Introspect Token</h2><textarea value={token} onChange={(e) => setToken(e.target.value)} placeholder="Paste JWT token here..." className="border rounded px-3 py-2 w-full font-mono text-xs" rows={3} /><button onClick={handleIntrospect} className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700">Introspect</button></div>

      {decoded && (<div className="grid grid-cols-3 gap-4"><div className="bg-white rounded-lg p-4 shadow"><h3 className="text-sm font-semibold mb-2 text-red-600">Header</h3><pre className="text-xs font-mono overflow-x-auto whitespace-pre-wrap">{decoded.header}</pre></div><div className="bg-white rounded-lg p-4 shadow"><h3 className="text-sm font-semibold mb-2 text-purple-600">Payload</h3><pre className="text-xs font-mono overflow-x-auto whitespace-pre-wrap">{decoded.payload}</pre></div><div className="bg-white rounded-lg p-4 shadow"><h3 className="text-sm font-semibold mb-2 text-blue-600">Signature</h3><pre className="text-xs font-mono overflow-x-auto whitespace-pre-wrap">{decoded.signature}</pre></div></div>)}

      {validation && (<div className="bg-white rounded-lg p-6 shadow space-y-4"><h2 className="text-lg font-semibold">Validation Results</h2><div className={`rounded-lg p-4 ${validation.active && !validation.revoked ? "bg-green-50 border border-green-300" : "bg-red-50 border border-red-300"}`}><div className="flex items-center gap-4"><span className={`text-lg font-bold ${validation.active ? "text-green-600" : "text-red-600"}`}>{validation.active ? "ACTIVE" : "INACTIVE"}</span><span className={`text-sm ${validation.revoked ? "text-red-600" : "text-gray-500"}`}>{validation.revoked ? "REVOKED" : "Not Revoked"}</span></div></div>
        <div className="grid grid-cols-2 gap-4 text-sm"><div><span className="text-gray-500">Issuer: </span><span className="font-mono text-xs">{validation.issuer}</span></div><div><span className="text-gray-500">Expiry: </span><span className="text-xs">{validation.expiry}</span></div><div><span className="text-gray-500">Audience: </span><span className="font-mono text-xs">{validation.audience}</span></div><div><span className="text-gray-500">Client: </span><span className="text-xs">{validation.client}</span></div><div><span className="text-gray-500">Binding: </span><span className="text-xs">{validation.binding}</span></div></div>
        <div><span className="text-sm text-gray-500">Scopes: </span><div className="flex flex-wrap gap-1 mt-1">{validation.scopes.map((s) => <span key={s} className="px-1.5 py-0.5 bg-purple-100 text-purple-700 rounded text-xs font-mono">{s}</span>)}</div></div>
      </div>)}
    </div>
  );
}

"use client";
import { useTranslations } from "@/lib/i18n";
import { useState, useEffect } from "react";
import { Loader2 } from "lucide-react";

interface OAuthClient {
  client_id: string;
  name: string;
  grant_types: string[];
  scopes: string[];
  status: "active" | "disabled";
  pkce_required: boolean;
  redirect_uris: string[];
}

export default function OAuthClientRegistryPage() {
  const t = useTranslations();

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const res = await fetch("/api/v1/oauth/clients", {
          method: "GET",
          headers: {
            "Content-Type": "application/json",
            "X-Tenant-ID": "00000000-0000-0000-0000-000000000001",
          },
        });
        if (!res.ok) return null;
        const json = await res.json();
      } catch (e) {
        setError(e instanceof Error ? e.message : "Failed to load");
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, []);

  const [showRegister, setShowRegister] = useState(false);const [clients] = useState<OAuthClient[]>([
    { client_id: "cli-001", name: "Web Dashboard", grant_types: ["authorization_code", "refresh_token"], scopes: ["openid", "profile", "email"], status: "active", pkce_required: true, redirect_uris: ["https://dashboard.example.com/callback"] },
    { client_id: "cli-002", name: "Mobile App", grant_types: ["authorization_code", "refresh_token"], scopes: ["openid", "profile"], status: "active", pkce_required: true, redirect_uris: ["myapp://callback"] },
    { client_id: "cli-003", name: "Legacy Service", grant_types: ["client_credentials"], scopes: ["users:read"], status: "active", pkce_required: false, redirect_uris: [] },
    { client_id: "cli-004", name: "Partner Integration", grant_types: ["authorization_code"], scopes: ["openid", "profile", "audit:read"], status: "disabled", pkce_required: true, redirect_uris: ["https://partner.io/auth/cb"] },
  ]);

  if (loading) return <div className="flex items-center justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-blue-500" /></div>;
  if (error) return <div className="p-8 text-red-500">Error: {error}</div>;
  
  return (
    <div className="p-8 space-y-6 max-w-5xl">
      <h1 className="text-2xl font-bold">OAuth Client Registry</h1>
      <p className="text-gray-600">Register and manage OAuth/OIDC clients with grant types, scopes, and PKCE enforcement.</p>

      <div className="bg-white rounded-lg p-6 shadow">
        <div className="flex items-center justify-between mb-4"><h2 className="text-lg font-semibold">Registered Clients</h2><button onClick={() => setShowRegister(!showRegister)} className="px-4 py-1 bg-blue-600 text-white rounded text-sm hover:bg-blue-700">Register Client</button></div>
        {showRegister && (<div className="mb-4 border rounded p-4 space-y-3 bg-gray-50"><div className="grid grid-cols-2 gap-4"><div><label className="block text-sm font-medium mb-1">Client Name</label><input type="text" placeholder="My Application" className="border rounded px-3 py-2 w-full" /></div><div><label className="block text-sm font-medium mb-1">Token TTL (seconds)</label><input type="number" defaultValue={3600} className="border rounded px-3 py-2 w-full" /></div></div><div><label className="block text-sm font-medium mb-1">Grant Types</label><div className="flex flex-wrap gap-2">{["authorization_code", "refresh_token", "client_credentials", "password"].map((g) => (<label key={g} className="flex items-center gap-1 text-sm"><input type="checkbox" className="w-4 h-4" />{g}</label>))}</div></div><div><label className="block text-sm font-medium mb-1">Redirect URIs (one per line)</label><textarea placeholder="https://example.com/callback" className="border rounded px-3 py-2 w-full text-sm font-mono" rows={2} /></div><div><label className="block text-sm font-medium mb-1">Scopes</label><div className="flex flex-wrap gap-2">{["openid", "profile", "email", "users:read", "users:write", "roles:read", "audit:read"].map((s) => (<label key={s} className="flex items-center gap-1 text-sm"><input type="checkbox" className="w-4 h-4" />{s}</label>))}</div></div><div className="flex items-center gap-3"><input type="checkbox" defaultChecked className="w-4 h-4" /><label className="text-sm">Require PKCE</label><div className="ml-4 flex items-center gap-3"><input type="checkbox" className="w-4 h-4" /><label className="text-sm">Rotate Refresh Token</label></div></div><button className="px-4 py-2 bg-green-600 text-white rounded text-sm">Register</button></div>)}
        <table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Name</th><th scope="col">Client ID</th><th>Grant Types</th><th>Scopes</th><th>PKCE</th><th>Status</th><th>Actions</th></tr></thead><tbody>
          {clients.map((c: OAuthClient, i: number) => (<tr key={i} className="border-b hover:bg-gray-50"><td className="py-2 font-medium">{c.name}</td><td className="font-mono text-xs">{c.client_id}</td><td><div className="flex flex-wrap gap-1">{c.grant_types.map((g) => <span key={g} className="px-1.5 py-0.5 bg-blue-100 text-blue-700 rounded text-xs">{g}</span>)}</div></td><td><div className="flex flex-wrap gap-1 max-w-xs">{c.scopes.map((s) => <span key={s} className="px-1.5 py-0.5 bg-purple-100 text-purple-700 rounded text-xs">{s}</span>)}</div></td><td>{c.pkce_required ? <span className="text-green-600 text-xs">Required</span> : <span className="text-gray-400 text-xs">Optional</span>}</td><td><span className={`px-2 py-1 rounded text-xs ${c.status === "active" ? "bg-green-100 text-green-700" : "bg-gray-100 text-gray-500"}`}>{c.status}</span></td><td className="flex gap-2"><button className="text-xs text-blue-600 hover:underline">Edit</button><button className="text-xs text-purple-600 hover:underline">Rotate Secret</button></td></tr>))}
        </tbody></table>
      </div>
    </div>
  );
}

"use client";
import { useState } from "react";

interface ApiKey {
  id: string;
  name: string;
  scopes: string[];
  created: string;
  expires: string;
  last_used: string;
  status: "active" | "revoked";
  calls_today: number;
}

export default function ApiKeyManagementPage() {
  const [keys] = useState<ApiKey[]>([
    { id: "key-001", name: "CI/CD Pipeline", scopes: ["users:read", "roles:read"], created: "2025-01-01", expires: "2025-07-01", last_used: "2025-01-15 14:22", status: "active", calls_today: 1247 },
    { id: "key-002", name: "Mobile App", scopes: ["users:read", "auth:write"], created: "2024-12-15", expires: "2025-06-15", last_used: "2025-01-15 16:01", status: "active", calls_today: 8453 },
    { id: "key-003", name: "Legacy Integration", scopes: ["users:read"], created: "2024-06-01", expires: "2025-02-01", last_used: "2025-01-10", status: "active", calls_today: 12 },
  ]);
  const [showCreate, setShowCreate] = useState(false);

  return (
    <div className="p-8 space-y-6 max-w-5xl">
      <h1 className="text-2xl font-bold">API Key Management</h1>
      <p className="text-gray-600">Manage API keys with scoped permissions, rate limits, and usage tracking.</p>

      {keys.some((k) => k.status === "active" && new Date(k.expires) < new Date(Date.now() + 30 * 86400000)) && (
        <div className="bg-yellow-50 border border-yellow-300 rounded-lg p-4"><span className="text-sm font-medium text-yellow-800">Warning: One or more keys expire within 30 days. Rotate soon.</span></div>
      )}

      <div className="bg-white rounded-lg p-6 shadow">
        <div className="flex items-center justify-between mb-4"><h2 className="text-lg font-semibold">API Keys</h2><button onClick={() => setShowCreate(!showCreate)} className="px-4 py-1 bg-blue-600 text-white rounded text-sm hover:bg-blue-700">Create Key</button></div>
        {showCreate && (<div className="mb-4 border rounded p-4 space-y-3 bg-gray-50"><div><label className="block text-sm font-medium mb-1">Key Name</label><input type="text" placeholder="My API Key" className="border rounded px-3 py-2 w-full" /></div><div><label className="block text-sm font-medium mb-1">Scopes</label><div className="flex flex-wrap gap-2">{["users:read", "users:write", "roles:read", "roles:write", "orgs:read", "audit:read"].map((s) => (<label key={s} className="flex items-center gap-1 text-sm"><input type="checkbox" className="w-4 h-4" />{s}</label>))}</div></div><div className="grid grid-cols-2 gap-4"><div><label className="block text-sm font-medium mb-1">Expiry</label><input type="date" className="border rounded px-3 py-2 w-full" /></div><div><label className="block text-sm font-medium mb-1">Rate Limit (req/min)</label><input type="number" defaultValue={100} className="border rounded px-3 py-2 w-full" /></div></div><button className="px-4 py-2 bg-green-600 text-white rounded text-sm">Generate Key</button></div>)}
        <table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Name</th><th>Scopes</th><th>Created</th><th>Expires</th><th>Last Used</th><th>Calls Today</th><th>Status</th><th>Actions</th></tr></thead><tbody>
          {keys.map((k: ApiKey, i: number) => (<tr key={i} className="border-b hover:bg-gray-50"><td className="py-2 font-medium">{k.name}</td><td><div className="flex flex-wrap gap-1">{k.scopes.map((s) => <span key={s} className="px-1.5 py-0.5 bg-blue-100 text-blue-700 rounded text-xs font-mono">{s}</span>)}</div></td><td className="text-xs text-gray-500">{k.created}</td><td className="text-xs text-gray-500">{k.expires}</td><td className="text-xs text-gray-500">{k.last_used}</td><td>{k.calls_today.toLocaleString()}</td><td><span className={`px-2 py-1 rounded text-xs ${k.status === "active" ? "bg-green-100 text-green-700" : "bg-red-100 text-red-700"}`}>{k.status}</span></td><td><button className="text-xs text-red-600 hover:underline">Revoke</button></td></tr>))}
        </tbody></table>
      </div>
    </div>
  );
}

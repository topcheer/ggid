"use client";
import { useState, useEffect, useCallback } from "react";
import { Settings2, Save, X } from "lucide-react";

interface ClaimMap { id: string; claim_name: string; source: "user_attr" | "group" | "static"; source_value: string; transform: string; }
interface ClientOverride { client_id: string; client_name: string; extra_claims: number; }
const scopesList = ["openid", "profile", "email", "groups", "offline_access"];

export default function OidcClaimMappingPage() {
  const [mappings, setMappings] = useState<ClaimMap[]>([]);
  const [overrides, setOverrides] = useState<ClientOverride[]>([]);
  const [scopeMatrix, setScopeMatrix] = useState<Record<string, string[]>>({});
  const [tokenType, setTokenType] = useState("id_token");
  const [loading, setLoading] = useState(false);
  const [showAdd, setShowAdd] = useState(false);
  const [form, setForm] = useState<{ claim_name: string; source: "user_attr" | "group" | "static"; source_value: string; transform: string }>({ claim_name: "", source: "user_attr", source_value: "", transform: "" });

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/oauth/claim-mapping", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const d = await res.json(); setMappings(d.mappings || []); setOverrides(d.client_overrides || []); setScopeMatrix(d.scope_claims || {}); setTokenType(d.token_type || "id_token"); }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold flex items-center gap-2"><Settings2 className="w-6 h-6 text-blue-500" /> OIDC Claim Mapping</h1><p className="text-sm text-gray-500 mt-1">Configure OIDC claim sources, transforms, and scope-to-claim mappings.</p></div>
        <button onClick={() => setShowAdd(true)} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium">Add Claim</button>
      </div>

      <div className="flex items-center gap-3"><label className="text-sm font-medium">Token Type:</label><select value={tokenType} onChange={(e) => setTokenType(e.target.value)} className="px-3 py-1.5 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"><option value="id_token">ID Token</option><option value="access_token">Access Token</option><option value="userinfo">UserInfo</option></select></div>

      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Claim</th><th className="px-4 py-3 text-left font-medium">Source</th><th className="px-4 py-3 text-left font-medium">Value</th><th className="px-4 py-3 text-left font-medium">Transform</th></tr></thead>
          <tbody className="divide-y dark:divide-gray-800">{mappings.map((m) => (<tr key={m.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3 font-mono text-xs font-medium">{m.claim_name}</td><td className="px-4 py-3"><span className="px-2 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800">{m.source}</span></td><td className="px-4 py-3 text-xs text-gray-500">{m.source_value || "-"}</td><td className="px-4 py-3 text-xs text-gray-500">{m.transform || "-"}</td></tr>))}{mappings.length === 0 && !loading && <tr><td colSpan={4} className="px-4 py-8 text-center text-gray-500">No claim mappings.</td></tr>}</tbody>
        </table>
      </div>

      <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">Scope-to-Claims Matrix</h3><div className="overflow-x-auto"><table className="w-full text-sm"><thead><tr><th className="px-3 py-2 text-left text-xs font-medium text-gray-500">Scope</th><th className="px-3 py-2 text-left text-xs font-medium text-gray-500">Claims</th></tr></thead><tbody>{scopesList.map((s) => (<tr key={s} className="border-t dark:border-gray-800"><td className="px-3 py-2 font-mono text-xs font-medium">{s}</td><td className="px-3 py-2"><div className="flex flex-wrap gap-1">{(scopeMatrix[s] || []).map((c) => (<span key={c} className="px-1.5 py-0.5 rounded text-xs bg-blue-50 dark:bg-blue-900/20 font-mono">{c}</span>))}{(scopeMatrix[s] || []).length === 0 && <span className="text-xs text-gray-400">None</span>}</div></td></tr>))}</tbody></table></div></div>

      {overrides.length > 0 && (<div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-2">Per-Client Overrides</h3>{overrides.map((o) => (<div key={o.client_id} className="flex items-center justify-between text-sm py-1"><span className="font-medium">{o.client_name}</span><span className="text-xs text-gray-500">{o.extra_claims} extra claim(s)</span></div>))}</div>)}

      {showAdd && (<div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setShowAdd(false)}><div className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4" onClick={(e) => e.stopPropagation()}><div className="flex items-center justify-between px-6 py-4 border-b dark:border-gray-800"><h3 className="font-semibold">Add Claim Mapping</h3><button onClick={() => setShowAdd(false)}><X className="w-5 h-5 text-gray-400" /></button></div><div className="px-6 py-4 space-y-3"><div><label className="text-sm font-medium">Claim Name</label><input type="text" value={form.claim_name} onChange={(e) => setForm({ ...form, claim_name: e.target.value })} placeholder="department" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div><div><label className="text-sm font-medium">Source</label><select value={form.source} onChange={(e) => setForm({ ...form, source: e.target.value as "user_attr" | "group" | "static" })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm"><option value="user_attr">User Attribute</option><option value="group">Group</option><option value="static">Static</option></select></div><div><label className="text-sm font-medium">Source Value</label><input type="text" value={form.source_value} onChange={(e) => setForm({ ...form, source_value: e.target.value })} placeholder="user.department" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div></div><div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800"><button onClick={() => setShowAdd(false)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">Cancel</button><button onClick={() => setShowAdd(false)} disabled={!form.claim_name} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium disabled:opacity-50">Save</button></div></div></div>)}
    </div>
  );
}

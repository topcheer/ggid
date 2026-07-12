"use client";

import { useState, useCallback } from "react";
import { Search, ShieldCheck, ShieldX, Shield, Clock } from "lucide-react";

interface Permission {
  id: string;
  resource: string;
  action: string;
  source: "direct" | "inherited";
  via_group: string | null;
  last_used: string | null;
  unused_90d: boolean;
  over_privileged: boolean;
  recommendation: "keep" | "revoke" | "reduce";
}

const recConfig: Record<string, { color: string; icon: typeof ShieldCheck; label: string }> = {
  keep: { color: "text-green-600", icon: ShieldCheck, label: "Keep" },
  revoke: { color: "text-red-600", icon: ShieldX, label: "Revoke" },
  reduce: { color: "text-yellow-600", icon: Shield, label: "Reduce" },
};

export default function EntitlementReviewPage() {
  const [search, setSearch] = useState("");
  const [perms, setPerms] = useState<Permission[]>([]);
  const [loading, setLoading] = useState(false);

  const searchUser = useCallback(async () => {
    if (!search) return;
    setLoading(true);
    try {
      const res = await fetch("/api/v1/identity/entitlement-review?user=" + encodeURIComponent(search), { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const d = await res.json(); setPerms(d.permissions || d || []); }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, [search]);

  const unused = perms.filter((p) => p.unused_90d).length;
  const overPriv = perms.filter((p) => p.over_privileged).length;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Shield className="w-6 h-6 text-blue-500" /> Entitlement Review</h1>
        <p className="text-sm text-gray-500 mt-1">Review user permissions with usage analytics and recommendations.</p>
      </div>

      <div className="flex items-center gap-2">
        <div className="relative flex-1 max-w-md"><Search className="absolute left-2 top-2.5 w-4 h-4 text-gray-400" /><input type="text" value={search} onChange={(e) => setSearch(e.target.value)} onKeyDown={(e) => { if (e.key === "Enter") searchUser(); }} placeholder="Search user ID or email..." className="w-full pl-8 pr-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div>
        <button onClick={searchUser} disabled={loading || !search} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50">Review</button>
      </div>

      {perms.length > 0 && (
        <>
          <div className="grid grid-cols-4 gap-4">
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Total Perms</span><p className="text-xl font-bold mt-1">{perms.length}</p></div>
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Direct</span><p className="text-xl font-bold text-blue-600 mt-1">{perms.filter((p) => p.source === "direct").length}</p></div>
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Unused 90d</span><p className="text-xl font-bold text-yellow-600 mt-1">{unused}</p></div>
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Over-Privileged</span><p className="text-xl font-bold text-red-600 mt-1">{overPriv}</p></div>
          </div>

          <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
            <table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Resource</th><th className="px-4 py-3 text-left font-medium">Action</th><th className="px-4 py-3 text-left font-medium">Source</th><th className="px-4 py-3 text-left font-medium">Last Used</th><th className="px-4 py-3 text-left font-medium">Flags</th><th className="px-4 py-3 text-left font-medium">Recommendation</th></tr></thead>
              <tbody className="divide-y dark:divide-gray-800">{perms.map((p) => { const cfg = recConfig[p.recommendation]; const Icon = cfg.icon; return (
                <tr key={p.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                  <td className="px-4 py-3 font-mono text-xs">{p.resource}</td>
                  <td className="px-4 py-3 text-xs">{p.action}</td>
                  <td className="px-4 py-3"><span className={"text-xs " + (p.source === "direct" ? "text-blue-600" : "text-purple-600")}>{p.source}</span>{p.via_group && <span className="text-xs text-gray-400 ml-1">({p.via_group})</span>}</td>
                  <td className="px-4 py-3 text-xs text-gray-500">{p.last_used ? <span className="flex items-center gap-1"><Clock className="w-3 h-3" />{p.last_used}</span> : "never"}</td>
                  <td className="px-4 py-3"><div className="flex gap-1">{p.unused_90d && <span className="px-1.5 py-0.5 rounded text-xs bg-yellow-100 dark:bg-yellow-900/30 dark:text-yellow-400">unused</span>}{p.over_privileged && <span className="px-1.5 py-0.5 rounded text-xs bg-red-100 dark:bg-red-900/30 dark:text-red-400">over-priv</span>}</div></td>
                  <td className="px-4 py-3"><span className={"flex items-center gap-1 text-xs " + cfg.color}><Icon className="w-3.5 h-3.5" /> {cfg.label}</span></td>
                </tr>
              ); })}</tbody>
            </table>
          </div>
        </>
      )}
    </div>
  );
}

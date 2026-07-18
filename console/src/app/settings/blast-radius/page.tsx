"use client";

import { useState, useCallback } from "react";
import { Bomb, Eye, Users, Shield, FileText, ChevronRight } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface BlastRadiusData {
  affected_users_count: number;
  affected_roles: string[];
  affected_resources: { name: string; type: string; children?: { name: string; type: string }[] }[];
  cascading_policies: string[];
}

interface Policy { id: string; name: string; }

export default function BlastRadiusPage() {
  const t = useTranslations();
  const [policies] = useState<Policy[]>([{ id: "p1", name: "Admin Access" }, { id: "p2", name: "Data Access" }, { id: "p3", name: "External Partner" }]);
  const [policyId, setPolicyId] = useState("");
  const [previewMode, setPreviewMode] = useState(true);
  const [data, setData] = useState<BlastRadiusData | null>(null);
  const [loading, setLoading] = useState(false);
  const [expanded, setExpanded] = useState<string | null>(null);

  const analyze = useCallback(async () => {
    if (!policyId) return;
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/policy/blast-radius?id=${encodeURIComponent(policyId)}`, { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, [policyId]);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Bomb className="w-6 h-6 text-red-500" /> Blast Radius</h1>
        <p className="text-sm text-gray-500 mt-1">Analyze the impact scope of modifying or removing a policy.</p>
      </div>

      <div className="rounded-lg border dark:border-gray-800 p-4 space-y-3">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          <div><label className="text-sm font-medium">Policy</label><select aria-label="Policy id" value={policyId} onChange={(e) => setPolicyId(e.target.value)} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"><option value="">Select Policy</option>{policies.map((p: any) => <option key={p.id} value={p.id}>{p.name}</option>)}</select></div>
          <div className="flex items-end"><label className="flex items-center gap-2 text-sm font-medium pb-2"><input aria-label="Preview mode" type="checkbox" checked={previewMode} onChange={(e) => setPreviewMode(e.target.checked)} className="rounded" /> Preview Mode (read-only)</label></div>
        </div>
        <button aria-label="Eye" onClick={analyze} disabled={loading || !policyId} className="px-4 py-2 rounded-lg bg-red-600 text-white text-sm font-medium hover:bg-red-700 disabled:opacity-50 flex items-center gap-2"><Eye className="w-4 h-4" /> {loading ? "Analyzing..." : "Analyze Blast Radius"}</button>
      </div>

      {data && (
        <>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="rounded-lg border p-4 dark:border-gray-800 flex items-center gap-3"><Users className="w-8 h-8 text-red-500" /><div><span className="text-sm text-gray-500">Affected Users</span><p className="text-xl font-bold text-red-600">{data.affected_users_count}</p></div></div>
            <div className="rounded-lg border p-4 dark:border-gray-800 flex items-center gap-3"><Shield className="w-8 h-8 text-orange-500" /><div><span className="text-sm text-gray-500">Roles</span><p className="text-xl font-bold text-orange-600">{data.affected_roles.length}</p></div></div>
            <div className="rounded-lg border p-4 dark:border-gray-800 flex items-center gap-3"><FileText className="w-8 h-8 text-blue-500" /><div><span className="text-sm text-gray-500">Resources</span><p className="text-xl font-bold text-blue-600">{data.affected_resources.length}</p></div></div>
            <div className="rounded-lg border p-4 dark:border-gray-800 flex items-center gap-3"><Bomb className="w-8 h-8 text-purple-500" /><div><span className="text-sm text-gray-500">Cascading</span><p className="text-xl font-bold text-purple-600">{data.cascading_policies.length}</p></div></div>
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="text-sm font-semibold mb-3">Affected Roles</h3>
              <div className="flex flex-wrap gap-2">{data.affected_roles.map((r: any) => <span key={r} className="px-2 py-1 rounded text-xs bg-orange-100 dark:bg-orange-900/30 dark:text-orange-400 font-mono">{r}</span>)}{data.affected_roles.length === 0 && <span className="text-xs text-gray-400">None</span>}</div>
            </div>

            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="text-sm font-semibold mb-3">Cascading Policies</h3>
              <div className="space-y-1">{data.cascading_policies.map((p: any) => <div key={p} className="text-sm font-mono text-purple-600">{p}</div>)}{data.cascading_policies.length === 0 && <span className="text-xs text-gray-400">None</span>}</div>
            </div>
          </div>

          <div className="rounded-lg border dark:border-gray-800 p-4">
            <h3 className="text-sm font-semibold mb-3">Affected Resources Tree</h3>
            <div className="space-y-1">
              {data.affected_resources.map((r: any) => (
                <div key={r.name}>
                  <button onClick={() => setExpanded(expanded === r.name ? null : r.name)} aria-label={`Toggle ${r.name}`} className="flex items-center gap-2 text-sm hover:bg-gray-50 dark:hover:bg-gray-900/30 w-full px-2 py-1 rounded">
                    <ChevronRight className={`w-3 h-3 text-gray-400 transition-transform ${expanded === r.name ? "rotate-90" : ""}`} />
                    <span className="font-mono text-xs">{r.name}</span>
                    <span className="text-xs text-gray-400">({r.type})</span>
                  </button>
                  {expanded === r.name && r.children && (
                    <div className="ml-6 space-y-1">
                      {r.children.map((c: any) => (
                        <div key={c.name} className="flex items-center gap-2 text-sm pl-2 py-0.5 border-l dark:border-gray-800"><span className="font-mono text-xs text-gray-500">{c.name}</span><span className="text-xs text-gray-400">({c.type})</span></div>
                      ))}
                    </div>
                  )}
                </div>
              ))}
            </div>
          </div>
        </>
      )}
    </div>
  );
}

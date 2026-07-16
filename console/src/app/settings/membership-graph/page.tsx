"use client";

import { useState, useCallback } from "react";
import { Network, Users, GitBranch, AlertCircle } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface GraphData {
  group_id: string;
  group_name: string;
  total_depth: number;
  direct_members: { id: string; name: string; type: "user" | "group" }[];
  nested_groups: { id: string; name: string; depth: number; children?: { id: string; name: string }[] }[];
  parent_groups: { id: string; name: string }[];
  circular_detected: boolean;
  circular_path?: string[];
}

interface Group { id: string; name: string; }

export default function MembershipGraphPage() {
  const t = useTranslations();

  const [groups] = useState<Group[]>([{ id: "g1", name: "Engineering" }, { id: "g2", name: "Admins" }, { id: "g3", name: "Contractors" }]);
  const [groupId, setGroupId] = useState("");
  const [data, setData] = useState<GraphData | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchGraph = useCallback(async () => {
    if (!groupId) return;
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/identity/membership-graph?group_id=${encodeURIComponent(groupId)}`, { headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, [groupId]);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Network className="w-6 h-6 text-indigo-500" /> {t("membershipGraph.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Visualize group membership hierarchies and detect circular dependencies.</p>
      </div>

      <select aria-label="Group id" value={groupId} onChange={(e) => setGroupId(e.target.value)} className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm">
        <option value="">Select Group</option>
        {groups.map((g) => <option key={g.id} value={g.id}>{g.name}</option>)}
      </select>

      {data && (
        <>
          <div className="flex items-center gap-3">
            <div className="rounded-lg border dark:border-gray-800 p-3 flex items-center gap-2"><GitBranch className="w-5 h-5 text-blue-500" /><span className="text-sm text-gray-500">Depth:</span><span className="font-bold">{data.total_depth}</span></div>
            <div className="rounded-lg border dark:border-gray-800 p-3 flex items-center gap-2"><Users className="w-5 h-5 text-green-500" /><span className="text-sm text-gray-500">Direct:</span><span className="font-bold">{data.direct_members.length}</span></div>
            <div className="rounded-lg border dark:border-gray-800 p-3 flex items-center gap-2"><Network className="w-5 h-5 text-purple-500" /><span className="text-sm text-gray-500">Nested:</span><span className="font-bold">{data.nested_groups.length}</span></div>
          </div>

          {data.circular_detected && (
            <div className="rounded-lg border border-red-300 dark:border-red-800 bg-red-50 dark:bg-red-900/20 p-4 flex items-center gap-2"><AlertCircle className="w-5 h-5 text-red-500" /><div><span className="font-semibold text-red-700 dark:text-red-400">Circular Dependency Detected</span>{data.circular_path && <p className="text-sm text-red-600 mt-1">Path: {data.circular_path.join(" -> ")}</p>}</div></div>
          )}

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="text-sm font-semibold mb-3">Direct Members</h3>
              <div className="space-y-1">{data.direct_members.map((m) => (
                <div key={m.id} className="flex items-center gap-2 text-sm"><span className={`w-2 h-2 rounded-full ${m.type === "user" ? "bg-green-500" : "bg-blue-500"}`} /><span className="flex-1">{m.name}</span><span className="text-xs text-gray-400 font-mono">{m.type}</span></div>
              ))}{data.direct_members.length === 0 && <p className="text-xs text-gray-400">No direct members.</p>}</div>
            </div>

            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="text-sm font-semibold mb-3">Parent Groups (Chain)</h3>
              <div className="space-y-1">{data.parent_groups.map((p) => (
                <div key={p.id} className="flex items-center gap-2 text-sm"><span className="w-2 h-2 rounded-full bg-orange-500" /><span className="flex-1">{p.name}</span></div>
              ))}{data.parent_groups.length === 0 && <p className="text-xs text-gray-400">No parent groups.</p>}</div>
            </div>
          </div>

          <div className="rounded-lg border dark:border-gray-800 p-4">
            <h3 className="text-sm font-semibold mb-3">Nested Groups Tree</h3>
            <div className="space-y-1">
              {data.nested_groups.map((ng) => (
                <div key={ng.id}>
                  <div className="flex items-center gap-2 text-sm py-1"><span className="text-xs text-gray-400">L{ng.depth}</span><span className="w-2 h-2 rounded-full bg-purple-500" /><span className="flex-1 font-medium">{ng.name}</span></div>
                  {ng.children && ng.children.length > 0 && (
                    <div className="ml-8 space-y-1 border-l dark:border-gray-800 pl-3">{ng.children.map((c) => (
                      <div key={c.id} className="flex items-center gap-2 text-sm py-0.5"><span className="w-1.5 h-1.5 rounded-full bg-gray-400" /><span className="text-gray-500">{c.name}</span></div>
                    ))}</div>
                  )}
                </div>
              ))}
              {data.nested_groups.length === 0 && <p className="text-xs text-gray-400">No nested groups.</p>}
            </div>
          </div>
        </>
      )}
      {!data && !loading && groupId && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
      {!groupId && <p className="text-sm text-gray-500 text-center py-8">Select a group to view membership graph.</p>}
    </div>
  );
}

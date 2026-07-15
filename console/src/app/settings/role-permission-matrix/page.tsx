"use client";
import { useState, useEffect, useCallback } from "react";
import { Grid3x3, Download } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface MatrixData { roles: string[]; permissions: string[]; assignments: Record<string, Record<string, "allow" | "deny" | "inherit">>; }

export default function RolePermissionMatrixPage() {
  const t = useTranslations();

  const [data, setData] = useState<MatrixData | null>(null);
  const [loading, setLoading] = useState(false);
  const [search, setSearch] = useState("");

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/policy/role-permission-matrix", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const toggleCell = (role: string, perm: string) => {
    if (!data) return;
    const current = data.assignments[role]?.[perm] || "inherit";
    const next = current === "allow" ? "deny" : current === "deny" ? "inherit" : "allow";
    setData({ ...data, assignments: { ...data.assignments, [role]: { ...data.assignments[role], [perm]: next } } });
  };

  if (!data) return <p className="text-sm text-gray-500 text-center py-8">Loading...</p>;

  const filteredPerms = data.permissions.filter((p) => !search || p.includes(search.toLowerCase()));

  const cellColors: Record<string, string> = { allow: "bg-green-500", deny: "bg-red-500", inherit: "bg-gray-300 dark:bg-gray-700" };
  const getPermCount = (role: string) => Object.values(data.assignments[role] || {}).filter((v) => v === "allow").length;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold flex items-center gap-2"><Grid3x3 className="w-6 h-6 text-blue-500" /> {t("rolePermissionMatrix.title")}</h1><p className="text-sm text-gray-500 mt-1">Click cells to toggle allow/deny/inherit for each role-permission combination.</p></div>
        <button className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm flex items-center gap-2"><Download className="w-4 h-4" /> Export</button>
      </div>

      <input type="text" value={search} onChange={(e) => setSearch(e.target.value)} placeholder="Search permissions..." className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm w-64" />

      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm"><thead><tr><th className="px-3 py-2 text-left text-xs font-medium text-gray-500 sticky left-0 bg-gray-50 dark:bg-gray-900">Permission</th>{data.roles.map((r) => (<th key={r} className="px-3 py-2 text-center text-xs font-medium"><div className="flex flex-col items-center gap-1"><span>{r}</span><span className="px-1.5 py-0.5 rounded bg-blue-100 dark:bg-blue-900/30 dark:text-blue-400 text-[10px]">{getPermCount(r)}</span></div></th>))}</tr></thead>
          <tbody className="divide-y dark:divide-gray-800">{filteredPerms.map((perm) => (<tr key={perm} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-3 py-2 font-mono text-xs font-medium sticky left-0 bg-white dark:bg-gray-900">{perm}</td>{data.roles.map((role) => { const val = data.assignments[role]?.[perm] || "inherit"; return (<td key={role} className="px-3 py-2"><button onClick={() => toggleCell(role, perm)} className={"w-8 h-8 rounded mx-auto block " + cellColors[val]} title={role + " / " + perm + ": " + val} /></td>); })}</tr>))}</tbody>
        </table>
      </div>

      <div className="flex items-center gap-4 text-xs"><div className="flex items-center gap-1"><span className="w-4 h-4 rounded bg-green-500" /> Allow</div><div className="flex items-center gap-1"><span className="w-4 h-4 rounded bg-red-500" /> Deny</div><div className="flex items-center gap-1"><span className="w-4 h-4 rounded bg-gray-300 dark:bg-gray-700" /> Inherit</div></div>
    </div>
  );
}

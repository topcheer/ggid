"use client";

import { useState, useEffect, useCallback } from "react";
import { Building2, ChevronRight, AlertTriangle, Layers as LayersIcon } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface TreeNode {
  id: string;
  name: string;
  title: string;
  reports: TreeNode[];
  span_of_control: number;
  layer: number;
  is_orphan: boolean;
}

interface OrgTreeData {
  root: TreeNode | null;
  total_layers: number;
  orphan_managers: { id: string; name: string }[];
  circular_detected: boolean;
  circular_path?: string[];
}

export default function ReportingStructurePage() {
  const t = useTranslations();

  const [data, setData] = useState<OrgTreeData | null>(null);
  const [loading, setLoading] = useState(false);
  const [expanded, setExpanded] = useState<Set<string>>(new Set());

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/identity/reporting-structure", { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const toggle = (id: string) => setExpanded((prev) => { const next = new Set(prev); if (next.has(id)) next.delete(id); else next.add(id); return next; });

  const renderNode = (node: TreeNode, depth: number = 0) => (
    <div key={node.id}>
      <div className="flex items-center gap-2" style={{ paddingLeft: depth * 24 }}>
        {node.reports.length > 0 && <button onClick={() => toggle(node.id)} className="text-gray-400 hover:text-gray-600"><ChevronRight className={`w-4 h-4 transition-transform ${expanded.has(node.id) ? "rotate-90" : ""}`} /></button>}
        {node.reports.length === 0 && <span className="w-4" />}
        <div className={`flex items-center gap-2 py-1 px-2 rounded flex-1 ${node.is_orphan ? "bg-red-50 dark:bg-red-900/20" : ""}`}>
          <span className="font-medium text-sm">{node.name}</span>
          <span className="text-xs text-gray-400">{node.title}</span>
          {node.span_of_control > 0 && <span className="px-1.5 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800">span: {node.span_of_control}</span>}
          {node.is_orphan && <AlertTriangle className="w-3 h-3 text-red-500" />}
        </div>
      </div>
      {expanded.has(node.id) && node.reports.length > 0 && (
        <div className="border-l dark:border-gray-800 ml-3">{node.reports.map((r) => renderNode(r, depth + 1))}</div>
      )}
    </div>
  );

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Building2 className="w-6 h-6 text-blue-500" /> {t("reportingStructure.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Organization hierarchy with span of control and anomaly detection.</p>
      </div>

      {data && (
        <>
          <div className="flex items-center gap-3">
            <div className="rounded-lg border dark:border-gray-800 p-3 flex items-center gap-2"><LayersIcon className="w-5 h-5 text-blue-500" /><span className="text-sm text-gray-500">Layers:</span><span className="font-bold">{data.total_layers}</span></div>
            {data.orphan_managers.length > 0 && <div className="rounded-lg border border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-900/20 p-3 flex items-center gap-2"><AlertTriangle className="w-5 h-5 text-red-500" /><span className="text-sm text-red-700 dark:text-red-400">{data.orphan_managers.length} orphan managers</span></div>}
          </div>

          {data.circular_detected && (
            <div className="rounded-lg border border-red-300 dark:border-red-800 bg-red-50 dark:bg-red-900/20 p-4 flex items-center gap-2"><AlertTriangle className="w-5 h-5 text-red-500" /><div><span className="font-semibold text-red-700 dark:text-red-400">Circular Reporting Detected</span>{data.circular_path && <p className="text-sm text-red-600 mt-1">{data.circular_path.join(" -> ")}</p>}</div></div>
          )}

          {data.root && <div className="rounded-lg border dark:border-gray-800 p-4">{renderNode(data.root)}</div>}
          {!data.root && <p className="text-sm text-gray-500 text-center py-8">No reporting structure data.</p>}
        </>
      )}
      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
    </div>
  );
}

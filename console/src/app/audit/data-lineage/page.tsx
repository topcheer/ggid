"use client";

import { useState, useEffect, useCallback } from "react";
import { Search, GitBranch, User, Download, FileText, Database } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface LineageNode {
  id: string;
  type: "source" | "creator" | "modifier" | "consumer" | "access";
  label: string;
  timestamp: string;
  metadata?: Record<string, string>;
}

interface LineageEdge {
  from: string;
  to: string;
  label: string;
}

interface LineageData {
  resource_id: string;
  resource_type: string;
  resource_name: string;
  nodes: LineageNode[];
  edges: LineageEdge[];
  created_by: string;
  created_at: string;
  last_modified_by: string;
  last_modified_at: string;
  access_events: AccessEvent[];
  downstream_consumers: Consumer[];
}

interface AccessEvent {
  actor: string;
  action: string;
  timestamp: string;
  ip: string;
}

interface Consumer {
  name: string;
  type: string;
  access_level: string;
}

const typeIcons: Record<string, typeof User> = {
  source: Database,
  creator: User,
  modifier: User,
  consumer: Download,
  access: FileText,
};

const typeColors: Record<string, string> = {
  source: "bg-purple-50 dark:bg-purple-900/20 text-purple-600",
  creator: "bg-green-50 dark:bg-green-900/20 text-green-600",
  modifier: "bg-yellow-50 dark:bg-yellow-900/20 text-yellow-600",
  consumer: "bg-blue-50 dark:bg-blue-900/20 text-blue-600",
  access: "bg-gray-50 dark:bg-gray-800 text-gray-600",
};

export default function DataLineagePage() {
  const t = useTranslations();

  const [data, setData] = useState<LineageData | null>(null);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(false);

  const fetchLineage = useCallback(async (resource: string) => {
    if (!resource) return;
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/audit/data-lineage?resource=${encodeURIComponent(resource)}`, { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const json = await res.json();
        setData(json);
      }
    } catch {
      /* noop */
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (!search) return;
    fetchLineage(search);
  }, [search, fetchLineage]);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><GitBranch className="w-6 h-6 text-purple-500" /> {t("auditDataLineage.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Trace resource provenance, modifications, access events, and downstream consumers.</p>
      </div>

      {/* Resource search */}
      <div className="relative max-w-md">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
        <input
          type="text"
          placeholder="Search by resource ID or name..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="w-full pl-9 pr-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"
        />
      </div>

      {loading && <p className="text-sm text-gray-500">Loading lineage...</p>}

      {data && (
        <div className="space-y-4">
          {/* Resource header */}
          <div className="rounded-lg border dark:border-gray-800 p-4">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-lg bg-purple-50 dark:bg-purple-900/20 flex items-center justify-center">
                <Database className="w-5 h-5 text-purple-500" />
              </div>
              <div>
                <h3 className="font-semibold">{data.resource_name}</h3>
                <p className="text-xs text-gray-500">{data.resource_type} &middot; {data.resource_id}</p>
              </div>
            </div>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mt-4 text-sm">
              <div>
                <span className="text-gray-400">Created By</span>
                <p className="font-medium">{data.created_by}</p>
              </div>
              <div>
                <span className="text-gray-400">Created At</span>
                <p className="font-medium">{data.created_at}</p>
              </div>
              <div>
                <span className="text-gray-400">Last Modified By</span>
                <p className="font-medium">{data.last_modified_by}</p>
              </div>
              <div>
                <span className="text-gray-400">Last Modified At</span>
                <p className="font-medium">{data.last_modified_at}</p>
              </div>
            </div>
          </div>

          {/* Lineage graph (simplified list view) */}
          <div className="rounded-lg border dark:border-gray-800">
            <div className="px-4 py-3 border-b dark:border-gray-800">
              <h3 className="font-semibold flex items-center gap-2"><GitBranch className="w-4 h-4" /> Lineage Graph</h3>
            </div>
            <div className="p-4">
              {data.nodes.map((node) => {
                const Icon = typeIcons[node.type] || FileText;
                return (
                  <div key={node.id} className="flex items-center gap-3 mb-2 last:mb-0">
                    <div className={`w-8 h-8 rounded-lg flex items-center justify-center ${typeColors[node.type] || typeColors.access}`}>
                      <Icon className="w-4 h-4" />
                    </div>
                    <div className="flex-1">
                      <span className="text-sm font-medium">{node.label}</span>
                      <span className="text-xs text-gray-400 ml-2">{node.timestamp}</span>
                    </div>
                    <span className="text-xs px-2 py-0.5 rounded bg-gray-100 dark:bg-gray-800 text-gray-500">{node.type}</span>
                  </div>
                );
              })}
              {data.nodes.length === 0 && <p className="text-sm text-gray-500">No lineage nodes.</p>}
            </div>
          </div>

          {/* Access events */}
          <div className="rounded-lg border dark:border-gray-800">
            <div className="px-4 py-3 border-b dark:border-gray-800">
              <h3 className="font-semibold flex items-center gap-2"><FileText className="w-4 h-4" /> Access Events</h3>
            </div>
            <div className="divide-y dark:divide-gray-800 max-h-48 overflow-y-auto">
              {data.access_events.map((evt, i) => (
                <div key={i} className="px-4 py-2 flex items-center justify-between text-sm">
                  <div className="flex items-center gap-3">
                    <span className="font-medium">{evt.actor}</span>
                    <span className="text-gray-500">{evt.action}</span>
                  </div>
                  <div className="flex items-center gap-2 text-xs text-gray-400">
                    <span className="font-mono">{evt.ip}</span>
                    <span>{evt.timestamp}</span>
                  </div>
                </div>
              ))}
              {data.access_events.length === 0 && <p className="px-4 py-3 text-sm text-gray-500">No access events.</p>}
            </div>
          </div>

          {/* Downstream consumers */}
          <div className="rounded-lg border dark:border-gray-800">
            <div className="px-4 py-3 border-b dark:border-gray-800">
              <h3 className="font-semibold flex items-center gap-2"><Download className="w-4 h-4" /> Downstream Consumers</h3>
            </div>
            <div className="divide-y dark:divide-gray-800">
              {data.downstream_consumers.map((c, i) => (
                <div key={i} className="px-4 py-2 flex items-center justify-between text-sm">
                  <div className="flex items-center gap-2">
                    <span className="font-medium">{c.name}</span>
                    <span className="text-xs text-gray-400">{c.type}</span>
                  </div>
                  <span className={`px-2 py-0.5 rounded text-xs ${c.access_level === "read" ? "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400" : "bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400"}`}>{c.access_level}</span>
                </div>
              ))}
              {data.downstream_consumers.length === 0 && <p className="px-4 py-3 text-sm text-gray-500">No downstream consumers.</p>}
            </div>
          </div>
        </div>
      )}

      {!data && !loading && search && <p className="text-sm text-gray-500">No lineage data found.</p>}
      {!data && !search && <p className="text-sm text-gray-500 text-center py-8">Search for a resource to view its lineage.</p>}
    </div>
  );
}

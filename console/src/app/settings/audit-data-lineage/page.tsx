"use client";

import { useAuditDataLineage } from "@ggid/sdk-react";
import { Database, ArrowRight, GitBranch, Trash2, Shield } from "lucide-react";

export default function AuditDataLineagePage() {
  const { data, loading, error, refresh } = useAuditDataLineage();

  if (loading) return <div className="p-8 text-gray-400">Loading data lineage...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Data Lineage</h1>
          <p className="text-sm text-gray-400 mt-1">Track data flow from source to destination with PII classification</p>
        </div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      {/* Data Flow Diagram */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
          <GitBranch className="w-4 h-4 text-blue-400" />
          Data Flow Diagram
        </h2>
        <div className="flex items-center justify-around gap-4 py-8">
          {data?.flow_nodes?.map((node, i) => (
            <div key={node.id} className="flex items-center gap-4">
              <div
                className={"p-4 rounded-xl border-2 text-center min-w-[120px] " + (
                  node.type === "source" ? "bg-blue-900/30 border-blue-700" :
                  node.type === "processor" ? "bg-purple-900/30 border-purple-700" :
                  "bg-green-900/30 border-green-700"
                )}
              >
                <Database className={"w-5 h-5 mx-auto mb-1 " + (
                  node.type === "source" ? "text-blue-400" :
                  node.type === "processor" ? "text-purple-400" :
                  "text-green-400"
                )} />
                <p className="text-xs font-medium">{node.name}</p>
                <p className="text-xs text-gray-500 capitalize">{node.type}</p>
              </div>
              {i < (data?.flow_nodes?.length ?? 0) - 1 && (
                <ArrowRight className="w-5 h-5 text-gray-600" />
              )}
            </div>
          ))}
        </div>
      </div>

      {/* Per-Dataset Lineage */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold mb-4">Dataset Lineage</h2>
        <div className="space-y-3">
          {(data?.datasets ?? []).map((ds) => (
            <div key={ds.dataset_name} className="bg-gray-800 rounded-lg p-4">
              <div className="flex items-center gap-2 mb-3">
                <Database className="w-4 h-4 text-gray-400" />
                <p className="text-sm font-semibold">{ds.dataset_name}</p>
                {ds.pii_classification !== "none" && (
                  <span className={"text-xs px-2 py-0.5 rounded " + (
                    ds.pii_classification === "high" ? "bg-red-900 text-red-300" :
                    ds.pii_classification === "medium" ? "bg-yellow-900 text-yellow-300" :
                    "bg-green-900 text-green-300"
                  )}>
                    PII: {ds.pii_classification}
                  </span>
                )}
              </div>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
                <div>
                  <p className="text-xs text-gray-500 mb-1">Source</p>
                  <p className="text-xs">{ds.source_system}</p>
                </div>
                <div>
                  <p className="text-xs text-gray-500 mb-1">Transformations</p>
                  <div className="flex flex-wrap gap-1">
                    {ds.transformations.map((t) => (
                      <span key={t} className="text-xs px-1.5 py-0.5 bg-gray-700 rounded">{t}</span>
                    ))}
                  </div>
                </div>
                <div>
                  <p className="text-xs text-gray-500 mb-1">Downstream Consumers</p>
                  <div className="flex flex-wrap gap-1">
                    {ds.downstream_consumers.map((c) => (
                      <span key={c} className="text-xs px-1.5 py-0.5 bg-gray-700 rounded">{c}</span>
                    ))}
                  </div>
                </div>
              </div>
              <div className="mt-3 flex items-center gap-4">
                <div className="flex items-center gap-1">
                  <Shield className="w-3 h-3 text-gray-500" />
                  <span className="text-xs text-gray-500">Retention: {ds.retention_path}</span>
                </div>
                <div className="flex items-center gap-1">
                  <Trash2 className="w-3 h-3 text-gray-500" />
                  <span className="text-xs text-gray-500">Deletion: {ds.deletion_propagation}</span>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

"use client";

import { useState } from "react";
import { useTranslations } from "@/lib/i18n";
import { useTenantMigrationTool } from "@ggid/sdk-react";
import { Database, Play, RotateCcw, CheckCircle } from "lucide-react";

export default function TenantMigrationToolPage() {
  const { data, loading, error, refresh, executeMigration } = useTenantMigrationTool();
  const [selectedScope, setSelectedScope] = useState<string[]>(["users", "roles"]);
  const t = useTranslations();

  if (loading) return <div className="p-8 text-gray-400">{t("tenantMigration.loading")}</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const toggleScope = (s: string) => {
    setSelectedScope((prev) => prev.includes(s) ? prev.filter((x) => x !== s) : [...prev, s]);
  };

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">{t("tenantMigration.title")}</h1>
          <p className="text-sm text-gray-400 mt-1">{t("tenantMigration.subtitle")}</p>
        </div>
      </div>

      {/* Source / Destination */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-6">
          <p className="text-xs text-gray-500 mb-1">{t("tenantMigration.source")}</p>
          <p className="text-sm font-medium">{data?.source_tenant ?? "--"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-6">
          <p className="text-xs text-gray-500 mb-1">{t("tenantMigration.destination")}</p>
          <p className="text-sm font-medium">{data?.destination_tenant ?? "--"}</p>
        </div>
      </div>

      {/* Migration Scope */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold mb-3">{t("tenantMigration.scope")}</h2>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
          {(data?.migration_scope ?? []).map((s) => (
            <button key={s.name} onClick={() => toggleScope(s.name)} className={"flex items-center gap-2 p-3 rounded-lg border transition " + (selectedScope.includes(s.name) ? "bg-blue-900 border-blue-700" : "bg-gray-800 border-gray-700")}>
              <Database className="w-4 h-4 text-gray-400" />
              <div className="text-left">
                <p className="text-sm font-medium">{s.name}</p>
                <p className="text-xs text-gray-400">{s.record_count.toLocaleString()} records</p>
              </div>
              {selectedScope.includes(s.name) && <CheckCircle className="w-4 h-4 text-blue-400 ml-auto" />}
            </button>
          ))}
        </div>
      </div>

      {/* Dry Run Preview */}
      {data?.dry_run && (
        <div className="bg-gray-900 rounded-xl p-6 mb-6">
          <h2 className="text-sm font-semibold mb-3">{t("tenantMigration.dryRun")}</h2>
          <div className="grid grid-cols-3 gap-4">
            <div><p className="text-xs text-gray-500">{t("tenantMigration.affectedRecords")}</p><p className="text-xl font-bold text-blue-400">{data.dry_run.affected_records.toLocaleString()}</p></div>
            <div><p className="text-xs text-gray-500">{t("tenantMigration.estimatedDuration")}</p><p className="text-xl font-bold">{data.dry_run.estimated_duration}</p></div>
            <div><p className="text-xs text-gray-500">{t("tenantMigration.conflicts")}</p><p className={"text-xl font-bold " + (data.dry_run.conflicts > 0 ? "text-red-400" : "text-green-400")}>{data.dry_run.conflicts}</p></div>
          </div>
        </div>
      )}

      {/* Actions */}
      <div className="flex items-center gap-3 mb-6">
        <button onClick={() => executeMigration(selectedScope)} className="flex items-center gap-2 px-4 py-2 bg-green-600 hover:bg-green-700 rounded-lg text-sm font-medium transition">
          <Play className="w-4 h-4" /> Execute Migration
        </button>
        <button className="flex items-center gap-2 px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg text-sm font-medium transition">
          <RotateCcw className="w-4 h-4" /> Rollback
        </button>
      </div>

      {/* Migration History */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-sm font-semibold mb-3">{t("tenantMigration.history")}</h2>
        <div className="space-y-2">
          {(data?.migration_history ?? []).map((h) => (
            <div key={h.id} className="flex items-center gap-3 bg-gray-800 rounded-lg p-3">
              <span className="text-xs text-gray-500">{h.timestamp}</span>
              <span className="text-sm">{h.scope}</span>
              <span className={"text-xs px-2 py-0.5 rounded ml-auto " + (h.status === "completed" ? "bg-green-900 text-green-300" : "bg-yellow-900 text-yellow-300")}>{h.status}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

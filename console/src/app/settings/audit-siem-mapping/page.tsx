"use client";

import { useAuditSiemMapping } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";
import { ArrowRight, Filter, Tag, Play, Activity, Zap } from "lucide-react";

export default function AuditSiemMappingPage() {
  const t = useTranslations();
  const { data, loading, error, refresh, testMapping } = useAuditSiemMapping();

  if (loading) return <div className="p-8 text-gray-400">Loading SIEM mapping...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">SIEM Field Mapping</h1>
          <p className="text-sm text-gray-400 mt-1">Map audit event fields to SIEM destination schemas</p>
        </div>
        <button
          onClick={refresh}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          Refresh
        </button>
      </div>

      {/* Top Metrics */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-blue-400">
            <ArrowRight className="w-4 h-4" />
            <span className="text-xs text-gray-400">Mapped Fields</span>
          </div>
          <p className="text-2xl font-bold">{data?.per_destination?.[0]?.field_mappings?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-green-400">
            <Filter className="w-4 h-4" />
            <span className="text-xs text-gray-400">Forwarded Event Types</span>
          </div>
          <p className="text-2xl font-bold">{data?.event_type_filter?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-purple-400">
            <Activity className="w-4 h-4" />
            <span className="text-xs text-gray-400">Throughput Estimate</span>
          </div>
          <p className="text-2xl font-bold">{data?.throughput_estimate ?? 0} ev/s</p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Per-Destination Field Mapping */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">Field Mapping (Splunk)</h2>
          <div className="space-y-2">
            {(data?.per_destination ?? []).flatMap((d) => d.field_mappings).map((fm: any, i: number) => (
              <div key={i} className="flex items-center gap-3 bg-gray-800 rounded-lg p-3">
                <code className="text-xs text-blue-400 font-mono flex-1">{fm.local_field}</code>
                <ArrowRight className="w-4 h-4 text-gray-500 flex-shrink-0" />
                <code className="text-xs text-green-400 font-mono flex-1">{fm.siem_field}</code>
              </div>
            ))}
          </div>
        </div>

        <div className="space-y-6">
          {/* Event Type Filter + Severity Mapping */}
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
              <Filter className="w-5 h-5 text-blue-400" />
              Event Type Filter
            </h2>
            <div className="flex flex-wrap gap-2">
              {(data?.event_type_filter ?? []).map((et) => (
                <span key={et} className="text-xs px-2 py-1 rounded bg-blue-900 text-blue-300 font-mono">{et}</span>
              ))}
            </div>
          </div>

          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
              <Tag className="w-5 h-5 text-purple-400" />
              Severity Mapping
            </h2>
            <div className="space-y-2">
              {(data?.severity_mapping ?? []).map((sm: any, i: number) => (
                <div key={i} className="flex items-center gap-3 bg-gray-800 rounded-lg p-3">
                  <span className={"text-xs px-2 py-0.5 rounded " + (
                    sm.our_severity === "critical" ? "bg-red-900 text-red-300" :
                    sm.our_severity === "high" ? "bg-orange-900 text-orange-300" :
                    sm.our_severity === "medium" ? "bg-yellow-900 text-yellow-300" :
                    "bg-blue-900 text-blue-300"
                  )}>
                    {sm.our_severity}
                  </span>
                  <ArrowRight className="w-4 h-4 text-gray-500" />
                  <span className="text-sm font-medium text-green-400">{sm.siem_severity}</span>
                </div>
              ))}
            </div>
          </div>

          {/* Custom Fields + Test */}
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
              <Zap className="w-5 h-5 text-yellow-400" />
              Custom Fields
            </h2>
            <div className="space-y-1 mb-4">
              {(data?.custom_fields ?? []).map((cf: any, i: number) => (
                <div key={i} className="flex items-center justify-between bg-gray-800 rounded px-3 py-1.5">
                  <code className="text-xs text-gray-400">{cf.key}</code>
                  <code className="text-xs text-blue-400">{cf.value}</code>
                </div>
              ))}
            </div>
            <button
              onClick={() => testMapping()}
              className="flex items-center gap-2 px-4 py-2 bg-green-600 hover:bg-green-700 rounded-lg text-sm font-medium transition"
            >
              <Play className="w-4 h-4" />
              Test Mapping
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

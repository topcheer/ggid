"use client";

import { useSiemLogFormats } from "@ggid/sdk-react";
import { FileCode, CheckCircle, AlertTriangle } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function SiemLogFormatsPage() {
  const t = useTranslations();

  const { data, loading, error, refresh } = useSiemLogFormats();

  if (loading) return <div className="p-8 text-gray-400">Loading log formats...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">SIEM Log Formats</h1>
          <p className="text-sm text-gray-400 mt-1">Configure field mappings and severity translation</p>
        </div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      {/* Format configs per destination */}
      <div className="space-y-4">
        {(data?.format_configs ?? []).map((cfg) => (
          <div key={cfg.destination} className="bg-gray-900 rounded-xl p-6">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-semibold flex items-center gap-2">
                <FileCode className="w-4 h-4 text-purple-400" />
                {cfg.destination} ({cfg.template})
              </h3>
              {cfg.validation_passed ? (
                <span className="text-xs px-2 py-0.5 rounded bg-green-900 text-green-300 flex items-center gap-1"><CheckCircle className="w-3 h-3" /> Valid</span>
              ) : (
                <span className="text-xs px-2 py-0.5 rounded bg-red-900 text-red-300 flex items-center gap-1"><AlertTriangle className="w-3 h-3" /> Issues</span>
              )}
            </div>

            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              {/* Field Mapping */}
              <div>
                <h4 className="text-xs text-gray-400 mb-2">Field Mapping</h4>
                <div className="space-y-1">
                  {cfg.field_mapping.map((m) => (
                    <div key={m.local_field} className="flex items-center gap-2 bg-gray-800 rounded p-2">
                      <span className="text-xs font-mono text-blue-400">{m.local_field}</span>
                      <span className="text-gray-600">{" -> "}</span>
                      <span className="text-xs font-mono text-green-400">{m.siem_field}</span>
                    </div>
                  ))}
                </div>
              </div>

              {/* Severity Mapping */}
              <div>
                <h4 className="text-xs text-gray-400 mb-2">Severity Mapping</h4>
                <div className="space-y-1">
                  {cfg.severity_mapping.map((s) => (
                    <div key={s.our_severity} className="flex items-center gap-2 bg-gray-800 rounded p-2">
                      <span className={"text-xs font-bold " + (
                        s.our_severity === "critical" ? "text-red-400" :
                        s.our_severity === "high" ? "text-orange-400" :
                        s.our_severity === "medium" ? "text-yellow-400" : "text-blue-400"
                      )}>{s.our_severity}</span>
                      <span className="text-gray-600">{" -> "}</span>
                      <span className="text-xs font-mono text-gray-300">{s.siem_severity}</span>
                    </div>
                  ))}
                </div>
              </div>
            </div>

            {/* Sample Output */}
            {cfg.sample_output && (
              <div className="mt-4">
                <h4 className="text-xs text-gray-400 mb-1">Sample Output</h4>
                <pre className="bg-gray-800 rounded p-3 text-xs text-gray-300 overflow-x-auto font-mono">{cfg.sample_output}</pre>
              </div>
            )}
          </div>
        ))}
      </div>

      {/* Template Library */}
      <div className="bg-gray-900 rounded-xl p-6 mt-6">
        <h2 className="text-sm font-semibold mb-3">Template Library</h2>
        <div className="flex flex-wrap gap-2">
          {(data?.template_library ?? []).map((t) => (
            <span key={t} className="text-xs px-3 py-1.5 bg-gray-800 rounded-lg text-gray-400">{t}</span>
          ))}
        </div>
      </div>
    </div>
  );
}

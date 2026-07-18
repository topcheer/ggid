"use client";

import { useDataClassificationPolicy } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";
import { Globe, FileText, Lock, ShieldAlert, Database, Sparkles } from "lucide-react";

export default function DataClassificationPolicyPage() {
  const t = useTranslations();
  const { data, loading, error, refresh } = useDataClassificationPolicy();

  if (loading) return <div className="p-8 text-gray-400">Loading data classification policy...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const levelConfig: Record<string, { icon: React.ReactNode; color: string; bg: string }> = {
    public: { icon: <Globe className="w-4 h-4" />, color: "text-green-400", bg: "bg-green-900" },
    internal: { icon: <FileText className="w-4 h-4" />, color: "text-blue-400", bg: "bg-blue-900" },
    confidential: { icon: <Lock className="w-4 h-4" />, color: "text-yellow-400", bg: "bg-yellow-900" },
    restricted: { icon: <ShieldAlert className="w-4 h-4" />, color: "text-red-400", bg: "bg-red-900" },
  };

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Data Classification Policy</h1>
          <p className="text-sm text-gray-400 mt-1">Define data sensitivity levels, handling rules, and PII inventory</p>
        </div>
        <button
          onClick={refresh}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          Refresh
        </button>
      </div>

      {/* Classification Levels */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
        {(data?.levels ?? []).map((level) => {
          const cfg = levelConfig[level.name] ?? levelConfig.internal;
          return (
            <div key={level.id} className="bg-gray-900 rounded-xl p-4">
              <div className={`flex items-center gap-2 mb-2 ${cfg.color}`}>
                {cfg.icon}
                <span className="text-sm font-semibold uppercase">{level.name}</span>
              </div>
              <p className="text-xs text-gray-400 mb-3">{level.description}</p>
              <div className="space-y-1">
                <div className="flex items-center justify-between text-xs">
                  <span className="text-gray-500">Records</span>
                  <span className="font-medium">{level.record_count.toLocaleString()}</span>
                </div>
                <div className="flex items-center justify-between text-xs">
                  <span className="text-gray-500">Handling Rules</span>
                  <span className="font-medium">{level.handling_rules.length}</span>
                </div>
              </div>
            </div>
          );
        })}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Handling Rules + Attribute Mapping */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">Handling Rules by Level</h2>
          <div className="space-y-4">
            {(data?.levels ?? []).map((level) => {
              const cfg = levelConfig[level.name] ?? levelConfig.internal;
              return (
                <div key={level.id}>
                  <div className={`flex items-center gap-2 mb-2 ${cfg.color}`}>
                    <span className="text-sm font-medium uppercase">{level.name}</span>
                  </div>
                  <div className="space-y-1 ml-6">
                    {level.handling_rules.map((rule: any, i: number) => (
                      <div key={i} className="flex items-center gap-2 text-xs text-gray-300">
                        <span className="w-1 h-1 rounded-full bg-gray-500" />
                        {rule}
                      </div>
                    ))}
                  </div>
                </div>
              );
            })}
          </div>

          <div className="mt-6 pt-4 border-t border-gray-800">
            <h3 className="text-sm font-semibold mb-2">Attribute Mapping</h3>
            <div className="space-y-1">
              {(data?.attribute_mapping ?? []).map((m: any, i: number) => (
                <div key={i} className="flex items-center justify-between bg-gray-800 rounded px-3 py-1.5">
                  <span className="text-xs text-gray-300">{m.attribute}</span>
                  <span
                    className={`text-xs px-2 py-0.5 rounded ${
                      levelConfig[m.classification]?.bg ?? "bg-gray-700"
                    } ${levelConfig[m.classification]?.color ?? "text-gray-300"}`}
                  >
                    {m.classification}
                  </span>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* PII Inventory + Auto Classify */}
        <div className="space-y-6">
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
              <Database className="w-5 h-5" />
              PII Inventory
            </h2>
            <div className="space-y-2">
              {(data?.pii_inventory ?? []).map((item: any, i: number) => (
                <div key={i} className="bg-gray-800 rounded-lg p-3">
                  <div className="flex items-center justify-between mb-1">
                    <span className="text-sm font-medium">{item.field}</span>
                    <span className="text-xs text-gray-400">{item.occurrences.toLocaleString()} records</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span
                      className={`text-xs px-2 py-0.5 rounded ${
                        levelConfig[item.classification]?.bg ?? "bg-gray-700"
                      } ${levelConfig[item.classification]?.color ?? "text-gray-300"}`}
                    >
                      {item.classification}
                    </span>
                    {item.masked && (
                      <span className="text-xs px-2 py-0.5 rounded bg-gray-700 text-gray-300">Masked</span>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </div>

          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
              <Sparkles className="w-5 h-5 text-cyan-400" />
              Auto-Classification
            </h2>
            <div className="space-y-2">
              <div className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
                <span className="text-sm">Status</span>
                <span
                  className={`text-xs px-2 py-0.5 rounded ${
                    data?.auto_classify?.enabled ? "bg-green-900 text-green-300" : "bg-gray-700 text-gray-400"
                  }`}
                >
                  {data?.auto_classify?.enabled ? "Enabled" : "Disabled"}
                </span>
              </div>
              <div className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
                <span className="text-sm">Confidence Threshold</span>
                <span className="text-sm font-medium">{Math.round((data?.auto_classify?.confidence_threshold ?? 0) * 100)}%</span>
              </div>
              <div className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
                <span className="text-sm">Fields Classified (24h)</span>
                <span className="text-sm font-medium">{data?.auto_classify?.fields_classified_24h ?? 0}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

"use client";

import { useState } from "react";
import { usePolicyMigrationWizard } from "@ggid/sdk-react";
import { Upload, ArrowRight, CheckCircle, AlertTriangle, XCircle, RotateCcw, FileJson } from "lucide-react";

export default function PolicyMigrationWizardPage() {
  const { data, loading, error, refresh, executeMigration, rollback } = usePolicyMigrationWizard();
  const [step, setStep] = useState(0);

  if (loading) return <div className="p-8 text-gray-400">Loading migration wizard...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const steps = ["Select Source", "Upload File", "Review Mapping", "Validate", "Execute"];

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Policy Migration Wizard</h1>
          <p className="text-sm text-gray-400 mt-1">Migrate policies from external IAM systems</p>
        </div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      {/* Stepper */}
      <div className="flex items-center gap-2 mb-8">
        {steps.map((s, i) => (
          <div key={s} className="flex items-center gap-2">
            <div
              className={"w-8 h-8 rounded-full flex items-center justify-center text-xs font-bold " + (
                i < step ? "bg-green-600 text-white" :
                i === step ? "bg-blue-600 text-white" :
                "bg-gray-800 text-gray-500"
              )}
            >
              {i < step ? <CheckCircle className="w-4 h-4" /> : i + 1}
            </div>
            <span className={"text-xs " + (i <= step ? "text-white" : "text-gray-500")}>{s}</span>
            {i < steps.length - 1 && <div className={"w-8 h-0.5 " + (i < step ? "bg-green-600" : "bg-gray-800")} />}
          </div>
        ))}
      </div>

      {/* Source System */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold mb-3">Source System</h2>
        <div className="flex flex-wrap gap-2">
          {(data?.source_systems ?? []).map((src) => (
            <button
              key={src}
              onClick={() => setStep(1)}
              className={"px-4 py-2 rounded-lg text-sm font-medium transition border " + (
                step >= 1 ? "bg-blue-900 border-blue-600 text-blue-300" : "bg-gray-800 border-gray-700 hover:bg-gray-700"
              )}
            >
              {src}
            </button>
          ))}
        </div>
      </div>

      {/* Upload Area */}
      {step >= 1 && (
        <div className="bg-gray-900 rounded-xl p-6 mb-6">
          <div className="border-2 border-dashed border-gray-700 rounded-lg p-8 text-center">
            <Upload className="w-8 h-8 text-gray-500 mx-auto mb-2" />
            <p className="text-sm text-gray-400">Drop policy export file here or click to browse</p>
            <p className="text-xs text-gray-600 mt-1">Supported: JSON, XML, YAML (max 10MB)</p>
          </div>
        </div>
      )}

      {/* Mapping Preview */}
      {step >= 2 && (
        <div className="bg-gray-900 rounded-xl p-6 mb-6">
          <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
            <ArrowRight className="w-4 h-4 text-blue-400" />
            Mapping Preview
          </h2>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-800 text-gray-400">
                  <th className="text-left py-2 pr-3">Source Rule</th>
                  <th className="text-left py-2 pr-3"></th>
                  <th className="text-left py-2 pr-3">GGID Policy</th>
                  <th className="text-left py-2 pr-3">Confidence</th>
                </tr>
              </thead>
              <tbody>
                {(data?.mapping_preview ?? []).map((m, i) => (
                  <tr key={i} className="border-b border-gray-800">
                    <td className="py-3 pr-3 font-mono text-xs text-gray-400">{m.source_rule}</td>
                    <td className="py-3 pr-3"><ArrowRight className="w-3 h-3 text-gray-500" /></td>
                    <td className="py-3 pr-3 font-mono text-xs text-green-400">{m.ggid_policy}</td>
                    <td className="py-3 pr-3">
                      <span className={"text-xs " + (m.confidence > 0.8 ? "text-green-400" : m.confidence > 0.5 ? "text-yellow-400" : "text-red-400")}>
                        {(m.confidence * 100).toFixed(0)}%
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Validation Report */}
      {step >= 3 && (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
          <div className="bg-gray-900 rounded-xl p-4">
            <CheckCircle className="w-5 h-5 text-green-400 mb-1" />
            <p className="text-xs text-gray-400">Migrated</p>
            <p className="text-xl font-bold text-green-400">{data?.validation_report?.migrated ?? 0}</p>
          </div>
          <div className="bg-gray-900 rounded-xl p-4">
            <AlertTriangle className="w-5 h-5 text-yellow-400 mb-1" />
            <p className="text-xs text-gray-400">Warnings</p>
            <p className="text-xl font-bold text-yellow-400">{data?.validation_report?.warnings ?? 0}</p>
          </div>
          <div className="bg-gray-900 rounded-xl p-4">
            <XCircle className="w-5 h-5 text-red-400 mb-1" />
            <p className="text-xs text-gray-400">Errors</p>
            <p className="text-xl font-bold text-red-400">{data?.validation_report?.errors ?? 0}</p>
          </div>
        </div>
      )}

      {/* Actions */}
      <div className="flex items-center gap-3">
        <button
          onClick={() => { executeMigration(); setStep(4); }}
          className="flex items-center gap-2 px-4 py-2 bg-green-600 hover:bg-green-700 rounded-lg text-sm font-medium transition"
        >
          <CheckCircle className="w-4 h-4" />
          Execute Migration
        </button>
        <button
          onClick={() => rollback()}
          className="flex items-center gap-2 px-4 py-2 bg-red-900 hover:bg-red-800 rounded-lg text-sm font-medium transition"
        >
          <RotateCcw className="w-4 h-4" />
          Rollback
        </button>
      </div>

      {/* Migration History */}
      <div className="bg-gray-900 rounded-xl p-6 mt-6">
        <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
          <FileJson className="w-4 h-4 text-blue-400" />
          Migration History
        </h2>
        <div className="space-y-2">
          {(data?.migration_history ?? []).map((h) => (
            <div key={h.id} className="flex items-center gap-3 bg-gray-800 rounded-lg p-3">
              <span className={"w-2 h-2 rounded-full " + (h.status === "completed" ? "bg-green-500" : h.status === "failed" ? "bg-red-500" : "bg-yellow-500")} />
              <div className="flex-1">
                <p className="text-xs font-medium">{h.source_system} - {h.policies_count} policies</p>
                <p className="text-xs text-gray-500">{h.date} by {h.executed_by}</p>
              </div>
              <span className="text-xs capitalize text-gray-400">{h.status}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

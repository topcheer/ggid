"use client";

import { useTranslations } from "@/lib/i18n";
import { usePolicyImportExport } from "@ggid/sdk-react";
import { Download, Upload, FileCode, CheckCircle } from "lucide-react";

export default function PolicyImportExportPage() {
  const { data, loading, error, refresh } = usePolicyImportExport();
  const t = useTranslations();

  if (loading) return <div className="p-8 text-gray-400">{t("policyImportExport.loading")}</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">{t("policyImportExport.title")}</h1>
          <p className="text-sm text-gray-400 mt-1">{t("policyImportExport.subtitle")}</p>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
        {/* Export Section */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-4 flex items-center gap-2"><Download className="w-4 h-4 text-blue-400" /> {t("policyImportExport.exportPolicies")}</h2>
          <div className="space-y-3">
            <div>
              <p className="text-xs text-gray-500 mb-1">{t("policyImportExport.scope")}</p>
              <select aria-label="Select option" className="w-full px-3 py-2 bg-gray-800 rounded-lg text-sm">
                <option>{t("policyImportExport.allPolicies")} ({data?.total_policies ?? 0})</option>
                <option>{t("policyImportExport.selectedPolicies")}</option>
              </select>
            </div>
            <div>
              <p className="text-xs text-gray-500 mb-1">{t("policyImportExport.format")}</p>
              <div className="flex gap-2">
                <button className="flex-1 px-3 py-2 bg-gray-800 rounded-lg text-xs font-medium">JSON</button>
                <button className="flex-1 px-3 py-2 bg-gray-800 rounded-lg text-xs font-medium">YAML</button>
              </div>
            </div>
            <button aria-label="action" className="w-full px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">{t("policyImportExport.export")}</button>
          </div>
        </div>

        {/* Import Section */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-4 flex items-center gap-2"><Upload className="w-4 h-4 text-green-400" /> {t("policyImportExport.importPolicies")}</h2>
          <div className="space-y-3">
            <div className="border-2 border-dashed border-gray-700 rounded-lg p-8 text-center">
              <FileCode className="w-8 h-8 text-gray-600 mx-auto mb-2" />
              <p className="text-sm text-gray-400">{t("policyImportExport.dropFile")}</p>
              <p className="text-xs text-gray-500 mt-1">{t("policyImportExport.jsonOrYaml")}</p>
            </div>
            <div>
              <p className="text-xs text-gray-500 mb-1">{t("policyImportExport.conflictResolution")}</p>
              <select aria-label="Select option" className="w-full px-3 py-2 bg-gray-800 rounded-lg text-sm">
                <option>Skip</option><option>Overwrite</option><option>Merge</option>
              </select>
            </div>
          </div>
        </div>
      </div>

      {/* Import Log */}
      {data?.import_log && (
        <div className="bg-gray-900 rounded-xl p-6 mb-6">
          <h2 className="text-sm font-semibold mb-3">{t("policyImportExport.lastImportLog")}</h2>
          <div className="grid grid-cols-3 gap-4">
            <div className="bg-gray-800 rounded-lg p-3 text-center"><p className="text-xs text-gray-500">{t("policyImportExport.imported")}</p><p className="text-xl font-bold text-green-400">{data.import_log.imported}</p></div>
            <div className="bg-gray-800 rounded-lg p-3 text-center"><p className="text-xs text-gray-500">{t("policyImportExport.skipped")}</p><p className="text-xl font-bold text-yellow-400">{data.import_log.skipped}</p></div>
            <div className="bg-gray-800 rounded-lg p-3 text-center"><p className="text-xs text-gray-500">{t("policyImportExport.errored")}</p><p className="text-xl font-bold text-red-400">{data.import_log.errored}</p></div>
          </div>
        </div>
      )}

      {/* Template Gallery */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-sm font-semibold mb-3">{t("policyImportExport.templateGallery")}</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          {(data?.template_gallery ?? []).map((t: any) => (
            <div key={t.name} className="bg-gray-800 rounded-lg p-3 flex items-center gap-3">
              <FileCode className="w-4 h-4 text-purple-400" />
              <div className="flex-1"><p className="text-sm font-medium">{t.name}</p><p className="text-xs text-gray-400">{t.description}</p></div>
              {t.compatible && <CheckCircle className="w-4 h-4 text-green-400" />}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

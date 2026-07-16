"use client";

import { useIncidentResponsePlaybook } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";
import { BookOpen, AlertTriangle, CheckCircle, Clock, FileText } from "lucide-react";

export default function IncidentResponsePlaybookPage() {
  const t = useTranslations();
  const { data, loading, error, refresh } = useIncidentResponsePlaybook();

  if (loading) return <div className="p-8 text-gray-400">{t("incidentPlaybook.loading")}</div>;
  if (error) return <div className="p-8 text-red-400">{t("common.error")}: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">{t("incidentPlaybook.title")}</h1>
          <p className="text-sm text-gray-400 mt-1">{t("incidentPlaybook.subtitle")}</p>
        </div>
        <button aria-label="action" onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">{t("incidentPlaybook.refresh")}</button>
      </div>

      {/* Active Incidents */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
          <AlertTriangle className="w-5 h-5 text-red-400" />
          {t("incidentPlaybook.activeIncidents")}
        </h2>
        <div className="space-y-2">
          {(data?.active_incidents ?? []).map((inc) => (
            <div key={inc.incident_id} className="bg-gray-800 rounded-lg p-4">
              <div className="flex items-start justify-between mb-2">
                <div>
                  <p className="text-sm font-semibold">{inc.incident_id}: {inc.type}</p>
                  <p className="text-xs text-gray-400">{t("incidentPlaybook.assignedTo").replace("{name}", inc.assigned_to)}</p>
                </div>
                <div className="flex items-center gap-2">
                  <span className={"text-xs px-2 py-0.5 rounded " + (
                    inc.severity === "critical" ? "bg-red-900 text-red-300" :
                    inc.severity === "high" ? "bg-orange-900 text-orange-300" :
                    "bg-yellow-900 text-yellow-300"
                  )}>{inc.severity}</span>
                  <span className="text-xs text-gray-400">{t("incidentPlaybook.sla").replace("{value}", inc.sla_countdown)}</span>
                </div>
              </div>
              {/* Step Progress */}
              <div className="flex items-center gap-1 mt-2">
                {inc.steps.map((step, i) => (
                  <div key={i} className="flex items-center gap-1">
                    <span className={"text-xs px-2 py-0.5 rounded " + (
                      step.status === "done" ? "bg-green-900 text-green-300" :
                      step.status === "active" ? "bg-blue-900 text-blue-300" :
                      "bg-gray-700 text-gray-400"
                    )}>
                      {step.name}
                    </span>
                    {i < inc.steps.length - 1 && <span className="text-gray-600 text-xs">{" -> "}</span>}
                  </div>
                ))}
              </div>
            </div>
          ))}
          {(data?.active_incidents?.length ?? 0) === 0 && <p className="text-sm text-green-400">{t("incidentPlaybook.noActiveIncidents")}</p>}
        </div>
      </div>

      {/* Playbook Library */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
          <BookOpen className="w-4 h-4 text-blue-400" />
          {t("incidentPlaybook.playbookLibrary")}
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
          {(data?.playbook_library ?? []).map((pb) => (
            <div key={pb.incident_type} className="bg-gray-800 rounded-lg p-3">
              <div className="flex items-center justify-between mb-2">
                <p className="text-sm font-semibold">{pb.incident_type}</p>
                <span className={"text-xs px-2 py-0.5 rounded " + (
                  pb.severity === "critical" ? "bg-red-900 text-red-300" : "bg-orange-900 text-orange-300"
                )}>{pb.severity}</span>
              </div>
              <p className="text-xs text-gray-400 mb-2">{t("incidentPlaybook.stepsAutomated").replace("{steps}", String(pb.steps_count)).replace("{automated}", String(pb.automated_actions_count))}</p>
              <p className="text-xs text-gray-500">{t("incidentPlaybook.escalation").replace("{chain}", pb.escalation_chain.join(" -> "))}</p>
            </div>
          ))}
        </div>
      </div>

      {/* Post Mortem Templates */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-sm font-semibold flex items-center gap-2 mb-3">
          <FileText className="w-4 h-4 text-purple-400" />
          {t("incidentPlaybook.postMortemTemplates")}
        </h2>
        <div className="space-y-1">
          {(data?.post_mortem_templates ?? []).map((tmpl, i) => (
            <div key={i} className="flex items-center justify-between bg-gray-800 rounded-lg p-2">
              <span className="text-sm">{tmpl.template_name}</span>
              <span className="text-xs text-gray-500">{t("incidentPlaybook.sectionsCount").replace("{count}", String(tmpl.sections_count))}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

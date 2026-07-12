"use client";

import { usePolicyGovernanceWorkflow } from "@ggid/sdk-react";
import { GitBranch, Users, Clock, ShieldAlert, CheckCircle, FileText, AlertTriangle } from "lucide-react";

export default function PolicyGovernanceWorkflowPage() {
  const { data, loading, error, refresh } = usePolicyGovernanceWorkflow();

  if (loading) return <div className="p-8 text-gray-400">Loading policy governance workflow...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const pipelineSteps = data?.policy_change_pipeline ?? [];

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Policy Governance Workflow</h1>
          <p className="text-sm text-gray-400 mt-1">Manage policy change pipelines, reviewers, and governance controls</p>
        </div>
        <button
          onClick={refresh}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          Refresh
        </button>
      </div>

      {/* Change Pipeline */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
          <GitBranch className="w-5 h-5 text-blue-400" />
          Policy Change Pipeline
        </h2>
        <div className="flex items-center gap-2 overflow-x-auto pb-2">
          {pipelineSteps.map((step, i) => (
            <div key={i} className="flex items-center gap-2 flex-shrink-0">
              <div className={"flex flex-col items-center gap-1 px-4 py-3 rounded-lg min-w-[120px] " + (
                step.status === "active" ? "bg-blue-600" :
                step.count > 0 ? "bg-gray-800" : "bg-gray-800/50"
              )}>
                <span className="text-2xl font-bold">{step.count}</span>
                <span className="text-xs capitalize">{step.stage}</span>
              </div>
              {i < pipelineSteps.length - 1 && (
                <div className={"w-8 h-px " + (step.status === "active" ? "bg-blue-500" : "bg-gray-700")} />
              )}
            </div>
          ))}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Reviewer Assignment + SoD */}
        <div className="space-y-6">
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
              <Users className="w-5 h-5 text-purple-400" />
              Reviewer Assignment by Category
            </h2>
            <div className="space-y-2">
              {(data?.reviewer_assignment ?? []).map((ra) => (
                <div key={ra.category} className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
                  <span className="text-sm font-medium capitalize">{ra.category}</span>
                  <div className="flex items-center gap-1">
                    {ra.reviewers.map((r) => (
                      <span key={r} className="text-xs px-2 py-0.5 rounded bg-purple-900 text-purple-300">{r}</span>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          </div>

          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
              <ShieldAlert className="w-5 h-5 text-green-400" />
              Segregation of Duties
            </h2>
            <div className="space-y-2">
              <div className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
                <span className="text-sm text-gray-300">Enforced</span>
                <span
                  className={"text-xs px-2 py-0.5 rounded " + (
                    data?.segregation_of_duties?.enforced ? "bg-green-900 text-green-300" : "bg-gray-700 text-gray-400"
                  )}
                >
                  {data?.segregation_of_duties?.enforced ? "Yes" : "No"}
                </span>
              </div>
              <p className="text-xs text-gray-400 px-1">{data?.segregation_of_duties?.description ?? ""}</p>
            </div>
          </div>
        </div>

        <div className="space-y-6">
          {/* Change Freeze Windows */}
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
              <Clock className="w-5 h-5 text-yellow-400" />
              Change Freeze Windows
            </h2>
            <div className="space-y-2">
              {(data?.change_freeze_windows ?? []).map((fw, i) => (
                <div key={i} className="bg-gray-800 rounded-lg p-3">
                  <div className="flex items-center justify-between mb-1">
                    <p className="text-sm font-medium">{fw.name}</p>
                    <span
                      className={"text-xs px-2 py-0.5 rounded " + (
                        fw.active ? "bg-red-900 text-red-300" : "bg-gray-700 text-gray-400"
                      )}
                    >
                      {fw.active ? "Active" : "Scheduled"}
                    </span>
                  </div>
                  <p className="text-xs text-gray-400">{fw.start} - {fw.end}</p>
                  <p className="text-xs text-gray-500 mt-1">{fw.reason}</p>
                </div>
              ))}
            </div>
          </div>

          {/* Emergency Bypass */}
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
              <AlertTriangle className="w-5 h-5 text-red-400" />
              Emergency Bypass
            </h2>
            <div className="space-y-2">
              <div className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
                <span className="text-sm text-gray-300">Allowed</span>
                <span
                  className={"text-xs px-2 py-0.5 rounded " + (
                    data?.emergency_bypass?.allowed ? "bg-yellow-900 text-yellow-300" : "bg-gray-700 text-gray-400"
                  )}
                >
                  {data?.emergency_bypass?.allowed ? "Yes" : "No"}
                </span>
              </div>
              <div className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
                <span className="text-sm text-gray-300">Requires Approval</span>
                <span
                  className={"text-xs px-2 py-0.5 rounded " + (
                    data?.emergency_bypass?.requires_approval ? "bg-red-900 text-red-300" : "bg-gray-700 text-gray-400"
                  )}
                >
                  {data?.emergency_bypass?.requires_approval ? "Yes" : "No"}
                </span>
              </div>
              <p className="text-xs text-gray-400">Approvers: {(data?.emergency_bypass?.approvers ?? []).join(", ")}</p>
            </div>
          </div>
        </div>
      </div>

      {/* Governance Audit Trail */}
      <div className="bg-gray-900 rounded-xl p-6 mt-6">
        <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
          <FileText className="w-5 h-5 text-blue-400" />
          Governance Audit Trail
        </h2>
        <div className="space-y-2 max-h-64 overflow-y-auto">
          {(data?.governance_audit_trail ?? []).map((entry, i) => (
            <div key={i} className="flex items-center gap-3 bg-gray-800 rounded-lg p-3">
              {entry.action === "approved" ? <CheckCircle className="w-4 h-4 text-green-400 flex-shrink-0" /> :
               entry.action === "rejected" ? <AlertTriangle className="w-4 h-4 text-red-400 flex-shrink-0" /> :
               <FileText className="w-4 h-4 text-blue-400 flex-shrink-0" />}
              <div className="flex-1">
                <p className="text-sm font-medium">{entry.policy_name}</p>
                <p className="text-xs text-gray-400">{entry.action} by {entry.actor} - {entry.timestamp}</p>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

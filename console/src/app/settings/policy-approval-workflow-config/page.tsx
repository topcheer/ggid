"use client";

import { usePolicyApprovalWorkflowConfig } from "@ggid/sdk-react";
import { GitBranch, Snowflake, ShieldCheck } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function PolicyApprovalWorkflowConfigPage() {
  const t = useTranslations();

  const { data, loading, error, refresh } = usePolicyApprovalWorkflowConfig();
  if (loading) return <div className="p-8 text-gray-400">Loading...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div><h1 className="text-2xl font-bold">Policy Approval Workflow</h1><p className="text-sm text-gray-400 mt-1">Governance pipeline configuration</p></div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Save</button>
      </div>

      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold mb-4 flex items-center gap-2"><GitBranch className="w-4 h-4 text-blue-400" /> Pipeline</h2>
        <div className="flex items-center gap-2">{(data?.pipeline ?? []).map((stage, i) => (
          <div key={stage.name} className="flex items-center gap-2">
            <div className={"px-3 py-2 rounded-lg text-sm " + (stage.enabled ? "bg-blue-900 text-blue-300" : "bg-gray-800 text-gray-500")}><p className="font-medium">{stage.name}</p><p className="text-xs text-gray-400">{stage.assignee}</p></div>
            {i < (data?.pipeline?.length ?? 0) - 1 && <span className="text-gray-600">{" -> "}</span>}
          </div>
        ))}</div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-3">Reviewer Assignment</h2>
          <div className="space-y-1">
            {(data?.reviewers ?? []).map((r) => (
              <div key={r.category} className="flex items-center gap-2 bg-gray-800 rounded p-2 text-xs">
                <span className="flex-1">{r.category}</span>
                <span className="text-blue-400">{r.reviewer}</span>
              </div>
            ))}
          </div>
        </div>
        <div className="space-y-6">
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-sm font-semibold mb-3 flex items-center gap-2"><Snowflake className="w-4 h-4 text-cyan-400" /> Change Freeze Windows</h2>
            <div className="space-y-1">
              {(data?.freeze_windows ?? []).map((f) => (
                <div key={f.name} className="text-xs bg-gray-800 rounded p-2">
                  <span className="font-medium">{f.name}</span>{" - "}<span className="text-gray-400">{f.period}</span>
                </div>
              ))}
            </div>
          </div>
          <div className="bg-gray-900 rounded-xl p-6 space-y-2">
            <h2 className="text-sm font-semibold mb-3 flex items-center gap-2"><ShieldCheck className="w-4 h-4 text-green-400" /> Controls</h2>
            <label className="flex items-center gap-2 text-sm"><input type="checkbox" defaultChecked={data?.sod_enforced} /> Enforce Segregation of Duties</label>
            <label className="flex items-center gap-2 text-sm"><input type="checkbox" defaultChecked={data?.emergency_bypass_enabled} /> Emergency bypass (requires C-level)</label>
          </div>
        </div>
      </div>
    </div>
  );
}

"use client";

import { useAccessRequestApprovalWorkflow } from "@ggid/sdk-react";
import { Clock, CheckCircle, XCircle, ChevronRight, Zap, User } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function AccessRequestApprovalWorkflowPage() {
  const t = useTranslations();

  const { data, loading, error, refresh, approve, reject } = useAccessRequestApprovalWorkflow();

  if (loading) return <div className="p-8 text-gray-400">Loading access approval workflow...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold"> {t("backend3.accessRequestApproval.title")}</h1>
          <p className="text-sm text-gray-400 mt-1">{t("backend3.accessRequestApproval.subtitle")}</p>
        </div>
        <button
          onClick={refresh}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          Refresh
        </button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Pending Requests */}
        <div className="lg:col-span-2 space-y-4">
          <h2 className="text-lg font-semibold flex items-center gap-2">
            <Clock className="w-5 h-5" />
            Pending Requests ({data?.pending_requests?.length ?? 0})
          </h2>

          {(data?.pending_requests ?? []).map((req: any) => {
            const slaExpired = req.sla_remaining_hours <= 0;
            const slaWarning = req.sla_remaining_hours <= 4 && !slaExpired;
            return (
              <div key={req.id} className="bg-gray-900 rounded-xl p-5 border border-gray-800">
                <div className="flex items-start justify-between mb-3">
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded-full bg-gray-700 flex items-center justify-center">
                      <User className="w-5 h-5 text-gray-400" />
                    </div>
                    <div>
                      <p className="font-semibold">{req.requester_name}</p>
                      <p className="text-xs text-gray-400">
                        Requesting: <span className="text-blue-400">{req.requested_role}</span>
                      </p>
                    </div>
                  </div>
                  <div className="text-right">
                    <p className="text-xs text-gray-500">{t("backend3.accessRequestApproval.slaRemaining")}</p>
                    <p
                      className={`text-sm font-bold ${
                        slaExpired
                          ? "text-red-400"
                          : slaWarning
                          ? "text-yellow-400"
                          : "text-green-400"
                      }`}
                    >
                      {slaExpired ? "Expired" : `${req.sla_remaining_hours}h`}
                    </p>
                  </div>
                </div>

                {/* Approval Chain Visual */}
                <div className="flex items-center gap-1 mb-4 overflow-x-auto pb-2">
                  {req.approval_chain.map((step: any, idx: number) => (
                    <div key={idx} className="flex items-center gap-1 flex-shrink-0">
                      <div
                        className={`flex items-center gap-1.5 px-2 py-1 rounded-md text-xs ${
                          step.status === "approved"
                            ? "bg-green-900 text-green-300"
                            : step.status === "pending"
                            ? "bg-yellow-900 text-yellow-300"
                            : step.status === "rejected"
                            ? "bg-red-900 text-red-300"
                            : "bg-gray-800 text-gray-500"
                        }`}
                      >
                        {step.status === "approved" && <CheckCircle className="w-3 h-3" />}
                        {step.status === "rejected" && <XCircle className="w-3 h-3" />}
                        <span>{step.role}</span>
                      </div>
                      {idx < req.approval_chain.length - 1 && (
                        <ChevronRight className="w-3 h-3 text-gray-600 flex-shrink-0" />
                      )}
                    </div>
                  ))}
                </div>

                {req.auto_approve_eligible && (
                  <div className="flex items-center gap-1 text-xs text-cyan-400 mb-3">
                    <Zap className="w-3 h-3" />
                    Eligible for auto-approval
                  </div>
                )}

                <div className="flex gap-2">
                  <button
                    onClick={() => approve(req.id)}
                    className="flex items-center gap-1 px-3 py-1.5 bg-green-600 hover:bg-green-700 rounded-md text-sm font-medium transition"
                  >
                    <CheckCircle className="w-4 h-4" />
                    Approve
                  </button>
                  <button
                    onClick={() => reject(req.id)}
                    className="flex items-center gap-1 px-3 py-1.5 bg-red-600 hover:bg-red-700 rounded-md text-sm font-medium transition"
                  >
                    <XCircle className="w-4 h-4" />
                    Reject
                  </button>
                </div>
              </div>
            );
          })}

          {(data?.pending_requests ?? []).length === 0 && (
            <div className="bg-gray-900 rounded-xl p-12 text-center text-gray-500">
              No pending requests.
            </div>
          )}
        </div>

        {/* Auto-Approve Rules Sidebar */}
        <div>
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
            <Zap className="w-5 h-5 text-cyan-400" />
            Auto-Approve Rules
          </h2>
          <div className="space-y-2">
            {(data?.auto_approve_rules ?? []).map((rule: any) => (
              <div key={rule.id} className="bg-gray-900 rounded-lg p-3 border border-gray-800">
                <div className="flex items-center justify-between mb-1">
                  <p className="text-sm font-medium">{rule.name}</p>
                  <span
                    className={`text-xs px-1.5 py-0.5 rounded ${
                      rule.enabled ? "bg-green-900 text-green-300" : "bg-gray-700 text-gray-400"
                    }`}
                  >
                    {rule.enabled ? "ON" : "OFF"}
                  </span>
                </div>
                <p className="text-xs text-gray-400">{rule.condition}</p>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

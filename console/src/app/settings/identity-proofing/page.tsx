"use client";

import { useState } from "react";
import { useIdentityProofing } from "@ggid/sdk-react";
import { CheckCircle, Clock, XCircle, Upload, ScanFace, FileQuestion, UserCheck, TrendingUp } from "lucide-react";

export default function IdentityProofingPage() {
  const { data, loading, error, refresh } = useIdentityProofing();
  const [docType, setDocType] = useState("passport");
  const [provider, setProvider] = useState("onfido");

  if (loading) return <div className="p-8 text-gray-400">Loading identity proofing...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const stepIcons: Record<string, React.ReactNode> = {
    document_upload: <Upload className="w-5 h-5" />,
    liveness_check: <ScanFace className="w-5 h-5" />,
    kba: <FileQuestion className="w-5 h-5" />,
    manual_review: <UserCheck className="w-5 h-5" />,
  };

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Identity Proofing</h1>
          <p className="text-sm text-gray-400 mt-1">Configure and monitor identity verification workflows</p>
        </div>
        <button
          onClick={refresh}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          Refresh
        </button>
      </div>

      {/* Completion Rate + Config */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm text-gray-400 mb-2">Completion Rate</h2>
          <p className="text-4xl font-bold text-green-400">{data?.completion_rate ?? 0}%</p>
          <div className="mt-3 w-full bg-gray-700 rounded-full h-2">
            <div className="bg-green-500 rounded-full h-2" style={{ width: `${data?.completion_rate ?? 0}%` }} />
          </div>
        </div>
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm text-gray-400 mb-2">Confidence Threshold</h2>
          <p className="text-4xl font-bold text-blue-400">{Math.round((data?.confidence_threshold ?? 0) * 100)}%</p>
          <p className="text-xs text-gray-400 mt-2">Minimum verification confidence to pass</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm text-gray-400 mb-2">In Progress</h2>
          <p className="text-4xl font-bold text-yellow-400">{data?.in_progress_count ?? 0}</p>
          <p className="text-xs text-gray-400 mt-2">Users currently being verified</p>
        </div>
      </div>

      {/* Config Selectors */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold mb-4">Configuration</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label className="text-xs text-gray-400 mb-1 block">Document Type</label>
            <select
              value={docType}
              onChange={(e) => setDocType(e.target.value)}
              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
            >
              <option value="passport">Passport</option>
              <option value="license">Driver License</option>
              <option value="id_card">National ID Card</option>
              <option value="residence_permit">Residence Permit</option>
            </select>
          </div>
          <div>
            <label className="text-xs text-gray-400 mb-1 block">Verification Provider</label>
            <select
              value={provider}
              onChange={(e) => setProvider(e.target.value)}
              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
            >
              <option value="onfido">Onfido</option>
              <option value="jumio">Jumio</option>
              <option value="persona">Persona</option>
              <option value="veriff">Veriff</option>
              <option value="internal">Internal (Manual)</option>
            </select>
          </div>
        </div>
      </div>

      {/* Proofing Steps */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold mb-4">Proofing Steps</h2>
        <div className="space-y-3">
          {(data?.proofing_steps ?? []).map((step) => (
            <div key={step.step} className="flex items-center gap-4 bg-gray-800 rounded-lg p-4">
              <div className={"w-10 h-10 rounded-lg flex items-center justify-center " + (
                step.status === "completed" ? "bg-green-900 text-green-300" :
                step.status === "in_progress" ? "bg-yellow-900 text-yellow-300" :
                step.status === "failed" ? "bg-red-900 text-red-300" :
                "bg-gray-700 text-gray-400"
              )}>
                {stepIcons[step.step] ?? <Clock className="w-5 h-5" />}
              </div>
              <div className="flex-1">
                <div className="flex items-center gap-2">
                  <p className="text-sm font-semibold">{step.step.replace(/_/g, " ")}</p>
                  <span className={"text-xs px-2 py-0.5 rounded " + (
                    step.status === "completed" ? "bg-green-900 text-green-300" :
                    step.status === "in_progress" ? "bg-yellow-900 text-yellow-300" :
                    step.status === "failed" ? "bg-red-900 text-red-300" :
                    "bg-gray-700 text-gray-400"
                  )}>
                    {step.status}
                  </span>
                </div>
                <p className="text-xs text-gray-400 mt-1">{step.description}</p>
              </div>
              {step.confidence !== undefined && (
                <div className="text-right">
                  <p className="text-xs text-gray-500">Confidence</p>
                  <p className={"text-sm font-bold " + (step.confidence >= (data?.confidence_threshold ?? 0.8) ? "text-green-400" : "text-yellow-400")}>
                    {Math.round(step.confidence * 100)}%
                  </p>
                </div>
              )}
            </div>
          ))}
        </div>
      </div>

      {/* Recent Verifications */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-lg font-semibold mb-4">Recent Verifications</h2>
        <div className="space-y-2">
          {(data?.recent_verifications ?? []).map((v, i) => (
            <div key={i} className="flex items-center gap-3 bg-gray-800 rounded-lg p-3">
              {v.status === "approved" ? <CheckCircle className="w-4 h-4 text-green-400" /> :
               v.status === "rejected" ? <XCircle className="w-4 h-4 text-red-400" /> :
               <Clock className="w-4 h-4 text-yellow-400" />}
              <div className="flex-1">
                <p className="text-sm font-medium">{v.user_name}</p>
                <p className="text-xs text-gray-400">{v.document_type} - {v.timestamp}</p>
              </div>
              <div className="flex items-center gap-2">
                <TrendingUp className="w-3 h-3 text-gray-400" />
                <span className="text-sm font-medium">{Math.round(v.confidence * 100)}%</span>
              </div>
              <span className={"text-xs px-2 py-0.5 rounded " + (
                v.status === "approved" ? "bg-green-900 text-green-300" :
                v.status === "rejected" ? "bg-red-900 text-red-300" :
                "bg-yellow-900 text-yellow-300"
              )}>
                {v.status}
              </span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

"use client";
import { useTranslations } from "@/lib/i18n";

import { useAuthStepUpOrchestrator } from "@ggid/sdk-react";
import { Zap, CheckCircle, AlertTriangle, ArrowRight, Clock } from "lucide-react";

export default function AuthStepUpOrchestratorPage() {
  const t = useTranslations();
  const { data, loading, error, refresh } = useAuthStepUpOrchestrator();

  if (loading) return <div className="p-8 text-gray-400">Loading step-up orchestrator...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Step-Up Authentication Orchestrator</h1>
          <p className="text-sm text-gray-400 mt-1">Manage multi-factor step-up flows and active challenges</p>
        </div>
        <button
          onClick={refresh}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          Refresh
        </button>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-blue-400">
            <Zap className="w-4 h-4" />
            <span className="text-xs text-gray-400">Step-Up Flows</span>
          </div>
          <p className="text-2xl font-bold">{data?.step_up_flows?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-yellow-400">
            <Clock className="w-4 h-4" />
            <span className="text-xs text-gray-400">Active Challenges</span>
          </div>
          <p className="text-2xl font-bold">{data?.active_challenges?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-green-400">
            <CheckCircle className="w-4 h-4" />
            <span className="text-xs text-gray-400">Avg Success Rate</span>
          </div>
          <p className="text-2xl font-bold text-green-400">{data?.avg_success_rate_pct ?? 0}%</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-red-400">
            <AlertTriangle className="w-4 h-4" />
            <span className="text-xs text-gray-400">Timeout Policy</span>
          </div>
          <p className="text-sm font-bold">{data?.challenge_timeout_policy ?? "expire"}</p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Step-Up Flows */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">Step-Up Flows</h2>
          <div className="space-y-2">
            {(data?.step_up_flows ?? []).map((f, i) => (
              <div key={i} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-center justify-between mb-2">
                  <p className="text-sm font-medium capitalize">{f.trigger_action.replace(/_/g, " ")}</p>
                  <span className="text-xs text-gray-400">{f.success_rate_pct}% success</span>
                </div>
                <div className="flex items-center gap-2 mb-1">
                  <span className="text-xs text-gray-400">Factors:</span>
                  {f.required_factors.map((fac) => (
                    <span key={fac} className="text-xs px-1.5 py-0.5 bg-gray-700 rounded">{fac}</span>
                  ))}
                </div>
                <div className="flex items-center gap-3 text-xs text-gray-500">
                  <span>Max attempts: {f.max_attempts}</span>
                  <span>Timeout: {f.timeout_seconds}s</span>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Active Challenges */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
            <Clock className="w-5 h-5 text-yellow-400" />
            Active Challenges
          </h2>
          <div className="space-y-2 max-h-64 overflow-y-auto">
            {(data?.active_challenges ?? []).map((c) => (
              <div key={c.id} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-center justify-between mb-1">
                  <p className="text-sm font-medium">{c.user}</p>
                  <span className="text-xs px-2 py-0.5 bg-gray-700 rounded">{c.challenge_type}</span>
                </div>
                <div className="flex items-center justify-between text-xs text-gray-400">
                  <span>Started: {c.started_at}</span>
                  <span className={c.expires_in_seconds < 30 ? "text-red-400" : "text-gray-400"}>
                    Expires in {c.expires_in_seconds}s
                  </span>
                </div>
              </div>
            ))}
            {(data?.active_challenges ?? []).length === 0 && (
              <p className="text-sm text-gray-500 text-center py-4">No active challenges.</p>
            )}
          </div>
        </div>
      </div>

      {/* Fallback Chain Visual */}
      <div className="bg-gray-900 rounded-xl p-6 mt-6">
        <h2 className="text-lg font-semibold mb-4">Fallback Chain</h2>
        <div className="flex items-center gap-2 flex-wrap">
          {[
            "1. WebAuthn",
            "2. TOTP (if no key)",
            "3. SMS OTP (if no authenticator)",
            "4. Email Link (last resort)",
          ].map((step, i) => (
            <div key={i} className="flex items-center gap-2">
              <span className="text-xs px-3 py-1.5 bg-gray-800 rounded-lg border border-gray-700">{step}</span>
              {i < 3 && <ArrowRight className="w-3 h-3 text-gray-500" />}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

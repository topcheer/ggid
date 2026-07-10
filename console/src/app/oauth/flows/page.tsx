"use client";

import { useEffect, useState, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  ArrowDown, ArrowRight, User, Server, Key, CheckCircle2,
  Clock, RefreshCw, Activity, Zap,
} from "lucide-react";

interface OAuthClient {
  id: string;
  name: string;
  client_id?: string;
  client_secret?: string;
  redirect_uris?: string[];
  grant_types?: string[];
  scopes?: string[];
}

interface FlowStep {
  num: number;
  name: string;
  actor: string;
  description: string;
  icon: React.ElementType;
}

interface FlowRecord {
  id: string;
  client: string;
  startedAt: string;
  status: "success" | "active" | "failed" | "pending";
  duration: number;
  currentStep?: number;
}

const FLOW_STEPS: FlowStep[] = [
  { num: 1, name: "Authorization Request", actor: "User", description: "User initiates OAuth login, redirected to authorization endpoint", icon: User },
  { num: 2, name: "User Consent", actor: "Auth Server", description: "Authorization server displays consent screen for requested scopes", icon: Server },
  { num: 3, name: "Authorization Code", actor: "User", description: "After consent, auth server redirects back with authorization code", icon: CheckCircle2 },
  { num: 4, name: "Token Exchange", actor: "Client", description: "Client exchanges authorization code for tokens at token endpoint", icon: Key },
  { num: 5, name: "Access Token + Refresh Token", actor: "Auth Server", description: "Server validates code and issues access + refresh tokens", icon: Server },
  { num: 6, name: "Resource API Call", actor: "Client", description: "Client uses access token to call protected resource APIs", icon: ArrowRight },
];

const STEP_STATUS = {
  success: { badge: "bg-green-100 text-green-700", border: "border-green-300", icon: CheckCircle2, label: "Success" },
  active:  { badge: "bg-blue-100 text-blue-700 animate-pulse", border: "border-blue-400", icon: Activity, label: "Active" },
  pending: { badge: "bg-gray-100 text-gray-500", border: "border-gray-200", icon: Clock, label: "Pending" },
  failed:  { badge: "bg-red-100 text-red-700", border: "border-red-300", icon: Clock, label: "Failed" },
};

export default function OAuthFlowsPage() {
  const { apiFetch } = useApi();
  const [clients, setClients] = useState<OAuthClient[]>([]);
  const [selectedClient, setSelectedClient] = useState<string>("");
  const [loading, setLoading] = useState(true);
  const [currentStep, setCurrentStep] = useState<number | null>(null);
  const [flowHistory, setFlowHistory] = useState<FlowRecord[]>([]);
  const [tokenDetails, setTokenDetails] = useState<{
    accessTokenLifetime: number;
    refreshTokenLifetime: number;
    scopes: string[];
    tokenType: string;
  } | null>(null);

  // Fetch clients
  useEffect(() => {
    const fetchClients = async () => {
      setLoading(true);
      try {
        const data = await apiFetch<{ clients?: OAuthClient[] } | OAuthClient[]>("/api/v1/oauth/clients");
        const list = Array.isArray(data) ? data : data.clients || [];
        setClients(list);
        if (list.length > 0 && !selectedClient) {
          setSelectedClient(list[0].id);
        }
      } catch {
        setClients([]);
      } finally {
        setLoading(false);
      }
    };
    fetchClients();
  }, [apiFetch]); // eslint-disable-line react-hooks/exhaustive-deps

  // Fetch flow history for selected client
  const loadFlowHistory = useCallback(async () => {
    if (!selectedClient) return;
    try {
      const data = await apiFetch<{ flows?: FlowRecord[] } | FlowRecord[]>(
        `/api/v1/oauth/clients/${selectedClient}/flows`,
      ).catch(() => null);

      if (data) {
        const list = Array.isArray(data) ? data : data.flows || [];
        setFlowHistory(list);
        // Check for active flow
        const active = list.find((f) => f.status === "active");
        if (active?.currentStep) {
          setCurrentStep(active.currentStep);
        } else {
          setCurrentStep(null);
        }
      } else {
        setFlowHistory([]);
        setCurrentStep(null);
      }

      // Fetch token details
      const details = await apiFetch<Record<string, unknown>>(
        `/api/v1/oauth/clients/${selectedClient}`,
      ).catch(() => null);

      if (details) {
        setTokenDetails({
          accessTokenLifetime: Number(details.access_token_lifetime) || 3600,
          refreshTokenLifetime: Number(details.refresh_token_lifetime) || 604800,
          scopes: (details.scopes as string[]) || ["openid", "profile", "email"],
          tokenType: (details.token_type as string) || "Bearer",
        });
      } else {
        setTokenDetails(null);
      }
    } catch {
      setFlowHistory([]);
      setCurrentStep(null);
    }
  }, [apiFetch, selectedClient]);

  useEffect(() => {
    loadFlowHistory();
    // Poll for updates every 5s if a flow is active
    const interval = setInterval(() => {
      if (currentStep !== null) loadFlowHistory();
    }, 5000);
    return () => clearInterval(interval);
  }, [loadFlowHistory, currentStep]);

  const getStepStatus = (stepNum: number): keyof typeof STEP_STATUS => {
    if (currentStep === null) {
      // No active flow — all steps show as success (last completed) or pending
      return stepNum <= 6 ? "success" : "pending";
    }
    if (stepNum < currentStep) return "success";
    if (stepNum === currentStep) return "active";
    return "pending";
  };

  const inputCls = "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";
  const cardCls = "rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const headingCls = "mb-4 text-lg font-semibold text-gray-900 dark:text-gray-100";

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-gray-100">
          <Zap className="h-7 w-7 text-brand-600" />
          OAuth Flow Visualizer
        </h1>
        <button
          onClick={loadFlowHistory}
          className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600"
        >
          <RefreshCw className="h-4 w-4" /> Refresh
        </button>
      </div>

      {/* Client Selector */}
      <div className="mb-6">
        <label className="mb-1 block text-xs font-medium text-gray-500">Select OAuth Client</label>
        {loading ? (
          <div className="animate-pulse rounded-lg bg-gray-200 dark:bg-gray-700" style={{ height: 38, width: 300 }} />
        ) : clients.length === 0 ? (
          <p className="text-sm text-gray-400">No OAuth clients found. Configure clients in OAuth settings.</p>
        ) : (
          <select
            value={selectedClient}
            onChange={(e) => setSelectedClient(e.target.value)}
            className={`${inputCls} max-w-md`}
          >
            {clients.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name || c.client_id || c.id}
              </option>
            ))}
          </select>
        )}
      </div>

      {selectedClient && (
        <div className="space-y-6">
          {/* Flow Diagram */}
          <div className={cardCls}>
            <h2 className={headingCls}>
              <Activity className="mr-2 inline h-5 w-5 text-brand-600" />
              Authorization Code Flow
              {currentStep !== null && (
                <span className="ml-3 inline-flex items-center gap-1.5 rounded-full bg-blue-100 px-3 py-1 text-sm text-blue-700">
                  <span className="h-2 w-2 animate-ping rounded-full bg-blue-500" />
                  Flow in progress
                </span>
              )}
            </h2>

            <div className="space-y-0">
              {FLOW_STEPS.map((step, idx) => {
                const status = getStepStatus(step.num);
                const statusCfg = STEP_STATUS[status];
                const isLast = idx === FLOW_STEPS.length - 1;

                return (
                  <div key={step.num}>
                    {/* Step Card */}
                    <div
                      className={`relative flex items-start gap-4 rounded-xl border-2 p-4 transition-all ${
                        status === "active"
                          ? `${statusCfg.border} bg-blue-50 dark:bg-blue-900/20 shadow-md`
                          : status === "success"
                            ? `${statusCfg.border} bg-white dark:bg-gray-800`
                            : `${statusCfg.border} bg-gray-50 opacity-60 dark:bg-gray-900`
                      }`}
                    >
                      {/* Step Number */}
                      <div
                        className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-full font-bold ${
                          status === "active"
                            ? "bg-blue-500 text-white animate-pulse"
                            : status === "success"
                              ? "bg-green-500 text-white"
                              : "bg-gray-200 text-gray-500 dark:bg-gray-700"
                        }`}
                      >
                        {step.num}
                      </div>

                      {/* Content */}
                      <div className="flex-1">
                        <div className="flex items-center justify-between">
                          <div className="flex items-center gap-2">
                            <step.icon className="h-4 w-4 text-gray-500 dark:text-gray-400" />
                            <span className="font-medium text-gray-900 dark:text-gray-100">
                              {step.name}
                            </span>
                          </div>
                          <div className="flex items-center gap-2">
                            <span className="text-xs text-gray-400">{step.actor}</span>
                            <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${statusCfg.badge}`}>
                              {statusCfg.label}
                            </span>
                          </div>
                        </div>
                        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                          {step.description}
                        </p>
                        {status === "success" && (
                          <p className="mt-1 flex items-center gap-1 text-xs text-green-600">
                            <CheckCircle2 className="h-3 w-3" />
                            Completed
                          </p>
                        )}
                        {status === "active" && (
                          <p className="mt-1 flex items-center gap-1 text-xs text-blue-600">
                            <Clock className="h-3 w-3 animate-spin" />
                            In progress...
                          </p>
                        )}
                      </div>
                    </div>

                    {/* Connector Arrow */}
                    {!isLast && (
                      <div className="flex justify-center py-1">
                        <ArrowDown
                          className={`h-5 w-5 ${
                            status === "success" || status === "active"
                              ? "text-gray-400"
                              : "text-gray-200 dark:text-gray-700"
                          }`}
                        />
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          </div>

          {/* Token Details Panel */}
          <div className="grid gap-6 lg:grid-cols-2">
            <div className={cardCls}>
              <h2 className={headingCls}>
                <Key className="mr-2 inline h-5 w-5 text-brand-600" />
                Token Details
              </h2>
              {tokenDetails ? (
                <div className="space-y-3">
                  <div className="flex items-center justify-between border-b border-gray-100 pb-3 dark:border-gray-700">
                    <span className="text-sm text-gray-500">Access Token Lifetime</span>
                    <span className="text-sm font-medium text-gray-900 dark:text-gray-100">
                      {tokenDetails.accessTokenLifetime}s ({(tokenDetails.accessTokenLifetime / 60).toFixed(0)} min)
                    </span>
                  </div>
                  <div className="flex items-center justify-between border-b border-gray-100 pb-3 dark:border-gray-700">
                    <span className="text-sm text-gray-500">Refresh Token Lifetime</span>
                    <span className="text-sm font-medium text-gray-900 dark:text-gray-100">
                      {tokenDetails.refreshTokenLifetime}s ({(tokenDetails.refreshTokenLifetime / 86400).toFixed(1)} days)
                    </span>
                  </div>
                  <div className="flex items-center justify-between border-b border-gray-100 pb-3 dark:border-gray-700">
                    <span className="text-sm text-gray-500">Token Type</span>
                    <span className="font-mono text-sm font-medium text-gray-900 dark:text-gray-100">
                      {tokenDetails.tokenType}
                    </span>
                  </div>
                  <div>
                    <span className="mb-2 block text-sm text-gray-500">Granted Scopes</span>
                    <div className="flex flex-wrap gap-2">
                      {tokenDetails.scopes.map((scope) => (
                        <span
                          key={scope}
                          className="rounded-full bg-brand-100 px-3 py-1 text-xs font-medium text-brand-700 dark:bg-brand-900/30 dark:text-brand-400"
                        >
                          {scope}
                        </span>
                      ))}
                    </div>
                  </div>
                </div>
              ) : (
                <p className="text-sm text-gray-400">Token details not available for this client.</p>
              )}
            </div>

            {/* Flow History Table */}
            <div className={cardCls}>
              <h2 className={headingCls}>
                <Clock className="mr-2 inline h-5 w-5 text-brand-600" />
                Flow History
              </h2>
              {flowHistory.length === 0 ? (
                <p className="text-sm text-gray-400">No recent flows recorded for this client.</p>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full">
                    <thead>
                      <tr className="border-b border-gray-100 dark:border-gray-700">
                        <th className="px-2 py-2 text-left text-xs font-medium text-gray-500">Started</th>
                        <th className="px-2 py-2 text-left text-xs font-medium text-gray-500">Client</th>
                        <th className="px-2 py-2 text-left text-xs font-medium text-gray-500">Status</th>
                        <th className="px-2 py-2 text-right text-xs font-medium text-gray-500">Duration</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-50 dark:divide-gray-700/50">
                      {flowHistory.slice(0, 10).map((flow) => (
                        <tr key={flow.id} className="hover:bg-gray-50 dark:hover:bg-gray-700/50">
                          <td className="px-2 py-2 text-xs text-gray-500">
                            {flow.startedAt
                              ? new Date(flow.startedAt).toLocaleString("en-US", {
                                  month: "short", day: "numeric", hour: "2-digit", minute: "2-digit",
                                })
                              : "-"}
                          </td>
                          <td className="px-2 py-2 text-sm text-gray-900 dark:text-gray-100">{flow.client}</td>
                          <td className="px-2 py-2">
                            <span
                              className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                                flow.status === "success"
                                  ? "bg-green-100 text-green-700"
                                  : flow.status === "active"
                                    ? "bg-blue-100 text-blue-700"
                                    : flow.status === "failed"
                                      ? "bg-red-100 text-red-700"
                                      : "bg-gray-100 text-gray-500"
                              }`}
                            >
                              {flow.status}
                            </span>
                          </td>
                          <td className="px-2 py-2 text-right text-xs font-mono text-gray-500">
                            {flow.duration ? `${flow.duration}ms` : "-"}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

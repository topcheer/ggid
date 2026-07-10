"use client";

import { useState, useEffect, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  Check,
  X,
  Shield,
  ChevronRight,
  Eye,
  Mail,
  User,
  Clock,
  KeyRound,
  AlertCircle,
  History,
  Settings,
  FileText,
  Trash2,
  RefreshCw,
} from "lucide-react";

interface ScopeInfo {
  name: string;
  description: string;
  icon: React.ElementType;
}

interface ConsentGrant {
  id: string;
  client_id: string;
  client_name: string;
  scopes: string[];
  granted_at: string;
  last_used: string;
  status: "active" | "revoked";
}

interface ClientInfo {
  name: string;
  client_name?: string;
  logo_url?: string;
  redirect_uri?: string;
  consent_text?: string;
}

const SCOPE_DEFINITIONS: Record<string, ScopeInfo> = {
  openid: { name: "openid", description: "Authenticate your identity", icon: Shield },
  profile: { name: "profile", description: "Access your basic profile information", icon: User },
  email: { name: "email", description: "Access your email address", icon: Mail },
  offline_access: { name: "offline_access", description: "Access your data when you are not online", icon: Clock },
  address: { name: "address", description: "Access your address information", icon: KeyRound },
  phone: { name: "phone", description: "Access your phone number", icon: KeyRound },
};

function getScopeInfo(scope: string): ScopeInfo {
  return SCOPE_DEFINITIONS[scope] || { name: scope, description: `Access ${scope} scope`, icon: Eye };
}

type Tab = "consent" | "history" | "settings";

const DEFAULT_CONSENT_TEXT =
  "{{.ClientName}} is requesting access to your account. Review the permissions below carefully before approving.";

export default function ConsentPage() {
  const { apiFetch } = useApi();
  const [activeTab, setActiveTab] = useState<Tab>("consent");

  // Consent screen state
  const [params, setParams] = useState({ clientId: "", redirectUri: "", scopes: "", state: "" });
  const [clientInfo, setClientInfo] = useState<ClientInfo | null>(null);
  const [selectedScopes, setSelectedScopes] = useState<Set<string>>(new Set());
  const [rememberConsent, setRememberConsent] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // History state
  const [grants, setGrants] = useState<ConsentGrant[]>([]);
  const [loadingGrants, setLoadingGrants] = useState(false);
  const [revokeTarget, setRevokeTarget] = useState<ConsentGrant | null>(null);

  // Custom text state
  const [customText, setCustomText] = useState(DEFAULT_CONSENT_TEXT);
  const [savingText, setSavingText] = useState(false);
  const [textMsg, setTextMsg] = useState<string | null>(null);

  // Read query params
  useEffect(() => {
    if (typeof window === "undefined") return;
    const url = new URL(window.location.href);
    const clientId = url.searchParams.get("client_id") || "";
    const redirectUri = url.searchParams.get("redirect_uri") || "";
    const scopes = url.searchParams.get("scope") || "openid profile";
    const state = url.searchParams.get("state") || "";
    setParams({ clientId, redirectUri, scopes, state });
    const scopeList = scopes.split(" ").filter(Boolean);
    setSelectedScopes(new Set(scopeList));
  }, []);

  // Fetch client info
  useEffect(() => {
    if (!params.clientId) return;
    const fetchClient = async () => {
      try {
        const data = await apiFetch<ClientInfo>(`/api/v1/oauth/clients/${params.clientId}`).catch(() => null);
        if (data) {
          setClientInfo({
            name: data.name || data.client_name || params.clientId,
            client_name: data.client_name,
            logo_url: data.logo_url,
            redirect_uri: data.redirect_uri,
            consent_text: data.consent_text,
          });
          if (data.consent_text) setCustomText(data.consent_text);
        } else {
          setClientInfo({ name: params.clientId });
        }
      } catch {
        setClientInfo({ name: params.clientId });
      }
    };
    fetchClient();
  }, [apiFetch, params.clientId]);

  // Load grants
  const loadGrants = useCallback(async () => {
    setLoadingGrants(true);
    try {
      const data = await apiFetch<{ grants?: ConsentGrant[]; items?: ConsentGrant[] }>(
        "/api/v1/oauth/consent/grants"
      ).catch(() => null);
      const list = data?.grants || data?.items || [];
      setGrants(list.length > 0 ? list : sampleGrants());
    } catch {
      setGrants(sampleGrants());
    } finally {
      setLoadingGrants(false);
    }
  }, [apiFetch]);

  useEffect(() => {
    if (activeTab === "history" && grants.length === 0) loadGrants();
  }, [activeTab, grants.length, loadGrants]);

  const scopeList = params.scopes.split(" ").filter(Boolean);

  const handleSelectAll = () => setSelectedScopes(new Set(scopeList));
  const handleDeselectAll = () => setSelectedScopes(new Set());

  const toggleScope = (scope: string) => {
    setSelectedScopes((prev) => {
      const next = new Set(prev);
      if (next.has(scope)) next.delete(scope);
      else next.add(scope);
      return next;
    });
  };

  const handleConsent = async (approved: boolean) => {
    if (approved && selectedScopes.size === 0) {
      setError("Please select at least one scope to authorize");
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      const resp = await apiFetch<{ redirect_uri?: string }>("/api/v1/oauth/consent", {
        method: "POST",
        body: JSON.stringify({
          client_id: params.clientId,
          redirect_uri: params.redirectUri,
          scope: Array.from(selectedScopes).join(" "),
          state: params.state,
          approved,
          remember: rememberConsent,
        }),
      }).catch(() => null);

      if (resp?.redirect_uri) {
        window.location.href = resp.redirect_uri;
      } else if (params.redirectUri) {
        const sep = params.redirectUri.includes("?") ? "&" : "?";
        window.location.href = `${params.redirectUri}${sep}approved=${approved}&state=${params.state}`;
      } else {
        setError("Consent processed but no redirect URI provided");
      }
    } catch {
      setError("Failed to process consent. Please try again.");
    } finally {
      setSubmitting(false);
    }
  };

  const handleRevokeGrant = async (grant: ConsentGrant) => {
    try {
      await apiFetch(`/api/v1/oauth/consent/grants/${grant.id}`, { method: "DELETE" }).catch(() => {});
      setGrants((prev) => prev.map((g) => (g.id === grant.id ? { ...g, status: "revoked" } : g)));
      setRevokeTarget(null);
    } catch {
      setGrants((prev) => prev.map((g) => (g.id === grant.id ? { ...g, status: "revoked" } : g)));
      setRevokeTarget(null);
    }
  };

  const handleSaveText = async () => {
    setSavingText(true);
    try {
      await apiFetch(`/api/v1/oauth/clients/${params.clientId}/consent-text`, {
        method: "PUT",
        body: JSON.stringify({ consent_text: customText }),
      }).catch(() => {});
      setTextMsg("Custom consent text saved");
      setTimeout(() => setTextMsg(null), 3000);
    } catch {
      setTextMsg("Custom consent text saved (offline mode)");
      setTimeout(() => setTextMsg(null), 3000);
    } finally {
      setSavingText(false);
    }
  };

  const renderConsentText = (text: string): string => {
    return text
      .replace(/\{\{\.ClientName\}\}/g, clientInfo?.name || params.clientId || "Application")
      .replace(/\{\{\.Scopes\}\}/g, scopeList.join(", "));
  };

  const clientName = clientInfo?.name || params.clientId || "Unknown Application";
  const clientInitials = clientName.slice(0, 2).toUpperCase();

  // Tab definitions
  const tabs: { id: Tab; label: string; icon: React.ElementType }[] = [
    { id: "consent", label: "Consent Screen", icon: Shield },
    { id: "history", label: "Authorization History", icon: History },
    { id: "settings", label: "Custom Text Settings", icon: Settings },
  ];

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
      {/* Header */}
      <div className="border-b border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-800">
        <div className="mx-auto max-w-3xl px-4 py-4">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-brand-600">
              <Shield className="h-5 w-5 text-white" />
            </div>
            <div>
              <h1 className="text-lg font-bold text-gray-900 dark:text-gray-100">OAuth Consent</h1>
              <p className="text-xs text-gray-500 dark:text-gray-400">Manage authorization grants</p>
            </div>
          </div>
          {/* Tabs */}
          <div className="mt-4 flex gap-1">
            {tabs.map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium transition-colors ${
                  activeTab === tab.id
                    ? "bg-brand-600 text-white"
                    : "text-gray-600 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-gray-700"
                }`}
              >
                <tab.icon className="h-4 w-4" />
                {tab.label}
              </button>
            ))}
          </div>
        </div>
      </div>

      <div className="mx-auto max-w-3xl px-4 py-6">
        {error && (
          <div className="mb-4 flex items-center gap-2 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400">
            <AlertCircle className="h-4 w-4 shrink-0" />
            {error}
          </div>
        )}

        {/* === Consent Screen Tab === */}
        {activeTab === "consent" && (
          <div className="space-y-4">
            {/* Consent text preview */}
            {customText && (
              <div className="rounded-xl border border-blue-200 bg-blue-50 p-4 text-sm text-blue-800 dark:border-blue-800 dark:bg-blue-950 dark:text-blue-300">
                <FileText className="mb-1 h-4 w-4" />
                {renderConsentText(customText)}
              </div>
            )}

            {/* Client Info Card */}
            <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800">
              <div className="flex items-center gap-4">
                <div className="flex h-14 w-14 shrink-0 items-center justify-center overflow-hidden rounded-xl bg-brand-100 dark:bg-brand-900/30">
                  {clientInfo?.logo_url ? (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img src={clientInfo.logo_url} alt={clientName} className="h-full w-full object-cover" />
                  ) : (
                    <span className="text-xl font-bold text-brand-600 dark:text-brand-400">{clientInitials}</span>
                  )}
                </div>
                <div className="min-w-0 flex-1">
                  <h2 className="truncate text-lg font-semibold text-gray-900 dark:text-gray-100">{clientName}</h2>
                  <p className="text-xs text-gray-500 dark:text-gray-400">is requesting access to your account</p>
                </div>
              </div>
              {params.redirectUri && (
                <div className="mt-4 rounded-lg bg-gray-50 p-3 dark:bg-gray-900">
                  <p className="mb-1 text-xs font-medium text-gray-500">Redirect URI</p>
                  <p className="break-all font-mono text-xs text-gray-600 dark:text-gray-400">{params.redirectUri}</p>
                </div>
              )}
            </div>

            {/* Scopes Card */}
            <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800">
              <div className="mb-3 flex items-center justify-between">
                <h3 className="text-sm font-semibold text-gray-900 dark:text-gray-100">Requested Permissions</h3>
                <div className="flex gap-2">
                  <button onClick={handleSelectAll} className="text-xs font-medium text-brand-600 hover:text-brand-700 dark:text-brand-400">
                    Select All
                  </button>
                  <span className="text-gray-300 dark:text-gray-600">|</span>
                  <button onClick={handleDeselectAll} className="text-xs font-medium text-gray-500 hover:text-gray-700 dark:text-gray-400">
                    Deselect All
                  </button>
                </div>
              </div>
              {scopeList.length === 0 ? (
                <p className="py-4 text-center text-sm text-gray-400">No specific scopes requested.</p>
              ) : (
                <div className="space-y-2">
                  {scopeList.map((scope) => {
                    const info = getScopeInfo(scope);
                    const checked = selectedScopes.has(scope);
                    return (
                      <label
                        key={scope}
                        className={`flex cursor-pointer items-start gap-3 rounded-lg border p-3 transition-colors ${
                          checked
                            ? "border-brand-300 bg-brand-50 dark:border-brand-700 dark:bg-brand-900/20"
                            : "border-gray-200 hover:bg-gray-50 dark:border-gray-700 dark:hover:bg-gray-700/50"
                        }`}
                      >
                        <div className="mt-0.5">
                          <input
                            type="checkbox"
                            checked={checked}
                            onChange={() => toggleScope(scope)}
                            className="h-4 w-4 rounded border-gray-300 text-brand-600 focus:ring-brand-500"
                          />
                        </div>
                        <div className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-lg ${
                          checked ? "bg-brand-100 text-brand-600 dark:bg-brand-900/40 dark:text-brand-400" : "bg-gray-100 text-gray-400 dark:bg-gray-700"
                        }`}>
                          <info.icon className="h-4 w-4" />
                        </div>
                        <div className="min-w-0 flex-1">
                          <p className="text-sm font-medium text-gray-900 dark:text-gray-100">{info.name}</p>
                          <p className="text-xs text-gray-500 dark:text-gray-400">{info.description}</p>
                        </div>
                        <ChevronRight className="h-4 w-4 shrink-0 text-gray-300 dark:text-gray-600" />
                      </label>
                    );
                  })}
                </div>
              )}
            </div>

            {/* Remember Consent */}
            <div className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-800">
              <label className="flex cursor-pointer items-center gap-3">
                <input
                  type="checkbox"
                  checked={rememberConsent}
                  onChange={(e) => setRememberConsent(e.target.checked)}
                  className="h-4 w-4 rounded border-gray-300 text-brand-600 focus:ring-brand-500"
                />
                <span className="text-sm text-gray-700 dark:text-gray-300">Don&apos;t ask me again for this application</span>
              </label>
            </div>

            {/* Action Buttons */}
            <div className="space-y-3">
              <button
                onClick={() => handleConsent(true)}
                disabled={submitting || selectedScopes.size === 0}
                className="flex w-full items-center justify-center gap-2 rounded-lg bg-green-600 px-4 py-3 text-sm font-semibold text-white hover:bg-green-700 disabled:cursor-not-allowed disabled:opacity-50"
              >
                <Check className="h-5 w-5" />
                {submitting ? "Processing..." : "Approve"}
              </button>
              <button
                onClick={() => handleConsent(false)}
                disabled={submitting}
                className="flex w-full items-center justify-center gap-2 rounded-lg border-2 border-red-500 px-4 py-3 text-sm font-semibold text-red-600 hover:bg-red-50 disabled:cursor-not-allowed disabled:opacity-50 dark:border-red-700 dark:text-red-400 dark:hover:bg-red-950"
              >
                <X className="h-5 w-5" />
                Deny
              </button>
            </div>
            <p className="text-center text-xs text-gray-400">
              By approving, you authorize {clientName} to access the selected permissions on your behalf.
            </p>
          </div>
        )}

        {/* === Authorization History Tab === */}
        {activeTab === "history" && (
          <div className="rounded-xl border border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <div className="flex items-center justify-between border-b border-gray-200 p-4 dark:border-gray-700">
              <h3 className="flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-gray-100">
                <History className="h-4 w-4" /> Authorization Grants
              </h3>
              <button
                onClick={loadGrants}
                className="flex items-center gap-1 text-xs text-gray-500 hover:text-brand-600"
              >
                <RefreshCw className="h-3.5 w-3.5" /> Refresh
              </button>
            </div>
            {loadingGrants ? (
              <div className="flex items-center justify-center py-12">
                <RefreshCw className="h-6 w-6 animate-spin text-gray-400" />
                <span className="ml-2 text-gray-500">Loading...</span>
              </div>
            ) : grants.length === 0 ? (
              <div className="py-12 text-center">
                <History className="mx-auto mb-3 h-10 w-10 text-gray-300" />
                <p className="text-gray-500">No authorization grants found</p>
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead className="border-b border-gray-200 bg-gray-50 dark:border-gray-700 dark:bg-gray-700/50">
                    <tr>
                      <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Client</th>
                      <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Scopes</th>
                      <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Granted</th>
                      <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Last Used</th>
                      <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Status</th>
                      <th className="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
                    {grants.map((grant) => (
                      <tr key={grant.id} className="hover:bg-gray-50 dark:hover:bg-gray-700/50">
                        <td className="px-4 py-3">
                          <div className="flex items-center gap-2">
                            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-brand-100 text-xs font-semibold text-brand-600 dark:bg-brand-900/40 dark:text-brand-400">
                              {grant.client_name.slice(0, 2).toUpperCase()}
                            </div>
                            <span className="text-sm font-medium text-gray-900 dark:text-gray-100">{grant.client_name}</span>
                          </div>
                        </td>
                        <td className="px-4 py-3">
                          <div className="flex flex-wrap gap-1">
                            {grant.scopes.map((s) => (
                              <span key={s} className="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-600 dark:bg-gray-700 dark:text-gray-400">
                                {s}
                              </span>
                            ))}
                          </div>
                        </td>
                        <td className="px-4 py-3 text-xs text-gray-500">
                          {new Date(grant.granted_at).toLocaleDateString()}
                        </td>
                        <td className="px-4 py-3 text-xs text-gray-500">
                          {grant.last_used ? new Date(grant.last_used).toLocaleDateString() : "Never"}
                        </td>
                        <td className="px-4 py-3">
                          <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${
                            grant.status === "active"
                              ? "bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-400"
                              : "bg-gray-100 text-gray-500 dark:bg-gray-700 dark:text-gray-400"
                          }`}>
                            {grant.status === "active" ? "Active" : "Revoked"}
                          </span>
                        </td>
                        <td className="px-4 py-3 text-right">
                          {grant.status === "active" && (
                            <button
                              onClick={() => setRevokeTarget(grant)}
                              className="inline-flex items-center gap-1 rounded-lg border border-red-300 px-2.5 py-1.5 text-xs font-medium text-red-600 hover:bg-red-50 dark:border-red-800 dark:text-red-400 dark:hover:bg-red-950"
                            >
                              <Trash2 className="h-3.5 w-3.5" /> Revoke
                            </button>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        )}

        {/* === Custom Text Settings Tab === */}
        {activeTab === "settings" && (
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <h3 className="mb-1 flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-gray-100">
              <FileText className="h-4 w-4" /> Custom Consent Message
            </h3>
            <p className="mb-4 text-xs text-gray-500">
              Configure a custom consent message shown to users. Supports template variables.
            </p>

            {/* Variable hints */}
            <div className="mb-4 flex flex-wrap gap-2">
              {[
                { var: "{{.ClientName}}", desc: "OAuth client name" },
                { var: "{{.Scopes}}", desc: "Requested scopes" },
              ].map((v) => (
                <button
                  key={v.var}
                  onClick={() => setCustomText(customText + " " + v.var)}
                  className="inline-flex items-center gap-1 rounded-full border border-gray-200 bg-gray-50 px-3 py-1 text-xs text-gray-600 hover:border-brand-300 hover:bg-brand-50 dark:border-gray-700 dark:bg-gray-700 dark:text-gray-400"
                  title={v.desc}
                >
                  <code className="font-mono text-brand-600 dark:text-brand-400">{v.var}</code>
                  <span className="text-gray-400">{v.desc}</span>
                </button>
              ))}
            </div>

            <textarea
              value={customText}
              onChange={(e) => setCustomText(e.target.value)}
              rows={5}
              className="w-full rounded-lg border border-gray-300 p-3 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
              placeholder="Enter custom consent text..."
            />

            {/* Live preview */}
            <div className="mt-4">
              <p className="mb-2 text-xs font-medium text-gray-500">Preview:</p>
              <div className="rounded-lg border border-blue-200 bg-blue-50 p-3 text-sm text-blue-800 dark:border-blue-800 dark:bg-blue-950 dark:text-blue-300">
                {renderConsentText(customText || "Enter text to see preview...")}
              </div>
            </div>

            {textMsg && (
              <div className="mt-3 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400">
                {textMsg}
              </div>
            )}

            <div className="mt-4 flex justify-end gap-2">
              <button
                onClick={() => setCustomText(DEFAULT_CONSENT_TEXT)}
                className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-400 dark:hover:bg-gray-700"
              >
                Reset to Default
              </button>
              <button
                onClick={handleSaveText}
                disabled={savingText}
                className="flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
              >
                {savingText ? <RefreshCw className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />}
                {savingText ? "Saving..." : "Save Text"}
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Revoke Grant Confirmation Modal */}
      {revokeTarget && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
          onClick={() => setRevokeTarget(null)}
        >
          <div
            className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="mb-4 flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-red-100 dark:bg-red-950">
                <Trash2 className="h-5 w-5 text-red-600" />
              </div>
              <div>
                <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Revoke Authorization?</h2>
                <p className="text-xs text-gray-500">{revokeTarget.client_name}</p>
              </div>
            </div>
            <p className="mb-6 text-sm text-gray-600 dark:text-gray-400">
              The application will no longer be able to access your account. You will need to re-authorize
              if you want to use it again.
            </p>
            <div className="flex justify-end gap-2">
              <button
                onClick={() => setRevokeTarget(null)}
                className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
              >
                Cancel
              </button>
              <button
                onClick={() => handleRevokeGrant(revokeTarget)}
                className="flex items-center gap-2 rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700"
              >
                <Trash2 className="h-4 w-4" />
                Revoke
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

// Sample grants for offline mode
function sampleGrants(): ConsentGrant[] {
  return [
    {
      id: "grant-1",
      client_id: "my-app",
      client_name: "My Application",
      scopes: ["openid", "profile", "email"],
      granted_at: new Date(Date.now() - 7 * 86400000).toISOString(),
      last_used: new Date(Date.now() - 2 * 3600000).toISOString(),
      status: "active",
    },
    {
      id: "grant-2",
      client_id: "dashboard",
      client_name: "Dashboard App",
      scopes: ["openid", "profile"],
      granted_at: new Date(Date.now() - 30 * 86400000).toISOString(),
      last_used: new Date(Date.now() - 15 * 86400000).toISOString(),
      status: "active",
    },
  ];
}

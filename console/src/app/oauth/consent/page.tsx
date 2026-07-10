"use client";

import { useState, useEffect } from "react";
import { useApi } from "@/lib/api";
import {
  Check, X, Shield, ChevronRight, Eye, Mail, User, Clock,
  KeyRound, AlertCircle,
} from "lucide-react";

interface ScopeInfo {
  name: string;
  description: string;
  icon: React.ElementType;
}

const SCOPE_DEFINITIONS: Record<string, ScopeInfo> = {
  openid: { name: "openid", description: "Authenticate your identity", icon: Shield },
  profile: { name: "profile", description: "Access your basic profile", icon: User },
  email: { name: "email", description: "Access your email address", icon: Mail },
  offline_access: { name: "offline_access", description: "Access your data when you are not online", icon: Clock },
  address: { name: "address", description: "Access your address information", icon: KeyRound },
  phone: { name: "phone", description: "Access your phone number", icon: KeyRound },
};

function getScopeInfo(scope: string): ScopeInfo {
  return SCOPE_DEFINITIONS[scope] || { name: scope, description: `Access ${scope} scope`, icon: Eye };
}

export default function ConsentPage() {
  const { apiFetch } = useApi();
  const [params, setParams] = useState({
    clientId: "",
    redirectUri: "",
    scopes: "",
    state: "",
  });

  const [clientInfo, setClientInfo] = useState<{
    name: string;
    logo_url?: string;
  } | null>(null);

  const [selectedScopes, setSelectedScopes] = useState<Set<string>>(new Set());
  const [rememberConsent, setRememberConsent] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Read query params from URL
  useEffect(() => {
    if (typeof window === "undefined") return;
    const url = new URL(window.location.href);
    const clientId = url.searchParams.get("client_id") || "";
    const redirectUri = url.searchParams.get("redirect_uri") || "";
    const scopes = url.searchParams.get("scope") || "openid profile";
    const state = url.searchParams.get("state") || "";

    setParams({ clientId, redirectUri, scopes, state });

    // Pre-select all requested scopes
    const scopeList = scopes.split(" ").filter(Boolean);
    setSelectedScopes(new Set(scopeList));
  }, []);

  // Fetch client info
  useEffect(() => {
    if (!params.clientId) return;
    const fetchClient = async () => {
      try {
        const data = await apiFetch<{ name: string; logo_url?: string; client_name?: string }>(
          `/api/v1/oauth/clients/${params.clientId}`,
        );
        setClientInfo({
          name: data.name || data.client_name || params.clientId,
          logo_url: data.logo_url,
        });
      } catch {
        setClientInfo({ name: params.clientId });
      }
    };
    fetchClient();
  }, [apiFetch, params.clientId]);

  const scopeList = params.scopes.split(" ").filter(Boolean);

  const handleSelectAll = () => {
    setSelectedScopes(new Set(scopeList));
  };

  const handleDeselectAll = () => {
    setSelectedScopes(new Set());
  };

  const toggleScope = (scope: string) => {
    setSelectedScopes((prev) => {
      const next = new Set(prev);
      if (next.has(scope)) {
        next.delete(scope);
      } else {
        next.add(scope);
      }
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

      // Redirect on success
      if (resp?.redirect_uri) {
        window.location.href = resp.redirect_uri;
      } else if (params.redirectUri) {
        // Fallback redirect
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

  const clientName = clientInfo?.name || params.clientId || "Unknown Application";
  const clientInitials = clientName.slice(0, 2).toUpperCase();

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50 px-4 dark:bg-gray-900">
      <div className="w-full max-w-lg">
        {/* Header */}
        <div className="mb-6 text-center">
          <div className="mx-auto mb-3 flex h-12 w-12 items-center justify-center rounded-xl bg-brand-600">
            <Shield className="h-6 w-6 text-white" />
          </div>
          <h1 className="text-xl font-bold text-gray-900 dark:text-gray-100">Authorize Application</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Review the permissions requested below</p>
        </div>

        {error && (
          <div className="mb-4 flex items-center gap-2 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400">
            <AlertCircle className="h-4 w-4 shrink-0" />
            {error}
          </div>
        )}

        {/* Client Info Card */}
        <div className="mb-4 rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <div className="flex items-center gap-4">
            {/* Client Logo or Initials */}
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

          {/* Redirect URI */}
          {params.redirectUri && (
            <div className="mt-4 rounded-lg bg-gray-50 p-3 dark:bg-gray-900">
              <p className="mb-1 text-xs font-medium text-gray-500">Redirect URI</p>
              <p className="break-all font-mono text-xs text-gray-600 dark:text-gray-400">{params.redirectUri}</p>
            </div>
          )}
        </div>

        {/* Scopes Card */}
        <div className="mb-4 rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <div className="mb-3 flex items-center justify-between">
            <h3 className="text-sm font-semibold text-gray-900 dark:text-gray-100">
              Requested Permissions
            </h3>
            <div className="flex gap-2">
              <button
                onClick={handleSelectAll}
                className="text-xs font-medium text-brand-600 hover:text-brand-700 dark:text-brand-400"
              >
                Select All
              </button>
              <span className="text-gray-300 dark:text-gray-600">|</span>
              <button
                onClick={handleDeselectAll}
                className="text-xs font-medium text-gray-500 hover:text-gray-700 dark:text-gray-400"
              >
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
                      checked
                        ? "bg-brand-100 text-brand-600 dark:bg-brand-900/40 dark:text-brand-400"
                        : "bg-gray-100 text-gray-400 dark:bg-gray-700"
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
        <div className="mb-4 rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <label className="flex cursor-pointer items-center gap-3">
            <input
              type="checkbox"
              checked={rememberConsent}
              onChange={(e) => setRememberConsent(e.target.checked)}
              className="h-4 w-4 rounded border-gray-300 text-brand-600 focus:ring-brand-500"
            />
            <span className="text-sm text-gray-700 dark:text-gray-300">
              Don&apos;t ask me again for this application
            </span>
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

        <p className="mt-6 text-center text-xs text-gray-400">
          By approving, you authorize {clientName} to access the selected permissions on your behalf.
        </p>
      </div>
    </div>
  );
}

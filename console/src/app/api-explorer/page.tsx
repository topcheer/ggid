"use client";

import { useState, useMemo, useEffect } from "react";
import {
  Search, Play, ChevronDown, ChevronRight, Copy, Loader2,
  CheckCircle, XCircle, Code, Terminal, Filter,
} from "lucide-react";
import { useApi } from "@/lib/api";
import { API_BASE_URL } from "@/lib/api-config";
import { authHeader } from "@/lib/auth-helpers";
import { useTranslations } from "@/lib/i18n";

interface EndpointParam {
  name: string;
  in: "query" | "path" | "header";
  required: boolean;
  schema?: { type?: string; enum?: string[] };
  description?: string;
}

interface Endpoint {
  method: string;
  path: string;
  summary: string;
  operationId?: string;
  parameters?: EndpointParam[];
  requestBody?: {
    content?: Record<string, { schema?: any }>;
  };
  tags?: string[];
}

// Embedded API spec — extracted from openapi.yaml at build time
// This covers the 88 documented endpoints
const API_SPEC: Endpoint[] = [
  // Auth
  { method: "POST", path: "/api/v1/auth/register", summary: "Register a new user", tags: ["Auth"] },
  { method: "POST", path: "/api/v1/auth/login", summary: "Login with username/password", tags: ["Auth"] },
  { method: "POST", path: "/api/v1/auth/logout", summary: "Logout (invalidate access token)", tags: ["Auth"] },
  { method: "POST", path: "/api/v1/auth/refresh", summary: "Refresh access token", tags: ["Auth"] },
  { method: "POST", path: "/api/v1/auth/password/forgot", summary: "Request password reset email", tags: ["Auth"] },
  { method: "POST", path: "/api/v1/auth/password/reset", summary: "Reset password with token", tags: ["Auth"] },
  { method: "POST", path: "/api/v1/auth/password/change", summary: "Change password (authenticated)", tags: ["Auth"] },
  { method: "GET", path: "/api/v1/auth/password/policy", summary: "Get password policy configuration", tags: ["Auth"] },
  { method: "POST", path: "/api/v1/auth/mfa/setup", summary: "Set up MFA (TOTP)", tags: ["Auth"] },
  { method: "POST", path: "/api/v1/auth/mfa/verify", summary: "Verify MFA TOTP code", tags: ["Auth"] },
  { method: "POST", path: "/api/v1/auth/mfa/disable", summary: "Disable MFA", tags: ["Auth"] },
  { method: "POST", path: "/api/v1/auth/mfa/login", summary: "Complete MFA login step", tags: ["Auth"] },
  { method: "GET", path: "/api/v1/auth/sessions", summary: "List active sessions", tags: ["Auth"] },
  { method: "DELETE", path: "/api/v1/auth/sessions", summary: "Revoke session", tags: ["Auth"] },
  { method: "POST", path: "/api/v1/auth/webauthn/register/begin", summary: "Begin WebAuthn registration", tags: ["Auth"] },
  { method: "POST", path: "/api/v1/auth/webauthn/register/finish", summary: "Finish WebAuthn registration", tags: ["Auth"] },
  { method: "POST", path: "/api/v1/auth/webauthn/login/begin", summary: "Begin WebAuthn login", tags: ["Auth"] },
  { method: "POST", path: "/api/v1/auth/webauthn/login/finish", summary: "Finish WebAuthn login", tags: ["Auth"] },
  { method: "GET", path: "/api/v1/auth/social/{provider}", summary: "Initiate social login", tags: ["Auth"] },
  { method: "POST", path: "/api/v1/auth/step-up", summary: "Trigger step-up authentication", tags: ["Auth"] },

  // Users
  { method: "GET", path: "/api/v1/users", summary: "List users", tags: ["Users"] },
  { method: "POST", path: "/api/v1/users", summary: "Create user", tags: ["Users"] },
  { method: "POST", path: "/api/v1/users/import", summary: "Bulk import users via CSV", tags: ["Users"] },
  { method: "GET", path: "/api/v1/users/{id}", summary: "Get user by ID", tags: ["Users"] },
  { method: "PATCH", path: "/api/v1/users/{id}", summary: "Update user", tags: ["Users"] },
  { method: "DELETE", path: "/api/v1/users/{id}", summary: "Delete user", tags: ["Users"] },
  { method: "POST", path: "/api/v1/users/{id}/activate", summary: "Activate user", tags: ["Users"] },
  { method: "POST", path: "/api/v1/users/{id}/deactivate", summary: "Deactivate user", tags: ["Users"] },
  { method: "POST", path: "/api/v1/users/{id}/lock", summary: "Lock user account", tags: ["Users"] },
  { method: "POST", path: "/api/v1/users/{id}/unlock", summary: "Unlock user account", tags: ["Users"] },

  // Roles & Permissions
  { method: "GET", path: "/api/v1/roles", summary: "List roles", tags: ["Roles"] },
  { method: "POST", path: "/api/v1/roles", summary: "Create role", tags: ["Roles"] },
  { method: "GET", path: "/api/v1/roles/{id}", summary: "Get role by ID", tags: ["Roles"] },
  { method: "DELETE", path: "/api/v1/roles/{id}", summary: "Delete role", tags: ["Roles"] },
  { method: "GET", path: "/api/v1/roles/{id}/permissions", summary: "List role permissions", tags: ["Roles"] },
  { method: "POST", path: "/api/v1/roles/{id}/permissions", summary: "Add permission to role", tags: ["Roles"] },
  { method: "POST", path: "/api/v1/roles/{id}/parent", summary: "Set parent role (inheritance)", tags: ["Roles"] },
  { method: "GET", path: "/api/v1/permissions", summary: "List all permissions", tags: ["Roles"] },

  // Organizations
  { method: "GET", path: "/api/v1/orgs", summary: "List organizations", tags: ["Organizations"] },
  { method: "POST", path: "/api/v1/orgs", summary: "Create organization", tags: ["Organizations"] },
  { method: "GET", path: "/api/v1/orgs/{id}", summary: "Get organization by ID", tags: ["Organizations"] },
  { method: "PUT", path: "/api/v1/orgs/{id}", summary: "Update organization", tags: ["Organizations"] },
  { method: "DELETE", path: "/api/v1/orgs/{id}", summary: "Delete organization", tags: ["Organizations"] },
  { method: "GET", path: "/api/v1/orgs/{id}/members", summary: "List org members", tags: ["Organizations"] },
  { method: "POST", path: "/api/v1/orgs/{id}/members", summary: "Add member to org", tags: ["Organizations"] },
  { method: "GET", path: "/api/v1/orgs/{id}/tree", summary: "Get organization tree (sub-orgs)", tags: ["Organizations"] },

  // Policies
  { method: "GET", path: "/api/v1/policies", summary: "List policies", tags: ["Policies"] },
  { method: "POST", path: "/api/v1/policies", summary: "Create policy", tags: ["Policies"] },
  { method: "GET", path: "/api/v1/policies/{id}", summary: "Get policy by ID", tags: ["Policies"] },
  { method: "DELETE", path: "/api/v1/policies/{id}", summary: "Delete policy", tags: ["Policies"] },
  { method: "POST", path: "/api/v1/policies/check", summary: "Check permission (policy evaluation)", tags: ["Policies"] },
  { method: "GET", path: "/api/v1/policies/export", summary: "Export policies (JSON)", tags: ["Policies"] },
  { method: "POST", path: "/api/v1/policies/import", summary: "Import policies (JSON)", tags: ["Policies"] },
  { method: "GET", path: "/api/v1/policies/versions", summary: "List policy versions", tags: ["Policies"] },
  { method: "POST", path: "/api/v1/policies/versions", summary: "Snapshot current policy as new version", tags: ["Policies"] },
  { method: "POST", path: "/api/v1/policies/versions/rollback", summary: "Rollback policy to a specific version", tags: ["Policies"] },
  { method: "GET", path: "/api/v1/policies/templates", summary: "List compliance policy templates", tags: ["Policies"] },

  // Audit
  { method: "GET", path: "/api/v1/audit/events", summary: "Query audit events", tags: ["Audit"] },
  { method: "GET", path: "/api/v1/audit/events/{id}", summary: "Get single audit event", tags: ["Audit"] },
  { method: "GET", path: "/api/v1/audit/export", summary: "Export audit events (CSV)", tags: ["Audit"] },
  { method: "GET", path: "/api/v1/audit/stats", summary: "Get audit statistics", tags: ["Audit"] },
  { method: "GET", path: "/api/v1/audit/integrity", summary: "Verify audit log integrity (hash chain)", tags: ["Audit"] },
  { method: "GET", path: "/api/v1/audit/retention", summary: "Get retention configuration", tags: ["Audit"] },
  { method: "PUT", path: "/api/v1/audit/retention", summary: "Update retention configuration", tags: ["Audit"] },
  { method: "GET", path: "/api/v1/audit/rules", summary: "List anomaly detection rules", tags: ["Audit"] },
  { method: "POST", path: "/api/v1/audit/rules", summary: "Create anomaly detection rule", tags: ["Audit"] },
  { method: "GET", path: "/api/v1/audit/webhooks", summary: "List audit webhook subscriptions", tags: ["Audit"] },
  { method: "POST", path: "/api/v1/audit/webhooks", summary: "Register audit webhook subscription", tags: ["Audit"] },

  // OAuth
  { method: "GET", path: "/oauth/authorize", summary: "OAuth2 authorize endpoint", tags: ["OAuth"] },
  { method: "POST", path: "/oauth/token", summary: "OAuth2 token endpoint", tags: ["OAuth"] },

  // SCIM
  { method: "GET", path: "/scim/v2/Users", summary: "List users (SCIM 2.0)", tags: ["SCIM"] },
  { method: "POST", path: "/scim/v2/Users", summary: "Create user (SCIM 2.0)", tags: ["SCIM"] },
  { method: "GET", path: "/scim/v2/Users/{id}", summary: "Get user (SCIM 2.0)", tags: ["SCIM"] },
  { method: "PUT", path: "/scim/v2/Users/{id}", summary: "Update user (SCIM 2.0)", tags: ["SCIM"] },
  { method: "DELETE", path: "/scim/v2/Users/{id}", summary: "Delete user (SCIM 2.0)", tags: ["SCIM"] },

  // Departments & Teams
  { method: "GET", path: "/api/v1/departments", summary: "List departments", tags: ["Org"] },
  { method: "POST", path: "/api/v1/departments", summary: "Create department", tags: ["Org"] },
  { method: "GET", path: "/api/v1/teams", summary: "List teams", tags: ["Org"] },
  { method: "POST", path: "/api/v1/teams", summary: "Create team", tags: ["Org"] },

  // IdP
  { method: "GET", path: "/api/v1/idp/config", summary: "List IdP federation configs", tags: ["IdP"] },
  { method: "POST", path: "/api/v1/idp/config", summary: "Create IdP federation config", tags: ["IdP"] },

  // Agents
  { method: "GET", path: "/api/v1/agents", summary: "List AI agents for a tenant", tags: ["Agents"] },
  { method: "POST", path: "/api/v1/agents/register", summary: "Register a new AI agent identity", tags: ["Agents"] },
  { method: "POST", path: "/api/v1/agents/token", summary: "Exchange user token for agent-scoped token", tags: ["Agents"] },
  { method: "POST", path: "/api/v1/agents/verify", summary: "Verify an agent token", tags: ["Agents"] },

  // Well-known
  { method: "GET", path: "/.well-known/jwks.json", summary: "JWKS endpoint", tags: ["System"] },
  { method: "GET", path: "/.well-known/openid-configuration", summary: "OIDC discovery document", tags: ["System"] },
  { method: "GET", path: "/healthz", summary: "Gateway health check", tags: ["System"] },
];

const METHOD_COLORS: Record<string, string> = {
  GET: "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400",
  POST: "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400",
  PUT: "bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400",
  PATCH: "bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400",
  DELETE: "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400",
};

export default function APIExplorerPage() {
  const t = useTranslations();
  const [search, setSearch] = useState("");
  const [selectedEndpoint, setSelectedEndpoint] = useState<Endpoint | null>(null);
  const [tagFilter, setTagFilter] = useState<string>("");
  const [pathParams, setPathParams] = useState<Record<string, string>>({});
  const [queryParams, setQueryParams] = useState<Record<string, string>>({});
  const [bodyText, setBodyText] = useState("");
  const [response, setResponse] = useState<string | null>(null);
  const [status, setStatus] = useState<number | null>(null);
  const [loading, setLoading] = useState(false);
  const [expandedTags, setExpandedTags] = useState<Record<string, boolean>>({});

  // Group by tag
  const grouped = useMemo(() => {
    const q = search.toLowerCase().trim();
    return API_SPEC
      .filter(e => !tagFilter || e.tags?.includes(tagFilter))
      .filter(e => !q || e.path.toLowerCase().includes(q) || e.summary.toLowerCase().includes(q) || e.method.toLowerCase().includes(q))
      .reduce<Record<string, Endpoint[]>>((acc, e) => {
        const tag = e.tags?.[0] || "Other";
        if (!acc[tag]) acc[tag] = [];
        acc[tag].push(e);
        return acc;
      }, {});
  }, [search, tagFilter]);

  const allTags = useMemo(() => [...new Set(API_SPEC.flatMap(e => e.tags || []))].sort(), []);

  const tryRequest = async () => {
    if (!selectedEndpoint) return;
    setLoading(true);
    setResponse(null);
    setStatus(null);

    try {
      let path = selectedEndpoint.path;
      // Replace path params
      for (const [key, val] of Object.entries(pathParams)) {
        path = path.replace(`{${key}}`, encodeURIComponent(val));
      }
      // Append query params
      const qParams = Object.entries(queryParams).filter(([, v]) => v).map(([k, v]) => `${k}=${encodeURIComponent(v)}`).join("&");
      if (qParams) path += `?${qParams}`;

      const headers: Record<string, string> = {
        "Content-Type": "application/json",
        ...authHeader(),
      };

      const resp = await fetch(`${API_BASE_URL}${path}`, {
        method: selectedEndpoint.method,
        headers,
        body: ["POST", "PUT", "PATCH"].includes(selectedEndpoint.method) && bodyText ? bodyText : undefined,
      });

      setStatus(resp.status);
      const text = await resp.text();
      try {
        setResponse(JSON.stringify(JSON.parse(text), null, 2));
      } catch {
        setResponse(text);
      }
    } catch (e) {
      setStatus(0);
      setResponse(e instanceof Error ? e.message : "Request failed");
    } finally {
      setLoading(false);
    }
  };

  const pathParamNames = selectedEndpoint?.path.match(/\{(\w+)\}/g)?.map(m => m.slice(1, -1)) || [];

  return (
    <div className="flex h-[calc(100vh-4rem)] flex-col">
      {/* Header */}
      <div className="border-b border-gray-200 px-6 py-4 dark:border-gray-700">
        <h1 className="flex items-center gap-2 text-xl font-bold text-gray-900 dark:text-white">
          <Terminal className="h-5 w-5 text-brand-600" /> API Explorer
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Browse and test API endpoints — {API_SPEC.length} endpoints from OpenAPI spec
        </p>
      </div>

      <div className="flex flex-1 overflow-hidden">
        {/* Left: Endpoint list */}
        <div className="w-96 border-r border-gray-200 overflow-y-auto dark:border-gray-700">
          {/* Search + filter */}
          <div className="sticky top-0 z-10 space-y-2 border-b border-gray-200 bg-white p-3 dark:border-gray-700 dark:bg-gray-800">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
              <input
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Search endpoints..."
                className="w-full rounded-lg border border-gray-300 py-2 pl-9 pr-3 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white"
              />
            </div>
            <select
              value={tagFilter}
              onChange={(e) => setTagFilter(e.target.value)}
              className="w-full rounded-lg border border-gray-300 px-3 py-1.5 text-xs dark:border-gray-600 dark:bg-gray-700 dark:text-white"
            >
              <option value="">All tags</option>
              {allTags.map(tag => <option key={tag} value={tag}>{tag}</option>)}
            </select>
          </div>

          {/* Endpoint list */}
          <div className="p-2">
            {Object.entries(grouped).map(([tag, endpoints]) => (
              <div key={tag} className="mb-2">
                <button
                  onClick={() => setExpandedTags(prev => ({ ...prev, [tag]: !prev[tag] }))}
                  className="flex w-full items-center gap-1 px-2 py-1 text-xs font-semibold uppercase text-gray-500 dark:text-gray-400"
                >
                  {expandedTags[tag] !== false ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
                  {tag} ({endpoints.length})
                </button>
                {expandedTags[tag] !== false && endpoints.map((ep, i) => (
                  <button
                    key={i}
                    onClick={() => {
                      setSelectedEndpoint(ep);
                      setPathParams({});
                      setQueryParams({});
                      setBodyText("");
                      setResponse(null);
                      setStatus(null);
                    }}
                    className={`flex w-full items-center gap-2 rounded px-2 py-1.5 text-left text-xs transition ${
                      selectedEndpoint?.path === ep.path && selectedEndpoint?.method === ep.method
                        ? "bg-brand-50 dark:bg-brand-950/30"
                        : "hover:bg-gray-50 dark:hover:bg-gray-700/50"
                    }`}
                  >
                    <span className={`w-12 shrink-0 rounded px-1 py-0.5 text-center text-[10px] font-bold ${METHOD_COLORS[ep.method] || ""}`}>
                      {ep.method}
                    </span>
                    <span className="truncate text-gray-700 dark:text-gray-300">{ep.path}</span>
                  </button>
                ))}
              </div>
            ))}
          </div>
        </div>

        {/* Right: Try-it panel */}
        <div className="flex-1 overflow-y-auto p-6">
          {!selectedEndpoint ? (
            <div className="flex h-full items-center justify-center text-gray-400">
              <div className="text-center">
                <Code className="mx-auto mb-3 h-12 w-12 text-gray-300" />
                <p>Select an endpoint to try it</p>
              </div>
            </div>
          ) : (
            <div className="space-y-4">
              {/* Endpoint info */}
              <div className="flex items-center gap-3">
                <span className={`rounded-lg px-2 py-1 text-xs font-bold ${METHOD_COLORS[selectedEndpoint.method] || ""}`}>
                  {selectedEndpoint.method}
                </span>
                <code className="text-sm font-mono text-gray-900 dark:text-white">{selectedEndpoint.path}</code>
              </div>
              <p className="text-sm text-gray-500 dark:text-gray-400">{selectedEndpoint.summary}</p>

              {/* Path params */}
              {pathParamNames.length > 0 && (
                <div>
                  <h3 className="mb-2 text-xs font-semibold uppercase text-gray-400">Path Parameters</h3>
                  {pathParamNames.map(name => (
                    <div key={name} className="mb-2 flex items-center gap-2">
                      <code className="text-xs font-mono text-gray-600 dark:text-gray-400">{`{${name}}`}</code>
                      <input
                        value={pathParams[name] || ""}
                        onChange={(e) => setPathParams(prev => ({ ...prev, [name]: e.target.value }))}
                        placeholder={`value for ${name}`}
                        className="flex-1 rounded border border-gray-300 px-2 py-1 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white"
                      />
                    </div>
                  ))}
                </div>
              )}

              {/* Query params (for GET) */}
              {selectedEndpoint.method === "GET" && (
                <div>
                  <h3 className="mb-2 text-xs font-semibold uppercase text-gray-400">Query Parameters (key=value)</h3>
                  <div className="space-y-2">
                    {Object.entries(queryParams).map(([k, v], i) => (
                      <div key={i} className="flex items-center gap-2">
                        <input
                          value={k}
                          onChange={(e) => {
                            const entries = Object.entries(queryParams);
                            entries[i][0] = e.target.value;
                            setQueryParams(Object.fromEntries(entries));
                          }}
                          placeholder="key"
                          className="w-32 rounded border border-gray-300 px-2 py-1 text-xs dark:border-gray-600 dark:bg-gray-700 dark:text-white"
                        />
                        <span className="text-gray-400">=</span>
                        <input
                          value={v}
                          onChange={(e) => {
                            const entries = Object.entries(queryParams);
                            entries[i][1] = e.target.value;
                            setQueryParams(Object.fromEntries(entries));
                          }}
                          placeholder="value"
                          className="flex-1 rounded border border-gray-300 px-2 py-1 text-xs dark:border-gray-600 dark:bg-gray-700 dark:text-white"
                        />
                      </div>
                    ))}
                    <button
                      onClick={() => setQueryParams(prev => ({ ...prev, [`param${Object.keys(prev).length}`]: "" }))}
                      className="text-xs text-brand-600 hover:underline"
                    >
                      + Add query param
                    </button>
                  </div>
                </div>
              )}

              {/* Request body (for POST/PUT/PATCH) */}
              {["POST", "PUT", "PATCH"].includes(selectedEndpoint.method) && (
                <div>
                  <h3 className="mb-2 text-xs font-semibold uppercase text-gray-400">Request Body (JSON)</h3>
                  <textarea
                    value={bodyText}
                    onChange={(e) => setBodyText(e.target.value)}
                    placeholder='{"key": "value"}'
                    className="h-40 w-full rounded-lg border border-gray-300 p-3 font-mono text-xs dark:border-gray-600 dark:bg-gray-700 dark:text-white"
                  />
                </div>
              )}

              {/* Send button */}
              <button
                onClick={tryRequest}
                disabled={loading}
                className="flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
              >
                {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Play className="h-4 w-4" />}
                {loading ? "Sending..." : "Send Request"}
              </button>

              {/* Response */}
              {response !== null && (
                <div>
                  <div className="mb-2 flex items-center gap-2">
                    <h3 className="text-xs font-semibold uppercase text-gray-400">Response</h3>
                    {status !== null && (
                      <span className={`flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${
                        status >= 200 && status < 300
                          ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400"
                          : "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400"
                      }`}>
                        {status >= 200 && status < 300 ? <CheckCircle className="h-3 w-3" /> : <XCircle className="h-3 w-3" />}
                        {status} {loading ? "" : ""}
                      </span>
                    )}
                    <button
                      onClick={() => navigator.clipboard.writeText(response)}
                      className="ml-auto text-xs text-gray-400 hover:text-gray-600"
                    >
                      <Copy className="h-3 w-3" />
                    </button>
                  </div>
                  <pre className="max-h-96 overflow-auto rounded-lg border border-gray-200 bg-gray-50 p-4 text-xs dark:border-gray-700 dark:bg-gray-900">
                    {response}
                  </pre>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
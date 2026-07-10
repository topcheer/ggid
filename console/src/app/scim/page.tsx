"use client";

import { useEffect, useState } from "react";
import { useApi } from "@/lib/api";
import { Copy, RefreshCw, Check, Server, Key, BookOpen, Users } from "lucide-react";

export default function SCIMPage() {
  const { API_BASE, TENANT_ID } = useApi();
  const [bearerToken, setBearerToken] = useState("");
  const [copied, setCopied] = useState(false);
  const [msg, setMsg] = useState<string | null>(null);

  const scimEndpoint = `${API_BASE}/scim/v2`;

  useEffect(() => {
    // Generate a display token (in production this would come from the API)
    setBearerToken("[Configure in Settings \u2192 Security]");
  }, []);

  const handleCopy = (text: string) => {
    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const endpoints = [
    { method: "GET", path: "/scim/v2/Users", desc: "List all users (paginated)" },
    { method: "POST", path: "/scim/v2/Users", desc: "Create a new user" },
    { method: "GET", path: "/scim/v2/Users/{id}", desc: "Get a specific user" },
    { method: "PUT", path: "/scim/v2/Users/{id}", desc: "Update user attributes" },
    { method: "PATCH", path: "/scim/v2/Users/{id}", desc: "Patch user (add/remove/replace)" },
    { method: "DELETE", path: "/scim/v2/Users/{id}", desc: "Deactivate a user" },
    { method: "GET", path: "/scim/v2/Groups", desc: "List all groups" },
    { method: "POST", path: "/scim/v2/Groups", desc: "Create a group" },
  ];

  const provisioningGuides = [
    {
      name: "Okta",
      steps: [
        "Go to Applications \u2192 Browse App Catalog",
        "Search for \"SCIM\" or create custom SCIM app",
        `Set SCIM Base URL: ${scimEndpoint}`,
        "Set authentication: Bearer Token",
        "Paste your bearer token from Settings \u2192 Security",
        "Map user attributes (userName, emails, displayName)",
        "Enable provisioning: Push Users, Push Groups",
      ],
    },
    {
      name: "Azure AD (Entra ID)",
      steps: [
        "Enterprise Applications \u2192 New Application",
        "Create Non-gallery application",
        "Provisioning \u2192 Automatic",
        `Tenant URL: ${scimEndpoint}`,
        "Secret Token: your bearer token",
        "Test connection, then start provisioning",
      ],
    },
    {
      name: "Google Workspace",
      steps: [
        "Admin Console \u2192 Apps \u2192 OAuth clients",
        "Configure SCIM provisioning",
        `SCIM URL: ${scimEndpoint}`,
        "Set bearer token authentication",
      ],
    },
  ];

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold">SCIM Provisioning</h1>
        <p className="text-sm text-gray-500">
          Configure automated user provisioning with SCIM 2.0 (RFC 7643/7644)
        </p>
      </div>

      {/* Connection Details */}
      <div className="mb-6 grid gap-4 lg:grid-cols-2">
        <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
          <div className="mb-4 flex items-center gap-2">
            <Server className="h-5 w-5 text-brand-600" />
            <h3 className="text-sm font-semibold">SCIM Endpoint</h3>
          </div>
          <div className="flex items-center gap-2">
            <code className="flex-1 truncate rounded-lg bg-gray-50 px-3 py-2 text-sm">
              {scimEndpoint}
            </code>
            <button
              onClick={() => handleCopy(scimEndpoint)}
              className="flex h-9 w-9 items-center justify-center rounded-lg border border-gray-200 hover:bg-gray-50"
            >
              {copied ? <Check className="h-4 w-4 text-green-500" /> : <Copy className="h-4 w-4 text-gray-400" />}
            </button>
          </div>
          <p className="mt-2 text-xs text-gray-400">
            Use this URL when configuring your IdP (Okta, Azure AD, etc.)
          </p>
        </div>

        <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
          <div className="mb-4 flex items-center gap-2">
            <Key className="h-5 w-5 text-brand-600" />
            <h3 className="text-sm font-semibold">Bearer Token</h3>
          </div>
          <div className="flex items-center gap-2">
            <code className="flex-1 truncate rounded-lg bg-gray-50 px-3 py-2 text-sm text-gray-500">
              {bearerToken}
            </code>
          </div>
          <p className="mt-2 text-xs text-gray-400">
            Generate a token in Settings \u2192 Security. Include as
            <code className="ml-1 rounded bg-gray-100 px-1">Authorization: Bearer {"<token>"}</code>
          </p>
        </div>
      </div>

      {/* Endpoints */}
      <div className="mb-6 rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
        <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold">
          <BookOpen className="h-4 w-4 text-brand-600" />
          SCIM 2.0 Endpoints
        </h3>
        <div className="overflow-hidden rounded-lg border border-gray-100">
          <table className="w-full">
            <thead className="border-b border-gray-100 bg-gray-50">
              <tr>
                <th className="px-3 py-2 text-left text-xs font-medium text-gray-500">Method</th>
                <th className="px-3 py-2 text-left text-xs font-medium text-gray-500">Path</th>
                <th className="px-3 py-2 text-left text-xs font-medium text-gray-500">Description</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-50">
              {endpoints.map((ep) => (
                <tr key={`${ep.method}-${ep.path}`} className="hover:bg-gray-50">
                  <td className="px-3 py-2">
                    <span className={`rounded px-2 py-0.5 text-xs font-bold ${
                      ep.method === "GET" ? "bg-blue-100 text-blue-700" :
                      ep.method === "POST" ? "bg-green-100 text-green-700" :
                      ep.method === "DELETE" ? "bg-red-100 text-red-700" :
                      "bg-amber-100 text-amber-700"
                    }`}>{ep.method}</span>
                  </td>
                  <td className="px-3 py-2 text-sm font-mono">{ep.path}</td>
                  <td className="px-3 py-2 text-sm text-gray-600">{ep.desc}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* IdP Configuration Guides */}
      <div className="grid gap-4 lg:grid-cols-3">
        {provisioningGuides.map((guide) => (
          <div key={guide.name} className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
            <div className="mb-3 flex items-center gap-2">
              <Users className="h-4 w-4 text-brand-600" />
              <h4 className="text-sm font-semibold">{guide.name}</h4>
            </div>
            <ol className="space-y-1.5">
              {guide.steps.map((step, i) => (
                <li key={i} className="flex gap-2 text-xs text-gray-600">
                  <span className="flex h-4 w-4 flex-shrink-0 items-center justify-center rounded-full bg-gray-100 text-xs font-medium text-gray-400">
                    {i + 1}
                  </span>
                  <span>{step}</span>
                </li>
              ))}
            </ol>
          </div>
        ))}
      </div>

      {msg && (
        <div className="mt-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700">{msg}</div>
      )}
    </div>
  );
}

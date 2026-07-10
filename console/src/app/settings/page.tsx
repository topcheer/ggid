"use client";

import { useEffect, useState } from "react";

const API_BASE =
  process.env.NEXT_PUBLIC_GGID_API || "http://localhost:8080";
const TENANT_ID =
  process.env.NEXT_PUBLIC_TENANT_ID ||
  "00000000-0000-0000-0000-000000000001";

function getToken() {
  if (typeof window === "undefined") return "";
  return localStorage.getItem("ggid_access_token") || "";
}

export default function SettingsPage() {
  const [oidcConfig, setOidcConfig] = useState<Record<string, unknown> | null>(
    null,
  );
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // Fetch OIDC discovery
    fetch(`${API_BASE}/oauth/.well-known/openid-configuration`)
      .then((r) => (r.ok ? r.json() : null))
      .then((d) => setOidcConfig(d))
      .catch(() => setOidcConfig(null))
      .finally(() => setLoading(false));
  }, []);

  const sections = [
    {
      title: "Tenant Information",
      items: [
        { label: "Tenant ID", value: TENANT_ID },
        { label: "Plan", value: "Enterprise" },
        { label: "Status", value: "Active" },
      ],
    },
    {
      title: "System Information",
      items: [
        { label: "Version", value: "1.0.0-dev" },
        { label: "License", value: "Apache 2.0" },
        { label: "API Gateway", value: API_BASE },
      ],
    },
  ];

  return (
    <div>
      <h1 className="mb-6 text-2xl font-bold">Settings</h1>

      {sections.map((section) => (
        <div
          key={section.title}
          className="mb-6 rounded-xl border border-gray-200 bg-white p-6 shadow-sm"
        >
          <h2 className="mb-4 text-lg font-semibold">{section.title}</h2>
          <div className="space-y-3">
            {section.items.map((item) => (
              <div
                key={item.label}
                className="flex items-center justify-between border-b border-gray-100 pb-3 last:border-0"
              >
                <span className="text-sm text-gray-500">{item.label}</span>
                <span className="text-sm font-medium">{item.value}</span>
              </div>
            ))}
          </div>
        </div>
      ))}

      {/* OIDC Configuration */}
      <div className="mb-6 rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
        <h2 className="mb-4 text-lg font-semibold">OIDC Configuration</h2>
        {loading ? (
          <p className="text-gray-500">Loading...</p>
        ) : oidcConfig ? (
          <div className="space-y-2">
            {Object.entries(oidcConfig).map(([key, value]) => (
              <div key={key} className="flex flex-col gap-1">
                <span className="text-xs font-medium text-gray-500">{key}</span>
                <span className="break-all text-sm text-gray-800">
                  {String(value)}
                </span>
              </div>
            ))}
          </div>
        ) : (
          <p className="text-sm text-gray-400">
            OIDC discovery endpoint not available
          </p>
        )}
      </div>

      {/* LDAP Status */}
      <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
        <h2 className="mb-4 text-lg font-semibold">LDAP / Active Directory</h2>
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <span className="text-sm text-gray-500">Status</span>
            <span className="rounded-full bg-green-100 px-2 py-0.5 text-xs font-medium text-green-700">
              Connected
            </span>
          </div>
          <div className="flex items-center justify-between">
            <span className="text-sm text-gray-500">Server</span>
            <span className="text-sm font-medium">ldap://ldap:389</span>
          </div>
          <div className="flex items-center justify-between">
            <span className="text-sm text-gray-500">Base DN</span>
            <span className="text-sm font-medium">dc=corp,dc=local</span>
          </div>
        </div>
      </div>
    </div>
  );
}

"use client";

import { useState, useEffect } from "react";
import { useApi } from "@/lib/api";
import {
  ShieldCheck,
  Download,
  Copy,
  Check,
  Loader2,
  UploadCloud,
  FileText,
  KeyRound,
  RefreshCw,
} from "lucide-react";

const GATEWAY_BASE =
  process.env.NEXT_PUBLIC_GGID_API || "https://gateway.ggid.dev";

type GgidField = "username" | "email" | "displayName";

const FIELD_OPTIONS: GgidField[] = ["username", "email", "displayName"];

interface SamlConfig {
  idp_metadata_url: string;
  idp_entity_id: string;
  idp_sso_url: string;
  idp_cert: string;
  attribute_mapping: Record<string, GgidField>;
  cert_fingerprint: string;
  cert_expiry: string;
  cert_pem: string;
}

const defaultConfig: SamlConfig = {
  idp_metadata_url: "",
  idp_entity_id: "",
  idp_sso_url: "",
  idp_cert: "",
  attribute_mapping: {
    "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name": "username",
    "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress": "email",
    "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/displayname": "displayName",
  },
  cert_fingerprint: "",
  cert_expiry: "",
  cert_pem: "",
};

const entityId = `${GATEWAY_BASE}/saml/metadata`;
const acsUrl = `${GATEWAY_BASE}/saml/acs`;

export default function SamlSettingsPage() {
  const { apiFetch } = useApi();
  const [config, setConfig] = useState<SamlConfig>(defaultConfig);
  const [msg, setMsg] = useState<{ type: "success" | "error"; text: string } | null>(null);
  const [saving, setSaving] = useState(false);
  const [importing, setImporting] = useState(false);
  const [copiedField, setCopiedField] = useState<string | null>(null);

  // Load config from localStorage or API
  useEffect(() => {
    const stored = typeof window !== "undefined" ? localStorage.getItem("ggid_saml_config") : null;
    if (stored) {
      try {
        setConfig({ ...defaultConfig, ...JSON.parse(stored) });
      } catch {
        // ignore
      }
    }
  }, []);

  useEffect(() => {
    if (msg) {
      const t = setTimeout(() => setMsg(null), 4000);
      return () => clearTimeout(t);
    }
  }, [msg]);

  const handleImport = async () => {
    if (!config.idp_metadata_url.trim()) {
      setMsg({ type: "error", text: "Please enter an IdP metadata URL" });
      return;
    }
    setImporting(true);
    try {
      const data = await apiFetch<{
        entity_id?: string;
        sso_url?: string;
        cert?: string;
        cert_fingerprint?: string;
      }>("/api/v1/saml/import-metadata", {
        method: "POST",
        body: JSON.stringify({ metadata_url: config.idp_metadata_url }),
      });
      setConfig((prev) => ({
        ...prev,
        idp_entity_id: data.entity_id || prev.idp_entity_id,
        idp_sso_url: data.sso_url || prev.idp_sso_url,
        idp_cert: data.cert || prev.idp_cert,
        cert_fingerprint: data.cert_fingerprint || prev.cert_fingerprint,
      }));
      setMsg({ type: "success", text: "IdP metadata imported successfully" });
    } catch (err) {
      setMsg({
        type: "error",
        text: err instanceof Error ? err.message : "Failed to import metadata",
      });
    } finally {
      setImporting(false);
    }
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      await apiFetch("/api/v1/saml/sp-config", {
        method: "POST",
        body: JSON.stringify(config),
      });
      setMsg({ type: "success", text: "SAML configuration saved" });
    } catch {
      localStorage.setItem("ggid_saml_config", JSON.stringify(config));
      setMsg({ type: "success", text: "Endpoint unavailable — saved locally" });
    } finally {
      setSaving(false);
    }
  };

  const copyToClipboard = async (text: string, field: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopiedField(field);
      setTimeout(() => setCopiedField(null), 2000);
    } catch {
      // clipboard unavailable
    }
  };

  const downloadFile = (filename: string, content: string, mime: string) => {
    const blob = new Blob([content], { type: mime });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(url);
  };

  const downloadCert = () => {
    const pem = config.cert_pem || config.idp_cert || "";
    downloadFile("saml-sp-cert.pem", pem || "-----BEGIN CERTIFICATE-----\n-----END CERTIFICATE-----", "application/x-pem-file");
  };

  const downloadSpMetadata = () => {
    const xml = `<?xml version="1.0" encoding="UTF-8"?>
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" entityID="${entityId}">
  <SPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</NameIDFormat>
    <AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="${acsUrl}" index="0" isDefault="true"/>
  </SPSSODescriptor>
</EntityDescriptor>`;
    downloadFile("saml-sp-metadata.xml", xml, "application/xml");
  };

  const updateMapping = (samlAttr: string, field: GgidField) => {
    setConfig((prev) => ({
      ...prev,
      attribute_mapping: { ...prev.attribute_mapping, [samlAttr]: field },
    }));
  };

  const readOnlyField = (label: string, value: string, fieldKey: string) => (
    <div>
      <label className="mb-1 block text-xs font-medium text-gray-500">{label}</label>
      <div className="flex items-center gap-2">
        <input
          readOnly
          value={value}
          className="flex-1 rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 font-mono text-sm text-gray-600 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-400"
        />
        <button
          onClick={() => copyToClipboard(value, fieldKey)}
          className="rounded-lg border border-gray-300 p-2 text-gray-500 hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700"
          title="Copy"
        >
          {copiedField === fieldKey ? (
            <Check className="h-4 w-4 text-green-500" />
          ) : (
            <Copy className="h-4 w-4" />
          )}
        </button>
      </div>
    </div>
  );

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-2xl font-bold dark:text-gray-100">
          <ShieldCheck className="h-6 w-6 text-brand-600" /> SAML Service Provider
        </h1>
        <button
          onClick={handleSave}
          disabled={saving}
          className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
        >
          {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />} Save
        </button>
      </div>

      {msg && (
        <div
          className={`mb-4 rounded-lg border p-3 text-sm ${
            msg.type === "success"
              ? "border-green-200 bg-green-50 text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400"
              : "border-red-200 bg-red-50 text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400"
          }`}
        >
          {msg.text}
        </div>
      )}

      <div className="space-y-6">
        {/* IdP Import Section */}
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <h2 className="mb-4 text-lg font-semibold dark:text-gray-100">Identity Provider</h2>

          <div className="space-y-4">
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">
                IdP Metadata URL
              </label>
              <div className="flex gap-2">
                <input
                  value={config.idp_metadata_url}
                  onChange={(e) =>
                    setConfig({ ...config, idp_metadata_url: e.target.value })
                  }
                  placeholder="https://idp.example.com/metadata"
                  className="flex-1 rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
                />
                <button
                  onClick={handleImport}
                  disabled={importing}
                  className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
                >
                  {importing ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <UploadCloud className="h-4 w-4" />
                  )}
                  Import
                </button>
              </div>
            </div>

            {config.idp_entity_id && (
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <div>
                  <label className="mb-1 block text-xs font-medium text-gray-500">
                    IdP Entity ID
                  </label>
                  <input
                    readOnly
                    value={config.idp_entity_id}
                    className="w-full rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 font-mono text-xs text-gray-600 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-400"
                  />
                </div>
                <div>
                  <label className="mb-1 block text-xs font-medium text-gray-500">
                    IdP SSO URL
                  </label>
                  <input
                    readOnly
                    value={config.idp_sso_url}
                    className="w-full rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 font-mono text-xs text-gray-600 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-400"
                  />
                </div>
              </div>
            )}
          </div>
        </div>

        {/* SP Endpoints Section */}
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <h2 className="mb-4 flex items-center gap-2 text-lg font-semibold dark:text-gray-100">
            <FileText className="h-5 w-5 text-brand-600" /> Service Provider Endpoints
          </h2>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            {readOnlyField("Entity ID (auto-generated)", entityId, "entityId")}
            {readOnlyField("ACS URL (Assertion Consumer)", acsUrl, "acsUrl")}
          </div>
        </div>

        {/* Attribute Mapping Section */}
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <h2 className="mb-4 text-lg font-semibold dark:text-gray-100">Attribute Mapping</h2>
          <p className="mb-4 text-xs text-gray-500">
            Map incoming SAML attributes to GGID user fields.
          </p>
          <div className="overflow-hidden rounded-lg border border-gray-200 dark:border-gray-600">
            <table className="w-full text-left text-sm">
              <thead className="border-b border-gray-200 bg-gray-50 dark:border-gray-600 dark:bg-gray-800">
                <tr>
                  <th className="px-4 py-2 font-semibold text-gray-600 dark:text-gray-300">
                    SAML Attribute
                  </th>
                  <th className="px-4 py-2 font-semibold text-gray-600 dark:text-gray-300">
                    GGID Field
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
                {Object.entries(config.attribute_mapping).map(([samlAttr, ggidField]) => (
                  <tr key={samlAttr}>
                    <td className="px-4 py-2.5">
                      <span className="font-mono text-xs text-gray-600 dark:text-gray-400">
                        {samlAttr}
                      </span>
                    </td>
                    <td className="px-4 py-2.5">
                      <select
                        value={ggidField}
                        onChange={(e) => updateMapping(samlAttr, e.target.value as GgidField)}
                        className="rounded-lg border border-gray-300 px-3 py-1.5 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
                      >
                        {FIELD_OPTIONS.map((opt) => (
                          <option key={opt} value={opt}>
                            {opt}
                          </option>
                        ))}
                      </select>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        {/* Certificate Section */}
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <h2 className="mb-4 flex items-center gap-2 text-lg font-semibold dark:text-gray-100">
            <KeyRound className="h-5 w-5 text-brand-600" /> Signing Certificate
          </h2>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">
                Certificate Fingerprint
              </label>
              <input
                readOnly
                value={config.cert_fingerprint || "Not configured"}
                className="w-full rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 font-mono text-xs text-gray-600 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-400"
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">
                Expiry Date
              </label>
              <input
                readOnly
                value={config.cert_expiry || "—"}
                className="w-full rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 text-sm text-gray-600 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-400"
              />
            </div>
          </div>
          <div className="mt-4 flex gap-3">
            <button
              onClick={downloadCert}
              className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
            >
              <Download className="h-4 w-4" /> Download .pem
            </button>
          </div>
        </div>

        {/* SP Metadata Download */}
        <div className="flex items-center justify-between rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <div>
            <h3 className="text-sm font-semibold dark:text-gray-100">SP Metadata</h3>
            <p className="text-xs text-gray-500">
              Download the SP metadata XML to register with your IdP.
            </p>
          </div>
          <button
            onClick={downloadSpMetadata}
            className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
          >
            <Download className="h-4 w-4" /> Download XML
          </button>
        </div>
      </div>
    </div>
  );
}

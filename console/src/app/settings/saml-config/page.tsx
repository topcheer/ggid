"use client";
import { useState, useEffect } from "react";
import {
  FileText, Download, Upload, Save, Loader2, CheckCircle2,
  AlertCircle, Globe, Shield, Copy,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { usePageTitle } from "@/lib/usePageTitle";
import { authHeader } from "@/lib/auth-helpers";
import { API_BASE_URL } from "@/lib/api-config";

const API_BASE = API_BASE_URL;

export default function SAMLConfigPage() {
  usePageTitle("SAML Configuration");
  const t = useTranslations();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  // Config state
  const [entityId, setEntityId] = useState("");
  const [acsUrl, setAcsUrl] = useState("");
  const [idpMetadataXml, setIdpMetadataXml] = useState("");
  const [idpMetadataUrl, setIdpMetadataUrl] = useState("");

  // SP metadata (read-only, displayed for download)
  const [spMetadata, setSpMetadata] = useState("");
  const [spCopied, setSpCopied] = useState(false);

  useEffect(() => {
    (async () => {
      try {
        const [configRes, spRes] = await Promise.all([
          fetch(`${API_BASE}/api/v1/system/config?key=saml_config`, { headers: { ...authHeader() } }),
          fetch(`${API_BASE}/saml/metadata`, { headers: { ...authHeader() } }).catch(() => null),
        ]);

        if (configRes.ok) {
          const d = await configRes.json();
          const cfg = d.saml_config || d.value || {};
          setEntityId(cfg.entity_id || cfg.entityId || "");
          setAcsUrl(cfg.acs_url || cfg.acsUrl || "");
          setIdpMetadataXml(cfg.idp_metadata_xml || "");
          setIdpMetadataUrl(cfg.idp_metadata_url || "");
        }

        if (spRes && spRes.ok) {
          const xml = await spRes.text();
          setSpMetadata(xml);
        } else {
          // Generate a placeholder SP metadata for display
          const host = typeof window !== "undefined" ? window.location.hostname : "ggid.local";
          setSpMetadata(generateSpMetadata(host));
        }
      } catch { /* config not yet set */ }
      setLoading(false);
    })();
  }, []);

  const handleSave = async () => {
    setSaving(true);
    setError("");
    setSuccess("");
    try {
      const res = await fetch(`${API_BASE}/api/v1/system/config`, {
        method: "PUT",
        headers: { "Content-Type": "application/json", ...authHeader() },
        body: JSON.stringify({
          key: "saml_config",
          value: {
            entity_id: entityId,
            acs_url: acsUrl,
            idp_metadata_xml: idpMetadataXml,
            idp_metadata_url: idpMetadataUrl,
          },
        }),
      });
      if (res.ok) {
        setSuccess("SAML configuration saved successfully.");
        setTimeout(() => setSuccess(""), 3000);
      } else {
        const d = await res.json().catch(() => ({}));
        setError(d.error?.message || "Failed to save SAML configuration");
      }
    } catch {
      setError("Network error");
    }
    setSaving(false);
  };

  const handleDownloadSp = () => {
    const blob = new Blob([spMetadata], { type: "application/xml" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "sp-metadata.xml";
    a.click();
    URL.revokeObjectURL(url);
  };

  const handleCopySp = () => {
    navigator.clipboard.writeText(spMetadata);
    setSpCopied(true);
    setTimeout(() => setSpCopied(false), 2000);
  };

  const handleUploadIdp = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = (ev) => {
      setIdpMetadataXml(ev.target?.result as string);
    };
    reader.readAsText(file);
  };

  if (loading) {
    return <div className="flex justify-center py-20"><Loader2 className="w-8 h-8 animate-spin text-blue-600" /></div>;
  }

  return (
    <div className="mx-auto max-w-3xl space-y-6 p-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">SAML Configuration</h1>
        <p className="mt-1 text-sm text-gray-500">
          Configure SAML 2.0 SSO integration with your Identity Provider (IdP).
        </p>
      </div>

      {error && (
        <div className="flex items-center gap-2 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950">
          <AlertCircle className="h-4 w-4 shrink-0" /> {error}
        </div>
      )}
      {success && (
        <div className="flex items-center gap-2 rounded-lg border border-green-200 bg-green-50 px-4 py-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950">
          <CheckCircle2 className="h-4 w-4 shrink-0" /> {success}
        </div>
      )}

      {/* SP Metadata (download/share) */}
      <div className="rounded-xl border border-gray-200 bg-white p-6 dark:border-gray-800 dark:bg-gray-900">
        <h2 className="mb-2 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400">
          <Download className="h-4 w-4" /> Service Provider Metadata
        </h2>
        <p className="mb-3 text-sm text-gray-500">
          Share this metadata with your IdP administrator to configure the SP connection.
        </p>
        <pre className="max-h-48 overflow-auto rounded-lg bg-gray-900 p-3 text-xs text-green-400 dark:bg-gray-950">
          {spMetadata || "Loading..."}
        </pre>
        <div className="mt-3 flex gap-2">
          <button onClick={handleDownloadSp} className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-1.5 text-sm dark:border-gray-700">
            <Download className="h-4 w-4" /> Download XML
          </button>
          <button onClick={handleCopySp} className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-1.5 text-sm dark:border-gray-700">
            {spCopied ? <CheckCircle2 className="h-4 w-4 text-green-500" /> : <Copy className="h-4 w-4" />}
            {spCopied ? "Copied!" : "Copy"}
          </button>
        </div>
      </div>

      {/* IdP Configuration */}
      <div className="rounded-xl border border-gray-200 bg-white p-6 dark:border-gray-800 dark:bg-gray-900">
        <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400">
          <Shield className="h-4 w-4" /> Identity Provider Configuration
        </h2>

        <div className="space-y-4">
          {/* Entity ID */}
          <div>
            <label className="text-sm font-medium text-gray-700 dark:text-gray-300">SP Entity ID</label>
            <input
              type="text"
              value={entityId}
              onChange={e => setEntityId(e.target.value)}
              placeholder="https://ggid-console.example.com/saml/metadata"
              className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-700 dark:bg-gray-800"
            />
            <p className="mt-1 text-xs text-gray-400">Unique identifier for this Service Provider.</p>
          </div>

          {/* ACS URL */}
          <div>
            <label className="text-sm font-medium text-gray-700 dark:text-gray-300">ACS URL</label>
            <input
              type="text"
              value={acsUrl}
              onChange={e => setAcsUrl(e.target.value)}
              placeholder="https://ggid-console.example.com/saml/acs"
              className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-700 dark:bg-gray-800"
            />
            <p className="mt-1 text-xs text-gray-400">Assertion Consumer Service URL where IdP POSTs SAML responses.</p>
          </div>

          {/* IdP Metadata URL */}
          <div>
            <label className="text-sm font-medium text-gray-700 dark:text-gray-300">IdP Metadata URL (optional)</label>
            <input
              type="text"
              value={idpMetadataUrl}
              onChange={e => setIdpMetadataUrl(e.target.value)}
              placeholder="https://idp.example.com/metadata"
              className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-700 dark:bg-gray-800"
            />
            <p className="mt-1 text-xs text-gray-400">GGID will fetch and auto-update IdP configuration from this URL.</p>
          </div>

          {/* IdP Metadata XML */}
          <div>
            <label className="text-sm font-medium text-gray-700 dark:text-gray-300">IdP Metadata XML</label>
            <div className="mt-1 flex gap-2">
              <textarea
                value={idpMetadataXml}
                onChange={e => setIdpMetadataXml(e.target.value)}
                rows={6}
                placeholder="Paste IdP metadata XML here, or upload a file..."
                className="flex-1 rounded-lg border border-gray-300 px-3 py-2 font-mono text-xs dark:border-gray-700 dark:bg-gray-800"
              />
            </div>
            <label className="mt-2 inline-flex cursor-pointer items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-1.5 text-sm dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-800">
              <Upload className="h-4 w-4" /> Upload XML File
              <input type="file" accept=".xml,text/xml" onChange={handleUploadIdp} className="hidden" />
            </label>
          </div>
        </div>

        <div className="mt-6 flex justify-end">
          <button
            onClick={handleSave}
            disabled={saving}
            className="flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
          >
            {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
            Save Configuration
          </button>
        </div>
      </div>
    </div>
  );
}

function generateSpMetadata(host: string): string {
  return `<?xml version="1.0" encoding="UTF-8"?>
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" entityID="https://${host}/saml/metadata">
  <SPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</NameIDFormat>
    <AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
      Location="https://${host}/saml/acs" index="0" />
    <SingleLogoutService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"
      Location="https://${host}/saml/slo" />
  </SPSSODescriptor>
</EntityDescriptor>`;
}

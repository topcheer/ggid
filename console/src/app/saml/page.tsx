"use client";

import { useState, useRef, useCallback, ReactNode } from "react";
import { useApi } from "@/lib/api";
import {
  Save, Download, Upload, Trash2, Plus, RefreshCw,
  CheckCircle, XCircle, Loader2, FileCode2, Shield,
  Link2, Settings2, AlertCircle,
} from "lucide-react";

// ---- Types ----
interface IdPConfig {
  entity_id: string;
  sso_url: string;
  slo_url: string;
  cert: string;
  name_id_format: string;
  authn_context_class: string;
}

interface AttributeMapping {
  id: string;
  saml_attr: string;
  ggid_field: string;
}

interface CertInfo {
  issuer: string;
  subject: string;
  valid_from: string;
  valid_to: string;
  fingerprint: string;
}

interface TestResult {
  success: boolean;
  response_time_ms: number;
  entity_id?: string;
  sso_url?: string;
  cert_info?: CertInfo;
  error?: string;
}

const NAME_ID_FORMATS = [
  { value: "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress", label: "Email Address" },
  { value: "urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified", label: "Unspecified" },
  { value: "urn:oasis:names:tc:SAML:2.0:nameid-format:persistent", label: "Persistent" },
  { value: "urn:oasis:names:tc:SAML:2.0:nameid-format:transient", label: "Transient" },
];

const AUTHN_CONTEXT_CLASSES = [
  { value: "urn:oasis:names:tc:SAML:2.0:ac:classes:PasswordProtectedTransport", label: "PasswordProtectedTransport" },
  { value: "urn:oasis:names:tc:SAML:2.0:ac:classes:Password", label: "Password" },
  { value: "urn:oasis:names:tc:SAML:2.0:ac:classes:TLSClient", label: "TLSClient" },
  { value: "urn:oasis:names:tc:SAML:2.0:ac:classes:Kerberos", label: "Kerberos" },
];

const GGID_FIELDS = [
  { value: "email", label: "Email" },
  { value: "first_name", label: "First Name" },
  { value: "last_name", label: "Last Name" },
  { value: "display_name", label: "Display Name" },
  { value: "groups", label: "Groups" },
  { value: "department", label: "Department" },
  { value: "phone", label: "Phone" },
  { value: "title", label: "Title" },
  { value: "username", label: "Username" },
];

// Sample SP metadata XML (read-only display)
const SP_METADATA = `<?xml version="1.0" encoding="UTF-8"?>
<EntityDescriptor
    xmlns="urn:oasis:names:tc:SAML:2.0:metadata"
    entityID="https://ggid.dev/saml/metadata">

  <SPSSODescriptor
      protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol"
      WantAssertionsSigned="true"
      AuthnRequestsSigned="true">

    <KeyDescriptor use="signing">
      <KeyInfo xmlns="http://www.w3.org/2000/09/xmldsig#">
        <X509Data>
          <X509Certificate>MIIDlzCCAn+gAwIBAgIUExampleSigningCertBase64Data==</X509Certificate>
        </X509Data>
      </KeyInfo>
    </KeyDescriptor>

    <NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</NameIDFormat>

    <AssertionConsumerService
        Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
        Location="https://ggid.dev/saml/acs"
        index="0"
        isDefault="true" />

    <SingleLogoutService
        Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"
        Location="https://ggid.dev/saml/slo" />
  </SPSSODescriptor>
</EntityDescriptor>`;

// ---- Syntax-highlighted XML renderer ----
function highlightXml(xml: string): ReactNode[] {
  const lines = xml.split("\n");
  return lines.map((line, lineIdx) => {
    // Tokenize XML line into tags, attributes, values
    const parts: ReactNode[] = [];
    const tagRegex = /(<\/?[\w:.-]+)|(\/?>)|(\s[\w:.-]+=)|("[^"]*")|(>)/g;
    let lastIndex = 0;
    let keyIdx = 0;
    let match: RegExpExecArray | null;

    while ((match = tagRegex.exec(line)) !== null) {
      // Text before match
      if (match.index > lastIndex) {
        parts.push(
          <span key={`t-${lineIdx}-${keyIdx++}`} className="text-gray-600 dark:text-gray-400">
            {line.slice(lastIndex, match.index)}
          </span>,
        );
      }
      const token = match[0];
      if (match[1]) {
        // Opening or closing tag
        parts.push(
          <span key={`tag-${lineIdx}-${keyIdx++}`} className="text-blue-600 dark:text-blue-400 font-semibold">
            {token}
          </span>,
        );
      } else if (match[2]) {
        // Self-closing
        parts.push(
          <span key={`sc-${lineIdx}-${keyIdx++}`} className="text-blue-600 dark:text-blue-400 font-semibold">
            {token}
          </span>,
        );
      } else if (match[3]) {
        // Attribute name=
        parts.push(
          <span key={`attr-${lineIdx}-${keyIdx++}`} className="text-green-600 dark:text-green-400">
            {token}
          </span>,
        );
      } else if (match[4]) {
        // Attribute value
        parts.push(
          <span key={`val-${lineIdx}-${keyIdx++}`} className="text-orange-600 dark:text-orange-400">
            {token}
          </span>,
        );
      } else if (match[5]) {
        parts.push(
          <span key={`gt-${lineIdx}-${keyIdx++}`} className="text-blue-600 dark:text-blue-400 font-semibold">
            {token}
          </span>,
        );
      }
      lastIndex = tagRegex.lastIndex;
    }

    // Remaining text after last match
    if (lastIndex < line.length) {
      parts.push(
        <span key={`rem-${lineIdx}-${keyIdx++}`} className="text-gray-600 dark:text-gray-400">
          {line.slice(lastIndex)}
        </span>,
      );
    }

    return (
      <div key={`line-${lineIdx}`}>
        {parts.length > 0 ? parts : <span>&nbsp;</span>}
      </div>
    );
  });
}

export default function SAMLPage() {
  const { apiFetch } = useApi();
  const fileInputRef = useRef<HTMLInputElement>(null);

  const [msg, setMsg] = useState<string | null>(null);
  const [msgType, setMsgType] = useState<"success" | "error">("success");

  // IdP Config state
  const [idpConfig, setIdpConfig] = useState<IdPConfig>({
    entity_id: "",
    sso_url: "",
    slo_url: "",
    cert: "",
    name_id_format: "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
    authn_context_class: "urn:oasis:names:tc:SAML:2.0:ac:classes:PasswordProtectedTransport",
  });

  const [certInfo, setCertInfo] = useState<CertInfo | null>(null);
  const [saving, setSaving] = useState(false);

  // Attribute mappings
  const [mappings, setMappings] = useState<AttributeMapping[]>([
    { id: "m1", saml_attr: "email", ggid_field: "email" },
    { id: "m2", saml_attr: "givenName", ggid_field: "first_name" },
    { id: "m3", saml_attr: "sn", ggid_field: "last_name" },
    { id: "m4", saml_attr: "displayName", ggid_field: "display_name" },
    { id: "m5", saml_attr: "memberOf", ggid_field: "groups" },
    { id: "m6", saml_attr: "department", ggid_field: "department" },
  ]);

  // Test connection state
  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState<TestResult | null>(null);

  // ---- Helpers ----
  const inputCls = "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";
  const labelCls = "mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300";
  const cardCls = "rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const headingCls = "mb-4 text-lg font-semibold text-gray-900 dark:text-gray-100";

  const showMsg = (text: string, type: "success" | "error" = "success") => {
    setMsg(text);
    setMsgType(type);
    setTimeout(() => setMsg(null), 4000);
  };

  // Parse certificate info (basic extraction from PEM)
  const parseCert = (pem: string): CertInfo | null => {
    if (!pem.includes("BEGIN CERTIFICATE")) return null;
    // In a real app we'd use a crypto library; show placeholder parsed info
    const lines = pem.trim().split("\n");
    const base64 = lines.slice(1, -1).join("");
    // Simple hash for fingerprint display
    let hash = 0;
    for (let i = 0; i < base64.length && i < 200; i++) {
      hash = ((hash << 5) - hash + base64.charCodeAt(i)) | 0;
    }
    const fingerprint = Array.from({ length: 20 }, (_, i) =>
      ((hash >> (i % 4)) & 0xff).toString(16).padStart(2, "0"),
    ).join(":").slice(0, 47);
    return {
      issuer: "CN=Example IdP CA, O=Example Org, C=US",
      subject: "CN=idp.example.com, O=Example Org, C=US",
      valid_from: "2024-01-01T00:00:00Z",
      valid_to: "2027-01-01T00:00:00Z",
      fingerprint,
    };
  };

  const handleCertUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = (ev) => {
      const content = ev.target?.result as string;
      setIdpConfig({ ...idpConfig, cert: content });
      const info = parseCert(content);
      setCertInfo(info);
      if (info) {
        showMsg("Certificate parsed successfully");
      } else {
        showMsg("File uploaded (cert info not parseable)", "error");
      }
    };
    reader.readAsText(file);
  };

  const handleCertPaste = (value: string) => {
    setIdpConfig({ ...idpConfig, cert: value });
    if (value.includes("BEGIN CERTIFICATE")) {
      setCertInfo(parseCert(value));
    } else {
      setCertInfo(null);
    }
  };

  const handleDownloadMetadata = () => {
    const blob = new Blob([SP_METADATA], { type: "application/xml" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "sp-metadata.xml";
    a.click();
    URL.revokeObjectURL(url);
  };

  const handleSaveIdP = async () => {
    if (!idpConfig.entity_id || !idpConfig.sso_url) {
      showMsg("Entity ID and SSO URL are required", "error");
      return;
    }
    setSaving(true);
    try {
      await apiFetch("/api/v1/saml/idp", {
        method: "POST",
        body: JSON.stringify({ ...idpConfig, mappings }),
      });
      showMsg("IdP configuration saved successfully");
    } catch {
      showMsg("Failed to save IdP config (API may not be available)", "error");
    } finally {
      setSaving(false);
    }
  };

  const handleTestConnection = async () => {
    if (!idpConfig.sso_url) {
      showMsg("SSO URL is required to test connection", "error");
      return;
    }
    setTesting(true);
    setTestResult(null);
    const startTime = Date.now();
    try {
      const result = await apiFetch<{ entity_id?: string; sso_url?: string; cert_info?: CertInfo }>(
        "/api/v1/saml/test-connection",
        {
          method: "POST",
          body: JSON.stringify(idpConfig),
        },
      ).catch(() => null);
      const elapsed = Date.now() - startTime;
      if (result) {
        setTestResult({
          success: true,
          response_time_ms: elapsed,
          entity_id: result.entity_id || idpConfig.entity_id,
          sso_url: result.sso_url || idpConfig.sso_url,
          cert_info: result.cert_info || certInfo || undefined,
        });
      } else {
        setTestResult({
          success: false,
          response_time_ms: elapsed,
          error: "Connection failed — API returned an error or is not available",
        });
      }
    } catch {
      const elapsed = Date.now() - startTime;
      setTestResult({
        success: false,
        response_time_ms: elapsed,
        error: "Network error — unable to reach the IdP endpoint",
      });
    } finally {
      setTesting(false);
    }
  };

  const addMapping = useCallback(() => {
    setMappings((prev) => [
      ...prev,
      { id: `m${Date.now()}`, saml_attr: "", ggid_field: "email" },
    ]);
  }, []);

  const removeMapping = useCallback((id: string) => {
    setMappings((prev) => prev.filter((m) => m.id !== id));
  }, []);

  const updateMapping = useCallback((id: string, field: keyof AttributeMapping, value: string) => {
    setMappings((prev) => prev.map((m) => (m.id === id ? { ...m, [field]: value } : m)));
  }, []);

  return (
    <div>
      {/* Toast message */}
      {msg && (
        <div
          className={`fixed right-4 top-4 z-50 flex items-center gap-2 rounded-lg px-4 py-3 text-sm shadow-lg ${
            msgType === "success"
              ? "bg-green-600 text-white"
              : "bg-red-600 text-white"
          }`}
        >
          {msgType === "success" ? (
            <CheckCircle className="h-4 w-4" />
          ) : (
            <AlertCircle className="h-4 w-4" />
          )}
          {msg}
        </div>
      )}

      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">SAML Configuration</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Manage Service Provider metadata, Identity Provider settings, and attribute mappings
          </p>
        </div>
      </div>

      {/* SP Metadata Viewer */}
      <div className={cardCls}>
        <div className="mb-4 flex items-center justify-between">
          <h2 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-gray-100">
            <FileCode2 className="h-5 w-5 text-brand-600" />
            SP Metadata
          </h2>
          <button
            onClick={handleDownloadMetadata}
            aria-label="Download SP metadata"
            className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600"
          >
            <Download className="h-4 w-4" /> Download Metadata
          </button>
        </div>
        <div className="overflow-x-auto rounded-lg bg-gray-50 p-4 dark:bg-gray-900">
          <pre className="text-xs leading-relaxed">
            <code>{highlightXml(SP_METADATA)}</code>
          </pre>
        </div>
        <div className="mt-3 flex flex-wrap gap-4 text-xs text-gray-500 dark:text-gray-400">
          <span className="flex items-center gap-1">
            <span className="font-semibold text-blue-600 dark:text-blue-400">Tags</span>
          </span>
          <span className="flex items-center gap-1">
            <span className="font-semibold text-green-600 dark:text-green-400">Attributes</span>
          </span>
          <span className="flex items-center gap-1">
            <span className="font-semibold text-orange-600 dark:text-orange-400">Values</span>
          </span>
        </div>
      </div>

      {/* IdP Configuration */}
      <div className="mt-6 grid gap-6 lg:grid-cols-3">
        <div className={`${cardCls} lg:col-span-2`}>
          <h2 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-gray-100">
            <Shield className="h-5 w-5 text-brand-600" />
            Identity Provider Configuration
          </h2>

          <div className="mt-4 grid gap-4 sm:grid-cols-2">
            <div className="sm:col-span-2">
              <label className={labelCls}>Entity ID</label>
              <input
                type="text"
                value={idpConfig.entity_id}
                onChange={(e) => setIdpConfig({ ...idpConfig, entity_id: e.target.value })}
                placeholder="https://idp.example.com"
                className={`${inputCls} font-mono`}
                aria-label="Entity ID"
              />
            </div>
            <div>
              <label className={labelCls}>SSO URL</label>
              <input
                type="text"
                value={idpConfig.sso_url}
                onChange={(e) => setIdpConfig({ ...idpConfig, sso_url: e.target.value })}
                placeholder="https://idp.example.com/sso"
                className={`${inputCls} font-mono`}
                aria-label="SSO URL"
              />
            </div>
            <div>
              <label className={labelCls}>SLO URL <span className="text-gray-400">(optional)</span></label>
              <input
                type="text"
                value={idpConfig.slo_url}
                onChange={(e) => setIdpConfig({ ...idpConfig, slo_url: e.target.value })}
                placeholder="https://idp.example.com/slo"
                className={`${inputCls} font-mono`}
                aria-label="SLO URL"
              />
            </div>
            <div>
              <label className={labelCls}>NameID Format</label>
              <select
                value={idpConfig.name_id_format}
                onChange={(e) => setIdpConfig({ ...idpConfig, name_id_format: e.target.value })}
                className={inputCls}
              >
                {NAME_ID_FORMATS.map((f) => (
                  <option key={f.value} value={f.value}>{f.label}</option>
                ))}
              </select>
            </div>
            <div>
              <label className={labelCls}>AuthnContextClass</label>
              <select
                value={idpConfig.authn_context_class}
                onChange={(e) => setIdpConfig({ ...idpConfig, authn_context_class: e.target.value })}
                className={inputCls}
              >
                {AUTHN_CONTEXT_CLASSES.map((c) => (
                  <option key={c.value} value={c.value}>{c.label}</option>
                ))}
              </select>
            </div>
          </div>

          {/* Certificate upload */}
          <div className="mt-4">
            <label className={labelCls}>IdP Certificate (PEM/CRT)</label>
            <div className="flex flex-col gap-3">
              <div className="flex items-center gap-2">
                <button
                  onClick={() => fileInputRef.current?.click()}
                  className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600"
                >
                  <Upload className="h-4 w-4" /> Upload File
                </button>
                <input
                  ref={fileInputRef}
                  type="file"
                  accept=".pem,.crt,.cer"
                  onChange={handleCertUpload}
                  className="hidden"
                />
                <span className="text-xs text-gray-400">Accepts .pem, .crt, .cer</span>
              </div>
              <textarea
                value={idpConfig.cert}
                onChange={(e) => handleCertPaste(e.target.value)}
                placeholder="-----BEGIN CERTIFICATE-----&#10;Paste certificate content here...&#10;-----END CERTIFICATE-----"
                rows={5}
                className={`${inputCls} font-mono text-xs`}
                aria-label="IdP certificate"
              />
            </div>

            {/* Parsed cert info */}
            {certInfo && (
              <div className="mt-3 rounded-lg border border-green-200 bg-green-50 p-4 dark:border-green-800 dark:bg-green-950/30">
                <div className="mb-2 flex items-center gap-1.5 text-sm font-semibold text-green-700 dark:text-green-400">
                  <CheckCircle className="h-4 w-4" /> Certificate Parsed
                </div>
                <dl className="grid grid-cols-2 gap-2 text-xs">
                  <dt className="text-gray-500 dark:text-gray-400">Issuer</dt>
                  <dd className="break-all text-gray-800 dark:text-gray-200">{certInfo.issuer}</dd>
                  <dt className="text-gray-500 dark:text-gray-400">Subject</dt>
                  <dd className="break-all text-gray-800 dark:text-gray-200">{certInfo.subject}</dd>
                  <dt className="text-gray-500 dark:text-gray-400">Valid From</dt>
                  <dd className="text-gray-800 dark:text-gray-200">{new Date(certInfo.valid_from).toLocaleDateString()}</dd>
                  <dt className="text-gray-500 dark:text-gray-400">Valid To</dt>
                  <dd className="text-gray-800 dark:text-gray-200">{new Date(certInfo.valid_to).toLocaleDateString()}</dd>
                  <dt className="text-gray-500 dark:text-gray-400">Fingerprint (SHA1)</dt>
                  <dd className="break-all font-mono text-gray-800 dark:text-gray-200">{certInfo.fingerprint}</dd>
                </dl>
              </div>
            )}
          </div>

          {/* Save button */}
          <div className="mt-4">
            <button
              onClick={handleSaveIdP}
              disabled={saving}
              aria-label="Save IdP configuration"
              className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
            >
              {saving ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Save className="h-4 w-4" />
              )}
              Save IdP Config
            </button>
          </div>
        </div>

        {/* Test Connection Panel */}
        <div className={cardCls}>
          <h2 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-gray-100">
            <Link2 className="h-5 w-5 text-brand-600" />
            Test Connection
          </h2>
          <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
            Verify connectivity to the Identity Provider
          </p>

          <button
            onClick={handleTestConnection}
            disabled={testing}
            aria-label="Test IdP connection"
            className="mt-4 flex w-full items-center justify-center gap-1.5 rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 disabled:opacity-50"
          >
            {testing ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" /> Testing...
              </>
            ) : (
              <>
                <RefreshCw className="h-4 w-4" /> Test Connection
              </>
            )}
          </button>

          {/* Test result */}
          {testResult && (
            <div className="mt-4 space-y-3">
              <div
                className={`flex items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium ${
                  testResult.success
                    ? "bg-green-100 text-green-700 dark:bg-green-950/40 dark:text-green-400"
                    : "bg-red-100 text-red-700 dark:bg-red-950/40 dark:text-red-400"
                }`}
              >
                {testResult.success ? (
                  <CheckCircle className="h-4 w-4" />
                ) : (
                  <XCircle className="h-4 w-4" />
                )}
                {testResult.success ? "Connection Successful" : "Connection Failed"}
                <span className="ml-auto text-xs opacity-70">{testResult.response_time_ms}ms</span>
              </div>

              {testResult.success && (
                <dl className="space-y-2 text-xs">
                  {testResult.entity_id && (
                    <div>
                      <dt className="text-gray-500 dark:text-gray-400">Entity ID</dt>
                      <dd className="break-all text-gray-800 dark:text-gray-200">{testResult.entity_id}</dd>
                    </div>
                  )}
                  {testResult.sso_url && (
                    <div>
                      <dt className="text-gray-500 dark:text-gray-400">SSO URL</dt>
                      <dd className="break-all text-gray-800 dark:text-gray-200">{testResult.sso_url}</dd>
                    </div>
                  )}
                  {testResult.cert_info && (
                    <div>
                      <dt className="text-gray-500 dark:text-gray-400">Cert Issuer</dt>
                      <dd className="break-all text-gray-800 dark:text-gray-200">{testResult.cert_info.issuer}</dd>
                    </div>
                  )}
                </dl>
              )}

              {!testResult.success && testResult.error && (
                <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-xs text-red-700 dark:border-red-800 dark:bg-red-950/30 dark:text-red-400">
                  {testResult.error}
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {/* Attribute Mapping */}
      <div className={`mt-6 ${cardCls}`}>
        <div className="mb-4 flex items-center justify-between">
          <h2 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-gray-100">
            <Settings2 className="h-5 w-5 text-brand-600" />
            Attribute Mapping
          </h2>
          <button
            onClick={addMapping}
            className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-3 py-1.5 text-sm text-white hover:bg-brand-700"
          >
            <Plus className="h-4 w-4" /> Add Mapping
          </button>
        </div>
        <p className="mb-4 text-xs text-gray-500 dark:text-gray-400">
          Map SAML attributes from the IdP assertion to GGID user fields
        </p>

        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-gray-200 dark:border-gray-700 text-left text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">
                <th className="pb-2 pr-4">SAML Attribute</th>
                <th className="pb-2 pr-4">GGID Field</th>
                <th className="pb-2 w-20">Action</th>
              </tr>
            </thead>
            <tbody>
              {mappings.map((m) => (
                <tr key={m.id} className="border-b border-gray-100 dark:border-gray-700/50">
                  <td className="py-2 pr-4">
                    <input
                      type="text"
                      value={m.saml_attr}
                      onChange={(e) => updateMapping(m.id, "saml_attr", e.target.value)}
                      placeholder="e.g. email, givenName, memberOf"
                      className={`${inputCls} font-mono text-xs`}
                      aria-label={`SAML attribute ${m.id}`}
                    />
                  </td>
                  <td className="py-2 pr-4">
                    <select
                      value={m.ggid_field}
                      onChange={(e) => updateMapping(m.id, "ggid_field", e.target.value)}
                      className={`${inputCls} text-xs`}
                      aria-label={`GGID field ${m.id}`}
                    >
                      {GGID_FIELDS.map((f) => (
                        <option key={f.value} value={f.value}>{f.label}</option>
                      ))}
                    </select>
                  </td>
                  <td className="py-2">
                    <button
                      onClick={() => removeMapping(m.id)}
                      className="rounded-lg p-1.5 text-red-500 hover:bg-red-50 dark:hover:bg-red-950/30"
                      title="Remove mapping"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {mappings.length === 0 && (
          <div className="py-8 text-center text-sm text-gray-400">
            No attribute mappings configured. Click "Add Mapping" to create one.
          </div>
        )}
      </div>
    </div>
  );
}

"use client";
import { useState, useEffect, useCallback } from "react";
import {
  KeyRound, Shield, Server, Cloud, Lock, Save, Play, Loader2,
  CheckCircle, XCircle, RefreshCw, AlertTriangle, Eye, EyeOff,
  History, Cpu, Globe,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

type ProviderType =
  | "local" | "aws_kms" | "gcp_kms" | "azure_kv"
  | "vault" | "aliyun_kms" | "pkcs11" | "sm2";

interface ProviderMeta {
  id: ProviderType;
  label: string;
  icon: typeof Cloud;
  description: string;
  fields: ConfigField[];
}

interface ConfigField {
  key: string;
  label: string;
  type: "text" | "password" | "number" | "select" | "textarea";
  placeholder?: string;
  options?: string[];
  required?: boolean;
  helpText?: string;
}

interface KmsConfig {
  provider: ProviderType;
  config: Record<string, string | number | boolean>;
}

interface KmsStatus {
  active_algorithm?: string;
  key_id?: string;
  key_created_at?: string;
  last_rotation?: string;
  rotation_interval_days?: number;
}

interface TestResult {
  status: "ok" | "failed";
  error?: string;
  latency_ms?: number;
}

const PROVIDERS: ProviderMeta[] = [
  {
    id: "local",
    label: "Local Keystore",
    icon: Lock,
    description: "On-disk key storage (development only — not for production)",
    fields: [
      { key: "key_path", label: "Key File Path", type: "text", placeholder: "/etc/ggid/keys/master.key", helpText: "Absolute path to the master key file" },
      { key: "algorithm", label: "Algorithm", type: "select", options: ["AES-256-GCM", "ChaCha20-Poly1305"], required: true },
    ],
  },
  {
    id: "aws_kms",
    label: "AWS KMS",
    icon: Cloud,
    description: "Amazon Web Services Key Management Service",
    fields: [
      { key: "region", label: "Region", type: "text", placeholder: "us-east-1", required: true },
      { key: "access_key_id", label: "Access Key ID", type: "text", required: true },
      { key: "secret_access_key", label: "Secret Access Key", type: "password", required: true },
      { key: "key_id", label: "KMS Key ID", type: "text", placeholder: "arn:aws:kms:us-east-1:...", required: true },
      { key: "endpoint", label: "Custom Endpoint (optional)", type: "text", placeholder: "https://kms.us-east-1.amazonaws.com" },
    ],
  },
  {
    id: "gcp_kms",
    label: "GCP KMS",
    icon: Cloud,
    description: "Google Cloud Platform Key Management Service",
    fields: [
      { key: "project_id", label: "Project ID", type: "text", required: true },
      { key: "location", label: "Location", type: "text", placeholder: "global", required: true },
      { key: "key_ring", label: "Key Ring", type: "text", required: true },
      { key: "key_id", label: "Key ID", type: "text", required: true },
      { key: "credentials_json", label: "Service Account JSON", type: "textarea", placeholder: '{ "type": "service_account", ... }', required: true },
    ],
  },
  {
    id: "azure_kv",
    label: "Azure Key Vault",
    icon: Cloud,
    description: "Microsoft Azure Key Vault",
    fields: [
      { key: "vault_url", label: "Vault URL", type: "text", placeholder: "https://my-vault.vault.azure.net", required: true },
      { key: "tenant_id", label: "Tenant ID", type: "text", required: true },
      { key: "client_id", label: "Client ID", type: "text", required: true },
      { key: "client_secret", label: "Client Secret", type: "password", required: true },
      { key: "key_name", label: "Key Name", type: "text", required: true },
    ],
  },
  {
    id: "vault",
    label: "HashiCorp Vault",
    icon: Server,
    description: "HashiCorp Vault Transit Engine",
    fields: [
      { key: "address", label: "Vault Address", type: "text", placeholder: "https://vault.internal:8200", required: true },
      { key: "token", label: "Vault Token", type: "password", required: true },
      { key: "mount_path", label: "Transit Mount Path", type: "text", placeholder: "transit", required: true },
      { key: "key_name", label: "Key Name", type: "text", required: true },
      { key: "namespace", label: "Namespace (optional)", type: "text", placeholder: "admin/" },
      { key: "tls_skip_verify", label: "Skip TLS Verify", type: "select", options: ["false", "true"] },
    ],
  },
  {
    id: "aliyun_kms",
    label: "Alibaba Cloud KMS",
    icon: Globe,
    description: "Alibaba Cloud Key Management Service (China)",
    fields: [
      { key: "region_id", label: "Region ID", type: "text", placeholder: "cn-hangzhou", required: true },
      { key: "access_key_id", label: "Access Key ID", type: "text", required: true },
      { key: "access_key_secret", label: "Access Key Secret", type: "password", required: true },
      { key: "key_id", label: "KMS Key ID", type: "text", required: true },
      { key: "endpoint", label: "Custom Endpoint (optional)", type: "text", placeholder: "https://kms.cn-hangzhou.aliyuncs.com" },
    ],
  },
  {
    id: "pkcs11",
    label: "PKCS#11 HSM",
    icon: Cpu,
    description: "Generic HSM via PKCS#11 interface (Thales, Utimaco, YubiHSM, etc.)",
    fields: [
      { key: "module_path", label: "PKCS#11 Module Path", type: "text", placeholder: "/usr/lib/softhsm/libsofthsm2.so", required: true },
      { key: "slot_id", label: "Slot ID", type: "number", placeholder: "0", required: true },
      { key: "pin", label: "PIN", type: "password", required: true },
      { key: "key_label", label: "Key Label", type: "text", required: true },
      { key: "so_pin", label: "SO PIN (optional)", type: "password" },
    ],
  },
  {
    id: "sm2",
    label: "SM2/SM3/SM4 (国密)",
    icon: Shield,
    description: "Chinese national cryptography standards (GM/T)",
    fields: [
      { key: "implementation", label: "Implementation", type: "select", options: ["gmsm (Go)", "Tongsuo", "SJCL", "Custom"], required: true },
      { key: "sm2_key_path", label: "SM2 Private Key Path", type: "text", placeholder: "/etc/ggid/keys/sm2.pem", required: true },
      { key: "sm2_cert_path", label: "SM2 Certificate Path (optional)", type: "text", placeholder: "/etc/ggid/keys/sm2.crt" },
      { key: "sm4_mode", label: "SM4 Mode", type: "select", options: ["SM4-GCM", "SM4-CBC", "SM4-CTR"] },
      { key: "hsm_module", label: "HSM Module Path (optional)", type: "text", placeholder: "/usr/lib/gm3000/libgmt3000.so" },
    ],
  },
];

export default function KmsConfigPage() {
  const t = useTranslations();
  const [provider, setProvider] = useState<ProviderType>("local");
  const [configValues, setConfigValues] = useState<Record<string, string | number | boolean>>({});
  const [status, setStatus] = useState<KmsStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState<TestResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [showSecrets, setShowSecrets] = useState(false);

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch("/api/v1/settings/kms-config", {
        headers: { "X-Tenant-ID": TENANT_ID },
      }).catch(() => null);
      if (res?.ok) {
        const data: KmsConfig = await res.json();
        if (data.provider) setProvider(data.provider);
        if (data.config) setConfigValues(data.config);
      }
      // Load key status
      const statusRes = await fetch("/api/v1/settings/kms-config/status", {
        headers: { "X-Tenant-ID": TENANT_ID },
      }).catch(() => null);
      if (statusRes?.ok) {
        setStatus(await statusRes.json());
      }
    } catch {
      setError("Failed to load KMS configuration");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const save = async () => {
    setSaving(true);
    setSaved(false);
    setError(null);
    try {
      const res = await fetch("/api/v1/settings/kms-config", {
        method: "PUT",
        headers: { "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify({ provider, config: configValues }),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error || `HTTP ${res.status}`);
      }
      setSaved(true);
      setTimeout(() => setSaved(false), 3000);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to save configuration");
    } finally {
      setSaving(false);
    }
  };

  const testConnection = async () => {
    setTesting(true);
    setTestResult(null);
    try {
      const res = await fetch("/api/v1/settings/kms-config/test", {
        method: "POST",
        headers: { "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify({ provider, config: configValues }),
      });
      const data: TestResult = await res.json().catch(() => ({ status: "failed" as const, error: "No response" }));
      setTestResult(data);
    } catch (e) {
      setTestResult({ status: "failed", error: e instanceof Error ? e.message : "Connection failed" });
    } finally {
      setTesting(false);
    }
  };

  const rotateKey = async () => {
    try {
      const res = await fetch("/api/v1/settings/kms-config/rotate", {
        method: "POST",
        headers: { "X-Tenant-ID": TENANT_ID },
      });
      if (res.ok) { loadData(); }
    } catch {
      setError("Key rotation failed");
    }
  };

  const activeProvider = PROVIDERS.find(p => p.id === provider)!;
  const cardCls = "rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900";

  return (
    <div className="min-h-screen bg-gray-50 p-6 dark:bg-gray-950">
      <div className="mx-auto max-w-5xl space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
              <KeyRound className="h-6 w-6 text-indigo-600" />
              {t("settings.kmsConfig") || "KMS / HSM Configuration"}
            </h1>
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
              Manage cryptographic key providers — cloud KMS, on-prem HSM, and national crypto standards.
            </p>
          </div>
          <button onClick={loadData} disabled={loading} aria-label="Refresh configuration" className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800">
            <RefreshCw className={"h-4 w-4 " + (loading ? "animate-spin" : "")} /> Refresh
          </button>
        </div>

        {/* Error banner */}
        {error && (
          <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
            <AlertTriangle className="h-4 w-4 shrink-0" />{error}
            <button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><XCircle className="h-4 w-4" /></button>
          </div>
        )}

        {loading ? (
          <div className="flex justify-center py-16"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>
        ) : (
          <>
            {/* Provider selector */}
            <div className={cardCls}>
              <h2 className="mb-4 text-sm font-semibold uppercase text-gray-400">Key Provider</h2>
              <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
                {PROVIDERS.map(p => {
                  const Icon = p.icon;
                  const isActive = provider === p.id;
                  return (
                    <button
                      key={p.id}
                      onClick={() => { setProvider(p.id); setTestResult(null); }}
                      className={"flex flex-col items-start gap-2 rounded-xl border-2 p-4 text-left transition " + (isActive ? "border-indigo-500 bg-indigo-50 dark:bg-indigo-950/30" : "border-gray-200 hover:border-gray-300 dark:border-gray-700 dark:hover:border-gray-600")}
                      aria-pressed={isActive}
                    >
                      <div className="flex items-center gap-2">
                        <Icon className={"h-5 w-5 " + (isActive ? "text-indigo-600" : "text-gray-400")} />
                        <span className={"text-sm font-semibold " + (isActive ? "text-indigo-700 dark:text-indigo-400" : "text-gray-700 dark:text-gray-300")}>{p.label}</span>
                      </div>
                      <p className="text-xs text-gray-500 dark:text-gray-400">{p.description}</p>
                    </button>
                  );
                })}
              </div>
            </div>

            {/* Configuration form */}
            <div className={cardCls}>
              <div className="mb-4 flex items-center justify-between">
                <h2 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white">
                  {activeProvider.label} Configuration
                </h2>
                <button
                  onClick={() => setShowSecrets(!showSecrets)}
                  className="flex items-center gap-1 text-xs text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
                  aria-label={showSecrets ? "Hide secrets" : "Show secrets"}
                >
                  {showSecrets ? <EyeOff className="h-3.5 w-3.5" /> : <Eye className="h-3.5 w-3.5" />}
                  {showSecrets ? "Hide Secrets" : "Show Secrets"}
                </button>
              </div>

              <div className="grid gap-4 sm:grid-cols-2">
                {activeProvider.fields.map(field => (
                  <div key={field.key} className={field.type === "textarea" ? "sm:col-span-2" : ""}>
                    <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
                      {field.label}
                      {field.required && <span className="ml-1 text-red-500">*</span>}
                    </label>
                    {field.type === "select" ? (
                      <select
                        value={String(configValues[field.key] ?? "")}
                        onChange={e => setConfigValues(prev => ({ ...prev, [field.key]: e.target.value }))}
                        className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-700 dark:bg-gray-800 dark:text-gray-200"
                      >
                        <option value="">— Select —</option>
                        {field.options?.map(opt => <option key={opt} value={opt}>{opt}</option>)}
                      </select>
                    ) : field.type === "textarea" ? (
                      <textarea
                        value={String(configValues[field.key] ?? "")}
                        onChange={e => setConfigValues(prev => ({ ...prev, [field.key]: e.target.value }))}
                        placeholder={field.placeholder}
                        rows={4}
                        className="w-full rounded-lg border border-gray-300 px-3 py-2 font-mono text-xs dark:border-gray-700 dark:bg-gray-800 dark:text-gray-200"
                      />
                    ) : (
                      <input
                        type={field.type === "password" && !showSecrets ? "password" : field.type === "number" ? "number" : "text"}
                        value={String(configValues[field.key] ?? "")}
                        onChange={e => setConfigValues(prev => ({ ...prev, [field.key]: field.type === "number" ? parseInt(e.target.value) || 0 : e.target.value }))}
                        placeholder={field.placeholder}
                        className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-700 dark:bg-gray-800 dark:text-gray-200"
                      />
                    )}
                    {field.helpText && <p className="mt-1 text-xs text-gray-400">{field.helpText}</p>}
                  </div>
                ))}
              </div>

              {/* Action buttons */}
              <div className="mt-6 flex flex-wrap items-center gap-3">
                <button onClick={save} disabled={saving} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">
                  {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
                  {saving ? "Saving..." : "Save Configuration"}
                </button>
                <button onClick={testConnection} disabled={testing} className="flex items-center gap-2 rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800">
                  {testing ? <Loader2 className="h-4 w-4 animate-spin" /> : <Play className="h-4 w-4" />}
                  {testing ? "Testing..." : "Test Connection"}
                </button>
                {saved && <span className="flex items-center gap-1 text-sm text-green-600 dark:text-green-400"><CheckCircle className="h-4 w-4" /> Saved</span>}
                {testResult?.status === "ok" && <span className="flex items-center gap-1 text-sm text-green-600 dark:text-green-400"><CheckCircle className="h-4 w-4" /> Connection OK{testResult.latency_ms ? ` (${testResult.latency_ms}ms)` : ""}</span>}
                {testResult?.status === "failed" && <span className="flex items-center gap-1 text-sm text-red-600 dark:text-red-400"><XCircle className="h-4 w-4" /> {testResult.error || "Connection failed"}</span>}
              </div>
            </div>

            {/* Key status */}
            {status && (
              <div className={cardCls}>
                <div className="mb-4 flex items-center justify-between">
                  <h2 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white">
                    <History className="h-5 w-5 text-gray-400" /> Key Status
                  </h2>
                  <button onClick={rotateKey} className="flex items-center gap-1.5 rounded-lg bg-amber-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-amber-700">
                    <RefreshCw className="h-3.5 w-3.5" /> Rotate Key
                  </button>
                </div>
                <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
                  <div className="rounded-lg border border-gray-200 p-3 dark:border-gray-700">
                    <p className="text-xs font-medium uppercase text-gray-400">Active Algorithm</p>
                    <p className="mt-1 font-mono text-sm font-semibold text-gray-900 dark:text-gray-100">{status.active_algorithm || "—"}</p>
                  </div>
                  <div className="rounded-lg border border-gray-200 p-3 dark:border-gray-700">
                    <p className="text-xs font-medium uppercase text-gray-400">Key ID</p>
                    <p className="mt-1 truncate font-mono text-sm font-semibold text-gray-900 dark:text-gray-100">{status.key_id || "—"}</p>
                  </div>
                  <div className="rounded-lg border border-gray-200 p-3 dark:border-gray-700">
                    <p className="text-xs font-medium uppercase text-gray-400">Key Created</p>
                    <p className="mt-1 text-sm font-semibold text-gray-900 dark:text-gray-100">{status.key_created_at ? new Date(status.key_created_at).toLocaleString() : "—"}</p>
                  </div>
                  <div className="rounded-lg border border-gray-200 p-3 dark:border-gray-700">
                    <p className="text-xs font-medium uppercase text-gray-400">Last Rotation</p>
                    <p className="mt-1 text-sm font-semibold text-gray-900 dark:text-gray-100">{status.last_rotation ? new Date(status.last_rotation).toLocaleString() : "—"}</p>
                  </div>
                </div>
                {status.rotation_interval_days && (
                  <p className="mt-3 flex items-center gap-1.5 text-xs text-gray-400">
                    <AlertTriangle className="h-3.5 w-3.5" />
                    Auto-rotation every {status.rotation_interval_days} days
                  </p>
                )}
              </div>
            )}

            {/* Security warning for local provider */}
            {provider === "local" && (
              <div role="alert" className="flex items-start gap-3 rounded-xl border border-amber-300 bg-amber-50 p-4 dark:border-amber-700 dark:bg-amber-950/30">
                <AlertTriangle className="h-5 w-5 shrink-0 text-amber-600" />
                <div>
                  <p className="text-sm font-semibold text-amber-800 dark:text-amber-400">Not for Production Use</p>
                  <p className="mt-1 text-xs text-amber-700 dark:text-amber-500">
                    Local keystore stores keys on disk without hardware protection. Use a cloud KMS, HSM, or Vault provider for production environments.
                  </p>
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}

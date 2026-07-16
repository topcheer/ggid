"use client";

import { useState, useEffect, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  Shield, Plus, Trash2, Upload, X, AlertCircle, Loader2, Check,
  Link2, FileText, Zap, RefreshCw,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

type ProviderType = "saml" | "oidc" | "ldap";

interface SSOProvider {
  id: string;
  name: string;
  type: ProviderType;
  enabled: boolean;
  config: Record<string, string>;
  created_at: string;
  last_tested?: string;
  test_status?: "success" | "failed" | "pending";
  attribute_mapping?: Record<string, string>;
  jit_provisioning?: boolean;
}

const PROVIDER_TYPES: { value: ProviderType; label: string; icon: typeof Shield }[] = [
  { value: "saml", label: "SAML 2.0", icon: FileText },
  { value: "oidc", label: "OpenID Connect", icon: Link2 },
  { value: "ldap", label: "LDAP", icon: Shield },
];

export default function SSOPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [providers, setProviders] = useState<SSOProvider[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showAdd, setShowAdd] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState<SSOProvider | null>(null);
  const [testing, setTesting] = useState<string | null>(null);
  const [uploadResult, setUploadResult] = useState<{ name: string; entities?: number } | null>(null);

  // Form state
  const [form, setForm] = useState({
    name: "",
    type: "saml" as ProviderType,
    entity_id: "",
    metadata_url: "",
    sso_url: "",
    certificate: "",
    jit_provisioning: true,
  });
  const [creating, setCreating] = useState(false);
  const [uploading, setUploading] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiFetch<{ providers?: SSOProvider[]; items?: SSOProvider[] }>("/api/v1/settings/sso/providers").catch(() => null);
      setProviders(data?.providers ?? data?.items ?? []);
    } catch {
      setError("Failed to load SSO providers");
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => { load(); }, [load]);

  const handleCreate = async () => {
    if (!form.name.trim()) return;
    setCreating(true);
    try {
      const body: Record<string, unknown> = {
        name: form.name,
        type: form.type,
        jit_provisioning: form.jit_provisioning,
      };
      if (form.type === "saml") {
        body.entity_id = form.entity_id;
        body.sso_url = form.sso_url;
        body.metadata_url = form.metadata_url;
        body.certificate = form.certificate;
      } else if (form.type === "oidc") {
        body.issuer_url = form.metadata_url;
        body.client_id = form.entity_id;
      } else if (form.type === "ldap") {
        body.server_url = form.sso_url;
        body.base_dn = form.entity_id;
      }
      await apiFetch("/api/v1/settings/sso/providers", { method: "POST", body: JSON.stringify(body) });
      setForm({ name: "", type: "saml", entity_id: "", metadata_url: "", sso_url: "", certificate: "", jit_provisioning: true });
      setShowAdd(false);
      await load();
    } catch {
      setError("Failed to create SSO provider");
    } finally {
      setCreating(false);
    }
  };

  const handleTest = async (p: SSOProvider) => {
    setTesting(p.id);
    try {
      await apiFetch(`/api/v1/settings/sso/providers/${p.id}/test`, { method: "POST" });
      setProviders((prev) => prev.map((x) => x.id === p.id ? { ...x, test_status: "success", last_tested: new Date().toISOString() } : x));
    } catch {
      setProviders((prev) => prev.map((x) => x.id === p.id ? { ...x, test_status: "failed", last_tested: new Date().toISOString() } : x));
    } finally {
      setTesting(null);
    }
  };

  const handleToggle = async (p: SSOProvider) => {
    try {
      await apiFetch(`/api/v1/settings/sso/providers/${p.id}`, {
        method: "PATCH", body: JSON.stringify({ enabled: !p.enabled }),
      });
      setProviders((prev) => prev.map((x) => x.id === p.id ? { ...x, enabled: !x.enabled } : x));
    } catch {
      setError("Failed to toggle provider");
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await apiFetch(`/api/v1/settings/sso/providers/${id}`, { method: "DELETE" });
      setConfirmDelete(null);
      await load();
    } catch {
      setError("Failed to delete provider");
    }
  };

  const handleMetadataUpload = async (file: File) => {
    setUploading(true);
    try {
      const text = await file.text();
      const resp = await apiFetch<{ entities?: number; entity_id?: string }>("/api/v1/settings/sso/metadata/import", {
        method: "POST", body: JSON.stringify({ metadata_xml: text }),
      });
      setUploadResult({ name: file.name, entities: resp.entities ?? 1 });
      await load();
    } catch {
      setError("Failed to parse SAML metadata");
    } finally {
      setUploading(false);
    }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <Shield className="h-6 w-6 text-indigo-600" /> SSO Providers
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Configure external identity providers for single sign-on.
          </p>
        </div>
        <div className="flex gap-2">
          <label className="flex cursor-pointer items-center gap-2 rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700">
            <Upload className="h-4 w-4" /> Upload Metadata
            <input aria-label="Input field" type="file" accept=".xml" className="hidden" onChange={(e) => { const f = e.target.files?.[0]; if (f) handleMetadataUpload(f); }} />
          </label>
          <button onClick={() => setShowAdd(true)} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700">
            <Plus className="h-4 w-4" /> Add Provider
          </button>
        </div>
      </div>

      {uploading && (
        <div className="flex items-center gap-2 rounded-lg bg-blue-50 px-4 py-3 text-sm text-blue-700 dark:bg-blue-900/20 dark:text-blue-400">
          <Loader2 className="h-4 w-4 animate-spin" /> Parsing SAML metadata...
        </div>
      )}
      {uploadResult && (
        <div role="status" className="flex items-center gap-2 rounded-lg bg-green-50 px-4 py-3 text-sm text-green-700 dark:bg-green-900/20 dark:text-green-400">
          <Check className="h-4 w-4" /> Imported <strong>{uploadResult.name}</strong> — {uploadResult.entities} entit{uploadResult.entities === 1 ? "y" : "ies"} found.
          <button onClick={() => setUploadResult(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}
      {error && (
        <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}
          <button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      {loading ? (
        <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>
      ) : providers.length === 0 ? (
        <div className={cardCls}>
          <div className="py-12 text-center">
            <Shield className="mx-auto h-12 w-12 text-gray-300" />
            <p className="mt-4 text-sm text-gray-400">No SSO providers configured. Add one or upload SAML metadata.</p>
          </div>
        </div>
      ) : (
        <div className="grid gap-4 md:grid-cols-2">
          {providers.map((p) => {
            const typeInfo = PROVIDER_TYPES.find((t) => t.value === p.type) ?? PROVIDER_TYPES[0];
            const TypeIcon = typeInfo.icon;
            return (
              <div key={p.id} className={cardCls}>
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-3">
                    <div className={`rounded-lg p-2 ${p.enabled ? "bg-indigo-100 dark:bg-indigo-900/30" : "bg-gray-100 dark:bg-gray-700"}`}>
                      <TypeIcon className={`h-5 w-5 ${p.enabled ? "text-indigo-600" : "text-gray-400"}`} />
                    </div>
                    <div>
                      <h3 className="font-semibold text-gray-800 dark:text-gray-200">{p.name}</h3>
                      <p className="text-xs text-gray-400">{typeInfo.label}</p>
                    </div>
                  </div>
                  <label className="relative inline-flex cursor-pointer items-center">
                    <input aria-label="P" type="checkbox" checked={p.enabled} onChange={() => handleToggle(p)} className="peer sr-only" />
                    <div className="h-5 w-9 rounded-full bg-gray-200 after:absolute after:left-[2px] after:top-[2px] after:h-4 after:w-4 after:rounded-full after:border after:transition-all peer-checked:bg-indigo-600 peer-checked:after:translate-x-full dark:bg-gray-700" />
                  </label>
                </div>

                {/* Config details */}
                <div className="mt-4 space-y-1 text-xs">
                  {p.config.entity_id && <div className="flex justify-between"><span className="text-gray-400">Entity ID:</span><span className="font-mono text-gray-600 dark:text-gray-300 truncate">{p.config.entity_id}</span></div>}
                  {p.config.sso_url && <div className="flex justify-between"><span className="text-gray-400">SSO URL:</span><span className="font-mono text-gray-600 dark:text-gray-300 truncate max-w-[200px]">{p.config.sso_url}</span></div>}
                  {p.jit_provisioning !== undefined && (
                    <div className="flex justify-between"><span className="text-gray-400">JIT:</span><span className={p.jit_provisioning ? "text-green-600" : "text-gray-400"}>{p.jit_provisioning ? "Enabled" : "Disabled"}</span></div>
                  )}
                  {p.last_tested && <div className="flex justify-between"><span className="text-gray-400">Last tested:</span><span className="text-gray-500">{new Date(p.last_tested).toLocaleString()}</span></div>}
                </div>

                {/* Test status badge */}
                {p.test_status && (
                  <div className="mt-3">
                    <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${
                      p.test_status === "success" ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400"
                      : p.test_status === "failed" ? "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400"
                      : "bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400"
                    }`}>
                      {p.test_status === "success" && <Check className="h-3 w-3" />}
                      {p.test_status === "failed" && <AlertCircle className="h-3 w-3" />}
                      Test: {p.test_status}
                    </span>
                  </div>
                )}

                {/* Actions */}
                <div className="mt-4 flex items-center gap-2">
                  <button
                    onClick={() => handleTest(p)}
                    disabled={testing === p.id}
                    className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-1.5 text-xs font-medium text-gray-600 hover:bg-gray-50 disabled:opacity-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
                  >
                    {testing === p.id ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Zap className="h-3.5 w-3.5" />}
                    Test Connection
                  </button>
                  <button onClick={() => setConfirmDelete(p)} className="rounded-lg border border-red-200 px-3 py-1.5 text-xs font-medium text-red-500 hover:bg-red-50 dark:border-red-800 dark:hover:bg-red-900/20">
                    <Trash2 className="h-3.5 w-3.5" />
                  </button>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Add provider modal */}
      {showAdd && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowAdd(false)}>
          <div role="dialog" aria-modal="true" className="max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Add SSO Provider</h2>
              <button onClick={() => setShowAdd(false)} aria-label="Close"><X className="h-5 w-5 text-gray-400" /></button>
            </div>
            <div className="mt-4 space-y-4">
              {/* Provider type selector */}
              <div>
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300">Provider Type</label>
                <div className="mt-2 grid grid-cols-3 gap-2">
                  {PROVIDER_TYPES.map((t) => {
                    const Icon = t.icon;
                    return (
                      <button key={t.value} onClick={() => setForm((p) => ({ ...p, type: t.value }))}
                        className={`flex flex-col items-center gap-1 rounded-lg border p-3 ${form.type === t.value ? "border-indigo-500 bg-indigo-50 dark:bg-indigo-900/30" : "border-gray-300 dark:border-gray-600"}`}>
                        <Icon className={`h-5 w-5 ${form.type === t.value ? "text-indigo-600" : "text-gray-400"}`} />
                        <span className="text-xs font-medium">{t.label}</span>
                      </button>
                    );
                  })}
                </div>
              </div>

              <div>
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300">Display Name</label>
                <input aria-label="e.g. Corporate Okta" value={form.name} onChange={(e) => setForm((p) => ({ ...p, name: e.target.value }))} placeholder="e.g. Corporate Okta" className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" />
              </div>

              {form.type === "saml" && (
                <>
                  <div>
                    <label className="text-sm font-medium text-gray-700 dark:text-gray-300">Entity ID</label>
                    <input aria-label="https://idp.example.com/entity" value={form.entity_id} onChange={(e) => setForm((p) => ({ ...p, entity_id: e.target.value }))} placeholder="https://idp.example.com/entity" className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" />
                  </div>
                  <div>
                    <label className="text-sm font-medium text-gray-700 dark:text-gray-300">SSO URL</label>
                    <input aria-label="https://idp.example.com/sso" value={form.sso_url} onChange={(e) => setForm((p) => ({ ...p, sso_url: e.target.value }))} placeholder="https://idp.example.com/sso" className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" />
                  </div>
                  <div>
                    <label className="text-sm font-medium text-gray-700 dark:text-gray-300">Metadata URL (optional)</label>
                    <input aria-label="https://idp.example.com/metadata" value={form.metadata_url} onChange={(e) => setForm((p) => ({ ...p, metadata_url: e.target.value }))} placeholder="https://idp.example.com/metadata" className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" />
                  </div>
                  <div>
                    <label className="text-sm font-medium text-gray-700 dark:text-gray-300">X.509 Certificate</label>
                    <textarea aria-label="-----BEGIN CERTIFICATE-----" value={form.certificate} onChange={(e) => setForm((p) => ({ ...p, certificate: e.target.value }))} placeholder="-----BEGIN CERTIFICATE-----" rows={3} className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 font-mono text-xs dark:border-gray-600 dark:bg-gray-700 dark:text-white" />
                  </div>
                </>
              )}
              {form.type === "oidc" && (
                <>
                  <div>
                    <label className="text-sm font-medium text-gray-700 dark:text-gray-300">Issuer URL</label>
                    <input aria-label="https://accounts.google.com" value={form.metadata_url} onChange={(e) => setForm((p) => ({ ...p, metadata_url: e.target.value }))} placeholder="https://accounts.google.com" className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" />
                  </div>
                  <div>
                    <label className="text-sm font-medium text-gray-700 dark:text-gray-300">Client ID</label>
                    <input aria-label="your-client-id" value={form.entity_id} onChange={(e) => setForm((p) => ({ ...p, entity_id: e.target.value }))} placeholder="your-client-id" className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" />
                  </div>
                </>
              )}
              {form.type === "ldap" && (
                <>
                  <div>
                    <label className="text-sm font-medium text-gray-700 dark:text-gray-300">Server URL</label>
                    <input aria-label="ldap://dc.example.com:389" value={form.sso_url} onChange={(e) => setForm((p) => ({ ...p, sso_url: e.target.value }))} placeholder="ldap://dc.example.com:389" className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" />
                  </div>
                  <div>
                    <label className="text-sm font-medium text-gray-700 dark:text-gray-300">Base DN</label>
                    <input aria-label="dc=example,dc=com" value={form.entity_id} onChange={(e) => setForm((p) => ({ ...p, entity_id: e.target.value }))} placeholder="dc=example,dc=com" className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 font-mono text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" />
                  </div>
                </>
              )}

              <label className="flex cursor-pointer items-center gap-2">
                <input aria-label="Form" type="checkbox" checked={form.jit_provisioning} onChange={(e) => setForm((p) => ({ ...p, jit_provisioning: e.target.checked }))} className="rounded border-gray-300 text-indigo-600" />
                <span className="text-sm text-gray-600 dark:text-gray-300">Enable JIT provisioning (auto-create users on first login)</span>
              </label>
            </div>
            <div className="mt-6 flex justify-end gap-2">
              <button onClick={() => setShowAdd(false)} className="rounded-lg px-4 py-2 text-sm text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700">Cancel</button>
              <button onClick={handleCreate} disabled={!form.name.trim() || creating} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">
                {creating && <Loader2 className="h-4 w-4 animate-spin" />} Add Provider
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Delete confirmation */}
      {confirmDelete && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setConfirmDelete(null)}>
          <div role="dialog" aria-modal="true" className="w-full max-w-sm rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center gap-3">
              <div className="rounded-full bg-red-100 p-2 dark:bg-red-900/30"><Trash2 className="h-5 w-5 text-red-600" /></div>
              <div>
                <h2 className="font-semibold text-gray-900 dark:text-white">Remove SSO Provider?</h2>
                <p className="text-sm text-gray-500"><strong>{confirmDelete.name}</strong> will be removed. Users from this IdP will need alternate login.</p>
              </div>
            </div>
            <div className="mt-5 flex justify-end gap-2">
              <button onClick={() => setConfirmDelete(null)} className="rounded-lg px-4 py-2 text-sm text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700">Cancel</button>
              <button onClick={() => handleDelete(confirmDelete.id)} className="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700">Remove</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

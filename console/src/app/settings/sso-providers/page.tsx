"use client";

import { useState, useEffect, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  Globe,
  Plus,
  Trash2,
  Save,
  Loader2,
  Shield,
  CheckCircle2,
  XCircle,
  Zap,
  ChevronDown,
  ChevronUp,
} from "lucide-react";

interface SSOProvider {
  id: string;
  name: string;
  type: "saml" | "oidc" | "ldap";
  enabled: boolean;
  domain: string;
  jit_provisioning: boolean;
  attribute_mapping: Record<string, string>;
  // SAML
  entityId?: string;
  ssoUrl?: string;
  certFingerprint?: string;
  // OIDC
  issuerUrl?: string;
  clientId?: string;
  clientSecret?: string;
  scopes?: string;
  // LDAP
  ldapUrl?: string;
  bindDn?: string;
  baseDn?: string;
}

const ATTR_KEYS = ["username", "email", "firstName", "lastName", "displayName", "groups", "department"];

export default function SSOProvidersPage() {
  const { apiFetch } = useApi();
  const [providers, setProviders] = useState<SSOProvider[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [msg, setMsg] = useState("");
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [testing, setTesting] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const data = await apiFetch<{ providers?: SSOProvider[] }>("/api/v1/settings/idp").catch(() => ({ providers: [] }));
      setProviders(data.providers ?? []);
    } catch { /* ignore */ } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => { load(); }, [load]);

  const persist = async (updated: SSOProvider[]) => {
    setProviders(updated);
    setSaving(true);
    try {
      await apiFetch("/api/v1/settings/idp", { method: "PUT", body: JSON.stringify({ providers: updated }) });
      setMsg("Saved");
    } catch {
      localStorage.setItem("ggid_sso_providers", JSON.stringify(updated));
      setMsg("Saved (offline)");
    } finally {
      setSaving(false);
      setTimeout(() => setMsg(""), 3000);
    }
  };

  const handleAdd = () => {
    const p: SSOProvider = {
      id: `sso-${Date.now()}`, name: "New Provider", type: "saml",
      enabled: false, domain: "", jit_provisioning: true, attribute_mapping: {},
    };
    persist([...providers, p]);
    setExpandedId(p.id);
  };

  const handleDelete = (id: string) => persist(providers.filter((p) => p.id !== id));
  const handleToggle = (id: string) => persist(providers.map((p) => (p.id === id ? { ...p, enabled: !p.enabled } : p)));
  const handleUpdate = (id: string, field: keyof SSOProvider, value: unknown) =>
    setProviders(providers.map((p) => (p.id === id ? { ...p, [field]: value } : p)));

  const handleTest = async (id: string) => {
    setTesting(id);
    try {
      const data = await apiFetch<{ success?: boolean; message?: string }>(`/api/v1/settings/idp/${id}/test`, { method: "POST" });
      setMsg(data.message ?? "Connection test completed");
    } catch {
      setMsg("Test failed — endpoint unavailable");
    } finally {
      setTesting(null);
      setTimeout(() => setMsg(""), 4000);
    }
  };

  const inputCls = "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";
  const cardCls = "rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const typeIcon = (t: string) => t === "saml" ? <Shield className="h-5 w-5 text-blue-500" /> : t === "oidc" ? <Globe className="h-5 w-5 text-green-500" /> : <Globe className="h-5 w-5 text-purple-500" />;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <Globe className="h-7 w-7 text-indigo-600" /> SSO Providers
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            External identity providers with attribute mapping and JIT provisioning.
          </p>
        </div>
        <div className="flex items-center gap-2">
          {msg && <span className="text-sm text-green-600">{msg}</span>}
          <button onClick={handleAdd} className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700">
            <Plus className="mr-1 inline h-4 w-4" /> Add Provider
          </button>
        </div>
      </div>

      {loading ? (
        <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>
      ) : providers.length === 0 ? (
        <div className={`${cardCls} text-center`}>
          <Globe className="mx-auto mb-3 h-12 w-12 text-gray-300" />
          <p className="text-gray-500 dark:text-gray-400">No SSO providers configured.</p>
        </div>
      ) : (
        <div className="space-y-4">
          {providers.map((p) => (
            <div key={p.id} className={cardCls}>
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <button onClick={() => handleToggle(p.id)} className={`flex h-6 w-11 items-center rounded-full transition-colors ${p.enabled ? "bg-indigo-600" : "bg-gray-300 dark:bg-gray-600"}`}>
                    <span className={`h-5 w-5 transform rounded-full bg-white shadow transition-transform ${p.enabled ? "translate-x-5" : "translate-x-0.5"}`} />
                  </button>
                  {typeIcon(p.type)}
                  <div>
                    <input className="border-none bg-transparent text-base font-semibold text-gray-900 outline-none dark:text-white" value={p.name} onChange={(e) => handleUpdate(p.id, "name", e.target.value)} />
                    <div className="mt-0.5 flex items-center gap-2">
                      <select className="rounded bg-gray-100 px-1.5 py-0.5 text-xs text-gray-500 dark:bg-gray-700" value={p.type} onChange={(e) => handleUpdate(p.id, "type", e.target.value)}>
                        <option value="saml">SAML</option><option value="oidc">OIDC</option><option value="ldap">LDAP</option>
                      </select>
                      <input className="rounded bg-gray-100 px-2 py-0.5 text-xs text-gray-500 dark:bg-gray-700" placeholder="domain.com" value={p.domain} onChange={(e) => handleUpdate(p.id, "domain", e.target.value)} />
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <button onClick={() => handleTest(p.id)} disabled={testing === p.id} className="rounded-lg border border-gray-300 px-3 py-1.5 text-xs font-medium text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700">
                    {testing === p.id ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Zap className="mr-1 inline h-3.5 w-3.5" />} Test
                  </button>
                  <button onClick={() => setExpandedId(expandedId === p.id ? null : p.id)} className="rounded-lg p-2 text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700">
                    {expandedId === p.id ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
                  </button>
                  <button onClick={() => handleDelete(p.id)} className="rounded-lg p-2 text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20">
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
              </div>

              {expandedId === p.id && (
                <div className="mt-4 space-y-4 border-t border-gray-200 pt-4 dark:border-gray-700">
                  {/* JIT */}
                  <div className="flex items-center gap-2">
                    <input type="checkbox" id={`jit-${p.id}`} checked={p.jit_provisioning} onChange={(e) => handleUpdate(p.id, "jit_provisioning", e.target.checked)} className="h-4 w-4 rounded border-gray-300 text-indigo-600" />
                    <label htmlFor={`jit-${p.id}`} className="text-sm text-gray-700 dark:text-gray-300">JIT provisioning (auto-create users on first login)</label>
                  </div>

                  {/* Type-specific fields */}
                  {p.type === "saml" && (
                    <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
                      <div><label className="mb-1 block text-xs font-medium text-gray-500">Entity ID</label><input className={inputCls} value={p.entityId ?? ""} onChange={(e) => handleUpdate(p.id, "entityId", e.target.value)} /></div>
                      <div><label className="mb-1 block text-xs font-medium text-gray-500">SSO URL</label><input className={inputCls} value={p.ssoUrl ?? ""} onChange={(e) => handleUpdate(p.id, "ssoUrl", e.target.value)} /></div>
                      <div><label className="mb-1 block text-xs font-medium text-gray-500">Cert Fingerprint</label><input className={inputCls} value={p.certFingerprint ?? ""} onChange={(e) => handleUpdate(p.id, "certFingerprint", e.target.value)} /></div>
                    </div>
                  )}
                  {p.type === "oidc" && (
                    <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
                      <div><label className="mb-1 block text-xs font-medium text-gray-500">Issuer URL</label><input className={inputCls} value={p.issuerUrl ?? ""} onChange={(e) => handleUpdate(p.id, "issuerUrl", e.target.value)} /></div>
                      <div><label className="mb-1 block text-xs font-medium text-gray-500">Client ID</label><input className={inputCls} value={p.clientId ?? ""} onChange={(e) => handleUpdate(p.id, "clientId", e.target.value)} /></div>
                      <div><label className="mb-1 block text-xs font-medium text-gray-500">Client Secret</label><input className={inputCls} type="password" value={p.clientSecret ?? ""} onChange={(e) => handleUpdate(p.id, "clientSecret", e.target.value)} /></div>
                      <div><label className="mb-1 block text-xs font-medium text-gray-500">Scopes</label><input className={inputCls} value={p.scopes ?? ""} onChange={(e) => handleUpdate(p.id, "scopes", e.target.value)} /></div>
                    </div>
                  )}
                  {p.type === "ldap" && (
                    <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
                      <div><label className="mb-1 block text-xs font-medium text-gray-500">LDAP URL</label><input className={inputCls} value={p.ldapUrl ?? ""} onChange={(e) => handleUpdate(p.id, "ldapUrl", e.target.value)} /></div>
                      <div><label className="mb-1 block text-xs font-medium text-gray-500">Bind DN</label><input className={inputCls} value={p.bindDn ?? ""} onChange={(e) => handleUpdate(p.id, "bindDn", e.target.value)} /></div>
                      <div><label className="mb-1 block text-xs font-medium text-gray-500">Base DN</label><input className={inputCls} value={p.baseDn ?? ""} onChange={(e) => handleUpdate(p.id, "baseDn", e.target.value)} /></div>
                    </div>
                  )}

                  {/* Attribute Mapping */}
                  <div>
                    <h4 className="mb-2 text-xs font-semibold uppercase text-gray-400">Attribute Mapping</h4>
                    <div className="grid grid-cols-1 gap-2 md:grid-cols-2">
                      {ATTR_KEYS.map((key) => (
                        <div key={key} className="flex items-center gap-2">
                          <span className="w-24 text-xs font-medium text-gray-500">{key}</span>
                          <input
                            className={`${inputCls} text-xs`}
                            placeholder={`IdP attribute for ${key}`}
                            value={p.attribute_mapping?.[key] ?? ""}
                            onChange={(e) => handleUpdate(p.id, "attribute_mapping", { ...p.attribute_mapping, [key]: e.target.value })}
                          />
                        </div>
                      ))}
                    </div>
                  </div>

                  <button onClick={() => persist(providers)} disabled={saving} className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">
                    {saving ? <Loader2 className="mr-1 inline h-4 w-4 animate-spin" /> : <Save className="mr-1 inline h-4 w-4" />} Save
                  </button>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

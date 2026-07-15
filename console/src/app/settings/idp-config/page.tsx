"use client";

import { useState, useEffect } from "react";
import { useApi } from "@/lib/api";
import {
  KeyRound,
  Plus,
  Trash2,
  Save,
  Loader2,
  Globe,
  Server,
  Shield,
  ChevronDown,
  ChevronUp,
  Copy,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface IdPConfig {
  id: string;
  name: string;
  type: "saml" | "oidc" | "ldap";
  enabled: boolean;
  // SAML
  entityId?: string;
  ssoUrl?: string;
  sloUrl?: string;
  certFingerprint?: string;
  // OIDC
  issuerUrl?: string;
  authorizationEndpoint?: string;
  tokenEndpoint?: string;
  userinfoEndpoint?: string;
  clientId?: string;
  clientSecret?: string;
  scopes?: string;
  // LDAP
  ldapUrl?: string;
  bindDn?: string;
  baseDn?: string;
  userFilter?: string;
  startTls?: boolean;
  // Common
  domain?: string;
  autoProvision?: boolean;
}

const STORAGE_KEY = "ggid_idp_configs";

export default function IdPConfigPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [configs, setConfigs] = useState<IdPConfig[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [msg, setMsg] = useState("");
  const [expandedId, setExpandedId] = useState<string | null>(null);

  useEffect(() => {
    const load = async () => {
      try {
        const data = await apiFetch<{ configs?: IdPConfig[] }>("/api/v1/settings/idp");
        setConfigs(data.configs ?? []);
      } catch {
        const stored = localStorage.getItem(STORAGE_KEY);
        if (stored) setConfigs(JSON.parse(stored));
      } finally {
        setLoading(false);
      }
    };
    load();
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const persist = async (updated: IdPConfig[]) => {
    setConfigs(updated);
    setSaving(true);
    try {
      await apiFetch("/api/v1/settings/idp", {
        method: "PUT",
        body: JSON.stringify({ configs: updated }),
      });
      setMsg("IdP configurations saved");
    } catch {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(updated));
      setMsg("IdP configurations saved (offline mode)");
    } finally {
      setSaving(false);
      setTimeout(() => setMsg(""), 4000);
    }
  };

  const handleAdd = (type: IdPConfig["type"]) => {
    const newConfig: IdPConfig = {
      id: `idp-${Date.now()}`,
      name: `New ${type.toUpperCase()} Provider`,
      type,
      enabled: false,
      autoProvision: true,
      domain: "",
      ...(type === "saml" && { entityId: "", ssoUrl: "", sloUrl: "", certFingerprint: "" }),
      ...(type === "oidc" && {
        issuerUrl: "",
        authorizationEndpoint: "",
        tokenEndpoint: "",
        userinfoEndpoint: "",
        clientId: "",
        clientSecret: "",
        scopes: "openid profile email",
      }),
      ...(type === "ldap" && {
        ldapUrl: "",
        bindDn: "",
        baseDn: "",
        userFilter: "(uid={username})",
        startTls: true,
      }),
    };
    persist([...configs, newConfig]);
    setExpandedId(newConfig.id);
  };

  const handleDelete = (id: string) => {
    persist(configs.filter((c) => c.id !== id));
  };

  const handleToggle = (id: string) => {
    persist(configs.map((c) => (c.id === id ? { ...c, enabled: !c.enabled } : c)));
  };

  const handleUpdate = (id: string, field: keyof IdPConfig, value: string | boolean) => {
    setConfigs(configs.map((c) => (c.id === id ? { ...c, [field]: value } : c)));
  };

  const handleSaveAll = () => persist(configs);

  const inputCls =
    "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";
  const smallCls =
    "rounded-lg border border-gray-300 px-2 py-1.5 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";
  const cardCls =
    "rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const typeIcon = (type: string) => {
    if (type === "saml") return <Shield className="h-5 w-5 text-blue-500" />;
    if (type === "oidc") return <Globe className="h-5 w-5 text-green-500" />;
    return <Server className="h-5 w-5 text-purple-500" />;
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <KeyRound className="h-7 w-7 text-indigo-600" />{t("big1.idpConfig.title")}</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("big1.idpConfig.configureSAMLOIDCAndLDAPIdentityProvidersForPerTenantFederatedAuthentication")}</p>
        </div>
        <div className="flex items-center gap-2">
          {msg && <span className="text-sm text-green-600">{msg}</span>}
          <button
            onClick={handleSaveAll}
            disabled={saving}
            className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
          >
            {saving ? <Loader2 className="mr-1 inline h-4 w-4 animate-spin" /> : <Save className="mr-1 inline h-4 w-4" />}{t("big1.idpConfig.saveAll")}</button>
        </div>
      </div>

      {/* Add buttons */}
      <div className="flex gap-2">
        <button
          onClick={() => handleAdd("saml")}
          className="flex items-center gap-1 rounded-lg border border-blue-300 px-4 py-2 text-sm font-medium text-blue-600 hover:bg-blue-50 dark:border-blue-700 dark:text-blue-400 dark:hover:bg-blue-900/20"
        >
          <Plus className="h-4 w-4" />{t("big1.idpConfig.addSAML")}</button>
        <button
          onClick={() => handleAdd("oidc")}
          className="flex items-center gap-1 rounded-lg border border-green-300 px-4 py-2 text-sm font-medium text-green-600 hover:bg-green-50 dark:border-green-700 dark:text-green-400 dark:hover:bg-green-900/20"
        >
          <Plus className="h-4 w-4" />{t("big1.idpConfig.addOIDC")}</button>
        <button
          onClick={() => handleAdd("ldap")}
          className="flex items-center gap-1 rounded-lg border border-purple-300 px-4 py-2 text-sm font-medium text-purple-600 hover:bg-purple-50 dark:border-purple-700 dark:text-purple-400 dark:hover:bg-purple-900/20"
        >
          <Plus className="h-4 w-4" />{t("big1.idpConfig.addLDAP")}</button>
      </div>

      {/* Config list */}
      {loading ? (
        <div className="flex justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-indigo-600" />
        </div>
      ) : configs.length === 0 ? (
        <div className={`${cardCls} text-center`}>
          <KeyRound className="mx-auto mb-3 h-12 w-12 text-gray-300" />
          <p className="text-gray-500 dark:text-gray-400">{t("big1.idpConfig.noIdentityProvidersConfiguredAddASAMLOIDCOrLDAPProviderAbove")}</p>
        </div>
      ) : (
        <div className="space-y-4">
          {configs.map((cfg) => (
            <div key={cfg.id} className={cardCls}>
              {/* Header row */}
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <button
                    onClick={() => handleToggle(cfg.id)}
                    className={`flex h-6 w-11 items-center rounded-full transition-colors ${
                      cfg.enabled ? "bg-indigo-600" : "bg-gray-300 dark:bg-gray-600"
                    }`}
                  >
                    <span
                      className={`h-5 w-5 transform rounded-full bg-white shadow transition-transform ${
                        cfg.enabled ? "translate-x-5" : "translate-x-0.5"
                      }`}
                    />
                  </button>
                  {typeIcon(cfg.type)}
                  <div>
                    <input
                      className="border-none bg-transparent text-base font-semibold text-gray-900 outline-none dark:text-white"
                      value={cfg.name}
                      onChange={(e) => handleUpdate(cfg.id, "name", e.target.value)}
                    />
                    <span className="ml-2 rounded-full bg-gray-100 px-2 py-0.5 text-xs uppercase text-gray-500 dark:bg-gray-700 dark:text-gray-400">
                      {cfg.type}
                    </span>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => setExpandedId(expandedId === cfg.id ? null : cfg.id)}
                    className="rounded-lg p-2 text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700"
                  >
                    {expandedId === cfg.id ? (
                      <ChevronUp className="h-4 w-4" />
                    ) : (
                      <ChevronDown className="h-4 w-4" />
                    )}
                  </button>
                  <button
                    onClick={() => handleDelete(cfg.id)}
                    className="rounded-lg p-2 text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
              </div>

              {/* Expanded details */}
              {expandedId === cfg.id && (
                <div className="mt-4 space-y-4 border-t border-gray-200 pt-4 dark:border-gray-700">
                  {/* Common fields */}
                  <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                    <div>
                      <label className="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">{t("big1.idpConfig.domainEmailDomain")}</label>
                      <input
                        className={inputCls}
                        placeholder="example.com"
                        value={cfg.domain ?? ""}
                        onChange={(e) => handleUpdate(cfg.id, "domain", e.target.value)}
                      />
                    </div>
                    <div className="flex items-center gap-2 pt-5">
                      <input
                        type="checkbox"
                        id={`autoprov-${cfg.id}`}
                        checked={cfg.autoProvision ?? false}
                        onChange={(e) => handleUpdate(cfg.id, "autoProvision", e.target.checked)}
                        className="h-4 w-4 rounded border-gray-300 text-indigo-600"
                      />
                      <label htmlFor={`autoprov-${cfg.id}`} className="text-sm text-gray-700 dark:text-gray-300">{t("big1.idpConfig.autoProvisionUsersOnFirstLogin")}</label>
                    </div>
                  </div>

                  {/* SAML fields */}
                  {cfg.type === "saml" && (
                    <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                      <div>
                        <label className="mb-1 block text-xs font-medium text-gray-500">{t("big1.idpConfig.entityId")}</label>
                        <input className={inputCls} placeholder="https://idp.example.com/entity" value={cfg.entityId ?? ""} onChange={(e) => handleUpdate(cfg.id, "entityId", e.target.value)} />
                      </div>
                      <div>
                        <label className="mb-1 block text-xs font-medium text-gray-500">{t("big1.idpConfig.ssoUrl")}</label>
                        <input className={inputCls} placeholder="https://idp.example.com/sso" value={cfg.ssoUrl ?? ""} onChange={(e) => handleUpdate(cfg.id, "ssoUrl", e.target.value)} />
                      </div>
                      <div>
                        <label className="mb-1 block text-xs font-medium text-gray-500">{t("big1.idpConfig.sloUrl")}</label>
                        <input className={inputCls} placeholder="https://idp.example.com/slo" value={cfg.sloUrl ?? ""} onChange={(e) => handleUpdate(cfg.id, "sloUrl", e.target.value)} />
                      </div>
                      <div>
                        <label className="mb-1 block text-xs font-medium text-gray-500">{t("big1.idpConfig.certificateFingerprintSHA256")}</label>
                        <input className={inputCls} placeholder="AB:CD:EF:..." value={cfg.certFingerprint ?? ""} onChange={(e) => handleUpdate(cfg.id, "certFingerprint", e.target.value)} />
                      </div>
                    </div>
                  )}

                  {/* OIDC fields */}
                  {cfg.type === "oidc" && (
                    <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                      <div>
                        <label className="mb-1 block text-xs font-medium text-gray-500">{t("big1.idpConfig.issuerUrl")}</label>
                        <input className={inputCls} placeholder="https://accounts.google.com" value={cfg.issuerUrl ?? ""} onChange={(e) => handleUpdate(cfg.id, "issuerUrl", e.target.value)} />
                      </div>
                      <div>
                        <label className="mb-1 block text-xs font-medium text-gray-500">{t("big1.idpConfig.clientId")}</label>
                        <input className={inputCls} placeholder="your-client-id" value={cfg.clientId ?? ""} onChange={(e) => handleUpdate(cfg.id, "clientId", e.target.value)} />
                      </div>
                      <div>
                        <label className="mb-1 block text-xs font-medium text-gray-500">{t("big1.idpConfig.clientSecret")}</label>
                        <input className={inputCls} type="password" placeholder="••••••••" value={cfg.clientSecret ?? ""} onChange={(e) => handleUpdate(cfg.id, "clientSecret", e.target.value)} />
                      </div>
                      <div>
                        <label className="mb-1 block text-xs font-medium text-gray-500">{t("big1.idpConfig.scopes")}</label>
                        <input className={inputCls} placeholder="openid profile email" value={cfg.scopes ?? ""} onChange={(e) => handleUpdate(cfg.id, "scopes", e.target.value)} />
                      </div>
                      <div>
                        <label className="mb-1 block text-xs font-medium text-gray-500">{t("big1.idpConfig.authorizationEndpoint")}</label>
                        <input className={inputCls} placeholder="https://idp/authorize" value={cfg.authorizationEndpoint ?? ""} onChange={(e) => handleUpdate(cfg.id, "authorizationEndpoint", e.target.value)} />
                      </div>
                      <div>
                        <label className="mb-1 block text-xs font-medium text-gray-500">{t("big1.idpConfig.tokenEndpoint")}</label>
                        <input className={inputCls} placeholder="https://idp/token" value={cfg.tokenEndpoint ?? ""} onChange={(e) => handleUpdate(cfg.id, "tokenEndpoint", e.target.value)} />
                      </div>
                    </div>
                  )}

                  {/* LDAP fields */}
                  {cfg.type === "ldap" && (
                    <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                      <div>
                        <label className="mb-1 block text-xs font-medium text-gray-500">{t("big1.idpConfig.ldapUrl")}</label>
                        <input className={inputCls} placeholder="ldap://ldap.example.com:389" value={cfg.ldapUrl ?? ""} onChange={(e) => handleUpdate(cfg.id, "ldapUrl", e.target.value)} />
                      </div>
                      <div>
                        <label className="mb-1 block text-xs font-medium text-gray-500">{t("big1.idpConfig.bindDn")}</label>
                        <input className={inputCls} placeholder="cn=admin,dc=example,dc=com" value={cfg.bindDn ?? ""} onChange={(e) => handleUpdate(cfg.id, "bindDn", e.target.value)} />
                      </div>
                      <div>
                        <label className="mb-1 block text-xs font-medium text-gray-500">{t("big1.idpConfig.baseDn")}</label>
                        <input className={inputCls} placeholder="ou=users,dc=example,dc=com" value={cfg.baseDn ?? ""} onChange={(e) => handleUpdate(cfg.id, "baseDn", e.target.value)} />
                      </div>
                      <div>
                        <label className="mb-1 block text-xs font-medium text-gray-500">{t("big1.idpConfig.userFilter")}</label>
                        <input className={inputCls} placeholder="(uid={username})" value={cfg.userFilter ?? ""} onChange={(e) => handleUpdate(cfg.id, "userFilter", e.target.value)} />
                      </div>
                      <div className="flex items-center gap-2 pt-5">
                        <input
                          type="checkbox"
                          id={`starttls-${cfg.id}`}
                          checked={cfg.startTls ?? true}
                          onChange={(e) => handleUpdate(cfg.id, "startTls", e.target.checked)}
                          className="h-4 w-4 rounded border-gray-300 text-indigo-600"
                        />
                        <label htmlFor={`starttls-${cfg.id}`} className="text-sm text-gray-700 dark:text-gray-300">{t("big1.idpConfig.useStartTLS")}</label>
                      </div>
                    </div>
                  )}
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

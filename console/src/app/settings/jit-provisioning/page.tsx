"use client";

import { useState } from "react";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";
import {
  Zap, Loader2, AlertCircle, X, Check, Save, Shield, Users,
} from "lucide-react";

interface JITProvider {
  id: string;
  name: string;
  type: "saml" | "oidc" | "social";
  enabled: boolean;
  attribute_mapping: { claim: string; attribute: string }[];
  auto_assign_role: string | null;
  default_org: string | null;
}

interface JITConfig {
  enabled: boolean;
  providers: JITProvider[];
}

export default function JITProvisioningPage() {
  const t = useTranslations();
  const { apiFetch } = useApi();
  const [config, setConfig] = useState<JITConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editProvider, setEditProvider] = useState<JITProvider | null>(null);
  const [saving, setSaving] = useState(false);

  useState(() => {
    (async () => {
      try { setConfig(await apiFetch<JITConfig>("/api/v1/settings/jit-provisioning").catch(() => null)); }
      catch { setError("Failed to load JIT config"); }
      finally { setLoading(false); }
    })();
  });

  const handleSave = async () => {
    if (!editProvider || !config) return;
    setSaving(true);
    try {
      const updated = { ...config, providers: config.providers.map((p) => p.id === editProvider.id ? editProvider : p) };
      await apiFetch("/api/v1/settings/jit-provisioning", { method: "PUT", body: JSON.stringify(updated) });
      setConfig(updated); setEditProvider(null);
    } catch { setError("Save failed"); }
    finally { setSaving(false); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Zap className="h-6 w-6 text-indigo-600" />{t("jitProvisioning.title")}</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Auto-provision users on first login from external identity providers.</p>
      </div>

      {error && <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>
      : config ? (
        <>
          {/* Global toggle */}
          <div className={`${cardCls} ${config.enabled ? "border-green-300 dark:border-green-700" : ""}`}>
            <div className="flex items-center gap-3">
              <div className={`rounded-lg p-2 ${config.enabled ? "bg-green-100 dark:bg-green-900/30" : "bg-gray-100 dark:bg-gray-700"}`}><Zap className={`h-5 w-5 ${config.enabled ? "text-green-600" : "text-gray-400"}`} /></div>
              <div><h3 className="font-semibold text-gray-800 dark:text-gray-200">JIT Provisioning {config.enabled ? "Enabled" : "Disabled"}</h3><p className="text-sm text-gray-400">Users are auto-created on first authentication from mapped providers</p></div>
            </div>
          </div>

          {/* Provider cards */}
          <div className="space-y-3">
            {config.providers.map((p) => (
              <div key={p.id} className={cardCls}>
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-3">
                    <div className="rounded-lg bg-indigo-100 p-2 dark:bg-indigo-900/30"><Shield className="h-4 w-4 text-indigo-600" /></div>
                    <div>
                      <div className="flex items-center gap-2"><span className="font-medium text-gray-800 dark:text-gray-200">{p.name}</span><span className="rounded-full bg-gray-100 px-2 py-0.5 text-xs uppercase text-gray-500 dark:bg-gray-700">{p.type}</span>{!p.enabled && <span className="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-400 dark:bg-gray-700">Disabled</span>}</div>
                      <div className="mt-2 flex flex-wrap gap-2 text-xs text-gray-400">
                        {p.auto_assign_role && <span className="flex items-center gap-1"><Shield className="h-3 w-3" />Role: {p.auto_assign_role}</span>}
                        {p.default_org && <span className="flex items-center gap-1"><Users className="h-3 w-3" />Org: {p.default_org}</span>}
                        <span>{p.attribute_mapping.length} mappings</span>
                      </div>
                    </div>
                  </div>
                  <button onClick={() => setEditProvider({ ...p })} className="rounded-lg border border-gray-300 px-3 py-1.5 text-xs font-medium text-gray-500 hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700">Edit</button>
                </div>
                {/* Attribute mapping preview */}
                {p.attribute_mapping.length > 0 && (
                  <div className="mt-3 rounded-lg bg-gray-50 p-2 dark:bg-gray-900/30">
                    <div className="flex flex-wrap gap-1">
                      {p.attribute_mapping.map((m, i) => <span key={i} className="rounded bg-indigo-100 px-1.5 py-0.5 font-mono text-xs text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400">{m.claim} → {m.attribute}</span>)}
                    </div>
                  </div>
                )}
              </div>
            ))}
          </div>
        </>
      ) : <div className={cardCls}><div className="py-12 text-center"><Zap className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No JIT provisioning configured.</p></div></div>}

      {/* Edit provider modal */}
      {editProvider && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => !saving && setEditProvider(null)}>
          <div className="w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between"><h2 className="text-lg font-semibold text-gray-900 dark:text-white">Edit {editProvider.name}</h2><button onClick={() => setEditProvider(null)}><X className="h-5 w-5 text-gray-400" /></button></div>
            <div className="mt-4 space-y-4">
              <label className="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300"><input type="checkbox" checked={editProvider.enabled} onChange={(e) => setEditProvider((p) => p ? { ...p, enabled: e.target.checked } : null)} className="rounded border-gray-300 text-indigo-600" />Enable provider</label>
              <div><label className="text-sm font-medium text-gray-700 dark:text-gray-300">Auto-Assign Role</label><input value={editProvider.auto_assign_role ?? ""} onChange={(e) => setEditProvider((p) => p ? { ...p, auto_assign_role: e.target.value || null } : null)} placeholder="viewer" className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" /></div>
              <div><label className="text-sm font-medium text-gray-700 dark:text-gray-300">Default Organization</label><input value={editProvider.default_org ?? ""} onChange={(e) => setEditProvider((p) => p ? { ...p, default_org: e.target.value || null } : null)} placeholder="org-uuid" className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" /></div>
            </div>
            <div className="mt-5 flex justify-end gap-2"><button onClick={() => setEditProvider(null)} className="rounded-lg px-4 py-2 text-sm text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700">Cancel</button><button onClick={handleSave} disabled={saving} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}Save</button></div>
          </div>
        </div>
      )}
    </div>
  );
}

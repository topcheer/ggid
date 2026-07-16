"use client";

import { useState, useEffect, useCallback } from "react";
import { Network, Plus, X, Plug, CheckCircle, XCircle } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface FederatedIdP {
  id: string;
  provider_type: "saml" | "oidc";
  entity_id: string;
  name: string;
  status: "active" | "inactive" | "error";
  last_sync: string;
  trust_level: "full" | "limited" | "conditional";
}

const statusColors: Record<string, string> = {
  active: "text-green-600", inactive: "text-gray-500", error: "text-red-600",
};
const trustColors: Record<string, string> = {
  full: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400",
  limited: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  conditional: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400",
};

export default function IdpFederationPage() {
  const t = useTranslations();

  const [idps, setIdps] = useState<FederatedIdP[]>([]);
  const [loading, setLoading] = useState(false);
  const [tab, setTab] = useState("saml");
  const [showAdd, setShowAdd] = useState(false);
  const [form, setForm] = useState({ provider_type: "saml", entity_id: "", name: "", trust_level: "limited" });
  const [testingId, setTestingId] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/identity/idp-federation", { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const d = await res.json(); setIdps(d.idps || d || []); }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const add = async () => {
    if (!form.entity_id) return;
    try { await fetch("/api/v1/identity/idp-federation", { method: "POST", headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify(form) }); setShowAdd(false); setForm({ provider_type: "saml", entity_id: "", name: "", trust_level: "limited" }); fetchData(); }
    catch { /* noop */ }
  };

  const testConnection = async (id: string) => {
    setTestingId(id);
    try { await fetch("/api/v1/identity/idp-federation/" + id + "/test", { method: "POST", headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); }
    catch { /* noop */ }
    finally { setTestingId(null); }
  };

  const filtered = idps.filter((i) => i.provider_type === tab);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold flex items-center gap-2"><Network className="w-6 h-6 text-blue-500" /> {t("big1.idpFederation.title")}</h1><p className="text-sm text-gray-500 mt-1">{t("big1.idpFederation.manageFederatedIdentityProvidersWithTrustLevelsAndConnectionTesting")}</p></div>
        <button onClick={() => setShowAdd(true)} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 flex items-center gap-2"><Plus className="w-4 h-4" />{t("big1.idpFederation.addFederation")}</button>
      </div>

      <div className="flex gap-2">
        <button onClick={() => setTab("saml")} className={"px-4 py-2 rounded-lg text-sm font-medium " + (tab === "saml" ? "bg-blue-600 text-white" : "border dark:border-gray-700")}>{t("big1.idpFederation.saml")}</button>
        <button onClick={() => setTab("oidc")} className={"px-4 py-2 rounded-lg text-sm font-medium " + (tab === "oidc" ? "bg-blue-600 text-white" : "border dark:border-gray-700")}>{t("big1.idpFederation.oidc")}</button>
      </div>

      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">{t("big1.idpFederation.name")}</th><th className="px-4 py-3 text-left font-medium">{t("big1.idpFederation.entityId")}</th><th className="px-4 py-3 text-left font-medium">{t("big1.idpFederation.status")}</th><th className="px-4 py-3 text-left font-medium">{t("big1.idpFederation.trustLevel")}</th><th className="px-4 py-3 text-left font-medium">{t("big1.idpFederation.lastSync")}</th><th className="px-4 py-3 text-left font-medium">{t("big1.idpFederation.action")}</th></tr></thead>
          <tbody className="divide-y dark:divide-gray-800">{filtered.map((idp) => (<tr key={idp.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3 font-medium">{idp.name}</td><td className="px-4 py-3 font-mono text-xs text-gray-500">{idp.entity_id}</td><td className="px-4 py-3"><span className={"flex items-center gap-1 text-xs " + statusColors[idp.status]}>{idp.status === "active" ? <CheckCircle className="w-3.5 h-3.5" /> : <XCircle className="w-3.5 h-3.5" />}{idp.status}</span></td><td className="px-4 py-3"><span className={"px-2 py-0.5 rounded text-xs " + trustColors[idp.trust_level]}>{idp.trust_level}</span></td><td className="px-4 py-3 text-xs text-gray-400">{idp.last_sync}</td><td className="px-4 py-3"><button onClick={() => testConnection(idp.id)} disabled={testingId === idp.id} className="text-xs font-medium text-blue-600 hover:underline disabled:opacity-50 flex items-center gap-1"><Plug className="w-3 h-3" /> {testingId === idp.id ? "Testing..." : "Test"}</button></td></tr>))}{filtered.length === 0 && !loading && <tr><td colSpan={6} className="px-4 py-8 text-center text-gray-500">{t("big1.idpFederation.noFederatedIdPs")}</td></tr>}</tbody>
        </table>
      </div>

      {showAdd && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setShowAdd(false)}>
          <div role="dialog" aria-modal="true" className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between px-6 py-4 border-b dark:border-gray-800"><h3 className="font-semibold">{t("big1.idpFederation.addFederation")}</h3><button onClick={() => setShowAdd(false)} aria-label="Close"><X className="w-5 h-5 text-gray-400" /></button></div>
            <div className="px-6 py-4 space-y-3">
              <div><label className="text-sm font-medium">{t("big1.idpFederation.providerType")}</label><select aria-label="Select option" value={form.provider_type} onChange={(e) => setForm({ ...form, provider_type: e.target.value })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm"><option value="saml">{t("big1.idpFederation.saml")}</option><option value="oidc">{t("big1.idpFederation.oidc")}</option></select></div>
              <div><label className="text-sm font-medium">{t("big1.idpFederation.name")}</label><input aria-label="form" type="text" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div>
              <div><label className="text-sm font-medium">{t("big1.idpFederation.entityIDIssuer")}</label><input aria-label="form" type="text" value={form.entity_id} onChange={(e) => setForm({ ...form, entity_id: e.target.value })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm font-mono" /></div>
              <div><label className="text-sm font-medium">{t("big1.idpFederation.trustLevel")}</label><select aria-label="form" value={form.trust_level} onChange={(e) => setForm({ ...form, trust_level: e.target.value })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm"><option value="full">{t("big1.idpFederation.full")}</option><option value="limited">{t("big1.idpFederation.limited")}</option><option value="conditional">{t("big1.idpFederation.conditional")}</option></select></div>
            </div>
            <div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800"><button onClick={() => setShowAdd(false)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">{t("big1.idpFederation.cancel")}</button><button onClick={add} disabled={!form.entity_id} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50" aria-label="Action">{t("big1.idpFederation.add")}</button></div>
          </div>
        </div>
      )}
    </div>
  );
}

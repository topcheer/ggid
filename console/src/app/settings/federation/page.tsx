"use client";
import { useState, useCallback, useEffect } from "react";
import {
  Globe, Loader2, AlertCircle, X, RefreshCw, Plus, Trash2, Check,
  Shield, Lock, Zap, ChevronRight, CheckCircle2, XCircle, Clock,
  KeyRound, ArrowRight, Server, FileText, Activity,
} from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";
import { useTranslations } from "@/lib/i18n";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface FederationEntity {
  id: string; entity_id: string; entity_name: string; entity_type: string;
  protocol: "saml" | "oidc" | "ws-fed"; metadata_url?: string; issuer?: string;
  trust_level: "pending" | "verified" | "trusted"; trust_direction: "inbound" | "outbound" | "bidirectional";
  jwks_url?: string; enabled: boolean; created_at: string; last_checked?: string;
}

type Tab = "entities" | "wizard" | "trust" | "monitoring";

const PROTOCOL_CFG: Record<string, { label: string; icon: typeof Globe; color: string }> = {
  saml: { label: "SAML 2.0", icon: FileText, color: "text-amber-500" },
  oidc: { label: "OpenID Connect", icon: KeyRound, color: "text-blue-500" },
  "ws-fed": { label: "WS-Federation", icon: Shield, color: "text-purple-500" },
};

const TRUST_CFG: Record<string, { label: string; color: string; bg: string }> = {
  trusted: { label: "Trusted", color: "text-green-600", bg: "bg-green-100 dark:bg-green-900/30" },
  verified: { label: "Verified", color: "text-blue-600", bg: "bg-blue-100 dark:bg-blue-900/30" },
  pending: { label: "Pending", color: "text-yellow-600", bg: "bg-yellow-100 dark:bg-yellow-900/30" },
};

export default function FederationPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("entities");
  const [entities, setEntities] = useState<FederationEntity[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  // Wizard state
  const [wStep, setWStep] = useState(0);
  const [wProtocol, setWProtocol] = useState<"saml" | "oidc">("saml");
  const [wName, setWName] = useState("");
  const [wEntityId, setWEntityId] = useState("");
  const [wMetadataUrl, setWMetadataUrl] = useState("");
  const [wIssuer, setWIssuer] = useState("");
  const [wJwksUrl, setWJwksUrl] = useState("");
  const [wDirection, setWDirection] = useState<"inbound" | "outbound" | "bidirectional">("inbound");
  const [wAutoImport, setWAutoImport] = useState(false);
  const [importing, setImporting] = useState(false);
  const [wizardDone, setWizardDone] = useState(false);

  const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
  const H = { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID };
  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/identity/federation/entities", { headers: h }).catch(() => null);
      if (res?.ok) { const d = await res.json(); setEntities(d.entities || []); }
    } catch { setError(t("federation.loadError")); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { load(); }, [load]);

  const createEntity = async () => {
    setActionLoading("create");
    try {
      await fetch("/api/v1/identity/federation/entities", {
        method: "POST", headers: H,
        body: JSON.stringify({
          entity_id: wEntityId || wName.toLowerCase().replace(/\s+/g, "-"),
          entity_name: wName, entity_type: "idp", protocol: wProtocol,
          metadata_url: wMetadataUrl || undefined, issuer: wIssuer || undefined,
          jwks_url: wJwksUrl || undefined, trust_level: "pending", trust_direction: wDirection,
        }),
      });
      setWizardDone(true); load();
    } catch { setError(t("federation.createError")); }
    finally { setActionLoading(null); }
  };

  const deleteEntity = async (id: string) => {
    setActionLoading(`del-${id}`);
    try { await fetch(`/api/v1/identity/federation/entities?id=${id}`, { method: "DELETE", headers: h }); load(); }
    catch { setError(t("federation.deleteError")); }
    finally { setActionLoading(null); }
  };

  const importMetadata = async () => {
    if (!wMetadataUrl) return;
    setImporting(true);
    try {
      // Simulate metadata fetch + parse
      await new Promise(r => setTimeout(r, 1000));
      if (wProtocol === "saml" && !wEntityId) setWEntityId(`https://${new URL(wMetadataUrl).hostname}/idp`);
      if (wProtocol === "saml" && !wIssuer) setWIssuer(`https://${new URL(wMetadataUrl).hostname}/idp`);
      if (wProtocol === "oidc" && !wJwksUrl) setWJwksUrl(`${new URL(wMetadataUrl).origin}/.well-known/jwks.json`);
    } catch { /* noop */ }
    finally { setImporting(false); }
  };

  const resetWizard = () => {
    setWStep(0); setWProtocol("saml"); setWName(""); setWEntityId(""); setWMetadataUrl("");
    setWIssuer(""); setWJwksUrl(""); setWDirection("inbound"); setWAutoImport(false); setWizardDone(false);
  };

  const activeEntities = entities.filter(e => e.enabled);
  const trustedEntities = entities.filter(e => e.trust_level === "trusted");

  const wizardSteps = [t("federation.stepProtocol"), t("federation.stepEntity"), t("federation.stepReview")];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <Globe className="h-6 w-6 text-cyan-500" /> {t("federation.title")}
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("federation.subtitle")}</p>
        </div>
        <button onClick={() => { resetWizard(); setTab("wizard"); }} className="flex items-center gap-1 rounded-lg bg-cyan-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-cyan-700">
          <Plus className="h-3 w-3" /> {t("federation.addProvider")}
        </button>
      </div>

      {error && (
        <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}
          <button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "entities" as Tab, label: t("federation.providers"), icon: Server },
          { id: "wizard" as Tab, label: t("federation.setupWizard"), icon: Zap },
          { id: "trust" as Tab, label: t("federation.trustLevels"), icon: Shield },
          { id: "monitoring" as Tab, label: t("federation.monitoring"), icon: Activity },
        ]).map(tb => {
          const Icon = tb.icon;
          return (
            <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id}
              className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === tb.id ? "border-cyan-600 text-cyan-600 dark:text-cyan-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}>
              <Icon className="h-4 w-4" /> {tb.label}
            </button>
          );
        })}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-cyan-500" /></div> : (<>

      {/* ════ ENTITIES ════ */}
      {tab === "entities" && (
        <div>
          {entities.length === 0 ? (
            <div className={card}><div className="py-12 text-center"><Globe className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">{t("federation.noProviders")}</p><button onClick={() => { resetWizard(); setTab("wizard"); }} className="mt-3 text-sm text-cyan-600 hover:underline">{t("federation.startWizard")}</button></div></div>
          ) : (
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">{entities.map(e => {
              const pCfg = PROTOCOL_CFG[e.protocol] || PROTOCOL_CFG.saml;
              const tCfg = TRUST_CFG[e.trust_level] || TRUST_CFG.pending;
              const PIcon = pCfg.icon;
              return (
                <div key={e.id} className={card + " hover:shadow-md transition"}>
                  <div className="flex items-start justify-between">
                    <div className="flex items-center gap-3">
                      <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-gray-100 dark:bg-gray-700"><PIcon className={`h-5 w-5 ${pCfg.color}`} /></div>
                      <div><h3 className="font-semibold text-sm">{e.entity_name}</h3><p className="text-xs text-gray-400">{pCfg.label}</p></div>
                    </div>
                    <div className="flex flex-col items-end gap-1">
                      <span className={`px-1.5 py-0.5 rounded text-xs font-medium ${tCfg.bg} ${tCfg.color}`}>{tCfg.label}</span>
                      <span className={`h-2 w-2 rounded-full ${e.enabled ? "bg-green-500 animate-pulse" : "bg-gray-400"}`} />
                    </div>
                  </div>
                  <div className="mt-3 space-y-1 text-xs text-gray-500">
                    <p>{t("federation.entityId")}: <span className="font-mono">{e.entity_id}</span></p>
                    {e.metadata_url && <p className="truncate">URL: <span className="font-mono">{e.metadata_url}</span></p>}
                    <p>{t("federation.direction")}: <span className="font-mono">{e.trust_direction}</span></p>
                    {e.last_checked && <p>{t("federation.lastChecked")}: {new Date(e.last_checked).toLocaleDateString()}</p>}
                  </div>
                  <div className="mt-3 flex justify-end">
                    <button onClick={() => deleteEntity(e.id)} disabled={actionLoading === `del-${e.id}`} aria-label={t("federation.delete")} className="rounded p-1 text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20">
                      {actionLoading === `del-${e.id}` ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Trash2 className="h-3.5 w-3.5" />}
                    </button>
                  </div>
                </div>
              );
            })}</div>
          )}
        </div>
      )}

      {/* ════ WIZARD ════ */}
      {tab === "wizard" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
          {/* Step indicator */}
          <div className={card + " lg:col-span-1"}>
            <h3 className="mb-4 text-sm font-semibold uppercase text-gray-400">{t("federation.steps")}</h3>
            <div className="space-y-3">
              {wizardSteps.map((label, i) => (
                <div key={i} className="flex items-center gap-3">
                  <div className={`flex h-8 w-8 items-center justify-center rounded-full ${i < wStep ? "bg-green-500 text-white" : i === wStep ? "bg-cyan-600 text-white" : "bg-gray-200 dark:bg-gray-700 text-gray-400"}`}>
                    {i < wStep ? <Check className="h-4 w-4" /> : i + 1}
                  </div>
                  <span className={`text-sm ${i === wStep ? "font-medium text-cyan-600" : i < wStep ? "text-gray-400" : "text-gray-400"}`}>{label}</span>
                </div>
              ))}
            </div>
          </div>

          {/* Step content */}
          <div className={card + " lg:col-span-2"}>
            {wizardDone ? (
              <div className="py-8 text-center">
                <CheckCircle2 className="mx-auto h-12 w-12 text-green-500" />
                <p className="mt-4 text-sm font-medium">{t("federation.setupComplete")}</p>
                <p className="mt-1 text-xs text-gray-400">{t("federation.providerAdded")}</p>
                <button onClick={() => { resetWizard(); setTab("entities"); }} className="mt-4 rounded-lg bg-cyan-600 px-4 py-2 text-sm font-medium text-white hover:bg-cyan-700">{t("federation.viewProviders")}</button>
              </div>
            ) : (
              <>
                {/* Step 0: Protocol */}
                {wStep === 0 && (
                  <div>
                    <h3 className="mb-4 text-sm font-semibold">{t("federation.chooseProtocol")}</h3>
                    <div className="grid grid-cols-2 gap-3">
                      {([["saml", "SAML 2.0", FileText, "text-amber-500"], ["oidc", "OpenID Connect", KeyRound, "text-blue-500"]] as const).map(([val, label, Icon, color]) => (
                        <button key={val} onClick={() => setWProtocol(val)} aria-pressed={wProtocol === val}
                          className={`flex flex-col items-center gap-2 rounded-xl border-2 p-6 transition ${wProtocol === val ? "border-cyan-500 bg-cyan-50 dark:bg-cyan-950/30" : "border-gray-200 dark:border-gray-700"}`}>
                          <Icon className={`h-8 w-8 ${color}`} /><span className="text-sm font-medium">{label}</span>
                        </button>
                      ))}
                    </div>
                    <div className="mt-4"><label className="text-sm font-medium">{t("federation.trustDirection")}</label>
                      <select value={wDirection} onChange={e => setWDirection(e.target.value as typeof wDirection)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">
                        <option value="inbound">{t("federation.inbound")}</option><option value="outbound">{t("federation.outbound")}</option><option value="bidirectional">{t("federation.bidirectional")}</option>
                      </select>
                    </div>
                    <div className="mt-4 flex justify-end"><button onClick={() => setWStep(1)} className="rounded-lg bg-cyan-600 px-4 py-2 text-sm font-medium text-white hover:bg-cyan-700">{t("common.next")} <ChevronRight className="inline h-4 w-4" /></button></div>
                  </div>
                )}

                {/* Step 1: Entity details */}
                {wStep === 1 && (
                  <div>
                    <h3 className="mb-4 text-sm font-semibold">{t("federation.entityDetails")}</h3>
                    <div className="space-y-3">
                      <div><label className="text-sm font-medium">{t("federation.providerName")}</label><input type="text" value={wName} onChange={e => setWName(e.target.value)} placeholder="Corporate Azure AD" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" autoFocus /></div>
                      <div><label className="text-sm font-medium">{t("federation.metadataUrl")}</label>
                        <div className="flex gap-2">
                          <input type="text" value={wMetadataUrl} onChange={e => setWMetadataUrl(e.target.value)} placeholder="https://login.microsoftonline.com/.../federationmetadata.xml" className="mt-1 flex-1 rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" />
                          <button onClick={importMetadata} disabled={!wMetadataUrl || importing} className="mt-1 flex items-center gap-1 rounded-lg border border-gray-300 px-3 py-2 text-xs dark:border-gray-700">
                            {importing ? <Loader2 className="h-3 w-3 animate-spin" /> : <Zap className="h-3 w-3" />} {t("federation.autoImport")}
                          </button>
                        </div>
                      </div>
                      <div><label className="text-sm font-medium">{t("federation.entityId")}</label><input type="text" value={wEntityId} onChange={e => setWEntityId(e.target.value)} placeholder="https://idp.example.com" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div>
                      {wProtocol === "saml" && <div><label className="text-sm font-medium">{t("federation.issuer")}</label><input type="text" value={wIssuer} onChange={e => setWIssuer(e.target.value)} placeholder="https://idp.example.com" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div>}
                      {wProtocol === "oidc" && <div><label className="text-sm font-medium">{t("federation.jwksUrl")}</label><input type="text" value={wJwksUrl} onChange={e => setWJwksUrl(e.target.value)} placeholder="https://idp.example.com/.well-known/jwks.json" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div>}
                    </div>
                    <div className="mt-4 flex justify-between"><button onClick={() => setWStep(0)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">{t("common.back")}</button><button onClick={() => setWStep(2)} disabled={!wName} className="rounded-lg bg-cyan-600 px-4 py-2 text-sm font-medium text-white hover:bg-cyan-700 disabled:opacity-50">{t("common.next")} <ChevronRight className="inline h-4 w-4" /></button></div>
                  </div>
                )}

                {/* Step 2: Review */}
                {wStep === 2 && (
                  <div>
                    <h3 className="mb-4 text-sm font-semibold">{t("federation.review")}</h3>
                    <div className="rounded-lg border p-4 dark:border-gray-700 space-y-2 text-sm">
                      <div className="flex justify-between"><span className="text-gray-400">{t("federation.protocol")}</span><span className="font-medium">{PROTOCOL_CFG[wProtocol]?.label}</span></div>
                      <div className="flex justify-between"><span className="text-gray-400">{t("federation.providerName")}</span><span className="font-medium">{wName}</span></div>
                      <div className="flex justify-between"><span className="text-gray-400">{t("federation.entityId")}</span><span className="font-mono text-xs">{wEntityId || wName.toLowerCase().replace(/\s+/g, "-")}</span></div>
                      {wMetadataUrl && <div className="flex justify-between"><span className="text-gray-400">Metadata URL</span><span className="font-mono text-xs truncate max-w-xs">{wMetadataUrl}</span></div>}
                      <div className="flex justify-between"><span className="text-gray-400">{t("federation.direction")}</span><span className="font-medium">{wDirection}</span></div>
                      <div className="flex justify-between"><span className="text-gray-400">{t("federation.trustLevel")}</span><span className="px-1.5 py-0.5 rounded bg-yellow-100 dark:bg-yellow-900/30 text-yellow-600 text-xs">Pending Verification</span></div>
                    </div>
                    <div className="mt-4 rounded-lg bg-blue-50 dark:bg-blue-900/20 p-3 text-xs text-blue-600 dark:text-blue-400">
                      <p className="flex items-center gap-1"><Lock className="h-3 w-3" /> {t("federation.trustNote")}</p>
                    </div>
                    <div className="mt-4 flex justify-between">
                      <button onClick={() => setWStep(1)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">{t("common.back")}</button>
                      <button onClick={createEntity} disabled={actionLoading === "create"} className="flex items-center gap-1 rounded-lg bg-cyan-600 px-4 py-2 text-sm font-medium text-white hover:bg-cyan-700 disabled:opacity-50">
                        {actionLoading === "create" ? <Loader2 className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />} {t("federation.create")}
                      </button>
                    </div>
                  </div>
                )}
              </>
            )}
          </div>
        </div>
      )}

      {/* ════ TRUST LEVELS ════ */}
      {tab === "trust" && (
        <div className="space-y-6">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
            <div className={`${card} border-green-200 dark:border-green-800`}>
              <div className="flex items-center gap-2"><CheckCircle2 className="h-5 w-5 text-green-500" /><h3 className="text-sm font-semibold">{t("federation.trusted")}</h3></div>
              <p className="mt-2 text-3xl font-bold text-green-600">{trustedEntities.length}</p>
              <p className="text-xs text-gray-400">{t("federation.fullTrust")}</p>
            </div>
            <div className={`${card} border-blue-200 dark:border-blue-800`}>
              <div className="flex items-center gap-2"><Shield className="h-5 w-5 text-blue-500" /><h3 className="text-sm font-semibold">{t("federation.verified")}</h3></div>
              <p className="mt-2 text-3xl font-bold text-blue-600">{entities.filter(e => e.trust_level === "verified").length}</p>
              <p className="text-xs text-gray-400">{t("federation.metadataVerified")}</p>
            </div>
            <div className={`${card} border-yellow-200 dark:border-yellow-800`}>
              <div className="flex items-center gap-2"><Clock className="h-5 w-5 text-yellow-500" /><h3 className="text-sm font-semibold">{t("federation.pending")}</h3></div>
              <p className="mt-2 text-3xl font-bold text-yellow-600">{entities.filter(e => e.trust_level === "pending").length}</p>
              <p className="text-xs text-gray-400">{t("federation.awaitingVerification")}</p>
            </div>
          </div>
          <div className={card}>
            <h3 className="mb-3 text-sm font-semibold uppercase text-gray-400">{t("federation.trustMatrix")}</h3>
            <div className="overflow-x-auto"><table className="w-full text-sm">
              <thead><tr><th className="px-3 py-2 text-left text-xs text-gray-400">{t("federation.provider")}</th><th className="px-3 py-2 text-left text-xs text-gray-400">{t("federation.protocol")}</th><th className="px-3 py-2 text-center text-xs text-gray-400">{t("federation.trustLevel")}</th><th className="px-3 py-2 text-center text-xs text-gray-400">{t("federation.direction")}</th></tr></thead>
              <tbody className="divide-y dark:divide-gray-800">
                {entities.map(e => { const tCfg = TRUST_CFG[e.trust_level] || TRUST_CFG.pending; return (
                  <tr key={e.id}><td className="px-3 py-2 text-xs font-medium">{e.entity_name}</td><td className="px-3 py-2 text-xs">{PROTOCOL_CFG[e.protocol]?.label || e.protocol}</td><td className="px-3 py-2 text-center"><span className={`px-1.5 py-0.5 rounded text-xs ${tCfg.bg} ${tCfg.color}`}>{tCfg.label}</span></td><td className="px-3 py-2 text-center"><span className="text-xs font-mono">{e.trust_direction}</span></td></tr>
                ); })}
              </tbody>
            </table></div>
          </div>
        </div>
      )}

      {/* ════ MONITORING ════ */}
      {tab === "monitoring" && (
        <div className="space-y-6">
          <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
            <div className={card + " text-center"}><Server className="mx-auto h-5 w-5 text-cyan-400" /><p className="mt-2 text-2xl font-bold">{activeEntities.length}</p><p className="text-xs text-gray-400">{t("federation.activeProviders")}</p></div>
            <div className={card + " text-center"}><CheckCircle2 className="mx-auto h-5 w-5 text-green-400" /><p className="mt-2 text-2xl font-bold">{trustedEntities.length}</p><p className="text-xs text-gray-400">{t("federation.trustedProviders")}</p></div>
            <div className={card + " text-center"}><Activity className="mx-auto h-5 w-5 text-blue-400" /><p className="mt-2 text-2xl font-bold">{entities.filter(e => e.last_checked).length}</p><p className="text-xs text-gray-400">{t("federation.healthChecked")}</p></div>
            <div className={card + " text-center"}><Clock className="mx-auto h-5 w-5 text-yellow-400" /><p className="mt-2 text-2xl font-bold">{entities.filter(e => !e.last_checked).length}</p><p className="text-xs text-gray-400">{t("federation.neverChecked")}</p></div>
          </div>
          <div className={card}>
            <h3 className="mb-3 text-sm font-semibold uppercase text-gray-400">{t("federation.healthStatus")}</h3>
            <div className="space-y-2">
              {entities.map(e => (
                <div key={e.id} className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700">
                  <div className="flex items-center gap-3">
                    <div className={`flex h-8 w-8 items-center justify-center rounded-lg ${e.enabled ? "bg-green-100 dark:bg-green-900/30" : "bg-gray-100 dark:bg-gray-800"}`}><Server className={`h-4 w-4 ${e.enabled ? "text-green-500" : "text-gray-400"}`} /></div>
                    <div><span className="text-sm font-medium">{e.entity_name}</span><p className="text-xs text-gray-400">{PROTOCOL_CFG[e.protocol]?.label}</p></div>
                  </div>
                  <div className="text-right">
                    {e.last_checked ? <span className="text-xs text-gray-400">{t("federation.checked")}: {new Date(e.last_checked).toLocaleDateString()}</span> : <span className="text-xs text-yellow-500">{t("federation.notChecked")}</span>}
                  </div>
                </div>
              ))}
              {entities.length === 0 && <p className="text-sm text-gray-400">{t("federation.noProviders")}</p>}
            </div>
          </div>
        </div>
      )}

      </>)}
    </div>
  );
}

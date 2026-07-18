"use client";

import { useState, useCallback, useEffect } from "react";
import {
  Network, Shield, Loader2, AlertCircle, X, RefreshCw, Plus, Trash2,
  Check, CheckCircle, XCircle, Globe, KeyRound, ArrowRight, Code,
  TestTube, Eye, Download, Zap, Settings, ChevronRight, Mail,
  AlertTriangle, Clock, FileJson,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface TrustRelation {
  id: string; entity_id: string; name: string; protocol: "saml" | "oidc" | "did:web" | "ldap" | "social";
  role: "idp" | "sp" | "both"; trust_level: "verified" | "trusted" | "provisional";
  cert_expires: string | null; connected: boolean; metadata_url: string;
}

interface TransformRule {
  id: string; source_attr: string; target_attr: string; transform: "direct" | "rename" | "concat" | "split" | "regex" | "constant";
  config: string;
}

interface DiscoveryRule {
  id: string; domain: string; idp_id: string; idp_name: string; priority: number; enabled: boolean;
}

interface SSOTestResult {
  success: boolean; latency_ms: number; claims: Record<string, string>; error: string | null;
}

const protoConfig = {
  saml: { color: "text-blue-500", bg: "bg-blue-100 dark:bg-blue-900/30", icon: Shield },
  oidc: { color: "text-green-500", bg: "bg-green-100 dark:bg-green-900/30", icon: Globe },
  "did:web": { color: "text-purple-500", bg: "bg-purple-100 dark:bg-purple-900/30", icon: Network },
  ldap: { color: "text-orange-500", bg: "bg-orange-100 dark:bg-orange-900/30", icon: KeyRound },
  social: { color: "text-pink-500", bg: "bg-pink-100 dark:bg-pink-900/30", icon: Globe },
};

const trustColors: Record<string, string> = {
  verified: "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400",
  trusted: "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400",
  provisional: "bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400",
};

type Tab = "topology" | "trust" | "transform" | "discovery" | "tester" | "metadata";

export default function FederationHubPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("topology");
  const [trustRels, setTrustRels] = useState<TrustRelation[]>([]);
  const [transforms, setTransforms] = useState<TransformRule[]>([]);
  const [rules, setRules] = useState<DiscoveryRule[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  // Trust CRUD
  const [showTrustForm, setShowTrustForm] = useState(false);
  const [trName, setTrName] = useState("");
  const [trEntity, setTrEntity] = useState("");
  const [trProto, setTrProto] = useState<TrustRelation["protocol"]>("saml");
  const [trMetaUrl, setTrMetaUrl] = useState("");
  const [saving, setSaving] = useState(false);
  // SSO tester
  const [testIdp, setTestIdp] = useState("");
  const [testResult, setTestResult] = useState<SSOTestResult | null>(null);
  const [testing, setTesting] = useState(false);

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
      const [trRes, tfRes, drRes] = await Promise.all([
        fetch("/api/v1/identity/federation/trust-relations", { headers: h }).catch(() => null),
        fetch("/api/v1/identity/federation/transforms", { headers: h }).catch(() => null),
        fetch("/api/v1/identity/federation/discovery-rules", { headers: h }).catch(() => null),
      ]);
      if (trRes?.ok) { const d = await trRes.json(); setTrustRels(d.relations || d.items || []); }
      if (tfRes?.ok) { const d = await tfRes.json(); setTransforms(d.transforms || d.items || []); }
      if (drRes?.ok) { const d = await drRes.json(); setRules(d.rules || d.items || []); }
    } catch { setError("Failed to load federation data"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const saveTrust = async () => {
    if (!trName || !trEntity) return;
    setSaving(true);
    try {
      await fetch("/api/v1/identity/federation/trust-relations", {
        method: "POST",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify({ name: trName, entity_id: trEntity, protocol: trProto, metadata_url: trMetaUrl, role: "idp", trust_level: "provisional" }),
      });
      setShowTrustForm(false); setTrName(""); setTrEntity(""); setTrMetaUrl(""); loadData();
    } catch { setError("Failed to save trust relation"); }
    finally { setSaving(false); }
  };

  const runSSOTest = async () => {
    if (!testIdp) return;
    setTesting(true); setTestResult(null);
    try {
      const res = await fetch(`/api/v1/identity/federation/${testIdp}/test-sso`, {
        method: "POST", headers: { ...authHeader(), "X-Tenant-ID": TENANT_ID },
      });
      if (res.ok) setTestResult(await res.json());
      else setTestResult({ success: false, latency_ms: 0, claims: {}, error: "SSO test failed" });
    } catch { setTestResult({ success: false, latency_ms: 0, claims: {}, error: "Network error" }); }
    finally { setTesting(false); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  // Topology layout: GGID center, IdPs around
  const centerX = 300, centerY = 200, radius = 140;
  const positions = trustRels.map((_, i) => {
    const angle = (i / Math.max(trustRels.length, 1)) * 2 * Math.PI - Math.PI / 2;
    return { x: centerX + radius * Math.cos(angle), y: centerY + radius * Math.sin(angle) };
  });

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Network className="h-6 w-6 text-indigo-500" /> Federation Hub</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Trust topology, assertion transforms, IdP discovery, and SSO testing.</p>
        </div>
        <button onClick={loadData} disabled={loading} aria-label="Refresh" className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800"><RefreshCw className={"h-4 w-4 " + (loading ? "animate-spin" : "")} /> Refresh</button>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "topology" as Tab, label: "Trust Topology", icon: Network },
          { id: "trust" as Tab, label: "Trust Registry", icon: Shield },
          { id: "transform" as Tab, label: "Assertion Transform", icon: ArrowRight },
          { id: "discovery" as Tab, label: "Discovery", icon: Mail },
          { id: "tester" as Tab, label: "SSO Tester", icon: TestTube },
          { id: "metadata" as Tab, label: "Metadata", icon: FileJson },
        ]).map(tb => { const Icon = tb.icon; return (
          <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id} className={"flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap " + (tab === tb.id ? "border-indigo-600 text-indigo-600 dark:text-indigo-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300")}><Icon className="h-4 w-4" /> {tb.label}</button>
        ); })}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-500" /></div> : (<>

      {/* TOPOLOGY */}
      {tab === "topology" && (
        <div className={cardCls}>
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Network className="h-4 w-4" /> Trust Topology</h2>
          <div className="overflow-x-auto"><svg width="600" height="400" viewBox="0 0 600 400" className="mx-auto" aria-label="Federation trust topology">
            {/* Edges */}
            {trustRels.map((tr: any, i: number) => {
              const pos = positions[i]; if (!pos) return null;
              const color = tr.connected ? "#10b981" : "#ef4444";
              return <line key={i} x1={centerX} y1={centerY} x2={pos.x} y2={pos.y} stroke={color} strokeWidth={tr.connected ? 2 : 1} strokeDasharray={tr.connected ? "0" : "5"} opacity={0.7} />;
            })}
            {/* GGID center */}
            <circle cx={centerX} cy={centerY} r={35} fill="#6366f1" />
            <text x={centerX} y={centerY + 4} textAnchor="middle" fontSize={11} fontWeight="bold" fill="white">GGID</text>
            {/* IdP nodes */}
            {trustRels.map((tr: any, i: number) => {
              const pos = positions[i]; if (!pos) return null;
              const cfg = protoConfig[tr.protocol] || protoConfig.saml;
              return (
                <g key={i} onClick={() => setTab("trust")} className="cursor-pointer">
                  <circle cx={pos.x} cy={pos.y} r={22} fill={tr.connected ? "#10b981" : "#6b7280"} opacity={0.8} />
                  <text x={pos.x} y={pos.y + 3} textAnchor="middle" fontSize={8} fontWeight="bold" fill="white">{tr.name.substring(0, 6)}</text>
                  <text x={pos.x} y={pos.y + 35} textAnchor="middle" fontSize={7} fill="#6b7280">{tr.protocol}</text>
                  {tr.connected && <circle cx={pos.x + 16} cy={pos.y - 16} r={4} fill="#10b981" />}
                </g>
              );
            })}
          </svg></div>
          {trustRels.length === 0 && <p className="text-center text-sm text-gray-400 mt-2">No trust relations configured. Add IdPs in the Trust Registry tab.</p>}
        </div>
      )}

      {/* TRUST REGISTRY */}
      {tab === "trust" && (
        <>
          <div className="flex justify-end"><button onClick={() => setShowTrustForm(true)} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-700"><Plus className="h-4 w-4" /> Add Trust Relation</button></div>
          {trustRels.length === 0 ? <div className={cardCls}><div className="py-8 text-center"><Shield className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No trust relations.</p></div></div> : (
            <div className="space-y-2">{trustRels.map(tr => { const cfg = protoConfig[tr.protocol] || protoConfig.saml; return (
              <div key={tr.id} className={cardCls}>
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <div className={"h-2.5 w-2.5 rounded-full " + (tr.connected ? "bg-green-500 animate-pulse" : "bg-gray-400")} />
                    <div><div className="flex items-center gap-2"><span className="font-medium text-sm">{tr.name}</span><span className={"px-1.5 py-0.5 rounded text-xs font-medium " + cfg.bg + " " + cfg.color}>{tr.protocol}</span><span className={"px-1.5 py-0.5 rounded text-xs " + (trustColors[tr.trust_level] || "")}>{tr.trust_level}</span></div><p className="text-xs font-mono text-gray-400 mt-0.5">{tr.entity_id}</p>{tr.cert_expires && <p className="text-xs text-gray-400">Cert expires: {new Date(tr.cert_expires).toLocaleDateString()}</p>}</div>
                  </div>
                  <button className="rounded p-1 text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20"><Trash2 className="h-4 w-4" /></button>
                </div>
              </div>
            ); })}</div>
          )}
        </>
      )}

      {/* ASSERTION TRANSFORM */}
      {tab === "transform" && (
        <div className={cardCls}>
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><ArrowRight className="h-4 w-4" /> Assertion Transformation DSL</h2>
          {transforms.length === 0 ? <div className="py-8 text-center"><ArrowRight className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No transform rules defined.</p></div> : (
            <div className="space-y-2">{transforms.map(tf => (
              <div key={tf.id} className="flex items-center gap-3 rounded-lg border p-3 dark:border-gray-700">
                <span className="font-mono text-xs text-blue-600 dark:text-blue-400 flex-1">{tf.source_attr}</span>
                <span className="px-1.5 py-0.5 rounded text-xs bg-indigo-100 dark:bg-indigo-900/30 font-mono">{tf.transform}</span>
                <ArrowRight className="h-3 w-3 text-gray-400" />
                <span className="font-mono text-xs text-green-600 dark:text-green-400 flex-1">{tf.target_attr}</span>
                {tf.config && <span className="text-xs text-gray-400 font-mono">{tf.config}</span>}
              </div>
            ))}</div>
          )}
        </div>
      )}

      {/* DISCOVERY */}
      {tab === "discovery" && (
        <div className={cardCls}>
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Mail className="h-4 w-4" /> IdP Discovery Rules (WAYF)</h2>
          <p className="text-sm text-gray-500 mb-4">Email domain → IdP automatic routing. Users enter email, auto-redirected to correct IdP.</p>
          {rules.length === 0 ? <div className="py-8 text-center"><Mail className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No discovery rules. Users will see the IdP picker.</p></div> : (
            <div className="space-y-2">{rules.map(r => (
              <div key={r.id} className="flex items-center gap-3 rounded-lg border p-3 dark:border-gray-700">
                <Mail className="h-4 w-4 text-gray-400" />
                <span className="font-mono text-sm flex-1">@{r.domain}</span>
                <ArrowRight className="h-3 w-3 text-gray-400" />
                <span className="text-sm font-medium flex-1">{r.idp_name}</span>
                <span className="text-xs text-gray-400">P{r.priority}</span>
                <span className={"px-1.5 py-0.5 rounded text-xs " + (r.enabled ? "bg-green-100 dark:bg-green-900/30 text-green-600" : "bg-gray-100 dark:bg-gray-800 text-gray-400")}>{r.enabled ? "Active" : "Off"}</span>
              </div>
            ))}</div>
          )}
        </div>
      )}

      {/* SSO TESTER */}
      {tab === "tester" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><TestTube className="h-4 w-4" /> SSO Login Simulator</h2>
            <div><label className="text-sm font-medium">Identity Provider</label><select aria-label="Select IdP" value={testIdp} onChange={e => setTestIdp(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm"><option value="">Select IdP...</option>{trustRels.map(tr => <option key={tr.id} value={tr.id}>{tr.name} ({tr.protocol})</option>)}</select></div>
            <button onClick={runSSOTest} disabled={!testIdp || testing} className="mt-4 flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{testing ? <Loader2 className="h-4 w-4 animate-spin" /> : <TestTube className="h-4 w-4" />} Simulate SSO Login</button>
          </div>
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Eye className="h-4 w-4" /> Result</h2>
            {testResult ? (
              <div><div className={"flex items-center gap-3 rounded-xl border-2 p-4 " + (testResult.success ? "border-green-300 bg-green-50 dark:border-green-700 dark:bg-green-950/30" : "border-red-300 bg-red-50 dark:border-red-700 dark:bg-red-950/30")}>{testResult.success ? <CheckCircle className="h-8 w-8 text-green-500" /> : <XCircle className="h-8 w-8 text-red-500" />}<div><p className={"text-lg font-bold " + (testResult.success ? "text-green-700 dark:text-green-400" : "text-red-700 dark:text-red-400")}>{testResult.success ? "SSO SUCCESS" : "SSO FAILED"}</p>{testResult.success && <p className="text-xs text-gray-500">Latency: {testResult.latency_ms}ms</p>}{testResult.error && <p className="text-xs text-red-500">{testResult.error}</p>}</div></div>
              {testResult.success && Object.keys(testResult.claims).length > 0 && <div className="mt-3"><p className="text-xs font-semibold text-gray-400 mb-1">Claims Received:</p><div className="flex flex-wrap gap-1">{Object.entries(testResult.claims).map(([k, v]) => <span key={k} className="px-1.5 py-0.5 rounded bg-blue-50 dark:bg-blue-950/30 text-xs font-mono">{k}={v}</span>)}</div></div>}</div>
            ) : <div className="py-8 text-center"><TestTube className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">Select an IdP and simulate.</p></div>}
          </div>
        </div>
      )}

      {/* METADATA */}
      {tab === "metadata" && (
        <div className={cardCls}>
          <div className="mb-4 flex items-center justify-between"><h2 className="flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><FileJson className="h-4 w-4" /> Federation Metadata Aggregate</h2><button className="flex items-center gap-1 rounded-lg border border-gray-300 px-3 py-1.5 text-xs dark:border-gray-700"><Download className="h-3 w-3" /> Download</button></div>
          <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-xs text-green-400 font-mono max-h-96 overflow-y-auto">{JSON.stringify({
            entity_id: "https://ggid.dev",
            endpoints: { sso: "https://ggid.dev/api/v1/auth/saml/sso", slo: "https://ggid.dev/api/v1/auth/saml/slo", oidc_discovery: "https://ggid.dev/.well-known/openid-configuration", did_web: "https://ggid.dev/.well-known/did.json" },
            certificates: [{ use: "signing", fingerprint: "AB:CD:EF:00:11:22...", expires: "2026-12-31" }],
            trusted_idps: trustRels.map(tr => ({ entity_id: tr.entity_id, protocol: tr.protocol, status: tr.connected ? "connected" : "disconnected" })),
          }, null, 2)}</pre>
        </div>
      )}

      </>)}

      {/* Trust form dialog */}
      {showTrustForm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowTrustForm(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <h3 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white"><Plus className="h-5 w-5 text-indigo-500" /> Add Trust Relation</h3>
            <div className="mt-4 space-y-3">
              <div><label className="text-sm font-medium">Name</label><input aria-label="Name" type="text" value={trName} onChange={e => setTrName(e.target.value)} placeholder="Corporate Azure AD" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" autoFocus /></div>
              <div><label className="text-sm font-medium">Entity ID</label><input aria-label="Entity ID" type="text" value={trEntity} onChange={e => setTrEntity(e.target.value)} placeholder="https://sts.windows.net/tenant-id/" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div>
              <div><label className="text-sm font-medium">Protocol</label><select aria-label="Protocol" value={trProto} onChange={e => setTrProto(e.target.value as TrustRelation["protocol"])} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm"><option value="saml">SAML 2.0</option><option value="oidc">OIDC</option><option value="did:web">DID:web</option><option value="ldap">LDAP</option><option value="social">Social (OAuth)</option></select></div>
              <div><label className="text-sm font-medium">Metadata URL</label><input aria-label="Metadata URL" type="text" value={trMetaUrl} onChange={e => setTrMetaUrl(e.target.value)} placeholder="https://...FederationMetadata.xml" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div>
            </div>
            <div className="mt-4 flex justify-end gap-2"><button onClick={() => setShowTrustForm(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">Cancel</button><button onClick={saveTrust} disabled={!trName || !trEntity || saving} className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{saving ? <Loader2 className="h-4 w-4 animate-spin" /> : "Save"}</button></div>
          </div>
        </div>
      )}
    </div>
  );
}

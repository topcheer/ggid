"use client";

import { useState, useCallback, useEffect } from "react";
import {
  Cloud, Shield, Loader2, AlertCircle, X, RefreshCw, Plus, Trash2,
  Check, CheckCircle, XCircle, ArrowRight, Download, Copy, Code,
  ChevronRight, ChevronLeft, Zap, Eye, Activity, Settings,
  CloudOff, FileJson, TestTube,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface FederationConfig {
  id: string;
  platform: "aws" | "azure" | "gcp";
  name: string;
  protocol: "saml" | "oidc";
  status: "connected" | "error" | "disconnected";
  metadata_url: string;
  acs_url: string;
  certificate_fingerprint: string;
  last_sync: string | null;
  error_count_24h: number;
  claim_mappings: { ggid_attr: string; cloud_claim: string }[];
  role_mappings: { ggid_role: string; cloud_role: string; cloud_role_arn?: string }[];
}

interface TestResult {
  success: boolean;
  claims_received: Record<string, string>;
  roles_assigned: string[];
  error: string | null;
}

interface HealthStatus {
  platform: string;
  connected: boolean;
  last_sync: string | null;
  latency_ms: number;
  errors_24h: number;
  warnings: string[];
}

const platformConfig = {
  aws: { name: "AWS IAM Identity Center", icon: Cloud, color: "text-orange-500", bg: "bg-orange-100 dark:bg-orange-900/30", ggid_attrs: ["email", "username", "groups", "department"], cloud_claims: ["Email", "UserName", "Groups", "Department"], role_prefix: "arn:aws:iam::ACCOUNT:role/", tf_provider: "aws" },
  azure: { name: "Azure AD", icon: Cloud, color: "text-blue-500", bg: "bg-blue-100 dark:bg-blue-900/30", ggid_attrs: ["email", "username", "display_name", "groups", "department"], cloud_claims: ["mail", "userPrincipalName", "displayName", "groups", "department"], role_prefix: "", tf_provider: "azurerm" },
  gcp: { name: "Google Cloud IAM", icon: Cloud, color: "text-green-500", bg: "bg-green-100 dark:bg-green-900/30", ggid_attrs: ["email", "username", "groups"], cloud_claims: ["email", "name", "groups"], role_prefix: "roles/", tf_provider: "google" },
};

const WIZARD_STEPS = ["Platform", "Protocol", "Claim Mapping", "Role Mapping", "SAML/OIDC", "Test", "Terraform"];

export default function CloudFederationPage() {
  const t = useTranslations();
  const [federations, setFederations] = useState<FederationConfig[]>([]);
  const [health, setHealth] = useState<HealthStatus[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  // Wizard
  const [showWizard, setShowWizard] = useState(false);
  const [wizStep, setWizStep] = useState(0);
  const [wizPlatform, setWizPlatform] = useState<"aws" | "azure" | "gcp">("aws");
  const [wizProtocol, setWizProtocol] = useState<"saml" | "oidc">("saml");
  const [wizName, setWizName] = useState("");
  const [wizClaims, setWizClaims] = useState<{ ggid_attr: string; cloud_claim: string }[]>([]);
  const [wizRoles, setWizRoles] = useState<{ ggid_role: string; cloud_role: string }[]>([]);
  const [wizSubmitting, setWizSubmitting] = useState(false);
  // Test
  const [testUser, setTestUser] = useState("");
  const [testResult, setTestResult] = useState<TestResult | null>(null);
  const [testing, setTesting] = useState(false);
  // TF
  const [tfCopied, setTfCopied] = useState(false);
  // Actions
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
      const [fedRes, healthRes] = await Promise.all([
        fetch("/api/v1/identity/cloud-federation", { headers: h }).catch(() => null),
        fetch("/api/v1/identity/cloud-federation/health", { headers: h }).catch(() => null),
      ]);
      if (fedRes?.ok) { const d = await fedRes.json(); setFederations(d.federations || d.items || []); }
      if (healthRes?.ok) { const d = await healthRes.json(); setHealth(d.status || d.items || []); }
    } catch { setError("Failed to load federation data"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const submitWizard = async () => {
    setWizSubmitting(true);
    try {
      await fetch("/api/v1/identity/cloud-federation", {
        method: "POST",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify({ platform: wizPlatform, protocol: wizProtocol, name: wizName, claim_mappings: wizClaims, role_mappings: wizRoles }),
      });
      setShowWizard(false); loadData();
    } catch { setError("Failed to create federation"); }
    finally { setWizSubmitting(false); }
  };

  const runTest = async (fedId: string) => {
    if (!testUser) return;
    setTesting(true);
    setTestResult(null);
    try {
      const res = await fetch(`/api/v1/identity/cloud-federation/${fedId}/test-login`, {
        method: "POST",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify({ user_id: testUser }),
      });
      if (res.ok) setTestResult(await res.json());
      else setTestResult({ success: false, claims_received: {}, roles_assigned: [], error: "Test failed" });
    } catch { setTestResult({ success: false, claims_received: {}, roles_assigned: [], error: "Network error" }); }
    finally { setTesting(false); }
  };

  const deleteFed = async (id: string) => {
    setDeletingId(id);
    try {
      await fetch(`/api/v1/identity/cloud-federation/${id}`, { method: "DELETE", headers: { ...authHeader(), "X-Tenant-ID": TENANT_ID } });
      setFederations(prev => prev.filter(f => f.id !== id));
    } catch { setError("Failed to delete"); }
    finally { setDeletingId(null); }
  };

  const generateTF = (fed: FederationConfig | undefined): string => {
    if (!fed) return "# Select a federation to generate Terraform";
    const cfg = platformConfig[fed.platform];
    if (fed.protocol === "saml") {
      return `# ${cfg.name} SAML Federation — generated by GGID
terraform {
  required_providers {
    ${cfg.tf_provider} = { source = "${fed.platform}/${cfg.tf_provider}" }
  }
}

# SAML provider
resource "${cfg.tf_provider}_iam_saml_provider" "ggid" {
  name                   = "GGID-${fed.name}"
  saml_metadata_document = data.http.ggid_metadata.body
}

data "http" "ggid_metadata" {
  url = "${fed.metadata_url}"
}

${fed.role_mappings.map(rm => `resource "${cfg.tf_provider}_iam_role" "${rm.ggid_role}" {
  name = "${rm.cloud_role}"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = { SAML = "${cfg.tf_provider}_iam_saml_provider.ggid.arn" }
      Action = "sts:AssumeRoleWithSAML"
      Condition = {
        StringEquals = { "SAML:aud" = "${fed.acs_url}" }
      }
    }]
  })
}`).join("\n\n")}
`;
    }
    return `# ${cfg.name} OIDC Federation — generated by GGID
resource "${cfg.tf_provider}_iam_openid_connect_provider" "ggid" {
  url             = "${fed.metadata_url}"
  client_id_list  = ["${fed.acs_url}"]
  thumbprint_list = ["${fed.certificate_fingerprint || "0000000000000000000000000000000000000000"}"]
}
`;
  };

  const copyTF = async (fed: FederationConfig | undefined) => {
    await navigator.clipboard.writeText(generateTF(fed));
    setTfCopied(true); setTimeout(() => setTfCopied(false), 3000);
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Cloud className="h-6 w-6 text-indigo-500" /> Cloud IAM Federation</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Configure SAML/OIDC federation with AWS IAM Identity Center, Azure AD, and GCP IAM.</p>
        </div>
        <div className="flex items-center gap-2">
          <button onClick={() => { setWizStep(0); setWizName(""); setShowWizard(true); }} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-700"><Plus className="h-4 w-4" /> Add Federation</button>
          <button onClick={loadData} disabled={loading} aria-label="Refresh" className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800"><RefreshCw className={"h-4 w-4 " + (loading ? "animate-spin" : "")} /></button>
        </div>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {/* Health monitoring bar */}
      {health.length > 0 && (
        <div className="flex flex-wrap gap-3">
          {health.map((h: any, i: number) => {
            const cfg = platformConfig[h.platform as keyof typeof platformConfig] || platformConfig.aws;
            return (
              <div key={i} className={"flex items-center gap-3 rounded-lg border px-4 py-3 " + (h.connected ? "border-green-200 bg-green-50 dark:border-green-700 dark:bg-green-950/20" : "border-red-200 bg-red-50 dark:border-red-700 dark:bg-red-950/20")}>
                <div className={"h-2.5 w-2.5 rounded-full " + (h.connected ? "bg-green-500 animate-pulse" : "bg-red-500")} />
                <div>
                  <div className="flex items-center gap-2"><span className="text-sm font-medium">{cfg.name}</span>{h.errors_24h > 0 && <span className="px-1.5 py-0.5 rounded text-xs bg-red-100 text-red-600 dark:bg-red-900/30">{h.errors_24h} errors</span>}</div>
                  <p className="text-xs text-gray-400">{h.connected ? `${h.latency_ms}ms latency` : "Disconnected"} · {h.last_sync ? `Synced ${new Date(h.last_sync).toLocaleTimeString()}` : "Never synced"}</p>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-500" /></div> : federations.length === 0 ? (
        <div className={cardCls}><div className="py-12 text-center"><Cloud className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No cloud federations configured.</p><button onClick={() => setShowWizard(true)} className="mt-3 text-sm text-indigo-600 hover:underline">Set up your first federation</button></div></div>
      ) : (
        <div className="space-y-4">
          {federations.map(fed => {
            const cfg = platformConfig[fed.platform];
            return (
              <div key={fed.id} className={cardCls}>
                {/* Federation header */}
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-3">
                    <div className={"h-10 w-10 rounded-lg flex items-center justify-center " + cfg.bg}><cfg.icon className={"h-5 w-5 " + cfg.color} /></div>
                    <div>
                      <div className="flex items-center gap-2">
                        <h3 className="font-semibold text-gray-900 dark:text-white">{fed.name}</h3>
                        <span className="px-1.5 py-0.5 rounded text-xs font-mono bg-gray-100 dark:bg-gray-700">{fed.protocol.toUpperCase()}</span>
                        <span className={"h-2 w-2 rounded-full " + (fed.status === "connected" ? "bg-green-500" : "bg-red-500")} />
                        <span className="text-xs text-gray-400">{fed.status}</span>
                      </div>
                      <p className="text-xs text-gray-400 font-mono mt-0.5 truncate max-w-md">{fed.metadata_url}</p>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <button onClick={() => { setTestResult(null); }} className="rounded-lg border border-gray-300 p-1.5 text-gray-400 hover:bg-gray-100 dark:border-gray-700 dark:hover:bg-gray-800" aria-label="Test"><TestTube className="h-4 w-4" /></button>
                    <button onClick={() => copyTF(fed)} className="rounded-lg border border-gray-300 p-1.5 text-gray-400 hover:bg-gray-100 dark:border-gray-700 dark:hover:bg-gray-800" aria-label="Copy Terraform">{tfCopied ? <Check className="h-4 w-4 text-green-500" /> : <Copy className="h-4 w-4" />}</button>
                    <button onClick={() => deleteFed(fed.id)} disabled={deletingId === fed.id} className="rounded-lg p-1.5 text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20" aria-label="Delete">{deletingId === fed.id ? <Loader2 className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4" />}</button>
                  </div>
                </div>

                <div className="mt-4 grid grid-cols-1 gap-4 lg:grid-cols-2">
                  {/* Claim mappings */}
                  <div>
                    <h4 className="text-xs font-semibold uppercase text-gray-400 mb-2">Claim Mapping</h4>
                    <div className="space-y-1">{fed.claim_mappings.map((cm: any, i: number) => (
                      <div key={i} className="flex items-center gap-2 text-xs">
                        <span className="font-mono text-blue-600 dark:text-blue-400">{cm.ggid_attr}</span>
                        <ArrowRight className="h-3 w-3 text-gray-400" />
                        <span className="font-mono text-green-600 dark:text-green-400">{cm.cloud_claim}</span>
                      </div>
                    ))}</div>
                  </div>
                  {/* Role mappings */}
                  <div>
                    <h4 className="text-xs font-semibold uppercase text-gray-400 mb-2">Role Mapping</h4>
                    <div className="space-y-1">{fed.role_mappings.map((rm: any, i: number) => (
                      <div key={i} className="flex items-center gap-2 text-xs">
                        <span className="font-mono text-purple-600 dark:text-purple-400">{rm.ggid_role}</span>
                        <ArrowRight className="h-3 w-3 text-gray-400" />
                        <span className="font-mono text-orange-600 dark:text-orange-400 truncate">{rm.cloud_role_arn || rm.cloud_role}</span>
                      </div>
                    ))}</div>
                  </div>
                </div>

                {/* SAML/OIDC config */}
                <div className="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-3">
                  <div className="rounded-lg border p-3 dark:border-gray-700"><span className="text-xs text-gray-400">ACS URL</span><p className="text-xs font-mono mt-1 break-all">{fed.acs_url || "—"}</p></div>
                  <div className="rounded-lg border p-3 dark:border-gray-700"><span className="text-xs text-gray-400">Certificate</span><p className="text-xs font-mono mt-1 truncate">{fed.certificate_fingerprint ? fed.certificate_fingerprint.substring(0, 40) + "..." : "—"}</p></div>
                  <div className="rounded-lg border p-3 dark:border-gray-700"><span className="text-xs text-gray-400">Last Sync</span><p className="text-xs mt-1">{fed.last_sync ? new Date(fed.last_sync).toLocaleString() : "Never"}</p></div>
                </div>

                {/* Test connection */}
                <div className="mt-4 rounded-lg border p-3 dark:border-gray-700">
                  <div className="flex items-end gap-2">
                    <div className="flex-1"><label className="text-xs font-medium text-gray-500">Test user ID</label><input aria-label="Test user" type="text" value={testUser} onChange={e => setTestUser(e.target.value)} placeholder="user:alice" className="mt-1 w-full rounded border dark:border-gray-700 dark:bg-gray-900 px-2 py-1 text-xs font-mono" /></div>
                    <button onClick={() => runTest(fed.id)} disabled={!testUser || testing} className="flex items-center gap-1 rounded-lg bg-indigo-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{testing ? <Loader2 className="h-3 w-3 animate-spin" /> : <TestTube className="h-3 w-3" />} Test Login</button>
                  </div>
                  {testResult && (
                    <div className="mt-3">
                      <div className={"flex items-center gap-2 rounded-lg p-2 " + (testResult.success ? "bg-green-50 dark:bg-green-950/20" : "bg-red-50 dark:bg-red-950/20")}>
                        {testResult.success ? <CheckCircle className="h-4 w-4 text-green-500" /> : <XCircle className="h-4 w-4 text-red-500" />}
                        <span className="text-xs font-medium">{testResult.success ? "Login simulation successful" : "Login failed"}</span>
                        {testResult.error && <span className="text-xs text-red-500">{testResult.error}</span>}
                      </div>
                      {testResult.success && Object.keys(testResult.claims_received).length > 0 && (
                        <div className="mt-2"><p className="text-xs font-semibold text-gray-400 mb-1">Claims Received:</p><div className="flex flex-wrap gap-1">{Object.entries(testResult.claims_received).map(([k, v]: any[]) => <span key={k} className="px-1.5 py-0.5 rounded bg-blue-50 dark:bg-blue-950/30 text-xs font-mono">{k}={v}</span>)}</div></div>
                      )}
                      {testResult.roles_assigned?.length > 0 && (
                        <div className="mt-2"><p className="text-xs font-semibold text-gray-400 mb-1">Roles Assigned:</p><div className="flex flex-wrap gap-1">{testResult.roles_assigned.map(r => <span key={r} className="px-1.5 py-0.5 rounded bg-orange-50 dark:bg-orange-950/30 text-xs font-mono">{r}</span>)}</div></div>
                      )}
                    </div>
                  )}
                </div>

                {/* Terraform preview */}
                <details className="mt-3">
                  <summary className="flex items-center gap-1 text-xs text-indigo-600 hover:underline cursor-pointer"><Code className="h-3 w-3" /> Terraform Configuration</summary>
                  <pre className="mt-2 overflow-x-auto rounded-lg bg-gray-900 p-3 text-xs text-green-400 font-mono max-h-48 overflow-y-auto">{generateTF(fed)}</pre>
                </details>
              </div>
            );
          })}
        </div>
      )}

      {/* WIZARD */}
      {showWizard && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowWizard(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 max-h-[90vh] w-full max-w-2xl overflow-y-auto rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <div className="mb-6 flex items-center justify-between"><h3 className="text-lg font-semibold text-gray-900 dark:text-white">Cloud Federation Setup</h3><button onClick={() => setShowWizard(false)}><X className="h-5 w-5 text-gray-400" /></button></div>
            {/* Steps */}
            <div className="mb-6 flex items-center gap-1">{WIZARD_STEPS.map((s: any, i: number) => (
              <div key={i} className="flex items-center gap-1 flex-1"><div className={"flex h-7 w-7 items-center justify-center rounded-full text-xs font-bold " + (i < wizStep ? "bg-green-600 text-white" : i === wizStep ? "bg-indigo-600 text-white" : "bg-gray-200 dark:bg-gray-700 text-gray-400")}>{i < wizStep ? <Check className="h-3.5 w-3.5" /> : i + 1}</div>{i < WIZARD_STEPS.length - 1 && <div className={"h-0.5 flex-1 " + (i < wizStep ? "bg-green-600" : "bg-gray-200 dark:bg-gray-700")} />}</div>
            ))}</div>
            <div className="min-h-[180px]">
              {wizStep === 0 && <div className="space-y-3">
                <div><label className="text-sm font-medium">Federation Name</label><input aria-label="Name" type="text" value={wizName} onChange={e => setWizName(e.target.value)} placeholder="Production AWS" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" autoFocus /></div>
                <div className="grid grid-cols-3 gap-3">{Object.entries(platformConfig).map(([key, cfg]: any[]) => (
                  <button key={key} onClick={() => { setWizPlatform(key as "aws" | "azure" | "gcp"); setWizClaims(cfg.ggid_attrs.map((a: any, i: number) => ({ ggid_attr: a, cloud_claim: cfg.cloud_claims[i] }))); }} aria-pressed={wizPlatform === key} className={"rounded-xl border-2 p-4 text-center transition " + (wizPlatform === key ? "border-indigo-500 bg-indigo-50 dark:bg-indigo-950/30" : "border-gray-200 dark:border-gray-700 hover:border-gray-300")}><cfg.icon className={"h-8 w-8 mx-auto " + cfg.color} /><p className="mt-2 text-xs font-medium">{cfg.name}</p></button>
                ))}</div>
              </div>}
              {wizStep === 1 && <div className="space-y-3">
                <div className="grid grid-cols-2 gap-3">
                  <button onClick={() => setWizProtocol("saml")} aria-pressed={wizProtocol === "saml"} className={"rounded-xl border-2 p-4 text-left " + (wizProtocol === "saml" ? "border-indigo-500" : "border-gray-200 dark:border-gray-700")}><Shield className="h-6 w-6 text-indigo-500" /><p className="mt-2 font-medium text-sm">SAML 2.0</p><p className="text-xs text-gray-400">For AWS IAM Identity Center, Azure AD enterprise apps</p></button>
                  <button onClick={() => setWizProtocol("oidc")} aria-pressed={wizProtocol === "oidc"} className={"rounded-xl border-2 p-4 text-left " + (wizProtocol === "oidc" ? "border-indigo-500" : "border-gray-200 dark:border-gray-700")}><Cloud className="h-6 w-6 text-green-500" /><p className="mt-2 font-medium text-sm">OIDC</p><p className="text-xs text-gray-400">For GCP workload identity, modern AWS</p></button>
                </div>
              </div>}
              {wizStep === 2 && <div className="space-y-2">{wizClaims.map((c: any, i: number) => (
                <div key={i} className="flex items-center gap-2 rounded-lg border p-2 dark:border-gray-700"><span className="font-mono text-xs text-blue-600 flex-1">{c.ggid_attr}</span><ArrowRight className="h-3 w-3 text-gray-400" /><input aria-label={`Cloud claim ${i+1}`} type="text" value={c.cloud_claim} onChange={e => { const n = [...wizClaims]; n[i] = { ...c, cloud_claim: e.target.value }; setWizClaims(n); }} className="flex-1 rounded border dark:border-gray-700 dark:bg-gray-900 px-2 py-1 text-xs font-mono" /></div>
              ))}<button onClick={() => setWizClaims([...wizClaims, { ggid_attr: "custom_attr", cloud_claim: "customClaim" }])} className="flex items-center gap-1 text-xs text-indigo-600 hover:underline"><Plus className="h-3 w-3" /> Add Mapping</button></div>}
              {wizStep === 3 && <div className="space-y-2">{(wizRoles.length ? wizRoles : [{ ggid_role: "admin", cloud_role: "Administrator" }, { ggid_role: "user", cloud_role: "Developer" }]).map((r: any, i: number) => (
                <div key={i} className="flex items-center gap-2 rounded-lg border p-2 dark:border-gray-700"><input aria-label={`GGID role ${i+1}`} type="text" value={r.ggid_role} onChange={e => { const n = [...(wizRoles.length ? wizRoles : [{ ggid_role: "admin", cloud_role: "Administrator" }, { ggid_role: "user", cloud_role: "Developer" }])]; n[i] = { ...r, ggid_role: e.target.value }; setWizRoles(n); }} className="w-32 rounded border dark:border-gray-700 dark:bg-gray-900 px-2 py-1 text-xs font-mono" /><ArrowRight className="h-3 w-3 text-gray-400" /><input aria-label={`Cloud role ${i+1}`} type="text" value={r.cloud_role} onChange={e => { const n = [...(wizRoles.length ? wizRoles : [{ ggid_role: "admin", cloud_role: "Administrator" }, { ggid_role: "user", cloud_role: "Developer" }])]; n[i] = { ...r, cloud_role: e.target.value }; setWizRoles(n); }} className="flex-1 rounded border dark:border-gray-700 dark:bg-gray-900 px-2 py-1 text-xs font-mono" /></div>
              ))}<button onClick={() => setWizRoles([...(wizRoles.length ? wizRoles : [{ ggid_role: "admin", cloud_role: "Administrator" }, { ggid_role: "user", cloud_role: "Developer" }]), { ggid_role: "viewer", cloud_role: "Viewer" }])} className="flex items-center gap-1 text-xs text-indigo-600 hover:underline"><Plus className="h-3 w-3" /> Add Role Mapping</button></div>}
              {wizStep === 4 && <div className="space-y-3"><div className="rounded-lg bg-blue-50 p-3 dark:bg-blue-950/30"><p className="text-xs text-blue-700 dark:text-blue-400">Configure these in your cloud provider's identity console:</p></div><div className="space-y-2"><div><span className="text-xs text-gray-400">Metadata URL</span><p className="font-mono text-xs break-all">https://ggid.dev/.well-known/saml-metadata</p></div><div><span className="text-xs text-gray-400">ACS URL</span><p className="font-mono text-xs">https://ggid.dev/api/v1/auth/saml/acs</p></div><div><span className="text-xs text-gray-400">Entity ID</span><p className="font-mono text-xs">https://ggid.dev/api/v1/auth/saml</p></div></div></div>}
              {wizStep === 5 && <div className="text-center space-y-3"><TestTube className="h-10 w-10 text-indigo-400 mx-auto" /><p className="text-sm text-gray-500">Test connection will be available after creation.</p></div>}
              {wizStep === 6 && <div className="space-y-3"><p className="text-sm text-gray-500">Review and create:</p><pre className="overflow-x-auto rounded-lg bg-gray-900 p-3 text-xs text-green-400 font-mono max-h-32">{JSON.stringify({ platform: wizPlatform, protocol: wizProtocol, name: wizName, claims: wizClaims, roles: wizRoles }, null, 2)}</pre></div>}
            </div>
            {/* Nav */}
            <div className="mt-6 flex justify-between"><button onClick={() => setWizStep(Math.max(0, wizStep - 1))} disabled={wizStep === 0} className="flex items-center gap-1 rounded-lg border border-gray-300 px-4 py-2 text-sm disabled:opacity-30 dark:border-gray-700"><ChevronLeft className="h-4 w-4" /> Back</button>
            {wizStep < WIZARD_STEPS.length - 1 ? <button onClick={() => setWizStep(wizStep + 1)} disabled={wizStep === 0 && !wizName} className="flex items-center gap-1 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">Next <ChevronRight className="h-4 w-4" /></button>
            : <button onClick={submitWizard} disabled={wizSubmitting} className="flex items-center gap-1 rounded-lg bg-green-600 px-4 py-2 text-sm font-medium text-white hover:bg-green-700 disabled:opacity-50">{wizSubmitting ? <Loader2 className="h-4 w-4 animate-spin" /> : <CheckCircle className="h-4 w-4" />} Create Federation</button>}</div>
          </div>
        </div>
      )}
    </div>
  );
}

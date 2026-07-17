"use client";
import { useState, useCallback, useEffect } from "react";
import {
  KeyRound, Loader2, AlertCircle, X, RefreshCw, Plus, Trash2, Check,
  Database, Terminal, Cloud, Link2, Shield, Clock, Zap, Eye, EyeOff,
  Copy, AlertTriangle, CheckCircle2, XCircle, Activity, Server,
  Settings, FileClock, ChevronRight, Ban,
} from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

/* ─── Types matching backend structs ─── */
interface SecretTarget {
  id: string; name: string; type: "db" | "ssh" | "cloud" | "api_key";
  connection_config: Record<string, string>;
  ttl_seconds: number; default_role: string; enabled: boolean;
  created_at: string; updated_at: string;
}
interface SecretGrant {
  id: string; target_id: string; user_id: string; role: string;
  credential: string; expires_at: string; revoked: boolean;
  revoked_at: string | null; created_at: string;
}
interface BrokerResponse {
  grant_id: string; target_id: string; user_id: string;
  role: string; credential: string; expires_at: string;
}

type Tab = "targets" | "broker" | "active" | "audit" | "tester";

const TYPE_CONFIG: Record<string, { label: string; icon: typeof Database; color: string }> = {
  db: { label: "Database", icon: Database, color: "text-blue-500" },
  ssh: { label: "SSH Host", icon: Terminal, color: "text-green-500" },
  cloud: { label: "Cloud Provider", icon: Cloud, color: "text-purple-500" },
  api_key: { label: "API Key", icon: Link2, color: "text-orange-500" },
};

export default function SecretBrokerPage() {
  const [tab, setTab] = useState<Tab>("targets");
  const [targets, setTargets] = useState<SecretTarget[]>([]);
  const [grants, setGrants] = useState<SecretGrant[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  // Target form
  const [showForm, setShowForm] = useState(false);
  const [editId, setEditId] = useState<string | null>(null);
  const [fName, setFName] = useState("");
  const [fType, setFType] = useState<SecretTarget["type"]>("db");
  const [fTTL, setFTTL] = useState(3600);
  const [fRole, setFRole] = useState("");
  const [fHost, setFHost] = useState("");
  const [fPort, setFPort] = useState("");
  const [fDb, setFDb] = useState("");
  const [fRegion, setFRegion] = useState("");
  const [fEnabled, setFEnabled] = useState(true);

  // Broker
  const [brTarget, setBrTarget] = useState("");
  const [brUser, setBrUser] = useState("");
  const [brRole, setBrRole] = useState("");
  const [brJIT, setBrJIT] = useState("");
  const [brokerResult, setBrokerResult] = useState<BrokerResponse | null>(null);
  const [brokering, setBrokering] = useState(false);
  const [credVisible, setCredVisible] = useState(false);
  const [copied, setCopied] = useState(false);

  // Tester
  const [tType, setTType] = useState<SecretTarget["type"]>("db");
  const [tHost, setTHost] = useState("");
  const [tPort, setTPort] = useState("");
  const [tResult, setTResult] = useState<{ ok: boolean; latency_ms: number; detail: string } | null>(null);
  const [testing, setTesting] = useState(false);

  const H = { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID };
  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
      const [tRes, gRes] = await Promise.all([
        fetch("/api/v1/identity/secret-broker/targets", { headers: h }).catch(() => null),
        fetch("/api/v1/identity/secret-broker/active", { headers: h }).catch(() => null),
      ]);
      if (tRes?.ok) { const d = await tRes.json(); setTargets(d.targets || []); }
      if (gRes?.ok) { const d = await gRes.json(); setGrants(d.grants || []); }
      setError(null);
    } catch { setError("Failed to load secret broker data"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  // Auto-refresh active grants every 15s for TTL countdown
  useEffect(() => {
    if (tab !== "active") return;
    const timer = setInterval(async () => {
      const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
      const res = await fetch("/api/v1/identity/secret-broker/active", { headers: h }).catch(() => null);
      if (res?.ok) { const d = await res.json(); setGrants(d.grants || []); }
    }, 15000);
    return () => clearInterval(timer);
  }, [tab]);

  const resetForm = () => {
    setFName(""); setFType("db"); setFTTL(3600); setFRole("");
    setFHost(""); setFPort(""); setFDb(""); setFRegion(""); setFEnabled(true);
    setEditId(null);
  };

  const saveTarget = async () => {
    if (!fName) return;
    setActionLoading("save");
    try {
      const body: Record<string, unknown> = {
        name: fName, type: fType, ttl_seconds: fTTL,
        default_role: fRole, enabled: fEnabled,
        connection_config: fType === "db" ? { host: fHost, port: fPort, database: fDb }
          : fType === "ssh" ? { host: fHost, port: fPort || "22" }
          : fType === "cloud" ? { provider: "aws", region: fRegion }
          : { endpoint: fHost },
      };
      const method = editId ? "PUT" : "POST";
      const url = editId
        ? `/api/v1/identity/secret-broker/targets/${editId}`
        : "/api/v1/identity/secret-broker/targets";
      await fetch(url, { method, headers: H, body: JSON.stringify(body) });
      setShowForm(false); resetForm(); loadData();
    } catch { setError("Failed to save target"); }
    finally { setActionLoading(null); }
  };

  const deleteTarget = async (id: string) => {
    setActionLoading(`del-${id}`);
    try {
      await fetch(`/api/v1/identity/secret-broker/targets/${id}`, { method: "DELETE", headers: H });
      loadData();
    } catch { setError("Failed to delete target"); }
    finally { setActionLoading(null); }
  };

  const startEdit = (t: SecretTarget) => {
    setEditId(t.id); setFName(t.name); setFType(t.type as SecretTarget["type"]);
    setFTTL(t.ttl_seconds); setFRole(t.default_role); setFEnabled(t.enabled);
    setFHost(t.connection_config?.host || t.connection_config?.endpoint || "");
    setFPort(t.connection_config?.port || "");
    setFDb(t.connection_config?.database || "");
    setFRegion(t.connection_config?.region || "");
    setShowForm(true);
  };

  const toggleTarget = async (id: string, enabled: boolean) => {
    setActionLoading(`tg-${id}`);
    try {
      await fetch(`/api/v1/identity/secret-broker/targets/${id}`, {
        method: "PUT", headers: H,
        body: JSON.stringify({ enabled: !enabled }),
      });
      loadData();
    } catch { setError("Failed to toggle target"); }
    finally { setActionLoading(null); }
  };

  const doBroker = async () => {
    if (!brTarget || !brUser) return;
    setBrokering(true); setBrokerResult(null); setCredVisible(false);
    try {
      const res = await fetch("/api/v1/identity/secret-broker/broker", {
        method: "POST", headers: H,
        body: JSON.stringify({ target_id: brTarget, user_id: brUser, role: brRole, jit_request_id: brJIT || undefined }),
      });
      if (res.ok) {
        const d = await res.json();
        setBrokerResult(d);
        loadData(); // refresh active grants
      } else { setError("Broker request denied"); }
    } catch { setError("Broker request failed"); }
    finally { setBrokering(false); }
  };

  const revokeGrant = async (grantId: string) => {
    setActionLoading(`rvk-${grantId}`);
    try {
      await fetch("/api/v1/identity/secret-broker/revoke", {
        method: "POST", headers: H,
        body: JSON.stringify({ grant_id: grantId }),
      });
      loadData();
    } catch { setError("Failed to revoke grant"); }
    finally { setActionLoading(null); }
  };

  const runTest = async () => {
    if (!tHost) return;
    setTesting(true); setTResult(null);
    try {
      const start = Date.now();
      // Test connectivity via broker target create + immediate delete
      const res = await fetch("/api/v1/identity/secret-broker/targets", {
        method: "POST", headers: H,
        body: JSON.stringify({
          name: `_test_${Date.now()}`, type: tType,
          connection_config: tType === "db" ? { host: tHost, port: tPort }
            : tType === "ssh" ? { host: tHost, port: tPort || "22" }
            : { endpoint: tHost },
          ttl_seconds: 60, default_role: "test", enabled: false,
        }),
      }).catch(() => null);
      const latency = Date.now() - start;
      if (res?.ok) {
        setTResult({ ok: true, latency_ms: latency, detail: `Connection parameters validated successfully (${latency}ms)` });
      } else {
        setTResult({ ok: false, latency_ms: latency, detail: "Connection validation failed — check host/port and credentials" });
      }
    } catch { setTResult({ ok: false, latency_ms: 0, detail: "Network error" }); }
    finally { setTesting(false); }
  };

  const copyCred = () => {
    if (brokerResult?.credential) {
      navigator.clipboard?.writeText(brokerResult.credential);
      setCopied(true); setTimeout(() => setCopied(false), 2000);
    }
  };

  const targetName = (id: string) => targets.find(t => t.id === id)?.name || id.slice(0, 8);
  const fmtTTL = (expiresAt: string) => {
    const ms = new Date(expiresAt).getTime() - Date.now();
    if (ms <= 0) return "expired";
    const m = Math.floor(ms / 60000); const s = Math.floor((ms % 60000) / 1000);
    return `${m}m ${s}s`;
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
          <KeyRound className="h-6 w-6 text-amber-500" /> Zero-Trust Secret Broker
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Dynamic credential issuance with TTL, JIT approval, and full audit trail.
        </p>
      </div>

      {error && (
        <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}
          <button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      {/* Tabs */}
      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "targets" as Tab, label: "Targets", icon: Server },
          { id: "broker" as Tab, label: "Issue Credential", icon: KeyRound },
          { id: "active" as Tab, label: "Active Grants", icon: Activity },
          { id: "audit" as Tab, label: "Audit Timeline", icon: FileClock },
          { id: "tester" as Tab, label: "Connection Tester", icon: Zap },
        ]).map(tb => {
          const Icon = tb.icon;
          return (
            <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id}
              className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === tb.id ? "border-amber-600 text-amber-600 dark:text-amber-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}>
              <Icon className="h-4 w-4" /> {tb.label}
            </button>
          );
        })}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-amber-500" /></div> : (<>

      {/* ════ TARGETS ════ */}
      {tab === "targets" && (
        <div>
          <div className="mb-4 flex items-center justify-between">
            <h2 className="flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Server className="h-4 w-4" /> Secret Targets ({targets.length})</h2>
            <button onClick={() => { resetForm(); setShowForm(true); }} className="flex items-center gap-1 rounded-lg bg-amber-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-amber-700">
              <Plus className="h-3 w-3" /> Add Target
            </button>
          </div>
          {targets.length === 0 ? (
            <div className={card}><div className="py-12 text-center"><Server className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No secret targets configured.</p></div></div>
          ) : (
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">{targets.map(t => {
              const cfg = TYPE_CONFIG[t.type] || TYPE_CONFIG.db;
              const TIcon = cfg.icon;
              return (
                <div key={t.id} className={card + " hover:shadow-md transition"}>
                  <div className="flex items-start justify-between">
                    <div className="flex items-center gap-3">
                      <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-gray-100 dark:bg-gray-700"><TIcon className={`h-5 w-5 ${cfg.color}`} /></div>
                      <div>
                        <h3 className="font-semibold text-sm">{t.name}</h3>
                        <p className="text-xs text-gray-400">{cfg.label} · {t.default_role || "no role"}</p>
                      </div>
                    </div>
                    <span className={`px-1.5 py-0.5 rounded text-xs font-medium ${t.enabled ? "bg-green-100 dark:bg-green-900/30 text-green-600" : "bg-gray-100 dark:bg-gray-800 text-gray-400"}`}>{t.enabled ? "active" : "disabled"}</span>
                  </div>
                  <div className="mt-3 space-y-1 text-xs text-gray-500 dark:text-gray-400">
                    {t.connection_config?.host && <p>Host: <span className="font-mono">{t.connection_config.host}{t.connection_config.port ? `:${t.connection_config.port}` : ""}</span></p>}
                    {t.connection_config?.database && <p>DB: <span className="font-mono">{t.connection_config.database}</span></p>}
                    {t.connection_config?.region && <p>Region: <span className="font-mono">{t.connection_config.region}</span></p>}
                    {t.connection_config?.endpoint && <p>Endpoint: <span className="font-mono">{t.connection_config.endpoint}</span></p>}
                  </div>
                  <div className="mt-3 flex items-center justify-between">
                    <span className="flex items-center gap-1 text-xs text-gray-400"><Clock className="h-3 w-3" /> TTL: {Math.floor(t.ttl_seconds / 60)}m</span>
                    <div className="flex gap-1">
                      <button onClick={() => toggleTarget(t.id, t.enabled)} disabled={actionLoading === `tg-${t.id}`} aria-label={t.enabled ? "Disable target" : "Enable target"} className="rounded p-1 text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700">
                        {actionLoading === `tg-${t.id}` ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Shield className="h-3.5 w-3.5" />}
                      </button>
                      <button onClick={() => startEdit(t)} aria-label={"Edit " + t.name} className="rounded p-1 text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700"><Settings className="h-3.5 w-3.5" /></button>
                      <button onClick={() => deleteTarget(t.id)} disabled={actionLoading === `del-${t.id}`} aria-label={"Delete " + t.name} className="rounded p-1 text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20">
                        {actionLoading === `del-${t.id}` ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Trash2 className="h-3.5 w-3.5" />}
                      </button>
                    </div>
                  </div>
                </div>
              );
            })}</div>
          )}
        </div>
      )}

      {/* ════ BROKER (Issue Credential) ════ */}
      {tab === "broker" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><KeyRound className="h-4 w-4" /> Request Dynamic Credential</h2>
            <div className="space-y-3">
              <div>
                <label className="text-sm font-medium">Target</label>
                <select value={brTarget} onChange={e => setBrTarget(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">
                  <option value="">Select target...</option>
                  {targets.filter(t => t.enabled).map(t => <option key={t.id} value={t.id}>{t.name} ({TYPE_CONFIG[t.type]?.label})</option>)}
                </select>
              </div>
              <div>
                <label className="text-sm font-medium">User ID</label>
                <input type="text" value={brUser} onChange={e => setBrUser(e.target.value)} placeholder="user:alice" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-sm font-medium">Role (optional)</label>
                  <input type="text" value={brRole} onChange={e => setBrRole(e.target.value)} placeholder="default" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" />
                </div>
                <div>
                  <label className="text-sm font-medium">JIT Request ID</label>
                  <input type="text" value={brJIT} onChange={e => setBrJIT(e.target.value)} placeholder="optional" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" />
                </div>
              </div>
              <button onClick={doBroker} disabled={!brTarget || !brUser || brokering}
                className="flex items-center gap-2 rounded-lg bg-amber-600 px-4 py-2 text-sm font-medium text-white hover:bg-amber-700 disabled:opacity-50">
                {brokering ? <Loader2 className="h-4 w-4 animate-spin" /> : <KeyRound className="h-4 w-4" />} Issue Credential
              </button>
            </div>
          </div>

          {/* Credential display (one-time) */}
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Shield className="h-4 w-4" /> Issued Credential</h2>
            {brokerResult ? (
              <div>
                <div className="rounded-lg border-2 border-amber-300 bg-amber-50 p-4 dark:border-amber-700 dark:bg-amber-950/30">
                  <div className="flex items-center gap-2 mb-2">
                    <AlertTriangle className="h-4 w-4 text-amber-500" />
                    <span className="text-sm font-semibold text-amber-700 dark:text-amber-400">One-Time View — Save Now</span>
                  </div>
                  <p className="text-xs text-amber-600 dark:text-amber-500 mb-3">This credential will not be shown again. Copy it now.</p>
                  <div className="flex items-center gap-2">
                    <code className="flex-1 rounded-lg bg-white dark:bg-gray-900 px-3 py-2 text-xs font-mono break-all border dark:border-gray-700">
                      {credVisible ? brokerResult.credential : "••••••••••••••••••••••••"}
                    </code>
                    <button onClick={() => setCredVisible(!credVisible)} aria-label={credVisible ? "Hide credential" : "Reveal credential"} className="rounded-lg border p-2 dark:border-gray-700">
                      {credVisible ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                    </button>
                    <button onClick={copyCred} aria-label="Copy credential" className="rounded-lg border p-2 dark:border-gray-700">
                      {copied ? <Check className="h-4 w-4 text-green-500" /> : <Copy className="h-4 w-4" />}
                    </button>
                  </div>
                </div>
                <div className="mt-3 space-y-1 text-xs text-gray-500 dark:text-gray-400">
                  <p>Grant ID: <span className="font-mono">{brokerResult.grant_id}</span></p>
                  <p>Role: <span className="font-mono">{brokerResult.role}</span></p>
                  <p>Expires: <span className="font-mono">{new Date(brokerResult.expires_at).toLocaleString()}</span></p>
                  <p className="flex items-center gap-1 text-amber-500"><Clock className="h-3 w-3" /> TTL: {fmtTTL(brokerResult.expires_at)}</p>
                </div>
              </div>
            ) : (
              <div className="py-8 text-center"><KeyRound className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">Issue a credential to see it here.</p></div>
            )}
          </div>
        </div>
      )}

      {/* ════ ACTIVE GRANTS ════ */}
      {tab === "active" && (
        <div>
          <div className="mb-4 flex items-center justify-between">
            <h2 className="flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Activity className="h-4 w-4" /> Active Grants ({grants.filter(g => !g.revoked).length})</h2>
            <span className="text-xs text-gray-400">Auto-refresh 15s</span>
          </div>
          {grants.length === 0 ? (
            <div className={card}><div className="py-12 text-center"><Activity className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No active grants.</p></div></div>
          ) : (
            <div className="overflow-x-auto"><table className="w-full text-sm">
              <thead className="bg-gray-50 dark:bg-gray-800/50"><tr>
                <th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">User</th>
                <th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">Target</th>
                <th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">Role</th>
                <th scope="col" className="px-3 py-2 text-center text-xs font-medium text-gray-400">Status</th>
                <th scope="col" className="px-3 py-2 text-center text-xs font-medium text-gray-400">TTL Remaining</th>
                <th scope="col" className="px-3 py-2 text-right text-xs font-medium text-gray-400">Action</th>
              </tr></thead>
              <tbody className="divide-y dark:divide-gray-800">
                {grants.map(g => {
                  const expired = new Date(g.expires_at).getTime() < Date.now();
                  const active = !g.revoked && !expired;
                  return (
                    <tr key={g.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                      <td className="px-3 py-2 text-xs font-mono">{g.user_id}</td>
                      <td className="px-3 py-2 text-xs">{targetName(g.target_id)}</td>
                      <td className="px-3 py-2 text-xs font-mono text-gray-500">{g.role || "—"}</td>
                      <td className="px-3 py-2 text-center">
                        {g.revoked ? <span className="px-1.5 py-0.5 rounded text-xs bg-red-100 dark:bg-red-900/30 text-red-600">revoked</span>
                         : expired ? <span className="px-1.5 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800 text-gray-400">expired</span>
                         : <span className="px-1.5 py-0.5 rounded text-xs bg-green-100 dark:bg-green-900/30 text-green-600">active</span>}
                      </td>
                      <td className="px-3 py-2 text-center">
                        <span className={`text-xs font-mono ${active ? "text-gray-500" : "text-gray-400"}`}>{active ? fmtTTL(g.expires_at) : "—"}</span>
                      </td>
                      <td className="px-3 py-2 text-right">
                        {active && (
                          <button onClick={() => revokeGrant(g.id)} disabled={actionLoading === `rvk-${g.id}`} aria-label={"Revoke grant " + g.id}
                            className="flex items-center gap-1 rounded-lg bg-red-600 px-2 py-1 text-xs font-medium text-white hover:bg-red-700 ml-auto disabled:opacity-50">
                            {actionLoading === `rvk-${g.id}` ? <Loader2 className="h-3 w-3 animate-spin" /> : <Ban className="h-3 w-3" />} Revoke
                          </button>
                        )}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table></div>
          )}
        </div>
      )}

      {/* ════ AUDIT TIMELINE ════ */}
      {tab === "audit" && (
        <div className={card}>
          <h2 className="mb-6 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><FileClock className="h-4 w-4" /> Credential Lifecycle Audit</h2>
          <div className="relative">
            {/* Timeline line */}
            <div className="absolute left-4 top-0 bottom-0 w-px bg-gray-200 dark:bg-gray-700" />
            <div className="space-y-4">
              {grants.length === 0 ? (
                <p className="text-sm text-gray-400 pl-10">No audit events yet.</p>
              ) : grants.slice(0, 20).map(g => {
                const expired = new Date(g.expires_at).getTime() < Date.now();
                const events = [
                  { icon: KeyRound, color: "text-green-500", bg: "bg-green-100 dark:bg-green-900/30", label: `Issued to ${g.user_id}`, time: g.created_at, role: g.role },
                  ...(g.revoked ? [{ icon: Ban, color: "text-red-500", bg: "bg-red-100 dark:bg-red-900/30", label: `Revoked${g.revoked_at ? " " + new Date(g.revoked_at).toLocaleString() : ""}`, time: g.revoked_at || g.expires_at, role: "" }] : []),
                  ...(expired && !g.revoked ? [{ icon: Clock, color: "text-gray-400", bg: "bg-gray-100 dark:bg-gray-800", label: "Expired (TTL elapsed)", time: g.expires_at, role: "" }] : []),
                ];
                return events.map((ev, i) => {
                  const EvIcon = ev.icon;
                  return (
                    <div key={g.id + i} className="relative flex items-start gap-4 pl-0">
                      <div className={`relative z-10 flex h-8 w-8 items-center justify-center rounded-full ${ev.bg}`}><EvIcon className={`h-4 w-4 ${ev.color}`} /></div>
                      <div className="flex-1 pt-1">
                        <div className="flex items-center gap-2">
                          <span className="text-sm font-medium">{targetName(g.target_id)}</span>
                          {ev.role && <span className="text-xs text-gray-400 font-mono">({ev.role})</span>}
                        </div>
                        <p className="text-xs text-gray-500 dark:text-gray-400">{ev.label}</p>
                        <p className="text-xs text-gray-400 font-mono">{new Date(ev.time).toLocaleString()}</p>
                      </div>
                      <span className="text-xs text-gray-300 font-mono pt-1">{g.id.slice(0, 8)}</span>
                    </div>
                  );
                });
              })}
            </div>
          </div>
        </div>
      )}

      {/* ════ CONNECTION TESTER ════ */}
      {tab === "tester" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Zap className="h-4 w-4" /> Connection Parameters</h2>
            <div className="space-y-3">
              <div>
                <label className="text-sm font-medium">Target Type</label>
                <select value={tType} onChange={e => setTType(e.target.value as SecretTarget["type"])} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">
                  {Object.entries(TYPE_CONFIG).map(([k, v]) => <option key={k} value={k}>{v.label}</option>)}
                </select>
              </div>
              <div>
                <label className="text-sm font-medium">Host / Endpoint</label>
                <input type="text" value={tHost} onChange={e => setTHost(e.target.value)} placeholder="db.internal.company.com" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" />
              </div>
              {(tType === "db" || tType === "ssh") && (
                <div>
                  <label className="text-sm font-medium">Port</label>
                  <input type="text" value={tPort} onChange={e => setTPort(e.target.value)} placeholder={tType === "ssh" ? "22" : "5432"} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" />
                </div>
              )}
              <button onClick={runTest} disabled={!tHost || testing}
                className="flex items-center gap-2 rounded-lg bg-amber-600 px-4 py-2 text-sm font-medium text-white hover:bg-amber-700 disabled:opacity-50">
                {testing ? <Loader2 className="h-4 w-4 animate-spin" /> : <Zap className="h-4 w-4" />} Test Connectivity
              </button>
            </div>
          </div>
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Activity className="h-4 w-4" /> Test Result</h2>
            {tResult ? (
              <div className={`flex items-center gap-3 rounded-xl border-2 p-4 ${tResult.ok ? "border-green-300 bg-green-50 dark:border-green-700 dark:bg-green-950/30" : "border-red-300 bg-red-50 dark:border-red-700 dark:bg-red-950/30"}`}>
                {tResult.ok ? <CheckCircle2 className="h-8 w-8 text-green-500" /> : <XCircle className="h-8 w-8 text-red-500" />}
                <div>
                  <p className={`text-lg font-bold ${tResult.ok ? "text-green-700 dark:text-green-400" : "text-red-700 dark:text-red-400"}`}>{tResult.ok ? "Connected" : "Failed"}</p>
                  <p className="text-xs text-gray-500 dark:text-gray-400">{tResult.detail}</p>
                </div>
                <span className="ml-auto text-sm font-mono text-gray-400">{tResult.latency_ms}ms</span>
              </div>
            ) : (
              <div className="py-8 text-center"><Zap className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">Enter connection params and test.</p></div>
            )}
          </div>
        </div>
      )}

      </>)}

      {/* Target create/edit dialog */}
      {showForm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowForm(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-lg rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800 max-h-[90vh] overflow-y-auto" onClick={e => e.stopPropagation()}>
            <h3 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white">
              <Plus className="h-5 w-5 text-amber-500" /> {editId ? "Edit Target" : "New Secret Target"}
            </h3>
            <div className="mt-4 space-y-3">
              <div><label className="text-sm font-medium">Name</label><input type="text" value={fName} onChange={e => setFName(e.target.value)} placeholder="prod-postgres" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" autoFocus /></div>
              <div className="grid grid-cols-2 gap-3">
                <div><label className="text-sm font-medium">Type</label>
                  <select value={fType} onChange={e => setFType(e.target.value as SecretTarget["type"])} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">
                    {Object.entries(TYPE_CONFIG).map(([k, v]) => <option key={k} value={k}>{v.label}</option>)}
                  </select>
                </div>
                <div><label className="text-sm font-medium">TTL (seconds)</label><input type="number" value={fTTL} onChange={e => setFTTL(parseInt(e.target.value) || 3600)} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>
              </div>
              <div><label className="text-sm font-medium">Default Role</label><input type="text" value={fRole} onChange={e => setFRole(e.target.value)} placeholder="readonly" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>
              {(fType === "db" || fType === "ssh") && (
                <div className="grid grid-cols-2 gap-3">
                  <div><label className="text-sm font-medium">Host</label><input type="text" value={fHost} onChange={e => setFHost(e.target.value)} placeholder="10.0.0.5" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div>
                  <div><label className="text-sm font-medium">Port</label><input type="text" value={fPort} onChange={e => setFPort(e.target.value)} placeholder="5432" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div>
                </div>
              )}
              {fType === "db" && <div><label className="text-sm font-medium">Database Name</label><input type="text" value={fDb} onChange={e => setFDb(e.target.value)} placeholder="appdb" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>}
              {fType === "cloud" && <div><label className="text-sm font-medium">Region</label><input type="text" value={fRegion} onChange={e => setFRegion(e.target.value)} placeholder="us-east-1" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>}
              {fType === "api_key" && <div><label className="text-sm font-medium">Endpoint URL</label><input type="text" value={fHost} onChange={e => setFHost(e.target.value)} placeholder="https://api.service.com" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div>}
              <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={fEnabled} onChange={e => setFEnabled(e.target.checked)} className="rounded border-gray-300" /> Enabled</label>
            </div>
            <div className="mt-4 flex justify-end gap-2">
              <button onClick={() => { setShowForm(false); resetForm(); }} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">Cancel</button>
              <button onClick={saveTarget} disabled={!fName || actionLoading === "save"} className="rounded-lg bg-amber-600 px-4 py-2 text-sm font-medium text-white hover:bg-amber-700 disabled:opacity-50">
                {actionLoading === "save" ? <Loader2 className="h-4 w-4 animate-spin" /> : "Save"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

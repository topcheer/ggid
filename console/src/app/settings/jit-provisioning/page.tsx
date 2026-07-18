"use client";
import { useState, useCallback, useEffect } from "react";
import { Users, Loader2, AlertCircle, X, RefreshCw, Plus, Trash2, Check, CheckCircle, XCircle, Code, TestTube, ArrowRight, ChevronRight, Settings, Eye, Shield, GitBranch, Zap } from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface AttrMapping { id: string; source_attr: string; ggid_field: string; transform: string; }
interface RoleMapping { id: string; source_group: string; ggid_role: string; }
interface IdPConfig { id: string; name: string; protocol: string; enabled: boolean; update_strategy: "always" | "first_only"; mapped_attrs: number; mapped_roles: number; }
interface DryRunResult { action: "created" | "updated" | "no_change"; username: string; assigned_roles: string[]; warnings: string[]; }

const PROTOCOLS = ["saml", "oidc", "ldap", "scim"];
const GGID_FIELDS = ["username", "email", "first_name", "last_name", "display_name", "phone", "department", "title", "manager", "employee_id"];

type Tab = "mapping" | "roles" | "flow" | "dryrun" | "idps";

export default function JITProvisioningPage() {
  const [tab, setTab] = useState<Tab>("mapping");
  const [attrMaps, setAttrMaps] = useState<AttrMapping[]>([]);
  const [roleMaps, setRoleMaps] = useState<RoleMapping[]>([]);
  const [idps, setIdps] = useState<IdPConfig[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  // dry run
  const [dryProtocol, setDryProtocol] = useState("saml");
  const [dryInput, setDryInput] = useState('{"email":"alice@corp.com","firstName":"Alice","lastName":"Zhang","groups":["Engineering","Admins"]}');
  const [dryResult, setDryResult] = useState<DryRunResult | null>(null);
  const [dryRunning, setDryRunning] = useState(false);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
      const [aRes, rRes, iRes] = await Promise.all([
        fetch("/api/v1/identity/jit/attr-mappings", { headers: h }).catch(() => null),
        fetch("/api/v1/identity/jit/role-mappings", { headers: h }).catch(() => null),
        fetch("/api/v1/identity/jit/idp-configs", { headers: h }).catch(() => null),
      ]);
      if (aRes?.ok) { const d = await aRes.json(); setAttrMaps(d.mappings || []); }
      if (rRes?.ok) { const d = await rRes.json(); setRoleMaps(d.mappings || []); }
      if (iRes?.ok) { const d = await iRes.json(); setIdps(d.idps || d.items || []); }
    } catch { setError("Failed to load JIT data"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const runDryRun = async () => {
    setDryRunning(true); setDryResult(null);
    try {
      const res = await fetch("/api/v1/identity/jit/dry-run", {
        method: "POST", headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify({ protocol: dryProtocol, assertion: JSON.parse(dryInput) }),
      });
      if (res.ok) setDryResult(await res.json());
      else setError("Dry run failed");
    } catch { setError("Invalid JSON or network error"); }
    finally { setDryRunning(false); }
  };

  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
          <Users className="h-6 w-6 text-indigo-500" /> JIT User Provisioning
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Just-In-Time user provisioning per IdP — attribute mapping, role mapping, dry-run simulation.
        </p>
      </div>

      {error && (
        <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}
          <button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "mapping" as Tab, label: "Attribute Mapping", icon: GitBranch },
          { id: "roles" as Tab, label: "Role Mapping", icon: Shield },
          { id: "flow" as Tab, label: "Flow Diagram", icon: Zap },
          { id: "dryrun" as Tab, label: "Dry Run", icon: TestTube },
          { id: "idps" as Tab, label: "IdP Config", icon: Settings },
        ]).map(tb => {
          const Icon = tb.icon;
          return (
            <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id}
              className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === tb.id ? "border-indigo-600 text-indigo-600 dark:text-indigo-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}>
              <Icon className="h-4 w-4" /> {tb.label}
            </button>
          );
        })}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-500" /></div> : (<>

      {/* ATTRIBUTE MAPPING */}
      {tab === "mapping" && (
        <div className={card}>
          <div className="mb-4 flex items-center justify-between">
            <h2 className="flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><GitBranch className="h-4 w-4" /> Attribute Mapping DSL</h2>
            <button onClick={() => setAttrMaps([...attrMaps, { id: Date.now().toString(), source_attr: "email", ggid_field: "email", transform: "direct" }])} className="flex items-center gap-1 rounded-lg bg-indigo-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-indigo-700"><Plus className="h-3 w-3" /> Add Mapping</button>
          </div>
          {attrMaps.length === 0 ? (
            <div className="py-8 text-center"><GitBranch className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No attribute mappings configured.</p></div>
          ) : (
            <div className="space-y-2">
              {attrMaps.map(m => (
                <div key={m.id} className="flex items-center gap-2 rounded-lg border p-2 dark:border-gray-700">
                  <input aria-label="Source attr" type="text" defaultValue={m.source_attr} className="w-32 rounded border dark:border-gray-700 dark:bg-gray-900 px-2 py-1 text-xs font-mono" />
                  <select aria-label="Transform" defaultValue={m.transform} className="rounded border dark:border-gray-700 dark:bg-gray-900 px-2 py-1 text-xs">
                    <option value="direct">direct</option><option value="rename">rename</option><option value="concat">concat</option><option value="regex">regex</option>
                  </select>
                  <ArrowRight className="h-3 w-3 text-gray-400" />
                  <select aria-label="GGID field" defaultValue={m.ggid_field} className="rounded border dark:border-gray-700 dark:bg-gray-900 px-2 py-1 text-xs">
                    {GGID_FIELDS.map(f => <option key={f}>{f}</option>)}
                  </select>
                  <button onClick={() => setAttrMaps(prev => prev.filter(x => x.id !== m.id))} className="ml-auto text-red-400"><Trash2 className="h-3.5 w-3.5" /></button>
                </div>
              ))}
            </div>
          )}
          <div className="mt-4"><p className="text-xs font-semibold uppercase text-gray-400 mb-1">YAML Preview</p><pre className="overflow-x-auto rounded-lg bg-gray-900 p-3 text-xs text-green-400 font-mono">{attrMaps.length > 0 ? `mappings:\n${attrMaps.map(m => `  - source: ${m.source_attr}\n    target: ${m.ggid_field}\n    transform: ${m.transform}`).join("\n")}` : "# Add mappings to see YAML"}</pre></div>
        </div>
      )}

      {/* ROLE MAPPING */}
      {tab === "roles" && (
        <div className={card}>
          <div className="mb-4 flex items-center justify-between">
            <h2 className="flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Shield className="h-4 w-4" /> External Group → GGID Role</h2>
            <button onClick={() => setRoleMaps([...roleMaps, { id: Date.now().toString(), source_group: "Engineering", ggid_role: "developer" }])} className="flex items-center gap-1 rounded-lg bg-indigo-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-indigo-700"><Plus className="h-3 w-3" /> Add</button>
          </div>
          {roleMaps.length === 0 ? (
            <div className="py-8 text-center"><Shield className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No role mappings.</p></div>
          ) : (
            <div className="space-y-2">{roleMaps.map(r => (
              <div key={r.id} className="flex items-center gap-2 rounded-lg border p-2 dark:border-gray-700">
                <span className="text-xs font-mono text-blue-600 dark:text-blue-400 flex-1">{r.source_group}</span>
                <ArrowRight className="h-3 w-3 text-gray-400" />
                <input aria-label="GGID role" type="text" defaultValue={r.ggid_role} className="w-32 rounded border dark:border-gray-700 dark:bg-gray-900 px-2 py-1 text-xs font-mono" />
                <button onClick={() => setRoleMaps(prev => prev.filter(x => x.id !== r.id))} className="ml-auto text-red-400"><Trash2 className="h-3.5 w-3.5" /></button>
              </div>
            ))}</div>
          )}
        </div>
      )}

      {/* FLOW DIAGRAM */}
      {tab === "flow" && (
        <div className={card}>
          <h2 className="mb-6 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Zap className="h-4 w-4" /> JIT Provisioning Flow</h2>
          <div className="space-y-3">
            {[
              { step: 1, title: "Extract", desc: "Parse assertion/claim from IdP response (SAML/OIDC/LDAP/SCIM)", color: "bg-blue-500" },
              { step: 2, title: "Resolve", desc: "Match external identity to existing user (email/username/external_id)", color: "bg-yellow-500" },
              { step: 3, title: "Create or Update", desc: "New user → create with mapped attrs. Existing → update per strategy.", color: "bg-indigo-500" },
              { step: 4, title: "Map Roles", desc: "Apply group→role mappings, assign/revoke per update strategy", color: "bg-purple-500" },
              { step: 5, title: "Audit", desc: "Log JIT event to audit trail with full before/after diff", color: "bg-green-500" },
            ].map(s => (
              <div key={s.step} className="flex items-start gap-4">
                <div className={`flex h-8 w-8 items-center justify-center rounded-full text-white text-xs font-bold shrink-0 ${s.color}`}>{s.step}</div>
                <div className="flex-1 rounded-lg border p-3 dark:border-gray-700">
                  <p className="text-sm font-medium">{s.title}</p>
                  <p className="text-xs text-gray-400">{s.desc}</p>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* DRY RUN */}
      {tab === "dryrun" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><TestTube className="h-4 w-4" /> Simulate JIT</h2>
            <div className="space-y-3">
              <div>
                <label className="text-sm font-medium">Protocol</label>
                <select aria-label="Protocol" value={dryProtocol} onChange={e => setDryProtocol(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">
                  {PROTOCOLS.map(p => <option key={p} value={p}>{p.toUpperCase()}</option>)}
                </select>
              </div>
              <div>
                <label className="text-sm font-medium">Assertion/Claims (JSON)</label>
                <textarea aria-label="Assertion JSON" value={dryInput} onChange={e => setDryInput(e.target.value)} rows={6} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 font-mono text-xs" />
              </div>
              <button onClick={runDryRun} disabled={dryRunning} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">
                {dryRunning ? <Loader2 className="h-4 w-4 animate-spin" /> : <TestTube className="h-4 w-4" />} Run JIT Simulation
              </button>
            </div>
          </div>
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Eye className="h-4 w-4" /> Result</h2>
            {dryResult ? (
              <div>
                <div className={`flex items-center gap-3 rounded-xl border-2 p-4 ${dryResult.action === "created" ? "border-green-300 bg-green-50 dark:border-green-700 dark:bg-green-950/30" : dryResult.action === "updated" ? "border-blue-300 bg-blue-50 dark:border-blue-700 dark:bg-blue-950/30" : "border-gray-300 dark:border-gray-700"}`}>
                  {dryResult.action === "created" ? <CheckCircle className="h-8 w-8 text-green-500" /> : dryResult.action === "updated" ? <RefreshCw className="h-8 w-8 text-blue-500" /> : <Check className="h-8 w-8 text-gray-400" />}
                  <div>
                    <p className="text-lg font-bold capitalize">{dryResult.action.replace("_", " ")}</p>
                    <p className="text-xs text-gray-500">User: {dryResult.username || "—"}</p>
                  </div>
                </div>
                {dryResult.assigned_roles?.length > 0 && (
                  <div className="mt-3">
                    <p className="text-xs font-semibold text-gray-400 mb-1">Assigned Roles</p>
                    <div className="flex flex-wrap gap-1">{dryResult.assigned_roles.map(r => <span key={r} className="px-1.5 py-0.5 rounded bg-purple-100 dark:bg-purple-900/30 text-xs font-mono">{r}</span>)}</div>
                  </div>
                )}
                {dryResult.warnings?.length > 0 && (
                  <div className="mt-3 rounded-lg bg-yellow-50 p-3 dark:bg-yellow-950/20">
                    {dryResult.warnings.map((w: any, i: number) => <p key={i} className="text-xs text-yellow-700 dark:text-yellow-400">{w}</p>)}
                  </div>
                )}
              </div>
            ) : (
              <div className="py-8 text-center"><TestTube className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">Run simulation to preview JIT result.</p></div>
            )}
          </div>
        </div>
      )}

      {/* IdP CONFIG */}
      {tab === "idps" && (
        <div className={card}>
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Settings className="h-4 w-4" /> Per-IdP JIT Configuration</h2>
          {idps.length === 0 ? (
            <div className="py-8 text-center"><Settings className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No IdP configurations. IdPs appear here after federation setup.</p></div>
          ) : (
            <div className="space-y-2">{idps.map(idp => (
              <div key={idp.id} className="rounded-lg border p-3 dark:border-gray-700">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <div className={`h-2.5 w-2.5 rounded-full ${idp.enabled ? "bg-green-500" : "bg-gray-400"}`} />
                    <span className="font-medium text-sm">{idp.name}</span>
                    <span className="px-1.5 py-0.5 rounded text-xs font-mono bg-gray-100 dark:bg-gray-700">{idp.protocol}</span>
                  </div>
                  <span className="text-xs text-gray-400">{idp.update_strategy === "always" ? "Sync every login" : "Create only"}</span>
                </div>
                <div className="mt-2 flex gap-4 text-xs text-gray-400">
                  <span>{idp.mapped_attrs} attr mappings</span>
                  <span>{idp.mapped_roles} role mappings</span>
                </div>
              </div>
            ))}</div>
          )}
        </div>
      )}

      </>)}
    </div>
  );
}

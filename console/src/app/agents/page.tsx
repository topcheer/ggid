"use client";

import { useState, useCallback, useEffect } from "react";
import {
  Bot, Plus, X, Key, Ban, CheckCircle2, ChevronDown, ChevronRight,
  Copy, Loader2, AlertCircle, Trash2, Eye, EyeOff, Terminal, Shield,
} from "lucide-react";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";

interface Agent {
  id: string;
  client_id: string;
  tenant_id: string;
  name: string;
  type: string;
  owner_user_id: string;
  status: string;
  allowed_scopes: string[];
  max_delegation_depth: number;
  allowed_mcp_servers: string[];
  created_at: string;
  updated_at: string;
}

const AGENT_TYPES = [
  "coding-assistant",
  "data-pipeline",
  "customer-service",
  "workflow-orchestrator",
  "research-agent",
  "custom",
] as const;

const inputCls = "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 focus:border-brand-500 focus:outline-none";
const cardCls = "rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800";
const btnPrimary = "flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50";
const btnGhost = "rounded-lg border border-gray-300 px-3 py-1.5 text-sm font-medium hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700";

export default function AgentsPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [agents, setAgents] = useState<Agent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [msg, setMsg] = useState("");
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [showRegister, setShowRegister] = useState(false);
  const [showTokenExchange, setShowTokenExchange] = useState(false);

  // Register form
  const [regForm, setRegForm] = useState({
    name: "",
    type: "coding-assistant" as string,
    owner_user_id: "",
    allowed_scopes: "read:users,write:users",
    max_delegation_depth: 3,
    allowed_mcp_servers: "",
  });
  const [registering, setRegistering] = useState(false);
  const [newClientId, setNewClientId] = useState("");
  const [newClientSecret, setNewClientSecret] = useState("");
  const [showSecret, setShowSecret] = useState(false);

  // Token exchange
  const [tokenForm, setTokenForm] = useState({
    agent_id: "",
    subject_token: "",
    scope: "read:users",
    mcp_servers: "",
    audience: "",
  });
  const [exchanging, setExchanging] = useState(false);
  const [exchangedToken, setExchangedToken] = useState("");
  const [verifyInput, setVerifyInput] = useState("");
  const [verifyResult, setVerifyResult] = useState("");
  const [verifying, setVerifying] = useState(false);

  const loadAgents = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const data = await apiFetch<{ agents?: Agent[]; items?: Agent[] }>("/api/v1/agents");
      setAgents(data.agents || data.items || []);
    } catch {
      // API may not be available yet — show empty state
      setAgents([]);
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => {
    loadAgents();
  }, [loadAgents]);

  const showMessage = (m: string) => {
    setMsg(m);
    setTimeout(() => setMsg(""), 4000);
  };

  const handleRegister = async () => {
    if (!regForm.name.trim()) {
      setError("Agent name is required");
      return;
    }
    setRegistering(true);
    setError("");
    try {
      const scopes = regForm.allowed_scopes.split(",").map((s: any) => s.trim()).filter(Boolean);
      const mcpServers = regForm.allowed_mcp_servers.split(",").map((s: any) => s.trim()).filter(Boolean);
      const data = await apiFetch<{ id: string; client_id: string; client_secret?: string }>("/api/v1/agents/register", {
        method: "POST",
        body: JSON.stringify({
          name: regForm.name,
          type: regForm.type,
          owner_user_id: regForm.owner_user_id || undefined,
          allowed_scopes: scopes,
          max_delegation_depth: regForm.max_delegation_depth,
          allowed_mcp_servers: mcpServers,
        }),
      });
      setNewClientId(data.client_id || data.id);
      setNewClientSecret(data.client_secret || "");
      setShowRegister(false);
      setRegForm({ name: "", type: "coding-assistant", owner_user_id: "", allowed_scopes: "read:users,write:users", max_delegation_depth: 3, allowed_mcp_servers: "" });
      showMessage(`Agent "${regForm.name}" registered successfully`);
      loadAgents();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to register agent");
    } finally {
      setRegistering(false);
    }
  };

  const handleSuspend = async (agent: Agent) => {
    try {
      await apiFetch(`/api/v1/agents/${agent.id}/suspend`, { method: "POST" });
      showMessage(`Agent "${agent.name}" suspended`);
      loadAgents();
    } catch {
      showMessage(`Failed to suspend agent (API may not be available)`);
    }
  };

  const handleActivate = async (agent: Agent) => {
    try {
      await apiFetch(`/api/v1/agents/${agent.id}/activate`, { method: "POST" });
      showMessage(`Agent "${agent.name}" activated`);
      loadAgents();
    } catch {
      showMessage(`Failed to activate agent (API may not be available)`);
    }
  };

  const handleRevoke = async (agent: Agent) => {
    if (!confirm(`Revoke agent "${agent.name}"? This permanently deletes the agent and invalidates all tokens.`)) return;
    try {
      await apiFetch(`/api/v1/agents/${agent.id}`, { method: "DELETE" });
      showMessage(`Agent "${agent.name}" revoked`);
      loadAgents();
    } catch {
      showMessage(`Failed to revoke agent (API may not be available)`);
    }
  };

  const handleTokenExchange = async () => {
    if (!tokenForm.agent_id.trim() || !tokenForm.subject_token.trim()) {
      setError("Agent ID and Subject Token are required");
      return;
    }
    setExchanging(true);
    setError("");
    setExchangedToken("");
    try {
      const mcpServers = tokenForm.mcp_servers.split(",").map((s: any) => s.trim()).filter(Boolean);
      const data = await apiFetch<{ access_token?: string; agent_token?: string; token?: string }>("/api/v1/agents/token", {
        method: "POST",
        body: JSON.stringify({
          agent_id: tokenForm.agent_id,
          subject_token: tokenForm.subject_token,
          scope: tokenForm.scope,
          mcp_servers: mcpServers,
          audience: tokenForm.audience || undefined,
        }),
      });
      setExchangedToken(data.access_token || data.agent_token || data.token || "");
      showMessage("Token exchange successful");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Token exchange failed");
    } finally {
      setExchanging(false);
    }
  };

  const handleVerify = async () => {
    if (!verifyInput.trim()) return;
    setVerifying(true);
    setVerifyResult("");
    try {
      const data = await apiFetch<{ active?: boolean; valid?: boolean; claims?: Record<string, unknown> }>("/api/v1/agents/verify", {
        method: "POST",
        body: JSON.stringify({ token: verifyInput }),
      });
      const isActive = data.active ?? data.valid ?? false;
      setVerifyResult(JSON.stringify({ active: isActive, claims: data.claims || {} }, null, 2));
    } catch (err) {
      setVerifyResult(JSON.stringify({ active: false, error: err instanceof Error ? err.message : "verification failed" }, null, 2));
    } finally {
      setVerifying(false);
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold dark:text-gray-100">
            <Bot className="h-6 w-6 text-brand-600" /> {t("agents.title")}
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {t("agents.subtitle")}
          </p>
        </div>
        <div className="flex gap-2">
          <button onClick={() => setShowTokenExchange(true)} className={btnGhost}>
            <Terminal className="mr-1.5 inline h-4 w-4" /> Token Exchange
          </button>
          <button onClick={() => setShowRegister(true)} className={btnPrimary}>
            <Plus className="h-4 w-4" /> Register Agent
          </button>
        </div>
      </div>

      {/* Messages */}
      {msg && (
        <div className="rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400">
          {msg}
        </div>
      )}
      {error && (
        <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400">
          {error}
        </div>
      )}

      {/* New client secret banner */}
      {newClientSecret && (
        <div className="rounded-lg border border-amber-300 bg-amber-50 p-4 dark:border-amber-800 dark:bg-amber-950">
          <div className="flex items-start gap-3">
            <AlertCircle className="mt-0.5 h-5 w-5 shrink-0 text-amber-600" />
            <div className="flex-1">
              <p className="font-medium text-amber-800 dark:text-amber-400">Client Secret Created — Copy NOW!</p>
              <p className="text-xs text-amber-700 dark:text-amber-500">This secret will NOT be shown again.</p>
              <div className="mt-2 flex items-center gap-2">
                <code className="flex-1 break-all rounded bg-white px-2 py-1 font-mono text-xs dark:bg-gray-800">
                  {showSecret ? newClientSecret : "••••••••••••••••"}
                </code>
                <button onClick={() => setShowSecret(!showSecret)} className="rounded-lg border border-gray-300 p-2 dark:border-gray-600" title="Toggle visibility">
                  {showSecret ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
                <button onClick={() => navigator.clipboard.writeText(newClientSecret)} className="rounded-lg border border-gray-300 p-2 dark:border-gray-600" title="Copy">
                  <Copy className="h-4 w-4" />
                </button>
                <button onClick={() => setNewClientSecret("")} className="rounded-lg border border-gray-300 px-3 py-2 text-xs dark:border-gray-600">Dismiss</button>
              </div>
              <p className="mt-1 text-xs text-gray-500">Client ID: <code className="font-mono">{newClientId}</code></p>
            </div>
          </div>
        </div>
      )}

      {/* Loading */}
      {loading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-6 w-6 animate-spin text-brand-600" />
        </div>
      ) : agents.length === 0 ? (
        <div className={cardCls}>
          <div className="py-8 text-center">
            <Bot className="mx-auto mb-3 h-10 w-10 text-gray-400" />
            <p className="text-gray-500 dark:text-gray-400">No AI agents registered yet</p>
            <p className="mt-1 text-xs text-gray-400">Click "Register Agent" to create your first AI agent</p>
          </div>
        </div>
      ) : (
        /* Agent Table */
        <div className="overflow-x-auto rounded-xl border border-gray-200 dark:border-gray-700">
          <table className="w-full min-w-[800px]">
            <thead className="border-b border-gray-200 bg-gray-50 dark:border-gray-700 dark:bg-gray-800">
              <tr>
                <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500"></th>
                <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Name</th>
                <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Type</th>
                <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Status</th>
                <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Scopes</th>
                <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Created</th>
                <th scope="col" className="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
              {agents.map((agent: any) => (
                <>
                  <tr key={agent.id} className="hover:bg-gray-50 dark:hover:bg-gray-800/50">
                    <td className="px-4 py-3">
                      <button
                        onClick={() => setExpandedId(expandedId === agent.id ? null : agent.id)}
                        className="text-gray-400 hover:text-gray-600"
                        aria-label="Toggle details"
                      >
                        {expandedId === agent.id ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
                      </button>
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2">
                        <Bot className="h-4 w-4 text-brand-600" />
                        <span className="font-medium dark:text-gray-100">{agent.name}</span>
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <span className="rounded-full bg-brand-100 px-2 py-0.5 text-xs font-medium text-brand-700 dark:bg-brand-900 dark:text-brand-400">
                        {agent.type}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      {agent.status === "active" ? (
                        <span className="inline-flex items-center gap-1 rounded-full bg-green-100 px-2 py-0.5 text-xs font-medium text-green-700 dark:bg-green-900 dark:text-green-400">
                          <CheckCircle2 className="h-3 w-3" /> Active
                        </span>
                      ) : agent.status === "suspended" ? (
                        <span className="inline-flex items-center gap-1 rounded-full bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-900 dark:text-amber-400">
                          <Ban className="h-3 w-3" /> Suspended
                        </span>
                      ) : (
                        <span className="inline-flex items-center gap-1 rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-600 dark:bg-gray-700 dark:text-gray-400">
                          {agent.status}
                        </span>
                      )}
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex flex-wrap gap-1">
                        {(agent.allowed_scopes || []).slice(0, 3).map((s: any) => (
                          <span key={s} className="rounded bg-gray-100 px-1.5 py-0.5 text-xs text-gray-600 dark:bg-gray-700 dark:text-gray-400">
                            {s}
                          </span>
                        ))}
                        {(agent.allowed_scopes || []).length > 3 && (
                          <span className="text-xs text-gray-400">+{agent.allowed_scopes.length - 3}</span>
                        )}
                      </div>
                    </td>
                    <td className="px-4 py-3 text-xs text-gray-500 dark:text-gray-400">
                      {new Date(agent.created_at).toLocaleDateString()}
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex justify-end gap-1">
                        {agent.status === "active" ? (
                          <button onClick={() => handleSuspend(agent)} title="Suspend" className="rounded p-1.5 text-gray-400 hover:bg-amber-50 hover:text-amber-600">
                            <Ban className="h-4 w-4" />
                          </button>
                        ) : (
                          <button onClick={() => handleActivate(agent)} title="Activate" className="rounded p-1.5 text-gray-400 hover:bg-green-50 hover:text-green-600">
                            <CheckCircle2 className="h-4 w-4" />
                          </button>
                        )}
                        <button onClick={() => handleRevoke(agent)} title="Revoke" className="rounded p-1.5 text-gray-400 hover:bg-red-50 hover:text-red-600">
                          <Trash2 className="h-4 w-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                  {/* Expanded detail row */}
                  {expandedId === agent.id && (
                    <tr className="bg-gray-50 dark:bg-gray-800/30">
                      <td colSpan={7} className="px-4 py-4">
                        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
                          <div>
                            <p className="text-xs font-medium uppercase text-gray-400">Agent ID</p>
                            <p className="font-mono text-xs text-gray-700 dark:text-gray-300">{agent.id}</p>
                          </div>
                          <div>
                            <p className="text-xs font-medium uppercase text-gray-400">Client ID</p>
                            <p className="font-mono text-xs text-gray-700 dark:text-gray-300">{agent.client_id}</p>
                          </div>
                          <div>
                            <p className="text-xs font-medium uppercase text-gray-400">Owner User ID</p>
                            <p className="font-mono text-xs text-gray-700 dark:text-gray-300">{agent.owner_user_id || "—"}</p>
                          </div>
                          <div>
                            <p className="text-xs font-medium uppercase text-gray-400">Max Delegation Depth</p>
                            <p className="text-sm font-medium text-gray-700 dark:text-gray-300">{agent.max_delegation_depth}</p>
                          </div>
                          <div>
                            <p className="text-xs font-medium uppercase text-gray-400">Allowed MCP Servers</p>
                            <div className="flex flex-wrap gap-1">
                              {(agent.allowed_mcp_servers || []).length > 0 ? agent.allowed_mcp_servers.map((s: any) => (
                                <span key={s} className="rounded bg-blue-100 px-1.5 py-0.5 text-xs text-blue-700 dark:bg-blue-900 dark:text-blue-400">
                                  {s}
                                </span>
                              )) : <span className="text-xs text-gray-400">None</span>}
                            </div>
                          </div>
                          <div>
                            <p className="text-xs font-medium uppercase text-gray-400">All Scopes</p>
                            <div className="flex flex-wrap gap-1">
                              {(agent.allowed_scopes || []).map((s: any) => (
                                <span key={s} className="rounded bg-gray-200 px-1.5 py-0.5 text-xs text-gray-700 dark:bg-gray-600 dark:text-gray-300">
                                  {s}
                                </span>
                              ))}
                            </div>
                          </div>
                        </div>
                        <div className="mt-3 flex gap-2">
                          <button
                            onClick={() => {
                              setTokenForm({ ...tokenForm, agent_id: agent.id });
                              setShowTokenExchange(true);
                            }}
                            className="flex items-center gap-1.5 rounded-lg border border-brand-300 px-3 py-1.5 text-xs font-medium text-brand-600 hover:bg-brand-50 dark:border-brand-700 dark:hover:bg-brand-950"
                          >
                            <Key className="h-3.5 w-3.5" /> Exchange Token
                          </button>
                        </div>
                      </td>
                    </tr>
                  )}
                </>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Register Modal */}
      {showRegister && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" onClick={() => setShowRegister(false)}>
          <div role="dialog" aria-modal="true" className="max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="mb-4 flex items-center justify-between">
              <h2 className="flex items-center gap-2 text-lg font-semibold dark:text-gray-100">
                <Plus className="h-5 w-5 text-brand-600" /> Register New Agent
              </h2>
              <button onClick={() => setShowRegister(false)} className="text-gray-400 hover:text-gray-600" aria-label="Close">
                <X className="h-5 w-5" />
              </button>
            </div>
            <div className="space-y-4">
              <div>
                <label className="mb-1 block text-sm font-medium dark:text-gray-300">Agent Name *</label>
                <input aria-label="My Coding Assistant" value={regForm.name} onChange={(e) => setRegForm({ ...regForm, name: e.target.value })} className={inputCls} placeholder="My Coding Assistant" autoFocus />
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium dark:text-gray-300">Agent Type</label>
                <select aria-label="reg Form" value={regForm.type} onChange={(e) => setRegForm({ ...regForm, type: e.target.value })} className={inputCls}>
                  {AGENT_TYPES.map((t: any) => <option key={t} value={t}>{t}</option>)}
                </select>
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium dark:text-gray-300">Owner User ID (optional)</label>
                <input aria-label="user-uuid" value={regForm.owner_user_id} onChange={(e) => setRegForm({ ...regForm, owner_user_id: e.target.value })} className={inputCls} placeholder="user-uuid" />
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium dark:text-gray-300">Allowed Scopes (comma-separated)</label>
                <input aria-label="read:users,write:users" value={regForm.allowed_scopes} onChange={(e) => setRegForm({ ...regForm, allowed_scopes: e.target.value })} className={inputCls} placeholder="read:users,write:users" />
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium dark:text-gray-300">Max Delegation Depth</label>
                <input aria-label="reg Form" type="number" min={1} max={10} value={regForm.max_delegation_depth} onChange={(e) => setRegForm({ ...regForm, max_delegation_depth: parseInt(e.target.value) || 3 })} className={inputCls} />
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium dark:text-gray-300">Allowed MCP Servers (comma-separated, optional)</label>
                <input aria-label="github-server,filesystem-server" value={regForm.allowed_mcp_servers} onChange={(e) => setRegForm({ ...regForm, allowed_mcp_servers: e.target.value })} className={inputCls} placeholder="github-server,filesystem-server" />
              </div>
            </div>
            <div className="mt-6 flex gap-2">
              <button onClick={handleRegister} disabled={registering || !regForm.name} className={btnPrimary}>
                {registering ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
                Register Agent
              </button>
              <button onClick={() => setShowRegister(false)} className={btnGhost}>Cancel</button>
            </div>
          </div>
        </div>
      )}

      {/* Token Exchange Modal */}
      {showTokenExchange && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" onClick={() => setShowTokenExchange(false)}>
          <div role="dialog" aria-modal="true" className="max-h-[90vh] w-full max-w-xl overflow-y-auto rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="mb-4 flex items-center justify-between">
              <h2 className="flex items-center gap-2 text-lg font-semibold dark:text-gray-100">
                <Terminal className="h-5 w-5 text-brand-600" /> Token Exchange & Verify
              </h2>
              <button onClick={() => setShowTokenExchange(false)} className="text-gray-400 hover:text-gray-600" aria-label="Close">
                <X className="h-5 w-5" />
              </button>
            </div>

            {/* Exchange Section */}
            <div className="space-y-3">
              <h3 className="text-sm font-semibold text-gray-700 dark:text-gray-300">Exchange Token</h3>
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500">Agent ID</label>
                <input aria-label="agent-uuid" value={tokenForm.agent_id} onChange={(e) => setTokenForm({ ...tokenForm, agent_id: e.target.value })} className={inputCls} placeholder="agent-uuid" />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500">Subject Token (user JWT)</label>
                <textarea aria-label="eyJhbGciOi..." value={tokenForm.subject_token} onChange={(e) => setTokenForm({ ...tokenForm, subject_token: e.target.value })} className={inputCls + " font-mono"} rows={3} placeholder="eyJhbGciOi..." />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="mb-1 block text-xs font-medium text-gray-500">Scope</label>
                  <input aria-label="read:users" value={tokenForm.scope} onChange={(e) => setTokenForm({ ...tokenForm, scope: e.target.value })} className={inputCls} placeholder="read:users" />
                </div>
                <div>
                  <label className="mb-1 block text-xs font-medium text-gray-500">Audience (optional)</label>
                  <input aria-label="mcp-server" value={tokenForm.audience} onChange={(e) => setTokenForm({ ...tokenForm, audience: e.target.value })} className={inputCls} placeholder="mcp-server" />
                </div>
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500">MCP Servers (optional)</label>
                <input aria-label="github-server" value={tokenForm.mcp_servers} onChange={(e) => setTokenForm({ ...tokenForm, mcp_servers: e.target.value })} className={inputCls} placeholder="github-server" />
              </div>
              <button onClick={handleTokenExchange} disabled={exchanging} className={btnPrimary}>
                {exchanging ? <Loader2 className="h-4 w-4 animate-spin" /> : <Key className="h-4 w-4" />} Exchange Token
              </button>

              {/* Result */}
              {exchangedToken && (
                <div className="rounded-lg border border-green-200 bg-green-50 p-3 dark:border-green-800 dark:bg-green-950">
                  <p className="mb-1 text-xs font-medium text-green-700 dark:text-green-400">Agent Token (copy now):</p>
                  <div className="flex items-center gap-2">
                    <code className="flex-1 break-all rounded bg-white px-2 py-1 font-mono text-xs dark:bg-gray-800">{exchangedToken}</code>
                    <button onClick={() => navigator.clipboard.writeText(exchangedToken)} className="rounded border border-gray-300 p-1.5 dark:border-gray-600" title="Copy">
                      <Copy className="h-3.5 w-3.5" />
                    </button>
                  </div>
                </div>
              )}
            </div>

            {/* Divider */}
            <div className="my-4 border-t border-gray-200 dark:border-gray-700" />

            {/* Verify Section */}
            <div className="space-y-3">
              <h3 className="text-sm font-semibold text-gray-700 dark:text-gray-300">Verify Token</h3>
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500">Token to verify</label>
                <textarea aria-label="eyJhbGciOi..." value={verifyInput} onChange={(e) => setVerifyInput(e.target.value)} className={inputCls + " font-mono"} rows={2} placeholder="eyJhbGciOi..." />
              </div>
              <button onClick={handleVerify} disabled={verifying || !verifyInput} className={btnGhost}>
                {verifying ? <Loader2 className="mr-1 inline h-3.5 w-3.5 animate-spin" /> : <Shield className="mr-1 inline h-3.5 w-3.5" />} Verify
              </button>
              {verifyResult && (
                <pre className="rounded-lg bg-gray-900 p-3 text-xs text-green-400 overflow-x-auto">{verifyResult}</pre>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

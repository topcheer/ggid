import { useState, useCallback } from "react";

export interface ScopeNarrowingRule {
  parent_scope: string;
  allowed_child_scopes: string[];
}

export interface PerAgentTrust {
  agent_id: string;
  agent_name: string;
  trust_level: "low" | "medium" | "high";
}

export interface AgentIdentityDelegationConfig {
  max_delegation_depth: number;
  scope_narrowing_rules: ScopeNarrowingRule[];
  token_exchange_policy: "strict" | "permissive";
  per_agent_trust_level: PerAgentTrust[];
  revocation_propagation: boolean;
  delegation_chain_preview: string;
}

export function useAgentIdentityDelegationConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<AgentIdentityDelegationConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try { const res = await fetch(`${baseUrl}/api/v1/settings/agent-identity-delegation-config`); if (!res.ok) throw new Error(`HTTP ${res.status}`); setConfig(await res.json()); }
    catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); } finally { setLoading(false); }
  }, [baseUrl]);
  const updateConfig = useCallback(async (patch: Partial<AgentIdentityDelegationConfig>) => {
    setLoading(true); setError(null);
    try { const res = await fetch(`${baseUrl}/api/v1/settings/agent-identity-delegation-config`, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch) }); if (!res.ok) throw new Error(`HTTP ${res.status}`); const data = await res.json(); setConfig(data); return data; }
    catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; } finally { setLoading(false); }
  }, [baseUrl]);
  return { config, loading, error, fetchConfig, updateConfig };
}

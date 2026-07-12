import { useState, useCallback } from "react";

export interface ImpactAssessment {
  component: string;
  current_alg: string;
  target_alg: string;
  migration_difficulty: "easy" | "medium" | "hard";
}

export interface PostQuantumMigrationConfig {
  current_algs: string[];
  target_algs: string[];
  hybrid_mode: boolean;
  migration_timeline_weeks: number;
  impact_assessment: ImpactAssessment[];
  test_toggle: boolean;
}

export function usePostQuantumMigrationConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<PostQuantumMigrationConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try { const res = await fetch(`${baseUrl}/api/v1/settings/post-quantum-migration-config`); if (!res.ok) throw new Error(`HTTP ${res.status}`); setConfig(await res.json()); }
    catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); } finally { setLoading(false); }
  }, [baseUrl]);
  const updateConfig = useCallback(async (patch: Partial<PostQuantumMigrationConfig>) => {
    setLoading(true); setError(null);
    try { const res = await fetch(`${baseUrl}/api/v1/settings/post-quantum-migration-config`, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch) }); if (!res.ok) throw new Error(`HTTP ${res.status}`); const data = await res.json(); setConfig(data); return data; }
    catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; } finally { setLoading(false); }
  }, [baseUrl]);
  return { config, loading, error, fetchConfig, updateConfig };
}

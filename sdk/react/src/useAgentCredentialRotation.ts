import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface RotationEntry {
  agent_id: string;
  agent_name: string;
  current_key_age_days: number;
  rotation_due_days: number;
  auto_rotate: boolean;
}

export interface RotationHistoryEntry {
  agent_name: string;
  rotated_at: string;
  rotated_by: string;
  key_thumbprint_before: string;
  key_thumbprint_after: string;
}

export interface AgentCredentialRotationData {
  rotation_schedule: RotationEntry[];
  rotation_history: RotationHistoryEntry[];
  compliance_pct: number;
}

export function useAgentCredentialRotation() {
  const [data, setData] = useState<AgentCredentialRotationData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      // Try real API first
      let res: Response | null = null;
      try {
        res = await fetch("/api/v1/data", {
          headers: { "Content-Type": "application/json" },
        });
      } catch { res = null; }
      
      if (res?.ok) {
        const realData = await res.json();
        setData(realData);
        setIsDemoData(false);
        return;
      }
      
      // Fallback: empty demo data (no dangerous flags)
      setIsDemoData(true);
      setData({
        rotation_schedule: [
          { agent_id: "agent-001", agent_name: "CI/CD Bot", current_key_age_days: 30, rotation_due_days: 60, auto_rotate: false },
          { agent_id: "agent-002", agent_name: "Monitoring Agent", current_key_age_days: 85, rotation_due_days: 5, auto_rotate: false },
          { agent_id: "agent-003", agent_name: "Data Pipeline", current_key_age_days: 95, rotation_due_days: -5, auto_rotate: false },
          { agent_id: "agent-004", agent_name: "Security Scanner", current_key_age_days: 15, rotation_due_days: 75, auto_rotate: false },
          { agent_id: "agent-005", agent_name: "Legacy Integration", current_key_age_days: 180, rotation_due_days: -90, auto_rotate: false },
        ],
        rotation_history: [
          { agent_name: "CI/CD Bot", rotated_at: "30d ago", rotated_by: "auto-rotation", key_thumbprint_before: "a1b2c3d4e5f6a7b8", key_thumbprint_after: "f8e7d6c5b4a39281" },
          { agent_name: "Security Scanner", rotated_at: "15d ago", rotated_by: "admin@ggid.dev", key_thumbprint_before: "b2c3d4e5f6a7b8c1", key_thumbprint_after: "e7d6c5b4a3928170" },
          { agent_name: "Monitoring Agent", rotated_at: "85d ago", rotated_by: "auto-rotation", key_thumbprint_before: "c3d4e5f6a7b8c1d2", key_thumbprint_after: "d6c5b4a3928170f1" },
        ],
        compliance_pct: 60,
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  const rotateNow = useCallback(async (agentId: string) => {
    console.log("Rotating credential for:", agentId);
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, rotateNow };
}

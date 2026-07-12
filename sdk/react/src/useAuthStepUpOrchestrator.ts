import { useState, useCallback, useEffect } from "react";

export interface StepUpFlow {
  trigger_action: string;
  required_factors: string[];
  max_attempts: number;
  timeout_seconds: number;
  success_rate_pct: number;
}

export interface ActiveChallenge {
  id: string;
  user: string;
  challenge_type: string;
  started_at: string;
  expires_in_seconds: number;
}

export interface AuthStepUpOrchestratorData {
  step_up_flows: StepUpFlow[];
  active_challenges: ActiveChallenge[];
  avg_success_rate_pct: number;
  challenge_timeout_policy: string;
}

export function useAuthStepUpOrchestrator() {
  const [data, setData] = useState<AuthStepUpOrchestratorData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        step_up_flows: [
          { trigger_action: "sensitive_action", required_factors: ["otp"], max_attempts: 3, timeout_seconds: 120, success_rate_pct: 94 },
          { trigger_action: "policy_change", required_factors: ["webauthn"], max_attempts: 2, timeout_seconds: 60, success_rate_pct: 88 },
          { trigger_action: "admin_access", required_factors: ["webauthn", "otp"], max_attempts: 2, timeout_seconds: 90, success_rate_pct: 91 },
          { trigger_action: "data_export", required_factors: ["otp"], max_attempts: 3, timeout_seconds: 180, success_rate_pct: 96 },
        ],
        active_challenges: [
          { id: "ch-1", user: "alice.chen", challenge_type: "webauthn", started_at: "30s ago", expires_in_seconds: 30 },
          { id: "ch-2", user: "bob.martinez", challenge_type: "otp", started_at: "1m ago", expires_in_seconds: 60 },
        ],
        avg_success_rate_pct: 92,
        challenge_timeout_policy: "expire_and_retry",
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData };
}

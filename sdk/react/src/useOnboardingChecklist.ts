import { useState, useCallback } from "react";

export interface ChecklistStep {
  key: string;
  label: string;
  description: string;
  completed: boolean;
  completed_at: string | null;
}

export interface Checklist {
  client_id: string;
  client_name: string;
  steps: ChecklistStep[];
  completion_pct: number;
}

export function useOnboardingChecklist(baseUrl: string = "") {
  const [checklist, setChecklist] = useState<Checklist | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchChecklist = useCallback(async (clientId: string) => {
    if (!clientId) return;
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/oauth/clients/${clientId}/onboarding`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data: Checklist = await res.json();
      setChecklist(data);
    } catch (e: any) {
      setError(e.message);
      setChecklist(null);
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  const checkStep = useCallback(async (clientId: string, stepKey: string, completed: boolean) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/oauth/clients/${clientId}/onboarding/${stepKey}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ completed }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setChecklist((prev) => prev ? {
        ...prev,
        steps: prev.steps.map((s) => s.key === stepKey ? { ...s, completed, completed_at: completed ? new Date().toISOString() : null } : s),
        completion_pct: Math.round((prev.steps.filter((s) => s.key === stepKey ? completed : s.completed).length / prev.steps.length) * 100),
      } : null);
      return true;
    } catch (e: any) {
      setError(e.message);
      return false;
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  return { checklist, loading, error, fetchChecklist, checkStep };
}

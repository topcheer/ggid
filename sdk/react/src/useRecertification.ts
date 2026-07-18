import { useState, useCallback } from "react";

export interface RecertDecision {
  user_id: string;
  decision: "keep" | "remove" | "modify";
  comment: string;
}

export interface RecertUser {
  user_id: string;
  username: string;
  email: string;
  current_roles: string[];
  last_login: string;
  decision: "pending" | "keep" | "remove" | "modify";
  comment: string;
}

export function useRecertification(baseUrl: string = "") {
  const [users, setUsers] = useState<RecertUser[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchUsers = useCallback(async (teamId: string) => {
    if (!teamId) return;
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/recertification/teams/${teamId}/users`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      setUsers(data.users || data || []);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  const submitDecisions = useCallback(async (teamId: string, decisions: RecertDecision[]) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/recertification/teams/${teamId}/submit`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ decisions }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setUsers((prev) => prev.map((u: any) => u.decision !== "pending" ? { ...u, decision: "pending", comment: "" } : u));
      return true;
    } catch (e: any) {
      setError(e.message);
      return false;
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  return { users, loading, error, fetchUsers, submitDecisions };
}

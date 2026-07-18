import { useState, useEffect, useCallback } from "react";

export interface PermissionGrant {
  permission: string;
  resource: string;
  last_used: string | null;
  usage_count: number;
}

export interface RoleRecommendation {
  current_role: string;
  recommended_role: string;
  unused_permissions: string[];
  over_granted: string[];
  risk_level: "low" | "medium" | "high";
  confidence: number;
}

export interface UserAnalysis {
  user_id: string;
  username: string;
  email: string;
  permissions: PermissionGrant[];
  unused_count: number;
  over_granted_count: number;
  recommendations: RoleRecommendation[];
}

export interface RoleMiningResult {
  users: UserAnalysis[];
}

export function useRoleMining(baseUrl: string = "") {
  const [analysis, setAnalysis] = useState<UserAnalysis[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchAnalysis = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/role-mining/analysis`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data: RoleMiningResult = await res.json();
      setAnalysis(data.users || data as any || []);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  const applyRecommendation = useCallback(async (userId: string, currentRole: string, recommendedRole: string) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/role-mining/apply`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ user_id: userId, current_role: currentRole, recommended_role: recommendedRole }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setAnalysis((prev: any) => prev.map((u: any) =>
        u.user_id === userId
          ? { ...u, recommendations: u.recommendations.filter((r: any) => r.current_role !== currentRole || r.recommended_role !== recommendedRole) }
          : u
      ));
      return true;
    } catch (e: any) {
      setError(e.message);
      return false;
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  useEffect(() => {
    fetchAnalysis();
  }, [fetchAnalysis]);

  return { analysis, loading, error, fetchAnalysis, applyRecommendation };
}

import { useState, useCallback } from "react";
export interface ConditionGroup { id: string; logic: "AND" | "OR"; conditions: { id: string; attribute: string; operator: string; value: string }[]; }
export function useConditionBuilder(baseUrl: string = "") {
  const [groups, setGroups] = useState<ConditionGroup[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const saveConditions = useCallback(async (policyId: string, newGroups: ConditionGroup[]) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/policy/" + policyId + "/conditions", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(newGroups) }); if (!res.ok) throw new Error("HTTP " + res.status); setGroups(newGroups); return true; } catch (e: any) { setError(e.message); return false; } finally { setLoading(false); } }, [baseUrl]);
  const validate = useCallback(async (newGroups: ConditionGroup[]) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/policy/conditions/validate", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(newGroups) }); if (!res.ok) throw new Error("HTTP " + res.status); return true; } catch (e: any) { setError(e.message); return false; } finally { setLoading(false); } }, [baseUrl]);
  return { groups, loading, error, saveConditions, validate };
}

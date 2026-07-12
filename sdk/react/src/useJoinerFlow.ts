import { useState, useCallback } from "react";

export interface JoinerData {
  employee_id: string;
  start_date: string;
  department: string;
  role_templates: string[];
  provision_apps: { id: string; name: string; auto: boolean }[];
  preboarding: { id: string; label: string; done: boolean }[];
  status: string;
}

export function useJoinerFlow(baseUrl: string = "") {
  const [data, setData] = useState<JoinerData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const submit = useCallback(async (payload: JoinerData) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/identity/joiner-flow`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(payload) });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setData(await res.json()); return true;
    } catch (e: any) { setError(e.message); return false; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, submit };
}

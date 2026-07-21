import { useState, useCallback } from "react";

export interface LineageNode {
  id: string;
  type: "source" | "creator" | "modifier" | "consumer" | "access";
  label: string;
  timestamp: string;
  metadata?: Record<string, string>;
}

export interface AccessEvent {
  actor: string;
  action: string;
  timestamp: string;
  ip: string;
}

export interface Consumer {
  name: string;
  type: string;
  access_level: string;
}

export interface LineageData {
  resource_id: string;
  resource_type: string;
  resource_name: string;
  nodes: LineageNode[];
  created_by: string;
  created_at: string;
  last_modified_by: string;
  last_modified_at: string;
  access_events: AccessEvent[];
  downstream_consumers: Consumer[];
}

export function useDataLineage(baseUrl: string = "") {
  const [lineage, setLineage] = useState<LineageData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchLineage = useCallback(async (resourceId: string) => {
    if (!resourceId) return;
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/audit/data-lineage?resource=${encodeURIComponent(resourceId)}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data: LineageData = await res.json();
      setLineage(data);
    } catch (e: any) {
      setError(e.message);
      setLineage(null);
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  return { lineage, loading, error, fetchLineage };
}

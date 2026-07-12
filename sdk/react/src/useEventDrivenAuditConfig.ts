import { useState, useCallback } from "react";

export interface StreamConfig {
  name: string;
  subjects: string[];
  retention: "limits" | "interest" | "work_queue";
}

export interface EventDrivenAuditConfig {
  stream_config: StreamConfig;
  consumer_pattern: "competing" | "shared" | "fanout";
  deduplication_window_ms: number;
  ordering: "per_tenant" | "global";
  backpressure_strategy: "block" | "drop" | "buffer";
  replay_enabled: boolean;
}

export function useEventDrivenAuditConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<EventDrivenAuditConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/event-driven-audit-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<EventDrivenAuditConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/event-driven-audit-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}

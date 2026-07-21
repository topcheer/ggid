import { useState, useCallback } from "react";

export interface EventTypeEntry {
  name: string;
  payload_schema_preview: string;
  subscribers_count: number;
  delivery_guarantee: "at_least_once" | "at_most_once" | "exactly_once";
}

export interface DeliveryStats {
  total_sent: number;
  delivered: number;
  failed: number;
  retrying: number;
}

export interface WebhookEventCatalogConfig {
  event_types: EventTypeEntry[];
  per_event_retry_policy: { event: string; max_attempts: number; backoff_seconds: number }[];
  sample_handlers: string[];
  delivery_stats: DeliveryStats;
}

export function useWebhookEventCatalogConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<WebhookEventCatalogConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/webhook-event-catalog-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<WebhookEventCatalogConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/webhook-event-catalog-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  const testEvent = useCallback(async (eventType: string) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/webhook-event-catalog-config/test`, {
        method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ event_type: eventType }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      return await res.json();
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig, testEvent };
}

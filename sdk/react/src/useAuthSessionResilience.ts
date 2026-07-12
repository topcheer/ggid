import { useState, useCallback, useEffect } from "react";

export interface ConnectionPoolStatus {
  active: number;
  idle: number;
  max: number;
}

export interface SessionFailoverConfig {
  primary_redis: string;
  fallback_memory: string;
}

export interface DegradedIndicator {
  indicator: string;
  active: boolean;
}

export interface AuthSessionResilienceData {
  connection_pool_status: ConnectionPoolStatus;
  session_failover_config: SessionFailoverConfig;
  grace_period_during_outage: number;
  offline_token_validation: boolean;
  degraded_mode_indicators: DegradedIndicator[];
}

export function useAuthSessionResilience() {
  const [data, setData] = useState<AuthSessionResilienceData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        connection_pool_status: {
          active: 42,
          idle: 18,
          max: 100,
        },
        session_failover_config: {
          primary_redis: "redis-cluster-prod:6379",
          fallback_memory: "in-process LRU cache",
        },
        grace_period_during_outage: 300,
        offline_token_validation: true,
        degraded_mode_indicators: [
          { indicator: "Redis connection pool degraded", active: false },
          { indicator: "Fallback memory store active", active: false },
          { indicator: "Session sync delayed", active: false },
          { indicator: "Token revocation propagation delayed", active: true },
          { indicator: "High session read latency (>100ms)", active: false },
        ],
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  const testRecovery = useCallback(async () => {
    console.log("Running session recovery test");
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { data, loading, error, refresh: fetchData, testRecovery };
}

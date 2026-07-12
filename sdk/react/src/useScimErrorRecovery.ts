import { useState, useCallback, useEffect } from "react";

export interface ScimErrorEntry {
  id: string;
  timestamp: string;
  operation: string;
  target_app: string;
  error_type: string;
  retry_count: number;
  status: string;
}

export interface ErrorPattern {
  error_type: string;
  count: number;
}

export interface AutoRetryConfig {
  max_retries: number;
  backoff_seconds: number;
  manual_override: boolean;
}

export interface ScimErrorRecoveryData {
  error_queue: ScimErrorEntry[];
  error_patterns: ErrorPattern[];
  auto_retry_config: AutoRetryConfig;
}

export function useScimErrorRecovery() {
  const [data, setData] = useState<ScimErrorRecoveryData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        error_queue: [
          { id: "e1", timestamp: "5m ago", operation: "User.Create", target_app: "Salesforce", error_type: "duplicate_value", retry_count: 2, status: "retrying" },
          { id: "e2", timestamp: "15m ago", operation: "Group.Update", target_app: "Slack", error_type: "rate_limit_exceeded", retry_count: 1, status: "pending" },
          { id: "e3", timestamp: "1h ago", operation: "User.Deactivate", target_app: "Google Workspace", error_type: "permission_denied", retry_count: 5, status: "failed" },
          { id: "e4", timestamp: "2h ago", operation: "User.Update", target_app: "Slack", error_type: "resource_not_found", retry_count: 3, status: "resolved" },
        ],
        error_patterns: [
          { error_type: "rate_limit_exceeded", count: 45 },
          { error_type: "duplicate_value", count: 23 },
          { error_type: "permission_denied", count: 12 },
        ],
        auto_retry_config: { max_retries: 5, backoff_seconds: 30, manual_override: true },
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);

  const retryError = useCallback((id: string) => { console.log("Retrying", id); }, []);
  const bulkRetry = useCallback(() => { console.log("Bulk retry"); }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, retryError, bulkRetry };
}

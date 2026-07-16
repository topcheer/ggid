import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface AppBinding {
  app: string;
  method: string;
  enforce: boolean;
}

export interface CrossDeviceTransfer {
  enabled: boolean;
  transfer_window_seconds: number;
  verification_required: boolean;
  max_per_day: number;
}

export interface AuthSessionBindingData {
  binding_method: string;
  per_application_binding: AppBinding[];
  binding_rotation_policy: string;
  session_hijack_protection: boolean;
  cross_device_session_transfer: CrossDeviceTransfer;
  fallback_method: string;
}

export function useAuthSessionBinding() {
  const [data, setData] = useState<AuthSessionBindingData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      // Try real API first
      let res: Response | null = null;
      try {
        res = await fetch("/api/v1/data", {
          headers: { "Content-Type": "application/json" },
        });
      } catch { res = null; }
      
      if (res?.ok) {
        const realData = await res.json();
        setData(realData);
        setIsDemoData(false);
        return;
      }
      
      // Fallback: empty demo data (no dangerous flags)
      setIsDemoData(true);
      setData({
        binding_method: "DPoP",
        per_application_binding: [
          { app: "Web Dashboard", method: "cookie", enforce: true },
          { app: "Mobile App", method: "DPoP", enforce: true },
          { app: "API Gateway", method: "mTLS", enforce: true },
          { app: "Service-to-Service", method: "bearer", enforce: false },
        ],
        binding_rotation_policy: "every 90 days",
        session_hijack_protection: true,
        cross_device_session_transfer: {
          enabled: true,
          transfer_window_seconds: 120,
          verification_required: true,
          max_per_day: 5,
        },
        fallback_method: "cookie",
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { data, loading, error, refresh: fetchData, isDemoData };
}

import { useState, useCallback, useEffect } from "react";

export interface BrandingConfigData {
  logo_url: string;
  primary_color: string;
  secondary_color: string;
  custom_css: string;
  theme: string;
  custom_domain: string;
}

export function useBrandingConfig() {
  const [data, setData] = useState<BrandingConfigData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try { await new Promise((r) => setTimeout(r, 400));
      setData({
        logo_url: "/logo.png",
        primary_color: "#1e40af",
        secondary_color: "#3b82f6",
        custom_css: "/* Custom tenant styles */\n.navbar { border-radius: 8px; }",
        theme: "dark",
        custom_domain: "id.acme-corp.com",
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); } finally { setLoading(false); }
  }, []);
  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData };
}

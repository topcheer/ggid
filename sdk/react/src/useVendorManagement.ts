import { useState, useCallback } from "react";

export interface Vendor {
  id: string;
  name: string;
  service_type: string;
  risk_rating: "low" | "medium" | "high" | "critical";
  data_access_scope: string;
  contract_expiry: string;
  days_to_expiry: number;
  compliance_status: "compliant" | "pending" | "non_compliant";
  last_assessment: string;
}

export function useVendorManagement(baseUrl: string = "") {
  const [vendors, setVendors] = useState<Vendor[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchVendors = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/identity/vendors`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setVendors(data.vendors || data || []);
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { vendors, loading, error, fetchVendors };
}

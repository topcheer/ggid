import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface ResidencyRegion {
  region: string;
  allowed: boolean;
  encryption_required: boolean;
}

export interface TransferRule {
  source_region: string;
  destination_region: string;
  transfer_mechanism: string;
  data_types: string[];
}

export interface PendingTransfer {
  id: string;
  data_type: string;
  source: string;
  destination: string;
  status: string;
}

export interface SovereigntyViolation {
  violation_type: string;
  description: string;
  region: string;
  severity: string;
  detected_at: string;
}

export interface AuditDataSovereigntyData {
  data_residency_regions: ResidencyRegion[];
  cross_border_transfer_rules: TransferRule[];
  gdpr_article_45: boolean;
  gdpr_article_49: boolean;
  data_localization_status: string;
  pending_transfers: PendingTransfer[];
  sovereignty_violations: SovereigntyViolation[];
}

export function useAuditDataSovereignty() {
  const [data, setData] = useState<AuditDataSovereigntyData | null>(null);
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
        data_residency_regions: [
          { region: "EU-West (Frankfurt)", allowed: true, encryption_required: true },
          { region: "US-East (Virginia)", allowed: true, encryption_required: false },
          { region: "AP-Southeast (Singapore)", allowed: true, encryption_required: true },
          { region: "CN-North (Beijing)", allowed: false, encryption_required: true },
        ],
        cross_border_transfer_rules: [
          { source_region: "EU-West", destination_region: "US-East", transfer_mechanism: "SCCs (Standard Contractual Clauses)", data_types: ["audit_logs", "user_profiles"] },
          { source_region: "US-East", destination_region: "AP-Southeast", transfer_mechanism: "Binding Corporate Rules", data_types: ["audit_logs"] },
        ],
        gdpr_article_45: true,
        gdpr_article_49: false,
        data_localization_status: "compliant",
        pending_transfers: [
          { id: "pt-1", data_type: "audit_logs", source: "EU-West", destination: "US-East", status: "awaiting_approval" },
        ],
        sovereignty_violations: [
          { violation_type: "unauthorized_transfer", description: "Data transferred to CN-North without approval", region: "CN-North", severity: "critical", detected_at: "3h ago" },
        ],
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

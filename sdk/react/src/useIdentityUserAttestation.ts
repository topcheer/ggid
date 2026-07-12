import { useState, useCallback, useEffect } from "react";

export interface AttestationUser {
  user_id: string;
  user_name: string;
  attestation_status: "pending" | "attested" | "revoked";
  last_attested_at: string | null;
  attested_by: string | null;
  permissions_at_time: number;
}

export interface AttestationCampaign {
  id: string;
  name: string;
  pending_count: number;
  attested_count: number;
  user_list: AttestationUser[];
}

export interface IdentityUserAttestationData {
  campaigns: AttestationCampaign[];
  overdue_attestations: number;
  auto_revoke_unattested_days: number;
}

export function useIdentityUserAttestation() {
  const [data, setData] = useState<IdentityUserAttestationData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        campaigns: [
          {
            id: "camp-q1-2026",
            name: "Q1 2026 Access Review",
            pending_count: 8,
            attested_count: 22,
            user_list: [
              { user_id: "u1", user_name: "Alice Chen", attestation_status: "attested", last_attested_at: "2026-01-10", attested_by: "manager.bob", permissions_at_time: 12 },
              { user_id: "u2", user_name: "Bob Martinez", attestation_status: "pending", last_attested_at: null, attested_by: null, permissions_at_time: 8 },
              { user_id: "u3", user_name: "Carol Jones", attestation_status: "pending", last_attested_at: null, attested_by: null, permissions_at_time: 15 },
              { user_id: "u4", user_name: "Dave Wilson", attestation_status: "attested", last_attested_at: "2026-01-08", attested_by: "manager.bob", permissions_at_time: 6 },
              { user_id: "u5", user_name: "Eve Brown", attestation_status: "revoked", last_attested_at: "2025-10-15", attested_by: "system.auto", permissions_at_time: 0 },
              { user_id: "u6", user_name: "Frank Lee", attestation_status: "pending", last_attested_at: null, attested_by: null, permissions_at_time: 20 },
            ],
          },
          {
            id: "camp-q4-2025",
            name: "Q4 2025 Access Review",
            pending_count: 2,
            attested_count: 30,
            user_list: [
              { user_id: "u7", user_name: "Grace Kim", attestation_status: "attested", last_attested_at: "2025-10-01", attested_by: "manager.alice", permissions_at_time: 10 },
              { user_id: "u8", user_name: "Henry Chen", attestation_status: "pending", last_attested_at: null, attested_by: null, permissions_at_time: 5 },
            ],
          },
        ],
        overdue_attestations: 3,
        auto_revoke_unattested_days: 30,
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  const bulkAttest = useCallback(async (campaignId: string) => {
    console.log("Bulk attesting campaign:", campaignId);
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { data, loading, error, refresh: fetchData, bulkAttest };
}

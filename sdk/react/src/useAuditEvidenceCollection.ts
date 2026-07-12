import { useState, useCallback, useEffect } from "react";

export interface EvidenceRequest {
  framework: string;
  control_id: string;
  requested_by: string;
  deadline: string;
  status: string;
}

export interface EvidenceFile {
  file_name: string;
  hash: string;
  uploaded_by: string;
  uploaded_at: string;
  verified: boolean;
}

export interface CollectionProgress {
  framework: string;
  progress_pct: number;
  collected: number;
  total: number;
}

export interface AuditEvidenceCollectionData {
  evidence_requests: EvidenceRequest[];
  evidence_repository: EvidenceFile[];
  collection_progress: CollectionProgress[];
}

export function useAuditEvidenceCollection() {
  const [data, setData] = useState<AuditEvidenceCollectionData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        evidence_requests: [
          { framework: "SOC2", control_id: "CC6.1", requested_by: "audit-lead@ggid.dev", deadline: "2024-03-15", status: "collected" },
          { framework: "SOC2", control_id: "CC7.2", requested_by: "audit-lead@ggid.dev", deadline: "2024-03-20", status: "pending" },
          { framework: "HIPAA", control_id: "164.312(a)(1)", requested_by: "compliance@ggid.dev", deadline: "2024-03-10", status: "overdue" },
          { framework: "ISO27001", control_id: "A.9.4.2", requested_by: "iso-lead@ggid.dev", deadline: "2024-03-25", status: "pending" },
          { framework: "GDPR", control_id: "Art.30", requested_by: "dpo@ggid.dev", deadline: "2024-03-18", status: "collected" },
        ],
        evidence_repository: [
          { file_name: "SOC2_CC6.1_access_controls_2024Q1.pdf", hash: "sha256:a1b2c3d4e5f6789abcdef0123456789abcdef0123456789abcdef0123456789", uploaded_by: "audit-lead@ggid.dev", uploaded_at: "2d ago", verified: true },
          { file_name: "GDPR_Art30_records_processing.pdf", hash: "sha256:b2c3d4e5f67890abcdef123456789abcdef0123456789abcdef01234567890a", uploaded_by: "dpo@ggid.dev", uploaded_at: "3d ago", verified: true },
          { file_name: "ISO27001_A.9.4.2_access_matrix.xlsx", hash: "sha256:c3d4e5f67890abcdef23456789abcdef0123456789abcdef01234567890abcd", uploaded_by: "iso-lead@ggid.dev", uploaded_at: "1h ago", verified: false },
        ],
        collection_progress: [
          { framework: "SOC2", progress_pct: 75, collected: 18, total: 24 },
          { framework: "HIPAA", progress_pct: 60, collected: 12, total: 20 },
          { framework: "ISO27001", progress_pct: 45, collected: 54, total: 120 },
          { framework: "GDPR", progress_pct: 90, collected: 9, total: 10 },
          { framework: "PCI-DSS", progress_pct: 30, collected: 36, total: 120 },
        ],
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData };
}

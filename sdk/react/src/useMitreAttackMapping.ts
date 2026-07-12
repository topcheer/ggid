import { useState, useCallback, useEffect } from "react";

export interface MitreTechnique {
  t_id: string;
  name: string;
  tactic: string;
  detection_status: string;
}

export interface MitreAttackMappingData {
  techniques: MitreTechnique[];
}

export function useMitreAttackMapping() {
  const [data, setData] = useState<MitreAttackMappingData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        techniques: [
          { t_id: "T1595.002", name: "Active Scanning: Vulnerability Scanning", tactic: "reconnaissance", detection_status: "detected" },
          { t_id: "T1592.004", name: "Gather Victim Host Info: Client Config", tactic: "reconnaissance", detection_status: "mitigated" },
          { t_id: "T1110.004", name: "Credential Stuffing", tactic: "credential_access", detection_status: "detected" },
          { t_id: "T1550.002", name: "Pass the Hash", tactic: "lateral_movement", detection_status: "detected" },
          { t_id: "T1558.001", name: "Golden Ticket", tactic: "lateral_movement", detection_status: "mitigated" },
          { t_id: "T1528", name: "Steal Application Access Token", tactic: "credential_access", detection_status: "detected" },
          { t_id: "T1530", name: "Data from Cloud Storage", tactic: "exfiltration", detection_status: "unknown" },
          { t_id: "T1537", name: "Transfer Data to Cloud Account", tactic: "exfiltration", detection_status: "unknown" },
        ],
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData };
}

import { useState, useCallback, useEffect } from "react";

export interface IncidentStep {
  name: string;
  status: string;
}

export interface ActiveIncident {
  incident_id: string;
  type: string;
  severity: string;
  status: string;
  assigned_to: string;
  sla_countdown: string;
  steps: IncidentStep[];
}

export interface Playbook {
  incident_type: string;
  severity: string;
  steps_count: number;
  automated_actions_count: number;
  escalation_chain: string[];
}

export interface PostMortemTemplate {
  template_name: string;
  sections_count: number;
}

export interface IncidentResponsePlaybookData {
  active_incidents: ActiveIncident[];
  playbook_library: Playbook[];
  post_mortem_templates: PostMortemTemplate[];
}

export function useIncidentResponsePlaybook() {
  const [data, setData] = useState<IncidentResponsePlaybookData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        active_incidents: [
          { incident_id: "INC-2024-007", type: "Credential Stuffing", severity: "critical", status: "active", assigned_to: "sec-team-alpha", sla_countdown: "2h 15m", steps: [
            { name: "Detect", status: "done" }, { name: "Contain", status: "done" }, { name: "Investigate", status: "active" }, { name: "Eradicate", status: "pending" }, { name: "Recover", status: "pending" },
          ] },
        ],
        playbook_library: [
          { incident_type: "Credential Stuffing", severity: "critical", steps_count: 8, automated_actions_count: 5, escalation_chain: ["on-call", "sec-lead", "ciso"] },
          { incident_type: "Account Takeover", severity: "critical", steps_count: 10, automated_actions_count: 4, escalation_chain: ["on-call", "sec-lead", "ciso", "legal"] },
          { incident_type: "Insider Threat", severity: "high", steps_count: 12, automated_actions_count: 2, escalation_chain: ["sec-lead", "hr", "ciso", "legal"] },
          { incident_type: "Malware Detection", severity: "high", steps_count: 9, automated_actions_count: 6, escalation_chain: ["on-call", "it-ops", "sec-lead"] },
        ],
        post_mortem_templates: [
          { template_name: "Security Incident - Standard", sections_count: 7 },
          { template_name: "Data Breach - Detailed", sections_count: 12 },
          { template_name: "Insider Threat - Confidential", sections_count: 9 },
        ],
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData };
}

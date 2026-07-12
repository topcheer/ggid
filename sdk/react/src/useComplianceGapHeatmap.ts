import { useState, useCallback, useEffect } from "react";

export interface HeatmapCell {
  status: string;
  controls_count: number;
}

export interface DrillDownItem {
  control: string;
  requirement: string;
  current_state: string;
  status: string;
  remediation: string;
}

export interface ComplianceGapHeatmapData {
  frameworks: string[];
  control_categories: string[];
  heatmap: Record<string, HeatmapCell>;
  drill_down: Record<string, DrillDownItem[]>;
}

export function useComplianceGapHeatmap() {
  const [data, setData] = useState<ComplianceGapHeatmapData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      const frameworks = ["SOC2", "HIPAA", "ISO27001", "GDPR", "PCI-DSS"];
      const categories = ["Access Control", "Audit Logging", "Encryption", "Incident Response", "Data Protection"];
      const statuses = ["compliant", "partial", "gap", "not_applicable"];
      const heatmap: Record<string, HeatmapCell> = {};
      const drill_down: Record<string, DrillDownItem[]> = {};

      frameworks.forEach((fw, fi) => {
        categories.forEach((cat, ci) => {
          const key = fw + ":" + cat;
          const statusIdx = (fi + ci) % 4;
          const status = statuses[statusIdx];
          const count = 3 + ((fi * 5 + ci) % 8);
          heatmap[key] = { status, controls_count: count };
          if (status !== "compliant" && status !== "not_applicable") {
            drill_down[key] = [
              { control: fw + "-" + cat + "-1", requirement: "Implement automated access reviews", current_state: "Manual quarterly reviews", status: status === "gap" ? "gap" : "partial", remediation: status === "gap" ? "Deploy automated review tooling" : "Increase frequency to monthly" },
              { control: fw + "-" + cat + "-2", requirement: "Centralized logging", current_state: "Partial - 3 of 7 services logging", status: "partial", remediation: "Add logging to remaining services" },
            ];
          }
        });
      });

      setData({ frameworks, control_categories: categories, heatmap, drill_down });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData };
}

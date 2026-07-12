import { useState, useCallback, useEffect } from "react";

export interface ImportLog {
  imported: number;
  skipped: number;
  errored: number;
}

export interface TemplateEntry {
  name: string;
  description: string;
  compatible: boolean;
}

export interface PolicyImportExportData {
  total_policies: number;
  import_log: ImportLog;
  template_gallery: TemplateEntry[];
}

export function usePolicyImportExport() {
  const [data, setData] = useState<PolicyImportExportData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        total_policies: 48,
        import_log: { imported: 12, skipped: 3, errored: 1 },
        template_gallery: [
          { name: "RBAC Starter Pack", description: "Basic role-based access control policies", compatible: true },
          { name: "Zero Trust Template", description: "Default-deny with explicit allow rules", compatible: true },
          { name: "SOC2 Compliance Set", description: "Policies mapped to SOC2 controls", compatible: true },
          { name: "Legacy v1.x Export", description: "Import from GGID v1.x", compatible: false },
        ],
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData };
}

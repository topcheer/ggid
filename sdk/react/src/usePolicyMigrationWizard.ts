import { useState, useCallback, useEffect } from "react";

export interface MappingEntry {
  source_rule: string;
  ggid_policy: string;
  confidence: number;
}

export interface ValidationReport {
  migrated: number;
  warnings: number;
  errors: number;
  unsupported_features: string[];
}

export interface MigrationRecord {
  id: string;
  source_system: string;
  policies_count: number;
  date: string;
  executed_by: string;
  status: string;
}

export interface PolicyMigrationWizardData {
  source_systems: string[];
  mapping_preview: MappingEntry[];
  validation_report: ValidationReport;
  migration_history: MigrationRecord[];
}

export function usePolicyMigrationWizard() {
  const [data, setData] = useState<PolicyMigrationWizardData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        source_systems: ["Keycloak", "Auth0", "Ping Identity", "Okta", "Azure AD"],
        mapping_preview: [
          { source_rule: "realm-admin-role", ggid_policy: "policy-admin-access", confidence: 0.95 },
          { source_rule: "user-role:developer", ggid_policy: "policy-developer-access", confidence: 0.92 },
          { source_rule: "ip-restrict-corp", ggid_policy: "policy-corp-network-only", confidence: 0.88 },
          { source_rule: "mfa-required-admins", ggid_policy: "policy-admin-webauthn", confidence: 0.85 },
          { source_rule: "session-timeout-30", ggid_policy: "policy-session-30m", confidence: 0.78 },
        ],
        validation_report: {
          migrated: 42,
          warnings: 8,
          errors: 2,
          unsupported_features: ["custom-auth-spi", "realm-theme"],
        },
        migration_history: [
          { id: "mig-003", source_system: "Auth0", policies_count: 15, date: "2025-07-01", executed_by: "admin@ggid.dev", status: "completed" },
          { id: "mig-002", source_system: "Keycloak", policies_count: 28, date: "2025-06-15", executed_by: "admin@ggid.dev", status: "completed" },
          { id: "mig-001", source_system: "Okta", policies_count: 8, date: "2025-06-01", executed_by: "migrator@ggid.dev", status: "failed" },
        ],
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  const executeMigration = useCallback(async () => {
    console.log("Executing migration");
  }, []);

  const rollback = useCallback(async () => {
    console.log("Rolling back migration");
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, executeMigration, rollback };
}

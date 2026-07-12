import { useState, useCallback, useEffect } from "react";

export interface ClassificationLevel {
  id: string;
  name: string;
  description: string;
  record_count: number;
  handling_rules: string[];
}

export interface AttributeMapping {
  attribute: string;
  classification: string;
}

export interface PiiInventoryItem {
  field: string;
  occurrences: number;
  classification: string;
  masked: boolean;
}

export interface DataClassificationPolicyData {
  levels: ClassificationLevel[];
  attribute_mapping: AttributeMapping[];
  pii_inventory: PiiInventoryItem[];
  auto_classify: {
    enabled: boolean;
    confidence_threshold: number;
    fields_classified_24h: number;
  };
}

export function useDataClassificationPolicy() {
  const [data, setData] = useState<DataClassificationPolicyData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        levels: [
          {
            id: "lvl-1",
            name: "public",
            description: "Freely shareable information",
            record_count: 125000,
            handling_rules: ["No restrictions", "Can be shared externally"],
          },
          {
            id: "lvl-2",
            name: "internal",
            description: "For internal use only",
            record_count: 89000,
            handling_rules: ["Internal access only", "Encrypt in transit", "Log access"],
          },
          {
            id: "lvl-3",
            name: "confidential",
            description: "Sensitive business data",
            record_count: 34000,
            handling_rules: ["Need-to-know basis", "Encrypt at rest and transit", "MFA required", "Audit trail mandatory"],
          },
          {
            id: "lvl-4",
            name: "restricted",
            description: "Highest sensitivity, legal/ regulatory",
            record_count: 5200,
            handling_rules: ["Explicit approval required", "Field-level encryption", "DLP scanning", "Quarterly access review"],
          },
        ],
        attribute_mapping: [
          { attribute: "email", classification: "internal" },
          { attribute: "phone_number", classification: "confidential" },
          { attribute: "ssn", classification: "restricted" },
          { attribute: "department", classification: "public" },
          { attribute: "salary", classification: "restricted" },
          { attribute: "address", classification: "confidential" },
        ],
        pii_inventory: [
          { field: "email", occurrences: 250000, classification: "internal", masked: false },
          { field: "ssn", occurrences: 250000, classification: "restricted", masked: true },
          { field: "phone_number", occurrences: 180000, classification: "confidential", masked: true },
          { field: "date_of_birth", occurrences: 250000, classification: "confidential", masked: true },
          { field: "home_address", occurrences: 150000, classification: "confidential", masked: false },
        ],
        auto_classify: {
          enabled: true,
          confidence_threshold: 0.85,
          fields_classified_24h: 342,
        },
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

  return { data, loading, error, refresh: fetchData };
}

import { useState, useCallback, useEffect } from "react";

export interface ProofHeader {
  typ: string;
  alg: string;
  jwk_thumbprint: string;
}

export interface ProofPayload {
  htm: string;
  htu: string;
  jti: string;
  ath: string;
  iat: number;
}

export interface DpopProof {
  valid: boolean;
  error_message?: string;
  header: ProofHeader;
  payload: ProofPayload;
  signature: string;
  key_binding_verified: boolean;
  key_binding_algorithm: string;
}

export interface ValidityStep {
  check: string;
  detail: string;
  passed: boolean;
}

export interface ProofError {
  code: string;
  severity: "error" | "warning";
  description: string;
  remediation?: string;
}

export interface OAuthDpopProofViewerData {
  proof: DpopProof;
  validity_timeline: ValidityStep[];
  error_analysis: ProofError[];
}

export function useOAuthDpopProofViewer() {
  const [data, setData] = useState<OAuthDpopProofViewerData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        proof: {
          valid: true,
          header: {
            typ: "dpop+jwt",
            alg: "ES256",
            jwk_thumbprint: "9nGq8p3vNwUMxkZqWtL2sYx7bD1cKu5rJaFmHnVeRqs=",
          },
          payload: {
            htm: "POST",
            htu: "https://idp.example.com/oauth/token",
            jti: "-wYQu9O9oSzZ3M8jKqP",
            ath: "czZmNGRlNjk4MzU2Nzc4NTQ0Njg=",
            iat: 1700000000,
          },
          signature: "MEUCIQDf3v2vZ8k7t5x1KpLm6rQ4nWxYbH0jF3aSdV6tNgIgCgVbZ7rN4mP8sK2xQwJ9fT3uM5oL1aY6dR8cN0w2s=",
          key_binding_verified: true,
          key_binding_algorithm: "ES256",
        },
        validity_timeline: [
          { check: "JWT Decoded", detail: "Header and payload parsed successfully", passed: true },
          { check: "typ Header Valid", detail: "typ is dpop+jwt as required", passed: true },
          { check: "alg Supported", detail: "ES256 is in the supported algorithms list", passed: true },
          { check: "htm Matches", detail: "HTTP method POST matches token endpoint", passed: true },
          { check: "htu Matches", detail: "URL matches expected token endpoint", passed: true },
          { check: "jti Unique", detail: "No replay detected for this jti", passed: true },
          { check: "ath Correct", detail: "Access token hash matches bound token", passed: true },
          { check: "iat Within Window", detail: "Proof created 12s ago, within 60s max age", passed: true },
          { check: "Signature Verified", detail: "ECDSA signature validated against public key", passed: true },
          { check: "Key Binding Confirmed", detail: "Proof key matches DPoP-bound token key", passed: true },
        ],
        error_analysis: [],
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

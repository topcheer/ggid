import { useState, useCallback, useEffect } from "react";

export interface TrustedCa {
  name: string;
  fingerprint: string;
  expiry: string;
  status: string;
}

export interface ClientMtls {
  client: string;
  required: boolean;
  cert_thumbprint_binding: string | null;
}

export interface OAuthMtlsConfigData {
  require_mtls: boolean;
  trusted_ca_certs: TrustedCa[];
  per_client_mtls: ClientMtls[];
  certificate_revocation_check: string;
  allow_self_signed: boolean;
  mtls_adoption_pct: number;
}

export function useOAuthMtlsConfig() {
  const [data, setData] = useState<OAuthMtlsConfigData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        require_mtls: true,
        trusted_ca_certs: [
          { name: "Internal PKI Root CA", fingerprint: "SHA256:A1B2C3D4E5F6...", expiry: "2028-12-31", status: "valid" },
          { name: "Let's Encrypt R3", fingerprint: "SHA256:G7H8I9J0K1L2...", expiry: "2025-09-15", status: "valid" },
          { name: "DigiCert Global Root", fingerprint: "SHA256:M3N4O5P6Q7R8...", expiry: "2031-04-18", status: "valid" },
        ],
        per_client_mtls: [
          { client: "client-api-003", required: true, cert_thumbprint_binding: "SHA256:X1Y2Z3..." },
          { client: "client-mobile-002", required: false, cert_thumbprint_binding: null },
          { client: "client-web-001", required: false, cert_thumbprint_binding: null },
        ],
        certificate_revocation_check: "OCSP",
        allow_self_signed: false,
        mtls_adoption_pct: 34,
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

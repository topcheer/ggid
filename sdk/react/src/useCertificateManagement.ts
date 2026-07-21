import { useState, useCallback } from "react";
export interface Cert { id: string; name: string; issuer: string; type: string; expiry_date: string; fingerprint: string; auto_renew: boolean; days_to_expiry: number; }
export function useCertificateManagement(baseUrl: string = "") {
  const [certs, setCerts] = useState<Cert[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchCerts = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/auth/certificates"); if (!res.ok) throw new Error("HTTP " + res.status); const d = await res.json(); setCerts(d.certificates || d || []); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  const renewCert = useCallback(async (id: string) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/auth/certificates/" + id + "/renew", { method: "POST" }); if (!res.ok) throw new Error("HTTP " + res.status); return true; } catch (e: any) { setError(e.message); return false; } finally { setLoading(false); } }, [baseUrl]);
  return { certs, loading, error, fetchCerts, renewCert };
}

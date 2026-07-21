import { useState, useCallback } from "react";
export interface Branding { logo_url: string; primary_color: string; secondary_color: string; accent_color: string; custom_css: string; theme: "light" | "dark" | "auto"; custom_domain: string; }
export function useTenantBranding(baseUrl: string = "") {
  const [branding, setBranding] = useState<Branding | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchBranding = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/admin/branding"); if (!res.ok) throw new Error("HTTP " + res.status); setBranding(await res.json()); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  const saveBranding = useCallback(async (b: Branding) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/admin/branding", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(b) }); if (!res.ok) throw new Error("HTTP " + res.status); setBranding(b); return true; } catch (e: any) { setError(e.message); return false; } finally { setLoading(false); } }, [baseUrl]);
  return { branding, loading, error, fetchBranding, saveBranding };
}

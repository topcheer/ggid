import { useState, useCallback } from "react";
export interface Device { device_id: string; user_id: string; username: string; device_name: string; platform: string; last_seen: string; trust_level: "managed" | "byod" | "untrusted"; enrolled_at: string; fingerprint: string; }
export function useDeviceManagement(baseUrl: string = "") {
  const [devices, setDevices] = useState<Device[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchDevices = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/identity/devices"); if (!res.ok) throw new Error("HTTP " + res.status); const data = await res.json(); setDevices(data.devices || data || []); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  const revokeDevice = useCallback(async (id: string) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/identity/devices/" + id, { method: "DELETE" }); if (!res.ok) throw new Error("HTTP " + res.status); return true; } catch (e: any) { setError(e.message); return false; } finally { setLoading(false); } }, [baseUrl]);
  return { devices, loading, error, fetchDevices, revokeDevice };
}

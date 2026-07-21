import { useState, useCallback } from "react";
export interface PostureData { rules: { disk_encrypted: boolean; os_version_min: string; antivirus_required: boolean; firewall_enabled: boolean; jailbreak_detected_action: string }; compliance_threshold: number; by_platform: { platform: string; total: number; compliant: number }[]; non_compliant: { device_id: string; username: string; platform: string; failed_checks: string[]; last_check: string }[]; }
export function useDevicePosture(baseUrl: string = "") {
  const [data, setData] = useState<PostureData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchData = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/identity/device-posture"); if (!res.ok) throw new Error("HTTP " + res.status); setData(await res.json()); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  return { data, loading, error, fetchData };
}

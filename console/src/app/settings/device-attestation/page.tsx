"use client";

import { useState, useEffect, useCallback } from "react";
import { Smartphone, ShieldCheck, ShieldX, ShieldAlert, Cpu, Lock } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface DeviceAttestation {
  id: string;
  device_name: string;
  device_type: string;
  user_id: string;
  username: string;
  tpm_status: "verified" | "missing" | "failed";
  secure_boot: boolean;
  code_integrity: boolean;
  trust_level: "trusted" | "managed" | "untrusted";
  last_attested: string;
  attestation_count: number;
}

const trustColors: Record<string, string> = {
  trusted: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400",
  managed: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  untrusted: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
};

export default function DeviceAttestationPage() {
  const t = useTranslations();

  const [devices, setDevices] = useState<DeviceAttestation[]>([]);
  const [loading, setLoading] = useState(false);
  const [filterTrust, setFilterTrust] = useState("all");

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/auth/device-attestations", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const data = await res.json();
        setDevices(data.devices || data || []);
      }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const filtered = filterTrust === "all" ? devices : devices.filter((d) => d.trust_level === filterTrust);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><ShieldCheck className="w-6 h-6 text-blue-500" /> {t("deviceAttestation.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Monitor device hardware attestation status and trust levels.</p>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Total Devices</span><p className="text-2xl font-bold mt-1">{devices.length}</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Trusted</span><p className="text-2xl font-bold mt-1 text-green-600">{devices.filter((d) => d.trust_level === "trusted").length}</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Managed</span><p className="text-2xl font-bold mt-1 text-yellow-600">{devices.filter((d) => d.trust_level === "managed").length}</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Untrusted</span><p className="text-2xl font-bold mt-1 text-red-600">{devices.filter((d) => d.trust_level === "untrusted").length}</p></div>
      </div>

      {/* Filter */}
      <select value={filterTrust} onChange={(e) => setFilterTrust(e.target.value)} className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm">
        <option value="all">All Trust Levels</option>
        <option value="trusted">Trusted</option>
        <option value="managed">Managed</option>
        <option value="untrusted">Untrusted</option>
      </select>

      {/* Device list */}
      <div className="rounded-lg border dark:border-gray-800">
        <div className="divide-y dark:divide-gray-800">
          {filtered.map((d) => (
            <div key={d.id} className="px-4 py-3">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-lg flex items-center justify-center" style={{ backgroundColor: d.trust_level === "trusted" ? "rgba(16,185,129,0.1)" : d.trust_level === "managed" ? "rgba(245,158,11,0.1)" : "rgba(239,68,68,0.1)" }}>
                    <Smartphone className="w-5 h-5" style={{ color: d.trust_level === "trusted" ? "#10b981" : d.trust_level === "managed" ? "#f59e0b" : "#ef4444" }} />
                  </div>
                  <div>
                    <div className="flex items-center gap-2">
                      <span className="font-medium">{d.device_name}</span>
                      <span className="text-xs text-gray-400">{d.device_type}</span>
                      <span className={`px-2 py-0.5 rounded text-xs ${trustColors[d.trust_level]}`}>{d.trust_level}</span>
                    </div>
                    <p className="text-xs text-gray-400 mt-0.5">{d.username} · Last attested: {d.last_attested}</p>
                  </div>
                </div>
                <div className="flex items-center gap-4 text-xs">
                  {/* TPM */}
                  <div className="flex items-center gap-1">
                    {d.tpm_status === "verified" ? <ShieldCheck className="w-4 h-4 text-green-500" /> : <ShieldX className="w-4 h-4 text-red-500" />}
                    <span className="text-gray-500">TPM</span>
                  </div>
                  {/* Secure Boot */}
                  <div className="flex items-center gap-1">
                    {d.secure_boot ? <Lock className="w-4 h-4 text-green-500" /> : <Lock className="w-4 h-4 text-gray-300" />}
                    <span className="text-gray-500">Boot</span>
                  </div>
                  {/* Code Integrity */}
                  <div className="flex items-center gap-1">
                    {d.code_integrity ? <Cpu className="w-4 h-4 text-green-500" /> : <Cpu className="w-4 h-4 text-gray-300" />}
                    <span className="text-gray-500">Integrity</span>
                  </div>
                </div>
              </div>
              {d.tpm_status === "failed" && (
                <div className="mt-2 flex items-center gap-2 text-xs text-red-600"><ShieldAlert className="w-3 h-3" /> TPM attestation failed — device may be compromised.</div>
              )}
            </div>
          ))}
          {filtered.length === 0 && !loading && <p className="px-4 py-8 text-center text-sm text-gray-500">No devices found.</p>}
        </div>
      </div>
    </div>
  );
}

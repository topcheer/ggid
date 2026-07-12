"use client";

import { useState, useEffect, useCallback } from "react";
import { Building, ShieldCheck, AlertTriangle, Clock } from "lucide-react";

interface Vendor {
  id: string;
  name: string;
  service_type: string;
  risk_rating: "low" | "medium" | "high" | "critical";
  data_access_scope: string;
  contract_expiry: string;
  days_to_expiry: number;
  compliance_status: "compliant" | "pending" | "non_compliant";
  last_assessment: string;
}

const riskColors: Record<string, string> = {
  low: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-400",
  medium: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  high: "bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400",
  critical: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
};

const complianceColors: Record<string, string> = {
  compliant: "text-green-600",
  pending: "text-yellow-600",
  non_compliant: "text-red-600",
};

export default function VendorManagementPage() {
  const [vendors, setVendors] = useState<Vendor[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/identity/vendors", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const d = await res.json(); setVendors(d.vendors || d || []); }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Building className="w-6 h-6 text-blue-500" /> Vendor Management</h1>
        <p className="text-sm text-gray-500 mt-1">Track third-party vendors with risk ratings, compliance status, and contract expiry.</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {vendors.map((v) => (
          <div key={v.id} className="rounded-lg border dark:border-gray-800 p-4 space-y-3">
            <div className="flex items-center justify-between"><div><span className="font-semibold">{v.name}</span><p className="text-xs text-gray-400">{v.service_type}</p></div><span className={`px-2 py-1 rounded text-xs ${riskColors[v.risk_rating]}`}>{v.risk_rating}</span></div>
            <div className="space-y-1 text-sm"><div className="flex items-center gap-2"><ShieldCheck className="w-3.5 h-3.5 text-gray-400" /><span className="text-gray-500">Data Access:</span><span className="font-mono text-xs">{v.data_access_scope}</span></div><div className="flex items-center gap-2"><span className="text-gray-500">Compliance:</span><span className={`font-medium ${complianceColors[v.compliance_status]}`}>{v.compliance_status.replace("_", " ")}</span></div><div className="flex items-center gap-2"><span className="text-gray-500">Last Assessment:</span><span className="text-xs">{v.last_assessment}</span></div></div>
            {v.days_to_expiry <= 30 && (
              <div className={"rounded p-2 text-xs flex items-center gap-2 " + (v.days_to_expiry <= 7 ? "bg-red-50 dark:bg-red-900/20 text-red-600" : "bg-yellow-50 dark:bg-yellow-900/20 text-yellow-600")}><Clock className="w-3.5 h-3.5" /> Contract expires in {v.days_to_expiry} days ({v.contract_expiry})</div>
            )}
            {v.days_to_expiry > 30 && <div className="text-xs text-gray-400 flex items-center gap-1"><Clock className="w-3 h-3" /> Expires: {v.contract_expiry}</div>}
          </div>
        ))}
        {vendors.length === 0 && !loading && <div className="col-span-full text-center text-gray-500 py-8">No vendors found.</div>}
      </div>
    </div>
  );
}

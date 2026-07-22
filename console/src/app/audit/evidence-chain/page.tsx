"use client";

import { useState, useEffect, useCallback } from "react";
import { Search, Link2, ShieldCheck, CheckCircle2, AlertCircle, Hash, User, Calendar } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface ChainEntry {
  id: string;
  step: number;
  action: string;
  collected_by: string;
  collected_at: string;
  hash: string;
  prev_hash: string;
  verified_by: string | null;
  verified_at: string | null;
  status: "pending" | "verified" | "failed";
  evidence_type: string;
  description: string;
}

interface EvidenceChain {
  control_id: string;
  control_name: string;
  framework: string;
  entries: ChainEntry[];
  chain_intact: boolean;
}

const statusIcons: Record<string, typeof CheckCircle2> = {
  verified: CheckCircle2,
  pending: AlertCircle,
  failed: AlertCircle,
};

const statusColors: Record<string, string> = {
  verified: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400",
  pending: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  failed: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
};

export default function EvidenceChainPage() {
  const t = useTranslations();

  const [data, setData] = useState<EvidenceChain | null>(null);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(false);
  const [verifying, setVerifying] = useState<string | null>(null);

  const fetchChain = useCallback(async (controlId: string) => {
    if (!controlId) return;
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/audit/evidence-chain?control_id=${encodeURIComponent(controlId)}`, { headers: { ...authHeader(), "X-Tenant-ID": localStorage.getItem("ggid_tenant_id") || "" } });
      if (res.ok) {
        const json = await res.json();
        setData(json);
      }
    } catch {
      /* noop */
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (!search) return;
    fetchChain(search);
  }, [search, fetchChain]);

  const verifyEntry = async (entryId: string) => {
    setVerifying(entryId);
    try {
      await fetch(`/api/v1/audit/evidence-chain/${entryId}/verify`, {
        method: "POST",
        headers: { ...authHeader(), "X-Tenant-ID": localStorage.getItem("ggid_tenant_id") || "" },
      });
      if (data) {
        setData({
          ...data,
          entries: data.entries.map((e: any) => e.id === entryId ? { ...e, status: "verified", verified_by: "current_user", verified_at: new Date().toISOString() } : e),
        });
      }
    } catch {
      /* noop */
    } finally {
      setVerifying(null);
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Link2 className="w-6 h-6 text-blue-500" /> {t("auditEvidenceChain.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Chain of custody timeline for compliance evidence with hash verification.</p>
      </div>

      {/* Control ID search */}
      <div className="relative max-w-md">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
        <input aria-label="Search by Control ID (e.g. SOC2-CC1.1)..." type="text" placeholder="Search by Control ID (e.g. SOC2-CC1.1)..." value={search} onChange={(e) => setSearch(e.target.value)} className="w-full pl-9 pr-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" />
      </div>

      {loading && <p className="text-sm text-gray-500">Loading chain...</p>}

      {data && (
        <div className="space-y-4">
          {/* Control header */}
          <div className="rounded-lg border dark:border-gray-800 p-4">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="font-semibold flex items-center gap-2"><ShieldCheck className="w-5 h-5 text-blue-500" /> {data.control_name}</h3>
                <p className="text-xs text-gray-500 mt-0.5 font-mono">{data.control_id} &middot; {data.framework}</p>
              </div>
              <span className={`px-3 py-1 rounded-lg text-sm font-medium ${data.chain_intact ? "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400" : "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400"}`}>
                {data.chain_intact ? "Chain Intact" : "Chain Broken"}
              </span>
            </div>
          </div>

          {/* Timeline */}
          <div className="rounded-lg border dark:border-gray-800">
            <div className="px-4 py-3 border-b dark:border-gray-800">
              <h3 className="font-semibold">Chain of Custody ({data.entries.length} entries)</h3>
            </div>
            <div className="relative">
              {data.entries.map((entry: any, i: any) => {
                const StatusIcon = statusIcons[entry.status] || AlertCircle;
                return (
                  <div key={entry.id} className="relative flex gap-4 px-4 py-4">
                    {/* Vertical line */}
                    {i < data.entries.length - 1 && (
                      <div className="absolute left-[27px] top-16 bottom-0 w-0.5 bg-gray-200 dark:bg-gray-800" />
                    )}
                    {/* Step number */}
                    <div className={`relative z-10 w-10 h-10 rounded-full flex items-center justify-center flex-shrink-0 ${statusColors[entry.status]}`}>
                      <StatusIcon className="w-5 h-5" />
                    </div>
                    {/* Content */}
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <span className="font-medium text-sm">Step {entry.step}: {entry.action}</span>
                        <span className={`px-2 py-0.5 rounded text-xs ${statusColors[entry.status]}`}>{entry.status}</span>
                        <span className="text-xs text-gray-400">{entry.evidence_type}</span>
                      </div>
                      <p className="text-xs text-gray-500 mt-1">{entry.description}</p>
                      <div className="grid grid-cols-2 gap-x-4 gap-y-1 mt-2 text-xs">
                        <div className="flex items-center gap-1"><User className="w-3 h-3 text-gray-400" /> Collected by: <span className="font-medium">{entry.collected_by}</span></div>
                        <div className="flex items-center gap-1"><Calendar className="w-3 h-3 text-gray-400" /> {entry.collected_at}</div>
                        <div className="flex items-center gap-1 font-mono"><Hash className="w-3 h-3 text-gray-400" /> {entry.hash.substring(0, 16)}...</div>
                        <div className="flex items-center gap-1 font-mono text-gray-400">prev: {entry.prev_hash ? entry.prev_hash.substring(0, 12) + "..." : "genesis"}</div>
                        {entry.verified_by && (
                          <div className="flex items-center gap-1"><CheckCircle2 className="w-3 h-3 text-green-500" /> Verified by: <span className="font-medium">{entry.verified_by}</span></div>
                        )}
                      </div>
                      {entry.status !== "verified" && (
                        <button onClick={() => verifyEntry(entry.id)} disabled={verifying === entry.id} className="mt-2 text-xs font-medium text-blue-600 hover:underline disabled:opacity-50">
                          {verifying === entry.id ? "Verifying..." : "Verify"}
                        </button>
                      )}
                    </div>
                  </div>
                );
              })}
              {data.entries.length === 0 && <p className="px-4 py-8 text-center text-sm text-gray-500">No evidence entries.</p>}
            </div>
          </div>
        </div>
      )}

      {!data && !loading && search && <p className="text-sm text-gray-500">No evidence chain found.</p>}
      {!data && !search && <p className="text-sm text-gray-500 text-center py-8">Search for a Control ID to view the chain of custody.</p>}
    </div>
  );
}

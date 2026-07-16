"use client";

import { useState, useEffect, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  Link2, ShieldCheck, ShieldX, RefreshCw, AlertCircle, Loader2,
  Check, X, Clock, Hash, FileLock,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface ChainBlock {
  index: number;
  hash: string;
  prev_hash: string;
  timestamp: string;
  event_count: number;
  verified: boolean;
}

interface ChainStatus {
  valid: boolean;
  total_blocks: number;
  total_events: number;
  last_verified_at: string;
  last_hash: string;
  blocks: ChainBlock[];
}

export default function HashChainPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [status, setStatus] = useState<ChainStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [verifying, setVerifying] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiFetch<ChainStatus>("/api/v1/audit/hash-chain/status").catch(() => null);
      if (data) setStatus(data);
    } catch {
      setError("Failed to load hash chain status");
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => { load(); }, [load]);

  const handleReverify = async () => {
    setVerifying(true);
    try {
      await apiFetch("/api/v1/audit/hash-chain/verify", { method: "POST" });
      await load();
    } catch {
      setError("Verification failed");
    } finally {
      setVerifying(false);
    }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  if (loading) {
    return <div className="flex justify-center py-24"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>;
  }

  const valid = status?.valid ?? false;
  const blocks = status?.blocks ?? [];

  // SVG chain graph layout
  const nodeW = 120, nodeH = 50, gap = 40;
  const svgW = blocks.length * (nodeW + gap) + gap;
  const svgH = nodeH + 80;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <Link2 className="h-6 w-6 text-indigo-600" /> Hash Chain Verification
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Cryptographic integrity verification of audit event logs.
          </p>
        </div>
        <button
          onClick={handleReverify}
          disabled={verifying}
          className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
        >
          {verifying ? <Loader2 className="h-4 w-4 animate-spin" /> : <RefreshCw className="h-4 w-4" />}
          Re-Verify Chain
        </button>
      </div>

      {error && (
        <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}
          <button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      {/* Status banner */}
      <div className={`${cardCls} ${valid ? "border-green-300 dark:border-green-700" : "border-red-300 dark:border-red-700"}`}>
        <div className="flex items-center gap-4">
          <div className={`rounded-xl p-3 ${valid ? "bg-green-100 dark:bg-green-900/30" : "bg-red-100 dark:bg-red-900/30"}`}>
            {valid ? <ShieldCheck className="h-8 w-8 text-green-600" /> : <ShieldX className="h-8 w-8 text-red-600" />}
          </div>
          <div className="flex-1">
            <h2 className={`text-lg font-bold ${valid ? "text-green-700 dark:text-green-400" : "text-red-700 dark:text-red-400"}`}>
              Chain {valid ? "Valid" : "Broken"}
            </h2>
            <p className="text-sm text-gray-500 dark:text-gray-400">
              {valid
                ? "All audit event blocks are cryptographically linked and verified."
                : "Chain integrity compromised. One or more blocks failed verification."}
            </p>
          </div>
        </div>
      </div>

      {/* Summary stats */}
      <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
        <div className={cardCls}>
          <div className="flex items-center gap-2"><Hash className="h-4 w-4 text-indigo-500" /><span className="text-xs font-semibold uppercase text-gray-500">Total Blocks</span></div>
          <p className="mt-2 text-2xl font-bold text-gray-800 dark:text-gray-200">{status?.total_blocks ?? 0}</p>
        </div>
        <div className={cardCls}>
          <div className="flex items-center gap-2"><FileLock className="h-4 w-4 text-blue-500" /><span className="text-xs font-semibold uppercase text-gray-500">Total Events</span></div>
          <p className="mt-2 text-2xl font-bold text-gray-800 dark:text-gray-200">{status?.total_events ?? 0}</p>
        </div>
        <div className={cardCls}>
          <div className="flex items-center gap-2"><Clock className="h-4 w-4 text-purple-500" /><span className="text-xs font-semibold uppercase text-gray-500">Last Verified</span></div>
          <p className="mt-2 text-sm font-bold text-gray-800 dark:text-gray-200">{status?.last_verified_at ? new Date(status.last_verified_at).toLocaleString() : "Never"}</p>
        </div>
        <div className={cardCls}>
          <div className="flex items-center gap-2"><Link2 className="h-4 w-4 text-green-500" /><span className="text-xs font-semibold uppercase text-gray-500">Last Hash</span></div>
          <p className="mt-2 truncate font-mono text-xs font-bold text-gray-500">{status?.last_hash?.substring(0, 24) ?? "—"}...</p>
        </div>
      </div>

      {/* Chain visualization (SVG) */}
      {blocks.length > 0 && (
        <div className={cardCls}>
          <h3 className="mb-4 text-sm font-semibold text-gray-700 dark:text-gray-300">Chain Graph</h3>
          {blocks.length <= 20 ? (
            <div className="overflow-x-auto">
              <svg width={Math.max(svgW, 300)} height={svgH} className="min-w-full">
                {blocks.map((b, i) => {
                  const x = gap + i * (nodeW + gap);
                  const y = 20;
                  const color = b.verified ? "rgb(34 197 94)" : "rgb(239 68 68)";
                  return (
                    <g key={b.index}>
                      {/* Connector line to previous */}
                      {i > 0 && (
                        <line
                          x1={x - gap} y1={y + nodeH / 2}
                          x2={x} y2={y + nodeH / 2}
                          stroke={blocks[i - 1].verified && b.verified ? "rgb(34 197 94)" : "rgb(239 68 68)"}
                          strokeWidth="2" strokeDasharray="4 2"
                        />
                      )}
                      {/* Block rect */}
                      <rect
                        x={x} y={y} width={nodeW} height={nodeH} rx={6}
                        fill={b.verified ? "rgb(220 252 231)" : "rgb(254 226 226)"}
                        stroke={color} strokeWidth="1.5"
                      />
                      {/* Block index */}
                      <text x={x + nodeW / 2} y={y + 20} textAnchor="middle" className="fill-gray-700 text-xs font-bold">#{b.index}</text>
                      {/* Event count */}
                      <text x={x + nodeW / 2} y={y + 38} textAnchor="middle" className="fill-gray-400 text-xs">{b.event_count} events</text>
                      {/* Status icon */}
                      <circle cx={x + nodeW - 8} cy={y + 8} r={4} fill={color} />
                    </g>
                  );
                })}
              </svg>
            </div>
          ) : (
            <p className="py-4 text-sm text-gray-400">Showing latest {blocks.length} blocks. Use re-verify for full chain check.</p>
          )}
        </div>
      )}

      {/* Block detail table */}
      {blocks.length > 0 && (
        <div>
          <h2 className="mb-3 text-sm font-semibold uppercase text-gray-500">Block Details</h2>
          <div className="hidden overflow-hidden rounded-xl border border-gray-200 shadow-sm md:block dark:border-gray-700">
            <table className="w-full text-sm">
              <thead className="bg-gray-50 dark:bg-gray-800">
                <tr className="text-left text-xs font-semibold uppercase text-gray-500">
                  <th className="px-4 py-3">#</th>
                  <th className="px-4 py-3">Hash</th>
                  <th className="px-4 py-3">Prev Hash</th>
                  <th className="px-4 py-3">Events</th>
                  <th className="px-4 py-3">Timestamp</th>
                  <th className="px-4 py-3">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
                {blocks.map((b) => (
                  <tr key={b.index} className="hover:bg-gray-50 dark:hover:bg-gray-800/50">
                    <td className="px-4 py-3 font-medium text-gray-700 dark:text-gray-300">{b.index}</td>
                    <td className="px-4 py-3 font-mono text-xs text-gray-500">{b.hash.substring(0, 20)}...</td>
                    <td className="px-4 py-3 font-mono text-xs text-gray-400">{b.prev_hash === "0000000000000000000000000000000000000000000000000000000000000000" ? "genesis" : `${b.prev_hash.substring(0, 20)}...`}</td>
                    <td className="px-4 py-3 text-gray-500">{b.event_count}</td>
                    <td className="px-4 py-3 text-gray-500">{new Date(b.timestamp).toLocaleString()}</td>
                    <td className="px-4 py-3">
                      <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${b.verified ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400" : "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400"}`}>
                        {b.verified ? <Check className="h-3 w-3" /> : <X className="h-3 w-3" />}
                        {b.verified ? "Verified" : "Failed"}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Mobile cards */}
          <div className="space-y-3 md:hidden">
            {blocks.map((b) => (
              <div key={b.index} className={cardCls}>
                <div className="flex items-center justify-between">
                  <span className="font-medium text-gray-700 dark:text-gray-300">Block #{b.index}</span>
                  <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${b.verified ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400" : "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400"}`}>{b.verified ? "Verified" : "Failed"}</span>
                </div>
                <p className="mt-1 font-mono text-xs text-gray-400">{b.hash.substring(0, 24)}...</p>
                <div className="mt-1 flex items-center gap-3 text-xs text-gray-400">
                  <span>{b.event_count} events</span>
                  <span>{new Date(b.timestamp).toLocaleString()}</span>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

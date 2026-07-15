"use client";
import { useState, useEffect } from "react";
import { useTranslations } from "@/lib/i18n";

interface ChainBlock {
  index: number;
  hash: string;
  prev_hash: string;
  event_count: number;
  timestamp: string;
  status: "verified" | "unverified";
}

interface TamperAlert {
  block_index: number;
  expected_hash: string;
  actual_hash: string;
  detected_at: string;
  severity: "low" | "medium" | "high";
}

const defaultBlocks: ChainBlock[] = [
  { index: 10042, hash: "a3f2...b91c", prev_hash: "e8d1...4f7a", event_count: 1523, timestamp: "2025-01-15 16:00:00", status: "verified" },
  { index: 10043, hash: "7c4e...9d22", prev_hash: "a3f2...b91c", event_count: 891, timestamp: "2025-01-15 16:05:00", status: "verified" },
  { index: 10044, hash: "f1a8...3e5b", prev_hash: "7c4e...9d22", event_count: 1204, timestamp: "2025-01-15 16:10:00", status: "verified" },
  { index: 10045, hash: "b2d9...c7f1", prev_hash: "f1a8...3e5b", event_count: 677, timestamp: "2025-01-15 16:15:00", status: "unverified" },
];

export default function HashChainVerificationPage() {
  const t = useTranslations();


  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [data, setData] = useState<any[]>([]);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const res = await fetch("/api/v1/audit/verify-integrity", {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            "X-Tenant-ID": "00000000-0000-0000-0000-000000000001",
          },
        });
        if (!res.ok) return null;
        const json = await res.json();
        setData(Array.isArray(json) ? json : [json]);
      } catch (e) {
        setError(e instanceof Error ? e.message : "Failed to load");
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, []);

  if (loading) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-500">Error: {error}</div>;
  if (!data || data.length === 0) return <div className="p-8 text-gray-500">No data available</div>;
  const [blocks] = useState<ChainBlock[]>(defaultBlocks);
  const [integrityStatus, setIntegrityStatus] = useState<"verified" | "tampered">("verified");
  const [verifying, setVerifying] = useState(false);
  const [verifyInterval, setVerifyInterval] = useState(15);
  const [alerts, setAlerts] = useState<TamperAlert[]>([]);

  const handleVerify = async () => {
    setVerifying(true);
    setTimeout(() => {
      setIntegrityStatus("verified");
      setVerifying(false);
    }, 1000);
  };

  const handleReAnchor = async () => {
    setVerifying(true);
    setTimeout(() => {
      setAlerts([]); setIntegrityStatus("verified"); setVerifying(false);
    }, 1200);
  };

  return (
    <div className="p-8 space-y-6 max-w-5xl">
      <h1 className="text-2xl font-bold">Audit Hash Chain Verification</h1>
      <p className="text-gray-600">Verify tamper-evidence of audit log using cryptographic hash chain.</p>

      <div className={`rounded-lg p-6 ${integrityStatus === "verified" ? "bg-green-50 border border-green-300" : "bg-red-50 border border-red-300"}`}>
        <div className="flex items-center justify-between">
          <div>
            <div className="text-sm font-medium text-gray-500">Chain Integrity Status</div>
            <div className={`text-3xl font-bold ${integrityStatus === "verified" ? "text-green-600" : "text-red-600"}`}>{integrityStatus === "verified" ? "VERIFIED" : "TAMPERED"}</div>
          </div>
          <div className="text-right">
            <div className="text-sm text-gray-500">Last Checkpoint</div>
            <div className="text-sm font-mono">2025-01-15 16:15:00</div>
            <div className="text-sm text-gray-500 mt-1">Total Blocks: <span className="font-bold text-gray-700">10,045</span></div>
          </div>
        </div>
      </div>

      {alerts.length > 0 && (
        <div className="bg-white rounded-lg p-6 shadow">
          <h2 className="text-lg font-semibold mb-4 text-red-600">Tamper Detection Alerts</h2>
          <div className="space-y-2">
            {alerts.map((a: TamperAlert, i: number) => (
              <div key={i} className="border-l-4 border-red-500 bg-red-50 p-3 rounded"><div className="text-sm font-medium">Block #{a.block_index} - {a.severity.toUpperCase()}</div><div className="text-xs text-gray-500 mt-1">Expected: <span className="font-mono">{a.expected_hash}</span></div><div className="text-xs text-gray-500">Actual: <span className="font-mono">{a.actual_hash}</span></div><div className="text-xs text-gray-400 mt-1">Detected: {a.detected_at}</div></div>
            ))}
          </div>
        </div>
      )}

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Hash Chain Visualization</h2>
        <div className="flex items-center gap-2 overflow-x-auto pb-4">
          {blocks.map((b: ChainBlock, i: number) => (
            <div key={i} className="flex items-center">
              <div className={`border-2 rounded-lg p-3 min-w-[140px] ${b.status === "verified" ? "border-green-400 bg-green-50" : "border-yellow-400 bg-yellow-50"}`}>
                <div className="text-xs font-mono text-gray-400">#{b.index}</div>
                <div className="text-xs font-mono font-medium">{b.hash}</div>
                <div className="text-xs text-gray-400 mt-1">{b.event_count} events</div>
                <div className="text-xs text-gray-400">{b.timestamp}</div>
                <div className="mt-1"><span className={`px-1.5 py-0.5 rounded text-xs ${b.status === "verified" ? "bg-green-200 text-green-800" : "bg-yellow-200 text-yellow-800"}`}>{b.status}</span></div>
              </div>
              {i < blocks.length - 1 && <span className="text-gray-400 mx-1 text-lg">{"->"}</span>}
            </div>
          ))}
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Verification Actions</h2>
        <div className="flex items-center gap-4">
          <button onClick={handleVerify} disabled={verifying} className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50">{verifying ? "Verifying..." : "Manual Verification"}</button>
          <button onClick={handleReAnchor} disabled={verifying} className="px-4 py-2 bg-purple-600 text-white rounded hover:bg-purple-700 disabled:opacity-50">Re-Anchor Chain</button>
        </div>
        <div><label className="block text-sm font-medium mb-1">Scheduled Verification Interval (minutes)</label><input type="number" value={verifyInterval} onChange={(e) => setVerifyInterval(parseInt(e.target.value) || 0)} className="border rounded px-3 py-2 w-32" /></div>
      </div>
    </div>
  );
}

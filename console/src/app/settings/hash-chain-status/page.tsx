'use client';
import { useState, useEffect } from 'react';
import { Loader2 } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface ChainBlock {
  index: number;
  hash: string;
  prevHash: string;
  timestamp: string;
  eventCount: number;
}

interface TamperAlert {
  id: string;
  blockIndex: number;
  expectedHash: string;
  actualHash: string;
  detectedAt: string;
}

export default function HashChainStatusPage() {
  const t = useTranslations();


  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const res = await fetch("/api/v1/audit/hash-chain/config", {
          method: "GET",
          headers: {
            "Content-Type": "application/json",
            "X-Tenant-ID": "00000000-0000-0000-0000-000000000001",
          },
        });
        if (!res.ok) return null;
        const json = await res.json();
      } catch (e) {
        setError(e instanceof Error ? e.message : "Failed to load");
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, []);

  const [chainValid, setChainValid] = useState(true);
  const [lastVerified, setLastVerified] = useState('2026-07-12 14:30');
  const [verifying, setVerifying] = useState(false);
  const [selectedBlock, setSelectedBlock] = useState<ChainBlock | null>(null);
  if (loading) return <div className="flex items-center justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-blue-500" /></div>;
  if (error) return <div className="p-8 text-red-500">Error: {error}</div>;
  const [blocks] = useState<ChainBlock[]>([
    { index: 0, hash: 'a1b2c3...', prevHash: '000000...', timestamp: '2026-07-10 00:00', eventCount: 142 },
    { index: 1, hash: 'd4e5f6...', prevHash: 'a1b2c3...', timestamp: '2026-07-11 00:00', eventCount: 98 },
    { index: 2, hash: 'g7h8i9...', prevHash: 'd4e5f6...', timestamp: '2026-07-12 00:00', eventCount: 175 },
    { index: 3, hash: 'j0k1l2...', prevHash: 'g7h8i9...', timestamp: '2026-07-13 00:00', eventCount: 63 },
  ]);

  const [alerts] = useState<TamperAlert[]>([
    { id: 'a1', blockIndex: 1, expectedHash: 'd4e5f6...', actualHash: 'x9y8z7...', detectedAt: '2026-07-12 03:15' },
  ]);


  const verify = () => {
    setVerifying(true);
    setTimeout(() => {
      setChainValid(true);
      setLastVerified(new Date().toISOString().slice(0, 16).replace('T', ' '));
      setVerifying(false);
    }, 1000);
  };

  const exportProof = () => {
    const proof = JSON.stringify({ chainValid, blockCount: blocks.length, lastBlockHash: blocks[blocks.length - 1]?.hash, verifiedAt: lastVerified, blocks }, null, 2);
    const blob = new Blob([proof], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url; a.download = 'chain-proof.json'; a.click();
  };

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Hash Chain Status</h1>
        <p className="text-gray-600">Audit log hash chain integrity monitoring and verification.</p>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Chain Integrity</h2>
        <div className="flex items-center gap-6">
          <div className="flex items-center gap-2">
            <span className={`w-4 h-4 rounded-full ${chainValid ? 'bg-green-500' : 'bg-red-500'}`} />
            <span className={`text-lg font-bold ${chainValid ? 'text-green-600' : 'text-red-600'}`}>{chainValid ? 'Valid' : 'Tampered'}</span>
          </div>
          <div>
            <div className="text-xs text-gray-500">Blocks</div>
            <div className="text-lg font-bold">{blocks.length}</div>
          </div>
          <div>
            <div className="text-xs text-gray-500">Last Block Hash</div>
            <div className="font-mono text-sm">{blocks[blocks.length - 1]?.hash}</div>
          </div>
          <div>
            <div className="text-xs text-gray-500">Last Verified</div>
            <div className="text-sm">{lastVerified}</div>
          </div>
        </div>
        <div className="flex gap-3 mt-4">
          <button onClick={verify} disabled={verifying} className="px-4 py-2 bg-blue-600 text-white rounded text-sm disabled:opacity-50">
            {verifying ? 'Verifying...' : 'Manual Verify'}
          </button>
          <button onClick={exportProof} className="px-4 py-2 border rounded text-sm">Export Chain Proof</button>
          <button className="px-4 py-2 border rounded text-sm">Re-anchor Chain</button>
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Block Explorer</h2>
        <div className="flex items-center gap-2 overflow-x-auto pb-2">
          {blocks.map(b => (
            <div key={b.index} className="flex items-center gap-2">
              <button
                onClick={() => setSelectedBlock(b)}
                className={`px-3 py-2 rounded border text-sm ${selectedBlock?.index === b.index ? 'border-blue-500 bg-blue-50' : 'border-gray-200 hover:bg-gray-50'}`}
              >
                <div className="font-mono text-xs">#{b.index}</div>
                <div className="text-xs text-gray-500">{b.hash}</div>
              </button>
              {b.index < blocks.length - 1 && <span className="text-gray-300">{'->'}</span>}
            </div>
          ))}
        </div>

        {selectedBlock && (
          <div className="mt-4 border rounded p-4 space-y-2 text-sm">
            <div><span className="text-gray-500">Index:</span> {selectedBlock.index}</div>
            <div><span className="text-gray-500">Hash:</span> <span className="font-mono">{selectedBlock.hash}</span></div>
            <div><span className="text-gray-500">Previous:</span> <span className="font-mono">{selectedBlock.prevHash}</span></div>
            <div><span className="text-gray-500">Timestamp:</span> {selectedBlock.timestamp}</div>
            <div><span className="text-gray-500">Events:</span> {selectedBlock.eventCount}</div>
          </div>
        )}
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Tamper Alert Log</h2>
        {alerts.length === 0 ? (
          <p className="text-sm text-gray-400">No tamper alerts detected.</p>
        ) : (
          <div className="space-y-2">
            {alerts.map(a => (
              <div key={a.id} className="border border-red-200 bg-red-50 rounded p-3 text-sm">
                <div className="font-medium text-red-700">Tamper detected at block #{a.blockIndex}</div>
                <div className="text-xs text-gray-500 mt-1">Expected: {a.expectedHash} | Actual: {a.actualHash}</div>
                <div className="text-xs text-gray-500">Detected: {a.detectedAt}</div>
              </div>
            ))}
          </div>
        )}
      </section>
    </div>
  );
}

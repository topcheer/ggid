"use client";

import { useState } from "react";
import { GitBranch, Search, Trash2, AlertTriangle, X, KeyRound } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface CascadeNode {
  token_id: string;
  token_masked: string;
  token_type: string;
  user_id: string;
  username: string;
  created_at: string;
  children?: CascadeNode[];
}

export default function RevokeCascadePage() {
  const t = useTranslations();
  const [tokenInput, setTokenInput] = useState("");
  const [tree, setTree] = useState<CascadeNode | null>(null);
  const [revokedCount, setRevokedCount] = useState<number | null>(null);
  const [loading, setLoading] = useState(false);
  const [revoking, setRevoking] = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);

  const lookup = async () => {
    if (!tokenInput) return;
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/oauth/revoke-cascade?token=${encodeURIComponent(tokenInput)}`, { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { setTree(await res.json()); setRevokedCount(null); }
    } catch { /* noop */ }
    finally { setLoading(false); }
  };

  const revokeAll = async () => {
    if (!tree) return;
    setRevoking(true);
    try {
      const count = (node: CascadeNode): number => 1 + (node.children || []).reduce((s, c) => s + count(c), 0);
      await fetch("/api/v1/oauth/revoke-cascade/execute", { method: "POST", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify({ token_id: tree.token_id }) });
      setRevokedCount(count(tree)); setShowConfirm(false); setTree(null);
    } catch { /* noop */ }
    finally { setRevoking(false); }
  };

  function TreeNode({ node, depth }: { node: CascadeNode; depth: number }) {
    return (
      <div>
        <div className="flex items-center gap-2 py-1.5" style={{ paddingLeft: `${depth * 24}px` }}>
          {node.children && node.children.length > 0 ? <GitBranch className="w-4 h-4 text-gray-400" /> : <KeyRound className="w-3 h-3 text-gray-300" />}
          <span className="font-mono text-xs text-gray-500">{node.token_masked}</span>
          <span className="px-1.5 py-0.5 rounded text-xs bg-blue-100 dark:bg-blue-900/30 dark:text-blue-400">{node.token_type}</span>
          <span className="text-xs text-gray-400">{node.username}</span>
        </div>
        {node.children?.map((child, i) => <TreeNode key={i} node={child} depth={depth + 1} />)}
      </div>
    );
  }

  const countNodes = (n: CascadeNode | null): number => { if (!n) return 0; return 1 + (n.children || []).reduce((s, c) => s + countNodes(c), 0); };

  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><GitBranch className="w-6 h-6 text-red-500" /> Revoke Cascade</h1><p className="text-sm text-gray-500 mt-1">Trace token derivation chains and revoke all related tokens.</p></div>

      <div className="flex items-center gap-2">
        <div className="relative flex-1 max-w-md"><Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" /><input aria-label="Enter token or token ID..." type="text" value={tokenInput} onChange={(e) => setTokenInput(e.target.value)} placeholder="Enter token or token ID..." className="w-full pl-9 pr-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono" /></div>
        <button aria-label="action" onClick={lookup} disabled={loading || !tokenInput} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50">{loading ? "Looking up..." : "Lookup"}</button>
      </div>

      {tree && (
        <>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2"><span className="text-sm text-gray-500">Cascade contains</span><span className="text-xl font-bold text-red-600">{countNodes(tree)}</span><span className="text-sm text-gray-500">tokens</span></div>
            <button onClick={() => setShowConfirm(true)} className="px-4 py-2 rounded-lg bg-red-600 text-white text-sm font-medium hover:bg-red-700 flex items-center gap-2"><Trash2 className="w-4 h-4" /> Revoke All</button>
          </div>
          <div className="rounded-lg border dark:border-gray-800 p-4"><div className="px-1 pb-2"><h3 className="font-semibold">Token Cascade Tree</h3></div><TreeNode node={tree} depth={0} /></div>
        </>
      )}

      {revokedCount !== null && <div className="rounded-lg border border-green-200 dark:border-green-900 bg-green-50 dark:bg-green-900/20 p-4 text-sm text-green-700 dark:text-green-400 flex items-center gap-2"><Trash2 className="w-4 h-4" /> Successfully revoked {revokedCount} tokens.</div>}

      {!tree && !loading && revokedCount === null && <p className="text-sm text-gray-500 text-center py-8">Enter a token to view its cascade chain.</p>}

      {showConfirm && tree && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setShowConfirm(false)}>
          <div role="dialog" aria-modal="true" className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between px-6 py-4 border-b dark:border-gray-800"><h3 className="font-semibold flex items-center gap-2"><AlertTriangle className="w-5 h-5 text-red-500" /> Confirm Cascade Revoke</h3><button onClick={() => setShowConfirm(false)} aria-label="Close"><X className="w-5 h-5 text-gray-400" /></button></div>
            <div className="px-6 py-4 text-sm space-y-2"><p>This will revoke <span className="font-bold text-red-600">{countNodes(tree)}</span> tokens in the cascade chain.</p><p className="text-red-600">All derived tokens will be immediately invalidated. Users will need to re-authenticate.</p></div>
            <div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800"><button onClick={() => setShowConfirm(false)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">Cancel</button><button onClick={revokeAll} disabled={revoking} className="px-4 py-2 rounded-lg bg-red-600 text-white text-sm font-medium hover:bg-red-700 disabled:opacity-50" aria-label="Action">{revoking ? "Revoking..." : "Revoke All"}</button></div>
          </div>
        </div>
      )}
    </div>
  );
}

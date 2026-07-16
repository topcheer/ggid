"use client";
import { useTranslations } from "@/lib/i18n";

import { useState, useCallback } from "react";
import { GitCompare, ArrowRight } from "lucide-react";

interface ScopeComparison {
  common: string[];
  only_a: string[];
  only_b: string[];
}

interface Token {
  id: string;
  label: string;
}

export default function TokenScopeDiffPage() {
  const t = useTranslations();
  const [tokens] = useState<Token[]>([{ id: "t1", label: "Access Token (Client A)" }, { id: "t2", label: "Access Token (Client B)" }, { id: "t3", label: "Refresh Token (Client A)" }]);
  const [tokenA, setTokenA] = useState("");
  const [tokenB, setTokenB] = useState("");
  const [comparison, setComparison] = useState<ScopeComparison | null>(null);
  const [loading, setLoading] = useState(false);

  const compare = useCallback(async () => {
    if (!tokenA || !tokenB) return;
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/oauth/token-scope-diff?a=${encodeURIComponent(tokenA)}&b=${encodeURIComponent(tokenB)}`, { headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setComparison(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, [tokenA, tokenB]);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><GitCompare className="w-6 h-6 text-blue-500" />{t("tokenScopeDiff.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Compare OAuth token scopes side by side to identify common and unique permissions.</p>
      </div>

      <div className="rounded-lg border dark:border-gray-800 p-4 space-y-3">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          <div><label className="text-sm font-medium">Token A</label><select aria-label="Token a" value={tokenA} onChange={(e) => setTokenA(e.target.value)} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"><option value="">Select Token</option>{tokens.map((t) => <option key={t.id} value={t.id}>{t.label}</option>)}</select></div>
          <div><label className="text-sm font-medium">Token B</label><select aria-label="token B" value={tokenB} onChange={(e) => setTokenB(e.target.value)} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"><option value="">Select Token</option>{tokens.map((t) => <option key={t.id} value={t.id}>{t.label}</option>)}</select></div>
        </div>
        <button aria-label="action" onClick={compare} disabled={loading || !tokenA || !tokenB} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50 flex items-center gap-2"><GitCompare className="w-4 h-4" /> {loading ? "Comparing..." : "Compare Scopes"}</button>
      </div>

      {comparison && (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div className="rounded-lg border dark:border-gray-800 p-4"><div className="flex items-center gap-2 mb-3"><div className="w-8 h-8 rounded-full bg-green-100 dark:bg-green-900/30 flex items-center justify-center text-green-600 font-bold text-xs">∩</div><h3 className="text-sm font-semibold">Common ({comparison.common.length})</h3></div><div className="space-y-1">{comparison.common.map((s) => <span key={s} className="block px-2 py-1 rounded bg-green-50 dark:bg-green-900/20 text-xs font-mono text-green-700 dark:text-green-400">{s}</span>)}{comparison.common.length === 0 && <span className="text-xs text-gray-400">None</span>}</div></div>
          <div className="rounded-lg border dark:border-gray-800 p-4"><div className="flex items-center gap-2 mb-3"><div className="w-8 h-8 rounded-full bg-blue-100 dark:bg-blue-900/30 flex items-center justify-center text-blue-600 font-bold text-xs">A</div><h3 className="text-sm font-semibold">Only in A ({comparison.only_a.length})</h3></div><div className="space-y-1">{comparison.only_a.map((s) => <span key={s} className="block px-2 py-1 rounded bg-blue-50 dark:bg-blue-900/20 text-xs font-mono text-blue-700 dark:text-blue-400">{s}</span>)}{comparison.only_a.length === 0 && <span className="text-xs text-gray-400">None</span>}</div></div>
          <div className="rounded-lg border dark:border-gray-800 p-4"><div className="flex items-center gap-2 mb-3"><div className="w-8 h-8 rounded-full bg-purple-100 dark:bg-purple-900/30 flex items-center justify-center text-purple-600 font-bold text-xs">B</div><h3 className="text-sm font-semibold">Only in B ({comparison.only_b.length})</h3></div><div className="space-y-1">{comparison.only_b.map((s) => <span key={s} className="block px-2 py-1 rounded bg-purple-50 dark:bg-purple-900/20 text-xs font-mono text-purple-700 dark:text-purple-400">{s}</span>)}{comparison.only_b.length === 0 && <span className="text-xs text-gray-400">None</span>}</div></div>
        </div>
      )}
    </div>
  );
}

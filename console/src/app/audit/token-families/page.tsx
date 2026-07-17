"use client";

import { useState, useEffect } from "react";
import { useApi } from "@/lib/api";
import {
  Share2, Loader2, AlertCircle, X, AlertOctagon, Shield, ChevronRight, KeyRound,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface TokenNode {
  id: string;
  token_type: "access" | "refresh" | "id" | "agent";
  client_id: string;
  user_id: string;
  issued_at: string;
  revoked: boolean;
  parent_id: string;
  suspicious: boolean;
  reason: string;
}

interface TokenFamily {
  family_id: string;
  root_token: TokenNode;
  rotation_chain: TokenNode[];
  theft_detected: boolean;
  theft_reason: string;
  detected_at: string;
}

export default function TokenFamiliesPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [families, setFamilies] = useState<TokenFamily[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      try { setFamilies(await apiFetch<TokenFamily[]>("/api/v1/oauth/token-families").catch(() => [])); }
      catch { setError("Failed to load token families"); }
      finally { setLoading(false); }
    })();
  }, []);

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const flagged = families.filter((f) => f.theft_detected);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Share2 className="h-6 w-6 text-purple-600" /> {t("auditTokenFamilies.title")}</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Token rotation chain visualization with automated theft detection.</p>
      </div>

      {flagged.length > 0 && (
        <div className="flex items-center gap-3 rounded-xl border border-red-200 bg-red-50 px-4 py-3 dark:border-red-800 dark:bg-red-900/20"><AlertOctagon className="h-5 w-5 text-red-600 shrink-0" /><div><span className="font-medium text-red-700 dark:text-red-400">{flagged.length} token theft alert{flagged.length > 1 ? "s" : ""} detected</span><p className="text-sm text-red-600 dark:text-red-400">Potential token replay or concurrent usage detected.</p></div></div>
      )}

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-purple-600" /></div>
      : families.length === 0 ? (
        <div className={cardCls}><div className="py-12 text-center"><Share2 className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No token families.</p></div></div>
      ) : (
        <div className="space-y-4">
          {families.map((fam) => (
            <div key={fam.family_id} className={`${cardCls} ${fam.theft_detected ? "border-red-300 dark:border-red-800" : ""}`}>
              <div className="mb-3 flex items-center justify-between">
                <div className="flex items-center gap-2"><KeyRound className="h-4 w-4 text-purple-500" /><span className="font-mono text-sm font-semibold text-gray-900 dark:text-white">{fam.family_id.slice(0, 16)}</span></div>
                {fam.theft_detected ? <span className="inline-flex items-center gap-1 rounded-full bg-red-100 px-2 py-0.5 text-xs font-medium text-red-700 dark:bg-red-900/30 dark:text-red-400"><Shield className="h-3 w-3" /> Theft Detected</span> : <span className="inline-flex rounded-full bg-green-100 px-2 py-0.5 text-xs font-medium text-green-700 dark:bg-green-900/30 dark:text-green-400">Healthy</span>}
              </div>
              {fam.theft_detected && fam.theft_reason && <p className="mb-3 rounded-lg bg-red-50 px-3 py-2 text-xs text-red-600 dark:bg-red-900/20 dark:text-red-400">{fam.theft_reason}</p>}
              {/* Rotation chain */}
              <div className="flex flex-wrap items-center gap-1">
                {[fam.root_token, ...fam.rotation_chain].map((node, idx, arr) => (
                  <div key={node.id} className="flex items-center gap-1">
                    <div className={`flex flex-col items-center rounded-lg border px-3 py-2 ${node.suspicious ? "border-red-300 bg-red-50 dark:border-red-800 dark:bg-red-900/20" : node.revoked ? "border-gray-200 opacity-50 dark:border-gray-700" : "border-gray-200 dark:border-gray-700"}`}>
                      <div className="flex items-center gap-1">
                        <span className={`h-2 w-2 rounded-full ${node.revoked ? "bg-gray-400" : node.suspicious ? "bg-red-500" : "bg-green-500"}`} />
                        <span className="text-xs font-medium text-gray-700 dark:text-gray-300">{node.token_type}</span>
                        {node.suspicious && <AlertOctagon className="h-3 w-3 text-red-500" />}
                      </div>
                      <span className="mt-0.5 text-xs text-gray-400">{new Date(node.issued_at).toLocaleDateString()}</span>
                    </div>
                    {idx < arr.length - 1 && <ChevronRight className="h-4 w-4 text-gray-300" />}
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

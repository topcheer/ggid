"use client";
import { useTranslations } from "@/lib/i18n";

import { useState, useEffect, useCallback } from "react";
import { GitBranch, Ban, AlertTriangle, ChevronRight } from "lucide-react";

interface TokenFamily {
  family_id: string;
  root_token: string;
  root_client: string;
  child_tokens: { id: string; client: string; issued_at: string; status: "active" | "revoked" }[];
  status: "active" | "revoked";
  reuse_detected: boolean;
}

export default function TokenFamilyPage() {
  const t = useTranslations();
  const [families, setFamilies] = useState<TokenFamily[]>([]);
  const [loading, setLoading] = useState(false);
  const [expanded, setExpanded] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/oauth/token-family", { headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const d = await res.json(); setFamilies(d.families || d || []); }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const revokeFamily = async (familyId: string) => {
    try { await fetch("/api/v1/oauth/token-family/" + familyId + "/revoke", { method: "POST", headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); fetchData(); }
    catch { /* noop */ }
  };

  const reuseCount = families.filter((f) => f.reuse_detected).length;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><GitBranch className="w-6 h-6 text-purple-500" /> {t("backend.tokenFamily.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Analyze token family relationships and detect token reuse.</p>
      </div>

      {reuseCount > 0 && (
        <div className="rounded-lg border border-red-300 dark:border-red-800 bg-red-50 dark:bg-red-900/20 p-3 flex items-center gap-2">
          <AlertTriangle className="w-5 h-5 text-red-500" />
          <span className="font-semibold text-red-700 dark:text-red-400">{reuseCount} families with token reuse detected</span>
        </div>
      )}

      <div className="space-y-2">
        {families.map((f) => (
          <div key={f.family_id} className="rounded-lg border dark:border-gray-800">
            <div className="flex items-center justify-between p-3 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-900/30" onClick={() => setExpanded(expanded === f.family_id ? null : f.family_id)}>
              <div className="flex items-center gap-2">
                <ChevronRight className={"w-4 h-4 text-gray-400 transition-transform " + (expanded === f.family_id ? "rotate-90" : "")} />
                <div>
                  <span className="font-mono text-sm font-medium">{f.family_id}</span>
                  <p className="text-xs text-gray-400">Root: {f.root_token} ({f.root_client}) - {f.child_tokens.length} children</p>
                </div>
              </div>
              <div className="flex items-center gap-2">
                {f.reuse_detected && <span className="px-2 py-0.5 rounded text-xs bg-red-100 dark:bg-red-900/30 dark:text-red-400">{t("backend.tokenFamily.reuse")}</span>}
                <span className={"px-2 py-0.5 rounded text-xs " + (f.status === "active" ? "bg-green-100 dark:bg-green-900/30 dark:text-green-400" : "bg-gray-100 dark:bg-gray-800 dark:text-gray-400")}>{f.status}</span>
                {f.status === "active" && (
                  <button onClick={(e) => { e.stopPropagation(); revokeFamily(f.family_id); }} className="text-xs text-red-600 hover:underline flex items-center gap-1">
                    <Ban className="w-3 h-3" /> Revoke
                  </button>
                )}
              </div>
            </div>
            {expanded === f.family_id && (
              <div className="border-t dark:border-gray-800 p-3 bg-gray-50 dark:bg-gray-900/30">
                <h4 className="text-xs font-semibold text-gray-500 mb-2">{t("backend.tokenFamily.childTokens")}</h4>
                <div className="space-y-1">
                  {f.child_tokens.map((c) => (
                    <div key={c.id} className="flex items-center gap-2 text-xs">
                      <span className="font-mono text-gray-500 w-40 truncate">{c.id}</span>
                      <span className="text-gray-400">{c.client}</span>
                      <span className="text-gray-400 ml-auto">{c.issued_at}</span>
                      <span className={c.status === "active" ? "text-green-600" : "text-red-600"}>{c.status}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        ))}
        {families.length === 0 && !loading && <p className="text-sm text-gray-500 text-center py-8">No token families.</p>}
      </div>
    </div>
  );
}

"use client";

import { useState, useEffect, useCallback } from "react";
import { Network, Search, Building2, User as UserIcon, Briefcase, ChevronRight } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface ChainMember {
  id: string;
  name: string;
  title: string;
  email: string;
  department: string;
  level: string;
}

interface ChainData {
  user: ChainMember;
  chain: ChainMember[];
}

export default function ManagementChainPage() {
  const t = useTranslations();

  const [search, setSearch] = useState("");
  const [data, setData] = useState<ChainData | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async (user: string) => {
    if (!user) return;
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/identity/management-chain?user=${encodeURIComponent(user)}`, { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => {
    if (!search) return;
    fetchData(search);
  }, [search, fetchData]);

  const levelColors: Record<string, string> = {
    IC: "bg-gray-100 dark:bg-gray-800 text-gray-600",
    Manager: "bg-blue-50 dark:bg-blue-900/20 text-blue-600",
    Director: "bg-purple-50 dark:bg-purple-900/20 text-purple-600",
    VP: "bg-orange-50 dark:bg-orange-900/20 text-orange-600",
    "C-level": "bg-red-50 dark:bg-red-900/20 text-red-600",
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Network className="w-6 h-6 text-blue-500" /> {t("managementChain.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">View the reporting hierarchy from user to executive level.</p>
      </div>

      <div className="relative max-w-md">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
        <input aria-label="Search by username or user ID..." type="text" placeholder="Search by username or user ID..." value={search} onChange={(e) => setSearch(e.target.value)} className="w-full pl-9 pr-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" />
      </div>

      {data && (
        <div className="space-y-4">
          {/* Current user card */}
          <div className="rounded-lg border dark:border-gray-800 p-4">
            <div className="flex items-center gap-3">
              <div className="w-12 h-12 rounded-full bg-blue-50 dark:bg-blue-900/20 flex items-center justify-center">
                <UserIcon className="w-6 h-6 text-blue-500" />
              </div>
              <div className="flex-1">
                <div className="flex items-center gap-2">
                  <span className="font-semibold text-lg">{data.user.name}</span>
                  <span className={`px-2 py-0.5 rounded text-xs ${levelColors[data.user.level] || levelColors.IC}`}>{data.user.level}</span>
                </div>
                <div className="flex items-center gap-2 text-sm text-gray-500 mt-0.5">
                  <Briefcase className="w-3 h-3" /> {data.user.title}
                  <span>·</span>
                  <Building2 className="w-3 h-3" /> {data.user.department}
                </div>
                <p className="text-xs text-gray-400 font-mono mt-0.5">{data.user.email}</p>
              </div>
            </div>
          </div>

          {/* Chain visualization */}
          {data.chain.length > 0 && (
            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="font-semibold mb-4">Reporting Chain ({data.chain.length} levels up)</h3>
              <div className="space-y-0">
                {data.chain.map((member: any, i: number) => (
                  <div key={member.id} className="flex items-start gap-3">
                    {/* Connector line */}
                    <div className="flex flex-col items-center">
                      <div className={`w-10 h-10 rounded-full flex items-center justify-center flex-shrink-0 ${levelColors[member.level] || levelColors.IC}`}>
                        <UserIcon className="w-5 h-5" />
                      </div>
                      {i < data.chain.length - 1 && <div className="w-0.5 flex-1 min-h-[2.5rem] bg-gray-200 dark:bg-gray-800" />}
                    </div>
                    {/* Content */}
                    <div className="flex-1 pb-4">
                      <div className="flex items-center gap-2">
                        <ChevronRight className="w-4 h-4 text-gray-300" />
                        <span className="font-medium">{member.name}</span>
                        <span className={`px-2 py-0.5 rounded text-xs ${levelColors[member.level] || levelColors.IC}`}>{member.level}</span>
                      </div>
                      <div className="flex items-center gap-2 text-sm text-gray-500 ml-8 mt-0.5">
                        <Briefcase className="w-3 h-3" /> {member.title}
                        <span>·</span>
                        <span className="font-mono text-xs">{member.email}</span>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Summary */}
          <div className="rounded-lg border dark:border-gray-800 p-4">
            <div className="grid grid-cols-3 gap-4 text-sm">
              <div><span className="text-xs text-gray-400">Chain Depth</span><p className="text-xl font-bold mt-0.5">{data.chain.length}</p></div>
              <div><span className="text-xs text-gray-400">Highest Level</span><p className="text-xl font-bold mt-0.5">{data.chain[data.chain.length - 1]?.level || "-"}</p></div>
              <div><span className="text-xs text-gray-400">Departments</span><p className="text-xl font-bold mt-0.5">{new Set([data.user.department, ...data.chain.map((c: any) => c.department)]).size}</p></div>
            </div>
          </div>
        </div>
      )}

      {!data && !loading && search && <p className="text-sm text-gray-500">No management chain found.</p>}
      {!data && !search && <p className="text-sm text-gray-500 text-center py-8">Search for a user to view their management chain.</p>}
    </div>
  );
}

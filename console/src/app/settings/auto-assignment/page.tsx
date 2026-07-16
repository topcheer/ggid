"use client";

import { useState, useEffect, useCallback } from "react";
import { Users, RefreshCw, Shuffle } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface Assignment {
  id: string;
  reviewer_id: string;
  reviewer_name: string;
  assigned_users: number;
  strategy: "org_manager" | "role_based" | "round_robin";
  last_assigned: string;
}

const strategyLabels: Record<string, string> = {
  org_manager: "Org Manager",
  role_based: "Role Based",
  round_robin: "Round Robin",
};

const strategyColors: Record<string, string> = {
  org_manager: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400",
  role_based: "bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-400",
  round_robin: "bg-teal-100 text-teal-800 dark:bg-teal-900/30 dark:text-teal-400",
};

export default function AutoAssignmentPage() {
  const t = useTranslations();

  const [campaign, setCampaign] = useState("");
  const [campaigns, setCampaigns] = useState<string[]>([]);
  const [assignments, setAssignments] = useState<Assignment[]>([]);
  const [loading, setLoading] = useState(false);
  const [reassigningId, setReassigningId] = useState<string | null>(null);

  const fetchCampaigns = useCallback(async () => {
    try {
      const res = await fetch("/api/v1/policy/auto-assignment/campaigns", { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const data = await res.json(); setCampaigns(data.campaigns || data || []); if (data.campaigns?.[0]) setCampaign(data.campaigns[0]); }
    } catch { /* noop */ }
  }, []);

  const fetchAssignments = useCallback(async () => {
    if (!campaign) return;
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/policy/auto-assignment?campaign=${encodeURIComponent(campaign)}`, { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const data = await res.json(); setAssignments(data.assignments || data || []); }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, [campaign]);

  useEffect(() => { fetchCampaigns(); }, [fetchCampaigns]);
  useEffect(() => { fetchAssignments(); }, [fetchAssignments]);

  const reassign = async (id: string) => {
    setReassigningId(id);
    try { await fetch(`/api/v1/policy/auto-assignment/${id}/reassign`, { method: "POST", headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); fetchAssignments(); }
    catch { /* noop */ }
    finally { setReassigningId(null); }
  };

  const totalUsers = assignments.reduce((sum, a) => sum + a.assigned_users, 0);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Users className="w-6 h-6 text-indigo-500" /> {t("autoAssignment.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Manage reviewer assignment strategies for access review campaigns.</p>
      </div>

      <div className="flex items-center gap-4">
        <select aria-label="Campaign" value={campaign} onChange={(e) => setCampaign(e.target.value)} className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm">
          <option value="">Select Campaign</option>
          {campaigns.map((c) => <option key={c} value={c}>{c}</option>)}
        </select>
        {campaign && <span className="text-sm text-gray-500">{assignments.length} reviewers · {totalUsers} users assigned</span>}
      </div>

      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Reviewer</th><th className="px-4 py-3 text-left font-medium">Assigned Users</th><th className="px-4 py-3 text-left font-medium">Strategy</th><th className="px-4 py-3 text-left font-medium">Last Assigned</th><th className="px-4 py-3 text-left font-medium">Action</th></tr></thead>
          <tbody className="divide-y dark:divide-gray-800">
            {assignments.map((a) => (
              <tr key={a.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                <td className="px-4 py-3"><span className="font-medium">{a.reviewer_name}</span><p className="text-xs text-gray-400 font-mono">{a.reviewer_id}</p></td>
                <td className="px-4 py-3"><span className="font-bold text-indigo-600">{a.assigned_users}</span></td>
                <td className="px-4 py-3"><span className={`px-2 py-0.5 rounded text-xs ${strategyColors[a.strategy]}`}>{strategyLabels[a.strategy]}</span></td>
                <td className="px-4 py-3 text-gray-500 text-xs">{a.last_assigned}</td>
                <td className="px-4 py-3"><button onClick={() => reassign(a.id)} disabled={reassigningId === a.id} className="text-xs font-medium text-blue-600 hover:underline disabled:opacity-50 flex items-center gap-1"><RefreshCw className={`w-3 h-3 ${reassigningId === a.id ? "animate-spin" : ""}`} /> {reassigningId === a.id ? "Reassigning..." : "Reassign"}</button></td>
              </tr>
            ))}
            {assignments.length === 0 && !loading && <tr><td colSpan={5} className="px-4 py-8 text-center text-gray-500">{campaign ? "No assignments." : "Select a campaign."}</td></tr>}
          </tbody>
        </table>
      </div>
    </div>
  );
}

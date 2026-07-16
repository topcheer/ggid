"use client";

import { useState, useEffect, useCallback } from "react";
import { useTranslations } from "@/lib/i18n";
import { ClipboardCheck, Search, Check, X, Edit3, MessageSquare, Users } from "lucide-react";

interface CertificationUser {
  user_id: string;
  username: string;
  email: string;
  current_role: string;
  last_login: string;
  status: "pending" | "certified" | "revoked" | "modified";
  comment: string;
}

interface Campaign {
  id: string;
  name: string;
  framework: string;
  deadline: string;
  total_users: number;
  completed: number;
}

const statusColors: Record<string, string> = {
  pending: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  certified: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400",
  revoked: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
  modified: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400",
};

export default function AccessCertificationPage() {
  const [campaigns, setCampaigns] = useState<Campaign[]>([]);
  const [selectedCampaign, setSelectedCampaign] = useState("");
  const [users, setUsers] = useState<CertificationUser[]>([]);
  const [loading, setLoading] = useState(false);
  const [search, setSearch] = useState("");
  const [commentUser, setCommentUser] = useState<string | null>(null);
  const [commentText, setCommentText] = useState("");

  const t = useTranslations();

  const fetchCampaigns = useCallback(async () => {
    try {
      const res = await fetch("/api/v1/policy/access-certification/campaigns", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const data = await res.json();
        setCampaigns(data.campaigns || data || []);
      }
    } catch { /* noop */ }
  }, []);

  const fetchUsers = useCallback(async () => {
    if (!selectedCampaign) return;
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/policy/access-certification/campaigns/${selectedCampaign}/users`, { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const data = await res.json();
        setUsers(data.users || data || []);
      }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, [selectedCampaign]);

  useEffect(() => { fetchCampaigns(); }, [fetchCampaigns]);
  useEffect(() => { fetchUsers(); }, [fetchUsers]);

  const submitDecision = async (userId: string, decision: "certified" | "revoked" | "modified", comment?: string) => {
    try {
      await fetch(`/api/v1/policy/access-certification/campaigns/${selectedCampaign}/users/${userId}`, {
        method: "POST",
        headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
        body: JSON.stringify({ decision, comment }),
      });
      setUsers((prev) => prev.map((u) => u.user_id === userId ? { ...u, status: decision, comment: comment || u.comment } : u));
    } catch { /* noop */ }
  };

  const filteredUsers = users.filter((u) => !search || u.username.toLowerCase().includes(search.toLowerCase()) || u.email.toLowerCase().includes(search.toLowerCase()));
  const selectedCampaignObj = campaigns.find((c) => c.id === selectedCampaign);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><ClipboardCheck className="w-6 h-6 text-blue-500" /> {t("accessCertification.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">{t("accessCertification.subtitle")}</p>
      </div>

      {/* Campaign selector + search */}
      <div className="flex items-center gap-3 flex-wrap">
        <select aria-label="Selected campaign" value={selectedCampaign} onChange={(e) => setSelectedCampaign(e.target.value)} className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm">
          <option value="">{t("accessCertification.selectCampaign")}</option>
          {campaigns.map((c) => (
            <option key={c.id} value={c.id}>{c.name} ({c.completed}/{c.total_users} done)</option>
          ))}
        </select>
        {selectedCampaignObj && (
          <span className="text-xs text-gray-500">{t("accessCertification.deadline")} {selectedCampaignObj.deadline} · {t("accessCertification.framework")} {selectedCampaignObj.framework}</span>
        )}
        <div className="relative flex-1 max-w-xs ml-auto">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
          <input type="text" placeholder={t("accessCertification.searchUsers")} value={search} onChange={(e) => setSearch(e.target.value)} className="w-full pl-9 pr-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" />
        </div>
      </div>

      {/* Progress bar */}
      {selectedCampaignObj && (
        <div className="rounded-lg border dark:border-gray-800 p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm font-medium flex items-center gap-2"><Users className="w-4 h-4" /> {t("accessCertification.campaignProgress")}</span>
            <span className="text-sm text-gray-500">{selectedCampaignObj.completed}/{selectedCampaignObj.total_users} {t("accessCertification.certified")}</span>
          </div>
          <div className="w-full h-2 rounded-full bg-gray-200 dark:bg-gray-800 overflow-hidden">
            <div className="h-full bg-blue-500" style={{ width: `${(selectedCampaignObj.completed / Math.max(1, selectedCampaignObj.total_users)) * 100}%` }} />
          </div>
        </div>
      )}

      {/* User list */}
      <div className="rounded-lg border dark:border-gray-800">
        <div className="divide-y dark:divide-gray-800">
          {filteredUsers.map((u) => (
            <div key={u.user_id} className="px-4 py-3">
              <div className="flex items-center justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-2">
                    <span className="font-medium">{u.username}</span>
                    <span className="text-xs text-gray-400">{u.email}</span>
                    <span className={`px-2 py-0.5 rounded text-xs ${statusColors[u.status]}`}>{u.status}</span>
                  </div>
                  <p className="text-xs text-gray-500 mt-0.5">Role: <span className="font-mono">{u.current_role}</span> · Last login: {u.last_login}</p>
                  {u.comment && <p className="text-xs text-gray-400 mt-0.5 italic">"{u.comment}"</p>}
                </div>
                {u.status === "pending" && (
                  <div className="flex items-center gap-1">
                    <button onClick={() => submitDecision(u.user_id, "certified")} className="px-2 py-1 rounded text-xs font-medium text-green-700 bg-green-50 dark:bg-green-900/20 hover:bg-green-100 flex items-center gap-1"><Check className="w-3 h-3" /> {t("accessCertification.certify")}</button>
                    <button onClick={() => submitDecision(u.user_id, "revoked")} className="px-2 py-1 rounded text-xs font-medium text-red-700 bg-red-50 dark:bg-red-900/20 hover:bg-red-100 flex items-center gap-1"><X className="w-3 h-3" /> {t("accessCertification.revoke")}</button>
                    <button onClick={() => submitDecision(u.user_id, "modified")} className="px-2 py-1 rounded text-xs font-medium text-blue-700 bg-blue-50 dark:bg-blue-900/20 hover:bg-blue-100 flex items-center gap-1"><Edit3 className="w-3 h-3" /> {t("accessCertification.modify")}</button>
                    <button onClick={() => { setCommentUser(u.user_id); setCommentText(u.comment); }} className="p-1 rounded text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800"><MessageSquare className="w-4 h-4" /></button>
                  </div>
                )}
              </div>
              {/* Inline comment editor */}
              {commentUser === u.user_id && (
                <div className="mt-2 flex items-center gap-2">
                  <input type="text" value={commentText} onChange={(e) => setCommentText(e.target.value)} placeholder={t("accessCertification.addComment")} className="flex-1 px-3 py-1.5 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" />
                  <button onClick={() => { submitDecision(u.user_id, u.status === "pending" ? "certified" : u.status, commentText); setCommentUser(null); }} className="px-3 py-1.5 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700">{t("accessCertification.saveComment")}</button>
                  <button onClick={() => setCommentUser(null)} className="px-3 py-1.5 rounded-lg border dark:border-gray-700 text-sm">{t("accessCertification.cancel")}</button>
                </div>
              )}
            </div>
          ))}
          {filteredUsers.length === 0 && !loading && (
            <p className="px-4 py-8 text-center text-sm text-gray-500">{selectedCampaign ? t("accessCertification.noUsers") : t("accessCertification.selectToBegin")}</p>
          )}
        </div>
      </div>
    </div>
  );
}

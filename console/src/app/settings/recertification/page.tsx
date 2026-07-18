"use client";

import { useState, useEffect, useCallback } from "react";
import { ClipboardCheck, Users, Check, X, Edit3, MessageSquare, Send } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface RecertUser {
  user_id: string;
  username: string;
  email: string;
  current_roles: string[];
  last_login: string;
  decision: "pending" | "keep" | "remove" | "modify";
  comment: string;
}

interface Team {
  id: string;
  name: string;
}

export default function RecertificationPage() {
  const t = useTranslations();

  const [teams, setTeams] = useState<Team[]>([]);
  const [selectedTeam, setSelectedTeam] = useState("");
  const [users, setUsers] = useState<RecertUser[]>([]);
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [commentUser, setCommentUser] = useState<string | null>(null);
  const [commentText, setCommentText] = useState("");

  const fetchTeams = useCallback(async () => {
    try {
      const res = await fetch("/api/v1/org/teams", { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const data = await res.json();
        setTeams(data.teams || data || []);
      }
    } catch { /* noop */ }
  }, []);

  const fetchUsers = useCallback(async () => {
    if (!selectedTeam) return;
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/policy/recertification/teams/${selectedTeam}/users`, { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const data = await res.json();
        setUsers(data.users || data || []);
      }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, [selectedTeam]);

  useEffect(() => { fetchTeams(); }, [fetchTeams]);
  useEffect(() => { if (selectedTeam) fetchUsers(); }, [selectedTeam, fetchUsers]);

  const setDecision = (userId: string, decision: RecertUser["decision"]) => {
    setUsers((prev) => prev.map((u) => u.user_id === userId ? { ...u, decision } : u));
  };

  const setCommentForUser = (userId: string, comment: string) => {
    setUsers((prev) => prev.map((u) => u.user_id === userId ? { ...u, comment } : u));
  };

  const submitAll = async () => {
    const decided = users.filter((u) => u.decision !== "pending");
    if (decided.length === 0) return;
    setSubmitting(true);
    try {
      await fetch(`/api/v1/policy/recertification/teams/${selectedTeam}/submit`, {
        method: "POST",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
        body: JSON.stringify({ decisions: decided.map((u) => ({ user_id: u.user_id, decision: u.decision, comment: u.comment })) }),
      });
      setUsers((prev) => prev.map((u) => u.decision !== "pending" ? { ...u, decision: "pending", comment: "" } : u));
    } catch { /* noop */ }
    finally { setSubmitting(false); }
  };

  const pending = users.filter((u) => u.decision === "pending").length;
  const decided = users.length - pending;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><ClipboardCheck className="w-6 h-6 text-blue-500" /> {t("recertification.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Review and certify team member access with batch decisions.</p>
      </div>

      {/* Team selector + progress */}
      <div className="flex items-center gap-3 flex-wrap">
        <select aria-label="Selected team" value={selectedTeam} onChange={(e) => setSelectedTeam(e.target.value)} className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm">
          <option value="">Select a team...</option>
          {teams.map((t) => <option key={t.id} value={t.id}>{t.name}</option>)}
        </select>
        {selectedTeam && (
          <>
            <div className="flex-1 max-w-xs h-2 rounded-full bg-gray-200 dark:bg-gray-800 overflow-hidden">
              <div className="h-full bg-blue-500" style={{ width: `${users.length > 0 ? (decided / users.length) * 100 : 0}%` }} />
            </div>
            <span className="text-xs text-gray-500">{decided}/{users.length} decided</span>
            <button onClick={submitAll} disabled={decided === 0 || submitting} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50 flex items-center gap-2"><Send className="w-4 h-4" /> {submitting ? "Submitting..." : `Submit & Notify (${decided})`}</button>
          </>
        )}
      </div>

      {/* User list */}
      <div className="rounded-lg border dark:border-gray-800">
        <div className="divide-y dark:divide-gray-800">
          {users.map((u) => (
            <div key={u.user_id} className="px-4 py-3">
              <div className="flex items-center justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-2">
                    <Users className="w-3 h-3 text-gray-400" />
                    <span className="font-medium">{u.username}</span>
                    <span className="text-xs text-gray-400">{u.email}</span>
                    <span className="text-xs text-gray-400">· Last login: {u.last_login}</span>
                  </div>
                  <div className="flex flex-wrap gap-1 mt-1">
                    {u.current_roles.map((r: any, i: number) => <span key={i} className="px-2 py-0.5 rounded text-xs bg-blue-100 dark:bg-blue-900/30 dark:text-blue-400 font-mono">{r}</span>)}
                  </div>
                  {u.comment && <p className="text-xs text-gray-400 mt-1 italic">"{u.comment}"</p>}
                  {commentUser === u.user_id && (
                    <div className="flex items-center gap-2 mt-2">
                      <input aria-label="Add a comment..." type="text" value={commentText} onChange={(e) => setCommentText(e.target.value)} placeholder="Add a comment..." className="flex-1 px-3 py-1.5 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" />
                      <button onClick={() => { setCommentForUser(u.user_id, commentText); setCommentUser(null); }} className="px-3 py-1.5 rounded-lg bg-blue-600 text-white text-sm">Save</button>
                      <button onClick={() => setCommentUser(null)} className="px-3 py-1.5 rounded-lg border dark:border-gray-700 text-sm">Cancel</button>
                    </div>
                  )}
                </div>
                <div className="flex items-center gap-1 ml-4">
                  {u.decision === "pending" ? (
                    <>
                      <button onClick={() => setDecision(u.user_id, "keep")} className="px-2 py-1 rounded text-xs font-medium text-green-700 bg-green-50 dark:bg-green-900/20 hover:bg-green-100 flex items-center gap-1"><Check className="w-3 h-3" /> Keep</button>
                      <button onClick={() => setDecision(u.user_id, "remove")} className="px-2 py-1 rounded text-xs font-medium text-red-700 bg-red-50 dark:bg-red-900/20 hover:bg-red-100 flex items-center gap-1"><X className="w-3 h-3" /> Remove</button>
                      <button onClick={() => setDecision(u.user_id, "modify")} className="px-2 py-1 rounded text-xs font-medium text-blue-700 bg-blue-50 dark:bg-blue-900/20 hover:bg-blue-100 flex items-center gap-1"><Edit3 className="w-3 h-3" /> Modify</button>
                      <button onClick={() => { setCommentUser(u.user_id); setCommentText(u.comment); }} className="p-1 rounded text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800"><MessageSquare className="w-4 h-4" /></button>
                    </>
                  ) : (
                    <div className="flex items-center gap-2">
                      <span className={`px-2 py-0.5 rounded text-xs font-medium ${u.decision === "keep" ? "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400" : u.decision === "remove" ? "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400" : "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400"}`}>{u.decision}</span>
                      <button onClick={() => setDecision(u.user_id, "pending")} className="text-xs text-gray-400 hover:underline">Undo</button>
                    </div>
                  )}
                </div>
              </div>
            </div>
          ))}
          {users.length === 0 && !loading && (
            <p className="px-4 py-8 text-center text-sm text-gray-500">{selectedTeam ? "No users in this team." : "Select a team to begin."}</p>
          )}
        </div>
      </div>
    </div>
  );
}

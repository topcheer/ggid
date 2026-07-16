"use client";

import { useState, useEffect, useCallback } from "react";
import { Search, Sparkles, AlertTriangle, Check, User, Shield, TrendingDown } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface PermissionGrant {
  permission: string;
  resource: string;
  last_used: string | null;
  usage_count: number;
}

interface Recommendation {
  current_role: string;
  recommended_role: string;
  unused_permissions: string[];
  over_granted: string[];
  risk_level: "low" | "medium" | "high";
  confidence: number;
}

interface UserAnalysis {
  user_id: string;
  username: string;
  email: string;
  permissions: PermissionGrant[];
  unused_count: number;
  over_granted_count: number;
  recommendations: Recommendation[];
}

export default function RoleMiningPage() {
  const t = useTranslations();
  const [users, setUsers] = useState<UserAnalysis[]>([]);
  const [selectedUserId, setSelectedUserId] = useState<string>("");
  const [loading, setLoading] = useState(false);
  const [applying, setApplying] = useState<string | null>(null);
  const [search, setSearch] = useState("");

  const fetchAnalysis = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/policy/role-mining/analysis", { headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const data = await res.json();
        setUsers(data.users || data || []);
      }
    } catch {
      /* noop */
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchAnalysis();
  }, [fetchAnalysis]);

  const applyRecommendation = async (userId: string, rec: Recommendation) => {
    setApplying(userId);
    try {
      await fetch("/api/v1/policy/role-mining/apply", {
        method: "POST",
        headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
        body: JSON.stringify({ user_id: userId, current_role: rec.current_role, recommended_role: rec.recommended_role }),
      });
      setUsers((prev) => prev.map((u) => u.user_id === userId ? { ...u, recommendations: u.recommendations.filter((r) => r !== rec) } : u));
    } catch {
      /* noop */
    } finally {
      setApplying(null);
    }
  };

  const selectedUser = users.find((u) => u.user_id === selectedUserId || u.username === selectedUserId);
  const filteredUsers = users.filter((u) => !search || u.username.toLowerCase().includes(search.toLowerCase()) || u.email.toLowerCase().includes(search.toLowerCase()));

  const riskColor: Record<string, string> = {
    low: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400",
    medium: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
    high: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Sparkles className="w-6 h-6 text-blue-500" /> Role Mining</h1>
        <p className="text-sm text-gray-500 mt-1">Analyze permission usage and recommend least-privilege role adjustments.</p>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="rounded-lg border p-4 dark:border-gray-800">
          <div className="flex items-center justify-between">
            <span className="text-sm text-gray-500">Users Analyzed</span>
            <User className="w-5 h-5 text-gray-400" />
          </div>
          <p className="text-2xl font-bold mt-2">{users.length}</p>
        </div>
        <div className="rounded-lg border p-4 dark:border-gray-800">
          <div className="flex items-center justify-between">
            <span className="text-sm text-gray-500">Unused Permissions</span>
            <TrendingDown className="w-5 h-5 text-gray-400" />
          </div>
          <p className="text-2xl font-bold mt-2 text-yellow-600">{users.reduce((s, u) => s + u.unused_count, 0)}</p>
        </div>
        <div className="rounded-lg border p-4 dark:border-gray-800">
          <div className="flex items-center justify-between">
            <span className="text-sm text-gray-500">Over-Granted</span>
            <AlertTriangle className="w-5 h-5 text-gray-400" />
          </div>
          <p className="text-2xl font-bold mt-2 text-red-600">{users.reduce((s, u) => s + u.over_granted_count, 0)}</p>
        </div>
      </div>

      {/* User selector */}
      <div className="flex items-center gap-3">
        <div className="relative flex-1 max-w-xs">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
          <input
            type="text"
            placeholder="Search users..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full pl-9 pr-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"
          />
        </div>
        <select
          value={selectedUserId}
          onChange={(e) => setSelectedUserId(e.target.value)}
          className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"
        >
          <option value="">Select a user...</option>
          {filteredUsers.map((u) => (
            <option key={u.user_id} value={u.user_id}>{u.username} ({u.unused_count} unused, {u.over_granted_count} over-granted)</option>
          ))}
        </select>
      </div>

      {loading && <p className="text-sm text-gray-500">Loading analysis...</p>}

      {/* Selected user detail */}
      {selectedUser && (
        <div className="space-y-4">
          {/* Unused permissions */}
          <div className="rounded-lg border dark:border-gray-800">
            <div className="px-4 py-3 border-b dark:border-gray-800">
              <h3 className="font-semibold flex items-center gap-2"><Shield className="w-4 h-4" /> Unused Permissions ({selectedUser.permissions.filter((p) => p.usage_count === 0).length})</h3>
            </div>
            <div className="divide-y dark:divide-gray-800">
              {selectedUser.permissions.filter((p) => p.usage_count === 0).map((p, i) => (
                <div key={i} className="px-4 py-2 flex items-center justify-between text-sm">
                  <div>
                    <span className="font-mono">{p.permission}</span>
                    <span className="text-gray-400 ml-2">on {p.resource}</span>
                  </div>
                  <span className="text-gray-400">Never used</span>
                </div>
              ))}
              {selectedUser.permissions.filter((p) => p.usage_count === 0).length === 0 && (
                <p className="px-4 py-3 text-sm text-gray-500">No unused permissions.</p>
              )}
            </div>
          </div>

          {/* Over-granted badges */}
          {selectedUser.recommendations.flatMap((r) => r.over_granted).length > 0 && (
            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="font-semibold mb-2">Over-Granted Permissions</h3>
              <div className="flex flex-wrap gap-2">
                {[...new Set(selectedUser.recommendations.flatMap((r) => r.over_granted))].map((perm, i) => (
                  <span key={i} className="px-2 py-1 rounded text-xs bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400">{perm}</span>
                ))}
              </div>
            </div>
          )}

          {/* Recommendations */}
          <div className="rounded-lg border dark:border-gray-800">
            <div className="px-4 py-3 border-b dark:border-gray-800">
              <h3 className="font-semibold flex items-center gap-2"><Sparkles className="w-4 h-4 text-blue-500" /> Recommended Roles ({selectedUser.recommendations.length})</h3>
            </div>
            <div className="divide-y dark:divide-gray-800">
              {selectedUser.recommendations.map((rec, i) => (
                <div key={i} className="px-4 py-3">
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="flex items-center gap-2">
                        <span className="font-medium line-through text-gray-400">{rec.current_role}</span>
                        <span className="text-gray-400">&rarr;</span>
                        <span className="font-medium text-blue-600">{rec.recommended_role}</span>
                        <span className={`px-2 py-0.5 rounded text-xs ${riskColor[rec.risk_level]}`}>{rec.risk_level}</span>
                        <span className="text-xs text-gray-400">{rec.confidence}% confidence</span>
                      </div>
                      {rec.unused_permissions.length > 0 && (
                        <p className="text-xs text-gray-500 mt-1">Removes {rec.unused_permissions.length} unused permissions</p>
                      )}
                    </div>
                    <button
                      onClick={() => applyRecommendation(selectedUser.user_id, rec)}
                      disabled={applying === selectedUser.user_id}
                      className="ml-4 px-3 py-1.5 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50 flex items-center gap-1"
                    >
                      <Check className="w-4 h-4" />
                      {applying === selectedUser.user_id ? "Applying..." : "Apply"}
                    </button>
                  </div>
                </div>
              ))}
              {selectedUser.recommendations.length === 0 && (
                <p className="px-4 py-3 text-sm text-gray-500">No recommendations. User is at least-privilege.</p>
              )}
            </div>
          </div>
        </div>
      )}

      {!selectedUser && !loading && users.length > 0 && (
        <p className="text-sm text-gray-500 text-center py-8">Select a user to view permission analysis and role recommendations.</p>
      )}
    </div>
  );
}

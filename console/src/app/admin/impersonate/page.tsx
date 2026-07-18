"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  UserSearch, Shield, AlertTriangle, User, X, Check,
  Search, UserCheck, LogOut, Clock, Mail, Building2,
  KeyRound, History,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface SearchResult {
  id: string;
  username: string;
  email: string;
  display_name?: string;
  department?: string;
  roles?: string[];
  status?: string;
}

interface ImpersonationState {
  userId: string;
  username: string;
  email: string;
  reason: string;
  startedAt: string;
}

interface HistoryEntry {
  id: string;
  adminUser: string;
  targetUser: string;
  targetEmail: string;
  startTime: string;
  endTime: string | null;
  reason: string;
}

const STORAGE_KEY = "ggid_impersonation_state";

export default function ImpersonatePage() {
  const t = useTranslations();

  const { apiFetch } = useApi();

  // Search state
  const [searchQuery, setSearchQuery] = useState("");
  const [searchResults, setSearchResults] = useState<SearchResult[]>([]);
  const [searching, setSearching] = useState(false);
  const [showResults, setShowResults] = useState(false);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Selection state
  const [selectedUser, setSelectedUser] = useState<SearchResult | null>(null);

  // Reason state
  const [reason, setReason] = useState("");
  const [reasonError, setReasonError] = useState<string | null>(null);
  const reasonTouched = useRef(false);

  // Impersonation state
  const [impersonating, setImpersonating] = useState<ImpersonationState | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [msg, setMsg] = useState<{ type: "success" | "error"; text: string } | null>(null);

  // History state
  const [history, setHistory] = useState<HistoryEntry[]>([]);
  const [loadingHistory, setLoadingHistory] = useState(false);

  // Restore impersonation state from localStorage on mount
  useEffect(() => {
    if (typeof window === "undefined") return;
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored) {
      try {
        const state: ImpersonationState = JSON.parse(stored);
        setImpersonating(state);
      } catch {
        localStorage.removeItem(STORAGE_KEY);
      }
    }
  }, []);

  // Load history on mount
  const loadHistory = useCallback(async () => {
    setLoadingHistory(true);
    try {
      const data = await apiFetch<{ sessions?: HistoryEntry[]; items?: HistoryEntry[] } | HistoryEntry[]>(
        "/api/v1/admin/impersonate/history?limit=20",
      ).catch(() => null);

      if (data) {
        const list = Array.isArray(data) ? data : data.sessions || data.items || [];
        setHistory(list);
      }
    } catch {
      // Silent fail — history is best-effort
    } finally {
      setLoadingHistory(false);
    }
  }, [apiFetch]);

  useEffect(() => {
    loadHistory();
  }, [loadHistory]);

  // Persist impersonation state to localStorage
  const persistImpersonation = (state: ImpersonationState | null) => {
    setImpersonating(state);
    if (state) {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
    } else {
      localStorage.removeItem(STORAGE_KEY);
    }
  };

  // Debounced search
  const performSearch = useCallback(async (query: string) => {
    if (!query.trim()) {
      setSearchResults([]);
      setSearching(false);
      return;
    }
    try {
      const data = await apiFetch<{ users?: SearchResult[]; items?: SearchResult[] } | SearchResult[]>(
        `/api/v1/users?q=${encodeURIComponent(query)}&limit=10`,
      ).catch(() => null);

      if (data) {
        const list = Array.isArray(data) ? data : data.users || data.items || [];
        setSearchResults(list);
      } else {
        setSearchResults([]);
      }
    } catch {
      setSearchResults([]);
    } finally {
      setSearching(false);
    }
  }, [apiFetch]);

  const handleSearchChange = (value: string) => {
    setSearchQuery(value);
    setShowResults(true);
    setSelectedUser(null);

    if (debounceRef.current) {
      clearTimeout(debounceRef.current);
    }

    if (!value.trim()) {
      setSearchResults([]);
      return;
    }

    setSearching(true);
    debounceRef.current = setTimeout(() => {
      performSearch(value);
    }, 300);
  };

  const handleSelectUser = (user: SearchResult) => {
    setSelectedUser(user);
    setSearchQuery(user.username);
    setShowResults(false);
  };

  // Validate reason on change once touched
  useEffect(() => {
    if (reasonTouched.current && reason.trim().length < 10 && reason.length > 0) {
      setReasonError("Reason for impersonation is required for compliance (minimum 10 characters)");
    } else if (reasonTouched.current && reason.length === 0) {
      setReasonError("Reason for impersonation is required for compliance");
    } else {
      setReasonError(null);
    }
  }, [reason]);

  const handleImpersonate = async () => {
    if (!selectedUser) {
      setMsg({ type: "error", text: "Please select a user first" });
      return;
    }
    if (!reason.trim() || reason.trim().length < 10) {
      reasonTouched.current = true;
      setReasonError("Reason for impersonation is required for compliance (minimum 10 characters)");
      return;
    }

    setSubmitting(true);
    setReasonError(null);
    try {
      await apiFetch("/api/v1/admin/impersonate", {
        method: "POST",
        body: JSON.stringify({
          user_id: selectedUser.id,
          reason: reason.trim(),
        }),
      }).catch(() => null);

      const newSession: ImpersonationState = {
        userId: selectedUser.id,
        username: selectedUser.username,
        email: selectedUser.email,
        reason: reason.trim(),
        startedAt: new Date().toISOString(),
      };
      persistImpersonation(newSession);

      // Add to history optimistically
      const entry: HistoryEntry = {
        id: `local-${Date.now()}`,
        adminUser: "You",
        targetUser: selectedUser.username,
        targetEmail: selectedUser.email,
        startTime: newSession.startedAt,
        endTime: null,
        reason: reason.trim(),
      };
      setHistory((prev) => [entry, ...prev]);

      setMsg({ type: "success", text: `Now impersonating ${selectedUser.username}` });
      setSelectedUser(null);
      setSearchQuery("");
      setReason("");
      reasonTouched.current = false;
    } catch {
      setMsg({ type: "error", text: "Failed to start impersonation" });
    } finally {
      setSubmitting(false);
    }
  };

  const handleEndImpersonation = async () => {
    setSubmitting(true);
    try {
      await apiFetch("/api/v1/admin/impersonate/end", {
        method: "POST",
      }).catch(() => null);

      const prevUser = impersonating?.username || "user";

      // Update history entry with end time
      setHistory((prev) =>
        prev.map((h: any, idx: any) =>
          idx === 0 && h.endTime === null
            ? { ...h, endTime: new Date().toISOString() }
            : h,
        ),
      );

      persistImpersonation(null);
      setMsg({ type: "success", text: `Impersonation of ${prevUser} ended` });
    } catch {
      setMsg({ type: "error", text: "Failed to end impersonation" });
    } finally {
      setSubmitting(false);
    }
  };

  useEffect(() => {
    if (msg) {
      const t = setTimeout(() => setMsg(null), 4000);
      return () => clearTimeout(t);
    }
  }, [msg]);

  const getInitials = (name: string): string => {
    return name.slice(0, 2).toUpperCase();
  };

  const formatDuration = (start: string, end: string | null): string => {
    const startMs = new Date(start).getTime();
    const endMs = end ? new Date(end).getTime() : Date.now();
    const diffMs = endMs - startMs;
    const mins = Math.floor(diffMs / 60000);
    const secs = Math.floor((diffMs % 60000) / 1000);
    if (mins >= 60) {
      const hrs = Math.floor(mins / 60);
      return `${hrs}h ${mins % 60}m`;
    }
    if (mins > 0) return `${mins}m ${secs}s`;
    return `${secs}s`;
  };

  const inputCls = "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";
  const cardCls = "rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const headingCls = "mb-4 text-lg font-semibold text-gray-900 dark:text-gray-100";

  const roleBadgeColor = (role: string): string => {
    const r = role.toLowerCase();
    if (r.includes("admin")) return "bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-400";
    if (r.includes("manager")) return "bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-400";
    if (r.includes("editor")) return "bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-400";
    if (r.includes("viewer")) return "bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-300";
    return "bg-purple-100 text-purple-700 dark:bg-purple-900/40 dark:text-purple-400";
  };

  return (
    <div>
      <h1 className="mb-6 flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-gray-100">
        <Shield className="h-7 w-7 text-brand-600" />
        Admin Impersonation
      </h1>

      {/* Impersonation Session Banner */}
      {impersonating && (
        <div className="mb-6 overflow-hidden rounded-xl border-2 border-yellow-400 bg-gradient-to-r from-yellow-50 to-amber-50 dark:border-yellow-600 dark:from-yellow-950 dark:to-amber-950">
          <div className="flex flex-col gap-3 p-4 sm:flex-row sm:items-center">
            <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-yellow-300 dark:bg-yellow-700">
              <AlertTriangle className="h-6 w-6 text-yellow-800 dark:text-yellow-200" />
            </div>
            <div className="flex-1">
              <p className="text-sm font-bold text-yellow-900 dark:text-yellow-100">
                IMPERSONATING: {impersonating.username} ({impersonating.email})
              </p>
              <p className="mt-0.5 text-xs text-yellow-700 dark:text-yellow-300">
                Reason: {impersonating.reason}
              </p>
              <p className="mt-0.5 text-xs text-yellow-600 dark:text-yellow-400">
                <Clock className="mr-1 inline h-3 w-3" />
                Started: {new Date(impersonating.startedAt).toLocaleString()} ({formatDuration(impersonating.startedAt, null)})
              </p>
            </div>
            <button
              onClick={handleEndImpersonation}
              disabled={submitting}
              className="flex items-center justify-center gap-2 rounded-lg bg-red-600 px-5 py-2.5 text-sm font-bold text-white shadow-sm transition-colors hover:bg-red-700 disabled:opacity-50"
            >
              {submitting ? (
                <LogOut className="h-4 w-4 animate-pulse" />
              ) : (
                <LogOut className="h-4 w-4" />
              )}
              End Impersonation
            </button>
          </div>
        </div>
      )}

      {msg && (
        <div className={`mb-4 rounded-lg border p-3 text-sm ${
          msg.type === "success"
            ? "border-green-200 bg-green-50 text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400"
            : "border-red-200 bg-red-50 text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400"
        }`}>
          <div className="flex items-center gap-2">
            {msg.type === "success" ? <Check className="h-4 w-4" /> : <AlertTriangle className="h-4 w-4" />}
            {msg.text}
          </div>
        </div>
      )}

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Left Column: Search & Select */}
        <div>
          {/* Search & Select User Card */}
          <div className={`${cardCls} mb-6`}>
            <h2 className={headingCls}>
              <UserSearch className="mr-2 inline h-5 w-5 text-brand-600" />
              Search User
            </h2>

            <div className="relative">
              {/* Search Input */}
              <div className="relative">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
                <input
                  type="text"
                  value={searchQuery}
                  onChange={(e) => handleSearchChange(e.target.value)}
                  onFocus={() => setShowResults(true)}
                  onBlur={() => setTimeout(() => setShowResults(false), 200)}
                  placeholder="Search by username or email..."
                  className={`${inputCls} pl-10`}
                />
                {searchQuery && (
                  <button
                    onClick={() => {
                      setSearchQuery("");
                      setSearchResults([]);
                      setSelectedUser(null);
                    }}
                    className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200"
                  >
                    <X className="h-4 w-4" />
                  </button>
                )}
              </div>

              {/* Search Results Dropdown */}
              {showResults && (searchQuery || searching) && (
                <div className="absolute z-20 mt-1 w-full overflow-hidden rounded-lg border border-gray-200 bg-white shadow-lg dark:border-gray-700 dark:bg-gray-800">
                  {searching ? (
                    <div className="px-4 py-3 text-center text-sm text-gray-400">
                      Searching...
                    </div>
                  ) : searchResults.length === 0 ? (
                    <div className="px-4 py-3 text-center text-sm text-gray-400">
                      {searchQuery ? "No users found matching your search." : "Start typing to search."}
                    </div>
                  ) : (
                    <ul className="max-h-72 overflow-y-auto py-1">
                      {searchResults.map((user: any) => (
                        <li key={user.id}>
                          <button
                            onClick={() => handleSelectUser(user)}
                            className="flex w-full items-center gap-3 px-4 py-2.5 text-left hover:bg-gray-50 dark:hover:bg-gray-700/50"
                          >
                            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-brand-100 dark:bg-brand-900/30">
                              <span className="text-xs font-bold text-brand-600 dark:text-brand-400">
                                {getInitials(user.username)}
                              </span>
                            </div>
                            <div className="min-w-0 flex-1">
                              <p className="truncate text-sm font-medium text-gray-900 dark:text-gray-100">
                                {user.username}
                                {user.status && user.status !== "active" && (
                                  <span className="ml-1.5 inline-block rounded bg-gray-200 px-1.5 py-0.5 text-[10px] text-gray-600 dark:bg-gray-700 dark:text-gray-400">
                                    {user.status}
                                  </span>
                                )}
                              </p>
                              <p className="truncate text-xs text-gray-500 dark:text-gray-400">
                                {user.email}
                              </p>
                              {user.department && (
                                <p className="truncate text-[11px] text-gray-400">
                                  <Building2 className="mr-0.5 inline h-3 w-3" />{user.department}
                                </p>
                              )}
                            </div>
                            {user.roles && user.roles.length > 0 && (
                              <div className="flex flex-shrink-0 flex-wrap justify-end gap-1">
                                {user.roles.slice(0, 2).map((r: any) => (
                                  <span key={r} className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${roleBadgeColor(r)}`}>
                                    {r}
                                  </span>
                                ))}
                                {user.roles.length > 2 && (
                                  <span className="text-[10px] text-gray-400">+{user.roles.length - 2}</span>
                                )}
                              </div>
                            )}
                            {selectedUser?.id === user.id && (
                              <Check className="h-4 w-4 flex-shrink-0 text-brand-600" />
                            )}
                          </button>
                        </li>
                      ))}
                    </ul>
                  )}
                </div>
              )}
            </div>

            {/* Selected User Confirmation Panel */}
            {selectedUser && (
              <div className="mt-4 rounded-xl border-2 border-brand-300 bg-brand-50 p-5 dark:border-brand-700 dark:bg-brand-900/20">
                <div className="mb-4 flex items-start gap-3">
                  <div className="flex h-14 w-14 shrink-0 items-center justify-center rounded-full bg-brand-600 shadow-md">
                    <span className="text-base font-bold text-white">
                      {getInitials(selectedUser.username)}
                    </span>
                  </div>
                  <div className="flex-1">
                    <p className="flex items-center gap-2 text-sm font-bold text-gray-900 dark:text-gray-100">
                      <UserCheck className="h-4 w-4 text-brand-600" />
                      {selectedUser.display_name || selectedUser.username}
                    </p>
                    <p className="mt-0.5 flex items-center gap-1 text-xs text-gray-500 dark:text-gray-400">
                      <Mail className="h-3 w-3" /> {selectedUser.email}
                    </p>
                    {selectedUser.department && (
                      <p className="mt-0.5 flex items-center gap-1 text-xs text-gray-500 dark:text-gray-400">
                        <Building2 className="h-3 w-3" /> {selectedUser.department}
                      </p>
                    )}
                    <p className="mt-0.5 text-xs text-gray-400">User ID: {selectedUser.id}</p>
                  </div>
                  <button
                    onClick={() => {
                      setSelectedUser(null);
                      setSearchQuery("");
                    }}
                    className="rounded-lg p-1 text-gray-400 hover:bg-gray-200 hover:text-red-500 dark:hover:bg-gray-700"
                  >
                    <X className="h-5 w-5" />
                  </button>
                </div>

                {/* Current Permissions Summary */}
                <div className="rounded-lg border border-brand-200 bg-white p-3 dark:border-brand-800 dark:bg-gray-800">
                  <p className="mb-2 flex items-center gap-1.5 text-xs font-semibold text-gray-700 dark:text-gray-300">
                    <KeyRound className="h-3.5 w-3.5 text-brand-600" />
                    Current Permissions Summary
                  </p>
                  {selectedUser.roles && selectedUser.roles.length > 0 ? (
                    <div className="flex flex-wrap gap-1.5">
                      {selectedUser.roles.map((r: any) => (
                        <span key={r} className={`rounded-full px-2.5 py-0.5 text-xs font-medium ${roleBadgeColor(r)}`}>
                          {r}
                        </span>
                      ))}
                    </div>
                  ) : (
                    <p className="text-xs text-gray-400">No roles assigned (default access only)</p>
                  )}
                </div>
              </div>
            )}
          </div>

          {/* Audit Reason Card */}
          <div className={cardCls}>
            <h2 className={headingCls}>
              <AlertTriangle className="mr-2 inline h-5 w-5 text-yellow-500" />
              Reason for Impersonation (required for compliance)
            </h2>
            <p className="mb-3 text-sm text-gray-500 dark:text-gray-400">
              All impersonation sessions are logged to the audit trail with your identity, the target user,
              and this reason. Provide a clear justification.
            </p>
            <textarea
              value={reason}
              onChange={(e) => {
                reasonTouched.current = true;
                setReason(e.target.value);
              }}
              onBlur={() => { reasonTouched.current = true; }}
              placeholder="e.g., Investigating user-reported login issue with SSO integration for ticket #1234..."
              rows={4}
              className={`${inputCls} ${
                reasonError
                  ? "border-red-400 ring-1 ring-red-400 dark:border-red-600 dark:ring-red-600"
                  : reason.trim().length >= 10
                    ? "border-green-400 dark:border-green-600"
                    : ""
              }`}
            />
            <div className="mt-1.5 flex items-center justify-between">
              {reasonError ? (
                <p className="text-xs font-medium text-red-500">{reasonError}</p>
              ) : reason.trim().length >= 10 ? (
                <p className="flex items-center gap-1 text-xs text-green-600">
                  <Check className="h-3 w-3" /> Reason meets minimum length
                </p>
              ) : (
                <p className="text-xs text-gray-400">Minimum 10 characters required for audit compliance</p>
              )}
              <p className={`text-xs font-medium ${
                reason.trim().length < 10 ? "text-gray-400" : "text-green-600"
              }`}>
                {reason.trim().length} / 10
              </p>
            </div>
          </div>

          {/* Action Button */}
          <div className="mt-6 flex justify-end">
            <button
              onClick={handleImpersonate}
              disabled={!selectedUser || reason.trim().length < 10 || submitting}
              className="flex items-center gap-2 rounded-lg bg-brand-600 px-6 py-3 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-brand-700 disabled:cursor-not-allowed disabled:opacity-50"
             aria-label="User">
              {submitting ? (
                "Starting..."
              ) : (
                <>
                  <User className="h-5 w-5" />
                  Start Impersonation
                </>
              )}
            </button>
          </div>
        </div>

        {/* Right Column: History */}
        <div>
          <div className={cardCls}>
            <div className="mb-4 flex items-center justify-between">
              <h2 className={headingCls + " mb-0"}>
                <History className="mr-2 inline h-5 w-5 text-brand-600" />
                Impersonation History
              </h2>
              <button
                onClick={loadHistory}
                disabled={loadingHistory}
                className="text-xs text-brand-600 hover:underline disabled:opacity-50"
               aria-label="Action">
                {loadingHistory ? "Loading..." : "Refresh"}
              </button>
            </div>

            {history.length === 0 ? (
              <div className="rounded-lg border border-dashed border-gray-300 py-12 text-center dark:border-gray-600">
                <History className="mx-auto mb-3 h-10 w-10 text-gray-300" />
                <p className="text-sm text-gray-500">No impersonation history yet</p>
                <p className="mt-1 text-xs text-gray-400">Past sessions will appear here for audit review</p>
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left text-sm">
                  <thead>
                    <tr className="border-b border-gray-200 dark:border-gray-700">
                      <th scope="col" className="px-3 py-2 text-xs font-semibold text-gray-500">Admin</th>
                      <th scope="col" className="px-3 py-2 text-xs font-semibold text-gray-500">Target</th>
                      <th scope="col" className="px-3 py-2 text-xs font-semibold text-gray-500">Duration</th>
                      <th scope="col" className="px-3 py-2 text-xs font-semibold text-gray-500">Reason</th>
                    </tr>
                  </thead>
                  <tbody>
                    {history.map((entry: any) => (
                      <tr
                        key={entry.id}
                        className="border-b border-gray-100 align-top dark:border-gray-700/50"
                      >
                        <td className="px-3 py-3">
                          <p className="font-medium text-gray-900 dark:text-gray-100">{entry.adminUser}</p>
                        </td>
                        <td className="px-3 py-3">
                          <div className="flex items-center gap-2">
                            <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-gray-200 dark:bg-gray-700">
                              <span className="text-[10px] font-bold text-gray-600 dark:text-gray-300">
                                {getInitials(entry.targetUser)}
                              </span>
                            </div>
                            <div>
                              <p className="font-medium text-gray-900 dark:text-gray-100">{entry.targetUser}</p>
                              <p className="text-xs text-gray-400">{entry.targetEmail}</p>
                            </div>
                          </div>
                        </td>
                        <td className="px-3 py-3">
                          <p className="font-mono text-xs text-gray-700 dark:text-gray-300">
                            {formatDuration(entry.startTime, entry.endTime)}
                          </p>
                          <p className="text-xs text-gray-400">
                            {new Date(entry.startTime).toLocaleDateString()} {new Date(entry.startTime).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}
                          </p>
                          {entry.endTime && (
                            <p className="text-xs text-gray-400">
                              to {new Date(entry.endTime).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}
                            </p>
                          )}
                          {!entry.endTime && (
                            <span className="mt-0.5 inline-block rounded-full bg-green-100 px-1.5 py-0.5 text-[10px] font-medium text-green-700 dark:bg-green-900/40 dark:text-green-400">
                              Active
                            </span>
                          )}
                        </td>
                        <td className="max-w-[200px] px-3 py-3">
                          <p className="text-xs text-gray-600 dark:text-gray-400" title={entry.reason}>
                            {entry.reason}
                          </p>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

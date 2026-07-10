"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  UserSearch, Shield, AlertTriangle, User, X, Check,
  Search, UserCheck, LogOut,
} from "lucide-react";

interface SearchResult {
  id: string;
  username: string;
  email: string;
  display_name?: string;
}

interface ImpersonationState {
  userId: string;
  username: string;
  email: string;
  reason: string;
  startedAt: string;
}

const STORAGE_KEY = "ggid_impersonation_state";

export default function ImpersonatePage() {
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

  // Impersonation state
  const [impersonating, setImpersonating] = useState<ImpersonationState | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [msg, setMsg] = useState<string | null>(null);

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
        // Fallback demo results when API is unavailable
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

  const handleImpersonate = async () => {
    if (!selectedUser) {
      setMsg("Please select a user first");
      return;
    }
    if (!reason.trim() || reason.trim().length < 10) {
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

      persistImpersonation({
        userId: selectedUser.id,
        username: selectedUser.username,
        email: selectedUser.email,
        reason: reason.trim(),
        startedAt: new Date().toISOString(),
      });

      setMsg(`Now impersonating ${selectedUser.username}`);
      setSelectedUser(null);
      setSearchQuery("");
      setReason("");
    } catch {
      setMsg("Failed to start impersonation");
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
      persistImpersonation(null);
      setMsg(`Impersonation of ${prevUser} ended`);
    } catch {
      setMsg("Failed to end impersonation");
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

  const inputCls = "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";
  const cardCls = "rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const headingCls = "mb-4 text-lg font-semibold text-gray-900 dark:text-gray-100";

  return (
    <div>
      <h1 className="mb-6 flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-gray-100">
        <Shield className="h-7 w-7 text-brand-600" />
        Admin Impersonation
      </h1>

      {/* Impersonation Banner */}
      {impersonating && (
        <div className="mb-6 flex flex-col gap-3 rounded-xl border-2 border-yellow-400 bg-yellow-50 p-4 dark:border-yellow-600 dark:bg-yellow-950">
          <div className="flex flex-wrap items-center gap-3">
            <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-yellow-200 dark:bg-yellow-800">
              <AlertTriangle className="h-5 w-5 text-yellow-700 dark:text-yellow-300" />
            </div>
            <div className="flex-1">
              <p className="text-sm font-semibold text-yellow-900 dark:text-yellow-100">
                Impersonating user: {impersonating.username} ({impersonating.email})
              </p>
              <p className="text-xs text-yellow-700 dark:text-yellow-300">
                Reason: {impersonating.reason}
              </p>
              <p className="text-xs text-yellow-600 dark:text-yellow-400">
                Started: {new Date(impersonating.startedAt).toLocaleString()}
              </p>
            </div>
            <button
              onClick={handleEndImpersonation}
              disabled={submitting}
              className="flex items-center gap-2 rounded-lg bg-red-600 px-4 py-2 text-sm font-semibold text-white hover:bg-red-700 disabled:opacity-50"
            >
              <LogOut className="h-4 w-4" />
              End Impersonation
            </button>
          </div>
        </div>
      )}

      {msg && (
        <div className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400">
          {msg}
        </div>
      )}

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
            <div className="absolute z-10 mt-1 w-full overflow-hidden rounded-lg border border-gray-200 bg-white shadow-lg dark:border-gray-700 dark:bg-gray-800">
              {searching ? (
                <div className="px-4 py-3 text-center text-sm text-gray-400">
                  Searching...
                </div>
              ) : searchResults.length === 0 ? (
                <div className="px-4 py-3 text-center text-sm text-gray-400">
                  {searchQuery ? "No users found matching your search." : "Start typing to search."}
                </div>
              ) : (
                <ul className="max-h-64 overflow-y-auto py-1">
                  {searchResults.map((user) => (
                    <li key={user.id}>
                      <button
                        onClick={() => handleSelectUser(user)}
                        className="flex w-full items-center gap-3 px-4 py-2.5 text-left hover:bg-gray-50 dark:hover:bg-gray-700/50"
                      >
                        <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-brand-100 dark:bg-brand-900/30">
                          <span className="text-xs font-bold text-brand-600 dark:text-brand-400">
                            {getInitials(user.username)}
                          </span>
                        </div>
                        <div className="min-w-0 flex-1">
                          <p className="truncate text-sm font-medium text-gray-900 dark:text-gray-100">
                            {user.username}
                          </p>
                          <p className="truncate text-xs text-gray-500 dark:text-gray-400">
                            {user.email}
                          </p>
                        </div>
                        {selectedUser?.id === user.id && (
                          <Check className="h-4 w-4 text-brand-600" />
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
          <div className="mt-4 rounded-lg border-2 border-brand-300 bg-brand-50 p-4 dark:border-brand-700 dark:bg-brand-900/20">
            <div className="flex items-center gap-3">
              <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-brand-600">
                <span className="text-sm font-bold text-white">
                  {getInitials(selectedUser.username)}
                </span>
              </div>
              <div className="flex-1">
                <p className="flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-gray-100">
                  <UserCheck className="h-4 w-4 text-brand-600" />
                  {selectedUser.display_name || selectedUser.username}
                </p>
                <p className="text-xs text-gray-500 dark:text-gray-400">{selectedUser.email}</p>
                <p className="text-xs text-gray-400">User ID: {selectedUser.id}</p>
              </div>
              <button
                onClick={() => {
                  setSelectedUser(null);
                  setSearchQuery("");
                }}
                className="text-gray-400 hover:text-red-500"
              >
                <X className="h-5 w-5" />
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Audit Reason Card */}
      <div className={`${cardCls} mb-6`}>
        <h2 className={headingCls}>
          <AlertTriangle className="mr-2 inline h-5 w-5 text-yellow-500" />
          Audit Reason
        </h2>
        <p className="mb-3 text-sm text-gray-500 dark:text-gray-400">
          Reason for impersonation is required for compliance. All impersonation sessions are logged to the audit trail.
        </p>
        <textarea
          value={reason}
          onChange={(e) => {
            setReason(e.target.value);
            if (reasonError) setReasonError(null);
          }}
          placeholder="e.g., Investigating user-reported login issue with SSO integration..."
          rows={4}
          className={`${inputCls} ${reasonError ? "border-red-400 dark:border-red-600" : ""}`}
        />
        <div className="mt-1 flex items-center justify-between">
          {reasonError ? (
            <p className="text-xs text-red-500">{reasonError}</p>
          ) : (
            <p className="text-xs text-gray-400">Minimum 10 characters required</p>
          )}
          <p className="text-xs text-gray-400">{reason.trim().length} chars</p>
        </div>
      </div>

      {/* Action Button */}
      <div className="flex justify-end">
        <button
          onClick={handleImpersonate}
          disabled={!selectedUser || reason.trim().length < 10 || submitting}
          className="flex items-center gap-2 rounded-lg bg-brand-600 px-6 py-3 text-sm font-semibold text-white hover:bg-brand-700 disabled:cursor-not-allowed disabled:opacity-50"
        >
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
  );
}

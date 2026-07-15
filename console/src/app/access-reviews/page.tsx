"use client";

import { useState, useEffect, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  ClipboardCheck,
  Plus,
  Trash2,
  Loader2,
  CheckCircle2,
  XCircle,
  Clock,
  Users,
  Shield,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface AccessReview {
  id: string;
  name: string;
  description: string;
  status: "draft" | "active" | "completed" | "cancelled";
  scope: string;
  start_date: string;
  end_date: string;
  reviewer_count: number;
  total_items: number;
  completed_items: number;
  certified_items: number;
  revoked_items: number;
  created_at: string;
}

interface ReviewItem {
  id: string;
  review_id: string;
  user_id: string;
  username: string;
  email: string;
  resource_type: string;
  resource_id: string;
  role: string;
  assigned_at: string;
  decision: "pending" | "certify" | "revoke" | "no_action";
  decided_by?: string;
  decided_at?: string;
  comment?: string;
}

export default function AccessReviewsPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [reviews, setReviews] = useState<AccessReview[]>([]);
  const [items, setItems] = useState<ReviewItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [decisionLoading, setDecisionLoading] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const data = await apiFetch<{ reviews?: AccessReview[] }>("/api/v1/access-reviews").catch(() => ({ reviews: [] }));
      setReviews(data.reviews ?? []);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => { load(); }, [load]);

  const handleExpand = async (reviewId: string) => {
    if (expandedId === reviewId) {
      setExpandedId(null);
      setItems([]);
      return;
    }
    setExpandedId(reviewId);
    try {
      const data = await apiFetch<{ items?: ReviewItem[] }>(`/api/v1/access-reviews/${reviewId}/items`);
      setItems(data.items ?? []);
    } catch {
      setItems([]);
    }
  };

  const handleDecision = async (itemId: string, decision: ReviewItem["decision"]) => {
    setDecisionLoading(itemId);
    setItems(items.map((it) => (it.id === itemId ? { ...it, decision, decided_at: new Date().toISOString() } : it)));
    try {
      await apiFetch(`/api/v1/access-reviews/items/${itemId}/decision`, {
        method: "POST",
        body: JSON.stringify({ decision }),
      });
    } catch {
      /* optimistic */
    } finally {
      setDecisionLoading(null);
    }
  };

  const handleCreate = async () => {
    const newReview = {
      name: `Recertification Campaign ${reviews.length + 1}`,
      description: "Quarterly access review",
      scope: "all",
      start_date: new Date().toISOString(),
      end_date: new Date(Date.now() + 30 * 86400000).toISOString(),
    };
    try {
      await apiFetch("/api/v1/access-reviews", { method: "POST", body: JSON.stringify(newReview) });
      await load();
    } catch {
      /* ignore */
    }
  };

  const handleDelete = async (id: string) => {
    setReviews(reviews.filter((r) => r.id !== id));
    try { await apiFetch(`/api/v1/access-reviews/${id}`, { method: "DELETE" }); } catch { /* optimistic */ }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const statusBadge = (status: string) => {
    const map: Record<string, string> = {
      active: "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400",
      draft: "bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400",
      completed: "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400",
      cancelled: "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400",
    };
    return map[status] ?? map["draft"];
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <ClipboardCheck className="h-7 w-7 text-indigo-600" />
            Access Reviews
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Recertification campaigns for periodic access review and governance.
          </p>
        </div>
        <button
          onClick={handleCreate}
          className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700"
        >
          <Plus className="mr-1 inline h-4 w-4" /> New Campaign
        </button>
      </div>

      {loading ? (
        <div className="flex justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-indigo-600" />
        </div>
      ) : reviews.length === 0 ? (
        <div className={`${cardCls} text-center`}>
          <ClipboardCheck className="mx-auto mb-3 h-12 w-12 text-gray-300" />
          <p className="text-gray-500 dark:text-gray-400">No recertification campaigns yet.</p>
        </div>
      ) : (
        <div className="space-y-4">
          {reviews.map((review) => (
            <div key={review.id} className={cardCls}>
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-2">
                    <h3 className="text-lg font-semibold text-gray-900 dark:text-white">{review.name}</h3>
                    <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${statusBadge(review.status)}`}>
                      {review.status}
                    </span>
                  </div>
                  <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{review.description}</p>
                  <div className="mt-3 flex flex-wrap gap-4 text-sm text-gray-600 dark:text-gray-300">
                    <span className="flex items-center gap-1"><Users className="h-4 w-4" /> {review.reviewer_count} reviewers</span>
                    <span className="flex items-center gap-1"><Shield className="h-4 w-4" /> {review.completed_items}/{review.total_items} reviewed</span>
                    {review.certified_items > 0 && <span className="text-green-600">{review.certified_items} certified</span>}
                    {review.revoked_items > 0 && <span className="text-red-600">{review.revoked_items} revoked</span>}
                    <span className="flex items-center gap-1"><Clock className="h-4 w-4" /> Ends {new Date(review.end_date).toLocaleDateString()}</span>
                  </div>
                </div>
                <div className="flex gap-2">
                  {review.status === "active" && (
                    <button
                      onClick={() => handleExpand(review.id)}
                      className="rounded-lg border border-gray-300 px-3 py-1.5 text-xs font-medium text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
                    >
                      {expandedId === review.id ? "Hide Items" : "Review Items"}
                    </button>
                  )}
                  <button
                    onClick={() => handleDelete(review.id)}
                    className="rounded-lg p-2 text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
              </div>

              {/* Progress bar */}
              {review.total_items > 0 && (
                <div className="mt-4">
                  <div className="h-2 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
                    <div
                      className="h-full rounded-full bg-indigo-500"
                      style={{ width: `${(review.completed_items / review.total_items) * 100}%` }}
                    />
                  </div>
                </div>
              )}

              {/* Review items */}
              {expandedId === review.id && (
                <div className="mt-4 border-t border-gray-200 pt-4 dark:border-gray-700">
                  {items.length === 0 ? (
                    <p className="text-sm text-gray-400">No items to review.</p>
                  ) : (
                    <div className="space-y-2">
                      {items.map((item) => (
                        <div key={item.id} className="flex items-center justify-between rounded-lg border border-gray-100 p-3 dark:border-gray-700/50">
                          <div className="flex items-center gap-3">
                            <div>
                              <span className="text-sm font-medium text-gray-800 dark:text-gray-200">{item.username}</span>
                              <span className="ml-2 text-xs text-gray-400">{item.email}</span>
                            </div>
                            <div className="flex items-center gap-1">
                              <code className="rounded bg-gray-100 px-1.5 py-0.5 text-xs dark:bg-gray-700">{item.resource_type}</code>
                              <span className="text-xs text-gray-400">{item.role}</span>
                            </div>
                          </div>
                          <div className="flex items-center gap-2">
                            {item.decision !== "pending" && (
                              <span className={`text-xs ${
                                item.decision === "certify" ? "text-green-600" : "text-red-600"
                              }`}>
                                {item.decision === "certify" ? <CheckCircle2 className="inline h-3 w-3" /> : <XCircle className="inline h-3 w-3" />}
                                {" "}{item.decision}
                              </span>
                            )}
                            {item.decision === "pending" && (
                              <>
                                <button
                                  onClick={() => handleDecision(item.id, "certify")}
                                  disabled={decisionLoading === item.id}
                                  className="rounded bg-green-50 px-2 py-1 text-xs font-medium text-green-600 hover:bg-green-100 dark:bg-green-900/20 dark:text-green-400"
                                >
                                  Certify
                                </button>
                                <button
                                  onClick={() => handleDecision(item.id, "revoke")}
                                  disabled={decisionLoading === item.id}
                                  className="rounded bg-red-50 px-2 py-1 text-xs font-medium text-red-600 hover:bg-red-100 dark:bg-red-900/20 dark:text-red-400"
                                >
                                  Revoke
                                </button>
                              </>
                            )}
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

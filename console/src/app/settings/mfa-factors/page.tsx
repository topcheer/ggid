"use client";

import { useState, useEffect, useCallback } from "react";
import { KeyRound, Search, Trash2, Smartphone, Monitor, MessageSquare, ShieldCheck, Clock } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface MFAFactor {
  id: string;
  type: "totp" | "webauthn" | "sms" | "backup";
  label: string;
  enabled: boolean;
  enrolled_at: string;
  last_used: string | null;
}

const typeIcons: Record<string, typeof Smartphone> = {
  totp: Smartphone,
  webauthn: KeyRound,
  sms: MessageSquare,
  backup: ShieldCheck,
};

const typeLabels: Record<string, string> = {
  totp: "Authenticator App",
  webauthn: "Security Key",
  sms: "SMS Code",
  backup: "Backup Codes",
};

export default function MFAFactorsPage() {
  const t = useTranslations();

  const [search, setSearch] = useState("");
  const [factors, setFactors] = useState<MFAFactor[]>([]);
  const [loading, setLoading] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const fetchFactors = useCallback(async (user: string) => {
    if (!user) return;
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/auth/mfa-factors?user=${encodeURIComponent(user)}`, { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const data = await res.json();
        setFactors(data.factors || data || []);
      }
    } catch {
      /* noop */
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (!search) return;
    fetchFactors(search);
  }, [search, fetchFactors]);

  const deleteFactor = async (id: string) => {
    setDeletingId(id);
    try {
      await fetch(`/api/v1/auth/mfa-factors/${id}`, {
        method: "DELETE",
        headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
      });
      setFactors((prev) => prev.filter((f) => f.id !== id));
    } catch {
      /* noop */
    } finally {
      setDeletingId(null);
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><KeyRound className="w-6 h-6 text-blue-500" /> {t("mfaFactors.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">View and remove multi-factor authentication factors per user.</p>
      </div>

      {/* User search */}
      <div className="relative max-w-md">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
        <input aria-label="Search by username or user ID..." type="text" placeholder="Search by username or user ID..." value={search} onChange={(e) => setSearch(e.target.value)} className="w-full pl-9 pr-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" />
      </div>

      {loading && <p className="text-sm text-gray-500">Loading...</p>}

      {/* Factor list */}
      {factors.length > 0 && (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div className="rounded-lg border p-4 dark:border-gray-800">
            <span className="text-sm text-gray-500">Total Factors</span>
            <p className="text-2xl font-bold mt-1">{factors.length}</p>
          </div>
          <div className="rounded-lg border p-4 dark:border-gray-800">
            <span className="text-sm text-gray-500">Enabled</span>
            <p className="text-2xl font-bold mt-1 text-green-600">{factors.filter((f) => f.enabled).length}</p>
          </div>
          <div className="rounded-lg border p-4 dark:border-gray-800">
            <span className="text-sm text-gray-500">Never Used</span>
            <p className="text-2xl font-bold mt-1 text-yellow-600">{factors.filter((f) => !f.last_used).length}</p>
          </div>
          <div className="rounded-lg border p-4 dark:border-gray-800">
            <span className="text-sm text-gray-500">Types</span>
            <p className="text-2xl font-bold mt-1">{new Set(factors.map((f) => f.type)).size}</p>
          </div>
        </div>
      )}

      <div className="rounded-lg border dark:border-gray-800">
        <div className="divide-y dark:divide-gray-800">
          {factors.map((f) => {
            const Icon = typeIcons[f.type] || KeyRound;
            return (
              <div key={f.id} className="px-4 py-3 flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className={`w-10 h-10 rounded-lg flex items-center justify-center ${f.enabled ? "bg-blue-50 dark:bg-blue-900/20 text-blue-500" : "bg-gray-100 dark:bg-gray-800 text-gray-400"}`}>
                    <Icon className="w-5 h-5" />
                  </div>
                  <div>
                    <div className="flex items-center gap-2">
                      <span className="font-medium">{f.label || typeLabels[f.type]}</span>
                      <span className="text-xs text-gray-400">({f.type})</span>
                      <span className={`px-2 py-0.5 rounded text-xs ${f.enabled ? "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400" : "bg-gray-100 text-gray-500 dark:bg-gray-800"}`}>{f.enabled ? "Enabled" : "Disabled"}</span>
                    </div>
                    <div className="flex items-center gap-3 text-xs text-gray-400 mt-0.5">
                      <span className="flex items-center gap-1"><Clock className="w-3 h-3" /> Enrolled {f.enrolled_at}</span>
                      <span>Last used: {f.last_used || "Never"}</span>
                    </div>
                  </div>
                </div>
                <button onClick={() => deleteFactor(f.id)} aria-label={`Delete ${f.type}`} disabled={deletingId === f.id} className="px-3 py-1.5 rounded-lg text-sm font-medium text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20 border border-red-200 dark:border-red-900 disabled:opacity-50 flex items-center gap-1">
                  <Trash2 className="w-4 h-4" />
                  {deletingId === f.id ? "Deleting..." : "Delete"}
                </button>
              </div>
            );
          })}
          {factors.length === 0 && !loading && search && <p className="px-4 py-8 text-center text-sm text-gray-500">No MFA factors found.</p>}
          {factors.length === 0 && !search && <p className="px-4 py-8 text-center text-sm text-gray-500">Search for a user to view their MFA factors.</p>}
        </div>
      </div>
    </div>
  );
}

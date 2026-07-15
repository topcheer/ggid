"use client";

import { useState, useEffect, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  Link2, Unlink, AlertCircle, Loader2, X, Check, Globe,
  Mail, Building2, AppWindow, Apple,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface LinkedAccount {
  id: string;
  provider: string;
  provider_type: "saml" | "oidc" | "social";
  linked_email: string;
  linked_name: string;
  linked_at: string;
  last_login?: string;
  status: "active" | "disabled";
}

const PROVIDER_ICON: Record<string, typeof Mail> = {
  github: Globe,
  google: Globe,
  microsoft: Building2,
  apple: Apple,
  email: Mail,
};

const AVAILABLE_PROVIDERS = [
  { id: "github", name: "GitHub", type: "social" as const },
  { id: "google", name: "Google", type: "social" as const },
  { id: "microsoft", name: "Microsoft", type: "social" as const },
  { id: "apple", name: "Apple", type: "social" as const },
];

export default function AccountLinkingPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [accounts, setAccounts] = useState<LinkedAccount[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [confirmUnlink, setConfirmUnlink] = useState<LinkedAccount | null>(null);
  const [linking, setLinking] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiFetch<{ accounts?: LinkedAccount[]; items?: LinkedAccount[] }>("/api/v1/users/me/linked-accounts").catch(() => null);
      setAccounts(data?.accounts ?? data?.items ?? []);
    } catch {
      setError("Failed to load linked accounts");
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => { load(); }, [load]);

  const handleLink = async (providerId: string) => {
    setLinking(providerId);
    try {
      const resp = await apiFetch<{ redirect_url?: string; authorization_url?: string; url?: string }>(`/api/v1/users/me/linked-accounts/${providerId}/link`, {
        method: "POST",
      });
      const url = resp.redirect_url ?? resp.authorization_url ?? resp.url;
      if (url && typeof window !== "undefined") {
        window.location.href = url;
      }
    } catch {
      setError(`Failed to initiate ${providerId} linking`);
    } finally {
      setLinking(null);
    }
  };

  const handleUnlink = async (id: string) => {
    try {
      await apiFetch(`/api/v1/users/me/linked-accounts/${id}`, { method: "DELETE" });
      setConfirmUnlink(null);
      await load();
    } catch {
      setError("Failed to unlink account");
    }
  };

  const linkedProviderIds = new Set(accounts.map((a) => a.provider));
  const availableToLink = AVAILABLE_PROVIDERS.filter((p) => !linkedProviderIds.has(p.id));
  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
          <Link2 className="h-6 w-6 text-indigo-600" /> Account Linking
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Connect external identity providers for single sign-on and social login.
        </p>
      </div>

      {error && (
        <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}
          <button onClick={() => setError(null)} className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      {loading ? (
        <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>
      ) : (
        <>
          {/* Linked accounts */}
          <div>
            <h2 className="mb-3 text-sm font-semibold uppercase text-gray-500">Connected Accounts</h2>
            {accounts.length === 0 ? (
              <div className={cardCls}>
                <div className="py-12 text-center">
                  <Link2 className="mx-auto h-12 w-12 text-gray-300" />
                  <p className="mt-4 text-sm text-gray-400">No accounts linked yet.</p>
                </div>
              </div>
            ) : (
              <div className="space-y-3">
                {accounts.map((a) => {
                  const Icon = PROVIDER_ICON[a.provider] ?? Mail;
                  return (
                    <div key={a.id} className={cardCls}>
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-3">
                          <div className="rounded-lg bg-gray-100 p-2 dark:bg-gray-700">
                            <Icon className="h-5 w-5 text-gray-600 dark:text-gray-300" />
                          </div>
                          <div>
                            <div className="flex items-center gap-2">
                              <span className="font-medium capitalize text-gray-800 dark:text-gray-200">{a.provider}</span>
                              <span className="rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium uppercase text-gray-500 dark:bg-gray-700">{a.provider_type}</span>
                              {a.status === "disabled" && <span className="rounded-full bg-red-100 px-2 py-0.5 text-xs text-red-700 dark:bg-red-900/30 dark:text-red-400">Disabled</span>}
                            </div>
                            <p className="text-sm text-gray-500 dark:text-gray-400">{a.linked_email}</p>
                            {a.linked_name && a.linked_name !== a.linked_email && <p className="text-xs text-gray-400">{a.linked_name}</p>}
                          </div>
                        </div>
                        <button onClick={() => setConfirmUnlink(a)} className="flex items-center gap-1.5 rounded-lg border border-red-200 px-3 py-1.5 text-xs font-medium text-red-500 hover:bg-red-50 dark:border-red-800 dark:hover:bg-red-900/20">
                          <Unlink className="h-3.5 w-3.5" /> Unlink
                        </button>
                      </div>
                      <div className="mt-2 flex items-center gap-4 text-xs text-gray-400">
                        <span>Linked: {new Date(a.linked_at).toLocaleDateString()}</span>
                        {a.last_login && <span>Last login: {new Date(a.last_login).toLocaleDateString()}</span>}
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>

          {/* Available to link */}
          {availableToLink.length > 0 && (
            <div>
              <h2 className="mb-3 text-sm font-semibold uppercase text-gray-500">Available Providers</h2>
              <div className="grid gap-3 sm:grid-cols-2">
                {availableToLink.map((p) => {
                  const Icon = PROVIDER_ICON[p.id] ?? Link2;
                  return (
                    <div key={p.id} className={`${cardCls} flex items-center justify-between`}>
                      <div className="flex items-center gap-3">
                        <div className="rounded-lg bg-gray-100 p-2 dark:bg-gray-700">
                          <Icon className="h-5 w-5 text-gray-500" />
                        </div>
                        <span className="font-medium text-gray-700 dark:text-gray-300">{p.name}</span>
                      </div>
                      <button
                        onClick={() => handleLink(p.id)}
                        disabled={linking === p.id}
                        className="flex items-center gap-1.5 rounded-lg bg-indigo-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
                      >
                        {linking === p.id ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Link2 className="h-3.5 w-3.5" />}
                        Link
                      </button>
                    </div>
                  );
                })}
              </div>
            </div>
          )}
        </>
      )}

      {/* Unlink confirmation */}
      {confirmUnlink && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setConfirmUnlink(null)}>
          <div className="w-full max-w-sm rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center gap-3">
              <div className="rounded-full bg-red-100 p-2 dark:bg-red-900/30"><Unlink className="h-5 w-5 text-red-600" /></div>
              <div>
                <h2 className="font-semibold text-gray-900 dark:text-white">Unlink {confirmUnlink.provider}?</h2>
                <p className="text-sm text-gray-500">You will no longer be able to sign in with <strong>{confirmUnlink.linked_email}</strong> via this provider.</p>
              </div>
            </div>
            <div className="mt-5 flex justify-end gap-2">
              <button onClick={() => setConfirmUnlink(null)} className="rounded-lg px-4 py-2 text-sm text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700">Cancel</button>
              <button onClick={() => handleUnlink(confirmUnlink.id)} className="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700">Unlink</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

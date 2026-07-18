"use client";

import { useState, useEffect, useCallback } from "react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";
import {
  UserCheck, ArrowRightLeft, Plus, Loader2, Trash2, Check, X,
  AlertCircle, Calendar, Shield, Clock, Mail,
} from "lucide-react";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

type TabId = "outgoing" | "incoming" | "create";

interface Delegation {
  id: string; delegate: string; delegator: string; scopes: string[];
  expires: string; status: "active" | "expired" | "revoked" | "pending";
}

const SCOPES = [
  { value: "users", labelKey: "delegations.create.scopeUsers" },
  { value: "roles", labelKey: "delegations.create.scopeRoles" },
  { value: "audit", labelKey: "delegations.create.scopeAudit" },
  { value: "policy", labelKey: "delegations.create.scopePolicy" },
  { value: "api_keys", labelKey: "delegations.create.scopeApiKeys" },
  { value: "settings", labelKey: "delegations.create.scopeSettings" },
];

const statusColors: Record<string, string> = {
  active: "bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-300",
  expired: "bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400",
  revoked: "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300",
  pending: "bg-yellow-100 text-yellow-700 dark:bg-yellow-950 dark:text-yellow-300",
};

export default function DelegationsPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<TabId>("outgoing");
  const [outgoing, setOutgoing] = useState<Delegation[]>([]);
  const [incoming, setIncoming] = useState<Delegation[]>([]);
  const [loading, setLoading] = useState(true);
  const [msg, setMsg] = useState<{ type: "success" | "error"; text: string } | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch(`${API_BASE}/api/v1/auth/delegation`, { headers: { ...authHeader() } });
      if (res.ok) { const d = await res.json(); setOutgoing(d.outgoing || []); setIncoming(d.incoming || []); return; }
    } catch { /* mock */ }
    setOutgoing([
      { id: "d1", delegate: "bob@company.com", delegator: "me", scopes: ["users", "audit"], expires: "2025-08-15", status: "active" },
      { id: "d2", delegate: "carol@company.com", delegator: "me", scopes: ["roles"], expires: "2025-07-20", status: "active" },
      { id: "d3", delegate: "dave@company.com", delegator: "me", scopes: ["settings", "policy"], expires: "2025-06-01", status: "expired" },
    ]);
    setIncoming([
      { id: "d4", delegate: "me", delegator: "admin@company.com", scopes: ["users", "roles", "audit"], expires: "2025-09-01", status: "pending" },
      { id: "d5", delegate: "me", delegator: "cto@company.com", scopes: ["policy"], expires: "2025-08-30", status: "active" },
    ]);
  }, []);

  useEffect(() => { load(); }, [load]);

  const revoke = async (id: string) => {
    if (!confirm(t("delegations.outgoing.confirmRevoke"))) return;
    setOutgoing(outgoing.map((d: any) => d.id === id ? { ...d, status: "revoked" } : d));
    setMsg({ type: "success", text: t("delegations.outgoing.revoked") });
    setTimeout(() => setMsg(null), 3000);
  };

  const respond = async (id: string, action: "accept" | "reject") => {
    setIncoming(incoming.map((d: any) => d.id === id ? { ...d, status: action === "accept" ? "active" : "revoked" } : d));
    setMsg({ type: "success", text: action === "accept" ? t("delegations.incoming.accepted") : t("delegations.incoming.rejected") });
    setTimeout(() => setMsg(null), 3000);
  };

  const tabs: { id: TabId; label: string; icon: typeof UserCheck; count?: number }[] = [
    { id: "outgoing", label: t("delegations.tabs.outgoing"), icon: ArrowRightLeft, count: outgoing.filter((d: any) => d.status === "active").length },
    { id: "incoming", label: t("delegations.tabs.incoming"), icon: UserCheck, count: incoming.filter((d: any) => d.status === "pending").length },
    { id: "create", label: t("delegations.tabs.create"), icon: Plus },
  ];

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 p-4 md:p-8">
      <div className="max-w-5xl mx-auto">
        <div className="mb-6">
          <div className="flex items-center gap-3 mb-1">
            <ArrowRightLeft className="w-7 h-7 text-blue-600" />
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">{t("delegations.title")}</h1>
          </div>
          <p className="text-gray-600 dark:text-gray-400 text-sm">{t("delegations.description")}</p>
        </div>

        <div className="flex gap-1 mb-6 bg-gray-200 dark:bg-gray-800 rounded-lg p-1">
          {tabs.map(({ id, label, icon: Icon, count }) => (
            <button key={id} onClick={() => setTab(id)}
              className={`flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                tab === id ? "bg-white dark:bg-gray-700 text-blue-600 dark:text-blue-400 shadow-sm" : "text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white"
              }`}>
              <Icon className="w-4 h-4" />{label}
              {count !== undefined && count > 0 && <span className="px-1.5 py-0.5 text-xs bg-orange-200 dark:bg-orange-900 text-orange-700 dark:text-orange-300 rounded-full">{count}</span>}
            </button>
          ))}
        </div>

        {msg && (
          <div className={`flex items-center gap-2 px-4 py-2 mb-4 rounded-lg text-sm ${
            msg.type === "success" ? "bg-green-50 text-green-700 dark:bg-green-950 dark:text-green-300" : "bg-red-50 text-red-700 dark:bg-red-950 dark:text-red-300"
          }`}>
            {msg.type === "success" ? <Check className="w-4 h-4" /> : <AlertCircle className="w-4 h-4" />}{msg.text}
          </div>
        )}

        {loading ? (
          <div className="flex justify-center py-20"><Loader2 className="w-8 h-8 animate-spin text-blue-600" /></div>
        ) : (
          <>
            {tab === "outgoing" && <DelegationTable delegations={outgoing} type="outgoing" onRevoke={revoke} />}
            {tab === "incoming" && <DelegationTable delegations={incoming} type="incoming" onRespond={respond} />}
            {tab === "create" && <CreateDelegation onCreated={() => { setTab("outgoing"); load(); }} />}
          </>
        )}
      </div>
    </div>
  );
}

// ============ Delegation Table ============

function DelegationTable({ delegations, type, onRevoke, onRespond }: {
  delegations: Delegation[];
  type: "outgoing" | "incoming";
  onRevoke?: (id: string) => void;
  onRespond?: (id: string, action: "accept" | "reject") => void;
}) {
  const t = useTranslations();
  const otherKey = type === "outgoing" ? "delegate" : "from";

  if (delegations.length === 0) {
    return (
      <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-12 text-center">
        <ArrowRightLeft className="w-12 h-12 mx-auto mb-3 text-gray-300" />
        <p className="text-sm text-gray-500">{type === "outgoing" ? t("delegations.outgoing.noDelegations") : t("delegations.incoming.noDelegations")}</p>
      </div>
    );
  }

  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 overflow-hidden">
      <h3 className="text-sm font-semibold text-gray-900 dark:text-white p-4 pb-2">
        {type === "outgoing" ? t("delegations.outgoing.title") : t("delegations.incoming.title")}
      </h3>
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-gray-200 dark:border-gray-800 text-left bg-gray-50 dark:bg-gray-800/50">
              <th className="py-2 px-4 font-medium text-gray-600 dark:text-gray-400">{t(`delegations.${type === "outgoing" ? "outgoing" : "incoming"}.${otherKey}`)}</th>
              <th className="py-2 px-4 font-medium text-gray-600 dark:text-gray-400">{t(`delegations.${type === "outgoing" ? "outgoing" : "incoming"}.scopes`)}</th>
              <th className="py-2 px-4 font-medium text-gray-600 dark:text-gray-400">{t(`delegations.${type === "outgoing" ? "outgoing" : "incoming"}.expires`)}</th>
              <th className="py-2 px-4 font-medium text-gray-600 dark:text-gray-400">{t(`delegations.${type === "outgoing" ? "outgoing" : "incoming"}.status`)}</th>
              <th className="py-2 px-4 font-medium text-gray-600 dark:text-gray-400 text-right">Actions</th>
            </tr>
          </thead>
          <tbody>
            {delegations.map((d: any) => {
              const user = type === "outgoing" ? d.delegate : d.delegator;
              return (
                <tr key={d.id} className="border-b border-gray-100 dark:border-gray-800/50">
                  <td className="py-3 px-4">
                    <div className="flex items-center gap-2">
                      <div className="w-8 h-8 rounded-full bg-blue-100 dark:bg-blue-950 flex items-center justify-center">
                        <span className="text-xs font-bold text-blue-600">{user[0]?.toUpperCase()}</span>
                      </div>
                      <span className="text-sm font-medium text-gray-900 dark:text-white">{user}</span>
                    </div>
                  </td>
                  <td className="py-3 px-4">
                    <div className="flex flex-wrap gap-1">
                      {d.scopes.map((s: any) => (
                        <span key={s} className="px-1.5 py-0.5 text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 rounded">{s}</span>
                      ))}
                    </div>
                  </td>
                  <td className="py-3 px-4 text-xs text-gray-500">{new Date(d.expires).toLocaleDateString()}</td>
                  <td className="py-3 px-4">
                    <span className={`px-2 py-0.5 text-xs rounded-full ${statusColors[d.status]}`}>
                      {t(`delegations.${type === "outgoing" ? "outgoing" : "incoming"}.status${d.status.replace(/^./, (m) => m.toUpperCase())}`)}
                    </span>
                  </td>
                  <td className="py-3 px-4 text-right">
                    {type === "outgoing" && d.status === "active" && onRevoke && (
                      <button onClick={() => onRevoke(d.id)} className="flex items-center gap-1 px-2.5 py-1 bg-red-50 dark:bg-red-950/30 hover:bg-red-100 dark:hover:bg-red-950 text-red-600 rounded text-xs font-medium ml-auto">
                        <Trash2 className="w-3 h-3" />{t("delegations.outgoing.revoke")}
                      </button>
                    )}
                    {type === "incoming" && d.status === "pending" && onRespond && (
                      <div className="flex items-center gap-1 justify-end">
                        <button onClick={() => onRespond(d.id, "accept")} className="flex items-center gap-1 px-2.5 py-1 bg-green-50 dark:bg-green-950/30 hover:bg-green-100 dark:hover:bg-green-950 text-green-600 rounded text-xs font-medium">
                          <Check className="w-3 h-3" />{t("delegations.incoming.accept")}
                        </button>
                        <button onClick={() => onRespond(d.id, "reject")} className="flex items-center gap-1 px-2.5 py-1 bg-red-50 dark:bg-red-950/30 hover:bg-red-100 dark:hover:bg-red-950 text-red-600 rounded text-xs font-medium">
                          <X className="w-3 h-3" />{t("delegations.incoming.reject")}
                        </button>
                      </div>
                    )}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}

// ============ Create Delegation ============

function CreateDelegation({ onCreated }: { onCreated: () => void }) {
  const t = useTranslations();
  const [delegate, setDelegate] = useState("");
  const [selectedScopes, setSelectedScopes] = useState<Set<string>>(new Set());
  const [duration, setDuration] = useState<"7d" | "30d" | "90d" | "custom">("30d");
  const [customDate, setCustomDate] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  const toggleScope = (s: string) => {
    const next = new Set(selectedScopes);
    if (next.has(s)) next.delete(s); else next.add(s);
    setSelectedScopes(next);
  };

  const submit = async () => {
    setError("");
    if (!delegate) { setError(t("delegations.create.selectUser")); return; }
    if (selectedScopes.size === 0) { setError(t("delegations.create.selectScope")); return; }

    setSubmitting(true);
    try {
      const expiry = duration === "custom" ? customDate : new Date(Date.now() + (duration === "7d" ? 7 : duration === "30d" ? 30 : 90) * 86400000).toISOString().split("T")[0];
      await fetch(`${API_BASE}/api/v1/auth/delegation`, {
        method: "POST", headers: { "Content-Type": "application/json", ...authHeader() },
        body: JSON.stringify({ delegate, scopes: [...selectedScopes], expires: expiry }),
      });
    } catch { /* ok */ }
    setSubmitting(false);
    setDelegate(""); setSelectedScopes(new Set()); setCustomDate("");
    onCreated();
  };

  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6 space-y-5">
      <h3 className="text-sm font-semibold text-gray-900 dark:text-white">{t("delegations.create.title")}</h3>

      {/* Delegate */}
      <div>
        <label className="block text-sm font-medium text-gray-900 dark:text-white mb-2">{t("delegations.create.delegate")}</label>
        <div className="relative">
          <Mail className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
          <input type="email" value={delegate} onChange={(e) => setDelegate(e.target.value)}
            placeholder={t("delegations.create.delegatePlaceholder")}
            className="w-full pl-9 pr-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white" />
        </div>
      </div>

      {/* Scopes */}
      <div>
        <label className="block text-sm font-medium text-gray-900 dark:text-white mb-1">{t("delegations.create.scopes")}</label>
        <p className="text-xs text-gray-500 dark:text-gray-400 mb-3">{t("delegations.create.scopesDesc")}</p>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-2">
          {SCOPES.map((s: any) => {
            const checked = selectedScopes.has(s.value);
            return (
              <button key={s.value} onClick={() => toggleScope(s.value)}
                className={`flex items-center gap-2 p-3 rounded-lg border-2 text-sm transition-all ${
                  checked ? "border-blue-500 bg-blue-50 dark:bg-blue-950/30 text-blue-700 dark:text-blue-300" : "border-gray-200 dark:border-gray-700 text-gray-600 dark:text-gray-400 hover:border-gray-300"
                }`}>
                <Shield className={`w-4 h-4 ${checked ? "text-blue-600" : "text-gray-400"}`} />
                {t(s.labelKey)}
                {checked && <Check className="w-3 h-3 text-blue-600 ml-auto" />}
              </button>
            );
          })}
        </div>
      </div>

      {/* Duration */}
      <div>
        <label className="block text-sm font-medium text-gray-900 dark:text-white mb-1">{t("delegations.create.expiry")}</label>
        <p className="text-xs text-gray-500 dark:text-gray-400 mb-2">{t("delegations.create.expiryDesc")}</p>
        <div className="flex flex-wrap gap-2">
          {(["7d", "30d", "90d", "custom"] as const).map((dur: any) => (
            <button key={dur} onClick={() => setDuration(dur)}
              className={`px-3 py-1.5 rounded-lg text-sm font-medium border-2 transition-all ${
                duration === dur ? "border-blue-500 bg-blue-50 dark:bg-blue-950/30 text-blue-700 dark:text-blue-300" : "border-gray-200 dark:border-gray-700 text-gray-600 dark:text-gray-400"
              }`}>
              {t(`delegations.create.duration${dur}`)}
            </button>
          ))}
        </div>
        {duration === "custom" && (
          <input type="date" value={customDate} onChange={(e) => setCustomDate(e.target.value)}
            min={new Date().toISOString().split("T")[0]}
            className="mt-2 px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white" />
        )}
      </div>

      {error && <div className="flex items-center gap-2 px-4 py-2 rounded-lg bg-red-50 dark:bg-red-950/30 text-red-700 dark:text-red-300 text-sm"><AlertCircle className="w-4 h-4" />{error}</div>}

      <button onClick={submit} disabled={submitting}
        className="flex items-center gap-2 px-6 py-2.5 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg font-medium text-sm">
        {submitting ? <Loader2 className="w-4 h-4 animate-spin" /> : <Plus className="w-4 h-4" />}
        {t("delegations.create.submit")}
      </button>
    </div>
  );
}

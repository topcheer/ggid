"use client";
import { useState, useCallback, useEffect } from "react";
import {
  Code, Loader2, AlertCircle, X, RefreshCw, Play, History,
  BookOpen, ChevronRight, Clock, Copy, Check, Trash2,
  Zap, Braces, Eye,
} from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";
import { useTranslations } from "@/lib/i18n";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface QueryHistoryItem { id: string; query: string; timestamp: string; success: boolean; }

type Tab = "playground" | "schema" | "history";

const SAMPLE_QUERY = `query GetUsers {
  users(first: 10) {
    edges {
      node {
        id
        email
        displayName
        roles {
          name
        }
        sessions {
          active
        }
      }
    }
    pageInfo {
      hasNextPage
      endCursor
    }
  }
}`;

const SCHEMA_TYPES = [
  { name: "Query", kind: "OBJECT", fields: [
    { name: "users", type: "UserConnection!", desc: "Paginated list of users" },
    { name: "user", type: "User", desc: "Get a single user by ID or email" },
    { name: "organizations", type: "OrganizationConnection!", desc: "Paginated organizations" },
    { name: "policies", type: "PolicyConnection!", desc: "Access policies" },
    { name: "auditEvents", type: "AuditEventConnection!", desc: "Query audit log" },
    { name: "me", type: "User!", desc: "Current authenticated user" },
  ]},
  { name: "Mutation", kind: "OBJECT", fields: [
    { name: "createUser", type: "User!", desc: "Create a new user account" },
    { name: "updateUser", type: "User!", desc: "Update user attributes" },
    { name: "deleteUser", type: "DeleteResult!", desc: "Delete a user" },
    { name: "assignRole", type: "Assignment!", desc: "Assign role to user" },
    { name: "revokeSession", type: "RevokeResult!", desc: "Revoke user session" },
  ]},
  { name: "User", kind: "TYPE", fields: [
    { name: "id", type: "UUID!", desc: "Unique identifier" },
    { name: "email", type: "String!", desc: "Email address" },
    { name: "displayName", type: "String", desc: "Display name" },
    { name: "status", type: "UserStatus!", desc: "active, suspended, archived" },
    { name: "roles", type: "[Role!]!", desc: "Assigned roles" },
    { name: "sessions", type: "[Session!]!", desc: "Active sessions" },
    { name: "mfaEnabled", type: "Boolean!", desc: "MFA enrollment status" },
    { name: "createdAt", type: "DateTime!", desc: "Account creation date" },
  ]},
  { name: "AuditEvent", kind: "TYPE", fields: [
    { name: "id", type: "UUID!", desc: "Event ID" },
    { name: "actor", type: "String!", desc: "User who performed action" },
    { name: "action", type: "String!", desc: "Action type" },
    { name: "resource", type: "String", desc: "Affected resource" },
    { name: "timestamp", type: "DateTime!", desc: "Event time" },
    { name: "ipAddress", type: "String", desc: "Source IP" },
  ]},
];

export default function GraphQLPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("playground");
  const [query, setQuery] = useState(SAMPLE_QUERY);
  const [variables, setVariables] = useState("{}");
  const [response, setResponse] = useState<string>("");
  const [executing, setExecuting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showVariables, setShowVariables] = useState(false);
  const [history, setHistory] = useState<QueryHistoryItem[]>([]);
  const [copied, setCopied] = useState(false);
  const [expandedType, setExpandedType] = useState<string | null>("Query");

  const H = { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID };
  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const executeQuery = async () => {
    setExecuting(true); setResponse(""); setError(null);
    try {
      const res = await fetch("/graphql", { method: "POST", headers: H, body: JSON.stringify({ query, variables: JSON.parse(variables || "{}") }) }).catch(() => null);
      if (res?.ok) {
        const d = await res.json();
        setResponse(JSON.stringify(d, null, 2));
        setHistory(prev => [{ id: `q-${Date.now()}`, query, timestamp: new Date().toISOString(), success: !d.errors }, ...prev].slice(0, 20));
      } else {
        setError(t("graphql.notDeployed"));
        setResponse(JSON.stringify({ error: "GraphQL endpoint not yet deployed", hint: "Backend KB-111/112 in progress" }, null, 2));
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : "Execution failed");
    }
    finally { setExecuting(false); }
  };

  const copyQuery = () => { navigator.clipboard?.writeText(query); setCopied(true); setTimeout(() => setCopied(false), 2000); };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Code className="h-6 w-6 text-pink-500" /> {t("graphql.title")}</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("graphql.subtitle")}</p>
      </div>

      {error && (
        <div role="alert" className="flex items-center gap-2 rounded-lg bg-amber-50 px-4 py-3 text-sm text-amber-700 dark:bg-amber-900/20 dark:text-amber-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "playground" as Tab, label: t("graphql.playground"), icon: Play },
          { id: "schema" as Tab, label: t("graphql.schemaExplorer"), icon: BookOpen },
          { id: "history" as Tab, label: `${t("graphql.history")} (${history.length})`, icon: History },
        ]).map(tb => {
          const Icon = tb.icon;
          return (
            <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id}
              className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === tb.id ? "border-pink-600 text-pink-600 dark:text-pink-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}>
              <Icon className="h-4 w-4" /> {tb.label}
            </button>
          );
        })}
      </div>

      {/* ════ PLAYGROUND ════ */}
      {tab === "playground" && (
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <button onClick={executeQuery} disabled={executing} className="flex items-center gap-2 rounded-lg bg-pink-600 px-4 py-2 text-sm font-medium text-white hover:bg-pink-700 disabled:opacity-50">{executing ? <Loader2 className="h-4 w-4 animate-spin" /> : <Play className="h-4 w-4" />} {t("graphql.run")}</button>
              <button onClick={copyQuery} className="flex items-center gap-1 rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-700">{copied ? <Check className="h-3 w-3 text-green-500" /> : <Copy className="h-3 w-3" />} {t("graphql.copy")}</button>
              <button onClick={() => setShowVariables(!showVariables)} aria-pressed={showVariables} className="flex items-center gap-1 rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-700"><Braces className="h-3 w-3" /> {t("graphql.variables")}</button>
            </div>
            <button onClick={() => setQuery(SAMPLE_QUERY)} className="text-xs text-pink-600 hover:underline">{t("graphql.resetQuery")}</button>
          </div>

          {showVariables && (
            <div className={card + " !p-3"}>
              <p className="mb-1 text-xs font-medium text-gray-400">{t("graphql.variablesJson")}</p>
              <textarea value={variables} onChange={e => setVariables(e.target.value)} rows={3} className="w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-2 py-1 font-mono text-xs" />
            </div>
          )}

          <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            <div>
              <p className="mb-1 text-xs font-semibold uppercase text-gray-400">{t("graphql.queryEditor")}</p>
              <textarea aria-label="GraphQL query" value={query} onChange={e => setQuery(e.target.value)} rows={18} spellCheck={false}
                className="w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 font-mono text-xs leading-relaxed" style={{ tabSize: 2 }} />
            </div>
            <div>
              <p className="mb-1 text-xs font-semibold uppercase text-gray-400">{t("graphql.response")}</p>
              <div className="min-h-[400px] rounded-lg border dark:border-gray-700 dark:bg-gray-900 p-3 font-mono text-xs overflow-auto whitespace-pre-wrap">{response || <span className="text-gray-400">{t("graphql.runToSee")}</span>}</div>
            </div>
          </div>
        </div>
      )}

      {/* ════ SCHEMA ════ */}
      {tab === "schema" && (
        <div className="space-y-3">
          {SCHEMA_TYPES.map(typ => (
            <div key={typ.name} className={card}>
              <button onClick={() => setExpandedType(expandedType === typ.name ? null : typ.name)} className="flex w-full items-center justify-between">
                <div className="flex items-center gap-2">
                  <span className={`px-2 py-0.5 rounded text-xs font-mono font-bold ${typ.kind === "OBJECT" ? "bg-pink-100 dark:bg-pink-900/30 text-pink-600" : "bg-blue-100 dark:bg-blue-900/30 text-blue-600"}`}>{typ.kind}</span>
                  <span className="font-semibold text-sm">{typ.name}</span>
                </div>
                <ChevronRight className={`h-4 w-4 text-gray-400 transition ${expandedType === typ.name ? "rotate-90" : ""}`} />
              </button>
              {expandedType === typ.name && (
                <div className="mt-3 space-y-1">
                  {typ.fields.map(f => (
                    <div key={f.name} className="flex items-start gap-3 rounded-lg border p-2 dark:border-gray-700">
                      <code className="text-xs font-mono text-pink-500 min-w-fit">{f.name}</code>
                      <code className="text-xs font-mono text-blue-500 min-w-fit">{f.type}</code>
                      <span className="text-xs text-gray-400">{f.desc}</span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {/* ════ HISTORY ════ */}
      {tab === "history" && (
        <div>
          {history.length === 0 ? (
            <div className={card}><div className="py-12 text-center"><History className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">{t("graphql.noHistory")}</p></div></div>
          ) : (
            <div className="space-y-2">
              {history.map(h => (
                <div key={h.id} className={`${card} flex items-center justify-between !p-3`}>
                  <div className="flex items-center gap-3 flex-1 min-w-0">
                    <div className={`flex h-8 w-8 items-center justify-center rounded-lg ${h.success ? "bg-green-100 dark:bg-green-900/30" : "bg-red-100 dark:bg-red-900/30"}`}>
                      {h.success ? <Check className="h-4 w-4 text-green-500" /> : <X className="h-4 w-4 text-red-500" />}
                    </div>
                    <div className="min-w-0">
                      <code className="text-xs font-mono truncate block max-w-md">{h.query.split("\n")[0]}</code>
                      <p className="text-xs text-gray-400">{new Date(h.timestamp).toLocaleString()}</p>
                    </div>
                  </div>
                  <div className="flex gap-1">
                    <button onClick={() => { setQuery(h.query); setTab("playground"); }} aria-label="Re-run" className="rounded p-1.5 text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700"><Play className="h-3.5 w-3.5" /></button>
                    <button onClick={() => setHistory(prev => prev.filter(x => x.id !== h.id))} aria-label="Delete" className="rounded p-1.5 text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20"><Trash2 className="h-3.5 w-3.5" /></button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

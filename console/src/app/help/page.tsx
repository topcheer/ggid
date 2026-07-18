"use client";
import { useState } from "react";
import {
  HelpCircle, Loader2, AlertCircle, X, Search, Plus, Check,
  ChevronRight, FileText, Ticket, Activity, CheckCircle2,
  XCircle, Clock, ExternalLink, AlertTriangle, Wrench,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

type Tab = "kb" | "tickets" | "status";

interface Article { id: string; title: string; category: string; readTime: string; }
interface Ticket2 { id: string; subject: string; priority: "low" | "medium" | "high" | "urgent"; status: "open" | "in_progress" | "resolved"; created: string; }
interface ServiceStatus { name: string; status: "operational" | "degraded" | "outage"; uptime: string; }

const ARTICLES: Article[] = [
  { id: "a1", title: "Getting Started with GGID Console", category: "Quick Start", readTime: "5 min" },
  { id: "a2", title: "Configuring Multi-Factor Authentication", category: "Security", readTime: "8 min" },
  { id: "a3", title: "Setting Up SCIM Provisioning", category: "Integration", readTime: "12 min" },
  { id: "a4", title: "Understanding Risk Scores", category: "Security", readTime: "6 min" },
  { id: "a5", title: "Managing OAuth Clients", category: "Integration", readTime: "10 min" },
  { id: "a6", title: "GDPR Data Subject Requests", category: "Compliance", readTime: "7 min" },
  { id: "a7", title: "Audit Log Export Guide", category: "Compliance", readTime: "4 min" },
  { id: "a8", title: "DLP Egress Configuration", category: "Security", readTime: "9 min" },
];

const TICKETS: Ticket2[] = [
  { id: "T-001", subject: "Cannot create new OAuth client", priority: "medium", status: "in_progress", created: new Date(Date.now() - 86400000).toISOString() },
  { id: "T-002", subject: "SCIM sync failing for Slack", priority: "high", status: "open", created: new Date(Date.now() - 3600000).toISOString() },
  { id: "T-003", subject: "Passkey registration error on iOS", priority: "low", status: "resolved", created: new Date(Date.now() - 172800000).toISOString() },
];

const SERVICES: ServiceStatus[] = [
  { name: "auth", status: "operational", uptime: "99.98%" },
  { name: "identity", status: "operational", uptime: "99.99%" },
  { name: "oauth", status: "operational", uptime: "100%" },
  { name: "policy", status: "operational", uptime: "99.97%" },
  { name: "audit", status: "operational", uptime: "100%" },
  { name: "gateway", status: "operational", uptime: "99.95%" },
];

const PRIO_CFG: Record<string, string> = { low: "bg-blue-100 dark:bg-blue-900/30 text-blue-600", medium: "bg-yellow-100 dark:bg-yellow-900/30 text-yellow-600", high: "bg-orange-100 dark:bg-orange-900/30 text-orange-600", urgent: "bg-red-100 dark:bg-red-900/30 text-red-600" };
const STATUS_CFG: Record<string, string> = { operational: "bg-green-100 dark:bg-green-900/30 text-green-600", degraded: "bg-yellow-100 dark:bg-yellow-900/30 text-yellow-600", outage: "bg-red-100 dark:bg-red-900/30 text-red-600" };

export default function HelpPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("kb");
  const [search, setSearch] = useState("");
  const [showTicketForm, setShowTicketForm] = useState(false);
  const [tSubject, setTSubject] = useState("");
  const [tPriority, setTPriority] = useState("medium");

  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const filtered = search ? ARTICLES.filter(a => a.title.toLowerCase().includes(search.toLowerCase()) || a.category.toLowerCase().includes(search.toLowerCase())) : ARTICLES;

  return (
    <div className="space-y-6">
      <div><h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><HelpCircle className="h-6 w-6 text-blue-500" /> {t("help.title")}</h1><p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("help.subtitle")}</p></div>

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([["kb", t("help.knowledgeBase"), FileText], ["tickets", `${t("help.tickets")} (${TICKETS.filter(x => x.status !== "resolved").length})`, Ticket], ["status", t("help.systemStatus"), Activity]] as const).map(([id, label, Icon]) => (
          <button key={id} onClick={() => setTab(id as Tab)} aria-pressed={tab === id} className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === id ? "border-blue-600 text-blue-600 dark:text-blue-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}><Icon className="h-4 w-4" /> {label}</button>
        ))}
      </div>

      {/* KB */}
      {tab === "kb" && (
        <div>
          <div className="mb-4"><div className="relative max-w-md"><Search className="absolute left-3 top-2.5 h-4 w-4 text-gray-400" /><input type="text" value={search} onChange={e => setSearch(e.target.value)} placeholder={t("help.searchArticles")} className="w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 pl-9 pr-3 py-2 text-sm" /></div></div>
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">{filtered.map(a => (
            <a key={a.id} href="#" className={`${card} group flex items-center justify-between hover:shadow-md transition`}>
              <div className="flex items-center gap-3"><FileText className="h-5 w-5 text-blue-400" /><div><h3 className="text-sm font-medium group-hover:text-blue-600">{a.title}</h3><p className="text-xs text-gray-400">{a.category} · {a.readTime}</p></div></div>
              <ChevronRight className="h-4 w-4 text-gray-300 group-hover:text-blue-500" />
            </a>
          ))}</div>
        </div>
      )}

      {/* TICKETS */}
      {tab === "tickets" && (
        <div>
          <div className="mb-4"><button onClick={() => setShowTicketForm(true)} className="flex items-center gap-1 rounded-lg bg-blue-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-blue-700"><Plus className="h-3 w-3" /> {t("help.newTicket")}</button></div>
          <div className="space-y-2">{TICKETS.map(tk => (
            <div key={tk.id} className={`${card} flex items-center justify-between !p-3`}>
              <div className="flex items-center gap-3"><div className="flex h-9 w-9 items-center justify-center rounded-lg bg-blue-100 dark:bg-blue-900/30"><Ticket className="h-4 w-4 text-blue-500" /></div><div><div className="flex items-center gap-2"><code className="text-xs font-mono">{tk.id}</code><span className="text-sm font-medium">{tk.subject}</span></div><p className="text-xs text-gray-400">{new Date(tk.created).toLocaleDateString()}</p></div></div>
              <div className="flex items-center gap-2"><span className={`px-1.5 py-0.5 rounded text-xs font-medium ${PRIO_CFG[tk.priority]}`}>{tk.priority}</span><span className={`px-1.5 py-0.5 rounded text-xs font-medium ${tk.status === "resolved" ? "bg-green-100 dark:bg-green-900/30 text-green-600" : tk.status === "in_progress" ? "bg-yellow-100 dark:bg-yellow-900/30 text-yellow-600" : "bg-blue-100 dark:bg-blue-900/30 text-blue-600"}`}>{tk.status.replace("_", " ")}</span></div>
            </div>
          ))}</div>
        </div>
      )}

      {/* STATUS */}
      {tab === "status" && (
        <div className="space-y-6">
          <div className={`${card} flex items-center justify-between`}>
            <div className="flex items-center gap-3"><div className="flex h-12 w-12 items-center justify-center rounded-full bg-green-100 dark:bg-green-900/30"><CheckCircle2 className="h-6 w-6 text-green-500" /></div><div><h3 className="font-bold text-green-600">{t("help.allOperational")}</h3><p className="text-xs text-gray-400">{t("help.updated")} {new Date().toLocaleTimeString()}</p></div></div>
          </div>
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">{SERVICES.map(s => (
            <div key={s.name} className={`${card} flex items-center justify-between`}><div className="flex items-center gap-3"><div className={`flex h-8 w-8 items-center justify-center rounded-lg ${STATUS_CFG[s.status]}`}><CheckCircle2 className="h-4 w-4" /></div><div><span className="text-sm font-medium font-mono">{s.name}</span><p className="text-xs text-gray-400">{s.uptime}</p></div></div><span className={`h-2.5 w-2.5 rounded-full ${s.status === "operational" ? "bg-green-500" : "bg-yellow-500"} ${s.status === "operational" ? "animate-pulse" : ""}`} /></div>
          ))}</div>
          <div className={card}>
            <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Wrench className="h-4 w-4" /> {t("help.maintenance")}</h3>
            <div className="rounded-lg bg-blue-50 dark:bg-blue-900/20 p-3"><p className="text-xs text-blue-600 dark:text-blue-400">{t("help.noMaintenance")}</p></div>
          </div>
        </div>
      )}

      {showTicketForm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowTicketForm(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <h3 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white"><Plus className="h-5 w-5 text-blue-500" /> {t("help.newTicket")}</h3>
            <div className="mt-4 space-y-3"><div><label className="text-sm font-medium">{t("help.subject")}</label><input type="text" value={tSubject} onChange={e => setTSubject(e.target.value)} placeholder="Describe your issue" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" autoFocus /></div><div><label className="text-sm font-medium">{t("help.priority")}</label><div className="mt-1 flex gap-2">{["low", "medium", "high", "urgent"].map(p => <button key={p} onClick={() => setTPriority(p)} aria-pressed={tPriority === p} className={`rounded-lg border px-3 py-1.5 text-sm ${tPriority === p ? "border-blue-500 bg-blue-50 dark:bg-blue-950/30 text-blue-600" : "border-gray-300 dark:border-gray-700"}`}>{p}</button>)}</div></div></div>
            <div className="mt-4 flex justify-end gap-2"><button onClick={() => setShowTicketForm(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">{t("common.cancel")}</button><button onClick={() => setShowTicketForm(false)} className="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700">{t("help.submit")}</button></div>
          </div>
        </div>
      )}
    </div>
  );
}

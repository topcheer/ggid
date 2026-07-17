"use client";
import { useState, useCallback, useEffect } from "react";
import { Hash, Loader2, AlertCircle, X, RefreshCw, Check, CheckCircle, XCircle, Search, Clock, TrendingUp, Download, Eye, AlertTriangle, Filter, Link2, Zap, FileText, Save } from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface ChainBlock { index: number; event_id: string; prev_hash: string; curr_hash: string; timestamp: string; event_type: string; user: string; verified: boolean; }
interface AuditEvent { id: string; timestamp: string; event_type: string; user_id: string; ip_address: string; resource: string; action: string; tenant_id: string; status: string; }
interface SavedQuery { id: string; name: string; filters: string; }
interface Anomaly { id: string; pattern: string; severity: "high" | "medium" | "low"; count: number; description: string; last_seen: string; }

type Tab = "chain" | "search" | "timeline" | "anomalies" | "export";

export default function AuditExplorerPage() {
  const [tab, setTab] = useState<Tab>("chain");
  const [blocks, setBlocks] = useState<ChainBlock[]>([]);
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [anomalies, setAnomalies] = useState<Anomaly[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  // Search filters
  const [fUser, setFUser] = useState("");
  const [fType, setFType] = useState("");
  const [fIp, setFIp] = useState("");
  const [fText, setFText] = useState("");
  const [savedQueries, setSavedQueries] = useState<SavedQuery[]>([]);
  const [showSaveQuery, setShowSaveQuery] = useState(false);
  const [queryName, setQueryName] = useState("");
  // Timeline
  const [timelineUser, setTimelineUser] = useState("");
  const [timelineEvents, setTimelineEvents] = useState<AuditEvent[]>([]);
  // Export
  const [selectedEvents, setSelectedEvents] = useState<Set<string>>(new Set());
  const [exportFormat, setExportFormat] = useState("pdf");
  const [exporting, setExporting] = useState(false);
  // Actions
  const [verifying, setVerifying] = useState(false);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
      const [cRes, eRes, aRes] = await Promise.all([
        fetch("/api/v1/audit/evidence/chain?limit=20", { headers: h }).catch(() => null),
        fetch("/api/v1/audit/events?page_size=100", { headers: h }).catch(() => null),
        fetch("/api/v1/audit/anomalies", { headers: h }).catch(() => null),
      ]);
      if (cRes?.ok) { const d = await cRes.json(); setBlocks(d.blocks || d.chain || []); }
      if (eRes?.ok) { const d = await eRes.json(); setEvents(d.events || d.items || []); }
      if (aRes?.ok) { const d = await aRes.json(); setAnomalies(d.anomalies || d.items || []); }
    } catch { setError("Failed to load audit data"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const verifyChain = async () => {
    setVerifying(true);
    try {
      const res = await fetch("/api/v1/audit/evidence/chain/verify", { method: "POST", headers: { ...authHeader(), "X-Tenant-ID": TENANT_ID } });
      if (res.ok) { const d = await res.json(); setBlocks(prev => prev.map(b => ({ ...b, verified: d.verified_blocks?.includes(b.index) ?? b.verified }))); }
    } catch { /* noop */ }
    finally { setVerifying(false); }
  };

  const searchEvents = async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams({ page_size: "100" });
      if (fUser) params.set("user_id", fUser);
      if (fType) params.set("event_type", fType);
      if (fIp) params.set("ip_address", fIp);
      if (fText) params.set("q", fText);
      const res = await fetch(`/api/v1/audit/events?${params}`, { headers: { ...authHeader(), "X-Tenant-ID": TENANT_ID } });
      if (res.ok) { const d = await res.json(); setEvents(d.events || d.items || []); }
    } catch { setError("Search failed"); }
    finally { setLoading(false); }
  };

  const loadTimeline = async () => {
    if (!timelineUser) return;
    try {
      const res = await fetch(`/api/v1/audit/events?user_id=${timelineUser}&page_size=50`, { headers: { ...authHeader(), "X-Tenant-ID": TENANT_ID } });
      if (res.ok) { const d = await res.json(); setTimelineEvents(d.events || d.items || []); }
    } catch { /* noop */ }
  };

  const doExport = async () => {
    if (selectedEvents.size === 0) return;
    setExporting(true);
    try {
      const res = await fetch("/api/v1/audit/evidence/export", {
        method: "POST", headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify({ event_ids: Array.from(selectedEvents), format: exportFormat }),
      });
      if (res.ok) { const blob = await res.blob(); const url = URL.createObjectURL(blob); const a = document.createElement("a"); a.href = url; a.download = `audit-evidence.${exportFormat}`; a.click(); }
    } catch { setError("Export failed"); }
    finally { setExporting(false); }
  };

  const toggleSelect = (id: string) => {
    setSelectedEvents(prev => { const n = new Set(prev); n.has(id) ? n.delete(id) : n.add(id); return n; });
  };

  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const filteredEvents = events;
  const hasBroken = blocks.some(b => !b.verified);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
          <Hash className="h-6 w-6 text-indigo-500" /> Audit Chain Explorer
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Hash chain verification, advanced search, user timeline, anomaly detection, and evidence export.
        </p>
      </div>

      {error && (
        <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}
          <button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "chain" as Tab, label: "Hash Chain", icon: Link2 },
          { id: "search" as Tab, label: "Search", icon: Search },
          { id: "timeline" as Tab, label: "Timeline", icon: Clock },
          { id: "anomalies" as Tab, label: "Anomalies", icon: AlertTriangle },
          { id: "export" as Tab, label: "Export", icon: Download },
        ]).map(tb => {
          const Icon = tb.icon;
          return (
            <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id}
              className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === tb.id ? "border-indigo-600 text-indigo-600 dark:text-indigo-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}>
              <Icon className="h-4 w-4" /> {tb.label}
            </button>
          );
        })}
      </div>

      {loading && tab !== "search" ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-500" /></div> : (<>

      {/* HASH CHAIN */}
      {tab === "chain" && (
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              {hasBroken ? (
                <span className="flex items-center gap-1.5 rounded-lg bg-red-50 px-3 py-1.5 text-sm font-medium text-red-700 dark:bg-red-950/20 dark:text-red-400"><XCircle className="h-4 w-4" /> Chain BROKEN — {blocks.filter(b => !b.verified).length} tampered blocks</span>
              ) : (
                <span className="flex items-center gap-1.5 rounded-lg bg-green-50 px-3 py-1.5 text-sm font-medium text-green-700 dark:bg-green-950/20 dark:text-green-400"><CheckCircle className="h-4 w-4" /> Chain INTACT — {blocks.length} blocks verified</span>
              )}
            </div>
            <button onClick={verifyChain} disabled={verifying} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">
              {verifying ? <Loader2 className="h-4 w-4 animate-spin" /> : <Hash className="h-4 w-4" />} Verify Chain
            </button>
          </div>

          <div className={card}>
            <div className="space-y-1">
              {blocks.length === 0 ? (
                <div className="py-8 text-center"><Link2 className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No chain data available.</p></div>
              ) : blocks.map((block, i) => (
                <div key={block.index} className={`flex items-center gap-3 rounded-lg border p-3 ${!block.verified ? "border-red-400 bg-red-50 dark:border-red-700 dark:bg-red-950/20" : "dark:border-gray-700"}`}>
                  <div className={`flex h-8 w-8 items-center justify-center rounded-lg text-xs font-bold ${block.verified ? "bg-green-100 dark:bg-green-900/30 text-green-600" : "bg-red-100 dark:bg-red-900/30 text-red-600"}`}>
                    {block.index}
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="text-xs font-mono text-blue-600 dark:text-blue-400">{block.event_type}</span>
                      <span className="text-xs text-gray-400">{block.user}</span>
                      <span className="text-xs text-gray-400">{new Date(block.timestamp).toLocaleString()}</span>
                      {!block.verified && <AlertTriangle className="h-3.5 w-3.5 text-red-500" />}
                    </div>
                    <div className="mt-1 flex items-center gap-2 text-xs font-mono text-gray-400 overflow-hidden">
                      <span className="truncate">{block.prev_hash.substring(0, 16)}...</span>
                      <ArrowRightSmall />
                      <span className="truncate">{block.curr_hash.substring(0, 16)}...</span>
                    </div>
                  </div>
                  {block.verified ? <CheckCircle className="h-5 w-5 text-green-500 shrink-0" /> : <XCircle className="h-5 w-5 text-red-500 shrink-0" />}
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* SEARCH */}
      {tab === "search" && (
        <div className={card}>
          <div className="mb-4 space-y-3">
            <div className="flex flex-wrap items-center gap-2">
              <input aria-label="Filter user" type="text" value={fUser} onChange={e => setFUser(e.target.value)} placeholder="User..." className="w-32 rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-1.5 text-sm" />
              <input aria-label="Filter type" type="text" value={fType} onChange={e => setFType(e.target.value)} placeholder="Event type..." className="w-32 rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-1.5 text-sm" />
              <input aria-label="Filter IP" type="text" value={fIp} onChange={e => setFIp(e.target.value)} placeholder="IP..." className="w-32 rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-1.5 text-sm font-mono" />
              <div className="relative flex-1 min-w-[200px]">
                <Search className="absolute left-2 top-2 h-4 w-4 text-gray-400" />
                <input aria-label="Full text search" type="text" value={fText} onChange={e => setFText(e.target.value)} placeholder="Full text search..." className="w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 pl-8 pr-3 py-1.5 text-sm" />
              </div>
              <button onClick={searchEvents} className="rounded-lg bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-indigo-700"><Search className="h-4 w-4 inline" /></button>
              <button onClick={() => setShowSaveQuery(true)} className="rounded-lg border border-gray-300 px-3 py-1.5 text-sm dark:border-gray-700"><Save className="h-4 w-4 inline" /></button>
            </div>
            {savedQueries.length > 0 && (
              <div className="flex items-center gap-2">{savedQueries.map(q => <button key={q.id} className="rounded-lg bg-gray-100 dark:bg-gray-700 px-2 py-1 text-xs">{q.name}</button>)}</div>
            )}
          </div>
          {loading ? <div className="flex justify-center py-8"><Loader2 className="h-6 w-6 animate-spin text-indigo-500" /></div> : filteredEvents.length === 0 ? (
            <div className="py-8 text-center"><Search className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No events found.</p></div>
          ) : (
            <div className="overflow-x-auto max-h-[400px] overflow-y-auto"><table className="w-full text-sm">
              <thead className="sticky top-0 bg-gray-50 dark:bg-gray-900/80"><tr>
                <th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">Time</th>
                <th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">Type</th>
                <th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">User</th>
                <th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">IP</th>
                <th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">Resource</th>
                <th scope="col" className="px-3 py-2 text-center text-xs font-medium text-gray-400">Status</th>
              </tr></thead>
              <tbody className="divide-y dark:divide-gray-800">{filteredEvents.map(e => (
                <tr key={e.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                  <td className="px-3 py-2 text-xs text-gray-500 whitespace-nowrap">{new Date(e.timestamp).toLocaleString()}</td>
                  <td className="px-3 py-2"><span className="px-1.5 py-0.5 rounded text-xs font-mono bg-blue-100 dark:bg-blue-900/30 text-blue-600">{e.event_type}</span></td>
                  <td className="px-3 py-2 text-xs font-mono">{e.user_id}</td>
                  <td className="px-3 py-2 text-xs font-mono text-gray-500">{e.ip_address}</td>
                  <td className="px-3 py-2 text-xs font-mono text-gray-500 max-w-[200px] truncate">{e.resource}</td>
                  <td className="px-3 py-2 text-center"><span className={`px-1.5 py-0.5 rounded text-xs ${e.status === "success" ? "bg-green-100 dark:bg-green-900/30 text-green-600" : "bg-red-100 dark:bg-red-900/30 text-red-600"}`}>{e.status}</span></td>
                </tr>
              ))}</tbody>
            </table></div>
          )}
        </div>
      )}

      {/* TIMELINE */}
      {tab === "timeline" && (
        <div className={card}>
          <div className="mb-4 flex items-center gap-2">
            <input aria-label="Timeline user" type="text" value={timelineUser} onChange={e => setTimelineUser(e.target.value)} placeholder="user:alice" className="w-48 rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-1.5 text-sm font-mono" />
            <button onClick={loadTimeline} className="rounded-lg bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-indigo-700"><Clock className="h-4 w-4 inline" /></button>
          </div>
          {timelineEvents.length === 0 ? (
            <div className="py-8 text-center"><Clock className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">Enter a user ID to load their activity timeline.</p></div>
          ) : (
            <div className="relative pl-6">
              <div className="absolute left-2 top-0 bottom-0 w-0.5 bg-gray-200 dark:bg-gray-700" />
              <div className="space-y-3">{timelineEvents.map(ev => (
                <div key={ev.id} className="relative">
                  <div className={`absolute -left-4 top-3 h-3 w-3 rounded-full ring-2 ring-white dark:ring-gray-800 ${ev.status === "success" ? "bg-green-500" : "bg-red-500"}`} />
                  <div className="rounded-lg border p-2 dark:border-gray-700">
                    <div className="flex items-center gap-2">
                      <span className="px-1.5 py-0.5 rounded text-xs font-mono bg-blue-100 dark:bg-blue-900/30 text-blue-600">{ev.event_type}</span>
                      <span className="text-xs text-gray-500">{ev.action} {ev.resource}</span>
                    </div>
                    <p className="mt-1 text-xs text-gray-400">{new Date(ev.timestamp).toLocaleString()} · IP: {ev.ip_address}</p>
                  </div>
                </div>
              ))}</div>
            </div>
          )}
        </div>
      )}

      {/* ANOMALIES */}
      {tab === "anomalies" && (
        <div className={card}>
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><AlertTriangle className="h-4 w-4" /> Detected Anomalies</h2>
          {anomalies.length === 0 ? (
            <div className="py-8 text-center"><CheckCircle className="mx-auto h-10 w-10 text-green-300" /><p className="mt-3 text-sm text-gray-400">No anomalies detected in the last 24h.</p></div>
          ) : (
            <div className="space-y-2">{anomalies.map(a => (
              <div key={a.id} className={`flex items-start gap-3 rounded-lg border p-3 ${a.severity === "high" ? "border-red-300 dark:border-red-700 bg-red-50 dark:bg-red-950/10" : "dark:border-gray-700"}`}>
                <div className={`flex h-7 w-7 items-center justify-center rounded-full ${a.severity === "high" ? "bg-red-100 text-red-600 dark:bg-red-900/30" : a.severity === "medium" ? "bg-yellow-100 text-yellow-600 dark:bg-yellow-900/30" : "bg-blue-100 text-blue-600 dark:bg-blue-900/30"}`}>
                  <AlertTriangle className="h-3.5 w-3.5" />
                </div>
                <div className="flex-1">
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium">{a.pattern}</span>
                    <span className={`px-1.5 py-0.5 rounded text-xs ${a.severity === "high" ? "bg-red-100 text-red-600 dark:bg-red-900/30" : "bg-yellow-100 text-yellow-600 dark:bg-yellow-900/30"}`}>{a.severity}</span>
                    <span className="text-xs text-gray-400">×{a.count}</span>
                  </div>
                  <p className="mt-0.5 text-xs text-gray-400">{a.description}</p>
                  <p className="text-xs text-gray-400">Last seen: {new Date(a.last_seen).toLocaleString()}</p>
                </div>
              </div>
            ))}</div>
          )}
        </div>
      )}

      {/* EXPORT */}
      {tab === "export" && (
        <div className={card}>
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Download className="h-4 w-4" /> Evidence Export</h2>
          <p className="text-sm text-gray-500 mb-3">Select events to export as signed evidence with hash chain proof and timestamp.</p>
          <div className="mb-4 max-h-[300px] overflow-y-auto space-y-1">
            {events.slice(0, 30).map(e => (
              <label key={e.id} className="flex items-center gap-2 rounded-lg border p-2 dark:border-gray-700 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-900/30">
                <input type="checkbox" checked={selectedEvents.has(e.id)} onChange={() => toggleSelect(e.id)} className="rounded" />
                <span className="px-1.5 py-0.5 rounded text-xs font-mono bg-blue-100 dark:bg-blue-900/30 text-blue-600">{e.event_type}</span>
                <span className="text-xs text-gray-400 flex-1">{e.user_id} · {e.action} · {new Date(e.timestamp).toLocaleString()}</span>
              </label>
            ))}
          </div>
          <div className="flex items-center gap-3">
            <select aria-label="Export format" value={exportFormat} onChange={e => setExportFormat(e.target.value)} className="rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">
              <option value="pdf">Signed PDF</option>
              <option value="csv">CSV Bundle</option>
            </select>
            <button onClick={doExport} disabled={selectedEvents.size === 0 || exporting} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">
              {exporting ? <Loader2 className="h-4 w-4 animate-spin" /> : <Download className="h-4 w-4" />} Export {selectedEvents.size} Events
            </button>
          </div>
        </div>
      )}

      </>)}

      {/* Save query dialog */}
      {showSaveQuery && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowSaveQuery(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-sm rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white">Save Query</h3>
            <input aria-label="Query name" type="text" value={queryName} onChange={e => setQueryName(e.target.value)} placeholder="Failed logins from external IPs" className="mt-3 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" autoFocus />
            <div className="mt-4 flex justify-end gap-2">
              <button onClick={() => setShowSaveQuery(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">Cancel</button>
              <button onClick={() => { setSavedQueries([...savedQueries, { id: Date.now().toString(), name: queryName, filters: `${fUser}|${fType}|${fIp}|${fText}` }]); setShowSaveQuery(false); setQueryName(""); }} disabled={!queryName} className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">Save</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function ArrowRightSmall() {
  return <span className="text-gray-300">→</span>;
}

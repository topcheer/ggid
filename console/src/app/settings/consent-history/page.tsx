"use client";

import { useState, useEffect, useCallback } from "react";
import { History, Check, X, Search } from "lucide-react";

interface ConsentEntry {
  id: string;
  action: "granted" | "revoked";
  user_id: string;
  username: string;
  client_id: string;
  client_name: string;
  scopes: string[];
  timestamp: string;
  ip_address: string;
}

export default function ConsentHistoryPage() {
  const [entries, setEntries] = useState<ConsentEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [filter, setFilter] = useState<string>("");
  const [search, setSearch] = useState("");

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/oauth/consent-history", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const d = await res.json(); setEntries(d.entries || d || []); }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const filtered = entries.filter((e) => {
    if (filter && e.action !== filter) return false;
    if (search) {
      const q = search.toLowerCase();
      return e.username.toLowerCase().includes(q) || e.client_name.toLowerCase().includes(q);
    }
    return true;
  });

  const granted = entries.filter((e) => e.action === "granted").length;
  const revoked = entries.filter((e) => e.action === "revoked").length;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2">
          <History className="w-6 h-6 text-blue-500" /> Consent History
        </h1>
        <p className="text-sm text-gray-500 mt-1">Track OAuth consent grants and revocations across all clients.</p>
      </div>

      <div className="grid grid-cols-3 gap-4">
        <div className="rounded-lg border p-4 dark:border-gray-800">
          <span className="text-sm text-gray-500">Total</span>
          <p className="text-xl font-bold mt-1">{entries.length}</p>
        </div>
        <div className="rounded-lg border p-4 dark:border-gray-800">
          <span className="text-sm text-gray-500">Granted</span>
          <p className="text-xl font-bold text-green-600 mt-1">{granted}</p>
        </div>
        <div className="rounded-lg border p-4 dark:border-gray-800">
          <span className="text-sm text-gray-500">Revoked</span>
          <p className="text-xl font-bold text-red-600 mt-1">{revoked}</p>
        </div>
      </div>

      <div className="flex items-center gap-3">
        <div className="flex items-center gap-1">
          <button onClick={() => setFilter("")} className={`px-3 py-1.5 rounded-lg text-xs font-medium ${filter === "" ? "bg-gray-200 dark:bg-gray-800" : "border dark:border-gray-700"}`}>All</button>
          <button onClick={() => setFilter("granted")} className={`px-3 py-1.5 rounded-lg text-xs font-medium ${filter === "granted" ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400" : "border dark:border-gray-700"}`}>Granted</button>
          <button onClick={() => setFilter("revoked")} className={`px-3 py-1.5 rounded-lg text-xs font-medium ${filter === "revoked" ? "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400" : "border dark:border-gray-700"}`}>Revoked</button>
        </div>
        <div className="relative flex-1 max-w-xs">
          <Search className="absolute left-2 top-2.5 w-4 h-4 text-gray-400" />
          <input type="text" value={search} onChange={(e) => setSearch(e.target.value)} placeholder="Search user or client..." className="w-full pl-8 pr-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" />
        </div>
      </div>

      <div className="relative pl-8">
        <div className="absolute left-3 top-0 bottom-0 w-px bg-gray-200 dark:bg-gray-800" />
        <div className="space-y-3">
          {filtered.map((e) => {
            const dotClass = e.action === "granted" ? "bg-green-500 border-green-200" : "bg-red-500 border-red-200";
            return (
              <div key={e.id} className="relative">
                <div className={`absolute -left-5 w-4 h-4 rounded-full border-2 flex items-center justify-center ${dotClass}`}>
                  {e.action === "granted" ? <Check className="w-2 h-2 text-white" /> : <X className="w-2 h-2 text-white" />}
                </div>
                <div className="rounded-lg border dark:border-gray-800 p-3 ml-2">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <span className={`text-xs font-medium ${e.action === "granted" ? "text-green-600" : "text-red-600"}`}>
                        {e.action === "granted" ? "Granted" : "Revoked"}
                      </span>
                      <span className="text-sm font-medium">{e.username}</span>
                      <span className="text-xs text-gray-400">to</span>
                      <span className="text-sm">{e.client_name}</span>
                    </div>
                    <span className="text-xs text-gray-400">{e.timestamp}</span>
                  </div>
                  <div className="mt-2 flex flex-wrap gap-1">
                    {e.scopes.map((s, i) => (
                      <span key={i} className="px-1.5 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800 font-mono">{s}</span>
                    ))}
                  </div>
                  <p className="mt-1 text-xs text-gray-400">IP: {e.ip_address}</p>
                </div>
              </div>
            );
          })}
          {filtered.length === 0 && !loading && (
            <p className="text-sm text-gray-500 py-4 ml-2">No consent entries.</p>
          )}
        </div>
      </div>
    </div>
  );
}

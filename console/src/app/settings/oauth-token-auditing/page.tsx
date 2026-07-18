"use client";

import { useState } from "react";
import { useOAuthTokenAuditing } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";
import { ScrollText, Filter, Download, AlertTriangle, Search } from "lucide-react";

export default function OAuthTokenAuditingPage() {
  const t = useTranslations();
  const { data, loading, error, refresh } = useOAuthTokenAuditing();
  const [filterClient, setFilterClient] = useState("all");
  const [filterUser, setFilterUser] = useState("");

  if (loading) return <div className="p-8 text-gray-400">Loading token audit...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const clients: string[] = Array.from(new Set((data?.audit_trail ?? []).map((t) => t.client)));
  const filtered = (data?.audit_trail ?? []).filter((t) => {
    if (filterClient !== "all" && t.client !== filterClient) return false;
    if (filterUser && !t.user.toLowerCase().includes(filterUser.toLowerCase())) return false;
    return true;
  });

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Token Auditing</h1>
          <p className="text-sm text-gray-400 mt-1">OAuth token lifecycle audit trail and suspicious pattern detection</p>
        </div>
        <div className="flex items-center gap-2">
          <button className="flex items-center gap-1 px-3 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg text-sm font-medium transition">
            <Download className="w-4 h-4" />
            Export
          </button>
          <button
            onClick={refresh}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
          >
            Refresh
          </button>
        </div>
      </div>

      {/* Filters */}
      <div className="bg-gray-900 rounded-xl p-4 mb-6">
        <div className="flex items-center gap-3 flex-wrap">
          <Filter className="w-4 h-4 text-gray-400" />
          <select
            value={filterClient}
            onChange={(e) => setFilterClient(e.target.value)}
            className="bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
          >
            <option value="all">All Clients</option>
            {clients.map((c) => (
              <option key={c} value={c}>{c}</option>
            ))}
          </select>
          <div className="flex items-center gap-1">
            <Search className="w-4 h-4 text-gray-400" />
            <input
              type="text"
              placeholder="Filter by user..."
              value={filterUser}
              onChange={(e) => setFilterUser(e.target.value)}
              className="bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
            />
          </div>
          <span className="text-xs text-gray-500 ml-auto">{filtered.length} records</span>
        </div>
      </div>

      {/* Suspicious Patterns */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
          <AlertTriangle className="w-5 h-5 text-yellow-400" />
          Suspicious Patterns Detected
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
          {(data?.suspicious_patterns ?? []).map((p: any, i: number) => (
            <div key={i} className="bg-gray-800 rounded-lg p-3">
              <div className="flex items-center justify-between mb-1">
                <p className="text-sm font-medium capitalize">{p.pattern_type.replace(/_/g, " ")}</p>
                <span
                  className={"text-xs px-2 py-0.5 rounded " + (
                    p.severity === "critical" ? "bg-red-900 text-red-300" :
                    p.severity === "high" ? "bg-orange-900 text-orange-300" :
                    "bg-yellow-900 text-yellow-300"
                  )}
                >
                  {p.severity}
                </span>
              </div>
              <p className="text-xs text-gray-400">{p.description}</p>
              <p className="text-xs text-gray-500 mt-1">Count: {p.count} - Last: {p.last_seen}</p>
            </div>
          ))}
        </div>
      </div>

      {/* Audit Trail Table */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
          <ScrollText className="w-5 h-5 text-blue-400" />
          Token Audit Trail
        </h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th scope="col" className="text-left py-2 pr-3">Token ID</th>
                <th scope="col" className="text-left py-2 pr-3">Client</th>
                <th scope="col" className="text-left py-2 pr-3">User</th>
                <th scope="col" className="text-left py-2 pr-3">Issued</th>
                <th scope="col" className="text-left py-2 pr-3">Scopes</th>
                <th scope="col" className="text-left py-2 pr-3">Revoked</th>
                <th scope="col" className="text-left py-2 pr-3">By</th>
                <th scope="col" className="text-left py-2 pr-3">Reason</th>
              </tr>
            </thead>
            <tbody>
              {filtered.slice(0, 20).map((t) => (
                <tr key={t.token_id} className="border-b border-gray-800">
                  <td className="py-2 pr-3 font-mono text-xs text-blue-400">{t.token_id.slice(0, 12)}</td>
                  <td className="py-2 pr-3 text-gray-300">{t.client}</td>
                  <td className="py-2 pr-3 text-gray-300">{t.user}</td>
                  <td className="py-2 pr-3 text-gray-400 text-xs">{t.issued_at}</td>
                  <td className="py-2 pr-3">
                    <div className="flex flex-wrap gap-1">
                      {t.scopes.slice(0, 3).map((s) => (
                        <span key={s} className="text-xs px-1.5 py-0.5 bg-gray-700 rounded text-gray-300">{s}</span>
                      ))}
                    </div>
                  </td>
                  <td className="py-2 pr-3 text-gray-400 text-xs">{t.revoked_at ?? "-"}</td>
                  <td className="py-2 pr-3 text-gray-400 text-xs">{t.revoked_by ?? "-"}</td>
                  <td className="py-2 pr-3 text-gray-400 text-xs">{t.revoke_reason ?? "-"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}

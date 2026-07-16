"use client";

import { useState } from "react";
import { usePolicyClauseLibrary } from "@ggid/sdk-react";
import { BookOpen, Plus, Search, FileText, Clock } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function PolicyClauseLibraryPage() {
  const t = useTranslations();

  const { data, loading, error, refresh } = usePolicyClauseLibrary();
  const [showModal, setShowModal] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [filterCategory, setFilterCategory] = useState("all");

  if (loading) return <div className="p-8 text-gray-400">Loading clause library...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const categories: string[] = Array.from(new Set((data?.clauses ?? []).map((c) => c.category)));
  const filtered = (data?.clauses ?? []).filter((c) => {
    if (filterCategory !== "all" && c.category !== filterCategory) return false;
    if (searchQuery && !c.text.toLowerCase().includes(searchQuery.toLowerCase()) && !c.clause_id.toLowerCase().includes(searchQuery.toLowerCase())) return false;
    return true;
  });

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Policy Clause Library</h1>
          <p className="text-sm text-gray-400 mt-1">Reusable, versioned policy clauses for compliance frameworks</p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setShowModal(true)}
            className="flex items-center gap-1 px-3 py-2 bg-green-600 hover:bg-green-700 rounded-lg text-sm font-medium transition"
          >
            <Plus className="w-4 h-4" />
            Add Clause
          </button>
          <button
            onClick={refresh}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
          >
            Refresh
          </button>
        </div>
      </div>

      {/* Category Tabs */}
      <div className="flex items-center gap-2 mb-4">
        <button
          onClick={() => setFilterCategory("all")}
          className={"text-xs px-3 py-1.5 rounded-lg font-medium transition " + (
            filterCategory === "all" ? "bg-blue-600 text-white" : "bg-gray-800 text-gray-400 hover:bg-gray-700"
          )}
        >
          All
        </button>
        {categories.map((cat) => (
          <button
            key={cat}
            onClick={() => setFilterCategory(cat)}
            className={"text-xs px-3 py-1.5 rounded-lg font-medium capitalize transition " + (
              filterCategory === cat ? "bg-blue-600 text-white" : "bg-gray-800 text-gray-400 hover:bg-gray-700"
            )}
          >
            {cat.replace(/_/g, " ")}
          </button>
        ))}
      </div>

      {/* Search */}
      <div className="bg-gray-900 rounded-xl p-4 mb-6">
        <div className="flex items-center gap-2">
          <Search className="w-4 h-4 text-gray-400" />
          <input
            type="text"
            placeholder="Search clauses by ID or text..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="flex-1 bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
          />
          <span className="text-xs text-gray-500 ml-auto">{filtered.length} clauses</span>
        </div>
      </div>

      {/* Clauses Table */}
      <div className="bg-gray-900 rounded-xl p-6">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th scope="col" className="text-left py-2 pr-3">Clause ID</th>
                <th scope="col" className="text-left py-2 pr-3">Category</th>
                <th scope="col" className="text-left py-2 pr-3">Text</th>
                <th scope="col" className="text-left py-2 pr-3">Version</th>
                <th scope="col" className="text-left py-2 pr-3">Used In</th>
              </tr>
            </thead>
            <tbody>
              {filtered.slice(0, 15).map((c) => (
                <tr key={c.clause_id} className="border-b border-gray-800">
                  <td className="py-3 pr-3 font-mono text-xs text-blue-400">{c.clause_id}</td>
                  <td className="py-3 pr-3">
                    <span className="text-xs px-2 py-0.5 rounded bg-gray-700 text-gray-300 capitalize">{c.category.replace(/_/g, " ")}</span>
                  </td>
                  <td className="py-3 pr-3 text-gray-300 text-xs max-w-md truncate">{c.text}</td>
                  <td className="py-3 pr-3 text-gray-300 text-xs">v{c.version}</td>
                  <td className="py-3 pr-3">
                    <span className="text-xs text-gray-400">{c.used_in_policies.length} policies</span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Add Clause Modal */}
      {showModal && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
          <div className="bg-gray-900 rounded-xl p-6 max-w-md w-full mx-4 border border-gray-700">
            <h2 className="text-lg font-bold mb-4">Add Clause</h2>
            <div className="space-y-3">
              <div>
                <label className="text-xs text-gray-400 mb-1 block">Category</label>
                <select className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm">
                  <option value="access_control">Access Control</option>
                  <option value="data_protection">Data Protection</option>
                  <option value="audit">Audit</option>
                  <option value="compliance">Compliance</option>
                </select>
              </div>
              <div>
                <label className="text-xs text-gray-400 mb-1 block">Clause Text</label>
                <textarea rows={3} placeholder="Enter clause text..." className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm" />
              </div>
              <div>
                <label className="text-xs text-gray-400 mb-1 block">Parameters (JSON)</label>
                <textarea rows={2} placeholder="{&quot;key&quot;: &quot;value&quot;}" className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm font-mono" />
              </div>
            </div>
            <div className="flex gap-2 mt-4">
              <button onClick={() => setShowModal(false)} className="flex-1 px-4 py-2 bg-green-600 hover:bg-green-700 rounded-lg text-sm font-medium">Create</button>
              <button onClick={() => setShowModal(false)} className="flex-1 px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg text-sm font-medium">Cancel</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

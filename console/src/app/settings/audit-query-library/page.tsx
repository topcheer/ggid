"use client";

import { useState } from "react";
import { useAuditQueryLibrary } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";
import { Bookmark, Share2, Clock, Tag, Play, Plus, Search } from "lucide-react";

export default function AuditQueryLibraryPage() {
  const t = useTranslations();
  const { data, loading, error, refresh } = useAuditQueryLibrary();
  const [showCreate, setShowCreate] = useState(false);
  const [tagFilter, setTagFilter] = useState("all");

  if (loading) return <div className="p-8 text-gray-400">Loading audit query library...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const allTags: string[] = Array.from(new Set((data?.saved_queries ?? []).flatMap((q: any) => q.tags)));
  const filteredQueries = tagFilter === "all"
    ? (data?.saved_queries ?? [])
    : (data?.saved_queries ?? []).filter((q: any) => q.tags.includes(tagFilter));

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Audit Query Library</h1>
          <p className="text-sm text-gray-400 mt-1">Save, share, and schedule reusable audit queries</p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setShowCreate(!showCreate)}
            className="flex items-center gap-1 px-4 py-2 bg-green-600 hover:bg-green-700 rounded-lg text-sm font-medium transition"
          >
            <Plus className="w-4 h-4" />
            New Query
          </button>
          <button
            onClick={refresh}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
          >
            Refresh
          </button>
        </div>
      </div>

      {/* Create Modal */}
      {showCreate && (
        <div className="bg-gray-900 rounded-xl p-6 mb-6 border border-blue-700">
          <h2 className="text-lg font-semibold mb-4">Create New Query</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
            <input
              type="text"
              placeholder="Query name"
              className="bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
            />
            <input
              type="text"
              placeholder="Description"
              className="bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
            />
          </div>
          <textarea
            placeholder="SQL or filter expression"
            rows={3}
            className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm font-mono focus:outline-none focus:border-blue-500 mb-3"
          />
          <div className="flex items-center gap-2">
            <button className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Save Query</button>
            <button onClick={() => setShowCreate(false)} className="px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg text-sm font-medium transition">Cancel</button>
          </div>
        </div>
      )}

      {/* Tag Filter */}
      <div className="flex items-center gap-2 mb-6 flex-wrap">
        <Tag className="w-4 h-4 text-gray-400" />
        <button
          onClick={() => setTagFilter("all")}
          className={tagFilter === "all" ? "px-3 py-1 rounded-md text-xs font-medium bg-blue-600 text-white" : "px-3 py-1 rounded-md text-xs font-medium bg-gray-800 text-gray-400 hover:bg-gray-700"}
        >
          All
        </button>
        {allTags.map((tag: any) => (
          <button
            key={tag}
            onClick={() => setTagFilter(tag)}
            className={tagFilter === tag ? "px-3 py-1 rounded-md text-xs font-medium bg-blue-600 text-white" : "px-3 py-1 rounded-md text-xs font-medium bg-gray-800 text-gray-400 hover:bg-gray-700"}
          >
            {tag}
          </button>
        ))}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Saved Queries */}
        <div className="lg:col-span-2">
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
            <Bookmark className="w-5 h-5" />
            Saved Queries ({filteredQueries.length})
          </h2>
          <div className="space-y-3">
            {filteredQueries.map((q: any) => (
              <div key={q.id} className="bg-gray-900 rounded-xl p-4">
                <div className="flex items-start justify-between mb-2">
                  <div>
                    <h3 className="font-semibold">{q.name}</h3>
                    <p className="text-xs text-gray-400 mt-0.5">{q.description}</p>
                  </div>
                  <div className="flex items-center gap-1">
                    <button className="flex items-center gap-1 px-2 py-1 bg-gray-700 hover:bg-gray-600 rounded text-xs font-medium transition">
                      <Play className="w-3 h-3" />
                      Run
                    </button>
                    <button className="flex items-center gap-1 px-2 py-1 bg-gray-700 hover:bg-gray-600 rounded text-xs font-medium transition">
                      <Share2 className="w-3 h-3" />
                      Share
                    </button>
                  </div>
                </div>
                <div className="bg-gray-800 rounded-lg p-2 mb-2">
                  <code className="text-xs text-blue-400 font-mono break-all">{q.query_body}</code>
                </div>
                <div className="flex items-center gap-3 flex-wrap">
                  {q.tags.map((tag: any) => (
                    <span key={tag} className="text-xs px-2 py-0.5 rounded bg-gray-800 text-gray-400">{tag}</span>
                  ))}
                  <span className="text-xs text-gray-500 flex items-center gap-1">
                    <Clock className="w-3 h-3" />
                    {q.last_run ?? "Never run"}
                  </span>
                  <span className="text-xs text-gray-500">{q.results_count.toLocaleString()} results</span>
                  {q.schedule && (
                    <span className="text-xs px-2 py-0.5 rounded bg-purple-900 text-purple-300">
                      {q.schedule}
                    </span>
                  )}
                </div>
              </div>
            ))}
            {filteredQueries.length === 0 && (
              <div className="bg-gray-900 rounded-xl p-12 text-center text-gray-500">No queries found.</div>
            )}
          </div>
        </div>

        {/* Popular Queries */}
        <div>
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
            <Search className="w-5 h-5 text-blue-400" />
            Popular Queries
          </h2>
          <div className="space-y-2">
            {(data?.popular_queries ?? []).map((q: any, i: number) => (
              <div key={i} className="bg-gray-900 rounded-lg p-3 cursor-pointer hover:bg-gray-800 transition">
                <div className="flex items-center justify-between">
                  <p className="text-sm font-medium">{q.name}</p>
                  <span className="text-xs text-gray-500">{q.run_count.toLocaleString()} runs</span>
                </div>
                <p className="text-xs text-gray-400 mt-1">{q.description}</p>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

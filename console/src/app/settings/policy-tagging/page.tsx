"use client";

import { useState } from "react";
import { usePolicyTagging } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";
import { Tag, Tags, Plus, Filter, Layers } from "lucide-react";

export default function PolicyTaggingPage() {
  const t = useTranslations();
  const { data, loading, error, refresh } = usePolicyTagging();
  const [selectedTag, setSelectedTag] = useState<string | null>(null);
  const [selectedCategory, setSelectedCategory] = useState<string>("all");

  if (loading) return <div className="p-8 text-gray-400">Loading policy tagging...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const categories: string[] = Array.from(new Set((data?.tag_taxonomy ?? []).map((t: any) => t.category)));

  const filteredTags = (data?.tag_taxonomy ?? []).filter(
    (t) => selectedCategory === "all" || t.category === selectedCategory
  );

  const filteredPolicies = (data?.tagged_policies ?? []).filter(
    (p) => !selectedTag || p.tags.includes(selectedTag)
  );

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Policy Tagging</h1>
          <p className="text-sm text-gray-400 mt-1">Organize policies with tags, bulk assignment, and auto-tag rules</p>
        </div>
        <button
          onClick={refresh}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          Refresh
        </button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
        {/* Tag Filter Sidebar */}
        <div className="lg:col-span-1">
          <div className="bg-gray-900 rounded-xl p-4 mb-4">
            <h2 className="text-sm font-semibold flex items-center gap-2 mb-3">
              <Filter className="w-4 h-4" />
              Category
            </h2>
            <div className="space-y-1">
              <button
                onClick={() => setSelectedCategory("all")}
                className={"w-full text-left px-2 py-1 rounded text-xs " + (
                  selectedCategory === "all" ? "bg-blue-600 text-white" : "bg-gray-800 text-gray-400 hover:bg-gray-700"
                )}
              >
                All Categories
              </button>
              {categories.map((cat: any) => (
                <button
                  key={cat}
                  onClick={() => setSelectedCategory(cat)}
                  className={"w-full text-left px-2 py-1 rounded text-xs capitalize " + (
                    selectedCategory === cat ? "bg-blue-600 text-white" : "bg-gray-800 text-gray-400 hover:bg-gray-700"
                  )}
                >
                  {cat}
                </button>
              ))}
            </div>
          </div>

          <div className="bg-gray-900 rounded-xl p-4">
            <h2 className="text-sm font-semibold flex items-center gap-2 mb-3">
              <Tag className="w-4 h-4" />
              Tags
            </h2>
            <div className="flex flex-wrap gap-1">
              <button
                onClick={() => setSelectedTag(null)}
                className={"text-xs px-2 py-1 rounded " + (
                  selectedTag === null ? "bg-blue-600 text-white" : "bg-gray-800 text-gray-400 hover:bg-gray-700"
                )}
              >
                All
              </button>
              {filteredTags.map((t: any) => (
                <button
                  key={t.name}
                  onClick={() => setSelectedTag(selectedTag === t.name ? null : t.name)}
                  className={"text-xs px-2 py-1 rounded " + (
                    selectedTag === t.name ? "bg-blue-600 text-white" : "bg-gray-800 text-gray-400 hover:bg-gray-700"
                  )}
                >
                  {t.name}
                  <span className="ml-1 text-gray-500">{t.usage_count}</span>
                </button>
              ))}
            </div>
          </div>
        </div>

        {/* Tagged Policies Table */}
        <div className="lg:col-span-3">
          <div className="bg-gray-900 rounded-xl p-6 mb-6">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-semibold">Tagged Policies ({filteredPolicies.length})</h2>
              <button className="flex items-center gap-1 px-3 py-1.5 bg-green-600 hover:bg-green-700 rounded-lg text-xs font-medium transition">
                <Plus className="w-3 h-3" />
                Bulk Tag
              </button>
            </div>
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-gray-800 text-gray-400">
                    <th scope="col" className="text-left py-2 pr-4">Policy</th>
                    <th scope="col" className="text-left py-2 pr-4">Status</th>
                    <th scope="col" className="text-left py-2 pr-4">Tags</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredPolicies.map((p: any) => (
                    <tr key={p.policy_id} className="border-b border-gray-800 hover:bg-gray-800/50">
                      <td className="py-3 pr-4">
                        <p className="font-medium">{p.policy_name}</p>
                        <p className="text-xs text-gray-400 font-mono">{p.policy_id}</p>
                      </td>
                      <td className="py-3 pr-4">
                        <span className={"text-xs px-2 py-0.5 rounded " + (
                          p.status === "active" ? "bg-green-900 text-green-300" :
                          p.status === "draft" ? "bg-gray-700 text-gray-400" :
                          "bg-red-900 text-red-300"
                        )}>
                          {p.status}
                        </span>
                      </td>
                      <td className="py-3 pr-4">
                        <div className="flex flex-wrap gap-1">
                          {p.tags.map((tag: any) => (
                            <span key={tag} className="text-xs px-2 py-0.5 rounded bg-blue-900 text-blue-300">{tag}</span>
                          ))}
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>

          {/* Tag Cloud + Auto-Tag Rules */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="bg-gray-900 rounded-xl p-6">
              <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
                <Tags className="w-5 h-5 text-blue-400" />
                Tag Cloud
              </h2>
              <div className="flex flex-wrap gap-2 items-center">
                {(data?.tag_taxonomy ?? []).map((t: any) => {
                  const size = Math.max(0.75, Math.min(1.8, 0.75 + (t.usage_count / 20)));
                  return (
                    <span
                      key={t.name}
                      className="text-blue-400 hover:text-blue-300 cursor-pointer transition"
                      style={{ fontSize: `${size}rem` }}
                    >
                      {t.name}
                    </span>
                  );
                })}
              </div>
            </div>

            <div className="bg-gray-900 rounded-xl p-6">
              <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
                <Layers className="w-5 h-5 text-purple-400" />
                Auto-Tag Rules
              </h2>
              <div className="space-y-2">
                {(data?.auto_tag_rules ?? []).map((rule: any) => (
                  <div key={rule.id} className="bg-gray-800 rounded-lg p-3">
                    <div className="flex items-center justify-between mb-1">
                      <p className="text-sm font-medium">{rule.name}</p>
                      <span
                        className={"text-xs px-2 py-0.5 rounded " + (
                          rule.enabled ? "bg-green-900 text-green-300" : "bg-gray-700 text-gray-400"
                        )}
                      >
                        {rule.enabled ? "ON" : "OFF"}
                      </span>
                    </div>
                    <p className="text-xs text-gray-400">{rule.condition}</p>
                    <p className="text-xs text-blue-400 mt-1">Tags: {rule.applied_tags.join(", ")}</p>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

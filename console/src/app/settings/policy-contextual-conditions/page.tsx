"use client";

import { useState } from "react";
import { usePolicyContextualConditions } from "@ggid/sdk-react";
import { GitBranch, Plus, Layers, TestTube, Save } from "lucide-react";

export default function PolicyContextualConditionsPage() {
  const { data, loading, error, refresh, testEvaluation } = usePolicyContextualConditions();
  const [selectedCategory, setSelectedCategory] = useState("");

  if (loading) return <div className="p-8 text-gray-400">Loading contextual conditions...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const categoryIcons: Record<string, string> = {
    time: "bg-blue-900 text-blue-300",
    geo: "bg-green-900 text-green-300",
    device: "bg-purple-900 text-purple-300",
    network: "bg-yellow-900 text-yellow-300",
    risk: "bg-red-900 text-red-300",
    behavioral: "bg-orange-900 text-orange-300",
  };

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Contextual Conditions</h1>
          <p className="text-sm text-gray-400 mt-1">Build context-aware access policies with time, geo, device, and risk signals</p>
        </div>
        <button
          onClick={refresh}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          Refresh
        </button>
      </div>

      {/* Condition Categories */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold mb-4">Condition Categories</h2>
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-3">
          {(data?.condition_categories ?? []).map((cat) => (
            <button
              key={cat.name}
              onClick={() => setSelectedCategory(cat.name)}
              className={"p-3 rounded-lg border transition text-left " + (
                selectedCategory === cat.name ? "border-blue-500 bg-gray-800" : "border-gray-700 hover:border-gray-600"
              )}
            >
              <span className={"inline-block text-xs px-2 py-0.5 rounded mb-2 " + (categoryIcons[cat.name] || "bg-gray-700 text-gray-300")}>
                {cat.name}
              </span>
              <p className="text-xs text-gray-400">{cat.available_attributes.length} attributes</p>
            </button>
          ))}
        </div>
      </div>

      {/* Available Attributes for Selected Category */}
      {selectedCategory && (
        <div className="bg-gray-900 rounded-xl p-6 mb-6">
          <h2 className="text-lg font-semibold capitalize mb-4">
            {selectedCategory} Attributes
          </h2>
          <div className="flex flex-wrap gap-2">
            {(data?.condition_categories ?? [])
              .find((c) => c.name === selectedCategory)
              ?.available_attributes.map((attr) => (
                <span
                  key={attr}
                  className="text-xs px-3 py-1.5 bg-gray-800 rounded-lg border border-gray-700 hover:border-blue-500 cursor-pointer"
                >
                  {attr}
                </span>
              ))}
          </div>
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Condition Builder */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
            <GitBranch className="w-5 h-5 text-blue-400" />
            Condition Builder
          </h2>
          <div className="bg-gray-800 rounded-lg p-4">
            <div className="flex items-center gap-2 mb-2">
              <span className="text-xs px-2 py-0.5 bg-blue-900 text-blue-300 rounded">AND</span>
              <button className="text-xs text-gray-400 hover:text-blue-400">+ Add OR</button>
            </div>
            <div className="space-y-2">
              {(data?.condition_categories ?? []).slice(0, 2).flatMap((cat) =>
                cat.available_attributes.slice(0, 1).map((attr) => (
                  <div key={cat.name + attr} className="flex items-center gap-2 bg-gray-900 rounded p-2">
                    <span className={"text-xs px-1.5 py-0.5 rounded " + (categoryIcons[cat.name] || "")}>{cat.name}</span>
                    <span className="text-xs font-mono text-gray-300">{attr}</span>
                    <select className="text-xs bg-gray-700 border border-gray-600 rounded px-1 py-0.5">
                      <option>equals</option>
                      <option>contains</option>
                      <option>in_range</option>
                      <option>greater_than</option>
                    </select>
                    <input
                      type="text"
                      placeholder="value"
                      className="text-xs bg-gray-700 border border-gray-600 rounded px-2 py-0.5 w-24"
                    />
                    <button className="text-xs text-red-400 hover:text-red-300">x</button>
                  </div>
                ))
              )}
            </div>
            <button className="flex items-center gap-1 mt-3 text-xs text-blue-400 hover:text-blue-300">
              <Plus className="w-3 h-3" />
              Add Condition
            </button>
          </div>
          <div className="flex gap-2 mt-3">
            <button
              onClick={() => testEvaluation()}
              className="flex items-center gap-1 px-3 py-1.5 bg-green-600 hover:bg-green-700 rounded text-xs font-medium"
            >
              <TestTube className="w-3 h-3" />
              Test Evaluation
            </button>
            <button className="flex items-center gap-1 px-3 py-1.5 bg-gray-700 hover:bg-gray-600 rounded text-xs font-medium">
              <Save className="w-3 h-3" />
              Save
            </button>
          </div>
        </div>

        {/* Saved Templates */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
            <Layers className="w-5 h-5 text-purple-400" />
            Saved Condition Templates
          </h2>
          <div className="space-y-2">
            {(data?.saved_condition_templates ?? []).map((tmpl, i) => (
              <div key={i} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-center justify-between mb-1">
                  <p className="text-sm font-medium">{tmpl.name}</p>
                  <span className="text-xs text-gray-400">{tmpl.categories.join(", ")}</span>
                </div>
                <p className="text-xs text-gray-400 font-mono">{tmpl.condition_summary}</p>
                <p className="text-xs text-gray-500 mt-1">Used {tmpl.usage_count}x</p>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

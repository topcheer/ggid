"use client";

import { useState } from "react";
import { usePolicyVisualEditor } from "@ggid/sdk-react";
import { Boxes, FileJson, Upload, CheckCircle, Layers, Play } from "lucide-react";

export default function PolicyVisualEditorPage() {
  const { data, loading, error, refresh, validateFlow } = usePolicyVisualEditor();
  const [selectedNode, setSelectedNode] = useState<string | null>(null);

  if (loading) return <div className="p-8 text-gray-400">Loading visual editor...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const nodeColors: Record<string, string> = {
    subject: "bg-blue-900 border-blue-600 text-blue-300",
    condition: "bg-yellow-900 border-yellow-600 text-yellow-300",
    action: "bg-green-900 border-green-600 text-green-300",
  };

  const selectedNodeData = (data?.nodes ?? []).find((n) => n.id === selectedNode);

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold">Policy Visual Editor</h1>
          <p className="text-sm text-gray-400 mt-1">Drag-and-drop node-based policy builder</p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => validateFlow()}
            className="flex items-center gap-1 px-3 py-2 bg-green-600 hover:bg-green-700 rounded-lg text-sm font-medium transition"
          >
            <CheckCircle className="w-4 h-4" />
            Validate
          </button>
          <button className="flex items-center gap-1 px-3 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg text-sm font-medium">
            <Upload className="w-4 h-4" />
            Import
          </button>
          <button
            onClick={refresh}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
          >
            Refresh
          </button>
        </div>
      </div>

      <div className="flex gap-4">
        {/* Node Palette */}
        <div className="w-48 flex-shrink-0">
          <h3 className="text-xs font-semibold text-gray-400 mb-2 uppercase">Nodes</h3>
          <div className="space-y-2">
            {[
              { type: "subject", label: "Subject" },
              { type: "condition", label: "Condition" },
              { type: "action", label: "Action" },
            ].map((node) => (
              <div
                key={node.type}
                className={"p-3 rounded-lg border-2 cursor-grab text-sm font-medium " + nodeColors[node.type]}
              >
                <Boxes className="w-4 h-4 inline mr-1" />
                {node.label}
              </div>
            ))}
          </div>

          {/* Template Gallery */}
          <div className="mt-6">
            <h3 className="text-xs font-semibold text-gray-400 mb-2 uppercase flex items-center gap-1">
              <Layers className="w-3 h-3" />
              Templates
            </h3>
            <div className="space-y-1">
              {(data?.template_gallery ?? []).map((t) => (
                <div key={t.name} className="text-xs px-2 py-1.5 bg-gray-800 rounded hover:bg-gray-700 cursor-pointer">
                  {t.name}
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Canvas */}
        <div className="flex-1 bg-gray-900 rounded-xl p-4 min-h-[500px] relative">
          <div className="absolute inset-0" style={{
            backgroundImage: "radial-gradient(circle, #374151 1px, transparent 1px)",
            backgroundSize: "20px 20px",
          }} />
          <div className="relative grid grid-cols-3 gap-8 pt-12 pb-12">
            {/* Subject Column */}
            <div className="space-y-4">
              <p className="text-xs text-gray-500 text-center uppercase">Subjects</p>
              {(data?.nodes ?? []).filter((n) => n.type === "subject").map((n) => (
                <div
                  key={n.id}
                  onClick={() => setSelectedNode(n.id)}
                  className={"p-3 rounded-lg border-2 cursor-pointer " + nodeColors.subject + (selectedNode === n.id ? " ring-2 ring-white" : "")}
                >
                  <p className="text-sm font-medium">{n.label}</p>
                </div>
              ))}
            </div>
            {/* Condition Column */}
            <div className="space-y-4">
              <p className="text-xs text-gray-500 text-center uppercase">Conditions</p>
              {(data?.nodes ?? []).filter((n) => n.type === "condition").map((n) => (
                <div
                  key={n.id}
                  onClick={() => setSelectedNode(n.id)}
                  className={"p-3 rounded-lg border-2 cursor-pointer " + nodeColors.condition + (selectedNode === n.id ? " ring-2 ring-white" : "")}
                >
                  <p className="text-sm font-medium">{n.label}</p>
                </div>
              ))}
            </div>
            {/* Action Column */}
            <div className="space-y-4">
              <p className="text-xs text-gray-500 text-center uppercase">Actions</p>
              {(data?.nodes ?? []).filter((n) => n.type === "action").map((n) => (
                <div
                  key={n.id}
                  onClick={() => setSelectedNode(n.id)}
                  className={"p-3 rounded-lg border-2 cursor-pointer " + nodeColors.action + (selectedNode === n.id ? " ring-2 ring-white" : "")}
                >
                  <p className="text-sm font-medium">{n.label}</p>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Properties Panel */}
        <div className="w-56 flex-shrink-0">
          <h3 className="text-xs font-semibold text-gray-400 mb-2 uppercase">Properties</h3>
          {selectedNodeData ? (
            <div className="bg-gray-800 rounded-lg p-3 space-y-2">
              <div>
                <p className="text-xs text-gray-400">Node ID</p>
                <p className="text-xs font-mono">{selectedNodeData.id}</p>
              </div>
              <div>
                <p className="text-xs text-gray-400">Type</p>
                <p className="text-xs capitalize">{selectedNodeData.type}</p>
              </div>
              <div>
                <p className="text-xs text-gray-400">Label</p>
                <p className="text-xs">{selectedNodeData.label}</p>
              </div>
              <div>
                <p className="text-xs text-gray-400">Properties</p>
                <pre className="text-xs font-mono text-gray-300 mt-1">{JSON.stringify(selectedNodeData.properties, null, 2)}</pre>
              </div>
            </div>
          ) : (
            <p className="text-xs text-gray-500">Select a node to edit properties</p>
          )}

          {/* Export */}
          <div className="mt-4">
            <button className="w-full flex items-center justify-center gap-1 px-3 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg text-xs font-medium">
              <FileJson className="w-3 h-3" />
              Export as JSON
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

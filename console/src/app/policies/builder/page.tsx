"use client";

import { useState, useCallback, useEffect } from "react";
import { useApi } from "@/lib/api";
import {
  User,
  FolderTree,
  Zap,
  Clock,
  Plus,
  Trash2,
  Save,
  Play,
  AlertCircle,
  CheckCircle,
  XCircle,
  GripVertical,
  FileJson,
  X,
} from "lucide-react";


// Node type definitions
type NodeType = "subject" | "resource" | "action" | "condition";

interface PolicyNode {
  id: string;
  type: NodeType;
  attribute: string;
  operator: string;
  value: string;
}

const NODE_CONFIG: Record<
  NodeType,
  {
    label: string;
    icon: React.ElementType;
    color: string;
    bgColor: string;
    borderColor: string;
    attributes: string[];
    operators?: string[];
  }
> = {
  subject: {
    label: "Subject",
    icon: User,
    color: "text-blue-600 dark:text-blue-400",
    bgColor: "bg-blue-50 dark:bg-blue-900/20",
    borderColor: "border-blue-300 dark:border-blue-700",
    attributes: ["role", "group", "department", "clearance_level"],
    operators: ["equals", "in", "contains"],
  },
  resource: {
    label: "Resource",
    icon: FolderTree,
    color: "text-purple-600 dark:text-purple-400",
    bgColor: "bg-purple-50 dark:bg-purple-900/20",
    borderColor: "border-purple-300 dark:border-purple-700",
    attributes: ["type", "owner", "department", "sensitivity"],
    operators: ["equals", "in", "contains"],
  },
  action: {
    label: "Action",
    icon: Zap,
    color: "text-amber-600 dark:text-amber-400",
    bgColor: "bg-amber-50 dark:bg-amber-900/20",
    borderColor: "border-amber-300 dark:border-amber-700",
    attributes: ["read", "write", "delete", "execute", "manage"],
  },
  condition: {
    label: "Condition",
    icon: Clock,
    color: "text-teal-600 dark:text-teal-400",
    bgColor: "bg-teal-50 dark:bg-teal-900/20",
    borderColor: "border-teal-300 dark:border-teal-700",
    attributes: ["time", "location", "device_trust"],
    operators: ["equals", "in", "contains", "between"],
  },
};

const ALL_OPERATORS = ["equals", "in", "contains", "between"];

let nodeSeq = 0;
function makeNode(type: NodeType): PolicyNode {
  const config = NODE_CONFIG[type];
  return {
    id: `node-${Date.now()}-${nodeSeq++}`,
    type,
    attribute: "",
    operator: config.operators ? config.operators[0] : "",
    value: "",
  };
}

// Syntax highlight JSON
function highlightJSON(obj: unknown): React.ReactNode {
  const json = JSON.stringify(obj, null, 2);
  return json.split("\n").map((line: any, i: any) => {
    const parts = line.split(/("(?:\\.|[^"\\])*"|\b\d+\b)/g);
    return (
      <div key={i} className="whitespace-pre">
        {parts.map((part: any, j: any) => {
          if (/^"(?:\\.|[^"\\])*"$/.test(part)) {
            // Check if it's a key (followed by colon)
            const nextChar = parts[j + 1];
            if (nextChar && nextChar.trim().startsWith(":")) {
              return (
                <span key={j} className="text-blue-500 dark:text-blue-400">
                  {part}
                </span>
              );
            }
            return (
              <span key={j} className="text-green-500 dark:text-green-400">
                {part}
              </span>
            );
          }
          if (/^\d+$/.test(part.trim())) {
            return (
              <span key={j} className="text-orange-500 dark:text-orange-400">
                {part}
              </span>
            );
          }
          if (part === "true" || part === "false" || part === "null") {
            return (
              <span key={j} className="text-purple-500 dark:text-purple-400">
                {part}
              </span>
            );
          }
          return <span key={j}>{part}</span>;
        })}
      </div>
    );
  });
}

export default function PolicyBuilderPage() {
  const { apiFetch } = useApi();
  const [nodes, setNodes] = useState<PolicyNode[]>([]);
  const [policyName, setPolicyName] = useState("");
  const [policyEffect, setPolicyEffect] = useState<"allow" | "deny">("allow");
  const [msg, setMsg] = useState<string | null>(null);
  const [msgType, setMsgType] = useState<"success" | "error">("success");

  // Dry-run state
  const [dryRunOpen, setDryRunOpen] = useState(false);
  const [dryRunResult, setDryRunResult] = useState<{ allow: boolean; detail?: string } | null>(null);
  const [dryRunLoading, setDryRunLoading] = useState(false);
  const [dryRunSubject, setDryRunSubject] = useState("");
  const [dryRunResource, setDryRunResource] = useState("");
  const [dryRunAction, setDryRunAction] = useState("read");

  // Drag state
  const [draggedType, setDraggedType] = useState<NodeType | null>(null);

  useEffect(() => {
    if (msg) {
      const t = setTimeout(() => setMsg(null), 3000);
      return () => clearTimeout(t);
    }
  }, [msg]);

  const addNode = (type: NodeType) => {
    setNodes((prev) => [...prev, makeNode(type)]);
  };

  const deleteNode = (id: string) => {
    setNodes((prev) => prev.filter((n: any) => n.id !== id));
  };

  const updateNode = (id: string, field: keyof PolicyNode, value: string) => {
    setNodes((prev) => prev.map((n: any) => (n.id === id ? { ...n, [field]: value } : n)));
  };

  // Validation
  const validationErrors: { id: string; message: string }[] = [];
  nodes.forEach((node: any) => {
    if (!node.attribute && node.type !== "action") {
      validationErrors.push({ id: node.id, message: "Attribute required" });
    }
    if (!node.value && node.type !== "action") {
      validationErrors.push({ id: node.id, message: "Value required" });
    }
    if (node.type !== "action" && node.operator && !ALL_OPERATORS.includes(node.operator)) {
      validationErrors.push({ id: node.id, message: "Invalid operator" });
    }
    if (node.type === "action" && !node.attribute) {
      validationErrors.push({ id: node.id, message: "Action required" });
    }
  });
  const errorCount = validationErrors.length;

  const getError = (id: string) => validationErrors.find((e: any) => e.id === id)?.message;

  // Generate policy JSON from nodes
  const buildPolicyJSON = useCallback(() => {
    const subjects = nodes.filter((n: any) => n.type === "subject");
    const resources = nodes.filter((n: any) => n.type === "resource");
    const actions = nodes.filter((n: any) => n.type === "action");
    const conditions = nodes.filter((n: any) => n.type === "condition");

    return {
      name: policyName || "Untitled Policy",
      effect: policyEffect,
      rules: [
        {
          subjects: subjects.map((n: any) => ({ attribute: n.attribute, operator: n.operator, value: n.value })),
          resources: resources.map((n: any) => ({ attribute: n.attribute, operator: n.operator, value: n.value })),
          actions: actions.map((n: any) => n.attribute),
          conditions: conditions.map((n: any) => ({
            attribute: n.attribute,
            operator: n.operator,
            value: n.value,
          })),
          effect: policyEffect,
        },
      ],
      nodes: nodes.map((n: any) => ({ type: n.type, attribute: n.attribute, operator: n.operator, value: n.value })),
    };
  }, [nodes, policyName, policyEffect]);

  const policyJSON = buildPolicyJSON();

  const handleSave = async () => {
    if (errorCount > 0) return;
    try {
      await apiFetch("/api/v1/policies", {
        method: "POST",
        body: JSON.stringify(policyJSON),
      });
      setMsg("Policy saved successfully");
      setMsgType("success");
    } catch (err) {
      setMsg(err instanceof Error ? err.message : "Failed to save policy");
      setMsgType("error");
    }
  };

  const handleDryRun = async () => {
    setDryRunLoading(true);
    setDryRunResult(null);
    try {
      const data = await apiFetch<{ allow?: boolean; decision?: string; detail?: string }>(
        "/api/v1/policies/dry-run",
        {
          method: "POST",
          body: JSON.stringify({
            ...policyJSON,
            subject: dryRunSubject || "user:admin",
            resource: dryRunResource || "resource:dashboard",
            action: dryRunAction,
          }),
        },
      );
      const allow = data.allow ?? data.decision === "allow";
      setDryRunResult({ allow, detail: data.detail });
    } catch (err) {
      setDryRunResult({
        allow: false,
        detail: err instanceof Error ? err.message : "Dry-run failed",
      });
    } finally {
      setDryRunLoading(false);
    }
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    if (draggedType) {
      addNode(draggedType);
      setDraggedType(null);
    }
  };

  const inputCls =
    "w-full rounded-md border border-gray-300 px-2 py-1.5 text-xs dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";

  const sidebarNodes: NodeType[] = ["subject", "resource", "action", "condition"];

  return (
    <div>
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
            Policy Visual Builder
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Design ABAC policies with drag-and-drop nodes
          </p>
        </div>
        <div className="flex items-center gap-2">
          {/* Validation badge */}
          {errorCount > 0 ? (
            <span className="flex items-center gap-1.5 rounded-lg bg-red-50 px-3 py-2 text-sm font-medium text-red-600 dark:bg-red-900/20 dark:text-red-400">
              <AlertCircle className="h-4 w-4" /> Validation: {errorCount} error{errorCount !== 1 ? "s" : ""}
            </span>
          ) : (
            <span className="flex items-center gap-1.5 rounded-lg bg-green-50 px-3 py-2 text-sm font-medium text-green-600 dark:bg-green-900/20 dark:text-green-400">
              <CheckCircle className="h-4 w-4" /> Valid
            </span>
          )}
          <button
            onClick={() => setDryRunOpen(true)}
            className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600"
          >
            <Play className="h-4 w-4" /> Dry Run
          </button>
          <button
            onClick={handleSave}
            disabled={errorCount > 0}
            className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            <Save className="h-4 w-4" /> Save Policy
          </button>
        </div>
      </div>

      {/* Message */}
      {msg && (
        <div
          className={`mb-4 rounded-lg border p-3 text-sm ${
            msgType === "success"
              ? "border-green-200 bg-green-50 text-green-700 dark:border-green-800 dark:bg-green-900/20 dark:text-green-400"
              : "border-red-200 bg-red-50 text-red-700 dark:border-red-800 dark:bg-red-900/20 dark:text-red-400"
          }`}
        >
          {msg}
        </div>
      )}

      <div className="flex gap-6">
        {/* Left: Sidebar with node types */}
        <div className="w-48 flex-shrink-0">
          <h3 className="mb-3 text-xs font-semibold uppercase tracking-wide text-gray-400">Node Types</h3>
          <div className="space-y-2">
            {sidebarNodes.map((type: any) => {
              const cfg = NODE_CONFIG[type as keyof typeof NODE_CONFIG];
              const Icon = cfg.icon;
              return (
                <div
                  key={type}
                  draggable
                  onDragStart={() => setDraggedType(type)}
                  onClick={() => addNode(type)}
                  className={`group flex cursor-grab items-center gap-2 rounded-lg border ${cfg.borderColor} ${cfg.bgColor} p-3 transition-all hover:shadow-md active:cursor-grabbing`}
                >
                  <GripVertical className="h-4 w-4 text-gray-300 group-hover:text-gray-400" />
                  <Icon className={`h-4 w-4 ${cfg.color}`} />
                  <span className={`text-sm font-medium ${cfg.color}`}>{cfg.label}</span>
                  <Plus className="ml-auto h-3.5 w-3.5 text-gray-400 group-hover:text-gray-600" />
                </div>
              );
            })}
          </div>

          {/* Policy settings */}
          <div className="mt-6 space-y-3">
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">
                Policy Name
              </label>
              <input
                type="text"
                value={policyName}
                onChange={(e) => setPolicyName(e.target.value)}
                placeholder="My Policy"
                className={inputCls}
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">
                Effect
              </label>
              <select
                value={policyEffect}
                onChange={(e) => setPolicyEffect(e.target.value as "allow" | "deny")}
                className={inputCls}
              >
                <option value="allow">Allow</option>
                <option value="deny">Deny</option>
              </select>
            </div>
          </div>
        </div>

        {/* Center: Canvas */}
        <div
          className="flex-1"
          onDragOver={(e) => e.preventDefault()}
          onDrop={handleDrop}
        >
          {nodes.length === 0 ? (
            <div className="flex min-h-[400px] items-center justify-center rounded-xl border-2 border-dashed border-gray-200 bg-gray-50/50 dark:border-gray-700 dark:bg-gray-800/50">
              <div className="text-center">
                <FileJson className="mx-auto mb-3 h-12 w-12 text-gray-300" />
                <p className="text-sm font-medium text-gray-500 dark:text-gray-400">
                  Drag node types here to start building
                </p>
                <p className="mt-1 text-xs text-gray-400">
                  Or click a node type in the sidebar to add it
                </p>
              </div>
            </div>
          ) : (
            <div className="space-y-3">
              {nodes.map((node: any, idx: any) => {
                const cfg = NODE_CONFIG[node.type as keyof typeof NODE_CONFIG];
                const Icon = cfg.icon;
                const err = getError(node.id);

                return (
                  <div
                    key={node.id}
                    className={`rounded-xl border-2 bg-white p-4 shadow-sm transition-all dark:bg-gray-800 ${
                      err ? "border-red-300 dark:border-red-700" : cfg.borderColor
                    }`}
                  >
                    <div className="flex items-start gap-3">
                      {/* Drag handle + icon */}
                      <div className="flex items-center gap-2">
                        <GripVertical className="h-4 w-4 cursor-grab text-gray-300" />
                        <div
                          className={`flex h-9 w-9 items-center justify-center rounded-lg ${cfg.bgColor}`}
                        >
                          <Icon className={`h-4 w-4 ${cfg.color}`} />
                        </div>
                      </div>

                      {/* Node content */}
                      <div className="flex-1">
                        <div className="mb-2 flex items-center gap-2">
                          <span className={`text-sm font-semibold ${cfg.color}`}>{cfg.label}</span>
                          <span className="text-xs text-gray-400">#{idx + 1}</span>
                        </div>

                        {node.type === "action" ? (
                          // Action node: single dropdown
                          <div className="grid grid-cols-1 gap-2 sm:grid-cols-3">
                            <select
                              value={node.attribute}
                              onChange={(e) => updateNode(node.id, "attribute", e.target.value)}
                              className={inputCls}
                            >
                              <option value="">Select action...</option>
                              {cfg.attributes.map((a: any) => (
                                <option key={a} value={a}>
                                  {a}
                                </option>
                              ))}
                            </select>
                            <div className="sm:col-span-2" />
                          </div>
                        ) : (
                          // Other nodes: attribute + operator + value
                          <div className="grid grid-cols-1 gap-2 sm:grid-cols-3">
                            <select
                              value={node.attribute}
                              onChange={(e) => updateNode(node.id, "attribute", e.target.value)}
                              className={inputCls}
                            >
                              <option value="">Select attribute...</option>
                              {cfg.attributes.map((a: any) => (
                                <option key={a} value={a}>
                                  {a}
                                </option>
                              ))}
                            </select>
                            <select
                              value={node.operator}
                              onChange={(e) => updateNode(node.id, "operator", e.target.value)}
                              className={inputCls}
                            >
                              <option value="">Select operator...</option>
                              {(cfg.operators || ALL_OPERATORS).map((op: any) => (
                                <option key={op} value={op}>
                                  {op}
                                </option>
                              ))}
                            </select>
                            <input
                              type="text"
                              value={node.value}
                              onChange={(e) => updateNode(node.id, "value", e.target.value)}
                              placeholder="Value..."
                              className={inputCls}
                            />
                          </div>
                        )}

                        {/* Validation error */}
                        {err && (
                          <p className="mt-1.5 flex items-center gap-1 text-xs text-red-500">
                            <AlertCircle className="h-3 w-3" /> {err}
                          </p>
                        )}
                      </div>

                      {/* Delete button */}
                      <button
                        onClick={() => deleteNode(node.id)}
                        className="rounded-lg p-1.5 text-gray-400 transition-colors hover:bg-red-50 hover:text-red-500 dark:hover:bg-red-900/20"
                        title="Delete node"
                      >
                        <Trash2 className="h-4 w-4" />
                      </button>
                    </div>
                  </div>
                );
              })}

              {/* Add node quick button */}
              <div className="flex gap-2 pt-2">
                {sidebarNodes.map((type: any) => {
                  const cfg = NODE_CONFIG[type as keyof typeof NODE_CONFIG];
                  const Icon = cfg.icon;
                  return (
                    <button
                      key={type}
                      onClick={() => addNode(type)}
                      className={`flex items-center gap-1.5 rounded-lg border border-dashed ${cfg.borderColor} px-3 py-2 text-xs font-medium ${cfg.color} transition-colors hover:bg-gray-50 dark:hover:bg-gray-800`}
                    >
                      <Plus className="h-3 w-3" />
                      <Icon className="h-3 w-3" />
                      {cfg.label}
                    </button>
                  );
                })}
              </div>
            </div>
          )}
        </div>

        {/* Right: JSON Preview */}
        <div className="w-96 flex-shrink-0">
          <div className="sticky top-0">
            <div className="mb-2 flex items-center gap-2">
              <FileJson className="h-4 w-4 text-gray-400" />
              <h3 className="text-xs font-semibold uppercase tracking-wide text-gray-400">
                JSON Preview
              </h3>
            </div>
            <div className="overflow-auto rounded-xl border border-gray-200 bg-gray-50 p-4 dark:border-gray-700 dark:bg-gray-900" style={{ maxHeight: "600px" }}>
              <pre className="font-mono text-xs leading-relaxed">
                {highlightJSON(policyJSON)}
              </pre>
            </div>
          </div>
        </div>
      </div>

      {/* Dry-run Modal */}
      {dryRunOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
          <div className="w-full max-w-lg rounded-2xl bg-white p-6 shadow-xl dark:bg-gray-800">
            <div className="mb-4 flex items-center justify-between">
              <h2 className="text-lg font-bold text-gray-900 dark:text-gray-100">
                Dry Run Policy
              </h2>
              <button
                onClick={() => {
                  setDryRunOpen(false);
                  setDryRunResult(null);
                }}
                className="rounded-lg p-1 text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700"
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            <p className="mb-4 text-sm text-gray-500 dark:text-gray-400">
              Test your policy against a specific subject, resource, and action.
            </p>

            <div className="space-y-3">
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">
                  Subject
                </label>
                <input
                  type="text"
                  value={dryRunSubject}
                  onChange={(e) => setDryRunSubject(e.target.value)}
                  placeholder="user:admin"
                  className={inputCls + " py-2"}
                />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">
                  Resource
                </label>
                <input
                  type="text"
                  value={dryRunResource}
                  onChange={(e) => setDryRunResource(e.target.value)}
                  placeholder="resource:dashboard"
                  className={inputCls + " py-2"}
                />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">
                  Action
                </label>
                <select
                  value={dryRunAction}
                  onChange={(e) => setDryRunAction(e.target.value)}
                  className={inputCls + " py-2"}
                >
                  <option value="read">read</option>
                  <option value="write">write</option>
                  <option value="delete">delete</option>
                  <option value="execute">execute</option>
                  <option value="manage">manage</option>
                </select>
              </div>
            </div>

            {dryRunResult && (
              <div
                className={`mt-4 flex items-start gap-3 rounded-lg border p-4 ${
                  dryRunResult.allow
                    ? "border-green-200 bg-green-50 dark:border-green-800 dark:bg-green-900/20"
                    : "border-red-200 bg-red-50 dark:border-red-800 dark:bg-red-900/20"
                }`}
              >
                {dryRunResult.allow ? (
                  <CheckCircle className="mt-0.5 h-5 w-5 flex-shrink-0 text-green-500" />
                ) : (
                  <XCircle className="mt-0.5 h-5 w-5 flex-shrink-0 text-red-500" />
                )}
                <div>
                  <p className={`font-semibold ${dryRunResult.allow ? "text-green-700 dark:text-green-400" : "text-red-700 dark:text-red-400"}`}>
                    {dryRunResult.allow ? "ALLOW" : "DENY"}
                  </p>
                  {dryRunResult.detail && (
                    <p className="mt-1 text-xs text-gray-600 dark:text-gray-400">
                      {dryRunResult.detail}
                    </p>
                  )}
                </div>
              </div>
            )}

            <div className="mt-4 flex justify-end gap-2">
              <button
                onClick={() => {
                  setDryRunOpen(false);
                  setDryRunResult(null);
                }}
                className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
              >
                Close
              </button>
              <button
                onClick={handleDryRun}
                disabled={dryRunLoading}
                className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
               aria-label="div">
                {dryRunLoading ? (
                  <>
                    <div className="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent" />
                    Running...
                  </>
                ) : (
                  <>
                    <Play className="h-4 w-4" /> Run Test
                  </>
                )}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

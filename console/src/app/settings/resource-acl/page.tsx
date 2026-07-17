"use client";

import React, { useEffect, useState } from "react";
import { useApi } from "@/lib/api";
import {
  FolderTree, Loader2, AlertCircle, X, Plus, Trash2, Save, ChevronRight, Folder, File,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface ACLRule {
  id: string;
  resource_path: string;
  principal: string;
  principal_type: "user" | "role" | "group" | "service";
  effect: "allow" | "deny";
  permissions: string[];
  conditions: string;
  inherited: boolean;
  created_at: string;
}

interface TreeNode {
  path: string;
  name: string;
  is_dir: boolean;
  children: TreeNode[];
}

const effectColors: Record<string, string> = {
  allow: "text-green-600 bg-green-100 dark:bg-green-900/30 dark:text-green-400",
  deny: "text-red-600 bg-red-100 dark:bg-red-900/30 dark:text-red-400",
};

const principalTypeColors: Record<string, string> = {
  user: "text-blue-600 bg-blue-100 dark:bg-blue-900/30 dark:text-blue-400",
  role: "text-purple-600 bg-purple-100 dark:bg-purple-900/30 dark:text-purple-400",
  group: "text-cyan-600 bg-cyan-100 dark:bg-cyan-900/30 dark:text-cyan-400",
  service: "text-orange-600 bg-orange-100 dark:bg-orange-900/30 dark:text-orange-400",
};

export default function ResourceACLPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [rules, setRules] = useState<ACLRule[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedPath, setSelectedPath] = useState<string | null>(null);
  const [editing, setEditing] = useState<ACLRule | null>(null);
  const [expandedPaths, setExpandedPaths] = useState<Set<string>>(new Set(["/"]));

  useEffect(() => {
    (async () => {
      try { setRules(await apiFetch<ACLRule[]>("/api/v1/policy/resource-acl").catch(() => [])); }
      catch { setError("Failed to load resource ACLs"); }
      finally { setLoading(false); }
    })();
  }, []);

  // Build tree from rules' resource paths
  const allPaths = Array.from(new Set(["/", ...rules.map((r) => r.resource_path)]));
  const tree: TreeNode = { path: "/", name: "root", is_dir: true, children: [] };
  const pathToNode = new Map<string, TreeNode>([["/", tree]]);
  for (const p of allPaths) {
    if (p === "/") continue;
    const parts = p.replace(/^\//, "").split("/");
    let curr = tree;
    let currPath = "";
    for (const part of parts) {
      currPath += "/" + part;
      let child = curr.children.find((c) => c.name === part);
      if (!child) {
        child = { path: currPath, name: part, is_dir: !part.includes("."), children: [] };
        curr.children.push(child);
        pathToNode.set(currPath, child);
      }
      curr = child;
    }
  }

  const sortTree = (node: TreeNode) => { node.children.sort((a, b) => a.is_dir === b.is_dir ? a.name.localeCompare(b.name) : a.is_dir ? -1 : 1); node.children.forEach(sortTree); };
  sortTree(tree);

  const toggleExpand = (path: string) => setExpandedPaths((prev) => { const n = new Set(prev); n.has(path) ? n.delete(path) : n.add(path); return n; });

  const handleSave = async () => {
    if (!editing) return;
    try {
      if (editing.id) {
        await apiFetch(`/api/v1/policy/resource-acl/${editing.id}`, { method: "PUT", body: JSON.stringify(editing) });
      } else {
        const created = await apiFetch<ACLRule>("/api/v1/policy/resource-acl", { method: "POST", body: JSON.stringify(editing) });
        setRules((p) => [...p, created]);
      }
      setEditing(null);
      setRules(await apiFetch<ACLRule[]>("/api/v1/policy/resource-acl").catch(() => rules));
    } catch { setError("Save failed"); }
  };

  const handleDelete = async (id: string) => {
    try { await apiFetch(`/api/v1/policy/resource-acl/${id}`, { method: "DELETE" }); setRules((p) => p.filter((r) => r.id !== id)); }
    catch { setError("Delete failed"); }
  };

  const renderTree = (node: TreeNode, depth: number): React.ReactNode => {
    if (depth > 0 && !expandedPaths.has(node.path.split("/").slice(0, -1).join("/") || "/") && depth > 1) return null;
    return (
      <div key={node.path}>
        <div className={`flex items-center gap-1 rounded px-2 py-1 ${selectedPath === node.path ? "bg-indigo-50 dark:bg-indigo-900/20" : "hover:bg-gray-50 dark:hover:bg-gray-800"}`} style={{ paddingLeft: `${depth * 16 + 8}px` }}>
          {node.children.length > 0 ? <button onClick={() => toggleExpand(node.path)}><ChevronRight className={`h-3 w-3 text-gray-400 transition-transform ${expandedPaths.has(node.path) ? "rotate-90" : ""}`} /></button> : <span className="w-3" />}
          {node.is_dir ? <Folder className="h-4 w-4 text-blue-400" /> : <File className="h-4 w-4 text-gray-400" />}
          <button onClick={() => setSelectedPath(node.path)} className="flex-1 text-left text-sm text-gray-700 dark:text-gray-300">{node.name}</button>
        </div>
        {expandedPaths.has(node.path) && node.children.map((c) => renderTree(c, depth + 1))}
      </div>
    );
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const pathRules = selectedPath ? rules.filter((r) => r.resource_path === selectedPath) : rules;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><FolderTree className="h-6 w-6 text-emerald-600" /> {t("resourceAcl.title")}</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Path-based access control with inherited rules and per-node ACLs.</p>
        </div>
        <button onClick={() => setEditing({ id: "", resource_path: selectedPath || "/", principal: "", principal_type: "role", effect: "allow", permissions: [], conditions: "", inherited: false, created_at: "" })} className="flex items-center gap-2 rounded-lg bg-emerald-600 px-4 py-2 text-sm font-medium text-white hover:bg-emerald-700"><Plus className="h-4 w-4" /> Add Rule</button>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-emerald-600" /></div>
      : (
        <div className="grid grid-cols-3 gap-6">
          {/* Resource tree */}
          <div className={cardCls}>
            <h3 className="mb-3 text-xs font-semibold uppercase text-gray-400">Resource Tree</h3>
            <div className="max-h-96 overflow-y-auto">{renderTree(tree, 0)}</div>
          </div>

          {/* ACL list */}
          <div className="col-span-2">
            <h3 className="mb-3 text-sm font-semibold uppercase text-gray-500">ACL Rules {selectedPath && selectedPath !== "/" ? `for ${selectedPath}` : "(all)"}</h3>
            {pathRules.length === 0 ? (
              <div className={cardCls}><div className="py-8 text-center"><FolderTree className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No ACL rules for this resource.</p></div></div>
            ) : (
              <div className="space-y-2">
                {pathRules.map((r) => (
                  <div key={r.id} className={`${cardCls} py-3`}>
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-3">
                        <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${effectColors[r.effect]}`}>{r.effect}</span>
                        <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${principalTypeColors[r.principal_type] || ""}`}>{r.principal_type}</span>
                        <span className="font-mono text-sm text-gray-900 dark:text-white">{r.principal}</span>
                        {r.inherited && <span className="text-xs text-gray-400">inherited</span>}
                      </div>
                      <div className="flex items-center gap-3">
                        {r.permissions.length > 0 && <div className="flex gap-1">{r.permissions.map((p) => <span key={p} className="rounded bg-gray-100 px-1.5 py-0.5 text-xs text-gray-500 dark:bg-gray-700">{p}</span>)}</div>}
                        <button onClick={() => setEditing({ ...r })} className="text-xs text-indigo-600 hover:underline">Edit</button>
                        <button onClick={() => handleDelete(r.id)} className="text-red-400 hover:text-red-600"><Trash2 className="h-3 w-3" /></button>
                      </div>
                    </div>
                    <div className="mt-1 text-xs text-gray-400">{r.resource_path}{r.conditions && ` · ${r.conditions}`}</div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}

      {/* Edit modal */}
      {editing && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setEditing(null)}>
          <div role="dialog" aria-modal="true" className="w-full max-w-lg rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="mb-4 flex items-center justify-between"><h3 className="text-lg font-bold text-gray-900 dark:text-white">{editing.id ? "Edit Rule" : "New ACL Rule"}</h3><button onClick={() => setEditing(null)} aria-label="Close"><X className="h-5 w-5 text-gray-400" /></button></div>
            <div className="space-y-4">
              <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Resource Path</label><input aria-label="/api/v1/users/*" value={editing.resource_path} onChange={(e) => setEditing({ ...editing, resource_path: e.target.value })} placeholder="/api/v1/users/*" className="w-full rounded-lg border border-gray-300 px-3 py-2 font-mono text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
              <div className="flex gap-4">
                <div className="flex-1"><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Principal</label><input aria-label="role:admin" value={editing.principal} onChange={(e) => setEditing({ ...editing, principal: e.target.value })} placeholder="role:admin" className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
                <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Type</label><select aria-label="editing" value={editing.principal_type} onChange={(e) => setEditing({ ...editing, principal_type: e.target.value as ACLRule["principal_type"] })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200"><option value="user">User</option><option value="role">Role</option><option value="group">Group</option><option value="service">Service</option></select></div>
                <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Effect</label><select aria-label="editing" value={editing.effect} onChange={(e) => setEditing({ ...editing, effect: e.target.value as "allow" | "deny" })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200"><option value="allow">Allow</option><option value="deny">Deny</option></select></div>
              </div>
              <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Permissions (comma-separated)</label><input aria-label="read, write" value={editing.permissions.join(", ")} onChange={(e) => setEditing({ ...editing, permissions: e.target.value.split(",").map((s) => s.trim()).filter(Boolean) })} placeholder="read, write" className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
              <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Conditions</label><input aria-label="time >= 9am AND time <= 17pm" value={editing.conditions} onChange={(e) => setEditing({ ...editing, conditions: e.target.value })} placeholder="time >= 9am AND time <= 17pm" className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
              <button onClick={handleSave} className="flex w-full items-center justify-center gap-2 rounded-lg bg-emerald-600 py-2 text-sm font-medium text-white hover:bg-emerald-700"><Save className="h-4 w-4" /> Save Rule</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

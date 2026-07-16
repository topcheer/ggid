"use client";

import { useEffect, useState, useCallback, useRef } from "react";
import { useApi } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import {
  Building2,
  Plus,
  ChevronDown,
  ChevronRight,
  X,
  Users,
  Search,
  Trash2,
  Edit3,
  ArrowUp,
  ArrowDown,
  FolderPlus,
} from "lucide-react";

// ===== Types =====

interface Organization {
  id: string;
  name: string;
  path: string;
  parent_id?: string;
  description?: string;
}

interface Member {
  id: string;
  user_id: string;
  org_id: string;
  status: string;
  title: string;
}

interface OrgNode extends Organization {
  children: OrgNode[];
  totalMembers: number;
}

// ===== Tree builder =====

function buildTree(orgs: Organization[], counts: Record<string, number>): OrgNode[] {
  const map = new Map<string, OrgNode>();
  const roots: OrgNode[] = [];

  for (const org of orgs) {
    map.set(org.id, { ...org, children: [], totalMembers: counts[org.id] || 0 });
  }
  for (const org of orgs) {
    const node = map.get(org.id)!;
    if (org.parent_id && map.has(org.parent_id)) {
      map.get(org.parent_id)!.children.push(node);
    } else {
      roots.push(node);
    }
  }

  // Propagate member counts up the tree (inherited from children)
  const propagate = (node: OrgNode): number => {
    let sum = node.totalMembers;
    for (const child of node.children) {
      sum += propagate(child);
    }
    node.totalMembers = sum;
    return sum;
  };
  roots.forEach(propagate);

  return roots;
}

// Find a node in the tree by id
function findNode(nodes: OrgNode[], id: string): OrgNode | null {
  for (const node of nodes) {
    if (node.id === id) return node;
    const found = findNode(node.children, id);
    if (found) return found;
  }
  return null;
}

// Check if targetId is a descendant of sourceId
function isDescendant(node: OrgNode, targetId: string): boolean {
  if (node.id === targetId) return true;
  return node.children.some((c) => isDescendant(c, targetId));
}

// Get flat ordered list of orgs at same depth level (siblings)
function getSiblings(nodes: OrgNode[], parentId: string | undefined, excludeId?: string): OrgNode[] {
  let siblings: OrgNode[];
  if (!parentId) {
    siblings = nodes;
  } else {
    const parent = findNode(nodes, parentId);
    siblings = parent ? parent.children : [];
  }
  return siblings.filter((n) => n.id !== excludeId);
}

// ===== Context Menu =====

interface ContextMenuState {
  x: number;
  y: number;
  orgId: string;
}

// ===== Main Component =====

export default function OrganizationTreePage() {
  const { apiFetch } = useApi();
  const [orgs, setOrgs] = useState<Organization[]>([]);
  const [memberCounts, setMemberCounts] = useState<Record<string, number>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [msg, setMsg] = useState<string | null>(null);

  // Expand/collapse state
  const [expanded, setExpanded] = useState<Set<string>>(new Set());

  // Drag-and-drop state
  const [draggingId, setDraggingId] = useState<string | null>(null);
  const [dragOverId, setDragOverId] = useState<string | null>(null);

  // Search filter
  const [search, setSearch] = useState("");
  const [matchedIds, setMatchedIds] = useState<Set<string>>(new Set());

  // Context menu
  const [contextMenu, setContextMenu] = useState<ContextMenuState | null>(null);

  // Modal state (create / rename)
  const [modalMode, setModalMode] = useState<"create-root" | "create-child" | "rename" | null>(null);
  const [modalOrgId, setModalOrgId] = useState<string | null>(null);
  const [modalName, setModalName] = useState("");
  const [modalDesc, setModalDesc] = useState("");
  const [saving, setSaving] = useState(false);

  const contextMenuRef = useRef<HTMLDivElement>(null);

  // ---- Data loading ----
  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const data = await apiFetch<{ organizations?: Organization[] }>("/api/v1/orgs");
      const list = data.organizations || [];
      setOrgs(list);

      // Fetch member counts in parallel
      const counts: Record<string, number> = {};
      await Promise.all(
        list.map(async (org) => {
          try {
            const memData = await apiFetch<{ members?: Member[] }>(`/api/v1/orgs/${org.id}/members`);
            counts[org.id] = memData.members?.length || 0;
          } catch {
            counts[org.id] = 0;
          }
        }),
      );
      setMemberCounts(counts);

      // Default: expand root orgs
      const roots = list.filter((o) => !o.parent_id);
      setExpanded(new Set(roots.map((r) => r.id)));
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load organizations");
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  // Auto-dismiss messages
  useEffect(() => {
    if (msg) {
      const t = setTimeout(() => setMsg(null), 3000);
      return () => clearTimeout(t);
    }
  }, [msg]);

  // Close context menu on outside click
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (contextMenuRef.current && !contextMenuRef.current.contains(e.target as Node)) {
        setContextMenu(null);
      }
    };
    if (contextMenu) {
      document.addEventListener("mousedown", handler);
      return () => document.removeEventListener("mousedown", handler);
    }
  }, [contextMenu]);

  // ---- Search filter ----
  useEffect(() => {
    if (!search.trim()) {
      setMatchedIds(new Set());
      return;
    }
    const q = search.toLowerCase();
    const matched = new Set<string>();
    orgs.forEach((org) => {
      if (org.name.toLowerCase().includes(q) || (org.description || "").toLowerCase().includes(q)) {
        matched.add(org.id);
      }
    });
    setMatchedIds(matched);
    // Auto-expand ancestors of matched nodes
    if (matched.size > 0) {
      const newExpanded = new Set(expanded);
      matched.forEach((id) => {
        let current = orgs.find((o) => o.id === id);
        while (current?.parent_id) {
          newExpanded.add(current.parent_id);
          current = orgs.find((o) => o.id === current!.parent_id);
        }
      });
      setExpanded(newExpanded);
    }
  }, [search, orgs]); // eslint-disable-line react-hooks/exhaustive-deps

  // ---- Build tree ----
  const tree = buildTree(orgs, memberCounts);
  const orgMap = new Map(orgs.map((o) => [o.id, o]));

  // ---- Expand/collapse toggle ----
  const toggleExpand = (id: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  // ---- Drag and Drop ----
  const handleDragStart = (e: React.DragEvent, orgId: string) => {
    setDraggingId(orgId);
    e.dataTransfer.effectAllowed = "move";
    e.dataTransfer.setData("text/plain", orgId);
  };

  const handleDragOver = (e: React.DragEvent, orgId: string) => {
    e.preventDefault();
    e.dataTransfer.dropEffect = "move";
    if (draggingId && orgId !== draggingId) {
      setDragOverId(orgId);
    }
  };

  const handleDragLeave = () => {
    setDragOverId(null);
  };

  const handleDrop = async (e: React.DragEvent, targetOrgId: string) => {
    e.preventDefault();
    setDragOverId(null);
    const sourceOrgId = draggingId;
    setDraggingId(null);

    if (!sourceOrgId || sourceOrgId === targetOrgId) return;

    // Prevent dropping onto own descendant
    const sourceNode = tree.find((n) => n.id === sourceOrgId);
    if (sourceNode && isDescendant(sourceNode, targetOrgId)) {
      setError("Cannot reassign an org to its own descendant");
      return;
    }

    // Optimistically update parent_id in local state
    setOrgs((prev) =>
      prev.map((o) => (o.id === sourceOrgId ? { ...o, parent_id: targetOrgId } : o)),
    );

    try {
      await apiFetch(`/api/v1/orgs/${sourceOrgId}`, {
        method: "PUT",
        body: JSON.stringify({ parent_id: targetOrgId }),
      });
      const sourceOrg = orgMap.get(sourceOrgId);
      const targetOrg = orgMap.get(targetOrgId);
      setMsg(`Moved "${sourceOrg?.name}" under "${targetOrg?.name}"`);
      loadData();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to reassign organization");
      loadData(); // revert
    }
  };

  const handleDragEnd = () => {
    setDraggingId(null);
    setDragOverId(null);
  };

  // ---- Context Menu ----
  const handleContextMenu = (e: React.MouseEvent, orgId: string) => {
    e.preventDefault();
    setContextMenu({ x: e.clientX, y: e.clientY, orgId });
  };

  const contextMenuItems = contextMenu
    ? (() => {
        const node = findNode(tree, contextMenu.orgId);
        if (!node) return null;
        const siblings = getSiblings(tree, node.parent_id, node.id);
        const myIndex = getSiblings(tree, node.parent_id).findIndex((n) => n.id === node.id);
        return (
          <div
            ref={contextMenuRef}
            className="fixed z-50 min-w-[180px] rounded-lg border border-gray-200 bg-white py-1 shadow-xl dark:border-gray-700 dark:bg-gray-800"
            style={{ left: contextMenu.x, top: contextMenu.y }}
          >
            <button
              onClick={() => {
                openCreateChild(contextMenu.orgId);
                setContextMenu(null);
              }}
              className="flex w-full items-center gap-2 px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-700"
            >
              <FolderPlus className="h-4 w-4 text-green-600" /> Add Child Org
            </button>
            <button
              onClick={() => {
                openRename(contextMenu.orgId);
                setContextMenu(null);
              }}
              className="flex w-full items-center gap-2 px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-700"
            >
              <Edit3 className="h-4 w-4 text-blue-600" /> Rename
            </button>
            <button
              onClick={() => {
                handleDelete(contextMenu.orgId);
                setContextMenu(null);
              }}
              className="flex w-full items-center gap-2 px-4 py-2 text-left text-sm text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20"
            >
              <Trash2 className="h-4 w-4" /> Delete
            </button>
            <div className="my-1 border-t border-gray-200 dark:border-gray-700" />
            <button
              disabled={myIndex <= 0}
              onClick={() => {
                handleMove(contextMenu.orgId, "up");
                setContextMenu(null);
              }}
              className="flex w-full items-center gap-2 px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 disabled:cursor-not-allowed disabled:opacity-40 dark:text-gray-300 dark:hover:bg-gray-700"
            >
              <ArrowUp className="h-4 w-4" /> Move Up
            </button>
            <button
              disabled={myIndex >= siblings.length}
              onClick={() => {
                handleMove(contextMenu.orgId, "down");
                setContextMenu(null);
              }}
              className="flex w-full items-center gap-2 px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 disabled:cursor-not-allowed disabled:opacity-40 dark:text-gray-300 dark:hover:bg-gray-700"
            >
              <ArrowDown className="h-4 w-4" /> Move Down
            </button>
          </div>
        );
      })()
    : null;

  // ---- CRUD actions ----
  const openCreateRoot = () => {
    setModalMode("create-root");
    setModalOrgId(null);
    setModalName("");
    setModalDesc("");
  };

  const openCreateChild = (parentId: string) => {
    setModalMode("create-child");
    setModalOrgId(parentId);
    setModalName("");
    setModalDesc("");
  };

  const openRename = (orgId: string) => {
    const org = orgMap.get(orgId);
    setModalMode("rename");
    setModalOrgId(orgId);
    setModalName(org?.name || "");
    setModalDesc(org?.description || "");
  };

  const closeModal = () => {
    setModalMode(null);
    setModalOrgId(null);
    setModalName("");
    setModalDesc("");
  };

  const handleSaveModal = async () => {
    if (!modalName.trim()) {
      setError("Organization name is required");
      return;
    }
    setSaving(true);
    setError(null);
    try {
      if (modalMode === "create-root") {
        await apiFetch("/api/v1/orgs", {
          method: "POST",
          body: JSON.stringify({ name: modalName, description: modalDesc }),
        });
        setMsg(`Created root org "${modalName}"`);
      } else if (modalMode === "create-child" && modalOrgId) {
        await apiFetch("/api/v1/orgs", {
          method: "POST",
          body: JSON.stringify({ name: modalName, description: modalDesc, parent_id: modalOrgId }),
        });
        setMsg(`Created child org "${modalName}"`);
      } else if (modalMode === "rename" && modalOrgId) {
        await apiFetch(`/api/v1/orgs/${modalOrgId}`, {
          method: "PUT",
          body: JSON.stringify({ name: modalName, description: modalDesc }),
        });
        setMsg(`Renamed to "${modalName}"`);
      }
      closeModal();
      loadData();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to save organization");
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (orgId: string) => {
    const org = orgMap.get(orgId);
    if (!org) return;
    if (!confirm(`Delete "${org.name}" and all its sub-organizations?`)) return;
    try {
      await apiFetch(`/api/v1/orgs/${orgId}`, { method: "DELETE" });
      setMsg(`Deleted "${org.name}"`);
      loadData();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete organization");
    }
  };

  const handleMove = async (orgId: string, direction: "up" | "down") => {
    // Reorder within siblings via API (if supported) or local sort
    const node = findNode(tree, orgId);
    if (!node) return;
    const siblings = getSiblings(tree, node.parent_id);
    const myIndex = siblings.findIndex((n) => n.id === orgId);
    if (direction === "up" && myIndex > 0) {
      const target = siblings[myIndex - 1];
      try {
        await apiFetch(`/api/v1/orgs/${orgId}`, {
          method: "PUT",
          body: JSON.stringify({ sort_order: (myIndex - 1) }),
        });
        setMsg(`Moved "${node.name}" up`);
      } catch {
        setMsg(`Moved "${node.name}" up (local only)`);
      }
      loadData();
    } else if (direction === "down" && myIndex < siblings.length - 1) {
      const target = siblings[myIndex + 1];
      try {
        await apiFetch(`/api/v1/orgs/${orgId}`, {
          method: "PUT",
          body: JSON.stringify({ sort_order: (myIndex + 1) }),
        });
        setMsg(`Moved "${node.name}" down`);
      } catch {
        setMsg(`Moved "${node.name}" down (local only)`);
      }
      loadData();
    }
  };

  // ---- Render tree node recursively ----
  const renderNode = (node: OrgNode, depth: number): React.ReactElement => {
    const isExpanded = expanded.has(node.id);
    const isDragOver = dragOverId === node.id;
    const isDragging = draggingId === node.id;
    const isMatched = matchedIds.size > 0 && matchedIds.has(node.id);
    const hasChildren = node.children.length > 0;

    // In search mode, hide non-matching leaf nodes
    if (matchedIds.size > 0 && !isMatched && !hasChildren) {
      // Check if any descendant matches
      const hasMatchedDescendant = (n: OrgNode): boolean => {
        if (matchedIds.has(n.id)) return true;
        return n.children.some(hasMatchedDescendant);
      };
      if (!node.children.some(hasMatchedDescendant)) {
        return <></>;
      }
    }

    return (
      <div key={node.id} className="relative">
        <div
          draggable
          onDragStart={(e) => handleDragStart(e, node.id)}
          onDragOver={(e) => handleDragOver(e, node.id)}
          onDragLeave={handleDragLeave}
          onDrop={(e) => handleDrop(e, node.id)}
          onDragEnd={handleDragEnd}
          onContextMenu={(e) => handleContextMenu(e, node.id)}
          className={`flex items-center gap-3 rounded-xl border p-3 shadow-sm transition-all ${
            isDragOver
              ? "border-blue-400 bg-blue-50 ring-2 ring-blue-300 dark:border-blue-500 dark:bg-blue-900/20"
              : isMatched
                ? "border-yellow-300 bg-yellow-50 dark:border-yellow-600 dark:bg-yellow-900/20"
                : "border-gray-200 bg-white hover:shadow-md dark:border-gray-700 dark:bg-gray-800"
          } ${isDragging ? "opacity-40" : ""} ${depth === 0 ? "ring-1 ring-brand-200" : ""}`}
          style={{ marginLeft: `${depth * 32}px` }}
        >
          {/* Connecting line for children */}
          {depth > 0 && (
            <div
              className="absolute border-l-2 border-gray-200 dark:border-gray-600"
              style={{
                left: `${(depth - 1) * 32 + 16}px`,
                top: 0,
                height: "50%",
              }}
            />
          )}
          {depth > 0 && (
            <div
              className="absolute border-t-2 border-gray-200 dark:border-gray-600"
              style={{
                left: `${(depth - 1) * 32 + 16}px`,
                top: "50%",
                width: "16px",
              }}
            />
          )}

          {/* Expand/collapse toggle */}
          <button
            onClick={() => toggleExpand(node.id)}
            className={`flex h-6 w-6 items-center justify-center rounded text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 ${
              hasChildren ? "" : "invisible"
            }`}
          >
            {isExpanded ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
          </button>

          {/* Org icon */}
          <div
            className={`flex h-9 w-9 items-center justify-center rounded-lg ${
              depth === 0 ? "bg-brand-100" : "bg-gray-100 dark:bg-gray-700"
            }`}
          >
            <Building2 className={`h-5 w-5 ${depth === 0 ? "text-brand-600" : "text-gray-500"}`} />
          </div>

          {/* Name + description */}
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <span
                className={`truncate ${
                  depth === 0 ? "text-base font-bold" : "text-sm font-semibold"
                } dark:text-gray-100`}
              >
                {node.name}
              </span>
              {node.path && (
                <span className="font-mono text-xs text-gray-400">{node.path}</span>
              )}
            </div>
            {node.description && (
              <p className="truncate text-xs text-gray-500">{node.description}</p>
            )}
          </div>

          {/* Member count badge (includes inherited from children) */}
          <span className="flex items-center gap-1 rounded-full bg-blue-50 px-2.5 py-1 text-xs font-medium text-blue-600 dark:bg-blue-900/30 dark:text-blue-400">
            <Users className="h-3 w-3" />
            {node.totalMembers}
          </span>

          {/* Child count */}
          {hasChildren && (
            <span className="text-xs text-gray-400">
              {node.children.length} sub-org{node.children.length !== 1 ? "s" : ""}
            </span>
          )}
        </div>

        {/* Render children */}
        {isExpanded && hasChildren && (
          <div className="mt-1 space-y-1">
            {node.children.map((child) => renderNode(child, depth + 1))}
          </div>
        )}
      </div>
    );
  };

  // ---- Render ----
  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <p className="text-gray-500">Loading organization tree...</p>
      </div>
    );
  }

  return (
    <div>
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold dark:text-gray-100">Organization Tree</h1>
          <p className="mt-1 text-sm text-gray-500">
            Drag cards to reassign parent. Right-click for context menu.
          </p>
        </div>
        <button
          onClick={openCreateRoot}
          className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
        >
          <Plus className="h-4 w-4" /> Add Root Org
        </button>
      </div>

      {/* Messages */}
      {msg && (
        <div className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400">
          {msg}
        </div>
      )}
      {error && (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400">
          {error}
        </div>
      )}

      {/* Search filter */}
      <div className="mb-4">
        <div className="relative max-w-md">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
          <input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search organizations by name..."
            className="w-full rounded-lg border border-gray-300 py-2 pl-9 pr-3 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 focus:border-brand-500 focus:outline-none"
          />
          {search && (
            <button
              onClick={() => setSearch("")}
              className="absolute right-2 top-1/2 -translate-y-1/2 rounded p-1 text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700"
            >
              <X className="h-4 w-4" />
            </button>
          )}
        </div>
      </div>

      {/* Tree */}
      {tree.length === 0 ? (
        <div className="rounded-xl border border-gray-200 bg-white p-12 text-center shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <Building2 className="mx-auto mb-4 h-12 w-12 text-gray-300" />
          <p className="mb-4 text-gray-500">No organizations found</p>
          <button
            onClick={openCreateRoot}
            className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 mx-auto"
          >
            <Plus className="h-4 w-4" /> Create First Org
          </button>
        </div>
      ) : (
        <div className="space-y-1">
          {tree.map((root) => renderNode(root, 0))}
        </div>
      )}

      {/* Context Menu */}
      {contextMenuItems}

      {/* Modal */}
      {modalMode && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div role="dialog" aria-modal="true" className="w-full max-w-md rounded-xl bg-white p-6 shadow-2xl dark:bg-gray-800">
            <div className="mb-4 flex items-center justify-between">
              <h2 className="text-lg font-bold dark:text-gray-100">
                {modalMode === "create-root" && "Add Root Organization"}
                {modalMode === "create-child" && "Add Child Organization"}
                {modalMode === "rename" && "Rename Organization"}
              </h2>
              <button
                onClick={closeModal}
                className="rounded-lg p-1 text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700"
               aria-label="Close">
                <X className="h-5 w-5" />
              </button>
            </div>

            <div className="space-y-4">
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
                  Name
                </label>
                <input
                  value={modalName}
                  onChange={(e) => setModalName(e.target.value)}
                  autoFocus
                  placeholder="Organization name"
                  className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 focus:border-brand-500 focus:outline-none"
                />
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
                  Description (optional)
                </label>
                <input
                  value={modalDesc}
                  onChange={(e) => setModalDesc(e.target.value)}
                  placeholder="Short description"
                  className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 focus:border-brand-500 focus:outline-none"
                />
              </div>
            </div>

            <div className="mt-6 flex justify-end gap-2">
              <button
                onClick={closeModal}
                className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
              >
                Cancel
              </button>
              <button
                onClick={handleSaveModal}
                disabled={saving}
                className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
              >
                {saving ? "Saving..." : "Save"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

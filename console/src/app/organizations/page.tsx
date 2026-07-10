"use client";

import { useEffect, useState, useCallback } from "react";
import Link from "next/link";
import { useApi } from "@/lib/api";
import {
  Building2,
  Plus,
  ChevronRight,
  ChevronDown,
  X,
  Users,
  GitBranch,
  Network,
  Trash2,
  Layers,
} from "lucide-react";

// ===== Types =====

interface Organization {
  id: string;
  name: string;
  path: string;
  parent_id?: string;
}

interface Department {
  id: string;
  org_id: string;
  name: string;
  path: string;
  parent_id?: string;
  manager_id?: string;
}

interface Team {
  id: string;
  org_id: string;
  name: string;
  description: string;
  created_by: string;
}

interface Member {
  id: string;
  user_id: string;
  tenant_id: string;
  org_id: string;
  status: string;
  title: string;
  dept_id?: string;
  team_id?: string;
}

type Tab = "orgs" | "depts" | "teams" | "tree" | "members";

interface TreeData {
  organizations: Organization[];
  departments: Department[];
}

// ===== Main Component =====

export default function OrganizationsPage() {
  const { apiFetch, TENANT_ID } = useApi();
  const [tab, setTab] = useState<Tab>("orgs");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [msg, setMsg] = useState<string | null>(null);

  // Data
  const [orgs, setOrgs] = useState<Organization[]>([]);
  const [depts, setDepts] = useState<Department[]>([]);
  const [teams, setTeams] = useState<Team[]>([]);
  const [memberCounts, setMemberCounts] = useState<Record<string, number>>({});
  const [orgMembers, setOrgMembers] = useState<Member[]>([]);
  const [membersLoading, setMembersLoading] = useState(false);

  // Tree view state
  const [treeData, setTreeData] = useState<TreeData | null>(null);
  const [treeLoading, setTreeLoading] = useState(false);
  const [treeRootId, setTreeRootId] = useState<string | null>(null);

  // UI state
  const [showCreate, setShowCreate] = useState(false);
  const [expandedOrgs, setExpandedOrgs] = useState<Set<string>>(new Set());
  const [selectedOrgId, setSelectedOrgId] = useState<string | null>(null);

  // Lazy-loading tree state: maps orgId → its children (fetched on expand)
  const [treeChildren, setTreeChildren] = useState<Record<string, Organization[]>>({});
  const [treeLoadingIds, setTreeLoadingIds] = useState<Set<string>>(new Set());

  const fetchChildren = useCallback(
    async (orgId: string) => {
      if (treeChildren[orgId]) return; // already loaded
      setTreeLoadingIds((prev) => new Set(prev).add(orgId));
      try {
        const data = await apiFetch<{ organizations?: Organization[] }>(
          `/api/v1/orgs?parent_id=${orgId}`,
        );
        const children = data.organizations || [];
        // Fetch member counts for newly loaded children
        const counts: Record<string, number> = {};
        await Promise.all(
          children.map(async (org) => {
            try {
              const memData = await apiFetch<{ members?: Member[] }>(
                `/api/v1/orgs/${org.id}/members`,
              );
              counts[org.id] = memData.members?.length || 0;
            } catch {
              counts[org.id] = 0;
            }
          }),
        );
        setMemberCounts((prev) => ({ ...prev, ...counts }));
        setTreeChildren((prev) => ({ ...prev, [orgId]: children }));
      } catch {
        setTreeChildren((prev) => ({ ...prev, [orgId]: [] }));
      } finally {
        setTreeLoadingIds((prev) => {
          const next = new Set(prev);
          next.delete(orgId);
          return next;
        });
      }
    },
    [apiFetch, treeChildren],
  );

  const loadOrgs = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiFetch<{ organizations?: Organization[] }>("/api/v1/orgs");
      const list = data.organizations || [];
      setOrgs(list);

      // Fetch member counts for each org in parallel
      const counts: Record<string, number> = {};
      await Promise.all(
        list.map(async (org) => {
          try {
            const memData = await apiFetch<{ members?: Member[] }>(
              `/api/v1/orgs/${org.id}/members`,
            );
            counts[org.id] = memData.members?.length || 0;
          } catch {
            counts[org.id] = 0;
          }
        }),
      );
      setMemberCounts(counts);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load organizations");
      setOrgs([]);
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  const loadDepts = useCallback(
    async (orgId: string) => {
      try {
        const data = await apiFetch<{ departments?: Department[] }>(
          `/api/v1/departments?org_id=${orgId}`,
        );
        setDepts(data.departments || []);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to load departments");
        setDepts([]);
      }
    },
    [apiFetch],
  );

  const loadTeams = useCallback(
    async (orgId: string) => {
      try {
        const data = await apiFetch<{ teams?: Team[] }>(`/api/v1/teams?org_id=${orgId}`);
        setTeams(data.teams || []);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to load teams");
        setTeams([]);
      }
    },
    [apiFetch],
  );

  useEffect(() => {
    loadOrgs();
  }, [loadOrgs]);

  // Auto-expand root orgs on load
  useEffect(() => {
    if (orgs.length > 0 && expandedOrgs.size === 0) {
      const roots = orgs.filter((o) => !o.parent_id);
      setExpandedOrgs(new Set(roots.map((r) => r.id)));
      // Fetch children for each root automatically
      roots.forEach((r) => {
        const childData = treeChildren[r.id];
        if (!childData) {
          fetchChildren(r.id);
        }
      });
    }
  }, [orgs, expandedOrgs.size, treeChildren, fetchChildren]);

  // Load depts/teams when switching tabs or selecting an org
  useEffect(() => {
    if (tab === "depts" && selectedOrgId) {
      loadDepts(selectedOrgId);
    } else if (tab === "teams" && selectedOrgId) {
      loadTeams(selectedOrgId);
    }
  }, [tab, selectedOrgId, loadDepts, loadTeams]);

  // Load tree data for tree tab
  const loadTree = useCallback(
    async (orgId: string) => {
      setTreeLoading(true);
      try {
        const data = await apiFetch<{ organizations?: Organization[]; departments?: Department[] }>(
          `/api/v1/orgs/${orgId}/tree`,
        );
        setTreeData({
          organizations: data.organizations || [],
          departments: data.departments || [],
        });
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to load tree");
        setTreeData(null);
      } finally {
        setTreeLoading(false);
      }
    },
    [apiFetch],
  );

  useEffect(() => {
    if (tab === "tree" && treeRootId) {
      loadTree(treeRootId);
    }
  }, [tab, treeRootId, loadTree]);

  // ===== Handlers =====

  const toggleExpand = (id: string) => {
    setExpandedOrgs((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
        // Lazy-load children when expanding
        if (!treeChildren[id]) {
          fetchChildren(id);
        }
      }
      return next;
    });
  };

  const handleCreateOrg = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const formData = new FormData(e.currentTarget);
    const body: Record<string, string> = {
      tenant_id: TENANT_ID,
      name: formData.get("name") as string,
    };
    const parentId = formData.get("parent_id") as string;
    if (parentId) body.parent_id = parentId;
    try {
      await apiFetch("/api/v1/orgs", { method: "POST", body: JSON.stringify(body) });
      setShowCreate(false);
      setMsg("Organization created");
      loadOrgs();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create organization");
    }
  };

  const handleCreateDept = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const formData = new FormData(e.currentTarget);
    const body: Record<string, string> = {
      org_id: (formData.get("org_id") as string) || selectedOrgId || "",
      name: formData.get("name") as string,
    };
    const parentId = formData.get("parent_id") as string;
    if (parentId) body.parent_id = parentId;
    try {
      await apiFetch("/api/v1/departments", { method: "POST", body: JSON.stringify(body) });
      setShowCreate(false);
      setMsg("Department created");
      if (body.org_id) loadDepts(body.org_id);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create department");
    }
  };

  const handleCreateTeam = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const formData = new FormData(e.currentTarget);
    const body: Record<string, string> = {
      org_id: (formData.get("org_id") as string) || selectedOrgId || "",
      name: formData.get("name") as string,
      description: (formData.get("description") as string) || "",
      created_by: formData.get("created_by") as string,
    };
    try {
      await apiFetch("/api/v1/teams", { method: "POST", body: JSON.stringify(body) });
      setShowCreate(false);
      setMsg("Team created");
      if (body.org_id) loadTeams(body.org_id);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create team");
    }
  };

  const handleDeleteOrg = async (id: string) => {
    if (!confirm("Delete this organization? This cannot be undone.")) return;
    try {
      await apiFetch(`/api/v1/orgs/${id}`, { method: "DELETE" });
      setMsg("Organization deleted");
      loadOrgs();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete");
    }
  };

  const handleDeleteDept = async (id: string) => {
    if (!confirm("Delete this department?")) return;
    try {
      await apiFetch(`/api/v1/departments/${id}`, { method: "DELETE" });
      setMsg("Department deleted");
      if (selectedOrgId) loadDepts(selectedOrgId);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete");
    }
  };

  const handleDeleteTeam = async (id: string) => {
    if (!confirm("Delete this team?")) return;
    try {
      await apiFetch(`/api/v1/teams/${id}`, { method: "DELETE" });
      setMsg("Team deleted");
      if (selectedOrgId) loadTeams(selectedOrgId);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete");
    }
  };

  // ===== Build Tree =====
  // Build tree from root orgs + lazily-loaded children
  const allOrgs = [
    ...orgs,
    ...Object.values(treeChildren).flat(),
  ];
  const uniqueOrgs = Array.from(
    new Map(allOrgs.map((o) => [o.id, o])).values(),
  );
  const tree = buildTree(uniqueOrgs);
  const orgMap = new Map(uniqueOrgs.map((o) => [o.id, o]));

  // Auto-dismiss messages
  useEffect(() => {
    if (msg) {
      const t = setTimeout(() => setMsg(null), 3000);
      return () => clearTimeout(t);
    }
  }, [msg]);

  return (
    <div>
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold dark:text-gray-100">Organizations</h1>
        <button
          onClick={() => {
            setShowCreate(!showCreate);
            setError(null);
          }}
          className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
        >
          <Plus className="h-4 w-4" />
          New {tab === "orgs" ? "Organization" : tab === "depts" ? "Department" : "Team"}
        </button>
      </div>

      {/* Messages */}
      {msg && (
        <div className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700">
          {msg}
        </div>
      )}
      {error && (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">
          {error}
          <p className="mt-1 text-xs">Make sure Org Service (:8071) is running.</p>
        </div>
      )}

      {/* Create Forms */}
      {showCreate && tab === "orgs" && (
        <CreateForm
          title="New Organization"
          onClose={() => setShowCreate(false)}
          onSubmit={handleCreateOrg}
        >
          <FormField label="Name" name="name" placeholder="e.g. Engineering" required />
          <FormField
            label="Parent Organization"
            name="parent_id"
            type="select"
            placeholder="-- None (root) --"
            options={orgs.map((o) => ({ value: o.id, label: o.name }))}
          />
        </CreateForm>
      )}

      {showCreate && tab === "depts" && (
        <CreateForm
          title="New Department"
          onClose={() => setShowCreate(false)}
          onSubmit={handleCreateDept}
        >
          <FormField
            label="Organization"
            name="org_id"
            type="select"
            placeholder="-- Select org --"
            required
            options={orgs.map((o) => ({ value: o.id, label: o.name }))}
            value={selectedOrgId || undefined}
          />
          <FormField label="Name" name="name" placeholder="e.g. Frontend" required />
          <FormField
            label="Parent Department"
            name="parent_id"
            type="select"
            placeholder="-- None (root) --"
            options={depts.map((d) => ({ value: d.id, label: d.name }))}
          />
        </CreateForm>
      )}

      {showCreate && tab === "teams" && (
        <CreateForm
          title="New Team"
          onClose={() => setShowCreate(false)}
          onSubmit={handleCreateTeam}
        >
          <FormField
            label="Organization"
            name="org_id"
            type="select"
            placeholder="-- Select org --"
            required
            options={orgs.map((o) => ({ value: o.id, label: o.name }))}
            value={selectedOrgId || undefined}
          />
          <FormField label="Name" name="name" placeholder="e.g. Platform Team" required />
          <FormField label="Description" name="description" placeholder="Optional" />
          <FormField
            label="Created By (User ID)"
            name="created_by"
            placeholder="UUID of the creator"
            required
          />
        </CreateForm>
      )}

      {/* Org Filter for Depts/Teams tabs */}
      {(tab === "depts" || tab === "teams") && (
        <div className="mb-4 flex items-center gap-3">
          <label className="text-sm font-medium text-gray-600">Filter by Organization:</label>
          <select
            value={selectedOrgId || ""}
            onChange={(e) => setSelectedOrgId(e.target.value || null)}
            className="rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 focus:border-brand-500 focus:outline-none"
          >
            <option value="">-- Select org --</option>
            {orgs.map((o) => (
              <option key={o.id} value={o.id}>
                {o.name}
              </option>
            ))}
          </select>
        </div>
      )}

      {/* Tree root selector */}
      {tab === "tree" && (
        <div className="mb-4 flex items-center gap-3">
          <label className="text-sm font-medium text-gray-600">Root Organization:</label>
          <select
            value={treeRootId || ""}
            onChange={(e) => setTreeRootId(e.target.value || null)}
            className="rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 focus:border-brand-500 focus:outline-none"
          >
            <option value="">-- Select org --</option>
            {orgs.map((o) => (
              <option key={o.id} value={o.id}>
                {o.name}
              </option>
            ))}
          </select>
        </div>
      )}

      {/* Tabs */}
      <div className="mb-4 flex gap-2 border-b border-gray-200">
        <TabButton active={tab === "orgs"} onClick={() => setTab("orgs")} icon={Building2} label={`Organizations (${orgs.length})`} />
        <TabButton active={tab === "depts"} onClick={() => setTab("depts")} icon={Network} label={`Departments (${depts.length})`} />
        <TabButton active={tab === "teams"} onClick={() => setTab("teams")} icon={Users} label={`Teams (${teams.length})`} />
        <TabButton active={tab === "tree"} onClick={() => setTab("tree")} icon={Layers} label="Tree View" />
        <TabButton active={tab === "members"} onClick={() => setTab("members")} icon={Users} label="Members" />
      </div>

      {/* Content */}
      {loading ? (
        <p className="text-gray-500">Loading...</p>
      ) : tab === "orgs" ? (
        /* ===== Organizations Tree View ===== */
        orgs.length === 0 ? (
          <EmptyState icon={Building2} title="No organizations yet" subtitle="Create an organization to start managing your team structure" />
        ) : (
          <div className="space-y-1">
            {tree.map((org) => (
              <OrgTreeNode
                key={org.id}
                org={org}
                depth={0}
                expanded={expandedOrgs}
                onToggle={toggleExpand}
                memberCount={memberCounts[org.id] || 0}
                onDelete={handleDeleteOrg}
                isLoading={treeLoadingIds.has(org.id)}
              />
            ))}
          </div>
        )
      ) : tab === "depts" ? (
        /* ===== Departments ===== */
        !selectedOrgId ? (
          <EmptyState icon={Network} title="Select an organization" subtitle="Choose an organization above to view its departments" />
        ) : depts.length === 0 ? (
          <EmptyState icon={Network} title="No departments" subtitle="Create a department under this organization" />
        ) : (
          <div className="rounded-xl border border-gray-200 bg-white shadow-sm">
            <table className="w-full">
              <thead className="border-b border-gray-200 bg-gray-50">
                <tr>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Department</th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Path</th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {depts.map((d) => (
                  <tr key={d.id} className="hover:bg-gray-50 dark:hover:bg-gray-700">
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2">
                        <Network className="h-4 w-4 text-blue-500" />
                        <span className="text-sm font-medium">{d.name}</span>
                        {d.parent_id && d.parent_id !== d.id && (
                          <span className="text-xs text-gray-400">
                            (under {depts.find((p) => p.id === d.parent_id)?.name || "parent"})
                          </span>
                        )}
                      </div>
                    </td>
                    <td className="px-4 py-3 font-mono text-xs text-gray-500">{d.path || "-"}</td>
                    <td className="px-4 py-3">
                      <button
                        onClick={() => handleDeleteDept(d.id)}
                        className="text-gray-400 hover:text-red-500"
                      >
                        <Trash2 className="h-4 w-4" />
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )
      ) : tab === "teams" ? (
        /* ===== Teams ===== */
        !selectedOrgId ? (
          <EmptyState icon={Users} title="Select an organization" subtitle="Choose an organization above to view its teams" />
        ) : teams.length === 0 ? (
          <EmptyState icon={Users} title="No teams" subtitle="Create a team under this organization" />
        ) : (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {teams.map((t) => (
              <div key={t.id} className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800">
                <div className="mb-3 flex items-start justify-between">
                  <div className="flex items-center gap-3">
                    <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-purple-100">
                      <Users className="h-5 w-5 text-purple-600" />
                    </div>
                    <div>
                      <h3 className="font-semibold">{t.name}</h3>
                      {t.description && (
                        <p className="text-xs text-gray-500">{t.description}</p>
                      )}
                    </div>
                  </div>
                  <button
                    onClick={() => handleDeleteTeam(t.id)}
                    className="text-gray-400 hover:text-red-500"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
                <p className="text-xs text-gray-400">
                  Org: {orgMap.get(t.org_id)?.name || t.org_id.slice(0, 8)}
                </p>
              </div>
            ))}
          </div>
        )
      ) : tab === "tree" ? (
        /* ===== Unified Tree View ===== */
        !treeRootId ? (
          <EmptyState icon={Layers} title="Select a root organization" subtitle="Choose an organization above to view its full hierarchy" />
        ) : treeLoading ? (
          <p className="text-gray-500">Loading tree...</p>
        ) : !treeData || treeData.organizations.length === 0 ? (
          <EmptyState icon={Layers} title="No tree data" subtitle="Failed to load or empty tree" />
        ) : (
          <UnifiedTreeView treeData={treeData} memberCounts={memberCounts} />
        )
      ) : tab === "members" ? (
        /* ===== Members Detail ===== */
        !selectedOrgId ? (
          <EmptyState icon={Users} title="Select an organization" subtitle="Choose an organization to view its members" />
        ) : (
          <MembersDetail
            orgId={selectedOrgId}
            orgName={orgs.find((o) => o.id === selectedOrgId)?.name || ""}
            apiFetch={apiFetch}
          />
        )
      ) : null}
    </div>
  );
}

// ===== Members Detail Component =====

function MembersDetail({
  orgId,
  orgName,
  apiFetch,
}: {
  orgId: string;
  orgName: string;
  apiFetch: <T>(path: string, options?: RequestInit) => Promise<T>;
}) {
  const [members, setMembers] = useState<Member[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const load = async () => {
      setLoading(true);
      try {
        const data = await apiFetch<{ members?: Member[] }>(`/api/v1/orgs/${orgId}/members`);
        setMembers(data.members || []);
      } catch {
        setMembers([]);
      } finally {
        setLoading(false);
      }
    };
    load();
  }, [orgId, apiFetch]);

  const statusColor = (status: string) => {
    switch (status) {
      case "active": return "bg-green-100 text-green-700";
      case "invited": return "bg-blue-100 text-blue-700";
      case "suspended": return "bg-red-100 text-red-700";
      default: return "bg-gray-100 text-gray-600";
    }
  };

  return (
    <div className="rounded-xl border border-gray-200 bg-white shadow-sm">
      <div className="border-b border-gray-200 p-4">
        <h3 className="text-sm font-semibold">
          Members of {orgName} ({members.length})
        </h3>
      </div>
      {loading ? (
        <p className="p-8 text-center text-gray-500">Loading...</p>
      ) : members.length === 0 ? (
        <p className="p-8 text-center text-gray-500">No members in this organization</p>
      ) : (
        <table className="w-full">
          <thead className="border-b border-gray-100 dark:border-gray-700 bg-gray-50">
            <tr>
              <th className="px-4 py-2 text-left text-xs font-medium uppercase text-gray-500">User ID</th>
              <th className="px-4 py-2 text-left text-xs font-medium uppercase text-gray-500">Title</th>
              <th className="px-4 py-2 text-left text-xs font-medium uppercase text-gray-500">Status</th>
              <th className="px-4 py-2 text-left text-xs font-medium uppercase text-gray-500">Department</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {members.map((m) => (
              <tr key={m.id} className="hover:bg-gray-50 dark:hover:bg-gray-700">
                <td className="px-4 py-2">
                  <Link href={`/users/${m.user_id}`} className="font-mono text-xs text-brand-600 hover:underline">
                    {m.user_id.slice(0, 12)}...
                  </Link>
                </td>
                <td className="px-4 py-2 text-sm">{m.title || "-"}</td>
                <td className="px-4 py-2">
                  <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${statusColor(m.status)}`}>
                    {m.status}
                  </span>
                </td>
                <td className="px-4 py-2 text-sm text-gray-500">
                  {m.dept_id ? m.dept_id.slice(0, 8) + "..." : "-"}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}

// ===== Unified Tree View Component =====

function UnifiedTreeView({
  treeData,
  memberCounts,
}: {
  treeData: TreeData;
  memberCounts: Record<string, number>;
}) {
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set());

  // Build org tree from the tree data
  const orgTree = buildTree(treeData.organizations);

  // Group departments by org_id
  const deptsByOrg = new Map<string, Department[]>();
  for (const d of treeData.departments) {
    const list = deptsByOrg.get(d.org_id) || [];
    list.push(d);
    deptsByOrg.set(d.org_id, list);
  }

  const toggle = (id: string) => {
    setExpandedNodes((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  // Auto-expand root nodes
  useEffect(() => {
    if (expandedNodes.size === 0 && orgTree.length > 0) {
      setExpandedNodes(new Set(orgTree.map((o) => o.id)));
    }
  }, [orgTree, expandedNodes.size]);

  return (
    <div className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm">
      <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold">
        <Layers className="h-4 w-4 text-brand-600" />
        Organization Structure
        <span className="text-xs font-normal text-gray-400">
          ({treeData.organizations.length} orgs, {treeData.departments.length} depts)
        </span>
      </h3>
      <div className="space-y-0.5">
        {orgTree.map((org) => (
          <UnifiedOrgNode
            key={org.id}
            org={org}
            depth={0}
            expanded={expandedNodes}
            onToggle={toggle}
            memberCount={memberCounts[org.id] || 0}
            deptsByOrg={deptsByOrg}
          />
        ))}
      </div>
    </div>
  );
}

function UnifiedOrgNode({
  org,
  depth,
  expanded,
  onToggle,
  memberCount,
  deptsByOrg,
}: {
  org: OrgNode;
  depth: number;
  expanded: Set<string>;
  onToggle: (id: string) => void;
  memberCount: number;
  deptsByOrg: Map<string, Department[]>;
}) {
  const isExpanded = expanded.has(org.id);
  const hasChildren = org.children.length > 0;
  const orgDepts = deptsByOrg.get(org.id) || [];
  const hasDepts = orgDepts.length > 0;
  const hasContent = hasChildren || hasDepts;

  return (
    <>
      <div
        className="flex items-center gap-2 rounded-lg px-2 py-1.5 hover:bg-gray-50 relative"
        style={{ paddingLeft: `${depth * 24 + 8}px` }}
      >
        {/* Tree connector lines */}
        {depth > 0 && (
          <span
            className="absolute left-0 top-0 bottom-1/2 w-px bg-gray-200"
            style={{ left: `${(depth - 1) * 24 + 16}px` }}
          />
        )}
        {depth > 0 && (
          <span
            className="absolute h-px bg-gray-200"
            style={{ left: `${(depth - 1) * 24 + 16}px`, width: "12px", top: "50%" }}
          />
        )}
        <button
          onClick={() => hasContent && onToggle(org.id)}
          className={`flex h-4 w-4 items-center justify-center ${hasContent ? "cursor-pointer text-gray-400" : "invisible"}`}
        >
          {hasContent && (isExpanded ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />)}
        </button>
        <Building2 className={`h-4 w-4 ${depth === 0 ? "text-brand-600" : "text-gray-400"}`} />
        <span className={`text-sm ${depth === 0 ? "font-semibold" : "font-medium"}`}>{org.name}</span>
        {org.path && (
          <span className="rounded bg-gray-100 px-1.5 py-0.5 font-mono text-xs text-gray-400">{org.path}</span>
        )}
        <span className="flex items-center gap-1 rounded-full bg-blue-50 px-2 py-0.5 text-xs font-medium text-blue-600">
          <Users className="h-3 w-3" />
          {memberCount}
        </span>
        {hasDepts && (
          <span className="flex items-center gap-1 rounded-full bg-purple-50 px-2 py-0.5 text-xs font-medium text-purple-600">
            <Network className="h-3 w-3" />
            {orgDepts.length} depts
          </span>
        )}
      </div>

      {isExpanded && (
        <>
          {/* Render departments under this org */}
          {orgDepts.map((dept) => (
            <div
              key={dept.id}
              className="flex items-center gap-2 rounded-lg px-2 py-1 hover:bg-blue-50/30"
              style={{ paddingLeft: `${(depth + 1) * 20 + 28}px` }}
            >
              <Network className="h-3.5 w-3.5 text-blue-400" />
              <span className="text-sm text-gray-600 dark:text-gray-400">{dept.name}</span>
              {dept.path && (
                <span className="font-mono text-xs text-gray-300">{dept.path}</span>
              )}
            </div>
          ))}
          {/* Render child organizations */}
          {org.children.map((child) => (
            <UnifiedOrgNode
              key={child.id}
              org={child}
              depth={depth + 1}
              expanded={expanded}
              onToggle={onToggle}
              memberCount={0}
              deptsByOrg={deptsByOrg}
            />
          ))}
        </>
      )}
    </>
  );
}

interface OrgNode extends Organization {
  children: OrgNode[];
}

function buildTree(orgs: Organization[]): OrgNode[] {
  const map = new Map<string, OrgNode>();
  const roots: OrgNode[] = [];

  for (const org of orgs) {
    map.set(org.id, { ...org, children: [] });
  }
  for (const org of orgs) {
    const node = map.get(org.id)!;
    if (org.parent_id && map.has(org.parent_id)) {
      map.get(org.parent_id)!.children.push(node);
    } else {
      roots.push(node);
    }
  }
  return roots;
}

function OrgTreeNode({
  org,
  depth,
  expanded,
  onToggle,
  memberCount,
  onDelete,
  isLoading,
}: {
  org: OrgNode;
  depth: number;
  expanded: Set<string>;
  onToggle: (id: string) => void;
  memberCount: number;
  onDelete: (id: string) => void;
  isLoading?: boolean;
}) {
  const isExpanded = expanded.has(org.id);
  const hasChildren = org.children.length > 0;

  return (
    <>
      <div
        className={`flex items-center gap-2 rounded-lg px-3 py-2.5 hover:bg-gray-50 ${depth === 0 ? "border-b border-gray-100 dark:border-gray-700" : ""}`}
        style={{ paddingLeft: `${depth * 24 + 12}px` }}
      >
        {/* Expand/Collapse toggle */}
        <button
          onClick={() => hasChildren && onToggle(org.id)}
          className={`flex h-5 w-5 items-center justify-center ${hasChildren ? "cursor-pointer text-gray-400" : "invisible"}`}
        >
          {hasChildren && (isExpanded ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />)}
        </button>

        {/* Icon */}
        <Building2 className={`h-4 w-4 ${depth === 0 ? "text-brand-600" : "text-gray-400"}`} />

        {/* Name */}
        <span className={`text-sm ${depth === 0 ? "font-semibold" : "font-medium"}`}>
          {org.name}
        </span>

        {/* Path badge */}
        {org.path && (
          <span className="rounded-full bg-gray-100 px-2 py-0.5 font-mono text-xs text-gray-400">
            {org.path}
          </span>
        )}

        {/* Member count badge */}
        <span className="flex items-center gap-1 rounded-full bg-blue-50 px-2 py-0.5 text-xs font-medium text-blue-600">
          <Users className="h-3 w-3" />
          {memberCount}
        </span>

        {/* Child count */}
        {hasChildren && (
          <span className="flex items-center gap-1 text-xs text-gray-400">
            <GitBranch className="h-3 w-3" />
            {org.children.length}
          </span>
        )}

        {/* Delete */}
        <button
          onClick={() => onDelete(org.id)}
          className="ml-auto text-gray-300 hover:text-red-500"
        >
          <Trash2 className="h-4 w-4" />
        </button>

        {/* View Details button */}
        <Link
          href={`/organizations/${org.id}`}
          className="flex items-center gap-1 rounded-lg border border-gray-300 px-2 py-1 text-xs font-medium text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
        >
          View Details
        </Link>
      </div>

      {/* Render children */}
      {isExpanded && isLoading && org.children.length === 0 && (
        <div style={{ paddingLeft: `${(depth + 1) * 24 + 12}px` }} className="py-2 text-xs text-gray-400">
          Loading children...
        </div>
      )}
      {isExpanded &&
        org.children.map((child) => (
          <OrgTreeNode
            key={child.id}
            org={child}
            depth={depth + 1}
            expanded={expanded}
            onToggle={onToggle}
            memberCount={0}
            onDelete={onDelete}
            isLoading={false}
          />
        ))}
    </>
  );
}

// ===== Reusable UI Components =====

function CreateForm({
  title,
  onClose,
  onSubmit,
  children,
}: {
  title: string;
  onClose: () => void;
  onSubmit: (e: React.FormEvent<HTMLFormElement>) => void;
  children: React.ReactNode;
}) {
  return (
    <form
      onSubmit={onSubmit}
      className="mb-6 rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800"
    >
      <div className="mb-4 flex items-center justify-between">
        <h3 className="text-sm font-semibold">{title}</h3>
        <button type="button" onClick={onClose}>
          <X className="h-4 w-4 text-gray-400" />
        </button>
      </div>
      <div className="grid gap-3 sm:grid-cols-2">{children}</div>
      <button
        type="submit"
        className="mt-4 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
      >
        Create
      </button>
    </form>
  );
}

function FormField({
  label,
  name,
  placeholder,
  required,
  type = "text",
  options,
  value,
}: {
  label: string;
  name: string;
  placeholder?: string;
  required?: boolean;
  type?: "text" | "select";
  options?: { value: string; label: string }[];
  value?: string;
}) {
  return (
    <div>
      <label className="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">
        {label}
        {required && <span className="text-red-500"> *</span>}
      </label>
      {type === "select" ? (
        <select
          name={name}
          defaultValue={value || ""}
          className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 focus:border-brand-500 focus:outline-none"
        >
          <option value="">{placeholder || "-- Select --"}</option>
          {options?.map((opt) => (
            <option key={opt.value} value={opt.value}>
              {opt.label}
            </option>
          ))}
        </select>
      ) : (
        <input
          name={name}
          required={required}
          placeholder={placeholder}
          className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 focus:border-brand-500 focus:outline-none"
        />
      )}
    </div>
  );
}

function TabButton({
  active,
  onClick,
  icon: Icon,
  label,
}: {
  active: boolean;
  onClick: () => void;
  icon: React.ElementType;
  label: string;
}) {
  return (
    <button
      onClick={onClick}
      className={`flex items-center gap-1.5 px-4 py-2 text-sm font-medium ${
        active
          ? "border-b-2 border-brand-600 text-brand-600"
          : "text-gray-500 hover:text-gray-700"
      }`}
    >
      <Icon className="h-4 w-4" />
      {label}
    </button>
  );
}

function EmptyState({
  icon: Icon,
  title,
  subtitle,
}: {
  icon: React.ElementType;
  title: string;
  subtitle: string;
}) {
  return (
    <div className="rounded-xl border border-gray-200 bg-white p-12 text-center shadow-sm">
      <Icon className="mx-auto mb-4 h-12 w-12 text-gray-300" />
      <p className="text-gray-500">{title}</p>
      <p className="mt-1 text-xs text-gray-400">{subtitle}</p>
    </div>
  );
}

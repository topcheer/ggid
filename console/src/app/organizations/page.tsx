"use client";

import { useEffect, useState, useCallback } from "react";
import { useApi } from "@/lib/api";
import { Building2, Plus, ChevronRight } from "lucide-react";

interface Organization {
  id: string;
  name: string;
  path: string;
  parent_id?: string;
  metadata?: Record<string, unknown>;
}

export default function OrganizationsPage() {
  const { apiFetch } = useApi();
  const [orgs, setOrgs] = useState<Organization[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);

  const loadOrgs = useCallback(async () => {
    setLoading(true);
    try {
      const data = await apiFetch<{ organizations?: Organization[]; items?: Organization[] }>(
        "/api/v1/orgs",
      );
      setOrgs(data.organizations || data.items || []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load organizations");
      setOrgs([]);
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => {
    loadOrgs();
  }, [loadOrgs]);

  const handleCreate = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const formData = new FormData(e.currentTarget);
    try {
      await apiFetch("/api/v1/orgs", {
        method: "POST",
        body: JSON.stringify({ name: formData.get("name") }),
      });
      setShowCreate(false);
      loadOrgs();
    } catch (err) {
      alert(err instanceof Error ? err.message : "Failed to create organization");
    }
  };

  // Build tree structure from flat list
  const tree = buildTree(orgs);

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold">Organizations</h1>
        <button
          onClick={() => setShowCreate(!showCreate)}
          className="flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
        >
          <Plus className="h-4 w-4" />
          New Organization
        </button>
      </div>

      {showCreate && (
        <form
          onSubmit={handleCreate}
          className="mb-6 rounded-xl border border-gray-200 bg-white p-6 shadow-sm"
        >
          <div className="flex gap-4">
            <input
              name="name"
              required
              placeholder="Organization name"
              className="w-full max-w-sm rounded-lg border border-gray-300 px-3 py-2"
            />
            <button
              type="submit"
              className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
            >
              Create
            </button>
            <button
              type="button"
              onClick={() => setShowCreate(false)}
              className="rounded-lg border border-gray-300 px-4 py-2 text-sm hover:bg-gray-50"
            >
              Cancel
            </button>
          </div>
        </form>
      )}

      {error && (
        <div className="mb-4 rounded-lg border border-orange-200 bg-orange-50 p-4 text-sm text-orange-700">
          {error}
          <p className="mt-1 text-xs">Make sure Org Service (:8071) is running.</p>
        </div>
      )}

      {loading ? (
        <p className="text-gray-500">Loading organizations...</p>
      ) : orgs.length === 0 ? (
        <div className="rounded-xl border border-gray-200 bg-white p-12 text-center shadow-sm">
          <Building2 className="mx-auto mb-4 h-12 w-12 text-gray-300" />
          <p className="text-gray-500">No organizations yet</p>
          <p className="mt-1 text-xs text-gray-400">
            Create an organization to start managing your team structure
          </p>
        </div>
      ) : (
        <div className="rounded-xl border border-gray-200 bg-white shadow-sm">
          <table className="w-full">
            <thead className="border-b border-gray-200 bg-gray-50">
              <tr>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">
                  Name
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">
                  Path
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">
                  ID
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {tree.map((org) => renderOrgRow(org, 0))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

interface OrgNode extends Organization {
  children: OrgNode[];
}

function buildTree(orgs: Organization[]): OrgNode[] {
  const map = new Map<string, OrgNode>();
  const roots: OrgNode[] = [];

  // Create nodes
  for (const org of orgs) {
    map.set(org.id, { ...org, children: [] });
  }

  // Build tree
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

function renderOrgRow(org: OrgNode, depth: number): React.JSX.Element {
  return (
    <>
      <tr key={org.id} className="hover:bg-gray-50">
        <td className="px-4 py-3">
          <div className="flex items-center gap-2" style={{ paddingLeft: `${depth * 20}px` }}>
            {org.children.length > 0 && <ChevronRight className="h-4 w-4 text-gray-400" />}
            <Building2 className="h-4 w-4 text-gray-400" />
            <span className="text-sm font-medium">{org.name}</span>
          </div>
        </td>
        <td className="px-4 py-3 font-mono text-xs text-gray-500">{org.path}</td>
        <td className="px-4 py-3 font-mono text-xs text-gray-400">{org.id.slice(0, 8)}</td>
      </tr>
      {org.children.map((child) => renderOrgRow(child, depth + 1))}
    </>
  );
}

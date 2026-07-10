"use client";

import { useState, useCallback } from "react";
import { useUsers, useApi, type User } from "@/lib/api";
import Link from "next/link";
import { Search, Plus, Lock, Unlock, Trash2, UserPlus, ChevronLeft, ChevronRight, Shield, Download, Upload, X } from "lucide-react";

const PAGE_SIZE = 10;

export default function UsersPage() {
  const { users, loading, error, refresh } = useUsers();
  const { apiFetch } = useApi();
  const [search, setSearch] = useState("");
  const [showCreate, setShowCreate] = useState(false);
  const [page, setPage] = useState(0);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [batchRole, setBatchRole] = useState("");
  const [msg, setMsg] = useState<string | null>(null);
  const [roles, setRoles] = useState<{ id: string; key: string; name: string }[]>([]);
  const [showImport, setShowImport] = useState(false);
  const [importText, setImportText] = useState("");
  const [importResult, setImportResult] = useState<string | null>(null);

  // Load roles for batch assign
  useCallback(async () => {
    const data = await apiFetch<{ roles?: { id: string; key: string; name: string }[] }>("/api/v1/roles").catch(() => ({ roles: [] }));
    setRoles(data.roles || []);
  }, [apiFetch]);

  const filtered = users.filter(
    (u) =>
      u.username.toLowerCase().includes(search.toLowerCase()) ||
      u.email.toLowerCase().includes(search.toLowerCase()),
  );

  const totalPages = Math.ceil(filtered.length / PAGE_SIZE);
  const paginated = filtered.slice(page * PAGE_SIZE, (page + 1) * PAGE_SIZE);

  const toggleSelect = (id: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const toggleSelectAll = () => {
    if (selected.size === paginated.length) {
      setSelected(new Set());
    } else {
      setSelected(new Set(paginated.map((u) => u.id)));
    }
  };

  const handleCreate = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const formData = new FormData(e.currentTarget);
    try {
      await apiFetch("/api/v1/users", {
        method: "POST",
        body: JSON.stringify({
          username: formData.get("username"),
          email: formData.get("email"),
          password: formData.get("password"),
        }),
      });
      setShowCreate(false);
      refresh();
    } catch (err) {
      alert(err instanceof Error ? err.message : "Failed to create user");
    }
  };

  const handleLock = async (userId: string, currentStatus: string) => {
    const action = currentStatus === "active" ? "lock" : "unlock";
    try {
      await apiFetch(`/api/v1/users/${userId}/${action}`, { method: "POST" });
      refresh();
    } catch (err) {
      alert(err instanceof Error ? err.message : `Failed to ${action} user`);
    }
  };

  const handleDelete = async (userId: string, username: string) => {
    if (!confirm(`Delete user "${username}"?`)) return;
    try {
      await apiFetch(`/api/v1/users/${userId}`, { method: "DELETE" });
      refresh();
    } catch (err) {
      alert(err instanceof Error ? err.message : "Failed");
    }
  };

  const handleBatchDelete = async () => {
    if (selected.size === 0) return;
    if (!confirm(`Delete ${selected.size} selected users?`)) return;
    try {
      await Promise.all([...selected].map((id) => apiFetch(`/api/v1/users/${id}`, { method: "DELETE" })));
      setSelected(new Set());
      setMsg(`Deleted ${selected.size} users`);
      refresh();
    } catch (err) {
      alert(err instanceof Error ? err.message : "Batch delete failed");
    }
  };

  const handleBatchAssignRole = async () => {
    if (selected.size === 0 || !batchRole) return;
    try {
      await Promise.all(
        [...selected].map((id) =>
          apiFetch(`/api/v1/users/${id}/roles`, { method: "POST", body: JSON.stringify({ role_id: batchRole }) }),
        ),
      );
      setMsg(`Role assigned to ${selected.size} users`);
      setSelected(new Set());
      setBatchRole("");
    } catch (err) {
      alert(err instanceof Error ? err.message : "Batch assign failed");
    }
  };

  const handleExportCSV = () => {
    const header = "username,email,status,created_at\n";
    const rows = filtered.map((u) => `${u.username},${u.email},${u.status || "active"},${u.created_at || ""}`).join("\n");
    const blob = new Blob([header + rows], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "users_export.csv";
    a.click();
    URL.revokeObjectURL(url);
  };

  const handleImportCSV = async () => {
    const lines = importText.trim().split("\n").filter((l) => l.trim() && !l.startsWith("username,"));
    let created = 0;
    const errors: string[] = [];
    for (let i = 0; i < lines.length; i++) {
      const [username, email, password] = lines[i].split(",").map((s) => s.trim());
      if (!username || !email) { errors.push(`Row ${i + 1}: missing username or email`); continue; }
      try {
        await apiFetch("/api/v1/users", {
          method: "POST",
          body: JSON.stringify({ username, email, password: password || "TempPass123!" }),
        });
        created++;
      } catch (err) {
        errors.push(`Row ${i + 1}: ${err instanceof Error ? err.message : "failed"}`);
      }
    }
    setImportResult(`Created ${created} users${errors.length ? `, ${errors.length} errors: ${errors.join("; ")}` : ""}`);
    setImportText("");
    refresh();
  };

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold">Users</h1>
        <div className="flex gap-2">
          <button onClick={handleExportCSV} className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-2 text-sm hover:bg-gray-50">
            <Download className="h-4 w-4" /> Export
          </button>
          <button onClick={() => setShowImport(!showImport)} className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-2 text-sm hover:bg-gray-50">
            <Upload className="h-4 w-4" /> Import
          </button>
          <button
            onClick={() => setShowCreate(!showCreate)}
            className="flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
          >
            <UserPlus className="h-4 w-4" /> New User
          </button>
        </div>
      </div>

      {msg && (
        <div className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700">{msg}</div>
      )}

      {showImport && (
        <div className="mb-6 rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-lg font-semibold">Import Users (CSV)</h2>
            <button onClick={() => setShowImport(false)}><X className="h-4 w-4 text-gray-400" /></button>
          </div>
          <p className="mb-2 text-xs text-gray-500">Format: username,email,password (one per line, password optional)</p>
          <textarea
            value={importText}
            onChange={(e) => setImportText(e.target.value)}
            rows={6}
            placeholder={"alice,alice@example.com,Pass123!\nbob,bob@example.com,Pass123!"}
            className="w-full rounded-lg border border-gray-300 px-3 py-2 font-mono text-sm"
          />
          <button onClick={handleImportCSV} disabled={!importText.trim()} className="mt-3 rounded-lg bg-brand-600 px-4 py-2 text-sm text-white hover:bg-brand-700 disabled:opacity-50">
            Import Users
          </button>
          {importResult && <p className="mt-3 text-sm text-gray-600">{importResult}</p>}
        </div>
      )}

      {showCreate && (
        <form onSubmit={handleCreate} className="mb-6 rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <h2 className="mb-4 text-lg font-semibold">Create New User</h2>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="mb-1 block text-sm font-medium">Username</label>
              <input name="username" required className="w-full rounded-lg border border-gray-300 px-3 py-2" placeholder="johndoe" />
            </div>
            <div>
              <label className="mb-1 block text-sm font-medium">Email</label>
              <input name="email" type="email" required className="w-full rounded-lg border border-gray-300 px-3 py-2" placeholder="john@example.com" />
            </div>
            <div className="col-span-2">
              <label className="mb-1 block text-sm font-medium">Password</label>
              <input name="password" type="password" required minLength={12} className="w-full rounded-lg border border-gray-300 px-3 py-2" placeholder="At least 12 characters" />
            </div>
          </div>
          <div className="mt-4 flex gap-2">
            <button type="submit" className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700">Create</button>
            <button type="button" onClick={() => setShowCreate(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium hover:bg-gray-50">Cancel</button>
          </div>
        </form>
      )}

      {error && <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">{error}</div>}

      {/* Search + Batch toolbar */}
      <div className="mb-4 flex flex-wrap items-center gap-2">
        <div className="flex items-center gap-2">
          <Search className="h-4 w-4 text-gray-400" />
          <input
            type="text"
            placeholder="Search users..."
            value={search}
            onChange={(e) => { setSearch(e.target.value); setPage(0); }}
            className="w-full max-w-xs rounded-lg border border-gray-300 px-3 py-2"
          />
        </div>
        {selected.size > 0 && (
          <div className="flex items-center gap-2 rounded-lg border border-amber-300 bg-amber-50 px-3 py-1.5">
            <span className="text-sm font-medium text-amber-800">{selected.size} selected</span>
            <select
              value={batchRole}
              onChange={(e) => setBatchRole(e.target.value)}
              className="rounded border border-gray-300 px-2 py-1 text-xs"
            >
              <option value="">Assign role...</option>
              {roles.map((r) => (
                <option key={r.id} value={r.id}>{r.name || r.key}</option>
              ))}
            </select>
            <button onClick={handleBatchAssignRole} disabled={!batchRole} className="flex items-center gap-1 rounded bg-brand-600 px-2 py-1 text-xs text-white disabled:opacity-50">
              <Shield className="h-3 w-3" /> Assign
            </button>
            <button onClick={handleBatchDelete} className="flex items-center gap-1 rounded bg-red-600 px-2 py-1 text-xs text-white">
              <Trash2 className="h-3 w-3" /> Delete
            </button>
          </div>
        )}
      </div>

      {/* Table */}
      <div className="overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm">
        <table className="w-full">
          <thead className="border-b border-gray-200 bg-gray-50">
            <tr>
              <th className="px-4 py-3 text-left">
                <input
                  type="checkbox"
                  checked={selected.size === paginated.length && paginated.length > 0}
                  onChange={toggleSelectAll}
                  className="rounded"
                />
              </th>
              <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">User</th>
              <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Status</th>
              <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Created</th>
              <th className="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {loading ? (
              <tr><td colSpan={5} className="px-4 py-8 text-center text-gray-500">Loading...</td></tr>
            ) : paginated.length === 0 ? (
              <tr><td colSpan={5} className="px-4 py-8 text-center text-gray-500">No users found</td></tr>
            ) : (
              paginated.map((user) => (
                <tr key={user.id} className={`hover:bg-gray-50 ${selected.has(user.id) ? "bg-blue-50/40" : ""}`}>
                  <td className="px-4 py-3">
                    <input
                      type="checkbox"
                      checked={selected.has(user.id)}
                      onChange={() => toggleSelect(user.id)}
                      className="rounded"
                    />
                  </td>
                  <td className="px-4 py-3">
                    <Link href={`/users/${user.id}`} className="flex items-center gap-3">
                      <div className="flex h-8 w-8 items-center justify-center rounded-full bg-gray-200 text-sm font-medium uppercase">
                        {user.username[0]}
                      </div>
                      <div>
                        <p className="text-sm font-medium hover:text-brand-600">{user.username}</p>
                        <p className="text-xs text-gray-500">{user.email}</p>
                      </div>
                    </Link>
                  </td>
                  <td className="px-4 py-3">
                    <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                      user.status === "active" ? "bg-green-100 text-green-700" : user.status === "locked" ? "bg-red-100 text-red-700" : "bg-gray-100 text-gray-600"
                    }`}>
                      {user.status}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-500">
                    {user.created_at ? new Date(user.created_at).toLocaleDateString() : "-"}
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex justify-end gap-1">
                      {user.status === "active" ? (
                        <button onClick={() => handleLock(user.id, user.status)} title="Lock" className="rounded p-1.5 text-gray-400 hover:bg-gray-100">
                          <Lock className="h-4 w-4" />
                        </button>
                      ) : (
                        <button onClick={() => handleLock(user.id, user.status)} title="Unlock" className="rounded p-1.5 text-gray-400 hover:bg-gray-100">
                          <Unlock className="h-4 w-4" />
                        </button>
                      )}
                      <button onClick={() => handleDelete(user.id, user.username)} title="Delete" className="rounded p-1.5 text-gray-400 hover:bg-red-50 hover:text-red-600">
                        <Trash2 className="h-4 w-4" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="mt-4 flex items-center justify-between">
          <p className="text-sm text-gray-500">
            Showing {page * PAGE_SIZE + 1}–{Math.min((page + 1) * PAGE_SIZE, filtered.length)} of {filtered.length}
          </p>
          <div className="flex gap-2">
            <button
              onClick={() => setPage(Math.max(0, page - 1))}
              disabled={page === 0}
              className="flex items-center gap-1 rounded-lg border border-gray-300 px-3 py-1.5 text-sm disabled:opacity-50"
            >
              <ChevronLeft className="h-4 w-4" /> Prev
            </button>
            <span className="flex items-center px-3 text-sm text-gray-500">
              {page + 1} / {totalPages}
            </span>
            <button
              onClick={() => setPage(Math.min(totalPages - 1, page + 1))}
              disabled={page >= totalPages - 1}
              className="flex items-center gap-1 rounded-lg border border-gray-300 px-3 py-1.5 text-sm disabled:opacity-50"
            >
              Next <ChevronRight className="h-4 w-4" />
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

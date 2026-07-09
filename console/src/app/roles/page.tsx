"use client";

import { Shield } from "lucide-react";

export default function RolesPage() {
  const systemRoles = [
    { key: "admin", name: "Administrator", description: "Full system access", permissions: 9 },
    { key: "editor", name: "Editor", description: "Read and write access, no admin", permissions: 8 },
    { key: "viewer", name: "Viewer", description: "Read-only access", permissions: 4 },
  ];

  return (
    <div>
      <h1 className="mb-6 text-2xl font-bold">Roles & Permissions</h1>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {systemRoles.map((role) => (
          <div
            key={role.key}
            className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm"
          >
            <div className="mb-3 flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-brand-100">
                <Shield className="h-5 w-5 text-brand-600" />
              </div>
              <div>
                <h3 className="font-semibold">{role.name}</h3>
                <p className="text-xs text-gray-500">{role.key}</p>
              </div>
            </div>
            <p className="mb-3 text-sm text-gray-600">{role.description}</p>
            <div className="flex items-center justify-between border-t border-gray-100 pt-3">
              <span className="text-xs text-gray-500">{role.permissions} permissions</span>
              <span className="rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-600">
                System
              </span>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

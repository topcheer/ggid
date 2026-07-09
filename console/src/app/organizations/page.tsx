"use client";

import { Building2 } from "lucide-react";

export default function OrganizationsPage() {
  return (
    <div>
      <h1 className="mb-6 text-2xl font-bold">Organizations</h1>
      <div className="rounded-xl border border-gray-200 bg-white p-12 text-center shadow-sm">
        <Building2 className="mx-auto mb-4 h-12 w-12 text-gray-300" />
        <p className="text-gray-500">No organizations yet</p>
        <button className="mt-4 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700">
          Create Organization
        </button>
      </div>
    </div>
  );
}

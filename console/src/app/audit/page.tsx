"use client";

import { ScrollText } from "lucide-react";

export default function AuditPage() {
  return (
    <div>
      <h1 className="mb-6 text-2xl font-bold">Audit Log</h1>
      <div className="rounded-xl border border-gray-200 bg-white p-12 text-center shadow-sm">
        <ScrollText className="mx-auto mb-4 h-12 w-12 text-gray-300" />
        <p className="text-gray-500">Audit log will appear here</p>
        <p className="mt-2 text-xs text-gray-400">Connect to Audit Service to view events</p>
      </div>
    </div>
  );
}

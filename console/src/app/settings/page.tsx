"use client";

import { Settings } from "lucide-react";

export default function SettingsPage() {
  return (
    <div>
      <h1 className="mb-6 text-2xl font-bold">Settings</h1>
      <div className="rounded-xl border border-gray-200 bg-white p-12 text-center shadow-sm">
        <Settings className="mx-auto mb-4 h-12 w-12 text-gray-300" />
        <p className="text-gray-500">Tenant settings will appear here</p>
      </div>
    </div>
  );
}

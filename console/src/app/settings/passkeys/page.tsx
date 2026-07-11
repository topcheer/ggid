"use client";

import { useState, useEffect, useCallback } from "react";
import { Key, Trash2, Smartphone, Monitor, Fingerprint, Check, X } from "lucide-react";

interface Passkey {
  id: string;
  label: string;
  device: string;
  platform: string;
  created_at: string;
  last_used: string | null;
  synced: boolean;
}

const platformIcons: Record<string, typeof Smartphone> = {
  ios: Smartphone,
  android: Smartphone,
  macos: Monitor,
  windows: Monitor,
  linux: Monitor,
  chrome: Monitor,
};

export default function PasskeysPage() {
  const [passkeys, setPasskeys] = useState<Passkey[]>([]);
  const [loading, setLoading] = useState(false);
  const [revokingId, setRevokingId] = useState<string | null>(null);

  const fetchPasskeys = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/auth/passkeys", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const data = await res.json();
        setPasskeys(data.passkeys || data || []);
      }
    } catch {
      /* noop */
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchPasskeys();
  }, [fetchPasskeys]);

  const revoke = async (id: string) => {
    setRevokingId(id);
    try {
      await fetch(`/api/v1/auth/passkeys/${id}`, {
        method: "DELETE",
        headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
      });
      setPasskeys((prev) => prev.filter((p) => p.id !== id));
    } catch {
      /* noop */
    } finally {
      setRevokingId(null);
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Key className="w-6 h-6 text-blue-500" /> Passkey Management</h1>
        <p className="text-sm text-gray-500 mt-1">Manage registered passkeys and WebAuthn credentials.</p>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="rounded-lg border p-4 dark:border-gray-800">
          <span className="text-sm text-gray-500">Total Passkeys</span>
          <p className="text-2xl font-bold mt-1">{passkeys.length}</p>
        </div>
        <div className="rounded-lg border p-4 dark:border-gray-800">
          <span className="text-sm text-gray-500">Synced (iCloud/Google)</span>
          <p className="text-2xl font-bold mt-1 text-blue-600">{passkeys.filter((p) => p.synced).length}</p>
        </div>
        <div className="rounded-lg border p-4 dark:border-gray-800">
          <span className="text-sm text-gray-500">Never Used</span>
          <p className="text-2xl font-bold mt-1 text-yellow-600">{passkeys.filter((p) => !p.last_used).length}</p>
        </div>
      </div>

      {/* Passkey list */}
      <div className="rounded-lg border dark:border-gray-800">
        <div className="px-4 py-3 border-b dark:border-gray-800">
          <h3 className="font-semibold">Registered Passkeys</h3>
        </div>
        <div className="divide-y dark:divide-gray-800">
          {passkeys.map((pk) => {
            const Icon = platformIcons[pk.platform?.toLowerCase()] || Fingerprint;
            return (
              <div key={pk.id} className="px-4 py-3 flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-lg bg-blue-50 dark:bg-blue-900/20 flex items-center justify-center">
                    <Icon className="w-5 h-5 text-blue-500" />
                  </div>
                  <div>
                    <div className="flex items-center gap-2">
                      <span className="font-medium">{pk.label}</span>
                      {pk.synced ? (
                        <span className="flex items-center gap-0.5 px-1.5 py-0.5 rounded text-xs bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"><Check className="w-3 h-3" /> Synced</span>
                      ) : (
                        <span className="flex items-center gap-0.5 px-1.5 py-0.5 rounded text-xs bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400"><X className="w-3 h-3" /> Device-only</span>
                      )}
                    </div>
                    <p className="text-xs text-gray-500 mt-0.5">{pk.device} &middot; {pk.platform} &middot; Created {pk.created_at}</p>
                    <p className="text-xs text-gray-400">Last used: {pk.last_used || "Never"}</p>
                  </div>
                </div>
                <button
                  onClick={() => revoke(pk.id)}
                  disabled={revokingId === pk.id}
                  className="px-3 py-1.5 rounded-lg text-sm font-medium text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20 border border-red-200 dark:border-red-900 disabled:opacity-50 flex items-center gap-1"
                >
                  <Trash2 className="w-4 h-4" />
                  {revokingId === pk.id ? "Revoking..." : "Revoke"}
                </button>
              </div>
            );
          })}
          {passkeys.length === 0 && !loading && (
            <p className="px-4 py-8 text-center text-gray-500">No passkeys registered.</p>
          )}
          {loading && <p className="px-4 py-8 text-center text-gray-500">Loading...</p>}
        </div>
      </div>
    </div>
  );
}

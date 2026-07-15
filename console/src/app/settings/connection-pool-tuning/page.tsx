'use client';

import { useState } from "react";
import { useTranslations } from "@/lib/i18n";

interface PoolConfig {
  service: string;
  minIdle: number;
  maxOpen: number;
  maxLifetime: number;
  idleTimeout: number;
}

export default function ConnectionPoolTuningPage() {
  const [pools, setPools] = useState<PoolConfig[]>([
    { service: "identity-db", minIdle: 5, maxOpen: 50, maxLifetime: 60, idleTimeout: 10 },
    { service: "audit-db", minIdle: 2, maxOpen: 20, maxLifetime: 60, idleTimeout: 10 },
  ]);

  const t = useTranslations();

  const addService = () => {
    setPools([
      ...pools,
      { service: "new-service", minIdle: 5, maxOpen: 50, maxLifetime: 60, idleTimeout: 10 },
    ]);
  };

  const deleteService = (index: number) => {
    setPools(pools.filter((_, i) => i !== index));
  };

  const updatePool = (index: number, patch: Partial<PoolConfig>) => {
    setPools(pools.map((p, i) => (i === index ? { ...p, ...patch } : p)));
  };

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      <h1 className="text-2xl font-bold">{t("backend2.connPoolTuning.title")}</h1>
      <p className="text-gray-600">Tune min/max connections, lifetime, and idle timeout per database or service pool.</p>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">Per-Service Pool Settings</h2>
          <button onClick={addService} className="px-3 py-1 bg-blue-600 text-white rounded text-sm">{"Add Service"}</button>
        </div>
        <div className="space-y-4">
          {pools.map((pool, index) => (
            <div key={index} className="border rounded p-4 grid grid-cols-5 gap-3 items-end">
              <div className="space-y-1">
                <label className="text-xs text-gray-500">{t("backend2.connPoolTuning.service")}</label>
                <input
                  type="text"
                  value={pool.service}
                  onChange={(e) => updatePool(index, { service: e.target.value })}
                  className="w-full border rounded px-2 py-1 text-sm font-mono"
                />
              </div>
              <div className="space-y-1">
                <label className="text-xs text-gray-500">{"Min Idle"}</label>
                <input
                  type="number"
                  min={0}
                  value={pool.minIdle}
                  onChange={(e) => updatePool(index, { minIdle: parseInt(e.target.value, 10) || 0 })}
                  className="w-full border rounded px-2 py-1 text-sm"
                />
              </div>
              <div className="space-y-1">
                <label className="text-xs text-gray-500">{"Max Open"}</label>
                <input
                  type="number"
                  min={1}
                  value={pool.maxOpen}
                  onChange={(e) => updatePool(index, { maxOpen: parseInt(e.target.value, 10) || 1 })}
                  className="w-full border rounded px-2 py-1 text-sm"
                />
              </div>
              <div className="space-y-1">
                <label className="text-xs text-gray-500">{"Max Lifetime"}</label>
                <input
                  type="number"
                  min={1}
                  value={pool.maxLifetime}
                  onChange={(e) => updatePool(index, { maxLifetime: parseInt(e.target.value, 10) || 1 })}
                  className="w-full border rounded px-2 py-1 text-sm"
                />
              </div>
              <div className="space-y-1">
                <label className="text-xs text-gray-500">{"Idle Timeout"}</label>
                <input
                  type="number"
                  min={1}
                  value={pool.idleTimeout}
                  onChange={(e) => updatePool(index, { idleTimeout: parseInt(e.target.value, 10) || 1 })}
                  className="w-full border rounded px-2 py-1 text-sm"
                />
              </div>
              <button
                onClick={() => deleteService(index)}
                className="text-sm text-red-600 hover:text-red-700"
              >
                {"Delete"}
              </button>
            </div>
          ))}
        </div>
      </section>

      <div className="flex justify-end">
        <button className="px-4 py-2 bg-blue-600 text-white rounded text-sm">
          {"Save"}
        </button>
      </div>
    </div>
  );
}

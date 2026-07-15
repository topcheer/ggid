'use client';

import { useState } from "react";
import { useTranslations } from "@/lib/i18n";

interface HeaderMatch { key: string; value: string; }

export default function CanaryDeploymentConfigPage() {
  const [service, setService] = useState("identity-service");
  const [canaryWeight, setCanaryWeight] = useState(10);
  const stableWeight = 100 - canaryWeight;
  const [headerMatches, setHeaderMatches] = useState<HeaderMatch[]>([
    { key: "x-canary", value: "true" },
  ]);
  const [newKey, setNewKey] = useState("");
  const [newValue, setNewValue] = useState("");

  const t = useTranslations();

  const addMatch = () => {
    if (newKey && newValue) {
      setHeaderMatches([...headerMatches, { key: newKey, value: newValue }]);
      setNewKey("");
      setNewValue("");
    }
  };

  const deleteMatch = (index: number) => {
    setHeaderMatches(headerMatches.filter((_, i) => i !== index));
  };

  return (
    <div className="p-6 max-w-3xl mx-auto space-y-6">
      <h1 className="text-2xl font-bold">{t("backend2.canaryConfig.title")}</h1>
      <p className="text-gray-600">Configure canary traffic split and header-based routing for safe deployments.</p>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <div className="space-y-1">
          <label className="text-sm text-gray-600">{"Service"}</label>
          <select
            value={service}
            onChange={(e) => setService(e.target.value)}
            className="w-full border rounded px-3 py-2 text-sm"
          >
            <option value="identity-service">identity-service</option>
            <option value="policy-service">policy-service</option>
            <option value="audit-service">audit-service</option>
          </select>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="space-y-1">
            <label className="text-sm text-gray-600">{"Canary Weight"}</label>
            <input
              type="range"
              min={0}
              max={100}
              value={canaryWeight}
              onChange={(e) => setCanaryWeight(parseInt(e.target.value, 10))}
              className="w-full"
            />
            <div className="text-sm font-medium text-center">{canaryWeight}%</div>
          </div>
          <div className="space-y-1">
            <label className="text-sm text-gray-600">{"Stable Weight"}</label>
            <div className="h-2 bg-gray-200 rounded-full mt-4">
              <div className="h-2 bg-blue-600 rounded-full" style={{ width: `${stableWeight}%` }} />
            </div>
            <div className="text-sm font-medium text-center">{stableWeight}%</div>
          </div>
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{"Header Match"}</h2>
        <div className="space-y-2">
          {headerMatches.map((match, index) => (
            <div key={index} className="flex items-center gap-2">
              <input
                type="text"
                value={match.key}
                readOnly
                className="w-1/3 border rounded px-2 py-1 text-sm font-mono"
              />
              <input
                type="text"
                value={match.value}
                readOnly
                className="flex-1 border rounded px-2 py-1 text-sm font-mono"
              />
              <button
                onClick={() => deleteMatch(index)}
                className="text-sm text-red-600 hover:text-red-700"
              >
                {"Delete"}
              </button>
            </div>
          ))}
        </div>
        <div className="flex gap-2">
          <input
            type="text"
            value={newKey}
            onChange={(e) => setNewKey(e.target.value)}
            placeholder="header-key"
            className="w-1/3 border rounded px-3 py-2 text-sm font-mono"
          />
          <input
            type="text"
            value={newValue}
            onChange={(e) => setNewValue(e.target.value)}
            placeholder="header-value"
            className="flex-1 border rounded px-3 py-2 text-sm font-mono"
          />
          <button
            onClick={addMatch}
            className="px-4 py-2 bg-blue-600 text-white rounded text-sm"
          >
            {"Add Match"}
          </button>
        </div>
      </section>

      <div className="flex justify-end gap-3">
        <button className="px-4 py-2 border rounded text-sm">{"Rollback"}</button>
        <button className="px-4 py-2 bg-blue-600 text-white rounded text-sm">{"Promote"}</button>
      </div>
    </div>
  );
}

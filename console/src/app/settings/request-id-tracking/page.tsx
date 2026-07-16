'use client';

import { useState } from "react";
import { useTranslations } from "@/lib/i18n";

export default function RequestIdTrackingPage() {
  const [enabled, setEnabled] = useState(true);
  const [headerName, setHeaderName] = useState("X-Request-ID");
  const [generateMissing, setGenerateMissing] = useState(true);
  const [propagate, setPropagate] = useState(true);
  const [includeInResponse, setIncludeInResponse] = useState(true);
  const [sample, setSample] = useState(1.0);

  const t = useTranslations();

  return (
    <div className="p-6 max-w-3xl mx-auto space-y-6">
      <h1 className="text-2xl font-bold">{t("backend2.requestIdTracking.title")}</h1>
      <p className="text-gray-600">Configure request ID generation, propagation, and sampling across the gateway.</p>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <label className="flex items-center gap-2">
          <input
            type="checkbox"
            checked={enabled}
            onChange={(e) => setEnabled(e.target.checked)}
            className="w-4 h-4"
          />
          <span className="font-medium">{"Enabled"}</span>
        </label>

        <div className="space-y-1">
          <label className="text-sm text-gray-600">{t("backend2.requestIdTracking.headerName")}</label>
          <input
            type="text"
            value={headerName}
            onChange={(e) => setHeaderName(e.target.value)}
            className="w-full border rounded px-3 py-2 text-sm font-mono"
            disabled={!enabled}
          />
        </div>

        <div className="space-y-2">
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={generateMissing}
              onChange={(e) => setGenerateMissing(e.target.checked)}
              disabled={!enabled}
            />
            {"Generate Missing"}
          </label>
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={propagate}
              onChange={(e) => setPropagate(e.target.checked)}
              disabled={!enabled}
            />
            {"Propagate"}
          </label>
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={includeInResponse}
              onChange={(e) => setIncludeInResponse(e.target.checked)}
              disabled={!enabled}
            />
            {"Include In Response"}
          </label>
        </div>

        <div className="space-y-1">
          <label className="text-sm text-gray-600">{"Sample"}</label>
          <input
            type="range"
            min={0}
            max={1}
            step={0.01}
            value={sample}
            onChange={(e) => setSample(parseFloat(e.target.value))}
            className="w-full"
            disabled={!enabled}
          />
          <div className="text-sm font-medium text-center">{(sample * 100).toFixed(0)}%</div>
        </div>
      </section>

      <div className="flex justify-end">
        <button className="px-4 py-2 bg-blue-600 text-white rounded text-sm" aria-label="Action">
          {"Save"}
        </button>
      </div>
    </div>
  );
}

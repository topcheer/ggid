'use client';

import { useState } from "react";
import { useTranslations } from "@/lib/i18n";

export default function DistributedTracingConfigPage() {
  const [enabled, setEnabled] = useState(true);
  const [sampler, setSampler] = useState("probability");
  const [sampleRate, setSampleRate] = useState(0.1);
  const [collectorUrl, setCollectorUrl] = useState("http://jaeger:4318/v1/traces");
  const [baggageKeys, setBaggageKeys] = useState([
    "tenant_id",
    "user_id",
    "request_id",
  ]);
  const [newBaggage, setNewBaggage] = useState("");

  const t = useTranslations();

  const addBaggage = () => {
    if (newBaggage && !baggageKeys.includes(newBaggage)) {
      setBaggageKeys([...baggageKeys, newBaggage]);
      setNewBaggage("");
    }
  };

  const removeBaggage = (key: string) => {
    setBaggageKeys(baggageKeys.filter((k: any) => k !== key));
  };

  return (
    <div className="p-6 max-w-3xl mx-auto space-y-6">
      <h1 className="text-2xl font-bold">{t("backend2.distributedTracing.title")}</h1>
      <p className="text-gray-600">Configure OpenTelemetry/Jaeger tracing across services.</p>

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
          <label className="text-sm text-gray-600">{"Sampler"}</label>
          <select
            value={sampler}
            onChange={(e) => setSampler(e.target.value)}
            className="w-full border rounded px-3 py-2 text-sm"
            disabled={!enabled}
          >
            <option value="always_on">Always On</option>
            <option value="always_off">Always Off</option>
            <option value="probability">Probability</option>
          </select>
        </div>

        <div className="space-y-1">
          <label className="text-sm text-gray-600">{t("backend2.distributedTracing.sampleRate")}</label>
          <input
            type="range"
            min={0.01}
            max={1}
            step={0.01}
            value={sampleRate}
            onChange={(e) => setSampleRate(parseFloat(e.target.value))}
            className="w-full"
            disabled={!enabled || sampler !== "probability"}
          />
          <div className="text-sm font-medium text-center">{(sampleRate * 100).toFixed(0)}%</div>
        </div>

        <div className="space-y-1">
          <label className="text-sm text-gray-600">{"Collector Url"}</label>
          <input
            type="text"
            value={collectorUrl}
            onChange={(e) => setCollectorUrl(e.target.value)}
            className="w-full border rounded px-3 py-2 text-sm font-mono"
            disabled={!enabled}
          />
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{"Baggage Keys"}</h2>
        <div className="flex flex-wrap gap-2">
          {baggageKeys.map((key: any) => (
            <span
              key={key}
              className="inline-flex items-center gap-1 px-3 py-1 bg-gray-100 rounded text-sm font-mono"
            >
              {key}
              <button
                onClick={() => removeBaggage(key)}
                className="text-red-500 hover:text-red-700 text-xs"
              >
                {"Delete"}
              </button>
            </span>
          ))}
        </div>
        <div className="flex gap-2">
          <input
            type="text"
            value={newBaggage}
            onChange={(e) => setNewBaggage(e.target.value)}
            placeholder="tenant_id"
            className="flex-1 border rounded px-3 py-2 text-sm font-mono"
          />
          <button
            onClick={addBaggage}
            className="px-4 py-2 bg-blue-600 text-white rounded text-sm"
           aria-label="Action">
            {"Add Baggage"}
          </button>
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

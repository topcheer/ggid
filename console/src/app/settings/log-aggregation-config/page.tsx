'use client';

import { useState } from "react";
import { useTranslations } from "@/lib/i18n";

type LogLevel = "debug" | "info" | "warn" | "error";
type LogOutput = { id: string; type: string; endpoint: string; };

export default function LogAggregationConfigPage() {
  const [enabled, setEnabled] = useState(true);
  const [level, setLevel] = useState<LogLevel>("info");
  const [flushInterval, setFlushInterval] = useState(5);
  const [bufferSize, setBufferSize] = useState(1000);
  const [outputs, setOutputs] = useState<LogOutput[]>([
    { id: "stdout", type: "console", endpoint: "stdout" },
    { id: "loki", type: "loki", endpoint: "http://loki:3100" },
  ]);

  const t = useTranslations();

  const addOutput = () => {
    const id = `output-${outputs.length + 1}`;
    setOutputs([...outputs, { id, type: "elasticsearch", endpoint: "" }]);
  };

  const deleteOutput = (id: string) => {
    setOutputs(outputs.filter((o) => o.id !== id));
  };

  const updateOutput = (id: string, patch: Partial<LogOutput>) => {
    setOutputs(outputs.map((o) => (o.id === id ? { ...o, ...patch } : o)));
  };

  return (
    <div className="p-6 max-w-3xl mx-auto space-y-6">
      <h1 className="text-2xl font-bold">{t("backend2.logAggregation.title")}</h1>
      <p className="text-gray-600">Configure log collection, buffering, and forwarding destinations.</p>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <label className="flex items-center gap-2">
          <input
            type="checkbox"
            checked={enabled}
            onChange={(e) => setEnabled(e.target.checked)}
            className="w-4 h-4"
          />
          <span className="font-medium">{t("backend2.logAggregation.enabled")}</span>
        </label>

        <div className="space-y-1">
          <label className="text-sm text-gray-600">{t("backend2.logAggregation.level")}</label>
          <select
            value={level}
            onChange={(e) => setLevel(e.target.value as LogLevel)}
            className="w-full border rounded px-3 py-2 text-sm"
            disabled={!enabled}
          >
            <option value="debug">Debug</option>
            <option value="info">Info</option>
            <option value="warn">Warn</option>
            <option value="error">Error</option>
          </select>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="space-y-1">
            <label className="text-sm text-gray-600">{t("backend2.logAggregation.flushInterval")}</label>
            <input
              type="number"
              min={1}
              value={flushInterval}
              onChange={(e) => setFlushInterval(parseInt(e.target.value, 10) || 1)}
              className="w-full border rounded px-3 py-2 text-sm"
              disabled={!enabled}
            />
          </div>
          <div className="space-y-1">
            <label className="text-sm text-gray-600">{t("backend2.logAggregation.bufferSize")}</label>
            <input
              type="number"
              min={100}
              value={bufferSize}
              onChange={(e) => setBufferSize(parseInt(e.target.value, 10) || 100)}
              className="w-full border rounded px-3 py-2 text-sm"
              disabled={!enabled}
            />
          </div>
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">{t("backend2.logAggregation.outputs")}</h2>
          <button
            onClick={addOutput}
            className="px-3 py-1 bg-blue-600 text-white rounded text-sm"
          >
            {t("backend2.logAggregation.addOutput")}
          </button>
        </div>
        <div className="space-y-3">
          {outputs.map((output) => (
            <div key={output.id} className="border rounded p-3 flex items-center gap-3">
              <input
                type="text"
                value={output.type}
                onChange={(e) => updateOutput(output.id, { type: e.target.value })}
                className="w-32 border rounded px-2 py-1 text-sm font-mono"
                placeholder="console"
              />
              <input
                type="text"
                value={output.endpoint}
                onChange={(e) => updateOutput(output.id, { endpoint: e.target.value })}
                className="flex-1 border rounded px-2 py-1 text-sm font-mono"
                placeholder="http://..."
              />
              <button
                onClick={() => deleteOutput(output.id)}
                className="text-sm text-red-600 hover:text-red-700"
              >
                {t("backend2.logAggregation.delete")}
              </button>
            </div>
          ))}
        </div>
      </section>

      <div className="flex justify-end">
        <button className="px-4 py-2 bg-blue-600 text-white rounded text-sm">
          {t("backend2.logAggregation.save")}
        </button>
      </div>
    </div>
  );
}

'use client';

import { useState } from "react";
import { useTranslations } from "@/lib/i18n";

interface Metric { service: string; cpu: number; memory: number; currentReplicas: number; }

export default function AutoScalingConfigPage() {
  const [enabled, setEnabled] = useState(true);
  const [minReplicas, setMinReplicas] = useState(2);
  const [maxReplicas, setMaxReplicas] = useState(10);
  const [targetCpu, setTargetCpu] = useState(70);
  const [targetMemory, setTargetMemory] = useState(80);
  const [scaleUpDelay, setScaleUpDelay] = useState(60);
  const [scaleDownDelay, setScaleDownDelay] = useState(300);
  const [metrics] = useState<Metric[]>([
    { service: "identity-service", cpu: 45, memory: 62, currentReplicas: 3 },
    { service: "policy-service", cpu: 78, memory: 55, currentReplicas: 4 },
    { service: "audit-service", cpu: 30, memory: 40, currentReplicas: 2 },
  ]);

  const t = useTranslations();

  return (
    <div className="p-6 max-w-3xl mx-auto space-y-6">
      <h1 className="text-2xl font-bold">{t("backend2.autoScaling.title")}</h1>
      <p className="text-gray-600">Configure horizontal pod autoscaling thresholds and stabilization windows.</p>

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

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="space-y-1">
            <label className="text-sm text-gray-600">{t("backend2.autoScaling.minReplicas")}</label>
            <input
              type="number"
              min={0}
              value={minReplicas}
              onChange={(e) => setMinReplicas(parseInt(e.target.value, 10) || 0)}
              className="w-full border rounded px-3 py-2 text-sm"
              disabled={!enabled}
            />
          </div>
          <div className="space-y-1">
            <label className="text-sm text-gray-600">{t("backend2.autoScaling.maxReplicas")}</label>
            <input
              type="number"
              min={1}
              value={maxReplicas}
              onChange={(e) => setMaxReplicas(parseInt(e.target.value, 10) || 1)}
              className="w-full border rounded px-3 py-2 text-sm"
              disabled={!enabled}
            />
          </div>
          <div className="space-y-1">
            <label className="text-sm text-gray-600">{"Target Cpu"}</label>
            <input
              type="range"
              min={10}
              max={100}
              value={targetCpu}
              onChange={(e) => setTargetCpu(parseInt(e.target.value, 10))}
              className="w-full"
              disabled={!enabled}
            />
            <div className="text-sm font-medium text-center">{targetCpu}%</div>
          </div>
          <div className="space-y-1">
            <label className="text-sm text-gray-600">{"Target Memory"}</label>
            <input
              type="range"
              min={10}
              max={100}
              value={targetMemory}
              onChange={(e) => setTargetMemory(parseInt(e.target.value, 10))}
              className="w-full"
              disabled={!enabled}
            />
            <div className="text-sm font-medium text-center">{targetMemory}%</div>
          </div>
          <div className="space-y-1">
            <label className="text-sm text-gray-600">{"Scale Up Delay"}</label>
            <input
              type="number"
              min={0}
              value={scaleUpDelay}
              onChange={(e) => setScaleUpDelay(parseInt(e.target.value, 10) || 0)}
              className="w-full border rounded px-3 py-2 text-sm"
              disabled={!enabled}
            />
          </div>
          <div className="space-y-1">
            <label className="text-sm text-gray-600">{"Scale Down Delay"}</label>
            <input
              type="number"
              min={0}
              value={scaleDownDelay}
              onChange={(e) => setScaleDownDelay(parseInt(e.target.value, 10) || 0)}
              className="w-full border rounded px-3 py-2 text-sm"
              disabled={!enabled}
            />
          </div>
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{"Metrics"}</h2>
        <div className="space-y-3">
          {metrics.map((m) => (
            <div key={m.service} className="border rounded p-3">
              <div className="flex items-center justify-between mb-2">
                <span className="font-mono text-sm">{m.service}</span>
                <span className="text-sm text-gray-500">Current replicas: {m.currentReplicas}</span>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-1">
                  <div className="flex justify-between text-xs"><span>CPU</span><span>{m.cpu}%</span></div>
                  <div className="w-full bg-gray-200 rounded-full h-2">
                    <div className="bg-blue-600 h-2 rounded-full" style={{ width: `${m.cpu}%` }} />
                  </div>
                </div>
                <div className="space-y-1">
                  <div className="flex justify-between text-xs"><span>Memory</span><span>{m.memory}%</span></div>
                  <div className="w-full bg-gray-200 rounded-full h-2">
                    <div className="bg-purple-600 h-2 rounded-full" style={{ width: `${m.memory}%` }} />
                  </div>
                </div>
              </div>
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

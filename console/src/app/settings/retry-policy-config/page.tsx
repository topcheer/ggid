'use client';

import { useState } from "react";
import { useTranslations } from "@/lib/i18n";

interface RouteRetry {
  route: string;
  attempts: number;
  backoff: number;
  conditions: string[];
}

export default function RetryPolicyConfigPage() {
  const [defaultAttempts, setDefaultAttempts] = useState(3);
  const [defaultBackoff, setDefaultBackoff] = useState(500);
  const [retryOn, setRetryOn] = useState([
    "5xx",
    "connect-failure",
    "refused-stream",
  ]);
  const [newCondition, setNewCondition] = useState("");
  const [routes, setRoutes] = useState<RouteRetry[]>([
    { route: "/api/v1/users", attempts: 3, backoff: 250, conditions: ["5xx"] },
    { route: "/api/v1/roles", attempts: 2, backoff: 100, conditions: ["connect-failure"] },
  ]);

  const t = useTranslations();

  const addRetryOn = () => {
    if (newCondition && !retryOn.includes(newCondition)) {
      setRetryOn([...retryOn, newCondition]);
      setNewCondition("");
    }
  };

  const removeRetryOn = (condition: string) => {
    setRetryOn(retryOn.filter((c) => c !== condition));
  };

  const addRoute = () => {
    setRoutes([
      ...routes,
      { route: "/api/v1/new", attempts: defaultAttempts, backoff: defaultBackoff, conditions: [...retryOn] },
    ]);
  };

  const removeRoute = (index: number) => {
    setRoutes(routes.filter((_, i) => i !== index));
  };

  const updateRoute = (index: number, patch: Partial<RouteRetry>) => {
    setRoutes(routes.map((r: any, i: number) => (i === index ? { ...r, ...patch } : r)));
  };

  return (
    <div className="p-6 max-w-3xl mx-auto space-y-6">
      <h1 className="text-2xl font-bold">{t("backend2.retryPolicy.title")}</h1>
      <p className="text-gray-600">Configure retry counts, backoff intervals, and retry conditions.</p>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="space-y-1">
            <label className="text-sm text-gray-600">{"Default Attempts"}</label>
            <input
              type="number"
              min={0}
              value={defaultAttempts}
              onChange={(e) => setDefaultAttempts(parseInt(e.target.value, 10) || 0)}
              className="w-full border rounded px-3 py-2 text-sm"
            />
          </div>
          <div className="space-y-1">
            <label className="text-sm text-gray-600">{"Default Backoff"}</label>
            <input
              type="number"
              min={0}
              value={defaultBackoff}
              onChange={(e) => setDefaultBackoff(parseInt(e.target.value, 10) || 0)}
              className="w-full border rounded px-3 py-2 text-sm"
            />
          </div>
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{"Retry On"}</h2>
        <div className="flex flex-wrap gap-2">
          {retryOn.map((condition) => (
            <span
              key={condition}
              className="inline-flex items-center gap-1 px-3 py-1 bg-gray-100 rounded text-sm font-mono"
            >
              {condition}
              <button
                onClick={() => removeRetryOn(condition)}
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
            value={newCondition}
            onChange={(e) => setNewCondition(e.target.value)}
            placeholder="gateway-timeout"
            className="flex-1 border rounded px-3 py-2 text-sm font-mono"
          />
          <button
            onClick={addRetryOn}
            className="px-4 py-2 bg-blue-600 text-white rounded text-sm"
           aria-label="Action">
            {"Add Retry On"}
          </button>
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">{"Per Route"}</h2>
          <button
            onClick={addRoute}
            className="px-3 py-1 bg-blue-600 text-white rounded text-sm"
           aria-label="Action">
            {"Add Route"}
          </button>
        </div>
        <div className="space-y-3">
          {routes.map((route: any, index: number) => (
            <div key={index} className="border rounded p-3 grid grid-cols-4 gap-3 items-center">
              <input
                type="text"
                value={route.route}
                onChange={(e) => updateRoute(index, { route: e.target.value })}
                className="border rounded px-2 py-1 text-sm font-mono"
              />
              <input
                type="number"
                min={0}
                value={route.attempts}
                onChange={(e) => updateRoute(index, { attempts: parseInt(e.target.value, 10) || 0 })}
                className="border rounded px-2 py-1 text-sm"
                placeholder="Attempts"
              />
              <input
                type="number"
                min={0}
                value={route.backoff}
                onChange={(e) => updateRoute(index, { backoff: parseInt(e.target.value, 10) || 0 })}
                className="border rounded px-2 py-1 text-sm"
                placeholder="Backoff ms"
              />
              <button
                onClick={() => removeRoute(index)}
                className="text-sm text-red-600 hover:text-red-700 text-left"
              >
                {"Delete"}
              </button>
            </div>
          ))}
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

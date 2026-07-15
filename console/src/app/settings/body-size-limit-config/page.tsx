'use client';

import { useState } from "react";
import { useTranslations } from "@/lib/i18n";

interface RouteLimit { route: string; maxRequestBody: number; maxResponseBody: number; }

export default function BodySizeLimitConfigPage() {
  const [maxRequestBody, setMaxRequestBody] = useState(1048576);
  const [maxResponseBody, setMaxResponseBody] = useState(2097152);
  const [allowFileUpload, setAllowFileUpload] = useState(true);
  const [maxFileSize, setMaxFileSize] = useState(16777216);
  const [routes, setRoutes] = useState<RouteLimit[]>([
    { route: "/api/v1/upload", maxRequestBody: 16777216, maxResponseBody: 2097152 },
    { route: "/api/v1/bulk", maxRequestBody: 5242880, maxResponseBody: 10485760 },
  ]);

  const t = useTranslations();

  const addRoute = () => {
    setRoutes([...routes, { route: "/api/v1/new", maxRequestBody, maxResponseBody }]);
  };

  const removeRoute = (index: number) => {
    setRoutes(routes.filter((_, i) => i !== index));
  };

  const updateRoute = (index: number, patch: Partial<RouteLimit>) => {
    setRoutes(routes.map((r, i) => (i === index ? { ...r, ...patch } : r)));
  };

  return (
    <div className="p-6 max-w-3xl mx-auto space-y-6">
      <h1 className="text-2xl font-bold">{t("backend2.bodySizeLimit.title")}</h1>
      <p className="text-gray-600">Configure maximum request and response body sizes globally and per route.</p>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="space-y-1">
            <label className="text-sm text-gray-600">{t("backend2.bodySizeLimit.maxRequestBody")}</label>
            <input
              type="number"
              min={0}
              value={maxRequestBody}
              onChange={(e) => setMaxRequestBody(parseInt(e.target.value, 10) || 0)}
              className="w-full border rounded px-3 py-2 text-sm font-mono"
            />
          </div>
          <div className="space-y-1">
            <label className="text-sm text-gray-600">{t("backend2.bodySizeLimit.maxResponseBody")}</label>
            <input
              type="number"
              min={0}
              value={maxResponseBody}
              onChange={(e) => setMaxResponseBody(parseInt(e.target.value, 10) || 0)}
              className="w-full border rounded px-3 py-2 text-sm font-mono"
            />
          </div>
        </div>

        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={allowFileUpload}
            onChange={(e) => setAllowFileUpload(e.target.checked)}
            className="w-4 h-4"
          />
          {t("backend2.bodySizeLimit.allowFileUpload")}
        </label>

        {allowFileUpload && (
          <div className="space-y-1">
            <label className="text-sm text-gray-600">{t("backend2.bodySizeLimit.maxFileSize")}</label>
            <input
              type="number"
              min={0}
              value={maxFileSize}
              onChange={(e) => setMaxFileSize(parseInt(e.target.value, 10) || 0)}
              className="w-full border rounded px-3 py-2 text-sm font-mono"
            />
          </div>
        )}
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">{t("backend2.bodySizeLimit.routeOverrides")}</h2>
          <button onClick={addRoute} className="px-3 py-1 bg-blue-600 text-white rounded text-sm">{t("backend2.bodySizeLimit.addRoute")}</button>
        </div>
        <div className="space-y-3">
          {routes.map((route, index) => (
            <div key={index} className="border rounded p-3 grid grid-cols-3 gap-3 items-center">
              <input
                type="text"
                value={route.route}
                onChange={(e) => updateRoute(index, { route: e.target.value })}
                className="border rounded px-2 py-1 text-sm font-mono"
              />
              <input
                type="number"
                min={0}
                value={route.maxRequestBody}
                onChange={(e) => updateRoute(index, { maxRequestBody: parseInt(e.target.value, 10) || 0 })}
                className="border rounded px-2 py-1 text-sm font-mono"
                placeholder="Max request"
              />
              <button
                onClick={() => removeRoute(index)}
                className="text-sm text-red-600 hover:text-red-700 text-left"
              >
                Delete
              </button>
            </div>
          ))}
        </div>
      </section>

      <div className="flex justify-end">
        <button className="px-4 py-2 bg-blue-600 text-white rounded text-sm">
          {t("backend2.bodySizeLimit.save")}
        </button>
      </div>
    </div>
  );
}

'use client';

import { useState } from "react";
import { useTranslations } from "@/lib/i18n";

export default function CompressionConfigPage() {
  const [enabled, setEnabled] = useState(true);
  const [minSize, setMinSize] = useState(1024);
  const [level, setLevel] = useState(6);
  const [mimeTypes, setMimeTypes] = useState([
    "application/json",
    "text/html",
    "text/css",
    "application/javascript",
  ]);
  const [newMime, setNewMime] = useState("");

  const t = useTranslations();

  const addMimeType = () => {
    if (newMime && !mimeTypes.includes(newMime)) {
      setMimeTypes([...mimeTypes, newMime]);
      setNewMime("");
    }
  };

  const removeMimeType = (type: string) => {
    setMimeTypes(mimeTypes.filter((t: any) => t !== type));
  };

  return (
    <div className="p-6 max-w-3xl mx-auto space-y-6">
      <h1 className="text-2xl font-bold">{t("backend2.compression.title")}</h1>
      <p className="text-gray-600">Configure response compression settings for the gateway.</p>

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
            <label className="text-sm text-gray-600">{"Min Size"}</label>
            <input
              type="number"
              min={0}
              value={minSize}
              onChange={(e) => setMinSize(parseInt(e.target.value, 10) || 0)}
              className="w-full border rounded px-3 py-2 text-sm"
              disabled={!enabled}
            />
          </div>

          <div className="space-y-1">
            <label className="text-sm text-gray-600">{"Level"}</label>
            <input
              type="range"
              min={1}
              max={9}
              value={level}
              onChange={(e) => setLevel(parseInt(e.target.value, 10))}
              className="w-full"
              disabled={!enabled}
            />
            <div className="text-sm font-medium text-center">{level}</div>
          </div>
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{"Mime Types"}</h2>
        <div className="flex flex-wrap gap-2">
          {mimeTypes.map((type: any) => (
            <span
              key={type}
              className="inline-flex items-center gap-1 px-3 py-1 bg-gray-100 rounded text-sm font-mono"
            >
              {type}
              <button
                onClick={() => removeMimeType(type)}
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
            value={newMime}
            onChange={(e) => setNewMime(e.target.value)}
            placeholder="text/plain"
            className="flex-1 border rounded px-3 py-2 text-sm font-mono"
          />
          <button
            onClick={addMimeType}
            className="px-4 py-2 bg-blue-600 text-white rounded text-sm"
           aria-label="Action">
            {"Add Mime Type"}
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

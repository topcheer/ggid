'use client';

import { useState } from "react";
import { useTranslations } from "@/lib/i18n";

export default function HostValidationConfigPage() {
  const [enabled, setEnabled] = useState(true);
  const [mode, setMode] = useState("whitelist");
  const [whitelist, setWhitelist] = useState<string[]>([]);
  const [blacklist, setBlacklist] = useState<string[]>([]);
  const [newHost, setNewHost] = useState("");

  const t = useTranslations();

  const activeList = mode === "whitelist" ? whitelist : blacklist;
  const setActiveList = mode === "whitelist" ? setWhitelist : setBlacklist;

  const addHost = () => {
    if (newHost && !activeList.includes(newHost)) {
      setActiveList([...activeList, newHost]);
      setNewHost("");
    }
  };

  const deleteHost = (host: string) => {
    setActiveList(activeList.filter((h: any) => h !== host));
  };

  return (
    <div className="p-6 max-w-3xl mx-auto space-y-6">
      <h1 className="text-2xl font-bold">{t("backend2.hostValidation.title")}</h1>
      <p className="text-gray-600">Restrict incoming requests by validating the Host header.</p>

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
          <label className="text-sm text-gray-600">{"Mode"}</label>
          <select
            value={mode}
            onChange={(e) => setMode(e.target.value)}
            className="w-full border rounded px-3 py-2 text-sm"
            disabled={!enabled}
          >
            <option value="whitelist">{"Whitelist"}</option>
            <option value="blacklist">{"Blacklist"}</option>
          </select>
        </div>

        <div className="space-y-2">
          <h2 className="text-lg font-semibold">{mode === "whitelist" ? "Whitelist" : "Blacklist"}</h2>
          <div className="flex flex-wrap gap-2">
            {activeList.map((host: any) => (
              <span
                key={host}
                className="inline-flex items-center gap-1 px-3 py-1 bg-gray-100 rounded text-sm font-mono"
              >
                {host}
                <button
                  onClick={() => deleteHost(host)}
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
              value={newHost}
              onChange={(e) => setNewHost(e.target.value)}
              placeholder="host.example.com"
              className="flex-1 border rounded px-3 py-2 text-sm font-mono"
            />
            <button
              onClick={addHost}
              className="px-4 py-2 bg-blue-600 text-white rounded text-sm"
             aria-label="Action">
              {"Add Host"}
            </button>
          </div>
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

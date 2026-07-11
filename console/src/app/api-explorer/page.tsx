"use client";

import { useState } from "react";
import { useApi } from "@/lib/api";
import { Send, Plus, Trash2, ChevronDown } from "lucide-react";

interface Endpoint {
  id: string;
  method: string;
  path: string;
  headers: { key: string; value: string }[];
  body: string;
}

const METHODS = ["GET", "POST", "PUT", "PATCH", "DELETE"];

const QUICK_ENDPOINTS = [
  { method: "GET", path: "/api/v1/users" },
  { method: "POST", path: "/api/v1/auth/register" },
  { method: "POST", path: "/api/v1/auth/login" },
  { method: "GET", path: "/api/v1/roles" },
  { method: "GET", path: "/api/v1/orgs" },
  { method: "GET", path: "/api/v1/audit/events" },
  { method: "GET", path: "/api/v1/policies" },
  { method: "GET", path: "/healthz" },
];

export default function APIExplorerPage() {
  const { API_BASE, TENANT_ID } = useApi();
  const [endpoints, setEndpoints] = useState<Endpoint[]>([
    {
      id: "1",
      method: "GET",
      path: "/api/v1/users",
      headers: [
        { key: "X-Tenant-ID", value: TENANT_ID },
      ],
      body: "",
    },
  ]);
  const [responses, setResponses] = useState<Record<string, { status: number; body: string; time: number }>>({});
  const [loading, setLoading] = useState<string | null>(null);
  const [snippets, setSnippets] = useState<Record<string, string>>({});

  const generateSnippet = (ep: Endpoint, lang: string): string => {
    const url = `${API_BASE}${ep.path}`;
    const hdrs = ep.headers.filter(h => h.key).map(h => `-H '${h.key}: ${h.value}'`).join(" ");
    switch (lang) {
      case "curl":
        return ep.method === "GET"
          ? `curl -X ${ep.method} '${url}' ${hdrs} -H 'Authorization: Bearer <JWT>'`
          : `curl -X ${ep.method} '${url}' ${hdrs} -H 'Authorization: Bearer <JWT>' -H 'Content-Type: application/json' -d '${ep.body}'`;
      case "javascript":
        return `const res = await fetch('${url}', {\n  method: '${ep.method}',\n  headers: { 'Authorization': 'Bearer <JWT>', ${ep.headers.map(h => `'${h.key}': '${h.value}'`).join(", ")} },${ep.body ? `\n  body: JSON.stringify(${ep.body}),` : ""}\n});\nconst data = await res.json();\nconsole.log(data);`;
      case "python":
        return `import requests\n\nresp = requests.${ep.method.toLowerCase()}(\n    '${url}',\n    headers={'Authorization': 'Bearer <JWT>', ${ep.headers.map(h => `'${h.key}': '${h.value}'`).join(", ")}},${ep.body ? `\n    json=${ep.body},` : ""}\n)\nprint(resp.json())`;
      case "go":
        return `req, _ := http.NewRequest("${ep.method}", "${url}", nil)\nreq.Header.Set("Authorization", "Bearer <JWT>")\n${ep.headers.map(h => `req.Header.Set("${h.key}", "${h.value}")`).join("\n")}\nclient := &http.Client{}\nresp, _ := client.Do(req)\ndefer resp.Body.Close()`;
      default: return "";
    }
  };

  const addEndpoint = () => {
    const id = String(Date.now());
    setEndpoints([...endpoints, {
      id, method: "GET", path: "/api/v1/users",
      headers: [{ key: "X-Tenant-ID", value: TENANT_ID }],
      body: "",
    }]);
  };

  const removeEndpoint = (id: string) => {
    setEndpoints(endpoints.filter(e => e.id !== id));
    setResponses(prev => { const c = { ...prev }; delete c[id]; return c; });
  };

  const updateEndpoint = (id: string, field: keyof Endpoint, value: any) => {
    setEndpoints(endpoints.map(e => e.id === id ? { ...e, [field]: value } : e));
  };

  const updateHeader = (epId: string, idx: number, field: string, value: string) => {
    setEndpoints(endpoints.map(e => {
      if (e.id !== epId) return e;
      const headers = [...e.headers];
      headers[idx] = { ...headers[idx], [field]: value };
      return { ...e, headers };
    }));
  };

  const addHeader = (epId: string) => {
    setEndpoints(endpoints.map(e => e.id === epId ? { ...e, headers: [...e.headers, { key: "", value: "" }] } : e));
  };

  const removeHeader = (epId: string, idx: number) => {
    setEndpoints(endpoints.map(e => {
      if (e.id !== epId) return e;
      return { ...e, headers: e.headers.filter((_, i) => i !== idx) };
    }));
  };

  const sendRequest = async (ep: Endpoint) => {
    setLoading(ep.id);
    const url = API_BASE + ep.path;
    const headers: Record<string, string> = {};
    ep.headers.forEach(h => { if (h.key) headers[h.key] = h.value; });
    const authToken = typeof window !== "undefined" ? localStorage.getItem("ggid_access_token") : null;
    if (authToken) headers["Authorization"] = `Bearer ${authToken}`;

    const start = performance.now();
    try {
      const resp = await fetch(url, {
        method: ep.method,
        headers,
        body: ep.body || undefined,
      });
      const elapsed = Math.round(performance.now() - start);
      const text = await resp.text();
      let body = text;
      try { body = JSON.stringify(JSON.parse(text), null, 2); } catch {}
      setResponses(prev => ({ ...prev, [ep.id]: { status: resp.status, body, time: elapsed } }));
    } catch (err: any) {
      const elapsed = Math.round(performance.now() - start);
      setResponses(prev => ({ ...prev, [ep.id]: { status: 0, body: `Error: ${err.message}`, time: elapsed } }));
    } finally {
      setLoading(null);
    }
  };

  const loadQuickEndpoint = (method: string, path: string) => {
    const id = String(Date.now());
    setEndpoints([...endpoints, {
      id, method, path,
      headers: [{ key: "X-Tenant-ID", value: TENANT_ID }],
      body: method === "POST" || method === "PUT" ? '{\n  \n}' : "",
    }]);
  };

  return (
    <div className="min-h-screen bg-gray-50 p-6 dark:bg-gray-900">
      <div className="mx-auto max-w-6xl">
        <h1 className="mb-2 text-2xl font-bold text-gray-900 dark:text-gray-100">API Explorer</h1>
        <p className="mb-6 text-sm text-gray-500 dark:text-gray-400">
          Interactive API testing — send requests to <code className="rounded bg-gray-200 px-1 dark:bg-gray-700">{API_BASE}</code>
        </p>

        {/* Quick endpoints */}
        <div className="mb-6 flex flex-wrap gap-2">
          {QUICK_ENDPOINTS.map(qe => (
            <button
              key={`${qe.method}-${qe.path}`}
              onClick={() => loadQuickEndpoint(qe.method, qe.path)}
              className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-1.5 text-xs font-medium text-gray-700 transition-colors hover:bg-gray-100 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
            >
              <span className={`rounded px-1.5 py-0.5 text-[10px] font-bold ${methodColor(qe.method)}`}>{qe.method}</span>
              {qe.path}
            </button>
          ))}
        </div>

        {/* Endpoints */}
        <div className="space-y-4">
          {endpoints.map(ep => (
            <div key={ep.id} className="rounded-xl border border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-800">
              {/* Request row */}
              <div className="flex items-center gap-2 p-4">
                <div className="relative">
                  <select
                    value={ep.method}
                    onChange={e => updateEndpoint(ep.id, "method", e.target.value)}
                    className={`appearance-none rounded-lg border-0 px-3 py-2 text-sm font-bold text-white focus:ring-2 focus:ring-blue-500 ${methodBg(ep.method)}`}
                  >
                    {METHODS.map(m => <option key={m} value={m}>{m}</option>)}
                  </select>
                  <ChevronDown className="pointer-events-none absolute right-2 top-1/2 h-4 w-4 -translate-y-1/2 text-white" />
                </div>
                <input
                  type="text"
                  value={ep.path}
                  onChange={e => updateEndpoint(ep.id, "path", e.target.value)}
                  placeholder="/api/v1/..."
                  className="flex-1 rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-900 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100"
                />
                <button
                  onClick={() => sendRequest(ep)}
                  disabled={loading === ep.id}
                  className="flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700 disabled:opacity-50"
                >
                  <Send className="h-4 w-4" />
                  {loading === ep.id ? "Sending..." : "Send"}
                </button>
                <button
                  onClick={() => removeEndpoint(ep.id)}
                  className="rounded-lg p-2 text-gray-400 hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-950"
                >
                  <Trash2 className="h-4 w-4" />
                </button>
              </div>

              {/* Headers */}
              <div className="border-t border-gray-100 px-4 py-3 dark:border-gray-700">
                <div className="mb-2 flex items-center justify-between">
                  <span className="text-xs font-semibold uppercase text-gray-500 dark:text-gray-400">Headers</span>
                  <button onClick={() => addHeader(ep.id)} className="flex items-center gap-1 text-xs text-blue-600 hover:underline">
                    <Plus className="h-3 w-3" /> Add
                  </button>
                </div>
                {ep.headers.map((h, i) => (
                  <div key={i} className="mb-1 flex gap-2">
                    <input
                      type="text"
                      value={h.key}
                      onChange={e => updateHeader(ep.id, i, "key", e.target.value)}
                      placeholder="Header name"
                      className="w-1/3 rounded border border-gray-200 px-2 py-1 text-xs dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100"
                    />
                    <input
                      type="text"
                      value={h.value}
                      onChange={e => updateHeader(ep.id, i, "value", e.target.value)}
                      placeholder="Header value"
                      className="flex-1 rounded border border-gray-200 px-2 py-1 text-xs dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100"
                    />
                    <button onClick={() => removeHeader(ep.id, i)} className="text-gray-400 hover:text-red-600">
                      <Trash2 className="h-3 w-3" />
                    </button>
                  </div>
                ))}
              </div>

              {/* Body */}
              {(ep.method === "POST" || ep.method === "PUT" || ep.method === "PATCH") && (
                <div className="border-t border-gray-100 px-4 py-3 dark:border-gray-700">
                  <span className="mb-2 block text-xs font-semibold uppercase text-gray-500 dark:text-gray-400">Body (JSON)</span>
                  <textarea
                    value={ep.body}
                    onChange={e => updateEndpoint(ep.id, "body", e.target.value)}
                    rows={5}
                    placeholder='{"key": "value"}'
                    className="w-full rounded-lg border border-gray-200 px-3 py-2 font-mono text-xs dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100"
                  />
                </div>
              )}

              {/* Code Snippet Generator */}
              <div className="border-t border-gray-100 px-4 py-3 dark:border-gray-700">
                <div className="mb-2 flex items-center gap-2">
                  <span className="text-xs font-semibold uppercase text-gray-500 dark:text-gray-400">Code Snippet</span>
                  <div className="flex gap-1 ml-auto">
                    {["curl", "javascript", "python", "go"].map(lang => (
                      <button key={lang} onClick={() => {
                        const newSnippets = { ...snippets };
                        newSnippets[ep.id] = lang;
                        setSnippets(newSnippets);
                      }} className={`px-2 py-0.5 text-[10px] font-medium rounded transition ${snippets[ep.id] === lang ? "bg-indigo-600 text-white" : "text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700"}`}>
                        {lang === "javascript" ? "JS" : lang.charAt(0).toUpperCase() + lang.slice(1)}
                      </button>
                    ))}
                  </div>
                </div>
                <pre className="max-h-48 overflow-auto rounded-lg bg-gray-900 p-3 text-xs text-gray-100 dark:bg-black">
                  {generateSnippet(ep, snippets[ep.id] || "curl")}
                </pre>
              </div>

              {/* Response */}
              {responses[ep.id] && (
                <div className="border-t border-gray-100 px-4 py-3 dark:border-gray-700">
                  <div className="mb-2 flex items-center gap-3">
                    <span className="text-xs font-semibold uppercase text-gray-500 dark:text-gray-400">Response</span>
                    <span className={`rounded px-2 py-0.5 text-xs font-bold ${statusColor(responses[ep.id].status)}`}>
                      {responses[ep.id].status}
                    </span>
                    <span className="text-xs text-gray-400">{responses[ep.id].time}ms</span>
                  </div>
                  <pre className="max-h-96 overflow-auto rounded-lg bg-gray-50 p-3 text-xs text-gray-800 dark:bg-gray-900 dark:text-gray-200">
                    {responses[ep.id].body}
                  </pre>
                </div>
              )}
            </div>
          ))}
        </div>

        <button
          onClick={addEndpoint}
          className="mt-4 flex items-center gap-2 rounded-lg border border-dashed border-gray-300 px-4 py-2 text-sm text-gray-600 hover:border-blue-400 hover:text-blue-600 dark:border-gray-600 dark:text-gray-400"
        >
          <Plus className="h-4 w-4" /> Add Request
        </button>
      </div>
    </div>
  );
}

function methodColor(method: string): string {
  switch (method) {
    case "GET": return "bg-blue-100 text-blue-700";
    case "POST": return "bg-green-100 text-green-700";
    case "PUT": return "bg-orange-100 text-orange-700";
    case "PATCH": return "bg-yellow-100 text-yellow-700";
    case "DELETE": return "bg-red-100 text-red-700";
    default: return "bg-gray-100 text-gray-700";
  }
}

function methodBg(method: string): string {
  switch (method) {
    case "GET": return "bg-blue-600";
    case "POST": return "bg-green-600";
    case "PUT": return "bg-orange-600";
    case "PATCH": return "bg-yellow-600";
    case "DELETE": return "bg-red-600";
    default: return "bg-gray-600";
  }
}

function statusColor(status: number): string {
  if (status >= 200 && status < 300) return "bg-green-100 text-green-700";
  if (status >= 300 && status < 400) return "bg-blue-100 text-blue-700";
  if (status >= 400 && status < 500) return "bg-yellow-100 text-yellow-700";
  if (status >= 500) return "bg-red-100 text-red-700";
  return "bg-gray-100 text-gray-700";
}

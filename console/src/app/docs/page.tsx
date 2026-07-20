"use client";
import { useState, useEffect } from "react";
import { Code, Loader2, Globe } from "lucide-react";
import { usePageTitle } from "@/lib/usePageTitle";
import { API_BASE_URL } from "@/lib/api-config";

const API_BASE = API_BASE_URL;

export default function DocsPage() {
  usePageTitle("API Documentation");
  const [swaggerUrl, setSwaggerUrl] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // Try common OpenAPI/Swagger endpoints
    const urls = [
      `${API_BASE}/swagger/`,
      `${API_BASE}/api/v1/openapi.json`,
      `${API_BASE}/docs/`,
      `${API_BASE}/swagger-ui/`,
    ];
    Promise.any(urls.map(u => fetch(u).then(r => { if (r.ok) return u; throw new Error("not found"); })))
      .then(u => setSwaggerUrl(u))
      .catch(() => setSwaggerUrl(`${API_BASE}/swagger/`))
      .finally(() => setLoading(false));
  }, []);

  return (
    <div className="h-screen flex flex-col">
      <div className="flex items-center gap-2 border-b border-gray-200 px-6 py-3 dark:border-gray-800">
        <Code className="h-5 w-5 text-blue-500" />
        <h1 className="text-lg font-semibold text-gray-900 dark:text-white">API Documentation</h1>
        <a href={swaggerUrl} target="_blank" rel="noopener" className="ml-auto flex items-center gap-1 text-sm text-blue-600 hover:underline">
          <Globe className="h-4 w-4" /> Open in new tab
        </a>
      </div>
      <div className="flex-1">
        {loading ? (
          <div className="flex justify-center py-20"><Loader2 className="w-8 h-8 animate-spin text-blue-600" /></div>
        ) : (
          <iframe src={swaggerUrl} className="h-full w-full border-0" title="Swagger UI" />
        )}
      </div>
    </div>
  );
}

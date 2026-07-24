"use client";
import { useEffect } from "react";
import { Loader2 } from "lucide-react";
import { API_BASE_URL } from "@/lib/api-config";

export default function SwaggerPage() {
  useEffect(() => {
    // Redirect to backend swagger UI
    const swaggerUrl = `${API_BASE_URL}/swagger/`;
    window.location.replace(swaggerUrl);
  }, []);

  return (
    <div className="flex min-h-[60vh] items-center justify-center">
      <div className="text-center">
        <Loader2 className="mx-auto mb-3 h-6 w-6 animate-spin text-blue-600" />
        <p className="text-sm text-gray-500 dark:text-gray-400">Redirecting to Swagger UI...</p>
        <a href={API_BASE_URL + "/swagger/"} className="mt-3 inline-block text-sm text-blue-600 hover:underline">
          Click here if not redirected
        </a>
      </div>
    </div>
  );
}

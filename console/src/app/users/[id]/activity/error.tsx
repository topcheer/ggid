"use client";
import { RefreshCw } from "lucide-react";
export default function Error({ reset }: { error: Error; reset: () => void }) {
  return (
    <div className="flex flex-col items-center justify-center py-20">
      <RefreshCw className="w-6 h-6 text-red-500 mb-4" />
      <p className="text-sm text-gray-500 mb-4">Failed to load.</p>
      <button onClick={reset} className="px-4 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium">Try again</button>
    </div>
  );
}

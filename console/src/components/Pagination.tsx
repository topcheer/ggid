"use client";

import { ChevronLeft, ChevronRight, ChevronsLeft, ChevronsRight } from "lucide-react";

interface PaginationProps {
  page: number;
  pageSize: number;
  total: number;
  onPageChange: (page: number) => void;
  className?: string;
}

export function Pagination({ page, pageSize, total, onPageChange, className }: PaginationProps) {
  const totalPages = Math.max(1, Math.ceil(total / pageSize));
  if (total <= pageSize) return null; // No pagination needed for single page

  const start = (page - 1) * pageSize + 1;
  const end = Math.min(page * pageSize, total);

  return (
    <div className={`flex items-center justify-between gap-2 ${className || ""}`}>
      <span className="text-xs text-gray-500 dark:text-gray-400">
        {start}-{end} of {total}
      </span>
      <div className="flex items-center gap-1">
        <button onClick={() => onPageChange(1)} disabled={page === 1}
          className="p-1.5 rounded-lg text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800 disabled:opacity-30" aria-label="First page">
          <ChevronsLeft className="w-3.5 h-3.5" />
        </button>
        <button onClick={() => onPageChange(page - 1)} disabled={page === 1}
          className="p-1.5 rounded-lg text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800 disabled:opacity-30" aria-label="Previous page">
          <ChevronLeft className="w-3.5 h-3.5" />
        </button>
        <span className="px-2 text-xs font-medium text-gray-900 dark:text-white">
          {page} / {totalPages}
        </span>
        <button onClick={() => onPageChange(page + 1)} disabled={page === totalPages}
          className="p-1.5 rounded-lg text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800 disabled:opacity-30" aria-label="Next page">
          <ChevronRight className="w-3.5 h-3.5" />
        </button>
        <button onClick={() => onPageChange(totalPages)} disabled={page === totalPages}
          className="p-1.5 rounded-lg text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800 disabled:opacity-30" aria-label="Last page">
          <ChevronsRight className="w-3.5 h-3.5" />
        </button>
      </div>
    </div>
  );
}

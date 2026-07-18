"use client";
import Link from "next/link";
import { useState } from "react";
import { Home, SearchX, Search, ArrowRight, Shield } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

const QUICK_LINKS = [
  { href: "/dashboard", label: "Dashboard" },
  { href: "/users", label: "Users" },
  { href: "/audit", label: "Audit" },
  { href: "/settings", label: "Settings" },
  { href: "/docs", label: "Documentation" },
];

export default function NotFound() {
  const t = useTranslations();
  const [search, setSearch] = useState("");

  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-gradient-to-br from-blue-50 via-white to-purple-50 dark:from-gray-950 dark:via-gray-900 dark:to-blue-950 px-4">
      {/* Brand Logo */}
      <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-gradient-to-br from-blue-600 to-purple-600 shadow-lg mb-6">
        <Shield className="h-8 w-8 text-white" />
      </div>

      {/* 404 Number */}
      <div className="flex items-end gap-2 mb-2">
        <span className="text-7xl font-bold bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">404</span>
      </div>
      <p className="text-lg font-semibold text-gray-900 dark:text-white">Page Not Found</p>
      <p className="mt-1 text-sm text-gray-500 dark:text-gray-400 text-center max-w-sm">
        {t("common.notFoundMessage") || "The page you're looking for doesn't exist or has been moved."}
      </p>

      {/* Search */}
      <div className="relative w-full max-w-sm mt-6">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
        <input
          type="text"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Search for a page..."
          className="w-full pl-9 pr-3 py-2.5 rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 text-sm text-gray-900 dark:text-white shadow-sm"
        />
      </div>

      {/* Quick Links */}
      <div className="flex flex-wrap items-center justify-center gap-2 mt-4">
        {QUICK_LINKS.filter((l) => !search || l.label.toLowerCase().includes(search.toLowerCase())).map((link) => (
          <Link key={link.href} href={link.href}
            className="flex items-center gap-1 px-3 py-1.5 rounded-lg bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 text-sm text-gray-600 dark:text-gray-400 hover:border-blue-400 hover:text-blue-600 transition-colors">
            {link.label}<ArrowRight className="w-3 h-3" />
          </Link>
        ))}
      </div>

      {/* Home button */}
      <Link href="/dashboard"
        className="mt-8 flex items-center gap-2 rounded-xl bg-gradient-to-r from-blue-600 to-purple-600 px-6 py-2.5 text-sm font-medium text-white hover:opacity-90 shadow-md transition-opacity">
        <Home className="h-4 w-4" />Back to Dashboard
      </Link>
    </div>
  );
}

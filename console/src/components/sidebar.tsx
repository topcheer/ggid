"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  Users,
  Shield,
  Building2,
  ScrollText,
  KeyRound,
  Settings,
  LayoutDashboard,
} from "lucide-react";

const navItems = [
  { href: "/", label: "Dashboard", icon: LayoutDashboard },
  { href: "/users", label: "Users", icon: Users },
  { href: "/roles", label: "Roles & Permissions", icon: Shield },
  { href: "/organizations", label: "Organizations", icon: Building2 },
  { href: "/audit", label: "Audit Log", icon: ScrollText },
  { href: "/oauth-clients", label: "OAuth Clients", icon: KeyRound },
  { href: "/settings", label: "Settings", icon: Settings },
];

export function Sidebar() {
  const pathname = usePathname();

  return (
    <aside
      className="flex flex-col border-r border-gray-200 bg-white"
      style={{ width: "var(--sidebar-width)" }}
    >
      <div className="flex h-16 items-center gap-2 border-b border-gray-200 px-6">
        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-brand-600 text-white font-bold">
          G
        </div>
        <span className="text-lg font-semibold">GGID</span>
      </div>

      <nav className="flex-1 space-y-1 p-3">
        {navItems.map((item) => {
          const Icon = item.icon;
          const active =
            pathname === item.href ||
            (item.href !== "/" && pathname.startsWith(item.href));
          return (
            <Link
              key={item.href}
              href={item.href}
              className={`flex items-center gap-3 rounded-lg px-3 py-2 text-sm transition-colors ${
                active
                  ? "bg-brand-50 text-brand-700 font-medium"
                  : "text-gray-600 hover:bg-gray-50 hover:text-gray-900"
              }`}
            >
              <Icon className="h-4 w-4" />
              {item.label}
            </Link>
          );
        })}
      </nav>

      <div className="border-t border-gray-200 p-4">
        <div className="flex items-center gap-3">
          <div className="flex h-8 w-8 items-center justify-center rounded-full bg-gray-200 text-sm font-medium">
            A
          </div>
          <div className="flex-1 min-w-0">
            <p className="truncate text-sm font-medium">admin@ggid.dev</p>
            <p className="truncate text-xs text-gray-500">Administrator</p>
          </div>
        </div>
      </div>
    </aside>
  );
}

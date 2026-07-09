"use client";

import { useUsers } from "@/lib/api";
import {
  Users as UsersIcon,
  ShieldCheck,
  Activity,
  TrendingUp,
} from "lucide-react";

export default function DashboardPage() {
  const { users, loading } = useUsers();

  const stats = [
    {
      label: "Total Users",
      value: loading ? "..." : String(users.length),
      icon: UsersIcon,
      color: "bg-blue-500",
    },
    {
      label: "Active Sessions",
      value: "—",
      icon: Activity,
      color: "bg-green-500",
    },
    {
      label: "Roles",
      value: "3",
      icon: ShieldCheck,
      color: "bg-purple-500",
    },
    {
      label: "Audit Events (24h)",
      value: "—",
      icon: TrendingUp,
      color: "bg-orange-500",
    },
  ];

  return (
    <div>
      <h1 className="mb-6 text-2xl font-bold">Dashboard</h1>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {stats.map((stat) => {
          const Icon = stat.icon;
          return (
            <div
              key={stat.label}
              className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm"
            >
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-gray-500">
                    {stat.label}
                  </p>
                  <p className="mt-1 text-3xl font-bold">{stat.value}</p>
                </div>
                <div
                  className={`flex h-12 w-12 items-center justify-center rounded-lg ${stat.color}`}
                >
                  <Icon className="h-6 w-6 text-white" />
                </div>
              </div>
            </div>
          );
        })}
      </div>

      <div className="mt-8 rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
        <h2 className="mb-4 text-lg font-semibold">Recent Users</h2>
        {loading ? (
          <p className="text-gray-500">Loading...</p>
        ) : (
          <div className="space-y-2">
            {users.slice(0, 5).map((user) => (
              <div
                key={user.id}
                className="flex items-center justify-between rounded-lg px-3 py-2 hover:bg-gray-50"
              >
                <div className="flex items-center gap-3">
                  <div className="flex h-8 w-8 items-center justify-center rounded-full bg-gray-200 text-sm font-medium uppercase">
                    {user.username[0]}
                  </div>
                  <div>
                    <p className="text-sm font-medium">{user.username}</p>
                    <p className="text-xs text-gray-500">{user.email}</p>
                  </div>
                </div>
                <span
                  className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                    user.status === "active"
                      ? "bg-green-100 text-green-700"
                      : "bg-gray-100 text-gray-600"
                  }`}
                >
                  {user.status}
                </span>
              </div>
            ))}
            {users.length === 0 && (
              <p className="text-gray-500">No users found</p>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

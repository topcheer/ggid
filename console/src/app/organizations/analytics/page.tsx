"use client";

import { useEffect, useState, useCallback } from "react";
import { useApi } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import { Building2, TrendingUp, Users, Shield, Layers, RefreshCw } from "lucide-react";
import {
  BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
  PieChart, Pie, Cell, Legend, AreaChart, Area,
} from "@/components/charts/lazy-charts";

interface OrgStats {
  total_orgs: number;
  total_members: number;
  total_departments: number;
  total_teams: number;
  members_by_org: { org_name: string; count: number }[];
  role_distribution: { role: string; count: number }[];
  growth_trend: { date: string; members: number }[];
}

const PIE_COLORS = ["#6366f1", "#10b981", "#f59e0b", "#ef4444", "#8b5cf6", "#06b6d4", "#ec4899", "#84cc16"];

export default function OrgAnalyticsPage() {
  const { apiFetch, TENANT_ID } = useApi();
  const [stats, setStats] = useState<OrgStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadStats = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiFetch<OrgStats>(`/api/v1/orgs/tree?tenant_id=${TENANT_ID}`).catch(() => null);
      // Transform tree data into analytics shape
      if (data) {
        setStats({
          total_orgs: data.total_orgs || 0,
          total_members: data.total_members || 0,
          total_departments: data.total_departments || 0,
          total_teams: data.total_teams || 0,
          members_by_org: data.members_by_org || [],
          role_distribution: data.role_distribution || [],
          growth_trend: data.growth_trend || [],
        });
      } else {
        // Fallback: build from org list
        const orgData = await apiFetch<{ organizations?: { id: string; name: string }[] }>(
          `/api/v1/orgs?tenant_id=${TENANT_ID}`,
        ).catch(() => ({ organizations: [] }));
        const orgs = orgData.organizations || [];
        const membersByOrg = await Promise.all(
          orgs.slice(0, 10).map(async (org) => {
            const m = await apiFetch<{ members?: unknown[] }>(
              `/api/v1/orgs/${org.id}/members?tenant_id=${TENANT_ID}`,
            ).catch(() => ({ members: [] }));
            return { org_name: org.name, count: m.members?.length || 0 };
          }),
        );
        setStats({
          total_orgs: orgs.length,
          total_members: membersByOrg.reduce((sum, o) => sum + o.count, 0),
          total_departments: 0,
          total_teams: 0,
          members_by_org: membersByOrg,
          role_distribution: [],
          growth_trend: [],
        });
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load analytics");
    } finally {
      setLoading(false);
    }
  }, [apiFetch, TENANT_ID]);

  useEffect(() => { loadStats(); }, [loadStats]);

  if (loading) return <p className="text-gray-500">Loading...</p>;
  if (error) return (
    <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">{error}</div>
  );
  if (!stats) return <p className="text-gray-500">No data available</p>;

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Organization Analytics</h1>
          <p className="text-sm text-gray-500">Member trends, role distribution, and department insights</p>
        </div>
        <button
          onClick={loadStats}
          className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm hover:bg-gray-50"
        >
          <RefreshCw className="h-4 w-4" /> Refresh
        </button>
      </div>

      {/* Stat cards */}
      <div className="mb-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard icon={Building2} label="Organizations" value={stats.total_orgs} color="indigo" />
        <StatCard icon={Users} label="Total Members" value={stats.total_members} color="green" />
        <StatCard icon={Layers} label="Departments" value={stats.total_departments} color="purple" />
        <StatCard icon={Shield} label="Teams" value={stats.total_teams} color="amber" />
      </div>

      {/* Growth trend */}
      {stats.growth_trend.length > 0 && (
        <div className="mb-6 rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
          <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold">
            <TrendingUp className="h-4 w-4 text-brand-600" />
            Member Growth Trend
          </h3>
          <ResponsiveContainer width="100%" height={250}>
            <AreaChart data={stats.growth_trend}>
              <defs>
                <linearGradient id="colorGrowth" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#10b981" stopOpacity={0.8} />
                  <stop offset="95%" stopColor="#10b981" stopOpacity={0} />
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" stroke="#f3f4f6" />
              <XAxis dataKey="date" tick={{ fontSize: 11 }} />
              <YAxis tick={{ fontSize: 11 }} allowDecimals={false} />
              <Tooltip contentStyle={{ fontSize: 12, borderRadius: 8 }} />
              <Area type="monotone" dataKey="members" stroke="#10b981" strokeWidth={2} fill="url(#colorGrowth)" />
            </AreaChart>
          </ResponsiveContainer>
        </div>
      )}

      <div className="grid gap-6 lg:grid-cols-2">
      {/* Members by Org */}
        {stats.members_by_org.length > 0 && (
          <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
            <h3 className="mb-4 text-sm font-semibold">Members by Organization</h3>
            <ResponsiveContainer width="100%" height={250}>
              <BarChart data={stats.members_by_org} layout="vertical">
                <CartesianGrid strokeDasharray="3 3" stroke="#f3f4f6" horizontal={false} />
                <XAxis type="number" tick={{ fontSize: 11 }} allowDecimals={false} />
                <YAxis type="category" dataKey="org_name" tick={{ fontSize: 11 }} width={120} />
                <Tooltip contentStyle={{ fontSize: 12, borderRadius: 8 }} />
                <Bar dataKey="count" fill="#6366f1" radius={[0, 4, 4, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        )}

      {/* Role Distribution */}
        {stats.role_distribution.length > 0 && (
          <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
            <h3 className="mb-4 text-sm font-semibold">Role Distribution</h3>
            <ResponsiveContainer width="100%" height={250}>
              <PieChart>
                <Pie
                  data={stats.role_distribution}
                  cx="50%" cy="50%" outerRadius={80}
                  dataKey="count"
                  label={(entry: { role?: string; percent?: number }) => {
                    const pct = entry.percent ? (entry.percent * 100).toFixed(0) : "0";
                    return `${entry.role || ""} ${pct}%`;
                  }}
                  labelLine={false}
                >
                  {stats.role_distribution.map((_, i) => (
                    <Cell key={i} fill={PIE_COLORS[i % PIE_COLORS.length]} />
                  ))}
                </Pie>
                <Tooltip contentStyle={{ fontSize: 12, borderRadius: 8 }} />
              </PieChart>
            </ResponsiveContainer>
          </div>
        )}
      </div>

      {stats.members_by_org.length === 0 && stats.role_distribution.length === 0 && (
        <div className="rounded-xl border border-gray-200 bg-white p-12 text-center shadow-sm">
          <Building2 className="mx-auto mb-4 h-12 w-12 text-gray-300" />
          <p className="text-gray-500">No analytics data available yet</p>
          <p className="mt-1 text-xs text-gray-400">
            Data will populate as organizations and members are created.
          </p>
        </div>
      )}
    </div>
  );
}

function StatCard({
  icon: Icon, label, value, color,
}: {
  icon: React.ElementType; label: string; value: number;
  color: "indigo" | "green" | "purple" | "amber";
}) {
  const colorMap = {
    indigo: { bg: "bg-indigo-100", text: "text-indigo-600" },
    green: { bg: "bg-green-100", text: "text-green-600" },
    purple: { bg: "bg-purple-100", text: "text-purple-600" },
    amber: { bg: "bg-amber-100", text: "text-amber-600" },
  };
  const c = colorMap[color];
  return (
    <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
      <div className="flex items-center gap-3">
        <div className={`flex h-10 w-10 items-center justify-center rounded-lg ${c.bg}`}>
          <Icon className={`h-5 w-5 ${c.text}`} />
        </div>
        <div>
          <p className="text-2xl font-bold">{value.toLocaleString()}</p>
          <p className="text-xs text-gray-500">{label}</p>
        </div>
      </div>
    </div>
  );
}

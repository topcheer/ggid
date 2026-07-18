"use client";

import { usePolicyTimeBasedAccess } from "@ggid/sdk-react";
import { Clock, Calendar, Globe, AlertTriangle, CheckCircle } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function PolicyTimeBasedAccessPage() {
  const t = useTranslations();

  const { data, loading, error, refresh } = usePolicyTimeBasedAccess();

  if (loading) return <div className="p-8 text-gray-400">Loading time-based access...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const days = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"];

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Time-Based Access Control</h1>
          <p className="text-sm text-gray-400 mt-1">Restrict access to specific time windows, days, and timezones</p>
        </div>
        <button
          onClick={refresh}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          Refresh
        </button>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-blue-400">
            <Clock className="w-4 h-4" />
            <span className="text-xs text-gray-400">Active Rules</span>
          </div>
          <p className="text-2xl font-bold">{data?.time_window_rules?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-purple-400">
            <Calendar className="w-4 h-4" />
            <span className="text-xs text-gray-400">Grace Period</span>
          </div>
          <p className="text-2xl font-bold">{data?.grace_period_minutes ?? 0}m</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-yellow-400">
            <Globe className="w-4 h-4" />
            <span className="text-xs text-gray-400">Timezones</span>
          </div>
          <p className="text-2xl font-bold">{data?.per_role_restrictions?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-red-400">
            <AlertTriangle className="w-4 h-4" />
            <span className="text-xs text-gray-400">Violations (24h)</span>
          </div>
          <p className="text-2xl font-bold text-red-400">{data?.violations_24h ?? 0}</p>
        </div>
      </div>

      {/* Time Window Rules Table */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold mb-4">Time Window Rules</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th scope="col" className="text-left py-2 pr-3">Policy</th>
                <th scope="col" className="text-left py-2 pr-3">Allowed Days</th>
                <th scope="col" className="text-left py-2 pr-3">Start</th>
                <th scope="col" className="text-left py-2 pr-3">End</th>
                <th scope="col" className="text-left py-2 pr-3">Timezone</th>
              </tr>
            </thead>
            <tbody>
              {(data?.time_window_rules ?? []).map((r: any) => (
                <tr key={r.policy} className="border-b border-gray-800">
                  <td className="py-3 pr-3 font-medium">{r.policy}</td>
                  <td className="py-3 pr-3">
                    <div className="flex gap-1">
                      {days.map((d: any) => (
                        <span
                          key={d}
                          className={"text-xs px-1.5 py-0.5 rounded " + (
                            r.allowed_days.includes(d) ? "bg-green-900 text-green-300" : "bg-gray-700 text-gray-500"
                          )}
                        >
                          {d[0]}
                        </span>
                      ))}
                    </div>
                  </td>
                  <td className="py-3 pr-3 text-gray-300 font-mono">{r.start_time}</td>
                  <td className="py-3 pr-3 text-gray-300 font-mono">{r.end_time}</td>
                  <td className="py-3 pr-3 text-gray-400 text-xs">{r.timezone}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* UTC Timeline Visual */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
            <Globe className="w-5 h-5 text-blue-400" />
            UTC Timeline (24h)
          </h2>
          <div className="space-y-2">
            {(data?.time_window_rules ?? []).slice(0, 4).map((r: any) => {
              const startHour = parseInt(r.start_time.split(":")[0]);
              const endHour = parseInt(r.end_time.split(":")[0]);
              return (
                <div key={r.policy}>
                  <p className="text-xs text-gray-400 mb-1">{r.policy}</p>
                  <div className="flex h-6 bg-gray-800 rounded">
                    {Array.from({ length: 24 }, (_, h) => (
                      <div
                        key={h}
                        className={"flex-1 border-r border-gray-700 last:border-r-0 " + (
                          h >= startHour && h < endHour ? "bg-green-600" : ""
                        )}
                        title={`${h}:00`}
                      />
                    ))}
                  </div>
                  <div className="flex justify-between text-xs text-gray-600 mt-0.5">
                    <span>00:00</span>
                    <span>06:00</span>
                    <span>12:00</span>
                    <span>18:00</span>
                    <span>24:00</span>
                  </div>
                </div>
              );
            })}
          </div>
        </div>

        {/* Holiday Calendar + Per-Role Restrictions */}
        <div className="space-y-6">
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold mb-4">Holiday Calendar Integration</h2>
            <div className="space-y-2">
              {(data?.holiday_calendar ?? []).map((h: any, i: number) => (
                <div key={i} className="flex items-center justify-between bg-gray-800 rounded-lg p-2">
                  <span className="text-sm">{h.name}</span>
                  <span className="text-xs text-gray-400">{h.date} - {h.access}</span>
                </div>
              ))}
            </div>
          </div>

          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold mb-4">Per-Role Restrictions</h2>
            <div className="space-y-2">
              {(data?.per_role_restrictions ?? []).map((r: any) => (
                <div key={r.role} className="flex items-center justify-between bg-gray-800 rounded-lg p-2">
                  <span className="text-sm font-medium">{r.role}</span>
                  <span className="text-xs text-gray-400">
                    {r.allowed_days.length === 5 ? "Weekdays" : "Custom"} {r.start_time}-{r.end_time}
                  </span>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

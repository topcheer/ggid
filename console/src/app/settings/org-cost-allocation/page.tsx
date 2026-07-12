"use client";

import { useOrgCostAllocation } from "@ggid/sdk-react";
import { DollarSign, TrendingUp, AlertTriangle, Download, FileText, Building } from "lucide-react";

export default function OrgCostAllocationPage() {
  const { data, loading, error, refresh } = useOrgCostAllocation();

  if (loading) return <div className="p-8 text-gray-400">Loading cost allocation...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const maxCost = Math.max(...(data?.monthly_cost_breakdown ?? [{ amount: 1 }]).map((d) => d.amount), 1);

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Cost Allocation</h1>
          <p className="text-sm text-gray-400 mt-1">Department chargeback rules and monthly cost breakdown</p>
        </div>
        <div className="flex items-center gap-2">
          <button className="flex items-center gap-1 px-3 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg text-sm font-medium transition">
            <Download className="w-4 h-4" />
            CSV
          </button>
          <button className="flex items-center gap-1 px-3 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg text-sm font-medium transition">
            <FileText className="w-4 h-4" />
            PDF
          </button>
          <button
            onClick={refresh}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
          >
            Refresh
          </button>
        </div>
      </div>

      {/* Allocation Rules Table */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
          <Building className="w-5 h-5 text-blue-400" />
          Allocation Rules
        </h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th className="text-left py-2 pr-4">Department</th>
                <th className="text-left py-2 pr-4">Cost Center</th>
                <th className="text-left py-2 pr-4">Allocation %</th>
                <th className="text-left py-2 pr-4">Chargeback Model</th>
              </tr>
            </thead>
            <tbody>
              {(data?.allocation_rules ?? []).map((r) => (
                <tr key={r.department} className="border-b border-gray-800">
                  <td className="py-3 pr-4 font-medium">{r.department}</td>
                  <td className="py-3 pr-4 text-gray-300 font-mono">{r.cost_center}</td>
                  <td className="py-3 pr-4">
                    <div className="flex items-center gap-2">
                      <div className="w-20 bg-gray-700 rounded-full h-1.5">
                        <div className="bg-blue-500 rounded-full h-1.5" style={{ width: `${r.allocation_pct}%` }} />
                      </div>
                      <span className="text-sm font-medium">{r.allocation_pct}%</span>
                    </div>
                  </td>
                  <td className="py-3 pr-4">
                    <span className="text-xs px-2 py-0.5 rounded bg-gray-700 text-gray-300 capitalize">{r.chargeback_model.replace(/_/g, " ")}</span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Monthly Cost Breakdown */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
            <TrendingUp className="w-5 h-5 text-blue-400" />
            Monthly Cost Breakdown
          </h2>
          <div className="space-y-2">
            {(data?.monthly_cost_breakdown ?? []).map((d) => (
              <div key={d.department} className="flex items-center gap-3">
                <span className="text-sm w-24 text-gray-300">{d.department}</span>
                <div className="flex-1 bg-gray-700 rounded-full h-5 relative">
                  <div
                    className="bg-blue-500 rounded-full h-5 flex items-center justify-end pr-2"
                    style={{ width: `${(d.amount / maxCost) * 100}%` }}
                  >
                    <span className="text-xs font-medium text-white">${d.amount.toLocaleString()}</span>
                  </div>
                </div>
              </div>
            ))}
          </div>
          <div className="mt-4 pt-4 border-t border-gray-800 flex items-center justify-between">
            <span className="text-sm text-gray-400">Total Monthly Cost</span>
            <span className="text-lg font-bold">${(data?.monthly_cost_breakdown ?? []).reduce((a, d) => a + d.amount, 0).toLocaleString()}</span>
          </div>
        </div>

        <div className="space-y-6">
          {/* Chargeback Report Preview */}
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
              <DollarSign className="w-5 h-5 text-green-400" />
              Chargeback Report Preview
            </h2>
            <div className="bg-gray-800 rounded-lg p-3 overflow-x-auto">
              <pre className="text-xs font-mono text-gray-300 whitespace-pre-wrap">{JSON.stringify(data?.chargeback_report_preview ?? {}, null, 2)}</pre>
            </div>
          </div>

          {/* Over Budget Alerts */}
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
              <AlertTriangle className="w-5 h-5 text-yellow-400" />
              Over Budget Alerts
            </h2>
            <div className="space-y-2">
              {(data?.over_budget_alerts ?? []).map((alert, i) => (
                <div key={i} className="bg-gray-800 rounded-lg p-3">
                  <div className="flex items-center justify-between mb-1">
                    <p className="text-sm font-medium">{alert.department}</p>
                    <span className="text-xs font-bold text-red-400">{alert.pct_over}% over</span>
                  </div>
                  <p className="text-xs text-gray-400">Budget: ${alert.budget.toLocaleString()} - Actual: ${alert.actual.toLocaleString()}</p>
                </div>
              ))}
              {(data?.over_budget_alerts ?? []).length === 0 && (
                <p className="text-sm text-gray-500 text-center py-4">All departments within budget.</p>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

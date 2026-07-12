"use client";
import { useEffect, useState } from "react";
import { useTokenBindingComparison, TokenBindingComparison, ComparisonRow, RecommendationEntry, BenchmarkResult, PerClientMethod } from "@ggid/sdk-react";

export default function TokenBindingComparisonPage() {
  const { config, loading, error, fetchConfig } = useTokenBindingComparison();
  const [form, setForm] = useState<TokenBindingComparison | null>(null);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">Token Binding Comparison</h1>
      <p className="text-gray-600">Compare token binding methods across security, performance, and deployment dimensions.</p>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Comparison Matrix</h2>
        <table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Method</th><th>Security</th><th>Deployment</th><th>Performance</th><th>Fallback</th></tr></thead><tbody>
          {form.comparison_table.map((r: ComparisonRow, i: number) => (
            <tr key={i} className="border-b"><td className="py-2 font-medium">{r.method}</td><td>{"*".repeat(r.security)}</td><td>{"*".repeat(r.deployment)}</td><td>{"*".repeat(r.performance)}</td><td>{"*".repeat(r.fallback)}</td></tr>
          ))}
        </tbody></table>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Recommendation Matrix</h2>
        <table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Use Case</th><th>Recommended</th><th>Rationale</th></tr></thead><tbody>
          {form.recommendation_matrix.map((r: RecommendationEntry, i: number) => (
            <tr key={i} className="border-b"><td className="py-2 font-medium">{r.use_case}</td><td><span className="px-2 py-1 bg-blue-100 text-blue-700 rounded text-xs">{r.recommended_method}</span></td><td className="text-xs">{r.rationale}</td></tr>
          ))}
        </tbody></table>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Benchmark Results</h2>
        <table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Method</th><th>Latency (ms)</th><th>CPU Overhead (%)</th></tr></thead><tbody>
          {form.benchmark_results.map((b: BenchmarkResult, i: number) => (
            <tr key={i} className="border-b"><td className="py-2 font-medium">{b.method}</td><td>{b.latency_ms}</td><td>{b.cpu_overhead_pct}</td></tr>
          ))}
        </tbody></table>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Per-Client Current Method</h2>
        <div className="space-y-2">
          {form.per_client_current_method.map((c: PerClientMethod, i: number) => (
            <div key={i} className="flex items-center justify-between border-b py-2">
              <div><span className="font-medium">{c.client_name}</span><span className="ml-2 text-xs text-gray-400">{c.client_id}</span></div>
              <span className="text-sm">{c.method}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

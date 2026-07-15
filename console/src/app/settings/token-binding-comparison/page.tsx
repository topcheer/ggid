"use client";
import { useEffect, useState } from "react";
import { useTranslations } from "@/lib/i18n";
import { useTokenBindingComparison, TokenBindingComparison, ComparisonRow, RecommendationEntry, BenchmarkResult, PerClientMethod } from "@ggid/sdk-react";

export default function TokenBindingComparisonPage() {
  const { config, loading, error, fetchConfig } = useTokenBindingComparison();
  const [form, setForm] = useState<TokenBindingComparison | null>(null);
  const t = useTranslations();

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  if (loading && !form) return <div className="p-8">{t("tokenBindingCompare.loading")}</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">{t("tokenBindingCompare.noData")}</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">{t("tokenBindingCompare.title")}</h1>
      <p className="text-gray-600">{t("tokenBindingCompare.subtitle")}</p>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">{t("tokenBindingCompare.comparisonMatrix")}</h2>
        <table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">{t("tokenBindingCompare.method")}</th><th>{t("tokenBindingCompare.security")}</th><th>{t("tokenBindingCompare.deployment")}</th><th>{t("tokenBindingCompare.performance")}</th><th>{t("tokenBindingCompare.fallback")}</th></tr></thead><tbody>
          {form.comparison_table.map((r: ComparisonRow, i: number) => (
            <tr key={i} className="border-b"><td className="py-2 font-medium">{r.method}</td><td>{"*".repeat(r.security)}</td><td>{"*".repeat(r.deployment)}</td><td>{"*".repeat(r.performance)}</td><td>{"*".repeat(r.fallback)}</td></tr>
          ))}
        </tbody></table>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">{t("tokenBindingCompare.recommendation")}</h2>
        <table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">{t("tokenBindingCompare.useCase")}</th><th>{t("tokenBindingCompare.recommended")}</th><th>{t("tokenBindingCompare.rationale")}</th></tr></thead><tbody>
          {form.recommendation_matrix.map((r: RecommendationEntry, i: number) => (
            <tr key={i} className="border-b"><td className="py-2 font-medium">{r.use_case}</td><td><span className="px-2 py-1 bg-blue-100 text-blue-700 rounded text-xs">{r.recommended_method}</span></td><td className="text-xs">{r.rationale}</td></tr>
          ))}
        </tbody></table>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">{t("tokenBindingCompare.benchmark")}</h2>
        <table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">{t("tokenBindingCompare.method")}</th><th>{t("tokenBindingCompare.latency")}</th><th>{t("tokenBindingCompare.cpuOverhead")}</th></tr></thead><tbody>
          {form.benchmark_results.map((b: BenchmarkResult, i: number) => (
            <tr key={i} className="border-b"><td className="py-2 font-medium">{b.method}</td><td>{b.latency_ms}</td><td>{b.cpu_overhead_pct}</td></tr>
          ))}
        </tbody></table>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">{t("tokenBindingCompare.perClient")}</h2>
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

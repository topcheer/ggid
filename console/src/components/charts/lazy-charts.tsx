"use client";

import dynamic from "next/dynamic";

/**
 * Lazy-loaded chart components from recharts.
 * Recharts is ~400KB minified — code-splitting keeps it out of the
 * initial bundle so pages that don't use charts load faster.
 *
 * Usage:
 *   import { LazyAreaChart, LazyBarChart, LazyPieChart } from "@/components/charts/lazy-charts";
 */

const LoadingFallback = () => (
  <div className="flex h-64 items-center justify-center">
    <div className="h-8 w-8 animate-spin rounded-full border-2 border-gray-300 border-t-brand-600" />
  </div>
);

// Dynamically import recharts with ssr disabled
const RechartAreaChart = dynamic(
  () => import("recharts").then((m) => m.AreaChart),
  { ssr: false, loading: () => <LoadingFallback /> }
);

const RechartBarChart = dynamic(
  () => import("recharts").then((m) => m.BarChart),
  { ssr: false, loading: () => <LoadingFallback /> }
);

const RechartPieChart = dynamic(
  () => import("recharts").then((m) => m.PieChart),
  { ssr: false, loading: () => <LoadingFallback /> }
);

// Re-export chart sub-components that are lightweight (no heavy SVG libs)
export {
  RechartAreaChart as AreaChart,
  RechartBarChart as BarChart,
  RechartPieChart as PieChart,
};

// These are small components, safe to import eagerly
export {
  Area,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Pie,
  Cell,
  Legend,
} from "recharts";

"use client";

import dynamic from "next/dynamic";

/**
 * Lazy-loaded chart components from recharts.
 *
 * recharts (~400KB) is code-split via next/dynamic with ssr:false.
 * Pages without charts never download the recharts chunk.
 *
 * Usage:
 *   import { AreaChart, Area, BarChart, ... } from "@/components/charts/lazy-charts";
 *
 * Note: Chart container components (AreaChart, BarChart, PieChart) are
 * dynamically imported. Sub-components (Area, Bar, XAxis, etc.) are
 * tree-shaken static imports — they are tiny and shared across chart types.
 */

const LoadingFallback = () => (
  <div className="flex h-64 items-center justify-center">
    <div className="h-8 w-8 animate-spin rounded-full border-2 border-gray-300 border-t-brand-600" />
  </div>
);

// Chart containers — dynamically imported (these pull in SVG rendering)
const AreaChart = dynamic(
  () => import("recharts").then((m) => m.AreaChart),
  { ssr: false, loading: () => <LoadingFallback /> }
);
const BarChart = dynamic(
  () => import("recharts").then((m) => m.BarChart),
  { ssr: false, loading: () => <LoadingFallback /> }
);
const PieChart = dynamic(
  () => import("recharts").then((m) => m.PieChart),
  { ssr: false, loading: () => <LoadingFallback /> }
);

// Lightweight sub-components — tree-shaken, no heavy SVG libs
export {
  AreaChart,
  BarChart,
  PieChart,
};

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

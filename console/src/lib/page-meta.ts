import type { Metadata } from "next";

/**
 * Helper to create per-page metadata with consistent title template.
 * Usage: export const metadata = pageMeta("Users");
 * → Title becomes "Users | GGID Console"
 */
export function pageMeta(title: string, description?: string): Metadata {
  return {
    title,
    description: description || `${title} — GGID Identity & Access Management`,
  };
}

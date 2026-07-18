import { useEffect } from "react";

/**
 * Set document.title for client component pages.
 * Usage: usePageTitle("Users");
 * → Title becomes "Users | GGID Console"
 */
export function usePageTitle(title: string) {
  useEffect(() => {
    document.title = `${title} | GGID Console`;
    return () => { document.title = "GGID Console"; };
  }, [title]);
}

"use client";

import { useEffect, useState, createContext, useContext } from "react";

export type Theme = "light" | "dark";
export type ThemeMode = "light" | "dark" | "system";

interface ThemeContextValue {
  mode: ThemeMode;
  theme: Theme;
  setMode: (mode: ThemeMode) => void;
  toggle: () => void;
}

const ThemeContext = createContext<ThemeContextValue>({
  mode: "system",
  theme: "light",
  setMode: () => {},
  toggle: () => {},
});

export function useTheme() {
  return useContext(ThemeContext);
}

function getSystemTheme(): Theme {
  if (typeof window === "undefined") return "light";
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

function applyDOMTheme(theme: Theme) {
  const root = document.documentElement;
  if (theme === "dark") {
    root.classList.add("dark");
  } else {
    root.classList.remove("dark");
  }
}

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [mode, setModeState] = useState<ThemeMode>("system");
  const [theme, setTheme] = useState<Theme>("light");
  const [mounted, setMounted] = useState(false);

  // Read persisted preference on mount
  useEffect(() => {
    const saved = localStorage.getItem("darkMode") as ThemeMode | null;
    if (saved === "dark" || saved === "light" || saved === "system") {
      setModeState(saved);
    } else {
      setModeState("system");
    }
    setMounted(true);
  }, []);

  // Apply theme whenever mode changes, listen to system changes if needed
  useEffect(() => {
    if (!mounted) return;

    const resolved = mode === "system" ? getSystemTheme() : mode;
    setTheme(resolved);
    applyDOMTheme(resolved);
    localStorage.setItem("darkMode", mode);

    if (mode === "system") {
      const mq = window.matchMedia("(prefers-color-scheme: dark)");
      const handler = () => {
        const sys = getSystemTheme();
        setTheme(sys);
        applyDOMTheme(sys);
      };
      mq.addEventListener("change", handler);
      return () => mq.removeEventListener("change", handler);
    }
  }, [mode, mounted]);

  const setMode = (m: ThemeMode) => setModeState(m);

  // Cycle: light → dark → system → light
  const toggle = () =>
    setModeState((prev) =>
      prev === "light" ? "dark" : prev === "dark" ? "system" : "light"
    );

  return (
    <ThemeContext.Provider value={{ mode, theme, setMode, toggle }}>
      {children}
    </ThemeContext.Provider>
  );
}

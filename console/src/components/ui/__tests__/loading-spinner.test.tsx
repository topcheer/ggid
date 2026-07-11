import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { LoadingSpinner, LoadingPage } from "../loading-spinner";

describe("LoadingSpinner", () => {
  it("renders with default props", () => {
    render(<LoadingSpinner />);
    expect(document.querySelector(".animate-spin")).toBeTruthy();
  });

  it("renders label when provided", () => {
    render(<LoadingSpinner label="Loading data..." />);
    expect(screen.getByText("Loading data...")).toBeTruthy();
  });

  it("applies size classes", () => {
    const { container: sm } = render(<LoadingSpinner size="sm" />);
    const spinner = sm.querySelector(".animate-spin");
    expect(spinner?.className).toContain("h-4");
    expect(spinner?.className).toContain("w-4");
  });
});

describe("LoadingPage", () => {
  it("renders with default label", () => {
    render(<LoadingPage />);
    expect(screen.getByText("Loading...")).toBeTruthy();
  });

  it("renders with custom label", () => {
    render(<LoadingPage label="Fetching users..." />);
    expect(screen.getByText("Fetching users...")).toBeTruthy();
  });
});

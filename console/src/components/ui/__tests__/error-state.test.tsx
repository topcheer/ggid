import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ErrorState, ErrorBanner } from "../error-state";

describe("ErrorState", () => {
  it("renders default title and message", () => {
    render(<ErrorState />);
    expect(screen.getByText("Something went wrong")).toBeTruthy();
    expect(screen.getByText(/An unexpected error occurred/)).toBeTruthy();
  });

  it("renders custom title and message", () => {
    render(<ErrorState title="API Error" message="Failed to load users" />);
    expect(screen.getByText("API Error")).toBeTruthy();
    expect(screen.getByText("Failed to load users")).toBeTruthy();
  });

  it("calls onRetry when button clicked", () => {
    const onRetry = vi.fn();
    render(<ErrorState onRetry={onRetry} />);
    fireEvent.click(screen.getByText("Try again"));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it("does not render retry button without onRetry", () => {
    render(<ErrorState />);
    expect(screen.queryByText("Try again")).toBeNull();
  });
});

describe("ErrorBanner", () => {
  it("renders message", () => {
    render(<ErrorBanner message="Connection failed" />);
    expect(screen.getByText("Connection failed")).toBeTruthy();
  });

  it("calls onDismiss when close button clicked", () => {
    const onDismiss = vi.fn();
    render(<ErrorBanner message="Error" onDismiss={onDismiss} />);
    fireEvent.click(screen.getByText("×"));
    expect(onDismiss).toHaveBeenCalledTimes(1);
  });
});

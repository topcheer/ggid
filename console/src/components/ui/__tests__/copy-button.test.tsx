import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { CopyButton } from "../copy-button";

// Mock clipboard API
const writeTextMock = vi.fn();
Object.assign(navigator, {
  clipboard: { writeText: writeTextMock },
});

describe("CopyButton", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers();
    writeTextMock.mockResolvedValue(undefined);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("renders icon variant by default", () => {
    render(<CopyButton value="test-key" />);
    const button = screen.getByTitle("Copy to clipboard");
    expect(button).toBeInTheDocument();
  });

  it("calls clipboard.writeText on click", () => {
    render(<CopyButton value="my-secret-key" />);
    fireEvent.click(screen.getByTitle("Copy to clipboard"));
    expect(writeTextMock).toHaveBeenCalledWith("my-secret-key");
  });

  it("shows label in button variant", () => {
    render(<CopyButton value="key" label="Copy API Key" variant="button" />);
    expect(screen.getByText("Copy API Key")).toBeInTheDocument();
  });

  it("shows copied state after click", () => {
    render(<CopyButton value="key" label="Copy" variant="button" />);
    fireEvent.click(screen.getByText("Copy"));
    expect(screen.getByText("Copied!")).toBeInTheDocument();
  });

  it("reverts to normal state after 2s", () => {
    render(<CopyButton value="key" label="Copy" variant="button" />);
    fireEvent.click(screen.getByText("Copy"));
    expect(screen.getByText("Copied!")).toBeInTheDocument();
    vi.advanceTimersByTime(2100);
    expect(screen.getByText("Copy")).toBeInTheDocument();
  });

  it("masks value in ghost variant", () => {
    render(
      <CopyButton
        value="sk-very-long-secret-key-1234567890"
        variant="ghost"
        masked
      />
    );
    const text = screen.getByTitle("Copy to clipboard").textContent;
    expect(text).toContain("••••");
    expect(text).not.toContain("very-long-secret");
  });

  it("uses custom title", () => {
    render(<CopyButton value="token" title="Copy JWT token" />);
    expect(screen.getByTitle("Copy JWT token")).toBeInTheDocument();
  });
});

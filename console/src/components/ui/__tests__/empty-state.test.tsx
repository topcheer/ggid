import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { EmptyState } from "../empty-state";
import { Inbox } from "lucide-react";

describe("EmptyState", () => {
  it("renders default title and description", () => {
    render(<EmptyState />);
    expect(screen.getByText("No data")).toBeTruthy();
    expect(screen.getByText(/nothing here yet/i)).toBeTruthy();
  });

  it("renders custom title and description", () => {
    render(<EmptyState title="No users found" description="Try adjusting your filters" />);
    expect(screen.getByText("No users found")).toBeTruthy();
    expect(screen.getByText("Try adjusting your filters")).toBeTruthy();
  });

  it("renders action button when provided", () => {
    const onClick = vi.fn();
    render(<EmptyState action={{ label: "Add User", onClick }} />);
    fireEvent.click(screen.getByText("Add User"));
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it("does not render action when not provided", () => {
    render(<EmptyState />);
    expect(screen.queryByRole("button")).toBeNull();
  });

  it("renders custom icon", () => {
    render(<EmptyState icon={Inbox} title="Inbox empty" />);
    expect(screen.getByText("Inbox empty")).toBeTruthy();
  });
});

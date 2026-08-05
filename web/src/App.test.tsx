import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { renderApp } from "./test/renderApp";

describe("App shell", () => {
  it("renders the honest overview and complete primary navigation", () => {
    renderApp();

    expect(
      screen.getByRole("heading", { name: "Your VPS at a glance" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("navigation", { name: "Primary navigation" }),
    ).toBeInTheDocument();
    for (const label of [
      "Overview",
      "CPU",
      "Memory",
      "Filesystems",
      "Docker",
      "Events",
      "Audit",
      "Backups",
    ]) {
      expect(screen.getByRole("link", { name: label })).toBeInTheDocument();
    }
    expect(screen.getAllByText("Unknown").length).toBeGreaterThan(1);
    expect(screen.getAllByText("Tailnet access only")).toHaveLength(2);
  });

  it("supports keyboard navigation between routes", async () => {
    const user = userEvent.setup();
    renderApp();
    const cpuLink = screen.getByRole("link", { name: "CPU" });

    cpuLink.focus();
    await user.keyboard("{Enter}");

    expect(
      await screen.findByRole("heading", { name: "CPU" }),
    ).toBeInTheDocument();
    expect(cpuLink).toHaveAttribute("aria-current", "page");
  });

  it("keeps navigation usable at an Android-sized viewport", () => {
    Object.defineProperty(window, "innerWidth", {
      configurable: true,
      value: 390,
    });
    window.dispatchEvent(new Event("resize"));
    renderApp("/docker");

    expect(screen.getByRole("navigation")).toBeVisible();
    expect(screen.getByRole("link", { name: "Docker" })).toHaveAttribute(
      "aria-current",
      "page",
    );
    expect(screen.getByRole("heading", { name: "Docker" })).toBeInTheDocument();
  });

  it("renders a localized not-found route", () => {
    renderApp("/does-not-exist");

    expect(
      screen.getByRole("heading", { name: "Page not found" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: "Return to overview" }),
    ).toBeInTheDocument();
  });
});

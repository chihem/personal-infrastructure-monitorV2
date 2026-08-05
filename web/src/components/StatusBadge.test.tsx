import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import type { HealthState } from "../api/contracts";
import { renderWithI18n } from "../test/renderWithI18n";
import { StatusBadge } from "./StatusBadge";

describe("StatusBadge", () => {
  it("combines a distinct icon, written label, and state hook", () => {
    const states: HealthState[] = ["healthy", "warning", "critical", "unknown"];
    const expectedLabels = ["Healthy", "Warning", "Critical", "Unknown"];
    const icons = new Set<string>();

    const { container } = renderWithI18n(
      <div>
        {states.map((state) => (
          <StatusBadge key={state} state={state} />
        ))}
      </div>,
    );

    for (const label of expectedLabels) {
      expect(screen.getByText(label)).toBeInTheDocument();
    }
    for (const badge of container.querySelectorAll("[data-state]")) {
      const iconName = badge.querySelector("svg")?.getAttribute("class") ?? "";
      expect(badge.textContent?.trim()).not.toBe("");
      icons.add(iconName);
    }
    expect(icons.size).toBe(states.length);
  });
});

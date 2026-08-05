import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { renderWithI18n } from "../test/renderWithI18n";
import { StatePanel, type StatePanelVariant } from "./StatePanel";

describe("StatePanel", () => {
  it("renders every shared non-success state with visible text", () => {
    const variants: StatePanelVariant[] = [
      "loading",
      "empty",
      "stale",
      "unavailable",
      "error",
    ];

    const { container } = renderWithI18n(
      <div>
        {variants.map((variant) => (
          <StatePanel key={variant} variant={variant} />
        ))}
      </div>,
    );

    expect(screen.getByText("Loading")).toBeInTheDocument();
    expect(screen.getByText("No data yet")).toBeInTheDocument();
    expect(screen.getByText("Data is stale")).toBeInTheDocument();
    expect(screen.getByText("Unavailable")).toBeInTheDocument();
    expect(screen.getByText("Something went wrong")).toBeInTheDocument();
    expect(container.querySelectorAll("[data-state]")).toHaveLength(
      variants.length,
    );
  });
});

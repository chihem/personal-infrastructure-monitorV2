import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { App } from "./App";

describe("App", () => {
  it("renders the scaffold status", () => {
    render(<App />);

    expect(screen.getByText("Infrastructure Monitor")).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: "Application scaffold ready" }),
    ).toBeInTheDocument();
  });
});

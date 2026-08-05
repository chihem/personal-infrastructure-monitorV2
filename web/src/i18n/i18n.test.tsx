import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { renderApp } from "../test/renderApp";
import { detectInitialLanguage, i18n, LANGUAGE_STORAGE_KEY } from ".";
import { resources } from "./resources";

describe("localization", () => {
  it("detects supported browser languages and falls back to English", () => {
    expect(detectInitialLanguage(null, ["fr-CA", "en-US"])).toBe("fr");
    expect(detectInitialLanguage(null, ["de-DE", "es-ES"])).toBe("en");
    expect(detectInitialLanguage("en", ["fr-FR"])).toBe("en");
  });

  it("switches at runtime and persists only the selected language", async () => {
    const user = userEvent.setup();
    await i18n.changeLanguage("en");
    renderApp();

    await user.click(screen.getByRole("button", { name: "French" }));

    expect(
      await screen.findByRole("heading", {
        name: "Votre VPS en un coup d’œil",
      }),
    ).toBeInTheDocument();
    expect(localStorage).toHaveLength(1);
    expect(localStorage.getItem(LANGUAGE_STORAGE_KEY)).toBe("fr");
    expect(document.documentElement).toHaveAttribute("lang", "fr");
    expect(document.title).toBe("Moniteur d’infrastructure");
  });

  it("keeps English and French translation keys complete", () => {
    const englishKeys = flattenKeys(resources.en.translation);
    const frenchKeys = flattenKeys(resources.fr.translation);

    expect(frenchKeys).toEqual(englishKeys);
    for (const value of flattenValues(resources.en.translation)) {
      expect(value.trim()).not.toBe("");
    }
    for (const value of flattenValues(resources.fr.translation)) {
      expect(value.trim()).not.toBe("");
    }
  });
});

function flattenKeys(value: object, prefix = ""): string[] {
  return Object.entries(value)
    .flatMap(([key, child]) => {
      const fullKey = prefix === "" ? key : `${prefix}.${key}`;
      return typeof child === "string"
        ? [fullKey]
        : flattenKeys(child, fullKey);
    })
    .sort();
}

function flattenValues(value: object): string[] {
  return Object.values(value).flatMap((child) =>
    typeof child === "string" ? [child] : flattenValues(child),
  );
}

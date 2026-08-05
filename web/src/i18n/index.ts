import { createInstance } from "i18next";

import { resources } from "./resources";

export const LANGUAGE_STORAGE_KEY = "pim.language";
export const supportedLanguages = ["en", "fr"] as const;
export type SupportedLanguage = (typeof supportedLanguages)[number];

export function normalizeLanguage(
  value: string | null | undefined,
): SupportedLanguage | null {
  if (value === null || value === undefined) {
    return null;
  }
  const baseLanguage = value.trim().toLowerCase().split("-")[0];
  return baseLanguage === "en" || baseLanguage === "fr" ? baseLanguage : null;
}

export function detectInitialLanguage(
  storedLanguage: string | null = readStoredLanguage(),
  browserLanguages: readonly string[] = readBrowserLanguages(),
): SupportedLanguage {
  const stored = normalizeLanguage(storedLanguage);
  if (stored !== null) {
    return stored;
  }
  for (const browserLanguage of browserLanguages) {
    const supported = normalizeLanguage(browserLanguage);
    if (supported !== null) {
      return supported;
    }
  }
  return "en";
}

export const i18n = createInstance();

void i18n.init({
  resources,
  lng: detectInitialLanguage(),
  fallbackLng: "en",
  supportedLngs: [...supportedLanguages],
  load: "languageOnly",
  initAsync: false,
  interpolation: {
    escapeValue: false,
  },
});

applyDocumentLanguage(normalizeLanguage(i18n.language) ?? "en");
i18n.on("languageChanged", (language) => {
  applyDocumentLanguage(normalizeLanguage(language) ?? "en");
});

export async function setLanguage(language: SupportedLanguage): Promise<void> {
  writeStoredLanguage(language);
  await i18n.changeLanguage(language);
}

function readStoredLanguage(): string | null {
  try {
    return globalThis.localStorage?.getItem(LANGUAGE_STORAGE_KEY) ?? null;
  } catch {
    return null;
  }
}

function writeStoredLanguage(language: SupportedLanguage): void {
  try {
    globalThis.localStorage?.setItem(LANGUAGE_STORAGE_KEY, language);
  } catch {
    // A blocked storage API must not prevent runtime language switching.
  }
}

function readBrowserLanguages(): readonly string[] {
  if (typeof navigator === "undefined") {
    return [];
  }
  return navigator.languages.length > 0
    ? navigator.languages
    : [navigator.language];
}

function applyDocumentLanguage(language: SupportedLanguage): void {
  if (typeof document !== "undefined") {
    document.documentElement.lang = language;
    document.title = resources[language].translation.brand.name;
  }
}

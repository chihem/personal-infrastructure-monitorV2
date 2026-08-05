import type { SupportedLanguage } from ".";

export const DISPLAY_TIMEZONE = "Africa/Tunis";

export function formatDashboardTime(
  value: Date | string | number,
  language: SupportedLanguage,
): string {
  const date = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(date.getTime())) {
    throw new TypeError("dashboard timestamp is invalid");
  }

  return new Intl.DateTimeFormat(language === "fr" ? "fr-FR" : "en-GB", {
    dateStyle: "medium",
    timeStyle: "medium",
    timeZone: DISPLAY_TIMEZONE,
  }).format(date);
}

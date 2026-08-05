import { Languages } from "lucide-react";
import { useTranslation } from "react-i18next";

import {
  normalizeLanguage,
  setLanguage,
  type SupportedLanguage,
} from "../i18n";
import styles from "./LanguageSwitcher.module.css";

const choices: ReadonlyArray<{
  code: SupportedLanguage;
  labelKey: "language.english" | "language.french";
}> = [
  { code: "en", labelKey: "language.english" },
  { code: "fr", labelKey: "language.french" },
];

export function LanguageSwitcher() {
  const { i18n, t } = useTranslation();
  const activeLanguage = normalizeLanguage(i18n.resolvedLanguage) ?? "en";

  return (
    <div
      className={styles.switcher}
      role="group"
      aria-label={t("language.label")}
    >
      <Languages aria-hidden="true" size={18} strokeWidth={1.8} />
      {choices.map((choice) => (
        <button
          className={styles.choice}
          data-active={activeLanguage === choice.code}
          key={choice.code}
          lang={choice.code}
          type="button"
          aria-label={t(choice.labelKey)}
          aria-pressed={activeLanguage === choice.code}
          onClick={() => void setLanguage(choice.code)}
        >
          <span className={styles.fullLabel} aria-hidden="true">
            {t(choice.labelKey)}
          </span>
          <span className={styles.shortLabel} aria-hidden="true">
            {choice.code.toUpperCase()}
          </span>
        </button>
      ))}
    </div>
  );
}

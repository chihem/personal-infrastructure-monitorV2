import {
  CircleSlash2,
  ClockAlert,
  Inbox,
  LoaderCircle,
  OctagonX,
  type LucideIcon,
} from "lucide-react";
import { useId } from "react";
import { useTranslation } from "react-i18next";

import styles from "./StatePanel.module.css";

export type StatePanelVariant =
  "loading" | "empty" | "stale" | "unavailable" | "error";

const stateIcons: Record<StatePanelVariant, LucideIcon> = {
  loading: LoaderCircle,
  empty: Inbox,
  stale: ClockAlert,
  unavailable: CircleSlash2,
  error: OctagonX,
};

interface StatePanelProps {
  variant: StatePanelVariant;
  title?: string;
  message?: string;
}

export function StatePanel({ variant, title, message }: StatePanelProps) {
  const { t } = useTranslation();
  const titleID = useId();
  const Icon = stateIcons[variant];

  return (
    <section
      className={styles.panel}
      data-state={variant}
      aria-labelledby={titleID}
      aria-live={
        variant === "loading" || variant === "error" ? "polite" : "off"
      }
    >
      <span className={styles.icon}>
        <Icon
          aria-hidden="true"
          className={variant === "loading" ? styles.spinner : undefined}
          size={22}
          strokeWidth={1.8}
        />
      </span>
      <span>
        <strong id={titleID}>{title ?? t(`state.${variant}.title`)}</strong>
        <span className={styles.message}>
          {message ?? t(`state.${variant}.message`)}
        </span>
      </span>
    </section>
  );
}

import {
  CircleCheck,
  CircleHelp,
  OctagonX,
  TriangleAlert,
  type LucideIcon,
} from "lucide-react";
import { useTranslation } from "react-i18next";

import type { HealthState } from "../api/contracts";
import styles from "./StatusBadge.module.css";

const statusIcons: Record<HealthState, LucideIcon> = {
  healthy: CircleCheck,
  warning: TriangleAlert,
  critical: OctagonX,
  unknown: CircleHelp,
};

interface StatusBadgeProps {
  state: HealthState;
  label?: string;
}

export function StatusBadge({ state, label }: StatusBadgeProps) {
  const { t } = useTranslation();
  const Icon = statusIcons[state];
  const visibleLabel = label ?? t(`status.${state}`);

  return (
    <span className={styles.badge} data-state={state}>
      <Icon aria-hidden="true" size={16} strokeWidth={2.2} />
      <span>{visibleLabel}</span>
    </span>
  );
}

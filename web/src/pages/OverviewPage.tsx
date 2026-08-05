import {
  Boxes,
  Cpu,
  HardDrive,
  MemoryStick,
  MoveUpRight,
  type LucideIcon,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link } from "wouter";

import { StatePanel } from "../components/StatePanel";
import { StatusBadge } from "../components/StatusBadge";
import styles from "./Pages.module.css";

const monitoringAreas: ReadonlyArray<{
  href: string;
  labelKey:
    | "navigation.cpu"
    | "navigation.memory"
    | "navigation.filesystems"
    | "navigation.docker";
  icon: LucideIcon;
}> = [
  { href: "/cpu", labelKey: "navigation.cpu", icon: Cpu },
  { href: "/memory", labelKey: "navigation.memory", icon: MemoryStick },
  {
    href: "/filesystems",
    labelKey: "navigation.filesystems",
    icon: HardDrive,
  },
  { href: "/docker", labelKey: "navigation.docker", icon: Boxes },
];

export function OverviewPage() {
  const { t } = useTranslation();

  return (
    <div className={styles.page}>
      <header className={styles.pageHeader}>
        <div>
          <p className={styles.eyebrow}>{t("overview.eyebrow")}</p>
          <h1>{t("overview.title")}</h1>
          <p className={styles.introduction}>{t("overview.introduction")}</p>
        </div>
        <StatusBadge state="unknown" />
      </header>

      <section
        className={styles.section}
        aria-labelledby="overall-health-title"
      >
        <div className={styles.sectionHeading}>
          <h2 id="overall-health-title">{t("overview.healthTitle")}</h2>
          <StatusBadge state="unknown" />
        </div>
        <StatePanel
          variant="unavailable"
          title={t("status.unknown")}
          message={t("overview.healthMessage")}
        />
      </section>

      <section className={styles.section} aria-labelledby="areas-title">
        <div className={styles.sectionHeading}>
          <h2 id="areas-title">{t("overview.sectionsTitle")}</h2>
        </div>
        <div className={styles.areaGrid}>
          {monitoringAreas.map((area) => {
            const Icon = area.icon;
            return (
              <Link
                className={styles.areaCard}
                key={area.href}
                href={area.href}
              >
                <span className={styles.areaIcon}>
                  <Icon aria-hidden="true" size={21} strokeWidth={1.8} />
                </span>
                <strong>{t(area.labelKey)}</strong>
                <span className={styles.areaState}>{t("status.unknown")}</span>
                <MoveUpRight
                  aria-hidden="true"
                  className={styles.areaArrow}
                  size={17}
                />
              </Link>
            );
          })}
        </div>
      </section>
    </div>
  );
}

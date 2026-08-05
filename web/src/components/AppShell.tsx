import {
  Activity,
  Boxes,
  Cpu,
  DatabaseBackup,
  Gauge,
  HardDrive,
  History,
  MemoryStick,
  ScrollText,
  ShieldCheck,
  type LucideIcon,
} from "lucide-react";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { Link, useLocation } from "wouter";

import { LanguageSwitcher } from "./LanguageSwitcher";
import styles from "./AppShell.module.css";

interface NavigationItem {
  href: string;
  labelKey:
    | "navigation.overview"
    | "navigation.cpu"
    | "navigation.memory"
    | "navigation.filesystems"
    | "navigation.docker"
    | "navigation.events"
    | "navigation.audit"
    | "navigation.backups";
  icon: LucideIcon;
}

const navigationItems: readonly NavigationItem[] = [
  { href: "/", labelKey: "navigation.overview", icon: Gauge },
  { href: "/cpu", labelKey: "navigation.cpu", icon: Cpu },
  { href: "/memory", labelKey: "navigation.memory", icon: MemoryStick },
  {
    href: "/filesystems",
    labelKey: "navigation.filesystems",
    icon: HardDrive,
  },
  { href: "/docker", labelKey: "navigation.docker", icon: Boxes },
  { href: "/events", labelKey: "navigation.events", icon: History },
  { href: "/audit", labelKey: "navigation.audit", icon: ScrollText },
  {
    href: "/backups",
    labelKey: "navigation.backups",
    icon: DatabaseBackup,
  },
];

interface AppShellProps {
  children: ReactNode;
}

export function AppShell({ children }: AppShellProps) {
  const { t } = useTranslation();
  const [location] = useLocation();

  return (
    <div className={styles.shell}>
      <a className={styles.skipLink} href="#main-content">
        {t("actions.skipToContent")}
      </a>

      <header className={styles.header}>
        <Link
          className={styles.brand ?? ""}
          href="/"
          aria-label={t("brand.name")}
        >
          <span className={styles.brandMark} aria-hidden="true">
            <Activity size={22} strokeWidth={2} />
          </span>
          <span>
            <strong>{t("brand.name")}</strong>
            <small>{t("brand.environment")}</small>
          </span>
        </Link>
        <LanguageSwitcher />
      </header>

      <aside className={styles.sidebar}>
        <nav aria-label={t("navigation.primary")}>
          <ul className={styles.navigationList}>
            {navigationItems.map((item) => {
              const Icon = item.icon;
              const isActive = location === item.href;
              return (
                <li key={item.href}>
                  <Link
                    className={`${styles.navigationLink} ${isActive ? styles.activeLink : ""}`}
                    href={item.href}
                    {...(isActive ? { "aria-current": "page" as const } : {})}
                  >
                    <Icon aria-hidden="true" size={19} strokeWidth={1.8} />
                    <span>{t(item.labelKey)}</span>
                  </Link>
                </li>
              );
            })}
          </ul>
        </nav>

        <div className={styles.privateNotice}>
          <ShieldCheck aria-hidden="true" size={18} strokeWidth={1.8} />
          <span>{t("footer.access")}</span>
        </div>
      </aside>

      <main className={styles.main} id="main-content" tabIndex={-1}>
        {children}
      </main>

      <footer className={styles.footer}>
        <span>{t("footer.access")}</span>
        <span aria-hidden="true">•</span>
        <span>{t("footer.timezone")}</span>
      </footer>
    </div>
  );
}

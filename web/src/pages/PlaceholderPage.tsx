import { useTranslation } from "react-i18next";

import { StatePanel } from "../components/StatePanel";
import styles from "./Pages.module.css";

export type PlaceholderPageName =
  "cpu" | "memory" | "filesystems" | "docker" | "events" | "audit" | "backups";

interface PlaceholderPageProps {
  page: PlaceholderPageName;
}

export function PlaceholderPage({ page }: PlaceholderPageProps) {
  const { t } = useTranslation();

  return (
    <div className={styles.page}>
      <header className={styles.pageHeader}>
        <div>
          <p className={styles.eyebrow}>{t("pages.eyebrow")}</p>
          <h1>{t(`pages.${page}.title`)}</h1>
          <p className={styles.introduction}>
            {t(`pages.${page}.description`)}
          </p>
        </div>
      </header>

      <StatePanel
        variant="empty"
        title={t("pages.notReady")}
        message={t(`pages.${page}.description`)}
      />
    </div>
  );
}

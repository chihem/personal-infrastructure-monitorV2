import { ArrowLeft } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link } from "wouter";

import styles from "./Pages.module.css";

export function NotFoundPage() {
  const { t } = useTranslation();

  return (
    <div className={styles.page}>
      <header className={styles.pageHeader}>
        <div>
          <p className={styles.eyebrow}>{t("notFound.eyebrow")}</p>
          <h1>{t("notFound.title")}</h1>
          <p className={styles.introduction}>{t("notFound.message")}</p>
        </div>
      </header>
      <Link className={styles.backLink} href="/">
        <ArrowLeft aria-hidden="true" size={18} />
        {t("actions.returnToOverview")}
      </Link>
    </div>
  );
}

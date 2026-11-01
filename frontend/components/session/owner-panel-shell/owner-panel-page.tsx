import type { ReactNode } from "react";

import styles from "./owner-panel-page.module.css";

type OwnerPanelPageProps = {
  actions?: ReactNode;
  children: ReactNode;
  description: string;
  title: string;
};

export function OwnerPanelPage({
  actions,
  children,
  description,
  title,
}: OwnerPanelPageProps) {
  return (
    <div className={styles.page}>
      <header className={styles.intro}>
        <span className={styles.eyebrow}>Painel do estabelecimento</span>
        <div className={styles.titleRow}>
          <h1 id="owner-panel-page-title">{title}</h1>
          {actions ? <div className={styles.actions}>{actions}</div> : null}
        </div>
        <p>{description}</p>
      </header>
      <div className={styles.content}>{children}</div>
    </div>
  );
}

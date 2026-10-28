import type { ReactNode } from "react";

import styles from "./owner-panel-page.module.css";

export function OwnerPanelPage({
  children,
  description,
  title,
}: {
  children: ReactNode;
  description: string;
  title: string;
}) {
  return (
    <div className={styles.page}>
      <header className={styles.intro}>
        <span className={styles.eyebrow}>Painel do estabelecimento</span>
        <h1 id="owner-panel-page-title">{title}</h1>
        <p>{description}</p>
      </header>
      <div className={styles.content}>{children}</div>
    </div>
  );
}

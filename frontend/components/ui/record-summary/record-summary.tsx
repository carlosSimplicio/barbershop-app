import type { ReactNode } from "react";

import styles from "./record-summary.module.css";

export type RecordSummaryProps = {
  title: string;
  children?: ReactNode;
  actions?: ReactNode;
  status?: {
    label: string;
    tone?: "neutral" | "success";
  };
};

export function RecordSummary({
  title,
  children,
  actions,
  status,
}: RecordSummaryProps) {
  return (
    <div className={styles.record}>
      <div className={styles.details}>
        <strong>{title}</strong>
        {children}
      </div>
      {status || actions ? (
        <div className={styles.actions}>
          {status ? (
            <span
              className={styles.status}
              data-tone={status.tone ?? "neutral"}
            >
              {status.label}
            </span>
          ) : null}
          {actions}
        </div>
      ) : null}
    </div>
  );
}

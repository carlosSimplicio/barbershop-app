import type { ReactNode } from "react";

import styles from "./modal-actions.module.css";

export type ModalActionsProps = {
  children: ReactNode;
};

export function ModalActions({ children }: ModalActionsProps) {
  return <div className={styles.actions}>{children}</div>;
}

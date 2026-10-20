import type { ComponentPropsWithRef } from "react";

import styles from "./select.module.css";

export type SelectProps = ComponentPropsWithRef<"select">;

export function Select({ className, hidden, ref, ...props }: SelectProps) {
  const classes = className
    ? `${styles.control} ${styles.select} ${className}`
    : `${styles.control} ${styles.select}`;

  return (
    <span className={styles.selectWrap} hidden={hidden}>
      <select className={classes} hidden={hidden} ref={ref} {...props} />
    </span>
  );
}

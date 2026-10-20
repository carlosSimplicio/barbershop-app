import { forwardRef } from "react";
import type { SelectHTMLAttributes } from "react";

import styles from "./form-control.module.css";

export type SelectProps = SelectHTMLAttributes<HTMLSelectElement>;

export const Select = forwardRef<HTMLSelectElement, SelectProps>(
  function Select({ className, hidden, ...props }, ref) {
    const classes = className
      ? `${styles.control} ${styles.select} ${className}`
      : `${styles.control} ${styles.select}`;

    return (
      <span className={styles.selectWrap} hidden={hidden}>
        <select className={classes} hidden={hidden} ref={ref} {...props} />
      </span>
    );
  },
);

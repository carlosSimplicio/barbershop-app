import { forwardRef } from "react";
import type { InputHTMLAttributes, ReactNode } from "react";

import styles from "./form-control.module.css";

export type AffixedInputProps = InputHTMLAttributes<HTMLInputElement> & {
  affix: ReactNode;
};

export const AffixedInput = forwardRef<HTMLInputElement, AffixedInputProps>(
  function AffixedInput({ affix, className, hidden, ...props }, ref) {
    const classes = className
      ? `${styles.affixedControl} ${className}`
      : styles.affixedControl;

    return (
      <span className={styles.affixWrap} hidden={hidden}>
        <span aria-hidden="true" className={styles.affix}>
          {affix}
        </span>
        <input className={classes} hidden={hidden} ref={ref} {...props} />
      </span>
    );
  },
);

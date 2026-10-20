import type { ComponentPropsWithRef, ReactNode } from "react";

import styles from "./affixed-input.module.css";

export type AffixedInputProps = ComponentPropsWithRef<"input"> & {
  affix: ReactNode;
};

export function AffixedInput({
  affix,
  className,
  hidden,
  ref,
  ...props
}: AffixedInputProps) {
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
}

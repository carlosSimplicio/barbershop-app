import type { ComponentPropsWithRef } from "react";

import styles from "./input.module.css";

export type InputProps = ComponentPropsWithRef<"input">;

export function Input({ className, ref, ...props }: InputProps) {
  const classes = className ? `${styles.control} ${className}` : styles.control;

  return <input className={classes} ref={ref} {...props} />;
}

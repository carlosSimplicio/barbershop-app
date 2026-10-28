import type { ComponentPropsWithRef } from "react";

import styles from "./textarea.module.css";

export type TextareaProps = ComponentPropsWithRef<"textarea">;

export function Textarea({ className, ref, ...props }: TextareaProps) {
  const classes = className ? `${styles.control} ${className}` : styles.control;

  return <textarea className={classes} ref={ref} {...props} />;
}

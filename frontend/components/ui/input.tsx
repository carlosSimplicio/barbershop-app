import { forwardRef } from "react";
import type { InputHTMLAttributes } from "react";

import styles from "./form-control.module.css";

export type InputProps = InputHTMLAttributes<HTMLInputElement>;

export const Input = forwardRef<HTMLInputElement, InputProps>(function Input(
  { className, ...props },
  ref,
) {
  const classes = className ? `${styles.control} ${className}` : styles.control;

  return <input className={classes} ref={ref} {...props} />;
});

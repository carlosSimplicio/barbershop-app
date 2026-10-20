import { forwardRef } from "react";
import type { ButtonHTMLAttributes } from "react";

import styles from "./button.module.css";

export type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement>;

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  function Button({ className, type = "button", ...props }, ref) {
    const classes = className ? `${styles.button} ${className}` : styles.button;

    return <button className={classes} ref={ref} type={type} {...props} />;
  },
);

import { forwardRef } from "react";
import type { ButtonHTMLAttributes } from "react";

import styles from "./icon-button.module.css";

export type IconButtonProps = Omit<
  ButtonHTMLAttributes<HTMLButtonElement>,
  "aria-label" | "aria-pressed"
> & {
  "aria-label": string;
  pressed?: boolean;
};

export const IconButton = forwardRef<HTMLButtonElement, IconButtonProps>(
  function IconButton({ className, pressed, type = "button", ...props }, ref) {
    const classes = className
      ? `${styles.iconButton} ${className}`
      : styles.iconButton;

    return (
      <button
        aria-pressed={pressed}
        className={classes}
        ref={ref}
        type={type}
        {...props}
      />
    );
  },
);

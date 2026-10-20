import type { ComponentPropsWithRef } from "react";

import styles from "./icon-button.module.css";

export type IconButtonProps = Omit<
  ComponentPropsWithRef<"button">,
  "aria-label" | "aria-pressed"
> & {
  "aria-label": string;
  pressed?: boolean;
};

export function IconButton({
  className,
  pressed,
  ref,
  type = "button",
  ...props
}: IconButtonProps) {
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
}

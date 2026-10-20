import type { ComponentPropsWithRef } from "react";

import styles from "./button.module.css";

export type ButtonProps = ComponentPropsWithRef<"button">;

export function Button({
  className,
  ref,
  type = "button",
  ...props
}: ButtonProps) {
  const classes = className ? `${styles.button} ${className}` : styles.button;

  return <button className={classes} ref={ref} type={type} {...props} />;
}

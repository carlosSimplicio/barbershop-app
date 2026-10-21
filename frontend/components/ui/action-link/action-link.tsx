import Link from "next/link";
import type { ComponentPropsWithRef } from "react";

import styles from "./action-link.module.css";

export type ActionLinkProps = ComponentPropsWithRef<typeof Link> & {
  variant?: "primary" | "secondary";
};

export function ActionLink({
  className,
  variant = "primary",
  ...props
}: ActionLinkProps) {
  const classes = className ? `${styles.link} ${className}` : styles.link;
  return <Link {...props} className={classes} data-variant={variant} />;
}

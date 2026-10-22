import { useId } from "react";
import type { ComponentPropsWithRef, ReactNode } from "react";

import styles from "./feedback.module.css";

export type FeedbackProps = Omit<ComponentPropsWithRef<"div">, "title"> & {
  action?: ReactNode;
  children: ReactNode;
  title: ReactNode;
  variant: "success" | "warning" | "error";
};

export function Feedback({
  "aria-describedby": describedBy,
  "aria-labelledby": labelledBy,
  action,
  children,
  className,
  ref,
  role,
  title,
  variant,
  ...props
}: FeedbackProps) {
  const generatedId = useId();
  const messageId = `${generatedId}-message`;
  const titleId = `${generatedId}-title`;
  const classes = className
    ? `${styles.feedback} ${className}`
    : styles.feedback;

  return (
    <div
      {...props}
      aria-atomic={props["aria-atomic"] ?? true}
      aria-describedby={describedBy ?? messageId}
      aria-labelledby={labelledBy ?? titleId}
      className={classes}
      data-variant={variant}
      ref={ref}
      role={role ?? (variant === "error" ? "alert" : "status")}
    >
      <span className={styles.icon} aria-hidden="true">
        <FeedbackIcon variant={variant} />
      </span>
      <div className={styles.content}>
        <strong className={styles.title} id={titleId}>
          {title}
        </strong>
        <div className={styles.message} id={messageId}>
          {children}
        </div>
      </div>
      {action ? <div className={styles.action}>{action}</div> : null}
    </div>
  );
}

function FeedbackIcon({ variant }: Pick<FeedbackProps, "variant">) {
  if (variant === "success") {
    return (
      <svg viewBox="0 0 24 24">
        <path d="m6 12 4 4 8-9" />
      </svg>
    );
  }

  if (variant === "warning") {
    return (
      <svg viewBox="0 0 24 24">
        <path d="M12 4 3.5 20h17L12 4Z" />
        <path d="M12 9v5m0 3h.01" />
      </svg>
    );
  }

  return (
    <svg viewBox="0 0 24 24">
      <path d="M7 7l10 10M17 7 7 17" />
      <circle cx="12" cy="12" r="9" />
    </svg>
  );
}

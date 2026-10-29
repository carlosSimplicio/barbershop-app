import { useId } from "react";
import type { AriaAttributes, ComponentPropsWithRef, ReactNode } from "react";

import styles from "./field.module.css";

export type FieldProps = ComponentPropsWithRef<"div"> & {
  children: ReactNode;
  error?: ReactNode;
  errorId?: string;
  hint?: ReactNode;
  hintId?: string;
  htmlFor: string;
  label: ReactNode;
};

type FieldControlOptions = {
  describedBy?: string;
  error?: ReactNode;
  id?: string;
  invalid?: AriaAttributes["aria-invalid"];
  hint?: ReactNode;
};

export function Field({
  children,
  className,
  error,
  errorId,
  hint,
  hintId,
  htmlFor,
  label,
  ...props
}: FieldProps) {
  const classes = className ? `${styles.field} ${className}` : styles.field;
  const hasError = hasContent(error);

  return (
    <div className={classes} {...props}>
      <label className={styles.label} htmlFor={htmlFor}>
        {label}
      </label>
      {children}
      {hasContent(hint) ? (
        <p className={styles.hint} id={hintId}>
          {hint}
        </p>
      ) : null}
      <p aria-live="polite" className={styles.error} id={errorId}>
        {hasError ? error : "\u00a0"}
      </p>
    </div>
  );
}

export function useFieldControl({
  describedBy,
  error,
  hint,
  id,
  invalid,
}: FieldControlOptions) {
  const generatedId = useId();
  const controlId = id ?? generatedId;
  const hintId = hasContent(hint) ? `${controlId}-hint` : undefined;
  const errorId = hasContent(error) ? `${controlId}-error` : undefined;
  const descriptionIds = [describedBy, hintId, errorId]
    .filter(Boolean)
    .join(" ");

  return {
    controlId,
    descriptionIds: descriptionIds || undefined,
    errorId,
    hintId,
    invalid: hasContent(error) ? true : invalid,
  };
}

function hasContent(value: ReactNode) {
  return (
    value !== undefined && value !== null && value !== false && value !== ""
  );
}

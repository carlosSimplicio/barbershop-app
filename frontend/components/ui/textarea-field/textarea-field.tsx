import type { ReactNode } from "react";

import { Field, useFieldControl } from "../field/field";
import { Textarea } from "../textarea/textarea";
import type { TextareaProps } from "../textarea/textarea";
import styles from "./textarea-field.module.css";

export type TextareaFieldProps = TextareaProps & {
  error?: ReactNode;
  fieldClassName?: string;
  hint?: ReactNode;
  label: ReactNode;
};

export function TextareaField({
  "aria-describedby": describedBy,
  "aria-invalid": invalid,
  error,
  fieldClassName,
  hidden,
  hint,
  id,
  label,
  ref,
  ...props
}: TextareaFieldProps) {
  const field = useFieldControl({ describedBy, error, hint, id, invalid });
  const fieldClasses = fieldClassName
    ? `${styles.field} ${fieldClassName}`
    : styles.field;

  return (
    <Field
      className={fieldClasses}
      error={error}
      errorId={field.errorId}
      hidden={hidden}
      hint={hint}
      hintId={field.hintId}
      htmlFor={field.controlId}
      label={label}
    >
      <Textarea
        aria-describedby={field.descriptionIds}
        aria-invalid={field.invalid}
        hidden={hidden}
        id={field.controlId}
        ref={ref}
        {...props}
      />
    </Field>
  );
}

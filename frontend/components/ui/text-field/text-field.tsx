import type { ReactNode } from "react";

import { Field, useFieldControl } from "../field/field";
import { Input } from "../input/input";
import type { InputProps } from "../input/input";
import styles from "./text-field.module.css";

export type TextFieldProps = InputProps & {
  error?: ReactNode;
  fieldClassName?: string;
  hint?: ReactNode;
  label: ReactNode;
};

export function TextField({
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
}: TextFieldProps) {
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
      <Input
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

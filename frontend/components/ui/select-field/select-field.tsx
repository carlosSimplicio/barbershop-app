import type { ReactNode } from "react";

import { Field, useFieldControl } from "../field/field";
import { Select } from "../select/select";
import type { SelectProps } from "../select/select";
import styles from "./select-field.module.css";

export type SelectFieldProps = SelectProps & {
  error?: ReactNode;
  fieldClassName?: string;
  hint?: ReactNode;
  label: ReactNode;
};

export function SelectField({
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
}: SelectFieldProps) {
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
      <Select
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

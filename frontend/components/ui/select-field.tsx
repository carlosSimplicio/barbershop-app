import { forwardRef } from "react";
import type { ReactNode } from "react";

import { Field, useFieldControl } from "./field";
import { Select } from "./select";
import type { SelectProps } from "./select";

export type SelectFieldProps = SelectProps & {
  error?: ReactNode;
  fieldClassName?: string;
  hint?: ReactNode;
  label: ReactNode;
};

export const SelectField = forwardRef<HTMLSelectElement, SelectFieldProps>(
  function SelectField(
    {
      "aria-describedby": describedBy,
      "aria-invalid": invalid,
      error,
      fieldClassName,
      hidden,
      hint,
      id,
      label,
      ...props
    },
    ref,
  ) {
    const field = useFieldControl({ describedBy, error, hint, id, invalid });

    return (
      <Field
        className={fieldClassName}
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
  },
);

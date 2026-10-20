import { forwardRef } from "react";
import type { ReactNode } from "react";

import { Field, useFieldControl } from "./field";
import { Input } from "./input";
import type { InputProps } from "./input";

export type TextFieldProps = InputProps & {
  error?: ReactNode;
  fieldClassName?: string;
  hint?: ReactNode;
  label: ReactNode;
};

export const TextField = forwardRef<HTMLInputElement, TextFieldProps>(
  function TextField(
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
  },
);

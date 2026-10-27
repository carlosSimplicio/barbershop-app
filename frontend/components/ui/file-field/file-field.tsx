"use client";

import { useState } from "react";
import type { ChangeEvent, ReactNode } from "react";

import { Field, useFieldControl } from "../field/field";
import { Input } from "../input/input";
import type { InputProps } from "../input/input";
import styles from "./file-field.module.css";

const supportedImageTypes = "image/png,image/jpeg,image/webp";

export type FileFieldProps = Omit<
  InputProps,
  "defaultValue" | "type" | "value"
> & {
  error?: ReactNode;
  fieldClassName?: string;
  hint?: ReactNode;
  label: ReactNode;
};

export function FileField({
  "aria-describedby": describedBy,
  "aria-invalid": invalid,
  accept,
  error,
  fieldClassName,
  hidden,
  hint,
  id,
  label,
  onChange,
  ref,
  ...props
}: FileFieldProps) {
  const [selectionLabel, setSelectionLabel] = useState(
    "Nenhum arquivo selecionado",
  );
  const field = useFieldControl({ describedBy, error, hint, id, invalid });
  const fieldClasses = fieldClassName
    ? `${styles.field} ${fieldClassName}`
    : styles.field;
  const isInvalid = field.invalid === true || field.invalid === "true";

  function handleChange(event: ChangeEvent<HTMLInputElement>) {
    const files = event.currentTarget.files;
    setSelectionLabel(
      !files?.length
        ? "Nenhum arquivo selecionado"
        : files.length === 1
          ? files[0].name
          : `${files.length} arquivos selecionados`,
    );
    onChange?.(event);
  }

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
      <span
        className={styles.control}
        data-disabled={props.disabled || undefined}
        data-invalid={isInvalid || undefined}
        hidden={hidden}
      >
        <span aria-hidden="true" className={styles.action}>
          Escolher arquivo
        </span>
        <span aria-hidden="true" className={styles.selection}>
          {selectionLabel}
        </span>
        <Input
          accept={accept ?? supportedImageTypes}
          aria-describedby={field.descriptionIds}
          aria-invalid={field.invalid}
          className={styles.input}
          hidden={hidden}
          id={field.controlId}
          onChange={handleChange}
          ref={ref}
          type="file"
          {...props}
        />
      </span>
    </Field>
  );
}

"use client";

import { useState } from "react";
import type { ReactNode } from "react";

import { Field, useFieldControl } from "../field/field";
import { IconButton } from "../icon-button/icon-button";
import { Input } from "../input/input";
import type { InputProps } from "../input/input";
import styles from "./password-field.module.css";

export type PasswordFieldProps = Omit<InputProps, "type"> & {
  error?: ReactNode;
  fieldClassName?: string;
  hidePasswordLabel?: string;
  hint?: ReactNode;
  label: ReactNode;
  onVisibilityChange?: (visible: boolean) => void;
  showPasswordLabel?: string;
};

export function PasswordField({
  "aria-describedby": describedBy,
  "aria-invalid": invalid,
  className,
  disabled,
  error,
  fieldClassName,
  hidden,
  hidePasswordLabel = "Ocultar senha",
  hint,
  id,
  label,
  onVisibilityChange,
  ref,
  showPasswordLabel = "Mostrar senha",
  ...props
}: PasswordFieldProps) {
  const [visible, setVisible] = useState(false);
  const field = useFieldControl({ describedBy, error, hint, id, invalid });
  const inputClasses = className
    ? `${styles.input} ${className}`
    : styles.input;

  function toggleVisibility() {
    const nextVisible = !visible;
    setVisible(nextVisible);
    onVisibilityChange?.(nextVisible);
  }

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
      <span className={styles.passwordWrap}>
        <Input
          aria-describedby={field.descriptionIds}
          aria-invalid={field.invalid}
          className={inputClasses}
          disabled={disabled}
          hidden={hidden}
          id={field.controlId}
          ref={ref}
          type={visible ? "text" : "password"}
          {...props}
        />
        <IconButton
          aria-controls={field.controlId}
          aria-label={visible ? hidePasswordLabel : showPasswordLabel}
          className={styles.toggle}
          disabled={disabled}
          hidden={hidden}
          onClick={toggleVisibility}
          pressed={visible}
        >
          {visible ? <EyeOpenIcon /> : <EyeClosedIcon />}
        </IconButton>
      </span>
    </Field>
  );
}

function EyeOpenIcon() {
  return (
    <svg aria-hidden="true" viewBox="0 0 24 24">
      <path d="M2.5 12s3.5-5 9.5-5 9.5 5 9.5 5-3.5 5-9.5 5-9.5-5-9.5-5Z" />
      <circle cx="12" cy="12" r="2.5" />
    </svg>
  );
}

function EyeClosedIcon() {
  return (
    <svg aria-hidden="true" viewBox="0 0 24 24">
      <path d="M2.5 12s3.5-5 9.5-5 9.5 5 9.5 5-3.5 5-9.5 5-9.5-5-9.5-5Z" />
      <circle cx="12" cy="12" r="2.5" />
      <path d="M4 4 20 20" />
    </svg>
  );
}

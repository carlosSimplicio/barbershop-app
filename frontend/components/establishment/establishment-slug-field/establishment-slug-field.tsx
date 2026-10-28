import { AffixedInput } from "@/components/ui/affixed-input/affixed-input";
import type { AffixedInputProps } from "@/components/ui/affixed-input/affixed-input";
import { Field, useFieldControl } from "@/components/ui/field/field";
import type { FieldProps } from "@/components/ui/field/field";

type EstablishmentSlugFieldProps = Omit<
  AffixedInputProps,
  "affix" | "aria-describedby" | "aria-invalid" | "id"
> &
  Pick<FieldProps, "error" | "hint" | "label"> & {
    id?: string;
  };

export function EstablishmentSlugField({
  error,
  hint,
  id,
  label,
  ref,
  ...inputProps
}: EstablishmentSlugFieldProps) {
  const field = useFieldControl({ error, hint, id });

  return (
    <Field
      error={error}
      errorId={field.errorId}
      hint={hint}
      hintId={field.hintId}
      htmlFor={field.controlId}
      label={label}
    >
      <AffixedInput
        affix="/"
        aria-describedby={field.descriptionIds}
        aria-invalid={field.invalid}
        id={field.controlId}
        ref={ref}
        {...inputProps}
      />
    </Field>
  );
}

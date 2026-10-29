import { AffixedInput, Field, TextField } from "@/components/ui";
import { useFieldControl } from "@/components/ui/field/field";
import {
  offeringFieldHints,
  type FieldErrors,
  type OfferingValues,
} from "@/lib/catalog/offering-form-model";

export function OfferingFields({
  values,
  errors,
  disabled,
  onChange,
}: {
  values: OfferingValues;
  errors: FieldErrors;
  disabled: boolean;
  onChange: (field: keyof OfferingValues, value: string) => void;
}) {
  const hint = offeringFieldHints.price;
  const field = useFieldControl({ error: errors.price_cents, hint });
  return (
    <>
      <TextField
        label="Nome do serviço"
        name="name"
        value={values.name}
        error={errors.name}
        hint={offeringFieldHints.name}
        disabled={disabled}
        onChange={(event) => onChange("name", event.target.value)}
      />
      <Field
        label="Preço"
        htmlFor={field.controlId}
        error={errors.price_cents}
        errorId={field.errorId}
        hint={hint}
        hintId={field.hintId}
      >
        <AffixedInput
          id={field.controlId}
          name="price_cents"
          affix="R$"
          inputMode="decimal"
          value={values.price}
          disabled={disabled}
          aria-describedby={field.descriptionIds}
          aria-invalid={field.invalid}
          onChange={(event) => onChange("price", event.target.value)}
        />
      </Field>
      <TextField
        label="Duração (minutos)"
        name="duration_minutes"
        inputMode="numeric"
        value={values.duration}
        error={errors.duration_minutes}
        disabled={disabled}
        hint={offeringFieldHints.duration}
        onChange={(event) => onChange("duration", event.target.value)}
      />
    </>
  );
}

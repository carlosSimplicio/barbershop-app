import { AffixedInput, Field, TextField } from "@/components/ui";
import { useFieldControl } from "@/components/ui/field/field";
import type {
  FieldErrors,
  OfferingValues,
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
  const hint =
    "De R$ 0,00 a R$ 99.999,99. Use vírgula e até duas casas decimais. Zero indica serviço gratuito.";
  const field = useFieldControl({ error: errors.price_cents, hint });
  return (
    <>
      <TextField
        label="Nome do serviço"
        name="name"
        value={values.name}
        error={errors.name}
        hint="Obrigatório, com até 120 caracteres após remover os espaços nas extremidades."
        disabled={disabled}
        onChange={(e) => onChange("name", e.target.value)}
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
          onChange={(e) => onChange("price", e.target.value)}
        />
      </Field>
      <TextField
        label="Duração (minutos)"
        name="duration_minutes"
        inputMode="numeric"
        value={values.duration}
        error={errors.duration_minutes}
        disabled={disabled}
        hint="De 1 a 1.440 minutos. Deixe vazio para usar 30 minutos."
        onChange={(e) => onChange("duration", e.target.value)}
      />
    </>
  );
}

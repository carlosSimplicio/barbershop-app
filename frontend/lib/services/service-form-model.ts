import type { components } from "@/types/generated/api";
export type ServiceValues = { name: string; price: string; duration: string };
export type FieldErrors = Partial<
  Record<"name" | "price_cents" | "duration_minutes", string>
>;
export const emptyServiceValues: ServiceValues = {
  name: "",
  price: "",
  duration: "",
};
export const serviceFieldMessages = {
  invalid_name: "Informe um nome com até 120 caracteres.",
  invalid_price:
    "Informe um preço de R$ 0,00 a R$ 99.999,99, com até duas casas decimais.",
  invalid_duration: "Informe de 1 a 1.440 minutos inteiros.",
};
export function validateServiceValues(values: ServiceValues): {
  errors: FieldErrors;
  request?: components["schemas"]["CreateServiceRequest"];
} {
  const errors: FieldErrors = {};
  const name = values.name.trim();
  if (!name || Array.from(name).length > 120)
    errors.name = serviceFieldMessages.invalid_name;
  const match = /^(\d+)(?:[,.](\d{1,2}))?$/.exec(values.price.trim());
  const cents = match
    ? BigInt(match[1]) * 100n + BigInt((match[2] ?? "").padEnd(2, "0"))
    : null;
  if (cents === null || cents > 9999999n)
    errors.price_cents = serviceFieldMessages.invalid_price;
  const duration = values.duration.trim();
  const minutes = duration === "" ? null : Number(duration);
  if (
    duration !== "" &&
    (!/^\d+$/.test(duration) ||
      !Number.isSafeInteger(minutes) ||
      minutes === null ||
      minutes < 1 ||
      minutes > 1440)
  )
    errors.duration_minutes = serviceFieldMessages.invalid_duration;
  if (Object.keys(errors).length || cents === null) return { errors };
  return {
    errors,
    request: { name, price_cents: Number(cents), duration_minutes: minutes },
  };
}
export function serverServiceErrors(
  fields: components["schemas"]["ErrorResponse"]["fields"],
): FieldErrors {
  const errors: FieldErrors = {};
  for (const issue of fields ?? []) {
    if (!(issue.code in serviceFieldMessages)) continue;
    if (
      issue.field === "name" ||
      issue.field === "price_cents" ||
      issue.field === "duration_minutes"
    )
      errors[issue.field] =
        serviceFieldMessages[issue.code as keyof typeof serviceFieldMessages];
  }
  return errors;
}
export function focusServiceError(
  form: HTMLFormElement | null,
  errors: FieldErrors,
) {
  const first = (["name", "price_cents", "duration_minutes"] as const).find(
    (field) => errors[field],
  );
  const control = first ? form?.elements.namedItem(first) : null;
  if (control instanceof HTMLElement) control.focus();
}

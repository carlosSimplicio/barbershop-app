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

export function recordToServiceValues(
  record: components["schemas"]["ServiceResponse"],
): ServiceValues {
  const cents = BigInt(record.price_cents);
  return {
    name: record.name,
    price: `${cents / 100n},${String(cents % 100n).padStart(2, "0")}`,
    duration: String(record.duration_minutes),
  };
}
export function buildServicePatch(
  values: ServiceValues,
  record: components["schemas"]["ServiceResponse"],
): {
  errors: FieldErrors;
  request?: components["schemas"]["UpdateServiceRequest"];
} {
  const checked = validateServiceValues(values);
  if (!checked.request) return { errors: checked.errors };
  const request: components["schemas"]["UpdateServiceRequest"] = {};
  if (checked.request.name !== record.name) request.name = checked.request.name;
  if (checked.request.price_cents !== record.price_cents)
    request.price_cents = checked.request.price_cents;
  if (
    checked.request.duration_minutes === null ||
    checked.request.duration_minutes !== record.duration_minutes
  )
    request.duration_minutes = checked.request.duration_minutes;
  return { errors: {}, request };
}

export function reconcileServiceValues(
  values: ServiceValues,
  previous: components["schemas"]["ServiceResponse"],
  current: components["schemas"]["ServiceResponse"],
): ServiceValues {
  const baseline = recordToServiceValues(previous);
  const refreshed = recordToServiceValues(current);
  const price = validateServiceValues({
    ...baseline,
    price: values.price,
  }).request;
  const duration = validateServiceValues({
    ...baseline,
    duration: values.duration,
  }).request;
  return {
    name: values.name.trim() !== previous.name ? values.name : refreshed.name,
    price:
      price?.price_cents !== previous.price_cents
        ? values.price
        : refreshed.price,
    duration:
      duration?.duration_minutes !== previous.duration_minutes
        ? values.duration
        : refreshed.duration,
  };
}

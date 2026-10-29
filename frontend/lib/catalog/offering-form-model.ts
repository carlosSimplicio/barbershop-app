import type { components } from "@/types/generated/api";
import { formatOfferingPrice } from "./money";
import { OFFERING_LIMITS } from "./offering-limits";
export type OfferingValues = { name: string; price: string; duration: string };
export type FieldErrors = Partial<
  Record<"name" | "price_cents" | "duration_minutes", string>
>;
export const emptyOfferingValues: OfferingValues = {
  name: "",
  price: "",
  duration: "",
};

const integers = new Intl.NumberFormat("pt-BR", { maximumFractionDigits: 0 });
const minPrice = formatOfferingPrice(OFFERING_LIMITS.priceCents.min).replaceAll(
  "\u00a0",
  " ",
);
const maxPrice = formatOfferingPrice(OFFERING_LIMITS.priceCents.max).replaceAll(
  "\u00a0",
  " ",
);
const minDuration = integers.format(OFFERING_LIMITS.durationMinutes.min);
const maxDuration = integers.format(OFFERING_LIMITS.durationMinutes.max);

export const offeringFieldMessages = {
  invalid_name: `Informe um nome com até ${OFFERING_LIMITS.nameCodePoints.max} caracteres.`,
  invalid_price: `Informe um preço de ${minPrice} a ${maxPrice}, com até duas casas decimais.`,
  invalid_duration: `Informe de ${minDuration} a ${maxDuration} minutos inteiros.`,
};

export const offeringFieldHints = {
  name: `Obrigatório, com até ${OFFERING_LIMITS.nameCodePoints.max} caracteres após remover os espaços nas extremidades.`,
  price: `De ${minPrice} a ${maxPrice}. Use vírgula e até duas casas decimais. Zero indica serviço gratuito.`,
  duration: `De ${minDuration} a ${maxDuration} minutos. Deixe vazio para usar ${integers.format(OFFERING_LIMITS.durationMinutes.default)} minutos.`,
};
export function validateOfferingValues(values: OfferingValues): {
  errors: FieldErrors;
  request?: components["schemas"]["CreateServiceRequest"];
} {
  const errors: FieldErrors = {};
  const name = values.name.trim();
  if (!name || Array.from(name).length > OFFERING_LIMITS.nameCodePoints.max)
    errors.name = offeringFieldMessages.invalid_name;
  const match = /^(\d+)(?:[,.](\d{1,2}))?$/.exec(values.price.trim());
  const cents = match
    ? BigInt(match[1]) * 100n + BigInt((match[2] ?? "").padEnd(2, "0"))
    : null;
  if (
    cents === null ||
    cents < BigInt(OFFERING_LIMITS.priceCents.min) ||
    cents > BigInt(OFFERING_LIMITS.priceCents.max)
  )
    errors.price_cents = offeringFieldMessages.invalid_price;
  const duration = values.duration.trim();
  const minutes = duration === "" ? null : Number(duration);
  if (
    duration !== "" &&
    (!/^\d+$/.test(duration) ||
      !Number.isSafeInteger(minutes) ||
      minutes === null ||
      minutes < OFFERING_LIMITS.durationMinutes.min ||
      minutes > OFFERING_LIMITS.durationMinutes.max)
  )
    errors.duration_minutes = offeringFieldMessages.invalid_duration;
  if (Object.keys(errors).length || cents === null) return { errors };
  return {
    errors,
    request: { name, price_cents: Number(cents), duration_minutes: minutes },
  };
}
export function serverOfferingErrors(
  fields: components["schemas"]["ErrorResponse"]["fields"],
): FieldErrors {
  const errors: FieldErrors = {};
  for (const issue of fields ?? []) {
    if (!(issue.code in offeringFieldMessages)) continue;
    if (
      issue.field === "name" ||
      issue.field === "price_cents" ||
      issue.field === "duration_minutes"
    )
      errors[issue.field] =
        offeringFieldMessages[issue.code as keyof typeof offeringFieldMessages];
  }
  return errors;
}
export function focusOfferingError(
  form: HTMLFormElement | null,
  errors: FieldErrors,
) {
  const first = (["name", "price_cents", "duration_minutes"] as const).find(
    (field) => errors[field],
  );
  const control = first ? form?.elements.namedItem(first) : null;
  if (control instanceof HTMLElement) control.focus();
}

export function offeringToFormValues(
  record: components["schemas"]["ServiceResponse"],
): OfferingValues {
  const cents = BigInt(record.price_cents);
  return {
    name: record.name,
    price: `${cents / 100n},${String(cents % 100n).padStart(2, "0")}`,
    duration: String(record.duration_minutes),
  };
}
export function buildOfferingPatch(
  values: OfferingValues,
  record: components["schemas"]["ServiceResponse"],
): {
  errors: FieldErrors;
  request?: components["schemas"]["UpdateServiceRequest"];
} {
  const checked = validateOfferingValues(values);
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

export function reconcileOfferingValues(
  values: OfferingValues,
  previous: components["schemas"]["ServiceResponse"],
  current: components["schemas"]["ServiceResponse"],
): OfferingValues {
  const baseline = offeringToFormValues(previous);
  const refreshed = offeringToFormValues(current);
  const price = validateOfferingValues({
    ...baseline,
    price: values.price,
  }).request;
  const duration = validateOfferingValues({
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

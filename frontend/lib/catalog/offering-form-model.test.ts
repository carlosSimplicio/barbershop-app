import { expect, test } from "vitest";
import {
  validateOfferingValues,
  offeringToFormValues,
  buildOfferingPatch,
} from "./offering-form-model";
test("converte centavos sem arredondar e aplica duração vazia", () => {
  for (const [price, cents] of [
    ["0,00", 0],
    ["0,01", 1],
    ["99999,99", 9999999],
    ["12.34", 1234],
  ] as const) {
    const result = validateOfferingValues({
      name: " Corte ",
      price,
      duration: "",
    });
    expect(result.errors).toEqual({});
    expect(result.request).toEqual({
      name: "Corte",
      price_cents: cents,
      duration_minutes: null,
    });
  }
  expect(
    validateOfferingValues({
      name: " " + "á".repeat(120) + " ",
      price: "0",
      duration: "17",
    }).request?.duration_minutes,
  ).toBe(17);
});
test("recusa vazio, frações excedentes e limites sem arredondamento", () => {
  for (const price of ["", "-1", "0,001", "100000,00", "1e2", "1.234,56"]) {
    expect(
      validateOfferingValues({ name: "Corte", price, duration: "30" }).errors
        .price_cents,
    ).toBeDefined();
  }
  for (const duration of ["0", "1441", "1,5", "1e2"]) {
    expect(
      validateOfferingValues({ name: "Corte", price: "0", duration }).errors
        .duration_minutes,
    ).toBeDefined();
  }
  expect(
    validateOfferingValues({ name: " ", price: "0", duration: "" }).errors.name,
  ).toBeDefined();
  expect(
    validateOfferingValues({ name: "á".repeat(121), price: "0", duration: "" })
      .errors.name,
  ).toBeDefined();
});

test("edição preserva centavos e envia somente campos alterados", () => {
  const record = {
    id: "id",
    name: "Corte",
    price_cents: 1,
    duration_minutes: 17,
    active: false,
  };
  const values = offeringToFormValues(record);
  expect(values).toEqual({ name: "Corte", price: "0,01", duration: "17" });
  expect(
    buildOfferingPatch({ ...values, price: "1,01" }, record).request,
  ).toEqual({ price_cents: 101 });
  expect(
    buildOfferingPatch({ ...values, duration: "" }, record).request,
  ).toEqual({ duration_minutes: null });
  expect(buildOfferingPatch(values, record).request).toEqual({});
});

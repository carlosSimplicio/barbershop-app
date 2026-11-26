import { expect, test } from "vitest";
import {
  isProfessionalService,
  classifyProfessionalServiceOperation,
  isProfessionalServicesPage,
} from "./professional-services-result";

const offering = {
  id: "service",
  name: "Corte",
  price_cents: 0,
  duration_minutes: 17,
  active: false,
  associated: false,
};
const page = {
  professional: { id: "professional", name: "Ana", active: false },
  items: [offering],
  limit: 20,
  offset: 0,
  has_more: false,
  can_write: false,
};
test("valida o estado do vínculo separadamente dos cadastros e protege formatação monetária", () => {
  expect(isProfessionalService(offering)).toBe(true);
  expect(isProfessionalServicesPage(page)).toBe(true);
  for (const invalid of [
    null,
    {},
    { ...offering, associated: undefined },
    { ...offering, associated: "false" },
    { ...offering, active: null },
    { ...offering, price_cents: 1.5 },
    { ...offering, price_cents: NaN },
    { ...offering, duration_minutes: 0 },
  ])
    expect(isProfessionalService(invalid)).toBe(false);
  for (const invalid of [
    null,
    {},
    { ...page, professional: null },
    { ...page, can_write: undefined },
    { ...page, items: null },
    { ...page, offset: -1 },
    { ...page, offset: 2147483648 },
    { ...page, limit: 0 },
    { ...page, items: [{ ...offering, associated: null }] },
  ])
    expect(isProfessionalServicesPage(invalid)).toBe(false);
});

test("só confirma 200 válido com serviço e estado solicitados", () => {
  expect(
    classifyProfessionalServiceOperation(
      { state: "success", status: 200, data: offering },
      offering.id,
      false,
    ),
  ).toEqual({ state: "confirmed", record: offering });
  for (const result of [
    { state: "success" as const, status: 201, data: offering },
    {
      state: "success" as const,
      status: 200,
      data: { ...offering, id: "other" },
    },
    {
      state: "success" as const,
      status: 200,
      data: { ...offering, associated: true },
    },
    {
      state: "success" as const,
      status: 200,
      data: { ...offering, price_cents: 1.5 },
    },
    {
      state: "error" as const,
      status: 500,
      error: { code: "internal_error", message: "raw", request_id: "id" },
    },
  ])
    expect(
      classifyProfessionalServiceOperation(result, offering.id, false),
    ).toEqual({ state: "rejected" });
  expect(
    classifyProfessionalServiceOperation(
      {
        state: "error",
        status: 403,
        error: { code: "forbidden", message: "raw", request_id: "id" },
      },
      offering.id,
      false,
    ),
  ).toEqual({ state: "forbidden" });
});

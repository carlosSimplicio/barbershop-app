import { expect, test } from "vitest";
import {
  isProfessionalService,
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

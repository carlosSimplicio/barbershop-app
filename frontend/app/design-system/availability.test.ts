import { expect, test } from "vitest";

import { isDesignSystemAvailable } from "./availability";

test("disponibiliza o catálogo durante o desenvolvimento", () => {
  expect(isDesignSystemAvailable("development")).toBe(true);
});

test("indisponibiliza o catálogo em produção", () => {
  expect(isDesignSystemAvailable("production")).toBe(false);
});

test.each(["test", undefined])(
  "indisponibiliza o catálogo fora do desenvolvimento (%s)",
  (environment) => {
    expect(isDesignSystemAvailable(environment)).toBe(false);
  },
);

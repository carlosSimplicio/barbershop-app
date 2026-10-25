import { expect, test } from "vitest";

import { validateEmail } from "./email-validation";

test("aceita e-mail válido com espaços externos e respeita o limite de 254 caracteres", () => {
  expect(validateEmail(" owner@example.test ")).toBeUndefined();
  expect(validateEmail(`${"a".repeat(242)}@example.com`)).toBeUndefined();
  expect(validateEmail(`${"a".repeat(243)}@example.com`)).toBe(
    "Digite um e-mail válido.",
  );
});

test("distingue e-mail ausente de formato inválido", () => {
  expect(validateEmail("  ")).toBe("Informe seu e-mail.");
  expect(validateEmail("owner@invalid")).toBe("Digite um e-mail válido.");
});

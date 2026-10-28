import { expect, test } from "vitest";

import {
  INVALID_PASSWORD_MESSAGE,
  validatePassword,
} from "./password-validation";

test("valida tamanho em caracteres e rejeita espaços em qualquer posição", () => {
  expect(validatePassword("12345678", "Obrigatória.")).toBeUndefined();
  expect(validatePassword("1234567", "Obrigatória.")).toBe(
    INVALID_PASSWORD_MESSAGE,
  );
  expect(validatePassword("a".repeat(32), "Obrigatória.")).toBeUndefined();
  expect(validatePassword("a".repeat(33), "Obrigatória.")).toBe(
    INVALID_PASSWORD_MESSAGE,
  );
  expect(validatePassword("senha\tsegura", "Obrigatória.")).toBe(
    INVALID_PASSWORD_MESSAGE,
  );
  expect(validatePassword("á".repeat(8), "Obrigatória.")).toBeUndefined();
});

test("mantém a mensagem de obrigatoriedade própria de cada formulário", () => {
  expect(validatePassword("", "Crie uma senha.")).toBe("Crie uma senha.");
  expect(validatePassword("", "Informe uma nova senha.")).toBe(
    "Informe uma nova senha.",
  );
});

test.each([
  "\u0085",
  "\u00a0",
  "\u1680",
  "\u2007",
  "\u2028",
  "\u202f",
  "\u3000",
])("rejeita whitespace Unicode %j, como o backend", (space) => {
  expect(validatePassword(`abcd${space}efgh`, "Obrigatória.")).toBe(
    INVALID_PASSWORD_MESSAGE,
  );
});

test("aceita U+FEFF e espaços de largura zero que o backend não classifica como whitespace", () => {
  expect(validatePassword("abcd\uFEFFefgh", "Obrigatória.")).toBeUndefined();
  expect(validatePassword("abcd\u200Befgh", "Obrigatória.")).toBeUndefined();
});

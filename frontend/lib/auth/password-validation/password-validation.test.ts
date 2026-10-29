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

test.each([
  "\u0000",
  "\u001B",
  "\u007F",
  "\u00AD",
  "\u034F",
  "\u061C",
  "\u115F",
  "\u17B4",
  "\u180B",
  "\u200B",
  "\u200C",
  "\u200D",
  "\u202E",
  "\u2060",
  "\u2066",
  "\u3164",
  "\uFE0F",
  "\uFEFF",
  "\uFFA0",
  "\u{E0020}",
  "\u{E0080}",
  "\u{E0100}",
])(
  "rejeita o caractere de controle ou invisível %j em qualquer posição",
  (character) => {
    for (const password of [
      `${character}abcdefgh`,
      `abcd${character}efgh`,
      `abcdefgh${character}`,
    ]) {
      expect(validatePassword(password, "Obrigatória.")).toBe(
        INVALID_PASSWORD_MESSAGE,
      );
    }
  },
);

test.each(["áéíóúçãõ", "Cafe\u0301-123", "عربي1234", "🔑🔒🙂😀😃😄😁😆"])(
  "preserva Unicode visível e acentos combinantes normais",
  (password) => {
    expect(validatePassword(password, "Obrigatória.")).toBeUndefined();
  },
);

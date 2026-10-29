import { expect, test } from "vitest";
import { validateProfessionalName } from "./professional-form-model";

test("compartilha o limite Unicode e a recusa de NUL entre cadastro e edição", () => {
  expect(validateProfessionalName("á".repeat(120))).toBeUndefined();
  for (const name of ["á".repeat(121), "Ana\0", " \t "])
    expect(validateProfessionalName(name)).toBeTypeOf("string");
  expect(validateProfessionalName(" Ana ")).toBeUndefined();
});

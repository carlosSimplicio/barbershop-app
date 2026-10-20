import { render, screen } from "@testing-library/react";
import { createRef } from "react";
import { expect, test } from "vitest";

import { Input } from "../input/input";
import { Field } from "./field";

test("permite compor um campo acessível com controle próprio e ref", () => {
  const ref = createRef<HTMLDivElement>();

  render(
    <Field
      error="Nome inválido."
      errorId="name-error"
      hint="Use o nome público."
      hintId="name-hint"
      htmlFor="name"
      label="Nome"
      ref={ref}
    >
      <Input
        aria-describedby="name-hint name-error"
        aria-invalid="true"
        id="name"
      />
    </Field>,
  );

  const input = screen.getByLabelText("Nome");

  expect(input).toHaveAccessibleDescription(
    "Use o nome público. Nome inválido.",
  );
  expect(input).toHaveAttribute("aria-invalid", "true");
  expect(ref.current).toContainElement(input);
});

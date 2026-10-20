import { render, screen } from "@testing-library/react";
import { createRef } from "react";
import { expect, test } from "vitest";

import { TextField } from "./text-field";

test("associa automaticamente label, ajuda, erro e ref", () => {
  const ref = createRef<HTMLInputElement>();

  render(
    <>
      <p id="external-description">Informação externa.</p>
      <TextField
        aria-describedby="external-description"
        error="Digite um e-mail válido."
        hint="Use seu e-mail principal."
        id="email"
        label="E-mail"
        ref={ref}
      />
    </>,
  );

  const input = screen.getByLabelText("E-mail");

  expect(input).toHaveAttribute("id", "email");
  expect(input).toHaveAttribute("aria-invalid", "true");
  expect(input).toHaveAccessibleDescription(
    "Informação externa. Use seu e-mail principal. Digite um e-mail válido.",
  );
  expect(ref.current).toBe(input);
});

test("gera um id estável quando não recebe id", () => {
  const { rerender } = render(<TextField label="Estabelecimento" />);

  const input = screen.getByLabelText("Estabelecimento");
  const generatedId = input.getAttribute("id");

  expect(input).toHaveAttribute("id");
  expect(generatedId).not.toBe("");

  rerender(<TextField label="Estabelecimento" />);

  expect(screen.getByLabelText("Estabelecimento")).toHaveAttribute(
    "id",
    generatedId,
  );
});

test("oculta o wrapper completo", () => {
  const { container } = render(<TextField hidden label="Campo oculto" />);

  expect(screen.queryByRole("textbox", { name: "Campo oculto" })).toBeNull();
  expect(container.firstElementChild).toHaveAttribute("hidden");
});

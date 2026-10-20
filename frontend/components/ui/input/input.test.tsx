import { fireEvent, render, screen } from "@testing-library/react";
import { createRef } from "react";
import { expect, test, vi } from "vitest";

import { Input } from "./input";

test("preserva atributos, estado, evento e ref do input nativo", () => {
  const handleChange = vi.fn();
  const ref = createRef<HTMLInputElement>();

  render(
    <Input
      aria-label="E-mail"
      aria-invalid="true"
      data-context="cadastro"
      defaultValue="voce@exemplo.com"
      onChange={handleChange}
      ref={ref}
    />,
  );

  const input = screen.getByRole("textbox", { name: "E-mail" });
  fireEvent.change(input, { target: { value: "novo@exemplo.com" } });

  expect(input).toHaveAttribute("aria-invalid", "true");
  expect(input).toHaveAttribute("data-context", "cadastro");
  expect(input).toHaveValue("novo@exemplo.com");
  expect(handleChange).toHaveBeenCalledOnce();
  expect(ref.current).toBe(input);
});

test("preserva o estado desabilitado", () => {
  render(<Input aria-label="Campo indisponível" disabled />);

  expect(
    screen.getByRole("textbox", { name: "Campo indisponível" }),
  ).toBeDisabled();
});

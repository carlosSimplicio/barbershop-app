import { fireEvent, render, screen } from "@testing-library/react";
import { createRef } from "react";
import { expect, test, vi } from "vitest";

import { AffixedInput } from "./affixed-input";

test("mantém o prefixo fora do valor e preserva a API do input", () => {
  const handleChange = vi.fn();
  const ref = createRef<HTMLInputElement>();

  render(
    <AffixedInput
      affix="https://"
      aria-label="Endereço da página"
      aria-invalid="true"
      className="slug-control"
      defaultValue="estudio-aurora"
      onChange={handleChange}
      ref={ref}
    />,
  );

  const input = screen.getByRole("textbox", { name: "Endereço da página" });
  fireEvent.change(input, { target: { value: "novo-endereco" } });

  expect(screen.getByText("https://")).toHaveAttribute("aria-hidden", "true");
  expect(input).toHaveAttribute("aria-invalid", "true");
  expect(input).toHaveClass("slug-control");
  expect(input).toHaveValue("novo-endereco");
  expect(handleChange).toHaveBeenCalledOnce();
  expect(ref.current).toBe(input);
});

test("preserva o estado desabilitado", () => {
  render(<AffixedInput affix="R$" aria-label="Valor indisponível" disabled />);

  expect(
    screen.getByRole("textbox", { name: "Valor indisponível" }),
  ).toBeDisabled();
});

test("oculta o controle e seu adorno por completo", () => {
  const { container } = render(
    <AffixedInput affix="R$" aria-label="Valor oculto" hidden />,
  );

  expect(screen.queryByRole("textbox", { name: "Valor oculto" })).toBeNull();
  expect(container.firstElementChild).toHaveAttribute("hidden");
});

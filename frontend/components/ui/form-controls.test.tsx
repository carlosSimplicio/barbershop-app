import { fireEvent, render, screen } from "@testing-library/react";
import { createRef } from "react";
import { expect, test, vi } from "vitest";

import { AffixedInput } from "./affixed-input";
import { Input } from "./input";
import { Select } from "./select";

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

test("preserva o estado desabilitado do input nativo", () => {
  render(<Input aria-label="Campo indisponível" disabled />);

  expect(
    screen.getByRole("textbox", { name: "Campo indisponível" }),
  ).toBeDisabled();
});

test("preserva opções, seleção, evento e ref do select nativo", () => {
  const handleChange = vi.fn();
  const ref = createRef<HTMLSelectElement>();

  render(
    <Select
      aria-label="Fuso horário"
      defaultValue="America/Sao_Paulo"
      onChange={handleChange}
      ref={ref}
    >
      <option value="America/Sao_Paulo">São Paulo</option>
      <option value="America/Manaus">Manaus</option>
    </Select>,
  );

  const select = screen.getByRole("combobox", { name: "Fuso horário" });
  fireEvent.change(select, { target: { value: "America/Manaus" } });

  expect(handleChange).toHaveBeenCalledOnce();
  expect(select).toHaveValue("America/Manaus");
  expect(ref.current).toBe(select);
});

test("preserva estados inválido e desabilitado do select", () => {
  render(
    <Select aria-label="Seleção indisponível" aria-invalid="true" disabled>
      <option>Opção</option>
    </Select>,
  );

  const select = screen.getByRole("combobox", {
    name: "Seleção indisponível",
  });

  expect(select).toBeDisabled();
  expect(select).toHaveAttribute("aria-invalid", "true");
});

test("oculta o select composto por completo", () => {
  const { container } = render(
    <Select aria-label="Seleção oculta" hidden>
      <option>Opção</option>
    </Select>,
  );

  expect(screen.queryByRole("combobox", { name: "Seleção oculta" })).toBeNull();
  expect(container.firstElementChild).toHaveAttribute("hidden");
});

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

test("preserva o estado desabilitado do input com prefixo", () => {
  render(<AffixedInput affix="R$" aria-label="Valor indisponível" disabled />);

  expect(
    screen.getByRole("textbox", { name: "Valor indisponível" }),
  ).toBeDisabled();
});

test("oculta o input com prefixo e seu adorno por completo", () => {
  const { container } = render(
    <AffixedInput affix="R$" aria-label="Valor oculto" hidden />,
  );

  expect(screen.queryByRole("textbox", { name: "Valor oculto" })).toBeNull();
  expect(container.firstElementChild).toHaveAttribute("hidden");
});

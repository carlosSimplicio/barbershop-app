import { fireEvent, render, screen } from "@testing-library/react";
import { createRef } from "react";
import { expect, test, vi } from "vitest";

import { Select } from "./select";

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

test("preserva estados inválido e desabilitado", () => {
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

test("oculta o controle composto por completo", () => {
  const { container } = render(
    <Select aria-label="Seleção oculta" hidden>
      <option>Opção</option>
    </Select>,
  );

  expect(screen.queryByRole("combobox", { name: "Seleção oculta" })).toBeNull();
  expect(container.firstElementChild).toHaveAttribute("hidden");
});

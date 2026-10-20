import { fireEvent, render, screen } from "@testing-library/react";
import { createRef } from "react";
import { expect, test, vi } from "vitest";

import { SelectField } from "./select-field";

test("associa o campo de seleção e preserva estado e ref", () => {
  const handleChange = vi.fn();
  const ref = createRef<HTMLSelectElement>();

  render(
    <SelectField
      disabled
      label="Fuso horário"
      onChange={handleChange}
      ref={ref}
    >
      <option value="America/Sao_Paulo">São Paulo</option>
      <option value="America/Manaus">Manaus</option>
    </SelectField>,
  );

  const select = screen.getByLabelText("Fuso horário");

  expect(select).toBeDisabled();
  expect(ref.current).toBe(select);
});

test("preserva a interação do controle habilitado", () => {
  const handleChange = vi.fn();

  render(
    <SelectField label="Fuso horário" onChange={handleChange}>
      <option value="America/Sao_Paulo">São Paulo</option>
      <option value="America/Manaus">Manaus</option>
    </SelectField>,
  );

  const select = screen.getByLabelText("Fuso horário");
  fireEvent.change(select, { target: { value: "America/Manaus" } });

  expect(select).toHaveValue("America/Manaus");
  expect(handleChange).toHaveBeenCalledOnce();
});

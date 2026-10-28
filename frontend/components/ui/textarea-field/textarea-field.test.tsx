import { fireEvent, render, screen } from "@testing-library/react";
import { createRef, useState } from "react";
import { expect, test, vi } from "vitest";

import { TextareaField } from "./textarea-field";

test("associa label, ajuda, erro e descrição externa e permite editar o valor", () => {
  const handleChange = vi.fn();
  const ref = createRef<HTMLTextAreaElement>();

  render(
    <>
      <p id="external-description">Informação pública do estabelecimento.</p>
      <TextareaField
        aria-describedby="external-description"
        defaultValue="Atendimento com hora marcada."
        error="Use até 500 caracteres."
        hint="Conte o que torna seu espaço especial."
        id="description"
        label="Descrição"
        maxLength={500}
        onChange={handleChange}
        ref={ref}
      />
    </>,
  );

  const textarea = screen.getByLabelText("Descrição");
  expect(textarea).toHaveAttribute("id", "description");
  expect(textarea).toHaveAttribute("maxlength", "500");
  expect(textarea).toHaveAttribute("aria-invalid", "true");
  expect(textarea).toHaveAccessibleDescription(
    "Informação pública do estabelecimento. Conte o que torna seu espaço especial. Use até 500 caracteres.",
  );
  expect(textarea).toHaveValue("Atendimento com hora marcada.");
  expect(ref.current).toBe(textarea);

  fireEvent.change(textarea, { target: { value: "Cortes e cuidados." } });

  expect(textarea).toHaveValue("Cortes e cuidados.");
  expect(handleChange).toHaveBeenCalledOnce();
});

test("gera um id estável quando não recebe id", () => {
  const { rerender } = render(<TextareaField label="Endereço" />);

  const textarea = screen.getByLabelText("Endereço");
  const generatedId = textarea.getAttribute("id");

  expect(generatedId).not.toBe("");
  rerender(<TextareaField label="Endereço" />);

  expect(screen.getByLabelText("Endereço")).toHaveAttribute("id", generatedId);
});

test("preserva edição de um valor controlado", () => {
  function ControlledTextareaField() {
    const [value, setValue] = useState("Texto inicial.");

    return (
      <TextareaField
        label="Descrição controlada"
        onChange={(event) => setValue(event.currentTarget.value)}
        value={value}
      />
    );
  }

  render(<ControlledTextareaField />);
  const textarea = screen.getByLabelText("Descrição controlada");

  fireEvent.change(textarea, { target: { value: "Texto atualizado." } });

  expect(textarea).toHaveValue("Texto atualizado.");
});

test("impede interação quando está desabilitado", () => {
  render(<TextareaField disabled label="Descrição indisponível" />);

  expect(screen.getByLabelText("Descrição indisponível")).toBeDisabled();
});

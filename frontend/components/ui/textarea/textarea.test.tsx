import { fireEvent, render, screen } from "@testing-library/react";
import { createRef } from "react";
import { expect, test, vi } from "vitest";

import { Textarea } from "./textarea";

test("preserva atributos nativos, edição e ref", () => {
  const handleChange = vi.fn();
  const ref = createRef<HTMLTextAreaElement>();

  render(
    <Textarea
      aria-label="Descrição pública"
      data-context="configuracoes"
      defaultValue="Atendimento com hora marcada."
      maxLength={500}
      onChange={handleChange}
      ref={ref}
      rows={4}
    />,
  );

  const textarea = screen.getByRole("textbox", { name: "Descrição pública" });
  expect(textarea).toHaveAttribute("data-context", "configuracoes");
  expect(textarea).toHaveAttribute("maxlength", "500");
  expect(textarea).toHaveAttribute("rows", "4");
  expect(ref.current).toBe(textarea);

  fireEvent.change(textarea, { target: { value: "Outro texto." } });

  expect(textarea).toHaveValue("Outro texto.");
  expect(handleChange).toHaveBeenCalledOnce();
});

test("preserva o estado desabilitado", () => {
  render(<Textarea aria-label="Campo indisponível" disabled />);

  expect(
    screen.getByRole("textbox", { name: "Campo indisponível" }),
  ).toBeDisabled();
});

import { fireEvent, render, screen } from "@testing-library/react";
import { createRef } from "react";
import { expect, test, vi } from "vitest";

import { Button } from "./button";

test("permite configurar conteúdo, ação, atributos nativos e ref", () => {
  const handleClick = vi.fn();
  const ref = createRef<HTMLButtonElement>();

  render(
    <Button data-context="cadastro" onClick={handleClick} ref={ref}>
      Criar meu espaço
    </Button>,
  );

  const button = screen.getByRole("button", { name: "Criar meu espaço" });
  fireEvent.click(button);

  expect(handleClick).toHaveBeenCalledOnce();
  expect(button).toHaveAttribute("data-context", "cadastro");
  expect(button).toHaveAttribute("type", "button");
  expect(ref.current).toBe(button);
});

test("não executa a ação quando está desabilitado", () => {
  const handleClick = vi.fn();

  render(
    <Button disabled onClick={handleClick}>
      Salvar
    </Button>,
  );

  fireEvent.click(screen.getByRole("button", { name: "Salvar" }));

  expect(handleClick).not.toHaveBeenCalled();
});

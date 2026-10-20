import { fireEvent, render, screen } from "@testing-library/react";
import { createRef } from "react";
import { expect, test, vi } from "vitest";

import { Button } from "./button";
import { IconButton } from "./icon-button";

test("permite configurar conteúdo, ação e atributos nativos do botão", () => {
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

test("não executa a ação de um botão desabilitado", () => {
  const handleClick = vi.fn();

  render(
    <Button disabled onClick={handleClick}>
      Salvar
    </Button>,
  );

  fireEvent.click(screen.getByRole("button", { name: "Salvar" }));

  expect(handleClick).not.toHaveBeenCalled();
});

test("expõe nome acessível, estado pressionado e ação do botão de ícone", () => {
  const handleClick = vi.fn();

  render(
    <IconButton
      aria-label="Mostrar senha"
      data-context="senha"
      onClick={handleClick}
      pressed
    >
      <span aria-hidden="true">ícone</span>
    </IconButton>,
  );

  const button = screen.getByRole("button", { name: "Mostrar senha" });
  fireEvent.click(button);

  expect(handleClick).toHaveBeenCalledOnce();
  expect(button).toHaveAttribute("aria-pressed", "true");
  expect(button).toHaveAttribute("data-context", "senha");
});

import { fireEvent, render, screen } from "@testing-library/react";
import { createRef } from "react";
import { expect, test, vi } from "vitest";

import { IconButton } from "./icon-button";

test("expõe nome acessível, estado pressionado, ação e ref", () => {
  const handleClick = vi.fn();
  const ref = createRef<HTMLButtonElement>();

  render(
    <IconButton
      aria-label="Mostrar senha"
      data-context="senha"
      onClick={handleClick}
      pressed
      ref={ref}
    >
      <span aria-hidden="true">ícone</span>
    </IconButton>,
  );

  const button = screen.getByRole("button", { name: "Mostrar senha" });
  fireEvent.click(button);

  expect(handleClick).toHaveBeenCalledOnce();
  expect(button).toHaveAttribute("aria-pressed", "true");
  expect(button).toHaveAttribute("data-context", "senha");
  expect(ref.current).toBe(button);
});

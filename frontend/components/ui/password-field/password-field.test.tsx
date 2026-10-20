import { fireEvent, render, screen } from "@testing-library/react";
import { createRef } from "react";
import { expect, test, vi } from "vitest";

import { PasswordField } from "./password-field";

test("alterna a visibilidade com mensagens, callback e ref configuráveis", () => {
  const handleVisibilityChange = vi.fn();
  const ref = createRef<HTMLInputElement>();

  render(
    <PasswordField
      className="password-control"
      defaultValue="senhasegura"
      hidePasswordLabel="Esconder conteúdo"
      label="Senha"
      onVisibilityChange={handleVisibilityChange}
      ref={ref}
      showPasswordLabel="Revelar conteúdo"
    />,
  );

  const input = screen.getByLabelText("Senha");
  const showButton = screen.getByRole("button", {
    name: "Revelar conteúdo",
  });

  expect(input).toHaveAttribute("type", "password");
  expect(input).toHaveClass("password-control");
  expect(ref.current).toBe(input);
  fireEvent.click(showButton);

  expect(input).toHaveAttribute("type", "text");
  expect(input).toHaveValue("senhasegura");
  expect(handleVisibilityChange).toHaveBeenLastCalledWith(true);

  fireEvent.click(screen.getByRole("button", { name: "Esconder conteúdo" }));

  expect(input).toHaveAttribute("type", "password");
  expect(handleVisibilityChange).toHaveBeenLastCalledWith(false);
});

test("desabilita o controle e a ação de visibilidade", () => {
  render(<PasswordField disabled label="Senha indisponível" />);

  expect(screen.getByLabelText("Senha indisponível")).toBeDisabled();
  expect(screen.getByRole("button", { name: "Mostrar senha" })).toBeDisabled();
});

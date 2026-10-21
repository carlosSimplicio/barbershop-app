import { render, screen } from "@testing-library/react";
import { createRef } from "react";
import { expect, test } from "vitest";

import { ActionLink } from "./action-link";

test("mantém navegação como link acessível nas duas variantes", () => {
  const ref = createRef<HTMLAnchorElement>();
  render(
    <>
      <ActionLink href="/cadastro" ref={ref}>
        Criar minha conta
      </ActionLink>
      <ActionLink href="/login" variant="secondary">
        Entrar
      </ActionLink>
    </>,
  );

  const register = screen.getByRole("link", { name: "Criar minha conta" });
  const login = screen.getByRole("link", { name: "Entrar" });
  expect(register).toHaveAttribute("href", "/cadastro");
  expect(register).toHaveAttribute("data-variant", "primary");
  expect(ref.current).toBe(register);
  expect(login).toHaveAttribute("href", "/login");
  expect(login).toHaveAttribute("data-variant", "secondary");
});

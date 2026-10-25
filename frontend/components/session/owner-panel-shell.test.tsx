import { render, screen } from "@testing-library/react";
import { expect, test } from "vitest";

import { OwnerPanelShell } from "./owner-panel-shell";

test("provides keyboard-accessible panel navigation and a main-content skip link", () => {
  render(
    <OwnerPanelShell>
      <h1>Profissionais</h1>
    </OwnerPanelShell>,
  );

  const navigation = screen.getByRole("navigation", {
    name: "Navegação do painel",
  });
  const professionalsLink = screen.getByRole("link", { name: "Profissionais" });
  const testsLink = screen.getByRole("link", { name: "Área de testes" });
  const skipLink = screen.getByRole("link", {
    name: "Pular para o conteúdo principal",
  });

  expect(navigation).toContainElement(professionalsLink);
  expect(navigation).toContainElement(testsLink);
  expect(professionalsLink).toHaveAttribute("href", "/painel/profissionais");
  expect(professionalsLink).toHaveAttribute("aria-current", "page");
  expect(testsLink).toHaveAttribute("href", "/testes");
  expect(skipLink).toHaveAttribute("href", "#panel-main");
  expect(screen.getByRole("main")).toHaveAttribute("id", "panel-main");

  professionalsLink.focus();
  expect(professionalsLink).toHaveFocus();
});

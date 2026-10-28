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
  const settingsLink = screen.getByRole("link", { name: "Configurações" });
  const testsLink = screen.getByRole("link", { name: "Área de testes" });
  const skipLink = screen.getByRole("link", {
    name: "Pular para o conteúdo principal",
  });

  expect(navigation).toContainElement(professionalsLink);
  expect(navigation).toContainElement(settingsLink);
  expect(navigation).toContainElement(testsLink);
  expect(professionalsLink).toHaveAttribute("href", "/painel/profissionais");
  expect(professionalsLink).toHaveAttribute("aria-current", "page");
  expect(settingsLink).toHaveAttribute("href", "/painel/configuracoes");
  expect(settingsLink).not.toHaveAttribute("aria-current");
  expect(testsLink).toHaveAttribute("href", "/testes");
  expect(skipLink).toHaveAttribute("href", "#panel-main");
  expect(screen.getByRole("main")).toHaveAttribute("id", "panel-main");

  professionalsLink.focus();
  expect(professionalsLink).toHaveFocus();
});

test("does not mark the professionals page current on the settings page", () => {
  render(
    <OwnerPanelShell activePage="settings">
      <h1>Configurações</h1>
    </OwnerPanelShell>,
  );

  expect(
    screen.getByRole("link", { name: "Profissionais" }),
  ).not.toHaveAttribute("aria-current");
  expect(screen.getByRole("link", { name: "Configurações" })).toHaveAttribute(
    "aria-current",
    "page",
  );
});

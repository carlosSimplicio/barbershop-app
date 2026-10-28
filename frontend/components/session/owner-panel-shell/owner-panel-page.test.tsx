import { render, screen } from "@testing-library/react";
import { expect, test } from "vitest";

import { OwnerPanelPage } from "./owner-panel-page";

test("renders owner panel pages with the shared heading and content layout", () => {
  render(
    <OwnerPanelPage
      description="Atualize os dados públicos e os limites para agendamentos."
      title="Configurações"
    >
      <form aria-label="Dados do estabelecimento" />
    </OwnerPanelPage>,
  );

  expect(
    screen.getByRole("heading", { level: 1, name: "Configurações" }),
  ).toBeVisible();
  expect(screen.getByText("Painel do estabelecimento")).toBeVisible();
  expect(
    screen.getByText(
      "Atualize os dados públicos e os limites para agendamentos.",
    ),
  ).toBeVisible();
  expect(
    screen.getByRole("form", { name: "Dados do estabelecimento" }),
  ).toBeVisible();
});

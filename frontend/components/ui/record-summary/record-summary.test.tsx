import { fireEvent, render, screen } from "@testing-library/react";
import { expect, test, vi } from "vitest";

import { RecordSummary } from "./record-summary";

test("compõe identificação, detalhes, situação textual e ações do fluxo consumidor", () => {
  const edit = vi.fn();
  render(
    <RecordSummary
      title="Registro de teste"
      status={{ label: "Inativo" }}
      actions={<button onClick={edit}>Editar registro</button>}
    >
      <span>Detalhe do registro</span>
    </RecordSummary>,
  );
  expect(screen.getByText("Registro de teste")).toBeVisible();
  expect(screen.getByText("Detalhe do registro")).toBeVisible();
  expect(screen.getByText("Inativo")).toBeVisible();
  fireEvent.click(screen.getByRole("button", { name: "Editar registro" }));
  expect(edit).toHaveBeenCalledOnce();
});

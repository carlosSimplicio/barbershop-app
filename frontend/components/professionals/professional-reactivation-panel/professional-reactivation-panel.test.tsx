import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { expect, test, vi } from "vitest";
import { reactivateProfessional } from "@/lib/professionals/professionals-api";
import { ProfessionalReactivationPanel } from "./professional-reactivation-panel";
vi.mock("@/lib/professionals/professionals-api", () => ({
  reactivateProfessional: vi.fn(),
  getProfessional: vi.fn(),
}));
const record = {
  id: "id",
  name: "Ana",
  email: "ana@example.test",
  active: false,
};
function renderPanel() {
  render(
    <ProfessionalReactivationPanel
      record={record}
      canWrite
      onClose={vi.fn()}
      onRecordConfirmed={vi.fn()}
      onReadOnly={vi.fn()}
      onUnauthorized={vi.fn()}
      onBusyChange={vi.fn()}
      onReconciliationChange={vi.fn()}
    />,
  );
  fireEvent.change(screen.getByLabelText("Senha temporária para reativação"), {
    target: { value: "senha-temporaria" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Confirmar reativação" }));
}
test("apresenta erro local de senha e foca o campo após a resposta", async () => {
  vi.mocked(reactivateProfessional).mockResolvedValue({
    state: "error",
    status: 400,
    error: {
      code: "validation_failed",
      message: "privado",
      request_id: "id",
      fields: [
        {
          field: "temporary_password",
          code: "invalid_password",
          message: "privado",
        },
      ],
    },
  });
  renderPanel();
  await waitFor(() =>
    expect(
      screen.getByLabelText("Senha temporária para reativação"),
    ).toHaveAttribute("aria-invalid", "true"),
  );
  expect(
    screen.getByLabelText("Senha temporária para reativação"),
  ).toHaveFocus();
  expect(screen.queryByText("privado")).not.toBeInTheDocument();
});
test("recurso ausente limpa senha e permite fechar sem repetir envio", async () => {
  vi.mocked(reactivateProfessional).mockResolvedValue({
    state: "error",
    status: 404,
    error: {
      code: "professional_not_found",
      message: "privado",
      request_id: "id",
    },
  });
  renderPanel();
  await screen.findByText("Profissional não encontrado");
  expect(
    screen.queryByLabelText("Senha temporária para reativação"),
  ).not.toBeInTheDocument();
  expect(
    screen.getByRole("button", { name: "Fechar reativação" }),
  ).toBeEnabled();
});

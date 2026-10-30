import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { expect, test, vi } from "vitest";
import { deactivateProfessional } from "@/lib/professionals/professionals-api";
import { ProfessionalDeactivationPanel } from "./professional-deactivation-panel";
vi.mock("@/lib/professionals/professionals-api", () => ({
  deactivateProfessional: vi.fn(),
  getProfessional: vi.fn(),
}));
test.each([
  [401, "authentication_required"],
  [404, "professional_not_found"],
])("trata recusa %s sem usar texto bruto", async (status, code) => {
  const onUnauthorized = vi.fn();
  vi.mocked(deactivateProfessional).mockResolvedValue({
    state: "error",
    status,
    error: { code, message: "privado", request_id: "id" },
  });
  render(
    <ProfessionalDeactivationPanel
      record={{
        id: "id",
        name: "Ana",
        email: "ana@example.test",
        active: true,
      }}
      canWrite
      onClose={vi.fn()}
      onRecordConfirmed={vi.fn()}
      onReadOnly={vi.fn()}
      onUnauthorized={onUnauthorized}
      onBusyChange={vi.fn()}
      onReconciliationChange={vi.fn()}
    />,
  );
  fireEvent.click(
    screen.getByRole("button", { name: "Confirmar desativação" }),
  );
  if (status === 401)
    await waitFor(() => expect(onUnauthorized).toHaveBeenCalledOnce());
  else
    expect(
      await screen.findByText("Profissional não encontrado"),
    ).toBeInTheDocument();
  expect(screen.queryByText("privado")).not.toBeInTheDocument();
});

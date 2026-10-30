import { act, render, screen, waitFor } from "@testing-library/react";
import { expect, test, vi } from "vitest";
import { getProfessional } from "@/lib/professionals/professionals-api";
import { EditProfessionalPanel } from "./edit-professional-panel";

vi.mock("@/lib/professionals/professionals-api", () => ({
  getProfessional: vi.fn(),
  updateProfessional: vi.fn(),
}));

test("ignora consulta da seleção anterior após abrir outro profissional", async () => {
  const pending: Array<{
    resolve: (result: Awaited<ReturnType<typeof getProfessional>>) => void;
    signal: AbortSignal;
  }> = [];
  vi.mocked(getProfessional).mockImplementation(
    (_id, signal) =>
      new Promise((resolve) => pending.push({ resolve, signal: signal! })),
  );
  const props = {
    canWrite: true,
    onClose: vi.fn(),
    onReadOnly: vi.fn(),
    onUnauthorized: vi.fn(),
    onBusyChange: vi.fn(),
    onRecordConfirmed: vi.fn(),
    onReconciliationChange: vi.fn(),
  };
  const view = render(
    <EditProfessionalPanel key="first" professionalId="first" {...props} />,
  );
  await waitFor(() => expect(pending).toHaveLength(1));
  view.rerender(
    <EditProfessionalPanel key="second" professionalId="second" {...props} />,
  );
  await waitFor(() => expect(pending).toHaveLength(2));
  expect(pending[0].signal.aborted).toBe(true);
  await act(async () =>
    pending[0].resolve({
      state: "success",
      status: 200,
      data: {
        id: "first",
        name: "Anterior",
        email: "first@example.test",
        active: true,
      },
    }),
  );
  expect(props.onRecordConfirmed).not.toHaveBeenCalled();
  await act(async () =>
    pending[1].resolve({
      state: "success",
      status: 200,
      data: {
        id: "second",
        name: "Atual",
        email: "second@example.test",
        active: false,
      },
    }),
  );
  expect(screen.getByLabelText("Nome do profissional")).toHaveValue("Atual");
  expect(props.onRecordConfirmed).toHaveBeenCalledOnce();
});

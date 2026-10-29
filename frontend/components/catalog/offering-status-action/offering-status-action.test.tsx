import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";
import {
  getOffering,
  setOfferingStatus,
  type Offering,
} from "@/lib/catalog/catalog-api";
import { OfferingStatusAction } from "./offering-status-action";
vi.mock("@/lib/catalog/catalog-api", () => ({
  getOffering: vi.fn(),
  setOfferingStatus: vi.fn(),
}));
const record: Offering = {
  id: "corte",
  name: "Corte",
  price_cents: 100,
  duration_minutes: 30,
  active: true,
};
function props() {
  return {
    record,
    canWrite: true,
    onChanged: vi.fn(),
    onReadOnly: vi.fn(),
    onUnauthorized: vi.fn(),
    onBusyChange: vi.fn(),
  };
}
beforeEach(() => {
  vi.mocked(getOffering).mockReset();
  vi.mocked(setOfferingStatus).mockReset();
});
test("confirma o serviço e cancelar não grava", () => {
  render(<OfferingStatusAction {...props()} />);
  fireEvent.click(screen.getByRole("button", { name: "Desativar Corte" }));
  expect(
    screen.getByRole("group", { name: "Confirmar alteração de Corte" }),
  ).toBeVisible();
  fireEvent.click(screen.getByRole("button", { name: "Cancelar alteração" }));
  expect(setOfferingStatus).not.toHaveBeenCalled();
  expect(screen.getByRole("button", { name: "Desativar Corte" })).toHaveFocus();
});
test("envia estado false explícito uma vez e atualiza o registro", async () => {
  let finish!: (value: Awaited<ReturnType<typeof setOfferingStatus>>) => void;
  vi.mocked(setOfferingStatus).mockImplementation(
    () =>
      new Promise((resolve) => {
        finish = resolve;
      }),
  );
  const callbacks = props();
  render(<OfferingStatusAction {...callbacks} />);
  fireEvent.click(screen.getByRole("button", { name: "Desativar Corte" }));
  const confirm = screen.getByRole("button", { name: "Confirmar desativação" });
  fireEvent.click(confirm);
  fireEvent.click(confirm);
  expect(setOfferingStatus).toHaveBeenCalledOnce();
  expect(setOfferingStatus).toHaveBeenCalledWith(
    "corte",
    false,
    expect.any(AbortSignal),
  );
  finish({ state: "success", status: 200, data: { ...record, active: false } });
  await waitFor(() =>
    expect(callbacks.onChanged).toHaveBeenCalledWith({
      ...record,
      active: false,
    }),
  );
  expect(await screen.findByText("Serviço desativado")).toBeVisible();
});
test("consulta estado antes de permitir repetir resultado ambíguo", async () => {
  vi.mocked(setOfferingStatus).mockRejectedValue(new Error("raw"));
  vi.mocked(getOffering).mockResolvedValue({
    state: "success",
    status: 200,
    data: { ...record, active: false },
  });
  const callbacks = props();
  render(<OfferingStatusAction {...callbacks} />);
  fireEvent.click(screen.getByRole("button", { name: "Desativar Corte" }));
  fireEvent.click(
    screen.getByRole("button", { name: "Confirmar desativação" }),
  );
  expect(
    await screen.findByText("Não foi possível confirmar a alteração de estado"),
  ).toBeVisible();
  expect(
    screen.getByRole("button", { name: "Desativar Corte" }),
  ).toBeDisabled();
  fireEvent.click(
    screen.getByRole("button", { name: "Consultar estado do serviço" }),
  );
  await waitFor(() =>
    expect(callbacks.onChanged).toHaveBeenCalledWith({
      ...record,
      active: false,
    }),
  );
  expect(setOfferingStatus).toHaveBeenCalledOnce();
  expect(await screen.findByText("Estado conferido")).toBeVisible();
});
test("bloqueia modo leitura e trata read_only recebido", async () => {
  const callbacks = props();
  const view = render(<OfferingStatusAction {...callbacks} canWrite={false} />);
  expect(
    screen.getByRole("button", { name: "Desativar Corte" }),
  ).toBeDisabled();
  view.rerender(<OfferingStatusAction {...callbacks} />);
  vi.mocked(setOfferingStatus).mockResolvedValue({
    state: "error",
    status: 403,
    error: { code: "read_only", message: "raw", request_id: "test" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Desativar Corte" }));
  fireEvent.click(
    screen.getByRole("button", { name: "Confirmar desativação" }),
  );
  await waitFor(() => expect(callbacks.onReadOnly).toHaveBeenCalledOnce());
  expect(await screen.findByText("Alteração indisponível")).toBeVisible();
  expect(screen.queryByText("raw")).not.toBeInTheDocument();
});
test("descarta sucesso de uma ação desmontada", async () => {
  let finish!: (value: Awaited<ReturnType<typeof setOfferingStatus>>) => void;
  vi.mocked(setOfferingStatus).mockImplementation(
    () =>
      new Promise((resolve) => {
        finish = resolve;
      }),
  );
  const callbacks = props();
  const view = render(<OfferingStatusAction {...callbacks} />);
  fireEvent.click(screen.getByRole("button", { name: "Desativar Corte" }));
  fireEvent.click(
    screen.getByRole("button", { name: "Confirmar desativação" }),
  );
  view.unmount();
  await act(async () =>
    finish({
      state: "success",
      status: 200,
      data: { ...record, active: false },
    }),
  );
  expect(callbacks.onChanged).not.toHaveBeenCalled();
});

test("ativa serviço inativo com estado true explícito", async () => {
  const active = { ...record, active: true };
  vi.mocked(setOfferingStatus).mockResolvedValue({
    state: "success",
    status: 200,
    data: active,
  });
  const callbacks = props();
  render(
    <OfferingStatusAction
      {...callbacks}
      record={{ ...record, active: false }}
    />,
  );
  fireEvent.click(screen.getByRole("button", { name: "Ativar Corte" }));
  fireEvent.click(screen.getByRole("button", { name: "Confirmar ativação" }));
  expect(await screen.findByText("Serviço ativado")).toBeVisible();
  expect(setOfferingStatus).toHaveBeenCalledWith(
    "corte",
    true,
    expect.any(AbortSignal),
  );
  expect(callbacks.onChanged).toHaveBeenCalledWith(active);
});
test.each([401, 403, 404])(
  "trata recusa HTTP %i sem exibir texto do servidor",
  async (status) => {
    const callbacks = props();
    vi.mocked(setOfferingStatus).mockResolvedValue({
      state: "error",
      status,
      error: {
        code:
          status === 401
            ? "authentication_required"
            : status === 404
              ? "not_found"
              : "forbidden",
        message: "raw internal message",
        request_id: "test",
      },
    });
    render(<OfferingStatusAction {...callbacks} />);
    fireEvent.click(screen.getByRole("button", { name: "Desativar Corte" }));
    fireEvent.click(
      screen.getByRole("button", { name: "Confirmar desativação" }),
    );
    if (status === 401)
      await waitFor(() =>
        expect(callbacks.onUnauthorized).toHaveBeenCalledOnce(),
      );
    else
      expect(
        await screen.findByText(
          status === 404
            ? "Serviço não encontrado"
            : "Não foi possível alterar o serviço",
        ),
      ).toBeVisible();
    expect(callbacks.onChanged).not.toHaveBeenCalled();
    expect(screen.queryByText("raw internal message")).not.toBeInTheDocument();
  },
);

import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";
import {
  getService,
  updateService,
  type ServiceRecord,
} from "@/lib/services/services-api";
import { EditServicePanel } from "./edit-service-panel";
vi.mock("@/lib/services/services-api", () => ({
  getService: vi.fn(),
  updateService: vi.fn(),
}));
const record: ServiceRecord = {
  id: "corte",
  name: "Corte",
  price_cents: 1,
  duration_minutes: 17,
  active: false,
};
function success(data = record) {
  return { state: "success" as const, status: 200, data };
}
function props() {
  return {
    serviceId: record.id,
    canWrite: true,
    onSaved: vi.fn(),
    onClose: vi.fn(),
    onReadOnly: vi.fn(),
    onUnauthorized: vi.fn(),
  };
}
beforeEach(() => {
  vi.mocked(getService).mockReset().mockResolvedValue(success());
  vi.mocked(updateService).mockReset();
});
test("carrega inativo e envia apenas preço alterado", async () => {
  vi.mocked(updateService).mockResolvedValue(
    success({ ...record, price_cents: 101 }),
  );
  const callbacks = props();
  render(<EditServicePanel {...callbacks} />);
  expect(await screen.findByLabelText("Preço")).toHaveValue("0,01");
  fireEvent.change(screen.getByLabelText("Preço"), {
    target: { value: "1,01" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Salvar alterações" }));
  expect(await screen.findByText("Serviço atualizado")).toBeVisible();
  expect(updateService).toHaveBeenCalledWith(
    "corte",
    { price_cents: 101 },
    expect.any(AbortSignal),
  );
  expect(callbacks.onSaved).toHaveBeenCalledOnce();
});
test("limpar duração envia null e cancelar não grava", async () => {
  vi.mocked(updateService).mockResolvedValue(
    success({ ...record, duration_minutes: 30 }),
  );
  const callbacks = props();
  render(<EditServicePanel {...callbacks} />);
  await screen.findByLabelText("Preço");
  fireEvent.change(screen.getByLabelText("Duração (minutos)"), {
    target: { value: "" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Salvar alterações" }));
  await waitFor(() =>
    expect(updateService).toHaveBeenCalledWith(
      "corte",
      { duration_minutes: null },
      expect.any(AbortSignal),
    ),
  );
  fireEvent.click(screen.getByRole("button", { name: "Cancelar edição" }));
  expect(callbacks.onClose).toHaveBeenCalledOnce();
  expect(updateService).toHaveBeenCalledOnce();
});
test("preserva rascunho e exige recarga após resposta ambígua", async () => {
  vi.mocked(updateService).mockRejectedValue(new Error("raw"));
  render(<EditServicePanel {...props()} />);
  await screen.findByLabelText("Preço");
  fireEvent.change(screen.getByLabelText("Nome do serviço"), {
    target: { value: "Novo nome" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Salvar alterações" }));
  expect(
    await screen.findByText("Não foi possível confirmar a alteração"),
  ).toBeVisible();
  expect(
    screen.getByRole("button", { name: "Salvar alterações" }),
  ).toBeDisabled();
  expect(screen.getByLabelText("Nome do serviço")).toHaveValue("Novo nome");
  fireEvent.click(screen.getByRole("button", { name: "Recarregar serviço" }));
  await waitFor(() =>
    expect(
      screen.getByRole("button", { name: "Salvar alterações" }),
    ).toBeEnabled(),
  );
  expect(screen.getByLabelText("Nome do serviço")).toHaveValue("Novo nome");
  expect(updateService).toHaveBeenCalledOnce();
});
test("distingue erro de consulta, registro ausente e somente leitura", async () => {
  vi.mocked(getService)
    .mockRejectedValueOnce(new Error("raw"))
    .mockResolvedValueOnce({
      state: "error",
      status: 404,
      error: { code: "not_found", message: "raw", request_id: "test" },
    });
  render(<EditServicePanel {...props()} canWrite={false} />);
  expect(
    await screen.findByText("Não foi possível consultar o serviço"),
  ).toBeVisible();
  fireEvent.click(screen.getByRole("button", { name: "Recarregar serviço" }));
  expect(await screen.findByText("Serviço não encontrado")).toBeVisible();
  expect(updateService).not.toHaveBeenCalled();
});
test("descarta resposta após selecionar outro serviço", async () => {
  let finish!: (value: ReturnType<typeof success>) => void;
  vi.mocked(getService)
    .mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finish = resolve;
        }),
    )
    .mockResolvedValueOnce(success({ ...record, id: "outro", name: "Outro" }));
  const callbacks = props();
  const view = render(<EditServicePanel key="corte" {...callbacks} />);
  await waitFor(() => expect(finish).toBeDefined());
  view.rerender(
    <EditServicePanel key="outro" {...callbacks} serviceId="outro" />,
  );
  expect(await screen.findByLabelText("Nome do serviço")).toHaveValue("Outro");
  await act(async () => finish(success()));
  expect(screen.getByLabelText("Nome do serviço")).toHaveValue("Outro");
});
test("foca erro da API após desbloquear o formulário", async () => {
  vi.mocked(updateService).mockResolvedValue({
    state: "error",
    status: 400,
    error: {
      code: "validation_failed",
      message: "raw",
      request_id: "test",
      fields: [{ field: "price_cents", code: "invalid_price", message: "raw" }],
    },
  });
  render(<EditServicePanel {...props()} />);
  await screen.findByLabelText("Preço");
  fireEvent.change(screen.getByLabelText("Preço"), {
    target: { value: "1,01" },
  });
  screen.getByLabelText("Nome do serviço").focus();
  fireEvent.click(screen.getByRole("button", { name: "Salvar alterações" }));
  await waitFor(() => expect(screen.getByLabelText("Preço")).toHaveFocus());
  expect(screen.queryByText("raw")).not.toBeInTheDocument();
});

test("desabilita edição em somente leitura e sinaliza expiração da API", async () => {
  vi.mocked(updateService).mockResolvedValue({
    state: "error",
    status: 403,
    error: { code: "read_only", message: "raw", request_id: "test" },
  });
  const callbacks = props();
  const view = render(<EditServicePanel {...callbacks} canWrite={false} />);
  await screen.findByLabelText("Preço");
  expect(
    screen.getByRole("button", { name: "Salvar alterações" }),
  ).toBeDisabled();
  view.rerender(<EditServicePanel {...callbacks} />);
  fireEvent.change(screen.getByLabelText("Preço"), {
    target: { value: "2,00" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Salvar alterações" }));
  await waitFor(() => expect(callbacks.onReadOnly).toHaveBeenCalledOnce());
  expect(await screen.findByText("Edição indisponível")).toBeVisible();
});

test("recarga preserva nome editado sem reenviar preço alterado por outro acesso", async () => {
  vi.mocked(getService)
    .mockResolvedValueOnce(success())
    .mockResolvedValueOnce(success({ ...record, price_cents: 200 }));
  vi.mocked(updateService)
    .mockRejectedValueOnce(new Error("network"))
    .mockResolvedValueOnce(
      success({ ...record, name: "Novo nome", price_cents: 200 }),
    );
  render(<EditServicePanel {...props()} />);
  await screen.findByLabelText("Preço");
  fireEvent.change(screen.getByLabelText("Nome do serviço"), {
    target: { value: "Novo nome" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Salvar alterações" }));
  await screen.findByText("Não foi possível confirmar a alteração");
  fireEvent.click(screen.getByRole("button", { name: "Recarregar serviço" }));
  await waitFor(() =>
    expect(
      screen.getByRole("button", { name: "Salvar alterações" }),
    ).toBeEnabled(),
  );
  expect(screen.getByLabelText("Nome do serviço")).toHaveValue("Novo nome");
  expect(screen.getByLabelText("Preço")).toHaveValue("2,00");
  fireEvent.click(screen.getByRole("button", { name: "Salvar alterações" }));
  await screen.findByText("Serviço atualizado");
  expect(updateService).toHaveBeenLastCalledWith(
    "corte",
    { name: "Novo nome" },
    expect.any(AbortSignal),
  );
});

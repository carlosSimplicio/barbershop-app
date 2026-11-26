import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";
import {
  listProfessionalServices,
  setProfessionalService,
  type ProfessionalServicesPage,
} from "@/lib/professionals/professional-services-api";
import { ProfessionalServicesPanel } from "./professional-services-panel";

vi.mock("@/lib/professionals/professional-services-api", () => ({
  listProfessionalServices: vi.fn(),
  setProfessionalService: vi.fn(),
}));

const professional = { id: "professional-id", name: "Ana", active: false };
const offering = {
  id: "offering-id",
  name: "Corte",
  price_cents: 1234,
  duration_minutes: 17,
  active: false,
  associated: true,
};
const props = {
  professional,
  canWrite: true,
  onReadOnly: vi.fn(),
  onBusyChange: vi.fn(),
  onClose: vi.fn(),
  onUnauthorized: vi.fn(),
};
function page(overrides: Partial<ProfessionalServicesPage> = {}) {
  return {
    state: "success" as const,
    status: 200,
    data: {
      professional,
      items: [offering],
      limit: 20,
      offset: 0,
      has_more: false,
      can_write: true,
      ...overrides,
    },
  };
}
function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((done) => {
    resolve = done;
  });
  return { promise, resolve };
}
beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(listProfessionalServices).mockReset();
  vi.mocked(setProfessionalService).mockReset();
});

test("consulta somente a página pedida e distingue inatividade de associação", async () => {
  const pending = deferred<ReturnType<typeof page>>();
  vi.mocked(listProfessionalServices)
    .mockReturnValueOnce(pending.promise)
    .mockResolvedValueOnce(page({ offset: 20, items: [], has_more: false }));
  render(<ProfessionalServicesPanel {...props} />);
  expect(screen.getByText("Carregando serviços…")).toBeVisible();
  await act(async () => pending.resolve(page({ has_more: true })));
  const dialog = screen.getByRole("dialog", { name: "Serviços de Ana" });
  expect(within(dialog).getByText("Corte")).toBeVisible();
  expect(within(dialog).getByText("Associado")).toBeVisible();
  expect(within(dialog).getByText("Inativo")).toBeVisible();
  expect(within(dialog).getByText(/12,34/)).toBeVisible();
  expect(within(dialog).getByText(/17 minutos/)).toBeVisible();
  expect(listProfessionalServices).toHaveBeenCalledOnce();
  fireEvent.click(within(dialog).getByRole("button", { name: "Próxima" }));
  await screen.findByText("Nenhum serviço cadastrado.");
  expect(listProfessionalServices).toHaveBeenLastCalledWith(
    professional.id,
    expect.objectContaining({ limit: 20, offset: 20 }),
  );
  expect(within(dialog).getByText("Página 2")).toBeVisible();
  fireEvent.click(within(dialog).getByRole("button", { name: "Fechar" }));
  expect(props.onClose).toHaveBeenCalledOnce();
});

test("falha temporária permite nova tentativa e atualização sem declarar logout", async () => {
  vi.mocked(listProfessionalServices)
    .mockRejectedValueOnce(new Error("network"))
    .mockResolvedValueOnce(
      page({
        items: [{ ...offering, active: true, associated: false }],
        can_write: false,
      }),
    )
    .mockResolvedValueOnce(page());
  render(<ProfessionalServicesPanel {...props} />);
  fireEvent.click(
    await screen.findByRole("button", { name: "Tentar novamente" }),
  );
  expect(await screen.findByText("Não associado")).toBeVisible();
  expect(screen.getByText("Ativo")).toBeVisible();
  expect(screen.getByText("Modo somente leitura")).toBeVisible();
  fireEvent.click(screen.getByRole("button", { name: "Atualizar lista" }));
  await screen.findByText("Associado");
  expect(listProfessionalServices).toHaveBeenCalledTimes(3);
  expect(props.onUnauthorized).not.toHaveBeenCalled();
});

test("recusa resposta de outro profissional ou página e recurso ausente tem mensagem local", async () => {
  vi.mocked(listProfessionalServices)
    .mockResolvedValueOnce(
      page({ professional: { ...professional, id: "other" } }),
    )
    .mockResolvedValueOnce(page({ offset: 20 }))
    .mockResolvedValueOnce({
      state: "error",
      status: 404,
      error: { code: "not_found", message: "raw", request_id: "id" },
    });
  render(<ProfessionalServicesPanel {...props} />);
  fireEvent.click(
    await screen.findByRole("button", { name: "Tentar novamente" }),
  );
  await waitFor(() =>
    expect(listProfessionalServices).toHaveBeenCalledTimes(2),
  );
  fireEvent.click(
    await screen.findByRole("button", { name: "Tentar novamente" }),
  );
  expect(await screen.findByText("Profissional não encontrado")).toBeVisible();
  expect(screen.queryByText("raw")).not.toBeInTheDocument();
});

test("desmontagem e troca de profissional cancelam leitura e descartam resposta atrasada", async () => {
  const pending = deferred<ReturnType<typeof page>>();
  vi.mocked(listProfessionalServices)
    .mockReturnValueOnce(pending.promise)
    .mockResolvedValueOnce(
      page({
        professional: { ...professional, id: "other", name: "Bia" },
        items: [],
      }),
    );
  const { rerender, unmount } = render(
    <ProfessionalServicesPanel {...props} />,
  );
  const signal = vi.mocked(listProfessionalServices).mock.calls[0][1].signal!;
  rerender(
    <ProfessionalServicesPanel
      {...props}
      professional={{ ...professional, id: "other", name: "Bia" }}
    />,
  );
  await screen.findByText("Nenhum serviço cadastrado.");
  expect(signal.aborted).toBe(true);
  await act(async () => pending.resolve(page()));
  expect(screen.queryByText("Corte")).not.toBeInTheDocument();
  const currentSignal = vi.mocked(listProfessionalServices).mock.calls[1][1]
    .signal!;
  unmount();
  expect(currentSignal.aborted).toBe(true);
});

test("401 retira conteúdo e resposta anterior não o restaura", async () => {
  const pending = deferred<ReturnType<typeof page>>();
  vi.mocked(listProfessionalServices)
    .mockResolvedValueOnce(page({ has_more: true }))
    .mockReturnValueOnce(pending.promise)
    .mockResolvedValueOnce({
      state: "error",
      status: 401,
      error: {
        code: "authentication_required",
        message: "raw",
        request_id: "id",
      },
    });
  const { rerender } = render(<ProfessionalServicesPanel {...props} />);
  await screen.findByText("Corte");
  fireEvent.click(screen.getByRole("button", { name: "Próxima" }));
  rerender(
    <ProfessionalServicesPanel
      {...props}
      professional={{ ...professional, id: "other" }}
    />,
  );
  await waitFor(() => expect(props.onUnauthorized).toHaveBeenCalledOnce());
  await act(async () => pending.resolve(page({ offset: 20 })));
  expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  expect(screen.queryByText("Corte")).not.toBeInTheDocument();
});

test("associa e remove inativos apenas após resposta confirmada, sem GET adicional", async () => {
  vi.mocked(listProfessionalServices).mockResolvedValue(
    page({ items: [{ ...offering, associated: false }], has_more: true }),
  );
  const pending = deferred<{
    state: "success";
    status: number;
    data: typeof offering;
  }>();
  vi.mocked(setProfessionalService)
    .mockReturnValueOnce(pending.promise)
    .mockResolvedValueOnce({
      state: "success",
      status: 200,
      data: { ...offering, associated: false },
    });
  render(<ProfessionalServicesPanel {...props} />);
  const action = await screen.findByRole("button", { name: "Associar Corte" });
  fireEvent.click(action);
  fireEvent.click(action);
  expect(setProfessionalService).toHaveBeenCalledOnce();
  expect(screen.getByText("Não associado")).toBeVisible();
  for (const name of [
    "Associar Corte",
    "Atualizar lista",
    "Fechar",
    "Fechar modal",
  ]) {
    expect(screen.getByRole("button", { name })).toBeDisabled();
  }
  fireEvent(
    screen.getByRole("dialog"),
    new Event("cancel", { bubbles: true, cancelable: true }),
  );
  expect(props.onClose).not.toHaveBeenCalled();
  await act(async () =>
    pending.resolve({ state: "success", status: 200, data: offering }),
  );
  expect(await screen.findByText("Serviço associado")).toBeVisible();
  expect(screen.getByText("Associado")).toBeVisible();
  expect(props.onBusyChange.mock.calls).toEqual([[true], [false]]);
  fireEvent.click(screen.getByRole("button", { name: "Remover Corte" }));
  expect(await screen.findByText("Associação removida")).toBeVisible();
  expect(
    vi
      .mocked(setProfessionalService)
      .mock.calls.map((call) => call.slice(0, 3)),
  ).toEqual([
    [professional.id, offering.id, { associated: true }],
    [professional.id, offering.id, { associated: false }],
  ]);
  expect(listProfessionalServices).toHaveBeenCalledOnce();
  expect(screen.getByRole("button", { name: "Fechar" })).toBeEnabled();
});

test("rede e resposta inválida liberam controles e repetição manual mantém o estado desejado", async () => {
  vi.mocked(listProfessionalServices).mockResolvedValue(
    page({ items: [{ ...offering, associated: false }] }),
  );
  vi.mocked(setProfessionalService)
    .mockRejectedValueOnce(new Error("network"))
    .mockResolvedValueOnce({
      state: "success",
      status: 200,
      data: { ...offering, associated: false },
    })
    .mockResolvedValueOnce({ state: "success", status: 200, data: offering });
  render(<ProfessionalServicesPanel {...props} />);
  fireEvent.click(
    await screen.findByRole("button", { name: "Associar Corte" }),
  );
  expect(
    await screen.findByRole("alert", {
      name: "Não foi possível salvar o vínculo",
    }),
  ).toBeVisible();
  expect(screen.getByRole("button", { name: "Fechar" })).toBeEnabled();
  fireEvent.click(screen.getByRole("button", { name: "Tentar novamente" }));
  await waitFor(() => expect(setProfessionalService).toHaveBeenCalledTimes(2));
  expect(screen.queryByText("Serviço associado")).not.toBeInTheDocument();
  expect(screen.getByText("Não associado")).toBeVisible();
  fireEvent.click(
    await screen.findByRole("button", { name: "Tentar novamente" }),
  );
  expect(await screen.findByText("Serviço associado")).toBeVisible();
  expect(
    vi.mocked(setProfessionalService).mock.calls.map((call) => call[2]),
  ).toEqual([{ associated: true }, { associated: true }, { associated: true }]);
  expect(listProfessionalServices).toHaveBeenCalledOnce();
});

test.each([401, 403, 404, 500])(
  "trata recusa %s localmente e libera atividade",
  async (status) => {
    vi.mocked(listProfessionalServices).mockResolvedValue(page());
    vi.mocked(setProfessionalService).mockResolvedValue({
      state: "error",
      status,
      error: {
        code: status === 403 ? "read_only" : "internal_error",
        message: "raw secret",
        request_id: "fixture",
      },
    });
    render(<ProfessionalServicesPanel {...props} />);
    fireEvent.click(
      await screen.findByRole("button", { name: "Remover Corte" }),
    );
    await waitFor(() =>
      expect(props.onBusyChange).toHaveBeenLastCalledWith(false),
    );
    expect(screen.queryByText("raw secret")).not.toBeInTheDocument();
    if (status === 401) {
      expect(props.onUnauthorized).toHaveBeenCalledOnce();
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    }
    if (status === 403) {
      expect(props.onReadOnly).toHaveBeenCalledOnce();
      expect(
        screen.getByRole("button", { name: "Remover Corte" }),
      ).toBeDisabled();
    }
    if (status === 404)
      expect(
        screen.getByText("Profissional ou serviço não encontrado"),
      ).toBeVisible();
    if (status === 500) {
      expect(
        screen.getByRole("button", { name: "Tentar novamente" }),
      ).toBeEnabled();
      expect(screen.getByRole("button", { name: "Fechar" })).toBeEnabled();
    }
    expect(listProfessionalServices).toHaveBeenCalledOnce();
  },
);

test("desmontagem aborta envio e libera atividade sem aceitar resposta antiga", async () => {
  const pending = deferred<{
    state: "success";
    status: number;
    data: typeof offering;
  }>();
  vi.mocked(listProfessionalServices).mockResolvedValue(page());
  vi.mocked(setProfessionalService).mockReturnValue(pending.promise);
  const { unmount } = render(<ProfessionalServicesPanel {...props} />);
  fireEvent.click(await screen.findByRole("button", { name: "Remover Corte" }));
  const signal = vi.mocked(setProfessionalService).mock.calls[0][3]!;
  unmount();
  expect(signal.aborted).toBe(true);
  expect(props.onBusyChange.mock.calls).toEqual([[true], [false]]);
  await act(async () =>
    pending.resolve({
      state: "success",
      status: 200,
      data: { ...offering, associated: false },
    }),
  );
  expect(props.onBusyChange.mock.calls).toEqual([[true], [false]]);
});

test.each(["workspace", "modal"])(
  "direitos de escrita dependem do %s",
  async (source) => {
    vi.mocked(listProfessionalServices).mockResolvedValue(
      page({ can_write: source !== "modal" }),
    );
    render(
      <ProfessionalServicesPanel
        {...props}
        canWrite={source !== "workspace"}
      />,
    );
    expect(
      await screen.findByRole("button", { name: "Remover Corte" }),
    ).toBeDisabled();
    expect(setProfessionalService).not.toHaveBeenCalled();
  },
);

test("troca de profissional cancela envio e não confirma o resultado na seleção nova", async () => {
  const pending = deferred<{
    state: "success";
    status: number;
    data: typeof offering;
  }>();
  vi.mocked(listProfessionalServices)
    .mockResolvedValueOnce(page())
    .mockResolvedValueOnce(
      page({
        professional: { ...professional, id: "other", name: "Bia" },
        items: [{ ...offering, associated: false }],
      }),
    );
  vi.mocked(setProfessionalService).mockReturnValue(pending.promise);
  const { rerender } = render(<ProfessionalServicesPanel {...props} />);
  fireEvent.click(await screen.findByRole("button", { name: "Remover Corte" }));
  const signal = vi.mocked(setProfessionalService).mock.calls[0][3]!;
  rerender(
    <ProfessionalServicesPanel
      {...props}
      professional={{ ...professional, id: "other", name: "Bia" }}
    />,
  );
  expect(
    await screen.findByRole("button", { name: "Associar Corte" }),
  ).toBeEnabled();
  expect(signal.aborted).toBe(true);
  await act(async () =>
    pending.resolve({
      state: "success",
      status: 200,
      data: { ...offering, associated: false },
    }),
  );
  expect(screen.queryByText("Associação removida")).not.toBeInTheDocument();
  expect(screen.getByRole("dialog", { name: "Serviços de Bia" })).toBeVisible();
});

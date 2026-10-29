import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";
import { useAuthSession } from "@/components/session/auth-provider/auth-provider";
import {
  getService,
  setServiceStatus,
  createService,
  listServices,
  updateService,
  type ServiceRecord,
} from "@/lib/services/services-api";
import { ServicesWorkspace } from "./services-workspace";

vi.mock("@/components/session/auth-provider/auth-provider", () => ({
  useAuthSession: vi.fn(),
}));
vi.mock("@/lib/services/services-api", () => ({
  getService: vi.fn(),
  setServiceStatus: vi.fn(),
  updateService: vi.fn(),
  listServices: vi.fn(),
  createService: vi.fn(),
}));
const records: ServiceRecord[] = [
  {
    id: "corte",
    name: "Corte",
    price_cents: 1,
    duration_minutes: 17,
    active: true,
  },
  {
    id: "barba",
    name: "Barba",
    price_cents: 9999999,
    duration_minutes: 30,
    active: false,
  },
];
function owner(id = "owner") {
  vi.mocked(useAuthSession).mockReturnValue({
    state: {
      status: "authenticated",
      principal: {
        user_id: id,
        establishment_id: id,
        role: "owner",
        password_change_required: false,
        email: "owner@example.test",
      },
    },
  } as ReturnType<typeof useAuthSession>);
}

test("libera cadastro quando a consulta volta a responder com catálogo vazio e escrita ativa", async () => {
  vi.mocked(listServices)
    .mockResolvedValueOnce({
      state: "error",
      status: 500,
      error: {
        code: "internal_error",
        message: "raw internal message",
        request_id: "test",
      },
    })
    .mockResolvedValueOnce(page([], 0, false, true));

  render(<ServicesWorkspace />);
  await screen.findByText("Não foi possível consultar os serviços");
  expect(screen.getByLabelText("Nome do serviço")).toBeDisabled();
  expect(screen.getByLabelText("Preço")).toBeDisabled();
  expect(screen.getByLabelText("Duração (minutos)")).toBeDisabled();

  fireEvent.click(screen.getByRole("button", { name: "Tentar novamente" }));

  await screen.findByText("Nenhum serviço cadastrado.");
  expect(screen.getByLabelText("Nome do serviço")).toBeEnabled();
  expect(screen.getByLabelText("Preço")).toBeEnabled();
  expect(screen.getByLabelText("Duração (minutos)")).toBeEnabled();
  expect(
    screen.getByRole("button", { name: "Cadastrar serviço" }),
  ).toBeEnabled();
  expect(screen.queryByText("Modo somente leitura")).not.toBeInTheDocument();
});
function page(items = records, offset = 0, has_more = false, can_write = true) {
  return {
    state: "success" as const,
    status: 200,
    data: { items, limit: 20, offset, has_more, can_write },
  };
}
beforeEach(() => {
  vi.mocked(getService).mockReset();
  vi.mocked(setServiceStatus).mockReset();
  vi.mocked(listServices).mockReset();
  vi.mocked(createService).mockReset();
  vi.mocked(updateService).mockReset();
  owner();
});
test("exibe preço exato, duração, estado e paginação", async () => {
  vi.mocked(listServices)
    .mockResolvedValueOnce(page(records, 0, true))
    .mockResolvedValueOnce(page([records[1]], 20));
  render(<ServicesWorkspace />);
  expect(screen.getByRole("status")).toHaveTextContent("Carregando serviços");
  const catalog = within(
    await screen.findByRole("list", { name: "Serviços cadastrados" }),
  );
  expect(catalog.getByText("Corte")).toBeVisible();
  expect(catalog.getByText(/R\$\s0,01/)).toBeVisible();
  expect(catalog.getByText(/R\$\s99.999,99/)).toBeVisible();
  expect(catalog.getByText(/17 minutos/)).toBeVisible();
  expect(catalog.getByText("Inativo")).toBeVisible();
  fireEvent.click(screen.getByRole("button", { name: "Próxima" }));
  await waitFor(() =>
    expect(screen.queryByText("Corte")).not.toBeInTheDocument(),
  );
  expect(await screen.findByText("Barba")).toBeVisible();
  expect(screen.getByRole("button", { name: "Próxima" })).toBeDisabled();
});
test("distingue vazio, erro com nova tentativa e somente leitura", async () => {
  vi.mocked(listServices)
    .mockRejectedValueOnce(new Error("internal detail"))
    .mockResolvedValueOnce(page([], 0, false, false));
  render(<ServicesWorkspace />);
  expect(
    await screen.findByText("Não foi possível consultar os serviços"),
  ).toBeVisible();
  expect(screen.queryByText("internal detail")).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole("button", { name: "Tentar novamente" }));
  expect(await screen.findByText("Nenhum serviço cadastrado.")).toBeVisible();
  expect(screen.getByText("Modo somente leitura")).toBeVisible();
});
test("remove conteúdo com 401", async () => {
  vi.mocked(listServices).mockResolvedValue({
    state: "error",
    status: 401,
    error: {
      code: "authentication_required",
      message: "raw",
      request_id: "test",
    },
  });
  render(<ServicesWorkspace />);
  await waitFor(() => expect(listServices).toHaveBeenCalled());
  await waitFor(() =>
    expect(screen.queryByText(/Carregando serviços/)).not.toBeInTheDocument(),
  );
  expect(screen.queryByRole("list")).not.toBeInTheDocument();
});
test("descarta resposta antiga após trocar identidade", async () => {
  let finish!: (value: ReturnType<typeof page>) => void;
  vi.mocked(listServices)
    .mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finish = resolve;
        }),
    )
    .mockResolvedValueOnce(page([records[1]]));
  const view = render(<ServicesWorkspace />);
  await waitFor(() => expect(finish).toBeDefined());
  owner("other");
  view.rerender(<ServicesWorkspace />);
  expect(await screen.findByText("Barba")).toBeVisible();
  await act(async () => finish(page([records[0]])));
  expect(screen.queryByText("Corte")).not.toBeInTheDocument();
});

test("mantém criação confirmada quando a recarga da lista falha", async () => {
  vi.mocked(listServices)
    .mockResolvedValueOnce(page())
    .mockRejectedValueOnce(new Error("network"));
  vi.mocked(createService).mockResolvedValue({
    state: "success",
    status: 201,
    data: records[0],
  });
  render(<ServicesWorkspace />);
  await screen.findByRole("list");
  fireEvent.change(screen.getByLabelText("Nome do serviço"), {
    target: { value: "Corte" },
  });
  fireEvent.change(screen.getByLabelText("Preço"), {
    target: { value: "1,00" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Cadastrar serviço" }));
  expect(await screen.findByText("Serviço cadastrado")).toBeVisible();
  expect(
    await screen.findByText("Não foi possível consultar os serviços"),
  ).toBeVisible();
  expect(screen.getByText("Serviço cadastrado")).toBeVisible();
});

test("retorna foco à ação do registro ao cancelar edição", async () => {
  vi.mocked(listServices).mockResolvedValue(page());
  vi.mocked(getService).mockResolvedValue({
    state: "success",
    status: 200,
    data: records[0],
  });
  render(<ServicesWorkspace />);
  const action = await screen.findByRole("button", { name: "Editar Corte" });
  fireEvent.click(action);
  const editor = await screen.findByRole("region", { name: "Editar serviço" });
  await within(editor).findByLabelText("Nome do serviço");
  fireEvent.click(
    within(editor).getByRole("button", { name: "Cancelar edição" }),
  );
  expect(action).toHaveFocus();
});

test("reconciliação após edição ambígua atualiza também o catálogo", async () => {
  const confirmed = { ...records[0], price_cents: 200 };
  vi.mocked(listServices).mockResolvedValue(page());
  vi.mocked(getService)
    .mockResolvedValueOnce({ state: "success", status: 200, data: records[0] })
    .mockResolvedValueOnce({ state: "success", status: 200, data: confirmed });
  vi.mocked(updateService).mockRejectedValue(new Error("response lost"));
  render(<ServicesWorkspace />);
  fireEvent.click(await screen.findByRole("button", { name: "Editar Corte" }));
  const editor = screen.getByRole("region", { name: "Editar serviço" });
  await within(editor).findByLabelText("Preço");
  fireEvent.change(within(editor).getByLabelText("Preço"), {
    target: { value: "2,00" },
  });
  fireEvent.click(
    within(editor).getByRole("button", { name: "Salvar alterações" }),
  );
  await screen.findByText("Não foi possível confirmar a alteração");
  fireEvent.click(
    within(editor).getByRole("button", { name: "Recarregar serviço" }),
  );
  await waitFor(() =>
    expect(
      within(editor).getByRole("button", { name: "Salvar alterações" }),
    ).toBeEnabled(),
  );
  expect(within(editor).getByLabelText("Preço")).toHaveValue("2,00");
  expect(
    screen.getByRole("list", { name: "Serviços cadastrados" }),
  ).toHaveTextContent(/R\$\s2,00/);
  expect(updateService).toHaveBeenCalledOnce();
  expect(getService).toHaveBeenCalledTimes(2);
});

test("estado confirmado atualiza lista e editor aberto, bloqueando ações enquanto aguarda", async () => {
  let finish!: (value: Awaited<ReturnType<typeof setServiceStatus>>) => void;
  vi.mocked(listServices).mockResolvedValue(page());
  vi.mocked(getService)
    .mockResolvedValueOnce({ state: "success", status: 200, data: records[0] })
    .mockResolvedValueOnce({
      state: "success",
      status: 200,
      data: { ...records[0], active: false },
    });
  vi.mocked(setServiceStatus).mockImplementation(
    () =>
      new Promise((resolve) => {
        finish = resolve;
      }),
  );
  render(<ServicesWorkspace />);
  fireEvent.click(await screen.findByRole("button", { name: "Editar Corte" }));
  const editor = await screen.findByRole("region", { name: "Editar serviço" });
  await within(editor).findByLabelText("Preço");
  fireEvent.click(screen.getByRole("button", { name: "Desativar Corte" }));
  fireEvent.click(
    screen.getByRole("button", { name: "Confirmar desativação" }),
  );
  expect(
    within(editor).getByRole("button", { name: "Salvar alterações" }),
  ).toBeDisabled();
  expect(
    screen.getByRole("button", { name: "Cadastrar serviço" }),
  ).toBeDisabled();
  expect(
    screen.getByRole("button", { name: "Atualizar lista" }),
  ).toBeDisabled();
  await act(async () =>
    finish({
      state: "success",
      status: 200,
      data: { ...records[0], active: false },
    }),
  );
  expect(
    await screen.findByRole("button", { name: "Ativar Corte" }),
  ).toBeVisible();
  await waitFor(() =>
    expect(within(editor).getByText("Estado: Inativo")).toBeVisible(),
  );
});

test("mantém bloqueio global até concluir todas as consultas de estado pendentes", async () => {
  vi.mocked(listServices).mockResolvedValue(page());
  vi.mocked(setServiceStatus).mockRejectedValue(new Error("network"));
  let first!: (value: Awaited<ReturnType<typeof getService>>) => void;
  let second!: (value: Awaited<ReturnType<typeof getService>>) => void;
  vi.mocked(getService)
    .mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          first = resolve;
        }),
    )
    .mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          second = resolve;
        }),
    );
  render(<ServicesWorkspace />);
  fireEvent.click(
    await screen.findByRole("button", { name: "Desativar Corte" }),
  );
  fireEvent.click(
    screen.getByRole("button", { name: "Confirmar desativação" }),
  );
  await screen.findByText("Não foi possível confirmar a alteração de estado");
  fireEvent.click(screen.getByRole("button", { name: "Ativar Barba" }));
  fireEvent.click(screen.getByRole("button", { name: "Confirmar ativação" }));
  await waitFor(() =>
    expect(
      screen.getAllByText("Não foi possível confirmar a alteração de estado"),
    ).toHaveLength(2),
  );
  const consult = screen.getAllByRole("button", {
    name: "Consultar estado do serviço",
  });
  fireEvent.click(consult[0]);
  fireEvent.click(consult[1]);
  expect(getService).toHaveBeenCalledTimes(2);
  await act(async () =>
    first({
      state: "success",
      status: 200,
      data: { ...records[0], active: false },
    }),
  );
  expect(
    screen.getByRole("button", { name: "Cadastrar serviço" }),
  ).toBeDisabled();
  expect(
    screen.getByRole("button", { name: "Atualizar lista" }),
  ).toBeDisabled();
  expect(screen.getByRole("button", { name: "Anterior" })).toBeDisabled();
  await act(async () =>
    second({
      state: "success",
      status: 200,
      data: { ...records[1], active: true },
    }),
  );
  await waitFor(() =>
    expect(
      screen.getByRole("button", { name: "Cadastrar serviço" }),
    ).toBeEnabled(),
  );
  expect(screen.getByRole("button", { name: "Atualizar lista" })).toBeEnabled();
});

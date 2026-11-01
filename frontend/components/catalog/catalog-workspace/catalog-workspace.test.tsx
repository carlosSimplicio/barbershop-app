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
  getOffering,
  setOfferingStatus,
  createOffering,
  listOfferings,
  updateOffering,
  type Offering,
} from "@/lib/catalog/catalog-api";
import { CatalogWorkspace } from "./catalog-workspace";

vi.mock("@/components/session/auth-provider/auth-provider", () => ({
  useAuthSession: vi.fn(),
}));
vi.mock("@/lib/catalog/catalog-api", () => ({
  getOffering: vi.fn(),
  setOfferingStatus: vi.fn(),
  updateOffering: vi.fn(),
  listOfferings: vi.fn(),
  createOffering: vi.fn(),
}));
const records: Offering[] = [
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
  vi.mocked(listOfferings)
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

  render(<CatalogWorkspace />);
  await screen.findByText("Não foi possível consultar os serviços");
  expect(
    screen.getByRole("button", { name: "Adicionar serviço" }),
  ).toBeDisabled();
  expect(screen.queryByRole("dialog")).not.toBeInTheDocument();

  fireEvent.click(screen.getByRole("button", { name: "Tentar novamente" }));

  await screen.findByText("Nenhum serviço cadastrado.");
  fireEvent.click(screen.getByRole("button", { name: "Adicionar serviço" }));
  expect(
    screen.getByRole("dialog", { name: "Cadastrar serviço" }),
  ).toBeVisible();
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
  vi.mocked(getOffering).mockReset();
  vi.mocked(setOfferingStatus).mockReset();
  vi.mocked(listOfferings).mockReset();
  vi.mocked(createOffering).mockReset();
  vi.mocked(updateOffering).mockReset();
  owner();
});
test("exibe preço exato, duração, estado e paginação", async () => {
  vi.mocked(listOfferings)
    .mockResolvedValueOnce(page(records, 0, true))
    .mockResolvedValueOnce(page([records[1]], 20));
  render(<CatalogWorkspace />);
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
  vi.mocked(listOfferings)
    .mockRejectedValueOnce(new Error("internal detail"))
    .mockResolvedValueOnce(page([], 0, false, false));
  render(<CatalogWorkspace />);
  expect(
    await screen.findByText("Não foi possível consultar os serviços"),
  ).toBeVisible();
  expect(screen.queryByText("internal detail")).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole("button", { name: "Tentar novamente" }));
  expect(await screen.findByText("Nenhum serviço cadastrado.")).toBeVisible();
  expect(screen.getByText("Modo somente leitura")).toBeVisible();
});
test("remove conteúdo com 401", async () => {
  vi.mocked(listOfferings).mockResolvedValue({
    state: "error",
    status: 401,
    error: {
      code: "authentication_required",
      message: "raw",
      request_id: "test",
    },
  });
  render(<CatalogWorkspace />);
  await waitFor(() => expect(listOfferings).toHaveBeenCalled());
  await waitFor(() =>
    expect(screen.queryByText(/Carregando serviços/)).not.toBeInTheDocument(),
  );
  expect(screen.queryByRole("list")).not.toBeInTheDocument();
});
test("descarta resposta antiga após trocar identidade", async () => {
  let finish!: (value: ReturnType<typeof page>) => void;
  vi.mocked(listOfferings)
    .mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finish = resolve;
        }),
    )
    .mockResolvedValueOnce(page([records[1]]));
  const view = render(<CatalogWorkspace />);
  await waitFor(() => expect(finish).toBeDefined());
  owner("other");
  view.rerender(<CatalogWorkspace />);
  expect(await screen.findByText("Barba")).toBeVisible();
  await act(async () => finish(page([records[0]])));
  expect(screen.queryByText("Corte")).not.toBeInTheDocument();
});

test("mantém criação confirmada quando a recarga da lista falha", async () => {
  vi.mocked(listOfferings)
    .mockResolvedValueOnce(page())
    .mockRejectedValueOnce(new Error("network"));
  vi.mocked(createOffering).mockResolvedValue({
    state: "success",
    status: 201,
    data: records[0],
  });
  render(<CatalogWorkspace />);
  await screen.findByRole("list");
  fireEvent.click(screen.getByRole("button", { name: "Adicionar serviço" }));
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

test("abre cadastro pelo cabeçalho, mantém rascunho e impede fechar durante envio", async () => {
  vi.mocked(listOfferings).mockResolvedValue(page());
  let finish!: (value: Awaited<ReturnType<typeof createOffering>>) => void;
  vi.mocked(createOffering).mockImplementation(
    () =>
      new Promise((resolve) => {
        finish = resolve;
      }),
  );
  render(<CatalogWorkspace />);
  await screen.findByRole("list");
  expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  const action = screen.getByRole("button", { name: "Adicionar serviço" });
  expect(action.closest("header")).toContainElement(
    screen.getByRole("heading", { level: 1, name: "Serviços" }),
  );
  action.focus();
  fireEvent.click(action);
  fireEvent.change(screen.getByLabelText("Nome do serviço"), {
    target: { value: "Corte" },
  });
  fireEvent.change(screen.getByLabelText("Preço"), {
    target: { value: "1,00" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Fechar modal" }));
  expect(action).toHaveFocus();
  fireEvent.click(action);
  expect(screen.getByLabelText("Preço")).toHaveValue("1,00");
  fireEvent.click(screen.getByRole("button", { name: "Cadastrar serviço" }));
  expect(screen.getByRole("button", { name: "Fechar modal" })).toBeDisabled();
  fireEvent(
    screen.getByRole("dialog"),
    new Event("cancel", { cancelable: true }),
  );
  expect(screen.getByRole("dialog")).toBeVisible();
  await act(async () =>
    finish({ state: "success", status: 201, data: records[0] }),
  );
  expect(await screen.findByText("Serviço cadastrado")).toBeVisible();
});

test("erro no cadastro permite fechar e reabrir sem conferir a lista", async () => {
  vi.mocked(createOffering).mockRejectedValue(new Error("network"));
  vi.mocked(listOfferings).mockResolvedValue(page());
  render(<CatalogWorkspace />);
  await screen.findByRole("list");
  fireEvent.click(screen.getByRole("button", { name: "Adicionar serviço" }));
  fireEvent.change(screen.getByLabelText("Nome do serviço"), {
    target: { value: "Corte" },
  });
  fireEvent.change(screen.getByLabelText("Preço"), {
    target: { value: "1,00" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Cadastrar serviço" }));
  await screen.findByText("Não foi possível cadastrar o serviço");
  expect(
    screen.getByRole("button", { name: "Cadastrar serviço" }),
  ).toBeEnabled();
  fireEvent.click(screen.getByRole("button", { name: "Fechar modal" }));
  fireEvent.click(screen.getByRole("button", { name: "Adicionar serviço" }));
  expect(
    screen.getByRole("button", { name: "Cadastrar serviço" }),
  ).toBeEnabled();
  expect(createOffering).toHaveBeenCalledOnce();
  expect(listOfferings).toHaveBeenCalledOnce();
});

test("retorna foco à ação do registro ao cancelar edição", async () => {
  vi.mocked(listOfferings).mockResolvedValue(page());
  vi.mocked(getOffering).mockResolvedValue({
    state: "success",
    status: 200,
    data: records[0],
  });
  render(<CatalogWorkspace />);
  const action = await screen.findByRole("button", { name: "Editar Corte" });
  fireEvent.click(action);
  const editor = await screen.findByRole("region", { name: "Editar serviço" });
  await within(editor).findByLabelText("Nome do serviço");
  fireEvent.click(
    within(editor).getByRole("button", { name: "Cancelar edição" }),
  );
  expect(action).toHaveFocus();
});

test("edição com erro preserva rascunho e atualiza catálogo somente após nova tentativa bem-sucedida", async () => {
  const confirmed = { ...records[0], price_cents: 200 };
  vi.mocked(listOfferings).mockResolvedValue(page());
  vi.mocked(getOffering).mockResolvedValue({
    state: "success",
    status: 200,
    data: records[0],
  });
  vi.mocked(updateOffering)
    .mockRejectedValueOnce(new Error("network"))
    .mockResolvedValueOnce({ state: "success", status: 200, data: confirmed });
  render(<CatalogWorkspace />);
  fireEvent.click(await screen.findByRole("button", { name: "Editar Corte" }));
  const editor = screen.getByRole("region", { name: "Editar serviço" });
  await within(editor).findByLabelText("Preço");
  fireEvent.change(within(editor).getByLabelText("Preço"), {
    target: { value: "2,00" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Salvar alterações" }));
  await screen.findByText("Não foi possível salvar o serviço");
  expect(within(editor).getByLabelText("Preço")).toHaveValue("2,00");
  expect(getOffering).toHaveBeenCalledOnce();
  expect(updateOffering).toHaveBeenCalledOnce();
  fireEvent.click(screen.getByRole("button", { name: "Salvar alterações" }));
  await screen.findByText("Serviço atualizado");
  expect(
    screen.getByRole("list", { name: "Serviços cadastrados" }),
  ).toHaveTextContent(/R\$\s2,00/);
  expect(getOffering).toHaveBeenCalledOnce();
});

test("estado confirmado atualiza lista e editor aberto, bloqueando ações enquanto aguarda", async () => {
  let finish!: (value: Awaited<ReturnType<typeof setOfferingStatus>>) => void;
  vi.mocked(listOfferings).mockResolvedValue(page());
  vi.mocked(getOffering)
    .mockResolvedValueOnce({ state: "success", status: 200, data: records[0] })
    .mockResolvedValueOnce({
      state: "success",
      status: 200,
      data: { ...records[0], active: false },
    });
  vi.mocked(setOfferingStatus).mockImplementation(
    () =>
      new Promise((resolve) => {
        finish = resolve;
      }),
  );
  render(<CatalogWorkspace />);
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
    screen.getByRole("button", { name: "Adicionar serviço" }),
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

test("erro na mudança de estado libera fechamento e nova tentativa sem consultar o serviço", async () => {
  vi.mocked(listOfferings).mockResolvedValue(page());
  vi.mocked(getOffering).mockResolvedValue({
    state: "success",
    status: 200,
    data: records[0],
  });
  vi.mocked(setOfferingStatus).mockResolvedValue({
    state: "error",
    status: 503,
    error: { code: "internal_error", message: "raw", request_id: "test" },
  });
  render(<CatalogWorkspace />);
  fireEvent.click(await screen.findByRole("button", { name: "Editar Corte" }));
  fireEvent.click(
    await screen.findByRole("button", { name: "Desativar Corte" }),
  );
  fireEvent.click(
    screen.getByRole("button", { name: "Confirmar desativação" }),
  );
  await screen.findByText("Não foi possível alterar o serviço");
  expect(
    screen.getByRole("button", { name: "Confirmar desativação" }),
  ).toBeEnabled();
  expect(screen.getByRole("button", { name: "Cancelar edição" })).toBeEnabled();
  expect(screen.getByRole("button", { name: "Fechar modal" })).toBeEnabled();
  expect(getOffering).toHaveBeenCalledOnce();
  expect(setOfferingStatus).toHaveBeenCalledOnce();
  fireEvent.click(screen.getByRole("button", { name: "Fechar modal" }));
  expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
});

test("lista oferece somente o lápis e a modal usa o estado recém-consultado para ativação", async () => {
  vi.mocked(listOfferings).mockResolvedValue(page());
  vi.mocked(getOffering).mockResolvedValue({
    state: "success",
    status: 200,
    data: { ...records[0], active: false },
  });
  render(<CatalogWorkspace />);
  const list = await screen.findByRole("list", {
    name: "Serviços cadastrados",
  });
  expect(
    within(list).queryByRole("button", { name: /Ativar|Desativar/ }),
  ).not.toBeInTheDocument();
  fireEvent.click(within(list).getByRole("button", { name: "Editar Corte" }));
  const modal = screen.getByRole("dialog", { name: "Editar serviço" });
  expect(
    await within(modal).findByRole("button", { name: "Ativar Corte" }),
  ).toBeEnabled();
  expect(
    within(modal).getByRole("heading", { name: "Editar serviço" }),
  ).toHaveFocus();
  expect(
    within(modal).queryByRole("button", { name: "Desativar Corte" }),
  ).not.toBeInTheDocument();
});

test("salvamento do serviço bloqueia fechar e mudar estado até a resposta", async () => {
  vi.mocked(listOfferings).mockResolvedValue(page());
  vi.mocked(getOffering).mockResolvedValue({
    state: "success",
    status: 200,
    data: records[0],
  });
  let finish!: (value: Awaited<ReturnType<typeof updateOffering>>) => void;
  vi.mocked(updateOffering).mockImplementation(
    () =>
      new Promise((resolve) => {
        finish = resolve;
      }),
  );
  render(<CatalogWorkspace />);
  const origin = await screen.findByRole("button", { name: "Editar Corte" });
  origin.focus();
  fireEvent.click(origin);
  const modal = screen.getByRole("dialog", { name: "Editar serviço" });
  fireEvent.change(await within(modal).findByLabelText("Preço"), {
    target: { value: "2,00" },
  });
  fireEvent.click(
    within(modal).getByRole("button", { name: "Salvar alterações" }),
  );
  expect(
    within(modal).getByRole("button", { name: "Fechar modal" }),
  ).toBeDisabled();
  expect(
    within(modal).getByRole("button", { name: "Desativar Corte" }),
  ).toBeDisabled();
  fireEvent(modal, new Event("cancel", { cancelable: true }));
  expect(modal).toBeVisible();
  await act(async () =>
    finish({
      state: "success",
      status: 200,
      data: { ...records[0], price_cents: 200 },
    }),
  );
  fireEvent.click(within(modal).getByRole("button", { name: "Fechar modal" }));
  expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  expect(origin).toHaveFocus();
});

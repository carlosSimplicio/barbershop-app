import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";

import { useAuthSession } from "@/components/session/auth-provider";
import { listProfessionals } from "@/lib/professionals/professionals-api";

import { ProfessionalsListPanel } from "./professionals-list-panel";

vi.mock("@/components/session/auth-provider", () => ({
  useAuthSession: vi.fn(),
}));
vi.mock("@/lib/professionals/professionals-api", () => ({
  listProfessionals: vi.fn(),
}));

const records = [
  { id: "ana-id", name: "Ana Lima", email: "ana@example.test", active: true },
  {
    id: "bia-id",
    name: "Beatriz Souza",
    email: "bia@example.test",
    active: false,
  },
];

beforeEach(() => {
  vi.mocked(listProfessionals).mockReset();
  setOwnerSession();
});

test("mostra carregamento e os dados públicos da página recebida", async () => {
  let finish:
    | ((result: Awaited<ReturnType<typeof listProfessionals>>) => void)
    | undefined;
  vi.mocked(listProfessionals).mockImplementation(
    () =>
      new Promise((resolve) => {
        finish = resolve;
      }),
  );

  render(<ProfessionalsListPanel />);

  expect(screen.getByRole("status")).toHaveTextContent(
    "Carregando profissionais",
  );
  await waitFor(() => expect(finish).toBeDefined());
  await act(async () => finish?.(successPage(records)));

  const list = screen.getByRole("list", { name: "Profissionais cadastrados" });
  expect(list).toHaveTextContent("Ana Lima");
  expect(list).toHaveTextContent("ana@example.test");
  expect(list).toHaveTextContent("Ativo");
  expect(list).toHaveTextContent("Beatriz Souza");
  expect(list).toHaveTextContent("Inativo");
  expect(document.body).not.toHaveTextContent("password");
});

test("apresenta estado vazio quando a resposta não tem profissionais", async () => {
  vi.mocked(listProfessionals).mockResolvedValue(successPage([]));

  render(<ProfessionalsListPanel />);

  expect(
    await screen.findByText("Nenhum profissional cadastrado."),
  ).toHaveAttribute("role", "status");
  expect(screen.queryByRole("listitem")).not.toBeInTheDocument();
});

test("pagina a lista e impede avançar além da última página", async () => {
  vi.mocked(listProfessionals)
    .mockResolvedValueOnce(successPage(records, 0, true))
    .mockResolvedValueOnce(successPage([records[1]], 20, false));

  render(<ProfessionalsListPanel />);

  expect(await screen.findByRole("list")).toHaveTextContent("Ana Lima");
  fireEvent.click(screen.getByRole("button", { name: "Próxima" }));

  expect(screen.getByRole("status")).toHaveTextContent(
    "Carregando profissionais",
  );
  expect(await screen.findByText("Beatriz Souza")).toBeInTheDocument();
  expect(screen.queryByText("Ana Lima")).not.toBeInTheDocument();
  expect(listProfessionals).toHaveBeenNthCalledWith(1, {
    limit: 20,
    offset: 0,
    signal: expect.any(AbortSignal),
  });
  expect(listProfessionals).toHaveBeenNthCalledWith(2, {
    limit: 20,
    offset: 20,
    signal: expect.any(AbortSignal),
  });
  expect(screen.getByRole("button", { name: "Anterior" })).toBeEnabled();
  expect(screen.getByRole("button", { name: "Próxima" })).toBeDisabled();
});

test("mostra erro genérico e permite tentar novamente", async () => {
  vi.mocked(listProfessionals)
    .mockResolvedValueOnce({
      state: "error",
      status: 500,
      error: {
        code: "internal_error",
        message: "detalhe bruto do servidor",
        request_id: "test",
      },
    })
    .mockResolvedValueOnce(successPage(records));

  render(<ProfessionalsListPanel />);

  expect(
    await screen.findByText("Não foi possível consultar os profissionais"),
  ).toBeInTheDocument();
  expect(
    screen.queryByText("detalhe bruto do servidor"),
  ).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole("button", { name: "Tentar novamente" }));
  expect(await screen.findByRole("list")).toHaveTextContent("Ana Lima");
});

test("remove a lista quando a API encerra a sessão", async () => {
  vi.mocked(listProfessionals)
    .mockResolvedValueOnce(successPage(records, 0, true))
    .mockResolvedValueOnce({
      state: "error",
      status: 401,
      error: {
        code: "authentication_required",
        message: "Entre novamente.",
        request_id: "test",
      },
    });

  render(<ProfessionalsListPanel />);

  expect(await screen.findByRole("list")).toHaveTextContent("Ana Lima");
  fireEvent.click(screen.getByRole("button", { name: "Próxima" }));
  await waitFor(() => expect(listProfessionals).toHaveBeenCalledTimes(2));
  expect(screen.queryByRole("list")).not.toBeInTheDocument();
  expect(screen.queryByRole("status")).not.toBeInTheDocument();
});

test("descarta resposta antiga após trocar a identidade do proprietário", async () => {
  const pending: Array<{
    finish: (result: Awaited<ReturnType<typeof listProfessionals>>) => void;
    signal: AbortSignal;
  }> = [];
  vi.mocked(listProfessionals).mockImplementation(({ signal }) => {
    return new Promise((resolve) => {
      pending.push({ finish: resolve, signal: signal! });
    });
  });

  const view = render(<ProfessionalsListPanel />);
  await waitFor(() => expect(pending).toHaveLength(1));
  setOwnerSession({ user_id: "other-user", establishment_id: "other-shop" });
  view.rerender(<ProfessionalsListPanel />);
  await waitFor(() => expect(pending).toHaveLength(2));
  expect(pending[0].signal.aborted).toBe(true);

  await act(async () => pending[0].finish(successPage(records)));
  expect(screen.queryByText("Ana Lima")).not.toBeInTheDocument();
  await act(async () =>
    pending[1].finish(
      successPage([
        {
          id: "dora-id",
          name: "Dora Reis",
          email: "dora@example.test",
          active: true,
        },
      ]),
    ),
  );
  expect(await screen.findByText("Dora Reis")).toBeInTheDocument();
  expect(screen.queryByText("Ana Lima")).not.toBeInTheDocument();
});

function setOwnerSession(
  overrides: { user_id?: string; establishment_id?: string } = {},
) {
  vi.mocked(useAuthSession).mockReturnValue({
    state: {
      status: "authenticated",
      principal: {
        user_id: "owner-id",
        establishment_id: "shop-id",
        email: "owner@example.test",
        role: "owner",
        password_change_required: false,
        ...overrides,
      },
    },
    transitioning: false,
    reload: vi.fn(),
    runAction: vi.fn(),
  } as unknown as ReturnType<typeof useAuthSession>);
}

function successPage(items: typeof records, offset = 0, has_more = false) {
  return {
    state: "success" as const,
    status: 200,
    data: { items, limit: 20, offset, has_more },
  };
}

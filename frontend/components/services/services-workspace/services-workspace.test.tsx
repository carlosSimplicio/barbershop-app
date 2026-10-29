import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";
import { useAuthSession } from "@/components/session/auth-provider/auth-provider";
import { listServices, type ServiceRecord } from "@/lib/services/services-api";
import { ServicesWorkspace } from "./services-workspace";

vi.mock("@/components/session/auth-provider/auth-provider", () => ({
  useAuthSession: vi.fn(),
}));
vi.mock("@/lib/services/services-api", () => ({ listServices: vi.fn() }));
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
function page(items = records, offset = 0, has_more = false, can_write = true) {
  return {
    state: "success" as const,
    status: 200,
    data: { items, limit: 20, offset, has_more, can_write },
  };
}
beforeEach(() => {
  vi.mocked(listServices).mockReset();
  owner();
});
test("exibe preço exato, duração, estado e paginação", async () => {
  vi.mocked(listServices)
    .mockResolvedValueOnce(page(records, 0, true))
    .mockResolvedValueOnce(page([records[1]], 20));
  render(<ServicesWorkspace />);
  expect(screen.getByRole("status")).toHaveTextContent("Carregando serviços");
  expect(
    await screen.findByRole("list", { name: "Serviços cadastrados" }),
  ).toHaveTextContent("Corte");
  expect(screen.getByText(/R\$\s0,01/)).toBeVisible();
  expect(screen.getByText(/R\$\s99.999,99/)).toBeVisible();
  expect(screen.getByText(/17 minutos/)).toBeVisible();
  expect(screen.getByText("Inativo")).toBeVisible();
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

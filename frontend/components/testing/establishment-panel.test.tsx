import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";

import { useAuthSession } from "@/components/session/auth-provider";
import { getEstablishment } from "@/lib/establishment/establishment-api";

import { EstablishmentPanel } from "./establishment-panel";

vi.mock("@/components/session/auth-provider", () => ({
  useAuthSession: vi.fn(),
}));
vi.mock("@/lib/establishment/establishment-api", () => ({
  getEstablishment: vi.fn(),
}));

beforeEach(() => {
  vi.mocked(getEstablishment).mockReset();
  setSession();
});

test("mostra nome, slug, fuso e ausência de trial após consulta", async () => {
  vi.mocked(getEstablishment).mockResolvedValue({
    state: "success",
    status: 200,
    data: establishment(null),
  });
  render(<EstablishmentPanel />);
  fireEvent.click(
    screen.getByRole("button", { name: "Consultar estabelecimento" }),
  );

  expect(await screen.findByText("Estúdio Aurora")).toBeInTheDocument();
  expect(screen.getByText("estudio-aurora")).toBeInTheDocument();
  expect(screen.getByText("America/Sao_Paulo")).toBeInTheDocument();
  expect(screen.getByText("Sem período de avaliação")).toBeInTheDocument();
  expect(getEstablishment).toHaveBeenCalledWith(expect.any(AbortSignal));
});

test.each([
  ["active", "Ativo"],
  ["expired", "Expirado"],
] as const)("mostra o estado do trial %s", async (status, label) => {
  vi.mocked(getEstablishment).mockResolvedValue({
    state: "success",
    status: 200,
    data: establishment({
      status,
      starts_at: "2026-10-01T12:00:00Z",
      ends_at: "2026-10-15T12:00:00Z",
    }),
  });
  render(<EstablishmentPanel />);
  fireEvent.click(
    screen.getByRole("button", { name: "Consultar estabelecimento" }),
  );

  expect(await screen.findByText(label)).toBeInTheDocument();
  expect(document.body).not.toHaveTextContent("2026-10-01T12:00:00Z");
});

test("mostra erro genérico e permite nova tentativa", async () => {
  vi.mocked(getEstablishment)
    .mockResolvedValueOnce({
      state: "error",
      status: 500,
      error: {
        code: "internal_error",
        message: "detalhe bruto",
        request_id: "test",
      },
    })
    .mockResolvedValueOnce({
      state: "success",
      status: 200,
      data: establishment(null),
    });
  render(<EstablishmentPanel />);
  fireEvent.click(
    screen.getByRole("button", { name: "Consultar estabelecimento" }),
  );

  expect(
    await screen.findByText(/não foi possível consultar o estabelecimento/i),
  ).toBeInTheDocument();
  expect(screen.queryByText("detalhe bruto")).not.toBeInTheDocument();
  fireEvent.click(
    screen.getByRole("button", { name: "Consultar estabelecimento" }),
  );
  expect(await screen.findByText("Estúdio Aurora")).toBeInTheDocument();
});

test("bloqueia a consulta durante a troca obrigatória de senha", () => {
  setSession({ password_change_required: true });
  render(<EstablishmentPanel />);

  expect(
    screen.getByRole("button", { name: "Consultar estabelecimento" }),
  ).toBeDisabled();
  expect(
    screen.getByText(/troque sua senha para consultar/i),
  ).toBeInTheDocument();
  expect(getEstablishment).not.toHaveBeenCalled();
});

test("descarta resposta antiga quando a identidade da sessão muda", async () => {
  let finish:
    | ((result: Awaited<ReturnType<typeof getEstablishment>>) => void)
    | undefined;
  vi.mocked(getEstablishment).mockImplementation(
    () =>
      new Promise((resolve) => {
        finish = resolve;
      }),
  );
  const view = render(<EstablishmentPanel />);
  fireEvent.click(
    screen.getByRole("button", { name: "Consultar estabelecimento" }),
  );
  const signal = vi.mocked(getEstablishment).mock.calls[0][0];

  setSession({
    user_id: "other-user",
    establishment_id: "other-establishment",
  });
  view.rerender(<EstablishmentPanel />);
  await waitFor(() => expect(signal?.aborted).toBe(true));
  await act(async () => {
    finish?.({ state: "success", status: 200, data: establishment(null) });
  });

  expect(screen.queryByText("Estúdio Aurora")).not.toBeInTheDocument();
  expect(
    screen.getByRole("button", { name: "Consultar estabelecimento" }),
  ).toBeEnabled();
});

function setSession(overrides: Record<string, unknown> = {}) {
  vi.mocked(useAuthSession).mockReturnValue({
    state: {
      status: "authenticated",
      principal: {
        user_id: "user-id",
        establishment_id: "establishment-id",
        email: "private@example.test",
        role: "owner",
        password_change_required: false,
        ...overrides,
      },
    },
    reload: vi.fn(),
    runAction: vi.fn(),
  } as unknown as ReturnType<typeof useAuthSession>);
}

function establishment(
  trial: {
    status: "active" | "expired";
    starts_at: string;
    ends_at: string;
  } | null,
) {
  return {
    id: "establishment-id",
    name: "Estúdio Aurora",
    slug: "estudio-aurora",
    time_zone: "America/Sao_Paulo",
    trial,
  };
}

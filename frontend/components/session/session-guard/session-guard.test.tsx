import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";

import { configureApiSessionObserver } from "@/lib/api/client";

import { AuthProvider } from "../auth-provider/auth-provider";
import { SessionGuard } from "./session-guard";

const replace = vi.hoisted(() => vi.fn());
vi.mock("next/navigation", () => ({ useRouter: () => ({ replace }) }));

afterEach(() => {
  configureApiSessionObserver(null);
  vi.restoreAllMocks();
  replace.mockReset();
});

test("não mostra conteúdo protegido antes de confirmar a sessão", async () => {
  let finish: ((response: Response) => void) | undefined;
  vi.spyOn(globalThis, "fetch").mockImplementation(
    () =>
      new Promise<Response>((resolve) => {
        finish = resolve;
      }),
  );

  render(
    <AuthProvider>
      <SessionGuard require="authenticated">Conteúdo protegido</SessionGuard>
    </AuthProvider>,
  );
  expect(screen.queryByText("Conteúdo protegido")).not.toBeInTheDocument();
  expect(screen.getByRole("status")).toHaveTextContent("Carregando sessão");

  await waitFor(() => expect(finish).toBeDefined());
  finish?.(jsonResponse());
  expect(await screen.findByText("Conteúdo protegido")).toBeInTheDocument();
  expect(replace).not.toHaveBeenCalled();
});

test("envia visitante sem sessão do painel de proprietário para login", async () => {
  vi.spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(errorResponse(401))
    .mockResolvedValueOnce(errorResponse(401));

  render(
    <AuthProvider>
      <SessionGuard require="owner">Conteúdo protegido</SessionGuard>
    </AuthProvider>,
  );

  await waitFor(() => expect(replace).toHaveBeenCalledWith("/login"));
  expect(screen.queryByText("Conteúdo protegido")).not.toBeInTheDocument();
});

test("envia proprietário autenticado do login para a entrada do painel", async () => {
  vi.spyOn(globalThis, "fetch").mockResolvedValue(jsonResponse());

  render(
    <AuthProvider>
      <SessionGuard require="anonymous">Formulário de login</SessionGuard>
    </AuthProvider>,
  );

  await waitFor(() => expect(replace).toHaveBeenCalledWith("/painel"));
  expect(screen.queryByText("Formulário de login")).not.toBeInTheDocument();
});

test("permite proprietário na rota protegida do painel", async () => {
  vi.spyOn(globalThis, "fetch").mockResolvedValue(jsonResponse());

  render(
    <AuthProvider>
      <SessionGuard require="owner">Painel do proprietário</SessionGuard>
    </AuthProvider>,
  );

  expect(await screen.findByText("Painel do proprietário")).toBeInTheDocument();
  expect(replace).not.toHaveBeenCalled();
});

test("redireciona profissional da rota de proprietário para /testes", async () => {
  vi.spyOn(globalThis, "fetch").mockResolvedValue(jsonResponse("professional"));

  render(
    <AuthProvider>
      <SessionGuard require="owner">Painel do proprietário</SessionGuard>
    </AuthProvider>,
  );

  await waitFor(() => expect(replace).toHaveBeenCalledWith("/testes"));
  expect(screen.queryByText("Painel do proprietário")).not.toBeInTheDocument();
});

test("mostra formulário público quando a sessão está ausente", async () => {
  vi.spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(errorResponse(401))
    .mockResolvedValueOnce(errorResponse(401));

  render(
    <AuthProvider>
      <SessionGuard require="anonymous">Formulário de login</SessionGuard>
    </AuthProvider>,
  );

  expect(await screen.findByText("Formulário de login")).toBeInTheDocument();
  expect(replace).not.toHaveBeenCalled();
});

test("erro temporário permite tentar novamente sem redirecionar", async () => {
  vi.spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(errorResponse(503))
    .mockResolvedValueOnce(jsonResponse());

  render(
    <AuthProvider>
      <SessionGuard require="authenticated">Conteúdo protegido</SessionGuard>
    </AuthProvider>,
  );

  fireEvent.click(
    await screen.findByRole("button", { name: "Tentar novamente" }),
  );
  expect(await screen.findByText("Conteúdo protegido")).toBeInTheDocument();
  expect(replace).not.toHaveBeenCalled();
});

function jsonResponse(role: "owner" | "professional" = "owner") {
  return new Response(
    JSON.stringify({
      user_id: "user-id",
      establishment_id: "establishment-id",
      email: "private@example.test",
      role,
      password_change_required: false,
    }),
    { headers: { "Content-Type": "application/json" }, status: 200 },
  );
}

function errorResponse(status: number) {
  return new Response(
    JSON.stringify({
      code: "authentication_required",
      message: "Entre novamente.",
      request_id: "id",
    }),
    { headers: { "Content-Type": "application/json" }, status },
  );
}

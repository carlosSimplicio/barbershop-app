import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";

import { AuthProvider } from "@/components/session/auth-provider";
import {
  configureApiSessionRecovery,
  runSessionAction,
} from "@/lib/api/client";

import TestingPage from "./page";

const replace = vi.hoisted(() => vi.fn());
vi.mock("next/navigation", () => ({ useRouter: () => ({ replace }) }));

beforeEach(() => {
  replace.mockReset();
  vi.spyOn(document, "cookie", "get").mockReturnValue("");
});

afterEach(async () => {
  configureApiSessionRecovery(null);
  await runSessionAction(
    async () => ({ state: "empty", status: 204 }),
    "authenticated",
  );
  vi.restoreAllMocks();
});

test("mostra apenas dados não sensíveis do principal autenticado", async () => {
  vi.spyOn(globalThis, "fetch").mockResolvedValue(sessionResponse());
  renderPage();

  expect(await screen.findByText("Proprietário")).toBeInTheDocument();
  expect(
    screen.getByRole("heading", { name: "Área de testes" }),
  ).toBeInTheDocument();
  expect(
    screen.getByRole("heading", { name: "Estabelecimento" }),
  ).toBeInTheDocument();
  expect(screen.getByText("user-id")).toBeInTheDocument();
  expect(screen.getByText("establishment-id")).toBeInTheDocument();
  expect(screen.getByText("Não")).toBeInTheDocument();
  expect(document.body).not.toHaveTextContent("private@example.test");
  expect(document.body).not.toHaveTextContent("token");
  expect(replace).not.toHaveBeenCalled();
});

test("consulta /auth/me novamente e atualiza a identidade exibida", async () => {
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(sessionResponse())
    .mockResolvedValueOnce(sessionResponse("professional", true));
  renderPage();
  await screen.findByText("Proprietário");

  fireEvent.click(
    screen.getByRole("button", { name: "Consultar sessão novamente" }),
  );

  expect(await screen.findByText("Profissional")).toBeInTheDocument();
  expect(screen.getByText("Sim")).toBeInTheDocument();
  expect(fetchMock.mock.calls.map(([path]) => path)).toEqual([
    "/api/v1/auth/me",
    "/api/v1/auth/me",
  ]);
});

test("falha da nova consulta mostra erro recuperável sem expor a identidade", async () => {
  vi.spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(sessionResponse())
    .mockResolvedValueOnce(errorResponse(503))
    .mockResolvedValueOnce(sessionResponse());
  renderPage();
  await screen.findByText("Proprietário");

  fireEvent.click(
    screen.getByRole("button", { name: "Consultar sessão novamente" }),
  );

  expect(
    await screen.findByText("Não foi possível verificar sua sessão"),
  ).toBeInTheDocument();
  expect(screen.queryByText("user-id")).not.toBeInTheDocument();
  expect(replace).not.toHaveBeenCalled();

  fireEvent.click(screen.getByRole("button", { name: "Tentar novamente" }));
  expect(await screen.findByText("Proprietário")).toBeInTheDocument();
});

test("sessão perdida na nova consulta oculta a página e substitui a rota", async () => {
  vi.spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(sessionResponse())
    .mockResolvedValueOnce(errorResponse(401));
  renderPage();
  await screen.findByText("Proprietário");

  fireEvent.click(
    screen.getByRole("button", { name: "Consultar sessão novamente" }),
  );

  await waitFor(() => expect(replace).toHaveBeenCalledWith("/login"));
  expect(screen.queryByText("user-id")).not.toBeInTheDocument();
});

test("renova a sessão com cookies e CSRF e reconsulta o principal", async () => {
  vi.spyOn(document, "cookie", "get").mockReturnValue(
    "__Secure-barbershop_csrf=csrf-current",
  );
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(sessionResponse())
    .mockResolvedValueOnce(new Response(null, { status: 204 }))
    .mockResolvedValueOnce(sessionResponse("professional"));
  renderPage();
  await screen.findByText("Proprietário");

  fireEvent.click(screen.getByRole("button", { name: "Renovar sessão" }));

  expect(await screen.findByText("Profissional")).toBeInTheDocument();
  expect(fetchMock.mock.calls.map(([path]) => path)).toEqual([
    "/api/v1/auth/me",
    "/api/v1/auth/refresh",
    "/api/v1/auth/me",
  ]);
  const refreshOptions = fetchMock.mock.calls[1]?.[1];
  expect(refreshOptions).toMatchObject({
    method: "POST",
    credentials: "include",
  });
  expect(new Headers(refreshOptions?.headers).get("X-CSRF-Token")).toBe(
    "csrf-current",
  );
  expect(refreshOptions?.body).toBeUndefined();
  expect(document.body).not.toHaveTextContent("csrf-current");
});

test("dois cliques enquanto renova enviam apenas um POST", async () => {
  let finishRefresh: ((response: Response) => void) | undefined;
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(sessionResponse())
    .mockImplementationOnce(
      () =>
        new Promise<Response>((resolve) => {
          finishRefresh = resolve;
        }),
    )
    .mockResolvedValueOnce(sessionResponse());
  renderPage();
  await screen.findByText("Proprietário");

  const button = screen.getByRole("button", { name: "Renovar sessão" });
  fireEvent.click(button);
  fireEvent.click(button);
  await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));
  expect(screen.getByRole("button", { name: "Renovando…" })).toBeDisabled();
  finishRefresh?.(new Response(null, { status: 204 }));
  await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(3));
});

test("consulta da sessão aguarda o refresh manual antes de decidir que ela acabou", async () => {
  let finishRefresh: ((response: Response) => void) | undefined;
  let refreshed = false;
  let initialRead = true;
  const fetchMock = vi.spyOn(globalThis, "fetch").mockImplementation((path) => {
    if (path === "/api/v1/auth/refresh") {
      return new Promise<Response>((resolve) => {
        finishRefresh = (response) => {
          refreshed = true;
          resolve(response);
        };
      });
    }
    if (initialRead) {
      initialRead = false;
      return Promise.resolve(sessionResponse());
    }
    return Promise.resolve(refreshed ? sessionResponse() : errorResponse(401));
  });
  renderPage();
  await screen.findByText("Proprietário");

  fireEvent.click(screen.getByRole("button", { name: "Renovar sessão" }));
  await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));
  expect(
    screen.getByRole("button", { name: "Consultar sessão novamente" }),
  ).toBeDisabled();
  expect(
    screen.getByRole("button", { name: "Consultar estabelecimento" }),
  ).toBeDisabled();
  fireEvent.click(
    screen.getByRole("button", { name: "Consultar sessão novamente" }),
  );

  expect(fetchMock).toHaveBeenCalledTimes(2);
  finishRefresh?.(new Response(null, { status: 204 }));
  await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(3));
  expect(await screen.findByText("Proprietário")).toBeInTheDocument();
  expect(replace).not.toHaveBeenCalled();
});

test("erro de renovação preserva a sessão e permite nova tentativa segura", async () => {
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(sessionResponse())
    .mockResolvedValueOnce(errorResponse(403))
    .mockResolvedValueOnce(new Response(null, { status: 204 }))
    .mockResolvedValueOnce(sessionResponse());
  renderPage();
  await screen.findByText("Proprietário");

  fireEvent.click(screen.getByRole("button", { name: "Renovar sessão" }));
  expect(
    await screen.findByText("Não foi possível renovar a sessão"),
  ).toBeInTheDocument();
  expect(screen.getByText("user-id")).toBeInTheDocument();
  expect(replace).not.toHaveBeenCalled();

  fireEvent.click(screen.getByRole("button", { name: "Renovar sessão" }));
  await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(4));
  expect(
    screen.queryByText("Não foi possível renovar a sessão"),
  ).not.toBeInTheDocument();
});

test("falha incerta preserva o principal sem reutilizar o refresh token", async () => {
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(sessionResponse())
    .mockResolvedValueOnce(errorResponse(503));
  renderPage();
  await screen.findByText("Proprietário");

  fireEvent.click(screen.getByRole("button", { name: "Renovar sessão" }));

  expect(await screen.findByText("Renovação incerta")).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Renovar sessão" })).toBeDisabled();
  expect(screen.getByText("user-id")).toBeInTheDocument();
  expect(fetchMock).toHaveBeenCalledTimes(2);
});

test("refresh inválido remove a identidade e envia para login", async () => {
  vi.spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(sessionResponse())
    .mockResolvedValueOnce(errorResponse(401));
  renderPage();
  await screen.findByText("Proprietário");

  fireEvent.click(screen.getByRole("button", { name: "Renovar sessão" }));

  await waitFor(() => expect(replace).toHaveBeenCalledWith("/login"));
  expect(screen.queryByText("user-id")).not.toBeInTheDocument();
});

function renderPage() {
  render(
    <AuthProvider>
      <TestingPage />
    </AuthProvider>,
  );
}

function sessionResponse(
  role: "owner" | "professional" = "owner",
  password_change_required = false,
) {
  return new Response(
    JSON.stringify({
      user_id: "user-id",
      establishment_id: "establishment-id",
      email: "private@example.test",
      role,
      password_change_required,
    }),
    { status: 200, headers: { "Content-Type": "application/json" } },
  );
}

function errorResponse(status: number) {
  return new Response(
    JSON.stringify({
      code: "session_error",
      message: "detalhe privado",
      request_id: "test",
    }),
    { status, headers: { "Content-Type": "application/json" } },
  );
}

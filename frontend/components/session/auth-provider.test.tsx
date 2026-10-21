import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";

import {
  configureApiSessionRecovery,
  runSessionAction,
} from "@/lib/api/client";

import { AuthProvider, useAuthSession } from "./auth-provider";

afterEach(async () => {
  configureApiSessionRecovery(null);
  await runSessionAction(
    async () => ({ state: "empty", status: 204 }),
    "authenticated",
  );
  vi.restoreAllMocks();
});

test("descobre a identidade autenticada ao montar", async () => {
  vi.spyOn(globalThis, "fetch").mockResolvedValue(
    jsonResponse(principal("owner")),
  );

  render(
    <AuthProvider>
      <SessionProbe />
    </AuthProvider>,
  );

  expect(await screen.findByText("owner")).toBeInTheDocument();
});

test("recupera a identidade após expirar o access token", async () => {
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(errorResponse(401))
    .mockResolvedValueOnce(new Response(null, { status: 204 }))
    .mockResolvedValueOnce(jsonResponse(principal("professional")));

  render(
    <AuthProvider>
      <SessionProbe />
    </AuthProvider>,
  );

  expect(await screen.findByText("professional")).toBeInTheDocument();
  expect(fetchMock.mock.calls.map(([path]) => path)).toEqual([
    "/api/v1/auth/me",
    "/api/v1/auth/refresh",
    "/api/v1/auth/me",
  ]);
});

test("distingue sessão ausente de erro temporário", async () => {
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(errorResponse(401))
    .mockResolvedValueOnce(errorResponse(401));
  const first = render(
    <AuthProvider>
      <SessionProbe />
    </AuthProvider>,
  );
  expect(await screen.findByText("anonymous")).toBeInTheDocument();
  first.unmount();

  fetchMock.mockResolvedValueOnce(errorResponse(503));
  render(
    <AuthProvider>
      <SessionProbe />
    </AuthProvider>,
  );
  expect(await screen.findByText("unavailable")).toBeInTheDocument();
});

test("não declara logout quando a rotação tem resultado indeterminado", async () => {
  vi.spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(errorResponse(401))
    .mockResolvedValueOnce(errorResponse(503));

  render(
    <AuthProvider>
      <SessionProbe />
    </AuthProvider>,
  );

  expect(await screen.findByText("unavailable")).toBeInTheDocument();
});

test("remontagem após falha indeterminada não reutiliza o refresh token", async () => {
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(errorResponse(401))
    .mockResolvedValueOnce(errorResponse(503))
    .mockResolvedValueOnce(errorResponse(401));
  const first = render(
    <AuthProvider>
      <SessionProbe />
    </AuthProvider>,
  );
  expect(await screen.findByText("unavailable")).toBeInTheDocument();
  first.unmount();

  render(
    <AuthProvider>
      <SessionProbe />
    </AuthProvider>,
  );
  expect(await screen.findByText("unavailable")).toBeInTheDocument();
  expect(fetchMock).toHaveBeenCalledTimes(3);
});

test("ação falha durante a descoberta não deixa a sessão carregando", async () => {
  let finishInitialRead: ((response: Response) => void) | undefined;
  let finishAction:
    | ((result: {
        state: "error";
        status: 400;
        error: { code: string; message: string; request_id: string };
      }) => void)
    | undefined;
  const fetchMock = vi.spyOn(globalThis, "fetch").mockImplementationOnce(
    () =>
      new Promise<Response>((resolve) => {
        finishInitialRead = resolve;
      }),
  );

  render(
    <AuthProvider>
      <PendingFailureProbe
        action={() =>
          new Promise((resolve) => {
            finishAction = resolve;
          })
        }
      />
    </AuthProvider>,
  );
  await waitFor(() => expect(fetchMock).toHaveBeenCalledOnce());
  fireEvent.click(screen.getByRole("button", { name: "Falhar" }));
  await waitFor(() => expect(finishAction).toBeDefined());
  finishInitialRead?.(errorResponse(401));
  expect(await screen.findByText("anonymous")).toBeInTheDocument();
  finishAction?.({
    state: "error",
    status: 400,
    error: {
      code: "invalid_token",
      message: "Token inválido.",
      request_id: "id",
    },
  });
  expect(screen.getByText("anonymous")).toBeInTheDocument();
});

test("uma nova sessão reabilita a recuperação e atualiza o principal", async () => {
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(errorResponse(401))
    .mockResolvedValueOnce(errorResponse(401))
    .mockResolvedValueOnce(jsonResponse(principal("owner")));

  render(
    <AuthProvider>
      <SessionProbe />
    </AuthProvider>,
  );
  expect(await screen.findByText("anonymous")).toBeInTheDocument();
  fireEvent.click(screen.getByRole("button", { name: "Entrar" }));

  expect(await screen.findByText("owner")).toBeInTheDocument();
  expect(fetchMock).toHaveBeenCalledTimes(3);
});

test("logout bem-sucedido limpa a identidade", async () => {
  vi.spyOn(globalThis, "fetch").mockResolvedValue(
    jsonResponse(principal("owner")),
  );
  render(
    <AuthProvider>
      <SessionProbe />
    </AuthProvider>,
  );
  expect(await screen.findByText("owner")).toBeInTheDocument();

  fireEvent.click(screen.getByRole("button", { name: "Sair" }));
  await waitFor(() =>
    expect(screen.getByText("anonymous")).toBeInTheDocument(),
  );
});

test("logout não deixa resposta antiga restaurar a identidade", async () => {
  let finishInitialRead: ((response: Response) => void) | undefined;
  const fetchMock = vi.spyOn(globalThis, "fetch").mockImplementation(
    () =>
      new Promise<Response>((resolve) => {
        finishInitialRead = resolve;
      }),
  );
  render(
    <AuthProvider>
      <SessionProbe />
    </AuthProvider>,
  );
  await waitFor(() => expect(fetchMock).toHaveBeenCalledOnce());
  fireEvent.click(screen.getByRole("button", { name: "Sair" }));
  expect(await screen.findByText("anonymous")).toBeInTheDocument();

  await act(async () => {
    finishInitialRead?.(jsonResponse(principal("owner")));
  });
  expect(screen.getByText("anonymous")).toBeInTheDocument();
  expect(screen.queryByText("owner")).not.toBeInTheDocument();
});

function SessionProbe() {
  const { state, runAction } = useAuthSession();
  return (
    <div>
      <output>
        {state.status === "authenticated" ? state.principal.role : state.status}
      </output>
      <button
        onClick={() =>
          void runAction(
            async () => ({ state: "empty", status: 204 }),
            "authenticated",
          )
        }
      >
        Entrar
      </button>
      <button
        onClick={() =>
          void runAction(
            async () => ({ state: "empty", status: 204 }),
            "anonymous",
          )
        }
      >
        Sair
      </button>
    </div>
  );
}

function PendingFailureProbe({
  action,
}: {
  action: () => Promise<{
    state: "error";
    status: 400;
    error: { code: string; message: string; request_id: string };
  }>;
}) {
  const { state, runAction } = useAuthSession();
  return (
    <div>
      <output>{state.status}</output>
      <button onClick={() => void runAction(action, "authenticated")}>
        Falhar
      </button>
    </div>
  );
}

function principal(role: "owner" | "professional") {
  return {
    user_id: "user-id",
    establishment_id: "establishment-id",
    email: "private@example.test",
    role,
    password_change_required: false,
  };
}

function jsonResponse(body: unknown) {
  return new Response(JSON.stringify(body), {
    headers: { "Content-Type": "application/json" },
    status: 200,
  });
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

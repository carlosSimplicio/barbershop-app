import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";

import { AuthProvider } from "@/components/session/auth-provider";
import { configureApiSessionObserver } from "@/lib/api/client";

import TestingPage from "./page";

const replace = vi.hoisted(() => vi.fn());
vi.mock("next/navigation", () => ({ useRouter: () => ({ replace }) }));

beforeEach(() => {
  replace.mockReset();
  vi.spyOn(document, "cookie", "get").mockReturnValue("");
});

afterEach(() => {
  configureApiSessionObserver(null);
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
  expect(
    screen.queryByRole("heading", { name: "Renovação da sessão" }),
  ).not.toBeInTheDocument();
  expect(document.body).not.toHaveTextContent("private@example.test");
  expect(document.body).not.toHaveTextContent("token");
  expect(
    screen.queryByRole("heading", { name: "Trocar senha temporária" }),
  ).not.toBeInTheDocument();
  expect(replace).not.toHaveBeenCalled();
});

test("mostra a troca obrigatória apenas para o profissional com senha temporária", async () => {
  vi.spyOn(globalThis, "fetch").mockResolvedValue(
    sessionResponse("professional", true),
  );
  renderPage();

  expect(
    await screen.findByRole("heading", { name: "Trocar senha temporária" }),
  ).toBeInTheDocument();
  expect(screen.getByLabelText("Nova senha")).toHaveAttribute(
    "autocomplete",
    "new-password",
  );
  expect(
    screen.getByRole("button", { name: "Consultar estabelecimento" }),
  ).toBeDisabled();
  expect(screen.getByRole("button", { name: "Sair da sessão" })).toBeEnabled();
});

test.each([
  ["", "Informe uma nova senha."],
  ["curta", "Use de 8 a 32 caracteres, sem espaços."],
  ["senha com espaço", "Use de 8 a 32 caracteres, sem espaços."],
] as const)(
  "valida a nova senha %s antes de enviar e foca o campo inválido",
  async (password, message) => {
    const fetchMock = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValue(sessionResponse("professional", true));
    renderPage();
    await screen.findByRole("heading", { name: "Trocar senha temporária" });

    fireEvent.change(screen.getByLabelText("Nova senha"), {
      target: { value: password },
    });
    fireEvent.click(screen.getByRole("button", { name: "Trocar senha" }));

    expect(screen.getByText(message)).toBeInTheDocument();
    expect(screen.getByLabelText("Nova senha")).toHaveFocus();
    expect(fetchMock).toHaveBeenCalledOnce();
  },
);

test.each([
  ["invalid_password", "Use de 8 a 32 caracteres, sem espaços."],
  ["password_unchanged", "Escolha uma senha diferente da temporária."],
] as const)("mostra o erro %s no campo de senha", async (code, message) => {
  vi.spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(sessionResponse("professional", true))
    .mockResolvedValueOnce(passwordFieldError(code));
  renderPage();
  await screen.findByRole("heading", { name: "Trocar senha temporária" });

  fireEvent.change(screen.getByLabelText("Nova senha"), {
    target: { value: "nova-senha-segura" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Trocar senha" }));

  expect(await screen.findByText(message)).toBeInTheDocument();
  expect(screen.getByLabelText("Nova senha")).toHaveFocus();
  expect(document.body).not.toHaveTextContent("detalhe privado");
});

test("troca a senha, reconsulta o principal e libera o estabelecimento", async () => {
  vi.spyOn(document, "cookie", "get").mockReturnValue(
    "__Secure-barbershop_csrf=csrf-current",
  );
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(sessionResponse("professional", true))
    .mockResolvedValueOnce(new Response(null, { status: 204 }))
    .mockResolvedValueOnce(sessionResponse("professional", false));
  renderPage();
  await screen.findByRole("heading", { name: "Trocar senha temporária" });

  fireEvent.change(screen.getByLabelText("Nova senha"), {
    target: { value: "nova-senha-segura" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Trocar senha" }));

  await waitFor(() =>
    expect(
      screen.getByRole("button", { name: "Consultar estabelecimento" }),
    ).toBeEnabled(),
  );
  expect(screen.queryByLabelText("Nova senha")).not.toBeInTheDocument();
  expect(screen.getByText("Não")).toBeInTheDocument();
  expect(
    screen.queryByRole("heading", { name: "Renovação da sessão" }),
  ).not.toBeInTheDocument();
  expect(fetchMock.mock.calls.map(([path]) => path)).toEqual([
    "/api/v1/auth/me",
    "/api/v1/auth/password/change",
    "/api/v1/auth/me",
  ]);
  const request = fetchMock.mock.calls[1]?.[1];
  expect(request).toMatchObject({ method: "POST", credentials: "include" });
  expect(new Headers(request?.headers).get("X-CSRF-Token")).toBe(
    "csrf-current",
  );
  expect(JSON.parse(String(request?.body))).toEqual({
    new_password: "nova-senha-segura",
  });
  expect(document.body).not.toHaveTextContent("nova-senha-segura");
});

test("falha geral mantém a troca disponível para uma nova tentativa", async () => {
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(sessionResponse("professional", true))
    .mockResolvedValueOnce(errorResponse(500))
    .mockResolvedValueOnce(new Response(null, { status: 204 }))
    .mockResolvedValueOnce(sessionResponse("professional", false));
  renderPage();
  await screen.findByRole("heading", { name: "Trocar senha temporária" });

  fireEvent.change(screen.getByLabelText("Nova senha"), {
    target: { value: "nova-senha-segura" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Trocar senha" }));

  expect(
    await screen.findByText("Não foi possível trocar a senha"),
  ).toBeInTheDocument();
  expect(screen.getByLabelText("Nova senha")).toBeInTheDocument();
  expect(
    screen.getByRole("button", { name: "Consultar estabelecimento" }),
  ).toBeDisabled();
  expect(document.body).not.toHaveTextContent("detalhe privado");

  fireEvent.click(screen.getByRole("button", { name: "Trocar senha" }));
  await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(4));
});

test("401 na troca encerra a sessão sem repetir a mutação", async () => {
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(sessionResponse("professional", true))
    .mockResolvedValueOnce(errorResponse(401));
  renderPage();
  await screen.findByRole("heading", { name: "Trocar senha temporária" });
  fireEvent.change(screen.getByLabelText("Nova senha"), {
    target: { value: "nova-senha-segura" },
  });

  fireEvent.click(screen.getByRole("button", { name: "Trocar senha" }));

  await waitFor(() => expect(replace).toHaveBeenCalledWith("/login"));
  expect(
    screen.queryByRole("heading", { name: "Trocar senha temporária" }),
  ).not.toBeInTheDocument();
  expect(fetchMock.mock.calls.map(([path]) => path)).toEqual([
    "/api/v1/auth/me",
    "/api/v1/auth/password/change",
  ]);
});

test("cliques repetidos não duplicam a troca de senha", async () => {
  let finish: ((response: Response) => void) | undefined;
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(sessionResponse("professional", true))
    .mockImplementationOnce(
      () =>
        new Promise<Response>((resolve) => {
          finish = resolve;
        }),
    )
    .mockResolvedValueOnce(sessionResponse("professional", false));
  renderPage();
  await screen.findByRole("heading", { name: "Trocar senha temporária" });
  fireEvent.change(screen.getByLabelText("Nova senha"), {
    target: { value: "nova-senha-segura" },
  });

  const button = screen.getByRole("button", { name: "Trocar senha" });
  fireEvent.click(button);
  fireEvent.click(button);
  await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));
  expect(
    screen.getByRole("button", { name: "Trocando senha…" }),
  ).toBeDisabled();
  finish?.(new Response(null, { status: 204 }));
  await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(3));
});

test("não consulta a sessão durante a troca de senha", async () => {
  let finishChange: ((response: Response) => void) | undefined;
  let changed = false;
  let initialRead = true;
  const fetchMock = vi.spyOn(globalThis, "fetch").mockImplementation((path) => {
    if (path === "/api/v1/auth/password/change") {
      return new Promise<Response>((resolve) => {
        finishChange = (response) => {
          changed = true;
          resolve(response);
        };
      });
    }
    if (path === "/api/v1/auth/me") {
      if (initialRead) {
        initialRead = false;
        return Promise.resolve(sessionResponse("professional", true));
      }
      return Promise.resolve(
        changed ? sessionResponse("professional", false) : errorResponse(401),
      );
    }
    return Promise.resolve(errorResponse(500));
  });
  renderPage();
  await screen.findByRole("heading", { name: "Trocar senha temporária" });
  fireEvent.change(screen.getByLabelText("Nova senha"), {
    target: { value: "nova-senha-segura" },
  });

  fireEvent.click(screen.getByRole("button", { name: "Trocar senha" }));
  await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));
  expect(
    screen.getByRole("button", { name: "Consultar sessão novamente" }),
  ).toBeDisabled();
  fireEvent.click(
    screen.getByRole("button", { name: "Consultar sessão novamente" }),
  );
  expect(fetchMock).toHaveBeenCalledTimes(2);
  finishChange?.(new Response(null, { status: 204 }));

  await waitFor(() =>
    expect(
      screen.getByRole("button", { name: "Consultar estabelecimento" }),
    ).toBeEnabled(),
  );
  expect(replace).not.toHaveBeenCalled();
  expect(fetchMock.mock.calls.map(([path]) => path)).toEqual([
    "/api/v1/auth/me",
    "/api/v1/auth/password/change",
    "/api/v1/auth/me",
  ]);
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

test("logout 204 limpa o principal e substitui a rota sem nova consulta", async () => {
  vi.spyOn(document, "cookie", "get").mockReturnValue(
    "__Secure-barbershop_csrf=csrf-current",
  );
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(sessionResponse())
    .mockResolvedValueOnce(new Response(null, { status: 204 }));
  renderPage();
  await screen.findByText("Proprietário");

  fireEvent.click(screen.getByRole("button", { name: "Sair da sessão" }));

  await waitFor(() => expect(replace).toHaveBeenCalledWith("/login"));
  expect(screen.queryByText("user-id")).not.toBeInTheDocument();
  expect(fetchMock.mock.calls.map(([path]) => path)).toEqual([
    "/api/v1/auth/me",
    "/api/v1/auth/logout",
  ]);
  const options = fetchMock.mock.calls[1]?.[1];
  expect(options).toMatchObject({ method: "POST", credentials: "include" });
  expect(new Headers(options?.headers).get("X-CSRF-Token")).toBe(
    "csrf-current",
  );
  expect(options?.body).toBeUndefined();
});

test("cliques duplicados não repetem o POST de logout", async () => {
  let finishLogout: ((response: Response) => void) | undefined;
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(sessionResponse())
    .mockImplementationOnce(
      () =>
        new Promise<Response>((resolve) => {
          finishLogout = resolve;
        }),
    );
  renderPage();
  await screen.findByText("Proprietário");

  const button = screen.getByRole("button", { name: "Sair da sessão" });
  fireEvent.click(button);
  fireEvent.click(button);
  await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));
  expect(screen.getByRole("button", { name: "Saindo…" })).toBeDisabled();
  finishLogout?.(new Response(null, { status: 204 }));
  await waitFor(() => expect(replace).toHaveBeenCalledWith("/login"));
  expect(fetchMock).toHaveBeenCalledTimes(2);
});

test("falha interna no logout preserva o principal e permite tentar novamente", async () => {
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(sessionResponse())
    .mockResolvedValueOnce(errorResponse(500))
    .mockResolvedValueOnce(new Response(null, { status: 204 }));
  renderPage();
  await screen.findByText("Proprietário");

  fireEvent.click(screen.getByRole("button", { name: "Sair da sessão" }));

  expect(await screen.findByText("Não foi possível sair")).toBeInTheDocument();
  expect(screen.getByText("user-id")).toBeInTheDocument();
  expect(replace).not.toHaveBeenCalled();
  expect(document.body).not.toHaveTextContent("detalhe privado");
  fireEvent.click(screen.getByRole("button", { name: "Sair da sessão" }));
  await waitFor(() => expect(replace).toHaveBeenCalledWith("/login"));
  expect(fetchMock).toHaveBeenCalledTimes(3);
});

test("resposta inesperada no logout não limpa a identidade", async () => {
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(sessionResponse())
    .mockResolvedValueOnce(new Response(null, { status: 205 }));
  renderPage();
  await screen.findByText("Proprietário");

  fireEvent.click(screen.getByRole("button", { name: "Sair da sessão" }));

  expect(await screen.findByText("Não foi possível sair")).toBeInTheDocument();
  expect(screen.getByText("user-id")).toBeInTheDocument();
  expect(replace).not.toHaveBeenCalled();
  expect(fetchMock).toHaveBeenCalledTimes(2);
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

function passwordFieldError(code: string) {
  return new Response(
    JSON.stringify({
      code: "invalid_request",
      message: "detalhe privado",
      request_id: "test",
      fields: [{ field: "new_password", code, message: "detalhe privado" }],
    }),
    { status: 400, headers: { "Content-Type": "application/json" } },
  );
}

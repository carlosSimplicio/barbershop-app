import { afterEach, expect, test, vi } from "vitest";

import { apiFetch, configureApiSessionObserver } from "./client";

afterEach(() => {
  configureApiSessionObserver(null);
  vi.restoreAllMocks();
});

test("401 em mutação autenticada encerra a sessão sem repetir a chamada", async () => {
  const onInvalidSession = vi.fn();
  configureApiSessionObserver({ onInvalidSession });
  const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(
    jsonResponse(
      {
        code: "authentication_required",
        message: "Entre novamente.",
        request_id: "id",
      },
      401,
    ),
  );

  await expect(
    apiFetch("/api/v1/auth/password/change", {
      csrf: true,
      method: "POST",
    }),
  ).resolves.toMatchObject({ state: "error", status: 401 });
  expect(fetchMock).toHaveBeenCalledOnce();
  expect(onInvalidSession).toHaveBeenCalledOnce();
});

test("401 em leitura autenticada encerra a sessão sem repetir a chamada", async () => {
  const onInvalidSession = vi.fn();
  configureApiSessionObserver({ onInvalidSession });
  const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(
    jsonResponse(
      {
        code: "authentication_required",
        message: "Entre novamente.",
        request_id: "id",
      },
      401,
    ),
  );

  await apiFetch("/api/v1/auth/me");

  expect(fetchMock).toHaveBeenCalledOnce();
  expect(onInvalidSession).toHaveBeenCalledOnce();
});

test("qualquer 401 encerra a sessão; 5xx não", async () => {
  const onInvalidSession = vi.fn();
  configureApiSessionObserver({ onInvalidSession });
  vi.spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(
      jsonResponse(
        { code: "invalid_credentials", message: "Erro.", request_id: "id" },
        401,
      ),
    )
    .mockResolvedValueOnce(
      jsonResponse(
        { code: "unavailable", message: "Erro.", request_id: "id" },
        503,
      ),
    );

  await apiFetch("/api/v1/auth/login", { method: "POST" });
  await apiFetch("/api/v1/auth/me");

  expect(onInvalidSession).toHaveBeenCalledOnce();
});

test("401 de leitura cancelada não encerra outra sessão", async () => {
  let finishOldRead: ((response: Response) => void) | undefined;
  vi.spyOn(globalThis, "fetch").mockImplementationOnce(
    () =>
      new Promise<Response>((resolve) => {
        finishOldRead = resolve;
      }),
  );
  const controller = new AbortController();
  const oldRead = apiFetch("/api/v1/auth/me", {
    signal: controller.signal,
  });
  const onInvalidSession = vi.fn();
  controller.abort();
  configureApiSessionObserver({ onInvalidSession });
  finishOldRead?.(
    jsonResponse(
      {
        code: "authentication_required",
        message: "Entre novamente.",
        request_id: "id",
      },
      401,
    ),
  );
  await oldRead;

  expect(onInvalidSession).not.toHaveBeenCalled();
});

test("representa resposta 202 sem corpo como vazia e preserva o status", async () => {
  vi.spyOn(globalThis, "fetch").mockResolvedValue(
    new Response(null, { status: 202 }),
  );

  await expect(
    apiFetch<void>("/api/v1/auth/register", { method: "POST" }),
  ).resolves.toEqual({ state: "empty", status: 202 });
});

test("preserva headers fornecidos e envia cookies", async () => {
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValue(new Response(null, { status: 204 }));

  await apiFetch<void>("/api/v1/auth/logout", {
    method: "POST",
    headers: new Headers({ "Content-Type": "application/json" }),
  });

  const options = fetchMock.mock.calls[0]?.[1];
  const headers = new Headers(options?.headers);
  expect(options?.credentials).toBe("include");
  expect(headers.get("Accept")).toBe("application/json");
  expect(headers.get("Content-Type")).toBe("application/json");
});

test("lê o CSRF atual apenas quando solicitado", async () => {
  vi.spyOn(document, "cookie", "get").mockReturnValue(
    "theme=dark; __Secure-barbershop_csrf=csrf-atual",
  );
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValue(new Response(null, { status: 204 }));

  await apiFetch<void>("/api/v1/auth/logout", {
    csrf: true,
    headers: { "X-CSRF-Token": "csrf-antigo" },
    method: "POST",
  });
  await apiFetch<void>("/api/v1/auth/login", { method: "POST" });

  expect(
    new Headers(fetchMock.mock.calls[0]?.[1]?.headers).get("X-CSRF-Token"),
  ).toBe("csrf-atual");
  expect(
    new Headers(fetchMock.mock.calls[1]?.[1]?.headers).has("X-CSRF-Token"),
  ).toBe(false);
});

test("conserva detalhes válidos de erro e rejeita formato inválido", async () => {
  const error = {
    code: "validation_failed",
    fields: [
      { code: "invalid_email", field: "email", message: "E-mail inválido." },
    ],
    message: "Verifique os campos.",
    request_id: "request-id",
  };
  vi.spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(jsonResponse(error, 400))
    .mockResolvedValueOnce(jsonResponse({ ...error, fields: "invalid" }, 400));

  await expect(apiFetch("/api/v1/auth/login")).resolves.toEqual({
    error,
    state: "error",
    status: 400,
  });
  await expect(apiFetch("/api/v1/auth/login")).resolves.toMatchObject({
    error: { code: "unexpected_response" },
    state: "error",
    status: 400,
  });
});

function jsonResponse(body: unknown, status: number) {
  return new Response(JSON.stringify(body), {
    headers: { "Content-Type": "application/json" },
    status,
  });
}

import { afterEach, expect, test, vi } from "vitest";

import { apiFetch, configureApiSessionRecovery } from "./client";

afterEach(() => {
  configureApiSessionRecovery(null);
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

test("não renova nem repete uma mutação autenticada que recebeu 401", async () => {
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(errorResponse(401));
  configureApiSessionRecovery({ onInvalidSession: vi.fn() });

  await expect(
    apiFetch("/api/v1/auth/password/change", {
      authenticated: true,
      csrf: true,
      method: "POST",
    }),
  ).resolves.toMatchObject({ state: "error", status: 401 });
  expect(fetchMock).toHaveBeenCalledOnce();
});

test("compartilha um refresh entre duas leituras autenticadas", async () => {
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(errorResponse(401))
    .mockResolvedValueOnce(errorResponse(401))
    .mockResolvedValueOnce(new Response(null, { status: 204 }))
    .mockResolvedValueOnce(jsonResponse({ user_id: "current-user" }))
    .mockResolvedValueOnce(jsonResponse({ value: "current" }));
  configureApiSessionRecovery({ onInvalidSession: vi.fn() });

  await expect(
    Promise.all([
      apiFetch("/api/v1/auth/me", { authenticated: true }),
      apiFetch("/api/v1/establishment", { authenticated: true }),
    ]),
  ).resolves.toEqual([
    expect.objectContaining({ state: "success", status: 200 }),
    expect.objectContaining({ state: "success", status: 200 }),
  ]);
  expect(fetchMock).toHaveBeenCalledTimes(5);
  expect(fetchMock.mock.calls[2]?.[0]).toBe("/api/v1/auth/refresh");
});

test("não repete leitura de domínio se outra aba pode ter trocado de conta", async () => {
  installWebLock();
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(errorResponse(401))
    .mockResolvedValueOnce(jsonResponse({ user_id: "another-user" }));
  configureApiSessionRecovery({ onInvalidSession: vi.fn() });

  await expect(
    apiFetch("/api/v1/establishment", { authenticated: true }),
  ).resolves.toMatchObject({ state: "error", status: 401 });
  expect(fetchMock).toHaveBeenCalledTimes(2);
});

test("reconsulta a identidade no Web Lock e evita uma segunda rotação", async () => {
  const requestLock = installWebLock();
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(errorResponse(401))
    .mockResolvedValueOnce(jsonResponse({ user_id: "current-user" }))
    .mockResolvedValueOnce(jsonResponse({ user_id: "current-user" }));
  configureApiSessionRecovery({ onInvalidSession: vi.fn() });

  await expect(
    apiFetch("/api/v1/auth/me", { authenticated: true }),
  ).resolves.toMatchObject({ state: "success", status: 200 });
  expect(requestLock).toHaveBeenCalledOnce();
  expect(fetchMock.mock.calls.map(([path]) => path)).toEqual([
    "/api/v1/auth/me",
    "/api/v1/auth/me",
    "/api/v1/auth/me",
  ]);
});

test("renova dentro do Web Lock se a identidade ainda responde 401", async () => {
  installWebLock();
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(errorResponse(401))
    .mockResolvedValueOnce(errorResponse(401))
    .mockResolvedValueOnce(new Response(null, { status: 204 }))
    .mockResolvedValueOnce(jsonResponse({ user_id: "current-user" }));
  configureApiSessionRecovery({ onInvalidSession: vi.fn() });

  await expect(
    apiFetch("/api/v1/auth/me", { authenticated: true }),
  ).resolves.toMatchObject({ state: "success", status: 200 });
  expect(fetchMock.mock.calls[2]?.[0]).toBe("/api/v1/auth/refresh");
});

test("requisição iniciada durante refresh lento compartilha o mesmo resultado", async () => {
  let finishRefresh: ((response: Response) => void) | undefined;
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(errorResponse(401))
    .mockImplementationOnce(
      () =>
        new Promise<Response>((resolve) => {
          finishRefresh = resolve;
        }),
    )
    .mockResolvedValueOnce(errorResponse(401))
    .mockResolvedValueOnce(jsonResponse({ user_id: "current-user" }))
    .mockResolvedValueOnce(jsonResponse({ value: "current" }));
  configureApiSessionRecovery({ onInvalidSession: vi.fn() });

  const first = apiFetch("/api/v1/auth/me", { authenticated: true });
  await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));
  const second = apiFetch("/api/v1/establishment", { authenticated: true });
  await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(3));
  finishRefresh?.(new Response(null, { status: 204 }));

  await expect(Promise.all([first, second])).resolves.toEqual([
    expect.objectContaining({ state: "success", status: 200 }),
    expect.objectContaining({ state: "success", status: 200 }),
  ]);
  expect(fetchMock).toHaveBeenCalledTimes(5);
});

test("401 atrasado de leitura anterior ao refresh não inicia outra rotação", async () => {
  let finishDelayedRead: ((response: Response) => void) | undefined;
  const fetchMock = vi.spyOn(globalThis, "fetch").mockImplementation(() => {
    const call = fetchMock.mock.calls.length;
    if (call === 1) return Promise.resolve(errorResponse(401));
    if (call === 2) {
      return new Promise<Response>((resolve) => {
        finishDelayedRead = resolve;
      });
    }
    if (call === 3) return Promise.resolve(new Response(null, { status: 204 }));
    return Promise.resolve(jsonResponse({ call }));
  });
  configureApiSessionRecovery({ onInvalidSession: vi.fn() });

  const first = apiFetch("/api/v1/auth/me", { authenticated: true });
  const delayed = apiFetch("/api/v1/establishment", { authenticated: true });
  await expect(first).resolves.toMatchObject({ state: "success" });
  finishDelayedRead?.(errorResponse(401));
  await expect(delayed).resolves.toMatchObject({ state: "success" });

  expect(fetchMock).toHaveBeenCalledTimes(5);
});

test("401 do refresh avisa uma vez e interrompe novas tentativas", async () => {
  const onInvalidSession = vi.fn();
  configureApiSessionRecovery({ onInvalidSession });
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockImplementation(() => Promise.resolve(errorResponse(401)));

  await Promise.all([
    apiFetch("/api/v1/auth/me", { authenticated: true }),
    apiFetch("/api/v1/establishment", { authenticated: true }),
  ]);
  await apiFetch("/api/v1/auth/me", { authenticated: true });

  expect(onInvalidSession).toHaveBeenCalledOnce();
  expect(fetchMock).toHaveBeenCalledTimes(4);
});

test("falha indeterminada do refresh não tenta reutilizar o token", async () => {
  const onInvalidSession = vi.fn();
  configureApiSessionRecovery({ onInvalidSession });
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(errorResponse(401))
    .mockResolvedValueOnce(errorResponse(503))
    .mockResolvedValueOnce(errorResponse(401));

  await apiFetch("/api/v1/auth/me", { authenticated: true });
  await apiFetch("/api/v1/auth/me", { authenticated: true });

  expect(onInvalidSession).not.toHaveBeenCalled();
  expect(fetchMock).toHaveBeenCalledTimes(3);
});

test("401 no retry não cria loop de recuperação", async () => {
  const onInvalidSession = vi.fn();
  configureApiSessionRecovery({ onInvalidSession });
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(errorResponse(401))
    .mockResolvedValueOnce(new Response(null, { status: 204 }))
    .mockResolvedValueOnce(errorResponse(401));

  await expect(
    apiFetch("/api/v1/auth/me", { authenticated: true }),
  ).resolves.toMatchObject({ state: "error", status: 401 });
  expect(onInvalidSession).toHaveBeenCalledOnce();
  expect(fetchMock).toHaveBeenCalledTimes(3);
});

test("chamada pública e refresh explícito não entram na recuperação automática", async () => {
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockImplementation(() => Promise.resolve(errorResponse(401)));
  configureApiSessionRecovery({ onInvalidSession: vi.fn() });

  await apiFetch("/api/v1/auth/me");
  await apiFetch("/api/v1/auth/refresh", {
    authenticated: true,
    csrf: true,
    method: "POST",
  });

  expect(fetchMock).toHaveBeenCalledTimes(2);
});

function errorResponse(status: number) {
  return jsonResponse(
    {
      code: "authentication_required",
      message: "Entre novamente.",
      request_id: "id",
    },
    status,
  );
}

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    headers: { "Content-Type": "application/json" },
    status,
  });
}

function installWebLock() {
  const request = vi.fn(
    async (
      _name: string,
      _options: LockOptions,
      callback: (lock: Lock | null) => Promise<unknown>,
    ) => callback(null),
  );
  vi.stubGlobal("navigator", { locks: { request } });
  return request;
}

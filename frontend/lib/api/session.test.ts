import { afterEach, beforeEach, expect, test, vi } from "vitest";

import {
  apiFetch,
  configureApiSessionRecovery,
  runManualRefresh,
  runSessionAction,
} from "./client";

beforeEach(() => {
  vi.spyOn(document, "cookie", "get").mockReturnValue(
    "__Secure-barbershop_csrf=csrf-current",
  );
});

afterEach(async () => {
  configureApiSessionRecovery(null);
  await runSessionAction(
    async () => ({ state: "empty", status: 204 }),
    "authenticated",
  );
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

test("visitante sem cookie CSRF permanece anônimo sem tentar refresh", async () => {
  vi.spyOn(document, "cookie", "get").mockReturnValue("");
  const onInvalidSession = vi.fn();
  configureApiSessionRecovery({ onInvalidSession });
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(errorResponse(401))
    .mockResolvedValueOnce(errorResponse(403));

  await expect(
    apiFetch("/api/v1/auth/me", { authenticated: true }),
  ).resolves.toMatchObject({ state: "error", status: 401 });
  expect(fetchMock).toHaveBeenCalledOnce();
  expect(onInvalidSession).toHaveBeenCalledOnce();
});

test("sem CSRF, Web Lock reconsulta a sessão sem tentar refresh", async () => {
  vi.spyOn(document, "cookie", "get").mockReturnValue("");
  const requestLock = installWebLock();
  const onInvalidSession = vi.fn();
  configureApiSessionRecovery({ onInvalidSession });
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(errorResponse(401))
    .mockResolvedValueOnce(errorResponse(401));

  await expect(
    apiFetch("/api/v1/auth/me", { authenticated: true }),
  ).resolves.toMatchObject({ state: "error", status: 401 });
  expect(requestLock).toHaveBeenCalledOnce();
  expect(fetchMock.mock.calls.map(([path]) => path)).toEqual([
    "/api/v1/auth/me",
    "/api/v1/auth/me",
  ]);
  expect(onInvalidSession).toHaveBeenCalledOnce();
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

test("renovação manual indeterminada bloqueia uma segunda tentativa", async () => {
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(errorResponse(503));
  const action = () =>
    apiFetch<void>("/api/v1/auth/refresh", {
      method: "POST",
      csrf: true,
    });

  await expect(runManualRefresh(action)).resolves.toMatchObject({
    state: "error",
    status: 503,
    error: { code: "session_recovery_unavailable" },
  });
  await expect(runManualRefresh(action)).resolves.toMatchObject({
    state: "error",
    status: 503,
  });
  expect(fetchMock).toHaveBeenCalledOnce();
});

test("renovação manual aguarda a tentativa automática em andamento", async () => {
  let finishAutomatic: ((response: Response) => void) | undefined;
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(errorResponse(401))
    .mockImplementationOnce(
      () =>
        new Promise<Response>((resolve) => {
          finishAutomatic = resolve;
        }),
    )
    .mockResolvedValueOnce(jsonResponse({ user_id: "current-user" }))
    .mockResolvedValueOnce(new Response(null, { status: 204 }));
  configureApiSessionRecovery({ onInvalidSession: vi.fn() });
  const read = apiFetch("/api/v1/auth/me", { authenticated: true });
  await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));

  const action = vi.fn(() =>
    apiFetch<void>("/api/v1/auth/refresh", { method: "POST", csrf: true }),
  );
  const manual = runManualRefresh(action);
  expect(action).not.toHaveBeenCalled();
  finishAutomatic?.(new Response(null, { status: 204 }));

  await expect(Promise.all([read, manual])).resolves.toEqual([
    expect.objectContaining({ state: "error", status: 401 }),
    expect.objectContaining({ state: "empty", status: 204 }),
  ]);
  expect(action).toHaveBeenCalledOnce();
  expect(fetchMock.mock.calls.map(([path]) => path)).toEqual([
    "/api/v1/auth/me",
    "/api/v1/auth/refresh",
    "/api/v1/auth/me",
    "/api/v1/auth/refresh",
  ]);
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

test("aguarda o refresh antes de uma mudança explícita de sessão", async () => {
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
    .mockResolvedValueOnce(jsonResponse({ user_id: "before" }));
  configureApiSessionRecovery({ onInvalidSession: vi.fn() });
  const read = apiFetch("/api/v1/auth/me", { authenticated: true });
  await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));

  const action = vi.fn(async () => ({ state: "empty" as const, status: 204 }));
  const change = runSessionAction(action, "authenticated");
  expect(action).not.toHaveBeenCalled();
  finishRefresh?.(new Response(null, { status: 204 }));
  await Promise.all([read, change]);

  expect(action).toHaveBeenCalledOnce();
});

test("nova sessão reabilita recuperação após refresh inválido", async () => {
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(errorResponse(401))
    .mockResolvedValueOnce(errorResponse(401))
    .mockResolvedValueOnce(errorResponse(401))
    .mockResolvedValueOnce(new Response(null, { status: 204 }))
    .mockResolvedValueOnce(jsonResponse({ user_id: "new" }));
  configureApiSessionRecovery({ onInvalidSession: vi.fn() });

  await apiFetch("/api/v1/auth/me", { authenticated: true });
  await runSessionAction(
    async () => ({ state: "empty", status: 204 }),
    "authenticated",
  );
  await expect(
    apiFetch("/api/v1/auth/me", { authenticated: true }),
  ).resolves.toMatchObject({ state: "success", status: 200 });
  expect(fetchMock).toHaveBeenCalledTimes(5);
});

test("leitura antiga não é repetida sob conta autenticada depois", async () => {
  let finishOldRead: ((response: Response) => void) | undefined;
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockImplementationOnce(
      () =>
        new Promise<Response>((resolve) => {
          finishOldRead = resolve;
        }),
    )
    .mockResolvedValueOnce(jsonResponse({ tenant: "new-account" }));
  configureApiSessionRecovery({ onInvalidSession: vi.fn() });

  const oldRead = apiFetch("/api/v1/establishment", { authenticated: true });
  await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledOnce());
  await runSessionAction(
    async () => ({ state: "empty", status: 204 }),
    "authenticated",
  );
  finishOldRead?.(errorResponse(401));

  await expect(oldRead).resolves.toMatchObject({ state: "error", status: 401 });
  expect(fetchMock).toHaveBeenCalledOnce();
});

test("descarta retry que termina depois de uma troca explícita de conta", async () => {
  let finishRetry: ((response: Response) => void) | undefined;
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(errorResponse(401))
    .mockResolvedValueOnce(new Response(null, { status: 204 }))
    .mockImplementationOnce(
      () =>
        new Promise<Response>((resolve) => {
          finishRetry = resolve;
        }),
    );
  configureApiSessionRecovery({ onInvalidSession: vi.fn() });
  const oldRead = apiFetch("/api/v1/establishment", { authenticated: true });
  await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(3));

  await runSessionAction(
    async () => ({ state: "empty", status: 204 }),
    "authenticated",
  );
  finishRetry?.(jsonResponse({ tenant: "new-account" }));

  await expect(oldRead).resolves.toMatchObject({ state: "error", status: 401 });
});

test("ação explícita usa o mesmo Web Lock da renovação automática", async () => {
  const requestLock = installWebLock();
  const action = vi.fn(async () => ({ state: "empty" as const, status: 204 }));

  await runSessionAction(action, "authenticated");

  expect(requestLock).toHaveBeenCalledWith(
    "barbershop-auth-refresh",
    { mode: "exclusive" },
    expect.any(Function),
  );
  expect(action).toHaveBeenCalledOnce();
});

test("não inicia refresh automático durante uma mudança explícita de sessão", async () => {
  let finishAction: (() => void) | undefined;
  configureApiSessionRecovery({ onInvalidSession: vi.fn() });
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValue(errorResponse(401));
  const change = runSessionAction(
    () =>
      new Promise<{ state: "empty"; status: 204 }>((resolve) => {
        finishAction = () => resolve({ state: "empty", status: 204 });
      }),
    "authenticated",
  );
  await vi.waitFor(() => expect(finishAction).toBeDefined());

  await expect(
    apiFetch("/api/v1/auth/me", { authenticated: true }),
  ).resolves.toMatchObject({ state: "error", status: 401 });
  expect(fetchMock).toHaveBeenCalledOnce();
  finishAction?.();
  await change;
});

test("desmontar o Provider durante refresh não inicia uma segunda rotação", async () => {
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
    .mockResolvedValueOnce(jsonResponse({ user_id: "current" }))
    .mockResolvedValueOnce(jsonResponse({ user_id: "current" }));
  configureApiSessionRecovery({ onInvalidSession: vi.fn() });
  const first = apiFetch("/api/v1/auth/me", { authenticated: true });
  await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));

  configureApiSessionRecovery(null);
  configureApiSessionRecovery({ onInvalidSession: vi.fn() });
  const second = apiFetch("/api/v1/auth/me", { authenticated: true });
  await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(3));
  finishRefresh?.(new Response(null, { status: 204 }));

  await expect(Promise.all([first, second])).resolves.toEqual([
    expect.objectContaining({ state: "success" }),
    expect.objectContaining({ state: "success" }),
  ]);
  expect(
    fetchMock.mock.calls.filter(([path]) => path === "/api/v1/auth/refresh"),
  ).toHaveLength(1);
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

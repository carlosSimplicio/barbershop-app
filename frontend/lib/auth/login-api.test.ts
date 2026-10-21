import { afterEach, expect, test, vi } from "vitest";

import { login } from "./login-api";

afterEach(() => vi.restoreAllMocks());

test("envia as credenciais no corpo e recebe o principal pelos cookies de sessão", async () => {
  const principal = {
    user_id: "user-id",
    establishment_id: "establishment-id",
    email: "owner@example.com",
    role: "owner" as const,
    password_change_required: false,
  };
  const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(
    new Response(JSON.stringify(principal), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    }),
  );

  await expect(
    login({ email: "owner@example.com", password: "senha-segura" }),
  ).resolves.toEqual({ state: "success", status: 200, data: principal });

  const [path, init] = fetchMock.mock.calls[0];
  expect(path).toBe("/api/v1/auth/login");
  expect(init?.method).toBe("POST");
  expect(init?.credentials).toBe("include");
  expect(JSON.parse(String(init?.body))).toEqual({
    email: "owner@example.com",
    password: "senha-segura",
  });
  expect(new Headers(init?.headers).has("X-CSRF-Token")).toBe(false);
});

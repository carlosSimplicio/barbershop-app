import { afterEach, expect, test, vi } from "vitest";

import { requestPasswordRecovery } from "./password-recovery-api";

afterEach(() => vi.restoreAllMocks());

test("solicita recuperação pelo endpoint público somente com o e-mail no corpo", async () => {
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValue(new Response(null, { status: 202 }));

  await expect(
    requestPasswordRecovery({ email: "owner@example.test" }),
  ).resolves.toEqual({ state: "empty", status: 202 });

  expect(fetchMock).toHaveBeenCalledOnce();
  const [url, init] = fetchMock.mock.calls[0];
  expect(url).toBe("/api/v1/auth/password/recovery");
  expect(init?.method).toBe("POST");
  expect(init?.credentials).toBe("include");
  expect(JSON.parse(String(init?.body))).toEqual({
    email: "owner@example.test",
  });
  expect(new Headers(init?.headers).has("X-CSRF-Token")).toBe(false);
});

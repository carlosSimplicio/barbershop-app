import { afterEach, expect, test, vi } from "vitest";

import { verifyEmail } from "./email-verification-api";

afterEach(() => vi.restoreAllMocks());

test("envia o token somente no corpo da verificação pública", async () => {
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

  await expect(verifyEmail({ token: "opaque-sample" })).resolves.toEqual({
    state: "success",
    status: 200,
    data: principal,
  });

  expect(fetchMock).toHaveBeenCalledOnce();
  const [path, init] = fetchMock.mock.calls[0];
  expect(path).toBe("/api/v1/auth/email-confirmation/verify");
  expect(init?.method).toBe("POST");
  expect(init?.credentials).toBe("include");
  expect(JSON.parse(String(init?.body))).toEqual({ token: "opaque-sample" });
  expect(new Headers(init?.headers).has("X-CSRF-Token")).toBe(false);
});

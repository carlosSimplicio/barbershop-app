import { render, screen, waitFor } from "@testing-library/react";
import { StrictMode } from "react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";

import { EmailConfirmationBoundary } from "./email-confirmation-boundary";

const observed = vi.hoisted(() => ({
  providerHash: vi.fn(),
  verificationToken: vi.fn(),
}));

vi.mock("@/components/session/auth-provider/auth-provider", () => ({
  AuthProvider: ({ children }: { children: ReactNode }) => {
    observed.providerHash(window.location.hash);
    return children;
  },
}));
vi.mock("../email-verification/email-verification", () => ({
  EmailVerification: ({ token }: { token: string | null }) => {
    observed.verificationToken(token);
    return <div>Verificação pronta</div>;
  },
}));

beforeEach(() => {
  window.history.replaceState(null, "", "/confirmar-email");
  observed.providerHash.mockClear();
  observed.verificationToken.mockClear();
});

afterEach(() => vi.restoreAllMocks());

test("remove o fragmento antes de montar o Provider e passa o token somente em memória", async () => {
  window.history.replaceState(
    null,
    "",
    "/confirmar-email?src=mail#token=opaque-sample",
  );
  render(<EmailConfirmationBoundary />);

  await waitFor(() =>
    expect(screen.getByText("Verificação pronta")).toBeInTheDocument(),
  );
  expect(observed.providerHash).toHaveBeenCalledWith("");
  expect(observed.verificationToken).toHaveBeenCalledWith("opaque-sample");
  expect(window.location.search).toBe("");
  expect(window.location.href).not.toContain("opaque-sample");
  expect(document.body.textContent).not.toContain("opaque-sample");
});

test("se a URL não puder ser limpa, não monta o Provider nem consome o token", async () => {
  window.history.replaceState(null, "", "/confirmar-email#token=opaque-sample");
  vi.spyOn(window.history, "replaceState").mockImplementation(() => {
    throw new Error("history unavailable");
  });
  render(<EmailConfirmationBoundary />);

  expect(
    await screen.findByText(/não foi possível preparar a confirmação/i),
  ).toBeInTheDocument();
  expect(observed.providerHash).not.toHaveBeenCalled();
  expect(observed.verificationToken).not.toHaveBeenCalled();
});

test("preserva a captura na repetição de efeitos do Strict Mode", async () => {
  window.history.replaceState(null, "", "/confirmar-email#token=opaque-sample");
  render(
    <StrictMode>
      <EmailConfirmationBoundary />
    </StrictMode>,
  );

  await waitFor(() =>
    expect(observed.verificationToken).toHaveBeenCalledWith("opaque-sample"),
  );
  expect(observed.providerHash).toHaveBeenCalledWith("");
  expect(window.location.hash).toBe("");
});

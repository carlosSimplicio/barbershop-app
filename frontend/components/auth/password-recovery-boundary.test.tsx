import { render, screen, waitFor } from "@testing-library/react";
import { StrictMode } from "react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";

import { PasswordRecoveryBoundary } from "./password-recovery-boundary";

const observed = vi.hoisted(() => ({ form: vi.fn() }));

vi.mock("./password-recovery-reset-form", () => ({
  PasswordRecoveryResetForm: ({ token }: { token: string }) => {
    observed.form(token, window.location.href);
    return <div>Formulário pronto</div>;
  },
}));

beforeEach(() => {
  window.history.replaceState(null, "", "/redefinir-senha");
  observed.form.mockClear();
});

afterEach(() => vi.restoreAllMocks());

test("remove fragmento e query antes de montar o formulário e guarda o token só em memória", async () => {
  window.history.replaceState(
    null,
    "",
    "/redefinir-senha?src=mail#token=opaque-sample",
  );
  render(<PasswordRecoveryBoundary />);

  expect(await screen.findByText("Formulário pronto")).toBeInTheDocument();
  expect(observed.form).toHaveBeenCalledWith(
    "opaque-sample",
    expect.stringMatching(/\/redefinir-senha$/),
  );
  expect(window.location.search).toBe("");
  expect(window.location.hash).toBe("");
  expect(document.body.textContent).not.toContain("opaque-sample");
});

test("link sem token mostra recusa e permite solicitar outro", async () => {
  window.history.replaceState(null, "", "/redefinir-senha?src=mail");
  render(<PasswordRecoveryBoundary />);

  expect(
    await screen.findByText(/link inválido ou expirado/i),
  ).toBeInTheDocument();
  expect(
    screen.getByRole("link", { name: /solicitar outro link/i }),
  ).toHaveAttribute("href", "/recuperar-senha");
  expect(observed.form).not.toHaveBeenCalled();
  expect(window.location.search).toBe("");
});

test("falha ao limpar a URL impede montar o formulário", async () => {
  window.history.replaceState(null, "", "/redefinir-senha#token=opaque-sample");
  vi.spyOn(window.history, "replaceState").mockImplementation(() => {
    throw new Error("history unavailable");
  });
  render(<PasswordRecoveryBoundary />);

  expect(
    await screen.findByText(/não foi possível abrir este link com segurança/i),
  ).toBeInTheDocument();
  expect(observed.form).not.toHaveBeenCalled();
});

test("preserva o token capturado na repetição de efeitos do Strict Mode", async () => {
  window.history.replaceState(null, "", "/redefinir-senha#token=opaque-sample");
  render(
    <StrictMode>
      <PasswordRecoveryBoundary />
    </StrictMode>,
  );

  await waitFor(() =>
    expect(observed.form).toHaveBeenCalledWith(
      "opaque-sample",
      expect.stringMatching(/\/redefinir-senha$/),
    ),
  );
});

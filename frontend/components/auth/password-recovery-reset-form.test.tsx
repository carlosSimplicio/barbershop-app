import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";

import { verifyPasswordRecovery } from "@/lib/auth/password-recovery-api";

import { PasswordRecoveryResetForm } from "./password-recovery-reset-form";

vi.mock("@/lib/auth/password-recovery-api", () => ({
  verifyPasswordRecovery: vi.fn(),
}));

beforeEach(() => {
  vi.mocked(verifyPasswordRecovery).mockReset();
});

test("valida a nova senha e mantém o foco no campo inválido", () => {
  render(<PasswordRecoveryResetForm token="opaque-sample" />);
  const button = screen.getByRole("button", { name: "Redefinir senha" });
  fireEvent.click(button);

  expect(verifyPasswordRecovery).not.toHaveBeenCalled();
  expect(screen.getByText("Informe uma nova senha.")).toBeInTheDocument();
  expect(screen.getByLabelText("Nova senha")).toHaveFocus();

  fireEvent.change(screen.getByLabelText("Nova senha"), {
    target: { value: "senha curta " },
  });
  fireEvent.click(button);
  expect(
    screen.getByText("Use de 8 a 32 caracteres, sem espaços."),
  ).toBeInTheDocument();
  expect(verifyPasswordRecovery).not.toHaveBeenCalled();
});

test("envia uma única redefinição, informa processamento e orienta login após 204", async () => {
  let finish:
    | ((result: Awaited<ReturnType<typeof verifyPasswordRecovery>>) => void)
    | undefined;
  vi.mocked(verifyPasswordRecovery).mockImplementation(
    () =>
      new Promise((resolve) => {
        finish = resolve;
      }),
  );
  render(<PasswordRecoveryResetForm token="opaque-sample" />);
  fillPassword("nova-senha-segura");
  fireEvent.click(screen.getByRole("button", { name: "Redefinir senha" }));
  expect(
    screen.getByRole("button", { name: "Redefinindo senha…" }),
  ).toBeDisabled();
  fireEvent.submit(
    screen.getByRole("button", { name: "Redefinindo senha…" }).closest("form")!,
  );
  expect(verifyPasswordRecovery).toHaveBeenCalledOnce();
  expect(verifyPasswordRecovery).toHaveBeenCalledWith({
    token: "opaque-sample",
    new_password: "nova-senha-segura",
  });

  finish?.({ state: "empty", status: 204 });
  expect(await screen.findByText("Senha redefinida")).toBeInTheDocument();
  await waitFor(() => expect(screen.getByRole("status")).toHaveFocus());
  expect(screen.getByRole("link", { name: "Ir para o login" })).toHaveAttribute(
    "href",
    "/login",
  );
  expect(
    screen.queryByRole("button", { name: "Redefinir senha" }),
  ).not.toBeInTheDocument();
});

test("link usado ou expirado mostra recusa sem revelar detalhes", async () => {
  vi.mocked(verifyPasswordRecovery).mockResolvedValue({
    state: "error",
    status: 400,
    error: {
      code: "invalid_password_recovery",
      message: "detalhe bruto",
      request_id: "test",
    },
  });
  render(<PasswordRecoveryResetForm token="opaque-sample" />);
  fillPassword("nova-senha-segura");
  fireEvent.click(screen.getByRole("button", { name: "Redefinir senha" }));

  expect(
    await screen.findByText("Link inválido ou expirado"),
  ).toBeInTheDocument();
  expect(
    screen.getByRole("link", { name: "Solicitar outro link" }),
  ).toHaveAttribute("href", "/recuperar-senha");
  expect(screen.queryByText("detalhe bruto")).not.toBeInTheDocument();
  expect(
    screen.queryByRole("button", { name: "Redefinir senha" }),
  ).not.toBeInTheDocument();
});

test.each([
  ["invalid_password", "Use de 8 a 32 caracteres, sem espaços."],
  ["password_unchanged", "Escolha uma senha diferente da atual."],
] as const)(
  "mostra %s junto ao campo e permite corrigir",
  async (code, message) => {
    vi.mocked(verifyPasswordRecovery).mockResolvedValue({
      state: "error",
      status: 400,
      error: {
        code: "validation_failed",
        message: "detalhe bruto",
        request_id: "test",
        fields: [{ field: "new_password", code, message: "não exibir" }],
      },
    });
    render(<PasswordRecoveryResetForm token="opaque-sample" />);
    fillPassword("nova-senha-segura");
    fireEvent.click(screen.getByRole("button", { name: "Redefinir senha" }));

    expect(await screen.findByText(message)).toBeInTheDocument();
    expect(screen.getByLabelText("Nova senha")).toHaveFocus();
    expect(
      screen.getByRole("button", { name: "Redefinir senha" }),
    ).toBeEnabled();
    expect(screen.queryByText("detalhe bruto")).not.toBeInTheDocument();
  },
);

test("falha de rede apresenta nova tentativa sem mostrar erro técnico", async () => {
  vi.mocked(verifyPasswordRecovery).mockRejectedValue(
    new Error("network secret"),
  );
  render(<PasswordRecoveryResetForm token="opaque-sample" />);
  fillPassword("nova-senha-segura");
  fireEvent.click(screen.getByRole("button", { name: "Redefinir senha" }));

  expect(
    await screen.findByText("Verifique sua conexão e tente novamente."),
  ).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Redefinir senha" })).toBeEnabled();
  expect(screen.queryByText("network secret")).not.toBeInTheDocument();
});

function fillPassword(value: string) {
  fireEvent.change(screen.getByLabelText("Nova senha"), {
    target: { value },
  });
}

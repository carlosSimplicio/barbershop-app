import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";

import { requestPasswordRecovery } from "@/lib/auth/password-recovery-api";

import { PasswordRecoveryRequestForm } from "./password-recovery-request-form";

vi.mock("@/lib/auth/password-recovery-api", () => ({
  requestPasswordRecovery: vi.fn(),
}));

beforeEach(() => vi.mocked(requestPasswordRecovery).mockReset());

test("valida o e-mail, mantém o foco e permite envio pelo formulário", async () => {
  vi.mocked(requestPasswordRecovery).mockResolvedValue({
    state: "empty",
    status: 202,
  });
  render(<PasswordRecoveryRequestForm />);
  const email = screen.getByRole("textbox", { name: "E-mail" });
  const form = email.closest("form");
  expect(form).not.toBeNull();

  fireEvent.submit(form!);
  expect(requestPasswordRecovery).not.toHaveBeenCalled();
  expect(email).toHaveFocus();
  expect(screen.getByText("Informe seu e-mail.")).toBeInTheDocument();

  fireEvent.change(email, { target: { value: "inválido" } });
  fireEvent.submit(form!);
  expect(requestPasswordRecovery).not.toHaveBeenCalled();
  expect(screen.getByText("Digite um e-mail válido.")).toBeInTheDocument();

  fireEvent.change(email, { target: { value: " owner@example.test " } });
  email.focus();
  fireEvent.submit(form!);
  await waitFor(() =>
    expect(requestPasswordRecovery).toHaveBeenCalledWith({
      email: "owner@example.test",
    }),
  );
});

test("exibe o mesmo sucesso genérico após aceite e impede novo envio", async () => {
  vi.mocked(requestPasswordRecovery).mockResolvedValue({
    state: "empty",
    status: 202,
  });
  render(<PasswordRecoveryRequestForm />);
  fillEmail();
  fireEvent.click(screen.getByRole("button", { name: "Solicitar link" }));

  expect(await screen.findByText("Solicitação recebida")).toBeInTheDocument();
  await waitFor(() => expect(screen.getByRole("status")).toHaveFocus());
  expect(
    screen.getByText(/Se houver uma conta apta para este e-mail/),
  ).toBeInTheDocument();
  expect(
    screen.queryByRole("textbox", { name: "E-mail" }),
  ).not.toBeInTheDocument();
  expect(requestPasswordRecovery).toHaveBeenCalledOnce();
});

test("informa limite e falha sem exibir mensagens brutas do servidor", async () => {
  vi.mocked(requestPasswordRecovery)
    .mockResolvedValueOnce({
      state: "error",
      status: 429,
      error: {
        code: "password_recovery_rate_limited",
        message: "detalhe interno",
        request_id: "test",
      },
    })
    .mockResolvedValueOnce({
      state: "error",
      status: 500,
      error: {
        code: "internal_error",
        message: "detalhe interno",
        request_id: "test",
      },
    });
  render(<PasswordRecoveryRequestForm />);
  fillEmail();
  fireEvent.click(screen.getByRole("button", { name: "Solicitar link" }));
  expect(
    await screen.findByText("Muitas solicitações. Tente novamente mais tarde."),
  ).toBeInTheDocument();
  expect(screen.queryByText("detalhe interno")).not.toBeInTheDocument();

  fireEvent.click(screen.getByRole("button", { name: "Solicitar link" }));
  expect(
    await screen.findByText("Tente novamente mais tarde."),
  ).toBeInTheDocument();
  expect(screen.queryByText("detalhe interno")).not.toBeInTheDocument();
});

test("associa erro de e-mail do backend ao campo sem mostrar a mensagem bruta", async () => {
  vi.mocked(requestPasswordRecovery).mockResolvedValue({
    state: "error",
    status: 400,
    error: {
      code: "validation_failed",
      message: "detalhe interno",
      request_id: "test",
      fields: [
        { field: "email", code: "invalid_email", message: "não exibir" },
      ],
    },
  });
  render(<PasswordRecoveryRequestForm />);
  fillEmail();
  fireEvent.click(screen.getByRole("button", { name: "Solicitar link" }));

  expect(
    await screen.findByText("Digite um e-mail válido."),
  ).toBeInTheDocument();
  expect(screen.getByRole("textbox", { name: "E-mail" })).toHaveFocus();
  expect(screen.queryByText("detalhe interno")).not.toBeInTheDocument();
  expect(screen.queryByText("não exibir")).not.toBeInTheDocument();
});

test("oferece nova tentativa após falha de rede", async () => {
  vi.mocked(requestPasswordRecovery).mockRejectedValueOnce(
    new Error("network detail"),
  );
  render(<PasswordRecoveryRequestForm />);
  fillEmail();
  fireEvent.click(screen.getByRole("button", { name: "Solicitar link" }));
  expect(
    await screen.findByText("Verifique sua conexão e tente novamente."),
  ).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Solicitar link" })).toBeEnabled();
  expect(screen.queryByText("network detail")).not.toBeInTheDocument();
});

test("bloqueia dois envios enquanto a solicitação está pendente", async () => {
  vi.mocked(requestPasswordRecovery).mockResolvedValue({
    state: "empty",
    status: 202,
  });
  render(<PasswordRecoveryRequestForm />);
  fillEmail();
  const button = screen.getByRole("button", { name: "Solicitar link" });
  fireEvent.click(button);
  fireEvent.submit(button.closest("form")!);
  expect(requestPasswordRecovery).toHaveBeenCalledOnce();
  expect(await screen.findByText("Solicitação recebida")).toBeInTheDocument();
});

function fillEmail() {
  fireEvent.change(screen.getByRole("textbox", { name: "E-mail" }), {
    target: { value: "owner@example.test" },
  });
}

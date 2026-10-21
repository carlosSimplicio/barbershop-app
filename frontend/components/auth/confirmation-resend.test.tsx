import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";

import { requestEmailConfirmation } from "@/lib/auth/confirmation-api";

import { ConfirmationResend } from "./confirmation-resend";

vi.mock("@/lib/auth/confirmation-api", () => ({
  requestEmailConfirmation: vi.fn(),
}));

beforeEach(() => {
  vi.mocked(requestEmailConfirmation).mockReset();
});

test("reenvia para o e-mail informado e usa resposta genérica após 202", async () => {
  vi.mocked(requestEmailConfirmation).mockResolvedValue({
    state: "empty",
    status: 202,
  });
  render(<ConfirmationResend email="owner@example.com" />);

  fireEvent.click(screen.getByRole("button", { name: /reenviar link/i }));

  await waitFor(() =>
    expect(requestEmailConfirmation).toHaveBeenCalledWith({
      email: "owner@example.com",
    }),
  );
  expect(
    await screen.findByText(/se houver uma conta pendente/i),
  ).toBeInTheDocument();
  expect(screen.queryByText(/owner@example.com/i)).not.toBeInTheDocument();
});

test("impede envio duplicado enquanto a solicitação está pendente", async () => {
  let finishRequest:
    ((result: { state: "empty"; status: 202 }) => void) | undefined;
  vi.mocked(requestEmailConfirmation).mockImplementation(
    () =>
      new Promise((resolve) => {
        finishRequest = resolve;
      }),
  );
  render(<ConfirmationResend email="owner@example.com" />);

  fireEvent.click(screen.getByRole("button", { name: /reenviar link/i }));
  fireEvent.click(screen.getByRole("button", { name: /enviando/i }));
  expect(requestEmailConfirmation).toHaveBeenCalledOnce();

  finishRequest?.({ state: "empty", status: 202 });
  expect(
    await screen.findByText(/se houver uma conta pendente/i),
  ).toBeInTheDocument();
});

test("permite nova solicitação depois do sucesso quando o link não chega", async () => {
  vi.mocked(requestEmailConfirmation).mockResolvedValue({
    state: "empty",
    status: 202,
  });
  render(<ConfirmationResend email="owner@example.com" />);

  fireEvent.click(screen.getByRole("button", { name: /reenviar link/i }));
  await screen.findByText(/se houver uma conta pendente/i);
  fireEvent.click(screen.getByRole("button", { name: /reenviar link/i }));

  await waitFor(() =>
    expect(requestEmailConfirmation).toHaveBeenCalledTimes(2),
  );
});

test("informa limite de solicitações sem mostrar a mensagem do servidor", async () => {
  vi.mocked(requestEmailConfirmation).mockResolvedValue({
    state: "error",
    status: 429,
    error: {
      code: "email_confirmation_rate_limited",
      message: "detalhe bruto",
      request_id: "test",
    },
  });
  render(<ConfirmationResend email="owner@example.com" />);

  fireEvent.click(screen.getByRole("button", { name: /reenviar link/i }));

  expect(await screen.findByText(/muitas solicitações/i)).toBeInTheDocument();
  expect(screen.queryByText("detalhe bruto")).not.toBeInTheDocument();
});

test("mantém a opção de tentar novamente após falha inesperada", async () => {
  vi.mocked(requestEmailConfirmation).mockResolvedValue({
    state: "error",
    status: 500,
    error: {
      code: "internal_error",
      message: "detalhe bruto",
      request_id: "test",
    },
  });
  render(<ConfirmationResend email="owner@example.com" />);

  fireEvent.click(screen.getByRole("button", { name: /reenviar link/i }));

  expect(
    await screen.findByText(/não foi possível solicitar outro link/i),
  ).toBeInTheDocument();
  expect(screen.getByRole("button", { name: /reenviar link/i })).toBeEnabled();
  expect(screen.queryByText("detalhe bruto")).not.toBeInTheDocument();
});

test("trata perda de conexão sem revelar detalhes", async () => {
  vi.mocked(requestEmailConfirmation).mockRejectedValue(new Error("network"));
  render(<ConfirmationResend email="owner@example.com" />);

  fireEvent.click(screen.getByRole("button", { name: /reenviar link/i }));

  expect(
    await screen.findByText(/não foi possível solicitar outro link/i),
  ).toBeInTheDocument();
});

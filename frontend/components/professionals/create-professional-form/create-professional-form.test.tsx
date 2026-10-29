import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";

import { createProfessional } from "@/lib/professionals/professionals-api";

import { CreateProfessionalForm } from "./create-professional-form";

vi.mock("@/lib/professionals/professionals-api", () => ({
  createProfessional: vi.fn(),
}));

beforeEach(() => vi.mocked(createProfessional).mockReset());

test("valida os campos antes do envio e foca o primeiro erro", () => {
  render(<CreateProfessionalForm onCreated={vi.fn()} />);
  fireEvent.click(
    screen.getByRole("button", { name: "Cadastrar profissional" }),
  );

  expect(screen.getByLabelText("Nome")).toHaveFocus();
  expect(
    screen.getByText("Informe o nome do profissional."),
  ).toBeInTheDocument();
  expect(createProfessional).not.toHaveBeenCalled();

  fireEvent.change(screen.getByLabelText("Nome"), {
    target: { value: "A".repeat(121) },
  });
  fireEvent.change(screen.getByLabelText("E-mail"), {
    target: { value: "email inválido" },
  });
  fireEvent.change(screen.getByLabelText("Senha temporária"), {
    target: { value: "curta" },
  });
  fireEvent.click(
    screen.getByRole("button", { name: "Cadastrar profissional" }),
  );

  expect(
    screen.getByText("Informe um nome com até 120 caracteres."),
  ).toBeInTheDocument();
  expect(screen.getByText("Digite um e-mail válido.")).toBeInTheDocument();
  expect(
    screen.getByText(
      "Use de 8 a 32 caracteres, sem espaços ou caracteres invisíveis.",
    ),
  ).toBeInTheDocument();
  expect(createProfessional).not.toHaveBeenCalled();
});

test("mostra conflito de e-mail sem revelar o texto bruto da API", async () => {
  vi.mocked(createProfessional).mockResolvedValue({
    state: "error",
    status: 409,
    error: {
      code: "conflict",
      message: "mensagem privada do servidor",
      request_id: "request-id",
      fields: [
        {
          field: "email",
          code: "email_unavailable",
          message: "mensagem privada do servidor",
        },
      ],
    },
  });
  render(<CreateProfessionalForm onCreated={vi.fn()} />);
  enterValidValues();
  fireEvent.click(
    screen.getByRole("button", { name: "Cadastrar profissional" }),
  );

  expect(
    await screen.findByText("Este e-mail já está cadastrado."),
  ).toBeInTheDocument();
  expect(
    screen.queryByText("mensagem privada do servidor"),
  ).not.toBeInTheDocument();
});

test("explica a leitura permitida e o bloqueio de cadastro após o trial", async () => {
  vi.mocked(createProfessional).mockResolvedValue(apiError(403, "read_only"));
  render(<CreateProfessionalForm onCreated={vi.fn()} />);
  enterValidValues();
  fireEvent.click(
    screen.getByRole("button", { name: "Cadastrar profissional" }),
  );

  expect(
    await screen.findByText(/período de avaliação não está ativo/i),
  ).toBeInTheDocument();
  expect(
    screen.getByText(/Você ainda pode consultar a lista/i),
  ).toBeInTheDocument();
});

test("trata falha de rede com mensagem genérica", async () => {
  vi.mocked(createProfessional).mockRejectedValueOnce(
    new Error("network detail"),
  );
  render(<CreateProfessionalForm onCreated={vi.fn()} />);
  enterValidValues();
  fireEvent.click(
    screen.getByRole("button", { name: "Cadastrar profissional" }),
  );

  expect(
    await screen.findByText("Verifique sua conexão e tente novamente."),
  ).toBeInTheDocument();
  expect(screen.queryByText("network detail")).not.toBeInTheDocument();
});

test("ignora nova submissão enquanto a primeira requisição está pendente", async () => {
  vi.mocked(createProfessional).mockResolvedValue({
    state: "success",
    status: 201,
    data: {
      id: "professional-id",
      name: "Ana Lima",
      email: "ana@example.test",
      active: true,
    },
  });
  const onCreated = vi.fn();
  render(<CreateProfessionalForm onCreated={onCreated} />);
  enterValidValues();
  const submit = screen.getByRole("button", { name: "Cadastrar profissional" });
  const form = submit.closest("form")!;
  act(() => {
    form.dispatchEvent(
      new Event("submit", { bubbles: true, cancelable: true }),
    );
    form.dispatchEvent(
      new Event("submit", { bubbles: true, cancelable: true }),
    );
  });

  expect(createProfessional).toHaveBeenCalledOnce();
  await waitFor(() => expect(onCreated).toHaveBeenCalledOnce());
});

test("limpa a senha e atualiza a lista depois de 201", async () => {
  vi.mocked(createProfessional).mockResolvedValue({
    state: "success",
    status: 201,
    data: {
      id: "professional-id",
      name: "Ana Lima",
      email: "ana@example.test",
      active: true,
    },
  });
  const onCreated = vi.fn();
  render(<CreateProfessionalForm onCreated={onCreated} />);
  enterValidValues();
  fireEvent.click(
    screen.getByRole("button", { name: "Cadastrar profissional" }),
  );

  expect(
    await screen.findByText(/não poderá ser recuperada/i),
  ).toBeInTheDocument();
  await waitFor(() => expect(onCreated).toHaveBeenCalledOnce(), {
    timeout: 1000,
  });
  expect(screen.getByLabelText("Senha temporária")).toHaveValue("");
  expect(onCreated).toHaveBeenCalledOnce();
});

function enterValidValues() {
  fireEvent.change(screen.getByLabelText("Nome"), {
    target: { value: " Ana Lima " },
  });
  fireEvent.change(screen.getByLabelText("E-mail"), {
    target: { value: "ana@example.test" },
  });
  fireEvent.change(screen.getByLabelText("Senha temporária"), {
    target: { value: "senha-segura" },
  });
}

function apiError(status: number, code: string) {
  return {
    state: "error" as const,
    status,
    error: { code, message: "mensagem privada do servidor", request_id: "id" },
  };
}

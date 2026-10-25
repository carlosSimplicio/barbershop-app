import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";
import { useRouter } from "next/navigation";

import { useAuthSession } from "@/components/session/auth-provider/auth-provider";
import { login } from "@/lib/auth/login-api/login-api";

import { LoginForm } from "./login-form";

vi.mock("next/navigation", () => ({ useRouter: vi.fn() }));
vi.mock("@/components/session/auth-provider/auth-provider", () => ({
  useAuthSession: vi.fn(),
}));
vi.mock("@/lib/auth/login-api/login-api", () => ({ login: vi.fn() }));
vi.mock("../confirmation-resend/confirmation-resend", () => ({
  ConfirmationResend: ({ email }: { email: string }) => (
    <div data-testid="resend-email">{email}</div>
  ),
}));

const replace = vi.fn();
const runAction = vi.fn(async (action: () => ReturnType<typeof login>) =>
  action(),
);

beforeEach(() => {
  replace.mockReset();
  runAction.mockClear();
  vi.mocked(login).mockReset();
  vi.mocked(useRouter).mockReturnValue({ replace } as unknown as ReturnType<
    typeof useRouter
  >);
  vi.mocked(useAuthSession).mockReturnValue({
    state: { status: "anonymous" },
    transitioning: false,
    reload: vi.fn(),
    runAction: runAction as unknown as ReturnType<
      typeof useAuthSession
    >["runAction"],
  });
});

test.each([
  ["owner", false],
  ["professional", true],
] as const)(
  "entra como %s, sincroniza a sessão e substitui a rota apropriada",
  async (role, password_change_required) => {
    vi.mocked(login).mockResolvedValue({
      state: "success",
      status: 200,
      data: {
        user_id: "user-id",
        establishment_id: "establishment-id",
        email: "user@example.com",
        role,
        password_change_required,
      },
    });
    render(<LoginForm />);
    fillForm();

    expect(screen.getByLabelText(/e-mail/i)).toHaveAttribute(
      "autocomplete",
      "username",
    );
    expect(screen.getByLabelText(/^senha$/i)).toHaveAttribute(
      "autocomplete",
      "current-password",
    );
    fireEvent.click(screen.getByRole("button", { name: /entrar/i }));

    await waitFor(() =>
      expect(login).toHaveBeenCalledWith({
        email: "user@example.com",
        password: "senha-segura",
      }),
    );
    expect(runAction).toHaveBeenCalledWith(
      expect.any(Function),
      "authenticated",
    );
    await waitFor(() =>
      expect(replace).toHaveBeenCalledWith(
        role === "owner" ? "/painel/profissionais" : "/testes",
      ),
    );
  },
);

test("recusa campos inválidos e foca o primeiro erro", () => {
  render(<LoginForm />);
  fireEvent.click(screen.getByRole("button", { name: /entrar/i }));

  expect(login).not.toHaveBeenCalled();
  expect(screen.getByLabelText(/e-mail/i)).toHaveFocus();
  expect(screen.getByText("Informe seu e-mail.")).toBeInTheDocument();
  expect(screen.getByText("Informe sua senha.")).toBeInTheDocument();
});

test("oferece acesso à recuperação de senha pelo login", () => {
  render(<LoginForm />);
  const recoveryLink = screen.getByRole("link", {
    name: "Esqueci minha senha",
  });
  expect(recoveryLink).toHaveAttribute("href", "/recuperar-senha");
  recoveryLink.focus();
  expect(recoveryLink).toHaveFocus();
});

test("mostra credenciais inválidas sem revelar detalhes do servidor", async () => {
  vi.mocked(login).mockResolvedValue({
    state: "error",
    status: 401,
    error: {
      code: "invalid_credentials",
      message: "detalhe bruto",
      request_id: "test",
    },
  });
  render(<LoginForm />);
  fillForm();
  fireEvent.click(screen.getByRole("button", { name: /entrar/i }));

  expect(
    await screen.findByText("E-mail ou senha inválidos."),
  ).toBeInTheDocument();
  expect(screen.queryByText("detalhe bruto")).not.toBeInTheDocument();
  expect(replace).not.toHaveBeenCalled();
});

test("associa erro de campo retornado pelo backend sem mostrar a mensagem bruta", async () => {
  vi.mocked(login).mockResolvedValue({
    state: "error",
    status: 400,
    error: {
      code: "invalid_request",
      message: "detalhe bruto",
      request_id: "test",
      fields: [
        { field: "email", code: "invalid_email", message: "não exibir" },
      ],
    },
  });
  render(<LoginForm />);
  fillForm();
  fireEvent.click(screen.getByRole("button", { name: /entrar/i }));

  expect(
    await screen.findByText("Digite um e-mail válido."),
  ).toBeInTheDocument();
  expect(screen.getByLabelText(/e-mail/i)).toHaveFocus();
  expect(screen.queryByText("não exibir")).not.toBeInTheDocument();
});

test("orienta confirmar o e-mail e reutiliza o reenvio para o endereço informado", async () => {
  vi.mocked(login).mockResolvedValue({
    state: "error",
    status: 403,
    error: {
      code: "email_confirmation_required",
      message: "detalhe bruto",
      request_id: "test",
    },
  });
  render(<LoginForm />);
  fillForm();
  fireEvent.click(screen.getByRole("button", { name: /entrar/i }));

  expect(await screen.findByText("Confirme seu e-mail")).toBeInTheDocument();
  expect(screen.getByTestId("resend-email")).toHaveTextContent(
    "user@example.com",
  );
  expect(screen.queryByText("detalhe bruto")).not.toBeInTheDocument();

  fireEvent.change(screen.getByRole("textbox", { name: "E-mail" }), {
    target: { value: "other@example.com" },
  });
  expect(screen.queryByTestId("resend-email")).not.toBeInTheDocument();
});

test("informa rate limit e falha inesperada sem mostrar mensagens brutas", async () => {
  vi.mocked(login)
    .mockResolvedValueOnce({
      state: "error",
      status: 429,
      error: {
        code: "authentication_rate_limited",
        message: "detalhe bruto",
        request_id: "test",
      },
    })
    .mockRejectedValueOnce(new Error("network"));
  render(<LoginForm />);
  fillForm();
  fireEvent.click(screen.getByRole("button", { name: /entrar/i }));

  expect(await screen.findByText(/muitas tentativas/i)).toBeInTheDocument();
  fireEvent.click(screen.getByRole("button", { name: /entrar/i }));
  expect(await screen.findByText(/verifique sua conexão/i)).toBeInTheDocument();
  expect(screen.queryByText("detalhe bruto")).not.toBeInTheDocument();
});

test("impede dois envios enquanto o login está pendente", async () => {
  let finish: ((result: Awaited<ReturnType<typeof login>>) => void) | undefined;
  vi.mocked(login).mockImplementation(
    () =>
      new Promise((resolve) => {
        finish = resolve;
      }),
  );
  render(<LoginForm />);
  fillForm();
  fireEvent.click(screen.getByRole("button", { name: /entrar/i }));
  fireEvent.submit(
    screen.getByRole("button", { name: /entrando/i }).closest("form")!,
  );
  expect(login).toHaveBeenCalledOnce();

  finish?.({
    state: "success",
    status: 200,
    data: {
      user_id: "user-id",
      establishment_id: "establishment-id",
      email: "user@example.com",
      role: "owner",
      password_change_required: false,
    },
  });
  await waitFor(() =>
    expect(replace).toHaveBeenCalledWith("/painel/profissionais"),
  );
});

function fillForm() {
  fireEvent.change(screen.getByLabelText(/e-mail/i), {
    target: { value: " user@example.com " },
  });
  fireEvent.change(screen.getByLabelText(/^senha$/i), {
    target: { value: "senha-segura" },
  });
}

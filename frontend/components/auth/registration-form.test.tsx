import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { renderToString } from "react-dom/server";
import { afterEach, beforeEach, expect, test, vi } from "vitest";

import { registerOwner } from "@/lib/auth/registration-api";

import { RegistrationForm } from "./registration-form";

vi.mock("@/lib/auth/registration-api", () => ({ registerOwner: vi.fn() }));

beforeEach(() => vi.mocked(registerOwner).mockReset());
afterEach(() => vi.restoreAllMocks());

test("sugere slug, usa o fuso detectado e mantém a confirmação pendente após 202", async () => {
  vi.spyOn(Intl.DateTimeFormat.prototype, "resolvedOptions").mockReturnValue({
    timeZone: "America/Manaus",
  } as Intl.ResolvedDateTimeFormatOptions);
  vi.mocked(registerOwner).mockResolvedValue({ state: "empty", status: 202 });
  render(<RegistrationForm />);

  fireEvent.change(
    screen.getByRole("textbox", { name: /nome do estabelecimento/i }),
    {
      target: { value: "Estúdio Aurora" },
    },
  );
  expect(
    screen.getByRole("textbox", { name: /endereço da sua página/i }),
  ).toHaveValue("estudio-aurora");
  expect(screen.getByLabelText(/fuso horário/i)).toHaveValue("America/Manaus");
  fireEvent.change(screen.getByRole("textbox", { name: /seu e-mail/i }), {
    target: { value: "owner@example.com" },
  });
  fireEvent.change(screen.getByLabelText(/crie uma senha/i), {
    target: { value: "senha-segura" },
  });
  fireEvent.click(screen.getByRole("button", { name: /criar meu espaço/i }));

  await waitFor(() =>
    expect(registerOwner).toHaveBeenCalledWith({
      establishment_name: "Estúdio Aurora",
      slug: "estudio-aurora",
      time_zone: "America/Manaus",
      email: "owner@example.com",
      password: "senha-segura",
    }),
  );
  expect(
    await screen.findByRole("heading", { name: /confira seu e-mail/i }),
  ).toBeInTheDocument();
  expect(
    screen.queryByRole("button", { name: /criar meu espaço/i }),
  ).not.toBeInTheDocument();
  expect(
    screen.getByText(/avaliação começa somente após a confirmação/i),
  ).toBeInTheDocument();
  expect(
    screen.getByRole("button", { name: /reenviar link de confirmação/i }),
  ).toBeInTheDocument();
});

test("preserva o slug editado e permite corrigir o fuso sugerido", () => {
  vi.spyOn(Intl.DateTimeFormat.prototype, "resolvedOptions").mockImplementation(
    () => {
      throw new Error("unavailable");
    },
  );
  render(<RegistrationForm />);

  expect(screen.getByLabelText(/fuso horário/i)).toHaveValue(
    "America/Sao_Paulo",
  );
  fireEvent.change(
    screen.getByRole("textbox", { name: /nome do estabelecimento/i }),
    {
      target: { value: "Salão Flor" },
    },
  );
  fireEvent.change(
    screen.getByRole("textbox", { name: /endereço da sua página/i }),
    {
      target: { value: "meu-salao" },
    },
  );
  fireEvent.change(
    screen.getByRole("textbox", { name: /nome do estabelecimento/i }),
    {
      target: { value: "Salão Novo" },
    },
  );
  fireEvent.change(screen.getByLabelText(/fuso horário/i), {
    target: { value: "America/Rio_Branco" },
  });

  expect(
    screen.getByRole("textbox", { name: /endereço da sua página/i }),
  ).toHaveValue("meu-salao");
  expect(screen.getByLabelText(/fuso horário/i)).toHaveValue(
    "America/Rio_Branco",
  );
});

test("mantém o fallback no HTML inicial para evitar divergência na hidratação", () => {
  vi.spyOn(Intl.DateTimeFormat.prototype, "resolvedOptions").mockReturnValue({
    timeZone: "America/Manaus",
  } as Intl.ResolvedDateTimeFormatOptions);

  const html = renderToString(<RegistrationForm />);

  expect(
    html.match(/<input[^>]*list="registration-time-zones"[^>]*>/)?.[0],
  ).toContain('value="America/Sao_Paulo"');
});

test("bloqueia dados inválidos e foca o primeiro campo", () => {
  render(<RegistrationForm />);
  fireEvent.click(screen.getByRole("button", { name: /criar meu espaço/i }));

  expect(registerOwner).not.toHaveBeenCalled();
  expect(
    screen.getByRole("textbox", { name: /nome do estabelecimento/i }),
  ).toHaveFocus();
  expect(
    screen.getByText("Informe o nome do estabelecimento."),
  ).toBeInTheDocument();
  expect(
    screen.getByText("Escolha o endereço da sua página."),
  ).toBeInTheDocument();
  expect(screen.getByText("Informe seu e-mail.")).toBeInTheDocument();
  expect(screen.getByText("Crie uma senha.")).toBeInTheDocument();
});

test("associa conflito de slug ao campo sem mostrar a mensagem bruta do servidor", async () => {
  vi.mocked(registerOwner).mockResolvedValue({
    state: "error",
    status: 409,
    error: {
      code: "conflict",
      message: "mensagem não confiável",
      request_id: "test",
      fields: [
        { field: "slug", code: "slug_unavailable", message: "não exibir" },
      ],
    },
  });
  render(<RegistrationForm />);
  fillValidForm();
  fireEvent.click(screen.getByRole("button", { name: /criar meu espaço/i }));

  expect(
    await screen.findByText("Este endereço não está disponível."),
  ).toBeInTheDocument();
  expect(
    screen.getByRole("textbox", { name: /endereço da sua página/i }),
  ).toHaveFocus();
  expect(screen.queryByText("não exibir")).not.toBeInTheDocument();
});

test("associa conflito de e-mail ao campo correto", async () => {
  vi.mocked(registerOwner).mockResolvedValue({
    state: "error",
    status: 409,
    error: {
      code: "conflict",
      message: "raw",
      request_id: "test",
      fields: [{ field: "email", code: "email_unavailable", message: "raw" }],
    },
  });
  render(<RegistrationForm />);
  fillValidForm();
  fireEvent.click(screen.getByRole("button", { name: /criar meu espaço/i }));

  expect(
    await screen.findByText("Este e-mail já está cadastrado."),
  ).toBeInTheDocument();
  expect(screen.getByRole("textbox", { name: /seu e-mail/i })).toHaveFocus();
});

test("aceita senha Unicode com oito caracteres como o backend", async () => {
  vi.mocked(registerOwner).mockResolvedValue({ state: "empty", status: 202 });
  render(<RegistrationForm />);
  fillValidForm();
  fireEvent.change(screen.getByLabelText(/crie uma senha/i), {
    target: { value: "🔐".repeat(8) },
  });
  fireEvent.click(screen.getByRole("button", { name: /criar meu espaço/i }));

  await waitFor(() => expect(registerOwner).toHaveBeenCalledOnce());
});

test("mostra erro geral seguro quando a API falha", async () => {
  vi.mocked(registerOwner).mockResolvedValue({
    state: "error",
    status: 500,
    error: { code: "internal_error", message: "segredo", request_id: "test" },
  });
  render(<RegistrationForm />);
  fillValidForm();
  fireEvent.click(screen.getByRole("button", { name: /criar meu espaço/i }));

  expect(
    await screen.findByText(
      "Não foi possível concluir o cadastro. Tente novamente.",
    ),
  ).toBeInTheDocument();
  expect(screen.queryByText("segredo")).not.toBeInTheDocument();
});

test("informa rate limit e impede envio duplicado enquanto aguarda", async () => {
  vi.mocked(registerOwner).mockResolvedValue({
    state: "error",
    status: 429,
    error: {
      code: "registration_rate_limited",
      message: "raw",
      request_id: "test",
    },
  });
  render(<RegistrationForm />);
  fillValidForm();
  fireEvent.click(screen.getByRole("button", { name: /criar meu espaço/i }));
  fireEvent.click(screen.getByRole("button", { name: /criando/i }));
  expect(registerOwner).toHaveBeenCalledOnce();

  expect(await screen.findByText(/muitas tentativas/i)).toBeInTheDocument();
  expect(
    screen.getByRole("button", { name: /criar meu espaço/i }),
  ).toBeEnabled();
});

function fillValidForm() {
  fireEvent.change(
    screen.getByRole("textbox", { name: /nome do estabelecimento/i }),
    {
      target: { value: "Estúdio Aurora" },
    },
  );
  fireEvent.change(screen.getByRole("textbox", { name: /seu e-mail/i }), {
    target: { value: "owner@example.com" },
  });
  fireEvent.change(screen.getByLabelText(/crie uma senha/i), {
    target: { value: "senha-segura" },
  });
}
